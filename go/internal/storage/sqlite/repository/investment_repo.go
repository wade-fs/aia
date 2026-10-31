package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/storage/sqlite"
)

type InvestmentRepository struct {
	db *sqlite.DB
}

func NewInvestmentRepository(db *sqlite.DB) *InvestmentRepository {
	return &InvestmentRepository{db: db}
}

// EnsureDefaultProfile creates a default profile and initial snapshot if none exist.
func (r *InvestmentRepository) EnsureDefaultProfile(ctx context.Context) (*investment.PortfolioProfile, error) {
	row := r.db.RawDB().QueryRowContext(ctx, "SELECT id, name, market, currency, capital_limit, capital_limit_basis, auto_trading_enabled, configuration_json, created_at, updated_at FROM investment_portfolio_profiles LIMIT 1")
	p, err := scanProfile(row)
	if err == nil {
		return p, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create default Taiwan portfolio
	now := time.Now().UTC()
	defaultProfile := &investment.PortfolioProfile{
		ID:                 uuid.New().String(),
		Name:               "Taiwan Stock Strategy",
		Market:             "TW",
		Currency:           "TWD",
		CapitalLimit:       1000000,
		CapitalLimitBasis:  "TOTAL_EQUITY",
		AutoTradingEnabled: false,
		ConfigurationJSON:  `{"broker_discount": 0.6, "default_lot_size": 1000}`,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	err = r.db.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO investment_portfolio_profiles (
				id, name, market, currency, capital_limit, capital_limit_basis,
				auto_trading_enabled, configuration_json, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, defaultProfile.ID, defaultProfile.Name, defaultProfile.Market, defaultProfile.Currency,
			defaultProfile.CapitalLimit, defaultProfile.CapitalLimitBasis, 0, defaultProfile.ConfigurationJSON,
			defaultProfile.CreatedAt.Format(time.RFC3339), defaultProfile.UpdatedAt.Format(time.RFC3339))
		if err != nil {
			return err
		}

		// Initial constraints
		constraints := []investment.Constraint{
			{ID: uuid.New().String(), PortfolioProfileID: defaultProfile.ID, ConstraintType: "max_position_pct", ConstraintValue: "0.25", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New().String(), PortfolioProfileID: defaultProfile.ID, ConstraintType: "min_cash_pct", ConstraintValue: "0.10", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New().String(), PortfolioProfileID: defaultProfile.ID, ConstraintType: "max_trade_amount", ConstraintValue: "250000", Enabled: true, CreatedAt: now, UpdatedAt: now},
		}
		for _, c := range constraints {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO investment_portfolio_constraints (id, portfolio_profile_id, constraint_type, constraint_value, enabled, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, c.ID, c.PortfolioProfileID, c.ConstraintType, c.ConstraintValue, 1, c.CreatedAt.Format(time.RFC3339), c.UpdatedAt.Format(time.RFC3339))
			if err != nil {
				return err
			}
		}

		// Initial Seed Instruments (popular TW stocks)
		instruments := []investment.Instrument{
			{ID: uuid.New().String(), Symbol: "2330", Exchange: "TWSE", AssetType: "EQUITY", Name: "台積電", Currency: "TWD", IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New().String(), Symbol: "2317", Exchange: "TWSE", AssetType: "EQUITY", Name: "鴻海", Currency: "TWD", IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New().String(), Symbol: "2454", Exchange: "TWSE", AssetType: "EQUITY", Name: "聯發科", Currency: "TWD", IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New().String(), Symbol: "0050", Exchange: "TWSE", AssetType: "ETF", Name: "元大台灣50", Currency: "TWD", IsActive: true, CreatedAt: now, UpdatedAt: now},
		}
		for _, inst := range instruments {
			_, _ = tx.ExecContext(ctx, `
				INSERT OR IGNORE INTO investment_instruments (id, symbol, exchange, asset_type, name, currency, is_active, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, inst.ID, inst.Symbol, inst.Exchange, inst.AssetType, inst.Name, inst.Currency, 1, inst.CreatedAt.Format(time.RFC3339), inst.UpdatedAt.Format(time.RFC3339))
		}

		// Initial Snapshot
		initialSnap := investment.PortfolioSnapshot{
			ID:                  uuid.New().String(),
			PortfolioProfileID:  defaultProfile.ID,
			AsOf:                now,
			Source:              "INITIAL_SEED",
			Currency:            "TWD",
			AvailableCash:       600000,
			TotalPortfolioValue: 1000000,
			ValidationStatus:    "VALIDATED",
			MetadataJSON:        `{"notes": "Initial setup"}`,
			CreatedAt:           now,
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO investment_portfolio_snapshots (id, portfolio_profile_id, as_of, source, currency, available_cash, total_portfolio_value, validation_status, metadata_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, initialSnap.ID, initialSnap.PortfolioProfileID, initialSnap.AsOf.Format(time.RFC3339), initialSnap.Source, initialSnap.Currency, initialSnap.AvailableCash, initialSnap.TotalPortfolioValue, initialSnap.ValidationStatus, initialSnap.MetadataJSON, initialSnap.CreatedAt.Format(time.RFC3339))
		if err != nil {
			return err
		}

		// Seed initial position: 400 shares of 2330 (TSMC) at NT$ 1,000 = NT$ 400,000
		var tsmcID string
		_ = tx.QueryRowContext(ctx, "SELECT id FROM investment_instruments WHERE symbol = '2330'").Scan(&tsmcID)
		if tsmcID != "" {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO investment_portfolio_positions (id, portfolio_snapshot_id, instrument_id, quantity, average_cost, last_known_price, market_value, currency, metadata_json)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, uuid.New().String(), initialSnap.ID, tsmcID, 400, 950.0, 1000.0, 400000, "TWD", `{}`)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to seed default portfolio profile: %w", err)
	}

	return defaultProfile, nil
}

func scanProfile(scanner interface{ Scan(dest ...any) error }) (*investment.PortfolioProfile, error) {
	var p investment.PortfolioProfile
	var autoTrading int
	var ca, ua string
	err := scanner.Scan(
		&p.ID, &p.Name, &p.Market, &p.Currency, &p.CapitalLimit, &p.CapitalLimitBasis,
		&autoTrading, &p.ConfigurationJSON, &ca, &ua,
	)
	if err != nil {
		return nil, err
	}
	p.AutoTradingEnabled = autoTrading == 1
	p.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &p, nil
}

func (r *InvestmentRepository) GetProfile(ctx context.Context, id domain.ID) (*investment.PortfolioProfile, error) {
	row := r.db.RawDB().QueryRowContext(ctx, "SELECT id, name, market, currency, capital_limit, capital_limit_basis, auto_trading_enabled, configuration_json, created_at, updated_at FROM investment_portfolio_profiles WHERE id = ?", id)
	return scanProfile(row)
}

func (r *InvestmentRepository) GetLatestSnapshot(ctx context.Context, profileID domain.ID) (*investment.PortfolioSnapshot, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, portfolio_profile_id, as_of, source, currency, available_cash, total_portfolio_value, validation_status, metadata_json, created_at
		FROM investment_portfolio_snapshots WHERE portfolio_profile_id = ? ORDER BY as_of DESC LIMIT 1
	`, profileID)

	var s investment.PortfolioSnapshot
	var asOf, ca string
	err := row.Scan(&s.ID, &s.PortfolioProfileID, &asOf, &s.Source, &s.Currency, &s.AvailableCash, &s.TotalPortfolioValue, &s.ValidationStatus, &s.MetadataJSON, &ca)
	if err != nil {
		return nil, err
	}
	s.AsOf, _ = time.Parse(time.RFC3339, asOf)
	s.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	return &s, nil
}

func (r *InvestmentRepository) ListPositions(ctx context.Context, snapshotID domain.ID) ([]investment.Position, error) {
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, portfolio_snapshot_id, instrument_id, quantity, average_cost, last_known_price, market_value, currency, metadata_json
		FROM investment_portfolio_positions WHERE portfolio_snapshot_id = ?
	`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.Position
	for rows.Next() {
		var p investment.Position
		if err := rows.Scan(&p.ID, &p.PortfolioSnapshotID, &p.InstrumentID, &p.Quantity, &p.AverageCost, &p.LastKnownPrice, &p.MarketValue, &p.Currency, &p.MetadataJSON); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) ListConstraints(ctx context.Context, profileID domain.ID) ([]investment.Constraint, error) {
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, portfolio_profile_id, constraint_type, constraint_value, enabled, created_at, updated_at
		FROM investment_portfolio_constraints WHERE portfolio_profile_id = ?
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.Constraint
	for rows.Next() {
		var c investment.Constraint
		var en int
		var ca, ua string
		if err := rows.Scan(&c.ID, &c.PortfolioProfileID, &c.ConstraintType, &c.ConstraintValue, &en, &ca, &ua); err != nil {
			return nil, err
		}
		c.Enabled = en == 1
		c.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) GetInstrumentBySymbol(ctx context.Context, exchange, symbol string) (*investment.Instrument, error) {
	query := "SELECT id, symbol, exchange, asset_type, name, currency, is_active, created_at, updated_at FROM investment_instruments WHERE symbol = ?"
	args := []any{symbol}
	if exchange != "" {
		query += " AND exchange = ?"
		args = append(args, exchange)
	}
	row := r.db.RawDB().QueryRowContext(ctx, query, args...)

	var inst investment.Instrument
	var act int
	var ca, ua string
	err := row.Scan(&inst.ID, &inst.Symbol, &inst.Exchange, &inst.AssetType, &inst.Name, &inst.Currency, &act, &ca, &ua)
	if err != nil {
		return nil, err
	}
	inst.IsActive = act == 1
	inst.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	inst.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &inst, nil
}

func (r *InvestmentRepository) ListInstruments(ctx context.Context) ([]investment.Instrument, error) {
	rows, err := r.db.RawDB().QueryContext(ctx, "SELECT id, symbol, exchange, asset_type, name, currency, is_active, created_at, updated_at FROM investment_instruments ORDER BY symbol ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.Instrument
	for rows.Next() {
		var inst investment.Instrument
		var act int
		var ca, ua string
		if err := rows.Scan(&inst.ID, &inst.Symbol, &inst.Exchange, &inst.AssetType, &inst.Name, &inst.Currency, &act, &ca, &ua); err != nil {
			return nil, err
		}
		inst.IsActive = act == 1
		inst.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		inst.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		list = append(list, inst)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) SaveDecision(ctx context.Context, dec *investment.Decision) error {
	query := `
		INSERT INTO investment_decisions (
			id, portfolio_profile_id, instrument_id, portfolio_snapshot_id, task_id, research_report_id,
			decision_type, confidence, reasoning_json, risks_json, invalidation_conditions_json,
			expected_horizon, strategy_version, action_result_json, raw_input_json, source, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.RawDB().ExecContext(ctx, query,
		dec.ID, dec.PortfolioProfileID, dec.InstrumentID, dec.PortfolioSnapshotID, dec.TaskID, dec.ResearchReportID,
		string(dec.DecisionType), dec.Confidence, dec.ReasoningJSON, dec.RisksJSON, dec.InvalidationConditionsJSON,
		dec.ExpectedHorizon, dec.StrategyVersion, dec.ActionResultJSON, dec.RawInputJSON, dec.Source, dec.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *InvestmentRepository) ListDecisions(ctx context.Context, profileID domain.ID, limit int) ([]investment.Decision, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, portfolio_profile_id, instrument_id, portfolio_snapshot_id, task_id, research_report_id,
			decision_type, confidence, reasoning_json, risks_json, invalidation_conditions_json,
			expected_horizon, strategy_version, action_result_json, raw_input_json, source, created_at
		FROM investment_decisions WHERE portfolio_profile_id = ? ORDER BY created_at DESC LIMIT ?
	`, profileID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.Decision
	for rows.Next() {
		var dec investment.Decision
		var dt, ca string
		if err := rows.Scan(
			&dec.ID, &dec.PortfolioProfileID, &dec.InstrumentID, &dec.PortfolioSnapshotID, &dec.TaskID, &dec.ResearchReportID,
			&dt, &dec.Confidence, &dec.ReasoningJSON, &dec.RisksJSON, &dec.InvalidationConditionsJSON,
			&dec.ExpectedHorizon, &dec.StrategyVersion, &dec.ActionResultJSON, &dec.RawInputJSON, &dec.Source, &ca,
		); err != nil {
			return nil, err
		}
		dec.DecisionType = investment.DecisionType(dt)
		dec.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		list = append(list, dec)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) SaveDecisionRejection(ctx context.Context, rej *investment.DecisionRejection) error {
	query := `
		INSERT INTO investment_decision_rejections (id, portfolio_profile_id, task_id, raw_output, errors_json, source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.RawDB().ExecContext(ctx, query,
		rej.ID, rej.PortfolioProfileID, rej.TaskID, rej.RawOutput, rej.ErrorsJSON, rej.Source, rej.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *InvestmentRepository) ListDecisionRejections(ctx context.Context, profileID domain.ID, limit int) ([]investment.DecisionRejection, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, portfolio_profile_id, task_id, raw_output, errors_json, source, created_at
		FROM investment_decision_rejections WHERE portfolio_profile_id = ? ORDER BY created_at DESC LIMIT ?
	`, profileID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.DecisionRejection
	for rows.Next() {
		var rej investment.DecisionRejection
		var ca string
		if err := rows.Scan(&rej.ID, &rej.PortfolioProfileID, &rej.TaskID, &rej.RawOutput, &rej.ErrorsJSON, &rej.Source, &ca); err != nil {
			return nil, err
		}
		rej.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		list = append(list, rej)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) SaveTradeProposal(ctx context.Context, p *investment.TradeProposal) error {
	query := `
		INSERT INTO investment_trade_proposals (
			id, portfolio_profile_id, decision_id, portfolio_snapshot_id, allocation_proposal_id, instrument_id,
			action, quantity, estimated_price, estimated_gross_amount, estimated_total_cost, proposal_json,
			status, status_reason, created_at, expires_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.RawDB().ExecContext(ctx, query,
		p.ID, p.PortfolioProfileID, p.DecisionID, p.PortfolioSnapshotID, p.AllocationProposalID, p.InstrumentID,
		string(p.Action), p.Quantity, p.EstimatedPrice, p.EstimatedGrossAmount, p.EstimatedTotalCost, p.ProposalJSON,
		string(p.Status), p.StatusReason, p.CreatedAt.Format(time.RFC3339), p.ExpiresAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *InvestmentRepository) GetTradeProposal(ctx context.Context, id domain.ID) (*investment.TradeProposal, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, portfolio_profile_id, decision_id, portfolio_snapshot_id, allocation_proposal_id, instrument_id,
			action, quantity, estimated_price, estimated_gross_amount, estimated_total_cost, proposal_json,
			status, status_reason, created_at, expires_at, updated_at
		FROM investment_trade_proposals WHERE id = ?
	`, id)

	var p investment.TradeProposal
	var act, st, ca, ea, ua string
	err := row.Scan(
		&p.ID, &p.PortfolioProfileID, &p.DecisionID, &p.PortfolioSnapshotID, &p.AllocationProposalID, &p.InstrumentID,
		&act, &p.Quantity, &p.EstimatedPrice, &p.EstimatedGrossAmount, &p.EstimatedTotalCost, &p.ProposalJSON,
		&st, &p.StatusReason, &ca, &ea, &ua,
	)
	if err != nil {
		return nil, err
	}
	p.Action = investment.DecisionType(act)
	p.Status = investment.ProposalStatus(st)
	p.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	p.ExpiresAt, _ = time.Parse(time.RFC3339, ea)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &p, nil
}

func (r *InvestmentRepository) ListTradeProposals(ctx context.Context, profileID domain.ID, status investment.ProposalStatus, limit int) ([]investment.TradeProposal, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, portfolio_profile_id, decision_id, portfolio_snapshot_id, allocation_proposal_id, instrument_id,
			action, quantity, estimated_price, estimated_gross_amount, estimated_total_cost, proposal_json,
			status, status_reason, created_at, expires_at, updated_at
		FROM investment_trade_proposals WHERE portfolio_profile_id = ?
	`
	args := []any{profileID}
	if status != "" {
		query += " AND status = ?"
		args = append(args, string(status))
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.TradeProposal
	for rows.Next() {
		var p investment.TradeProposal
		var act, st, ca, ea, ua string
		if err := rows.Scan(
			&p.ID, &p.PortfolioProfileID, &p.DecisionID, &p.PortfolioSnapshotID, &p.AllocationProposalID, &p.InstrumentID,
			&act, &p.Quantity, &p.EstimatedPrice, &p.EstimatedGrossAmount, &p.EstimatedTotalCost, &p.ProposalJSON,
			&st, &p.StatusReason, &ca, &ea, &ua,
		); err != nil {
			return nil, err
		}
		p.Action = investment.DecisionType(act)
		p.Status = investment.ProposalStatus(st)
		p.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		p.ExpiresAt, _ = time.Parse(time.RFC3339, ea)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) UpdateTradeProposalStatus(ctx context.Context, id domain.ID, status investment.ProposalStatus, reason string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.RawDB().ExecContext(ctx, "UPDATE investment_trade_proposals SET status = ?, status_reason = ?, updated_at = ? WHERE id = ?", string(status), reason, now, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return sql.ErrNoRows
	}
	return nil
}
