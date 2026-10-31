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

// UpdateCapitalLimit updates the capital ceiling of a portfolio profile.
func (r *InvestmentRepository) UpdateCapitalLimit(ctx context.Context, profileID domain.ID, limit int64) error {
	now := time.Now().UTC()
	_, err := r.db.RawDB().ExecContext(ctx, `
		UPDATE investment_portfolio_profiles
		SET capital_limit = ?, updated_at = ?
		WHERE id = ?
	`, limit, now.Format(time.RFC3339), profileID)
	return err
}

// UpdateAvailableCash creates a new immutable point-in-time snapshot with adjusted available cash,
// copying existing positions and recomputing total portfolio value.
func (r *InvestmentRepository) UpdateAvailableCash(ctx context.Context, profileID domain.ID, newCash int64, note string) (*investment.PortfolioSnapshot, error) {
	snap, err := r.GetLatestSnapshot(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest snapshot: %w", err)
	}
	positions, err := r.ListPositions(ctx, snap.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list positions: %w", err)
	}

	var totalPositionsValue int64 = 0
	for _, p := range positions {
		totalPositionsValue += p.MarketValue
	}

	now := time.Now().UTC()
	newSnap := &investment.PortfolioSnapshot{
		ID:                  uuid.New().String(),
		PortfolioProfileID:  profileID,
		AsOf:                now,
		Source:              "MANUAL_ADJUSTMENT",
		Currency:            snap.Currency,
		AvailableCash:       newCash,
		TotalPortfolioValue: newCash + totalPositionsValue,
		ValidationStatus:    "VALIDATED",
		MetadataJSON:        fmt.Sprintf(`{"reason": %q, "previous_cash": %d}`, note, snap.AvailableCash),
		CreatedAt:           now,
	}

	err = r.db.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO investment_portfolio_snapshots (
				id, portfolio_profile_id, as_of, source, currency,
				available_cash, total_portfolio_value, validation_status, metadata_json, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, newSnap.ID, newSnap.PortfolioProfileID, newSnap.AsOf.Format(time.RFC3339),
			newSnap.Source, newSnap.Currency, newSnap.AvailableCash, newSnap.TotalPortfolioValue,
			newSnap.ValidationStatus, newSnap.MetadataJSON, newSnap.CreatedAt.Format(time.RFC3339))
		if err != nil {
			return err
		}

		// Copy over positions to the new snapshot
		for _, pos := range positions {
			newPosID := uuid.New().String()
			_, err = tx.ExecContext(ctx, `
				INSERT INTO investment_portfolio_positions (
					id, portfolio_snapshot_id, instrument_id, quantity, average_cost,
					last_known_price, market_value, currency, metadata_json
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, newPosID, newSnap.ID, pos.InstrumentID, pos.Quantity, pos.AverageCost,
				pos.LastKnownPrice, pos.MarketValue, pos.Currency, pos.MetadataJSON)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save new snapshot: %w", err)
	}

	return newSnap, nil
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
		FROM investment_trade_proposals
		WHERE id = ? OR (length(?) >= 6 AND id LIKE ? || '%')
		LIMIT 1
	`, id, id, id)

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
	res, err := r.db.RawDB().ExecContext(ctx, `
		UPDATE investment_trade_proposals
		SET status = ?, status_reason = ?, updated_at = ?
		WHERE id = ? OR (length(?) >= 6 AND id LIKE ? || '%')
	`, string(status), reason, now, id, id, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// EnsureDefaultDataSources provisions built-in Taiwan financial and news sources (AIA-05 §6).
func (r *InvestmentRepository) EnsureDefaultDataSources(ctx context.Context) error {
	sources := []investment.DataSource{
		{
			ID:                 "ds-twse-openapi",
			SourceName:         "TWSE_OPENAPI",
			SourceType:         "OFFICIAL_EXCHANGE",
			BaseURL:            "https://openapi.twse.com.tw/v1",
			DocumentationURL:   "https://openapi.twse.com.tw/",
			AuthenticationType: "NONE",
			ConfigurationJSON:  `{"rate_limit_rpm": 60}`,
			Status:             "ACTIVE",
			HealthStatus:       "HEALTHY",
		},
		{
			ID:                 "ds-mops-announcements",
			SourceName:         "MOPS_ANNOUNCEMENTS",
			SourceType:         "GOVERNMENT_MOPS",
			BaseURL:            "https://mops.twse.com.tw",
			DocumentationURL:   "https://mops.twse.com.tw/mops/web/index",
			AuthenticationType: "NONE",
			ConfigurationJSON:  `{"rate_limit_rpm": 30}`,
			Status:             "ACTIVE",
			HealthStatus:       "HEALTHY",
		},
		{
			ID:                 "ds-tpex-openapi",
			SourceName:         "TPEX_OPENAPI",
			SourceType:         "OFFICIAL_EXCHANGE",
			BaseURL:            "https://www.tpex.org.tw/openapi/v1",
			DocumentationURL:   "https://www.tpex.org.tw/openapi/",
			AuthenticationType: "NONE",
			ConfigurationJSON:  `{"rate_limit_rpm": 60}`,
			Status:             "ACTIVE",
			HealthStatus:       "HEALTHY",
		},
		{
			ID:                 "ds-market-news",
			SourceName:         "FINANCIAL_NEWS_FEED",
			SourceType:         "NEWS_RSS",
			BaseURL:            "https://news.cnyes.com",
			DocumentationURL:   "https://news.cnyes.com",
			AuthenticationType: "NONE",
			ConfigurationJSON:  `{"categories": ["tw_stock", "headline"]}`,
			Status:             "ACTIVE",
			HealthStatus:       "HEALTHY",
		},
	}

	now := time.Now().UTC()
	for _, s := range sources {
		_, err := r.db.RawDB().ExecContext(ctx, `
			INSERT OR IGNORE INTO investment_data_sources (
				id, source_name, source_type, base_url, documentation_url,
				authentication_type, configuration_json, status, health_status,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.ID, s.SourceName, s.SourceType, s.BaseURL, s.DocumentationURL,
			s.AuthenticationType, s.ConfigurationJSON, s.Status, s.HealthStatus,
			now.Format(time.RFC3339), now.Format(time.RFC3339))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *InvestmentRepository) ListDataSources(ctx context.Context) ([]investment.DataSource, error) {
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, source_name, source_type, base_url, documentation_url, authentication_type,
			configuration_json, status, health_status, last_success_at, last_failure_at, created_at, updated_at
		FROM investment_data_sources ORDER BY source_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.DataSource
	for rows.Next() {
		var d investment.DataSource
		var ls, lf sql.NullString
		var ca, ua string
		if err := rows.Scan(
			&d.ID, &d.SourceName, &d.SourceType, &d.BaseURL, &d.DocumentationURL, &d.AuthenticationType,
			&d.ConfigurationJSON, &d.Status, &d.HealthStatus, &ls, &lf, &ca, &ua,
		); err != nil {
			return nil, err
		}
		if ls.Valid {
			t, _ := time.Parse(time.RFC3339, ls.String)
			d.LastSuccessAt = &t
		}
		if lf.Valid {
			t, _ := time.Parse(time.RFC3339, lf.String)
			d.LastFailureAt = &t
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		d.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) SaveMarketData(ctx context.Context, md *investment.MarketData) error {
	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO investment_market_data (
			id, instrument_id, data_type, period, value_json, source_id, fetched_at, effective_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, md.ID, md.InstrumentID, md.DataType, md.Period, md.ValueJSON, md.SourceID,
		md.FetchedAt.Format(time.RFC3339), md.EffectiveAt.Format(time.RFC3339))
	return err
}

func (r *InvestmentRepository) GetLatestMarketData(ctx context.Context, instrumentID domain.ID) (*investment.MarketData, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, instrument_id, data_type, period, value_json, source_id, fetched_at, effective_at
		FROM investment_market_data
		WHERE instrument_id = ?
		ORDER BY effective_at DESC LIMIT 1
	`, instrumentID)
	var md investment.MarketData
	var fa, ea string
	if err := row.Scan(&md.ID, &md.InstrumentID, &md.DataType, &md.Period, &md.ValueJSON, &md.SourceID, &fa, &ea); err != nil {
		return nil, err
	}
	md.FetchedAt, _ = time.Parse(time.RFC3339, fa)
	md.EffectiveAt, _ = time.Parse(time.RFC3339, ea)
	return &md, nil
}

func (r *InvestmentRepository) ListHistoricalMarketData(ctx context.Context, instrumentID domain.ID, limit int) ([]investment.MarketData, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, instrument_id, data_type, period, value_json, source_id, fetched_at, effective_at
		FROM investment_market_data
		WHERE instrument_id = ?
		ORDER BY effective_at DESC LIMIT ?
	`, instrumentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.MarketData
	for rows.Next() {
		var md investment.MarketData
		var fa, ea string
		if err := rows.Scan(&md.ID, &md.InstrumentID, &md.DataType, &md.Period, &md.ValueJSON, &md.SourceID, &fa, &ea); err != nil {
			return nil, err
		}
		md.FetchedAt, _ = time.Parse(time.RFC3339, fa)
		md.EffectiveAt, _ = time.Parse(time.RFC3339, ea)
		list = append(list, md)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) SaveRawDocument(ctx context.Context, doc *investment.RawDocument) error {
	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO investment_raw_documents (
			id, source_id, instrument_id, source_record_id, source_url, title, summary,
			content_reference, content_hash, content_type, published_at, fetched_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, doc.ID, doc.SourceID, doc.InstrumentID, doc.SourceRecordID, doc.SourceURL, doc.Title, doc.Summary,
		doc.ContentReference, doc.ContentHash, doc.ContentType, doc.PublishedAt.Format(time.RFC3339),
		doc.FetchedAt.Format(time.RFC3339), doc.MetadataJSON)
	return err
}

func (r *InvestmentRepository) ListRawDocuments(ctx context.Context, instrumentID domain.ID, limit int) ([]investment.RawDocument, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows *sql.Rows
	var err error
	if instrumentID != "" {
		rows, err = r.db.RawDB().QueryContext(ctx, `
			SELECT id, source_id, instrument_id, source_record_id, source_url, title, summary,
				content_reference, content_hash, content_type, published_at, fetched_at, metadata_json
			FROM investment_raw_documents WHERE instrument_id = ?
			ORDER BY published_at DESC LIMIT ?
		`, instrumentID, limit)
	} else {
		rows, err = r.db.RawDB().QueryContext(ctx, `
			SELECT id, source_id, instrument_id, source_record_id, source_url, title, summary,
				content_reference, content_hash, content_type, published_at, fetched_at, metadata_json
			FROM investment_raw_documents
			ORDER BY published_at DESC LIMIT ?
		`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.RawDocument
	for rows.Next() {
		var doc investment.RawDocument
		var pa, fa string
		if err := rows.Scan(
			&doc.ID, &doc.SourceID, &doc.InstrumentID, &doc.SourceRecordID, &doc.SourceURL, &doc.Title, &doc.Summary,
			&doc.ContentReference, &doc.ContentHash, &doc.ContentType, &pa, &fa, &doc.MetadataJSON,
		); err != nil {
			return nil, err
		}
		doc.PublishedAt, _ = time.Parse(time.RFC3339, pa)
		doc.FetchedAt, _ = time.Parse(time.RFC3339, fa)
		list = append(list, doc)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) SavePrediction(ctx context.Context, p *investment.Prediction) error {
	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO investment_predictions (
			id, decision_id, instrument_id, prediction_type, target, expected_direction,
			probability, target_value, base_price, evaluation_horizon, due_at, strategy_version, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.DecisionID, p.InstrumentID, p.PredictionType, p.Target, p.ExpectedDirection,
		p.Probability, p.TargetValue, p.BasePrice, p.EvaluationHorizon, p.DueAt.Format(time.RFC3339),
		p.StrategyVersion, string(p.Status), p.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *InvestmentRepository) GetPrediction(ctx context.Context, id domain.ID) (*investment.Prediction, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, decision_id, instrument_id, prediction_type, target, expected_direction,
			probability, target_value, base_price, evaluation_horizon, due_at, strategy_version, status, created_at
		FROM investment_predictions WHERE id = ?
	`, id)
	var p investment.Prediction
	var da, ca, st string
	if err := row.Scan(
		&p.ID, &p.DecisionID, &p.InstrumentID, &p.PredictionType, &p.Target, &p.ExpectedDirection,
		&p.Probability, &p.TargetValue, &p.BasePrice, &p.EvaluationHorizon, &da, &p.StrategyVersion, &st, &ca,
	); err != nil {
		return nil, err
	}
	p.DueAt, _ = time.Parse(time.RFC3339, da)
	p.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	p.Status = investment.PredictionStatus(st)
	return &p, nil
}

func (r *InvestmentRepository) ListPredictions(ctx context.Context, status investment.PredictionStatus, limit int) ([]investment.Prediction, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, decision_id, instrument_id, prediction_type, target, expected_direction,
			probability, target_value, base_price, evaluation_horizon, due_at, strategy_version, status, created_at
		FROM investment_predictions
	`
	args := []any{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, string(status))
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.Prediction
	for rows.Next() {
		var p investment.Prediction
		var da, ca, st string
		if err := rows.Scan(
			&p.ID, &p.DecisionID, &p.InstrumentID, &p.PredictionType, &p.Target, &p.ExpectedDirection,
			&p.Probability, &p.TargetValue, &p.BasePrice, &p.EvaluationHorizon, &da, &p.StrategyVersion, &st, &ca,
		); err != nil {
			return nil, err
		}
		p.DueAt, _ = time.Parse(time.RFC3339, da)
		p.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		p.Status = investment.PredictionStatus(st)
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) SavePredictionOutcome(ctx context.Context, oc *investment.PredictionOutcome) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO investment_prediction_outcomes (
				id, prediction_id, actual_value, actual_direction, outcome_status,
				brier_score, evaluation_source_id, evaluated_at, details_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, oc.ID, oc.PredictionID, oc.ActualValue, oc.ActualDirection, oc.OutcomeStatus,
			oc.BrierScore, oc.EvaluationSourceID, oc.EvaluatedAt.Format(time.RFC3339), oc.DetailsJSON)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE investment_predictions SET status = ? WHERE id = ?", string(investment.PredictionEvaluated), oc.PredictionID)
		return err
	})
}

func (r *InvestmentRepository) ListPredictionOutcomes(ctx context.Context, limit int) ([]investment.PredictionOutcome, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, prediction_id, actual_value, actual_direction, outcome_status,
			brier_score, evaluation_source_id, evaluated_at, details_json
		FROM investment_prediction_outcomes
		ORDER BY evaluated_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.PredictionOutcome
	for rows.Next() {
		var oc investment.PredictionOutcome
		var ea string
		if err := rows.Scan(
			&oc.ID, &oc.PredictionID, &oc.ActualValue, &oc.ActualDirection, &oc.OutcomeStatus,
			&oc.BrierScore, &oc.EvaluationSourceID, &ea, &oc.DetailsJSON,
		); err != nil {
			return nil, err
		}
		oc.EvaluatedAt, _ = time.Parse(time.RFC3339, ea)
		list = append(list, oc)
	}
	return list, rows.Err()
}

func (r *InvestmentRepository) SaveEvaluationResult(ctx context.Context, ev *investment.EvaluationResult) error {
	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO investment_evaluation_results (
			id, evaluation_type, period_start, period_end, sample_count, metrics_json, strategy_version, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, ev.ID, ev.EvaluationType, ev.PeriodStart.Format(time.RFC3339), ev.PeriodEnd.Format(time.RFC3339),
		ev.SampleCount, ev.MetricsJSON, ev.StrategyVersion, ev.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *InvestmentRepository) ListEvaluationResults(ctx context.Context, limit int) ([]investment.EvaluationResult, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, evaluation_type, period_start, period_end, sample_count, metrics_json, strategy_version, created_at
		FROM investment_evaluation_results ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.EvaluationResult
	for rows.Next() {
		var ev investment.EvaluationResult
		var ps, pe, ca string
		if err := rows.Scan(
			&ev.ID, &ev.EvaluationType, &ps, &pe, &ev.SampleCount, &ev.MetricsJSON, &ev.StrategyVersion, &ca,
		); err != nil {
			return nil, err
		}
		ev.PeriodStart, _ = time.Parse(time.RFC3339, ps)
		ev.PeriodEnd, _ = time.Parse(time.RFC3339, pe)
		ev.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		list = append(list, ev)
	}
	return list, rows.Err()
}
