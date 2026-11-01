package investment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/autonomy/clock"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type AuditWriter interface {
	Append(ctx context.Context, entry *repository.AuditLogEntry) error
}

type Service struct {
	repo         *repository.InvestmentRepository
	auditRepo    AuditWriter
	bus          eventbus.Publisher
	costCfg      investment.CostConfig
	dataProvider MarketDataProvider
}

func NewService(repo *repository.InvestmentRepository, auditRepo AuditWriter, bus eventbus.Publisher) *Service {
	return &Service{
		repo:         repo,
		auditRepo:    auditRepo,
		bus:          bus,
		costCfg:      investment.DefaultTWCostConfig(),
		dataProvider: NewTWSEMarketProvider("", ""),
	}
}

func (s *Service) SetDataProvider(p MarketDataProvider) {
	if p != nil {
		s.dataProvider = p
	}
}

// EnsureDefaultProfile guarantees an active investment portfolio is initialized.
func (s *Service) EnsureDefaultProfile(ctx context.Context) (*investment.PortfolioProfile, error) {
	return s.repo.EnsureDefaultProfile(ctx)
}

func (s *Service) GetProfile(ctx context.Context) (*investment.PortfolioProfile, error) {
	return s.repo.EnsureDefaultProfile(ctx)
}

func (s *Service) SetCapitalLimit(ctx context.Context, limit int64) error {
	if limit < 0 {
		return fmt.Errorf("capital limit must be non-negative")
	}
	prof, err := s.GetProfile(ctx)
	if err != nil {
		return err
	}
	return s.repo.UpdateCapitalLimit(ctx, prof.ID, limit)
}

func (s *Service) SetAvailableCash(ctx context.Context, amount int64, reason string) (*investment.PortfolioSnapshot, error) {
	if amount < 0 {
		return nil, fmt.Errorf("available cash must be non-negative")
	}
	prof, err := s.GetProfile(ctx)
	if err != nil {
		return nil, err
	}
	if reason == "" {
		reason = "Manual cash update"
	}
	return s.repo.UpdateAvailableCash(ctx, prof.ID, amount, reason)
}

func (s *Service) GetSnapshot(ctx context.Context) (*investment.PortfolioSnapshot, []investment.Position, error) {
	prof, err := s.GetProfile(ctx)
	if err != nil {
		return nil, nil, err
	}
	snap, err := s.repo.GetLatestSnapshot(ctx, prof.ID)
	if err != nil {
		return nil, nil, err
	}
	positions, err := s.repo.ListPositions(ctx, snap.ID)
	if err != nil {
		return nil, nil, err
	}
	return snap, positions, nil
}

func (s *Service) ListInstruments(ctx context.Context) ([]investment.Instrument, error) {
	return s.repo.ListInstruments(ctx)
}

func (s *Service) ListConstraints(ctx context.Context) ([]investment.Constraint, error) {
	prof, err := s.GetProfile(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListConstraints(ctx, prof.ID)
}

func (s *Service) ListDecisions(ctx context.Context, limit int) ([]investment.Decision, error) {
	prof, err := s.GetProfile(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListDecisions(ctx, prof.ID, limit)
}

func (s *Service) ListRejections(ctx context.Context, limit int) ([]investment.DecisionRejection, error) {
	prof, err := s.GetProfile(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListDecisionRejections(ctx, prof.ID, limit)
}

func (s *Service) ListProposals(ctx context.Context, status investment.ProposalStatus, limit int) ([]investment.TradeProposal, error) {
	prof, err := s.GetProfile(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListTradeProposals(ctx, prof.ID, status, limit)
}

// ProcessAgyDecision receives raw investment reasoning from Antigravity CLI / LLM,
// validates all domain rules and constraints in Go, calculates fees/taxes and position sizing,
// and produces a valid TradeProposal (or records a rejection).
func (s *Service) ProcessAgyDecision(ctx context.Context, input investment.AgyDecisionInput, price float64) (*investment.TradeProposal, error) {
	prof, err := s.GetProfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	snap, positions, err := s.GetSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	rawInputJSON, _ := json.Marshal(input)
	now := clock.Now(ctx)

	// 1. Validate Instrument Symbol
	symbol := strings.TrimSpace(input.Symbol)
	inst, err := s.repo.GetInstrumentBySymbol(ctx, input.Exchange, symbol)
	if err != nil || inst == nil || !inst.IsActive {
		rej := &investment.DecisionRejection{
			ID:                 uuid.New().String(),
			PortfolioProfileID: prof.ID,
			RawOutput:          string(rawInputJSON),
			ErrorsJSON:         fmt.Sprintf(`["unrecognized or inactive instrument symbol: %s"]`, symbol),
			Source:             "VALIDATION_GO",
			CreatedAt:          now,
		}
		_ = s.repo.SaveDecisionRejection(ctx, rej)
		return nil, fmt.Errorf("invalid instrument symbol: %s", symbol)
	}

	// 2. Validate Action
	action := strings.ToUpper(strings.TrimSpace(input.Action))
	if action != "BUY" && action != "SELL" && action != "HOLD" {
		rej := &investment.DecisionRejection{
			ID:                 uuid.New().String(),
			PortfolioProfileID: prof.ID,
			RawOutput:          string(rawInputJSON),
			ErrorsJSON:         fmt.Sprintf(`["unsupported action: %s"]`, input.Action),
			Source:             "VALIDATION_GO",
			CreatedAt:          now,
		}
		_ = s.repo.SaveDecisionRejection(ctx, rej)
		return nil, fmt.Errorf("unsupported action: %s", input.Action)
	}

	// 3. Validate Confidence (Minimum 0.50 required for actionable trade proposal)
	if input.Confidence < 0.50 {
		rej := &investment.DecisionRejection{
			ID:                 uuid.New().String(),
			PortfolioProfileID: prof.ID,
			RawOutput:          string(rawInputJSON),
			ErrorsJSON:         fmt.Sprintf(`["confidence %.2f is below threshold 0.50"]`, input.Confidence),
			Source:             "VALIDATION_GO",
			CreatedAt:          now,
		}
		_ = s.repo.SaveDecisionRejection(ctx, rej)
		return nil, fmt.Errorf("decision confidence %.2f is below actionable threshold (0.50)", input.Confidence)
	}

	// 4. Save Decision Record
	reasoningJSON, _ := json.Marshal(input.Reasoning)
	risksJSON, _ := json.Marshal(input.Risks)
	invalJSON, _ := json.Marshal(input.InvalidationRules)
	actionResJSON, _ := json.Marshal(map[string]any{"reference_price": price})

	dec := &investment.Decision{
		ID:                         uuid.New().String(),
		PortfolioProfileID:         prof.ID,
		InstrumentID:               inst.ID,
		PortfolioSnapshotID:        snap.ID,
		DecisionType:               investment.DecisionType(action),
		Confidence:                 input.Confidence,
		ReasoningJSON:              string(reasoningJSON),
		RisksJSON:                  string(risksJSON),
		InvalidationConditionsJSON: string(invalJSON),
		ExpectedHorizon:            input.ExpectedHorizon,
		StrategyVersion:            input.StrategyVersion,
		ActionResultJSON:           string(actionResJSON),
		RawInputJSON:               string(rawInputJSON),
		Source:                     "ANTIGRAVITY_LLM",
		CreatedAt:                  now,
	}

	if err := s.repo.SaveDecision(ctx, dec); err != nil {
		return nil, fmt.Errorf("failed to save decision: %w", err)
	}

	if dec.DecisionType == investment.DecisionHold {
		// HOLD does not generate a trade proposal
		if s.bus != nil {
			s.bus.Publish(eventbus.InvestmentUpdated, map[string]any{"type": "decision", "action": "HOLD", "symbol": symbol})
		}
		return nil, nil
	}

	// 5. Check constraints & calculate quantity and costs in Go
	constraints, err := s.repo.ListConstraints(ctx, prof.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list constraints: %w", err)
	}

	var currentQty int64 = 0
	for _, p := range positions {
		if p.InstrumentID == inst.ID {
			currentQty = p.Quantity
			break
		}
	}

	sizing := investment.EvaluateTradeLimits(
		prof,
		snap,
		constraints,
		dec.DecisionType,
		input.TargetAmount,
		price,
		currentQty,
		s.costCfg,
	)

	if sizing.Rejected {
		rej := &investment.DecisionRejection{
			ID:                 uuid.New().String(),
			PortfolioProfileID: prof.ID,
			RawOutput:          string(rawInputJSON),
			ErrorsJSON:         fmt.Sprintf(`["risk limits check failed: %s"]`, sizing.RejectionError),
			Source:             "LIMITS_CHECK_GO",
			CreatedAt:          now,
		}
		_ = s.repo.SaveDecisionRejection(ctx, rej)
		return nil, fmt.Errorf("trade rejected by investment limits: %s", sizing.RejectionError)
	}

	// 6. Generate TradeProposal
	proposalPayload := map[string]any{
		"symbol":           symbol,
		"instrument_name":  inst.Name,
		"action":           string(dec.DecisionType),
		"quantity":         sizing.Quantity,
		"estimated_price":  price,
		"gross_amount":     sizing.Cost.GrossAmount,
		"broker_fee":       sizing.Cost.BrokerFee,
		"transaction_tax":  sizing.Cost.TransactionTax,
		"total_cost":       sizing.Cost.TotalCost,
		"effective_rate":   sizing.Cost.EffectiveRate,
		"was_adjusted":     sizing.WasAdjusted,
		"adjustment_note":  sizing.AdjustmentNote,
		"confidence":       input.Confidence,
		"expected_horizon": input.ExpectedHorizon,
	}
	propJSON, _ := json.Marshal(proposalPayload)

	proposal := &investment.TradeProposal{
		ID:                   uuid.New().String(),
		PortfolioProfileID:   prof.ID,
		DecisionID:           dec.ID,
		PortfolioSnapshotID:  snap.ID,
		AllocationProposalID: "",
		InstrumentID:         inst.ID,
		Action:               dec.DecisionType,
		Quantity:             sizing.Quantity,
		EstimatedPrice:       price,
		EstimatedGrossAmount: sizing.Cost.GrossAmount,
		EstimatedTotalCost:   sizing.Cost.TotalCost,
		ProposalJSON:         string(propJSON),
		Status:               investment.ProposalPending,
		StatusReason:         sizing.AdjustmentNote,
		CreatedAt:            now,
		ExpiresAt:            now.Add(24 * time.Hour),
		UpdatedAt:            now,
	}

	if err := s.repo.SaveTradeProposal(ctx, proposal); err != nil {
		return nil, fmt.Errorf("failed to save trade proposal: %w", err)
	}

	// Write audit log
	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "AGENT",
			ActorID:      "investment-agent",
			Action:       "INVESTMENT_PROPOSAL_GENERATED",
			ResourceType: "INVESTMENT_PROPOSAL",
			ResourceID:   proposal.ID,
			Outcome:      "SUCCESS",
			Details:      json.RawMessage(propJSON),
			CreatedAt:    now,
		})
	}

	// Emit bus event
	if s.bus != nil {
		s.bus.Publish(eventbus.InvestmentUpdated, map[string]any{
			"type":        "trade_proposal",
			"proposal_id": proposal.ID,
			"symbol":      symbol,
			"action":      string(proposal.Action),
			"quantity":    proposal.Quantity,
			"total_cost":  proposal.EstimatedTotalCost,
		})
	}

	return proposal, nil
}

func (s *Service) ApproveProposal(ctx context.Context, proposalID domain.ID) error {
	p, err := s.repo.GetTradeProposal(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("proposal not found: %w", err)
	}
	if p.Status != investment.ProposalPending {
		return fmt.Errorf("proposal is not in PENDING state (current: %s)", p.Status)
	}

	if err := s.repo.UpdateTradeProposalStatus(ctx, proposalID, investment.ProposalApproved, "Approved by user via Web Console"); err != nil {
		return err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "USER",
			ActorID:      "console-user",
			Action:       "INVESTMENT_PROPOSAL_APPROVED",
			ResourceType: "INVESTMENT_PROPOSAL",
			ResourceID:   proposalID,
			Outcome:      "SUCCESS",
			Details:      json.RawMessage(fmt.Sprintf(`{"proposal_id": "%s", "action": "APPROVED"}`, proposalID)),
			CreatedAt:    time.Now().UTC(),
		})
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.InvestmentUpdated, map[string]any{
			"type":        "proposal_approved",
			"proposal_id": proposalID,
		})
	}

	return nil
}

func (s *Service) RejectProposal(ctx context.Context, proposalID domain.ID, reason string) error {
	p, err := s.repo.GetTradeProposal(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("proposal not found: %w", err)
	}
	if p.Status != investment.ProposalPending {
		return fmt.Errorf("proposal is not in PENDING state (current: %s)", p.Status)
	}

	if reason == "" {
		reason = "Rejected by user via Web Console"
	}

	if err := s.repo.UpdateTradeProposalStatus(ctx, proposalID, investment.ProposalRejected, reason); err != nil {
		return err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "USER",
			ActorID:      "console-user",
			Action:       "INVESTMENT_PROPOSAL_REJECTED",
			ResourceType: "INVESTMENT_PROPOSAL",
			ResourceID:   proposalID,
			Outcome:      "SUCCESS",
			Details:      json.RawMessage(fmt.Sprintf(`{"proposal_id": "%s", "reason": "%s"}`, proposalID, reason)),
			CreatedAt:    time.Now().UTC(),
		})
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.InvestmentUpdated, map[string]any{
			"type":        "proposal_rejected",
			"proposal_id": proposalID,
			"reason":      reason,
		})
	}

	return nil
}

// EnsureDataSources makes sure default market and announcement providers exist.
func (s *Service) EnsureDataSources(ctx context.Context) error {
	return s.repo.EnsureDefaultDataSources(ctx)
}

func (s *Service) ListDataSources(ctx context.Context) ([]investment.DataSource, error) {
	_ = s.repo.EnsureDefaultDataSources(ctx)
	return s.repo.ListDataSources(ctx)
}

// SyncMarketData fetches the latest quotes/bars from TWSE and persists it.
func (s *Service) SyncMarketData(ctx context.Context, symbol string) (*investment.MarketData, error) {
	if clock.IsReplay(ctx) {
		return nil, fmt.Errorf("live market data fetch is prohibited during Point-in-Time replay (AIA-13 §21.2)")
	}
	_ = s.EnsureDataSources(ctx)
	inst, err := s.repo.GetInstrumentBySymbol(ctx, "", symbol)
	if err != nil {
		return nil, fmt.Errorf("instrument %s not found: %w", symbol, err)
	}

	bar, err := s.dataProvider.FetchLatestBar(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market bar for %s: %w", symbol, err)
	}

	valJSON, _ := json.Marshal(bar)
	now := time.Now().UTC()

	md := &investment.MarketData{
		ID:           uuid.New().String(),
		InstrumentID: inst.ID,
		DataType:     "DAILY_BAR",
		Period:       "1D",
		ValueJSON:    string(valJSON),
		SourceID:     "ds-twse-openapi",
		FetchedAt:    now,
		EffectiveAt:  now,
	}

	if err := s.repo.SaveMarketData(ctx, md); err != nil {
		return nil, fmt.Errorf("failed to save market data: %w", err)
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.InvestmentUpdated, map[string]any{
			"type":   "market_data_synced",
			"symbol": symbol,
			"price":  bar.Close,
		})
	}

	return md, nil
}

// SyncAnnouncements fetches and persists recent MOPS / financial news for a symbol.
func (s *Service) SyncAnnouncements(ctx context.Context, symbol string) ([]investment.RawDocument, error) {
	if clock.IsReplay(ctx) {
		return nil, fmt.Errorf("live announcement fetch is prohibited during Point-in-Time replay (AIA-13 §21.2)")
	}
	_ = s.EnsureDataSources(ctx)
	inst, err := s.repo.GetInstrumentBySymbol(ctx, "", symbol)
	if err != nil {
		return nil, fmt.Errorf("instrument %s not found: %w", symbol, err)
	}

	docs, err := s.dataProvider.FetchNewsAndAnnouncements(ctx, symbol, 5)
	if err != nil {
		return nil, err
	}

	for i := range docs {
		docs[i].InstrumentID = inst.ID
		_ = s.repo.SaveRawDocument(ctx, &docs[i])
	}

	return docs, nil
}

func (s *Service) ListHistoricalData(ctx context.Context, symbol string, limit int) ([]investment.MarketData, error) {
	inst, err := s.repo.GetInstrumentBySymbol(ctx, "", symbol)
	if err != nil {
		return nil, err
	}
	bars, err := s.repo.ListHistoricalMarketData(ctx, inst.ID, limit)
	if err == nil && len(bars) == 0 {
		if md, syncErr := s.SyncMarketData(ctx, symbol); syncErr == nil && md != nil {
			bars = append(bars, *md)
		}
	}
	return bars, err
}

func (s *Service) ListRawDocuments(ctx context.Context, symbol string, limit int) ([]investment.RawDocument, error) {
	instID := ""
	if symbol != "" {
		inst, err := s.repo.GetInstrumentBySymbol(ctx, "", symbol)
		if err == nil {
			instID = inst.ID
		}
	}
	docs, err := s.repo.ListRawDocuments(ctx, instID, limit)
	if err == nil && len(docs) == 0 && symbol != "" {
		docs, _ = s.SyncAnnouncements(ctx, symbol)
	}
	return docs, err
}

// CreatePrediction records a verifiable forecast produced by agy (AIA-05 §16.1).
func (s *Service) CreatePrediction(ctx context.Context, p *investment.Prediction) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.Status == "" {
		p.Status = investment.PredictionPending
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = clock.Now(ctx)
	}
	if p.DueAt.IsZero() {
		p.DueAt = p.CreatedAt.Add(5 * 24 * time.Hour) // default 5-day horizon
	}
	return s.repo.SavePrediction(ctx, p)
}

func (s *Service) ListPredictions(ctx context.Context, status investment.PredictionStatus, limit int) ([]investment.Prediction, error) {
	return s.repo.ListPredictions(ctx, status, limit)
}

// EvaluatePredictions scans due predictions, validates them against market data, and produces calibration scores.
func (s *Service) EvaluatePredictions(ctx context.Context) ([]investment.PredictionOutcome, *investment.EvaluationResult, error) {
	// 1. Get all pending predictions
	pending, err := s.repo.ListPredictions(ctx, investment.PredictionPending, 100)
	if err != nil {
		return nil, nil, err
	}

	now := clock.Now(ctx)
	isReplay := clock.IsReplay(ctx)
	var evaluatedOutcomes []investment.PredictionOutcome

	for _, pred := range pending {
		if pred.CreatedAt.After(now) {
			continue // Never evaluate future-created predictions in PiT replay
		}
		isDue := now.After(pred.DueAt) || now.Equal(pred.DueAt)
		if !isDue && !isReplay && pred.DueAt.Sub(pred.CreatedAt) <= 10*time.Minute {
			isDue = true
		}
		if isDue {
			settleCtx := ctx
			if isReplay && pred.DueAt.Before(now) {
				settleCtx = clock.WithAsOf(ctx, pred.DueAt, true)
			}
			// Get settlement price for instrument strictly <= DueAt / AsOf
			md, err := s.repo.GetLatestMarketData(settleCtx, pred.InstrumentID)
			currentPrice := pred.BasePrice
			if err == nil && md != nil {
				var bar investment.MarketBarValue
				if json.Unmarshal([]byte(md.ValueJSON), &bar) == nil && bar.Close > 0 {
					currentPrice = bar.Close
				}
			}

			outcome := investment.EvaluateSinglePrediction(pred, currentPrice, "TWSE_SETTLEMENT")
			outcome.EvaluatedAt = now
			if err := s.repo.SavePredictionOutcome(ctx, outcome); err == nil {
				evaluatedOutcomes = append(evaluatedOutcomes, *outcome)
			}
		}
	}

	// 2. Compute aggregate calibration metrics across all evaluated outcomes
	allOutcomes, err := s.repo.ListPredictionOutcomes(ctx, 100)
	if err != nil {
		return evaluatedOutcomes, nil, err
	}

	metrics := investment.ComputeCalibrationSummary(allOutcomes)
	periodStart := now.AddDate(0, -1, 0)
	evalResult := investment.BuildEvaluationResult(metrics, periodStart, now, "v1.0-baseline")

	if metrics.TotalEvaluated > 0 {
		_ = s.repo.SaveEvaluationResult(ctx, evalResult)
	}

	return evaluatedOutcomes, evalResult, nil
}

func (s *Service) ListEvaluationResults(ctx context.Context, limit int) ([]investment.EvaluationResult, error) {
	return s.repo.ListEvaluationResults(ctx, limit)
}

// GetActiveSnapshotOptional inspects the active portfolio snapshot and positions ONLY if
// an investment profile already exists, never mutating or seeding the DB as a side effect.
func (s *Service) GetActiveSnapshotOptional(ctx context.Context) (*investment.PortfolioProfile, *investment.PortfolioSnapshot, []investment.Position, error) {
	prof, err := s.repo.GetActiveProfileOptional(ctx)
	if err != nil || prof == nil {
		return nil, nil, nil, err
	}
	snap, err := s.repo.GetLatestSnapshot(ctx, prof.ID)
	if err != nil {
		return prof, nil, nil, err
	}
	positions, err := s.repo.ListPositions(ctx, snap.ID)
	if err != nil {
		return prof, snap, nil, err
	}
	return prof, snap, positions, nil
}

// GetInstrumentByID resolves an instrument record by its ID.
func (s *Service) GetInstrumentByID(ctx context.Context, id domain.ID) (*investment.Instrument, error) {
	return s.repo.GetInstrumentByID(ctx, id)
}

// EvaluateLocalMateriality performs deterministic, zero-network materiality evaluation
// strictly against persisted SQLite market bars, disclosures, and prior decisions.
func (s *Service) EvaluateLocalMateriality(ctx context.Context, symbol string, thresholdPct float64) (*MaterialityResult, error) {
	if thresholdPct <= 0 {
		thresholdPct = 3.0
	}
	prof, err := s.repo.GetActiveProfileOptional(ctx)
	if err != nil {
		return nil, err
	}
	if prof == nil {
		return nil, nil
	}

	inst, err := s.repo.GetInstrumentBySymbol(ctx, "", symbol)
	if err != nil {
		return nil, err
	}

	prevDecision, _ := s.repo.GetLatestDecisionForInstrument(ctx, prof.ID, inst.ID)
	prevProposal, _ := s.repo.GetLatestTradeProposalForInstrument(ctx, prof.ID, inst.ID)

	var bar *investment.MarketBarValue
	if md, mErr := s.repo.GetLatestMarketData(ctx, inst.ID); mErr == nil && md != nil {
		var b investment.MarketBarValue
		if json.Unmarshal([]byte(md.ValueJSON), &b) == nil {
			bar = &b
		}
	}

	res := &MaterialityResult{
		Symbol:           symbol,
		InstrumentID:     inst.ID,
		PreviousDecision: prevDecision,
		PreviousProposal: prevProposal,
		LatestBar:        bar,
	}

	if prevDecision == nil {
		res.IsMaterial = true
		res.Reason = fmt.Sprintf("Initial thesis required: No prior investment decision recorded for %s (%s)", inst.Name, symbol)
		return res, nil
	}

	var prevPrice float64
	if prevProposal != nil && prevProposal.EstimatedPrice > 0 {
		prevPrice = prevProposal.EstimatedPrice
	} else {
		var actRes struct {
			ReferencePrice float64 `json:"reference_price"`
		}
		if json.Unmarshal([]byte(prevDecision.ActionResultJSON), &actRes) == nil && actRes.ReferencePrice > 0 {
			prevPrice = actRes.ReferencePrice
		} else {
			var rawInput struct {
				Price float64 `json:"price"`
			}
			if json.Unmarshal([]byte(prevDecision.RawInputJSON), &rawInput) == nil && rawInput.Price > 0 {
				prevPrice = rawInput.Price
			}
		}
	}

	if bar != nil && bar.Close > 0 && prevPrice > 0 {
		diff := bar.Close - prevPrice
		if diff < 0 {
			diff = -diff
		}
		pctChange := diff / prevPrice * 100.0
		res.PriceChangePct = pctChange
		if pctChange >= thresholdPct {
			res.IsMaterial = true
			res.Reason = fmt.Sprintf("Material price shift on %s: %.2f%% change (%.1f -> %.1f, threshold %.1f%%)", symbol, pctChange, prevPrice, bar.Close, thresholdPct)
			return res, nil
		}
	}

	if docs, dErr := s.repo.ListRawDocuments(ctx, inst.ID, 20); dErr == nil {
		newCount := 0
		for _, d := range docs {
			if d.PublishedAt.After(prevDecision.CreatedAt) {
				newCount++
			}
		}
		res.NewDocumentsCount = newCount
		if newCount > 0 {
			res.IsMaterial = true
			res.Reason = fmt.Sprintf("New corporate disclosures for %s: %d announcements published after prior thesis", symbol, newCount)
			return res, nil
		}
	}

	res.IsMaterial = false
	res.Reason = fmt.Sprintf("Calm market for %s: price moved %.2f%% (< %.1f%%) with 0 new disclosures; existing %s thesis intact", symbol, res.PriceChangePct, thresholdPct, prevDecision.DecisionType)
	return res, nil
}

// ExecuteVirtualProposal settles an approved or autonomous virtual trade proposal into the virtual portfolio ledger.
func (s *Service) ExecuteVirtualProposal(ctx context.Context, proposalID domain.ID) (*investment.PortfolioSnapshot, error) {
	prop, err := s.repo.GetTradeProposal(ctx, proposalID)
	if err != nil {
		return nil, fmt.Errorf("proposal not found: %w", err)
	}
	if prop.Status != investment.ProposalApproved && prop.Status != investment.ProposalPending {
		return nil, fmt.Errorf("proposal %s cannot be settled from status %s", proposalID, prop.Status)
	}

	newSnap, err := s.repo.SettleVirtualProposal(ctx, prop, s.costCfg)
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "AGENT",
			ActorID:      "investment-agent",
			Action:       "VIRTUAL_TRADE_SETTLED",
			ResourceType: "INVESTMENT_SNAPSHOT",
			ResourceID:   newSnap.ID,
			Outcome:      "SUCCESS",
			Details: json.RawMessage(fmt.Sprintf(
				`{"proposal_id":%q,"action":%q,"quantity":%d,"price":%.2f,"available_cash":%d,"total_value":%d}`,
				prop.ID, prop.Action, prop.Quantity, prop.EstimatedPrice, newSnap.AvailableCash, newSnap.TotalPortfolioValue,
			)),
			CreatedAt: time.Now().UTC(),
		})
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.InvestmentUpdated, map[string]any{
			"type":                  "virtual_trade_settled",
			"proposal_id":           prop.ID,
			"snapshot_id":           newSnap.ID,
			"available_cash":        newSnap.AvailableCash,
			"total_portfolio_value": newSnap.TotalPortfolioValue,
		})
	}

	return newSnap, nil
}

// Repo returns the underlying InvestmentRepository for controlled multi-portfolio experiment orchestration.
func (s *Service) Repo() *repository.InvestmentRepository {
	return s.repo
}

// CreateIsolatedProfile initializes a clean NT$100,000 (or custom) isolated portfolio for a generational experiment cohort.
func (s *Service) CreateIsolatedProfile(ctx context.Context, profileID domain.ID, name string, initialCashTWD int64, maxPositionPct float64) (*investment.PortfolioProfile, *investment.PortfolioSnapshot, error) {
	return s.repo.CreateIsolatedProfile(ctx, profileID, name, initialCashTWD, maxPositionPct)
}

// GetSnapshotForProfile returns the latest point-in-time snapshot and positions for a specific portfolio profile.
func (s *Service) GetSnapshotForProfile(ctx context.Context, profileID domain.ID) (*investment.PortfolioProfile, *investment.PortfolioSnapshot, []investment.Position, error) {
	prof, err := s.repo.GetProfile(ctx, profileID)
	if err != nil {
		return nil, nil, nil, err
	}
	snap, err := s.repo.GetLatestSnapshot(ctx, profileID)
	if err != nil {
		return prof, nil, nil, err
	}
	positions, err := s.repo.ListPositions(ctx, snap.ID)
	if err != nil {
		return prof, snap, nil, err
	}
	return prof, snap, positions, nil
}

// ProcessDecisionForProfile evaluates and records a decision (BUY, SELL, ADD, REDUCE, HOLD) against a specific isolated portfolio profile,
// returning both the persisted Decision and any resulting risk-checked TradeProposal.
func (s *Service) ProcessDecisionForProfile(
	ctx context.Context,
	profileID domain.ID,
	sourceLabel string,
	enforceMinConfidence bool,
	input investment.AgyDecisionInput,
	price float64,
) (*investment.Decision, *investment.TradeProposal, error) {
	prof, snap, positions, err := s.GetSnapshotForProfile(ctx, profileID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load profile %s: %w", profileID, err)
	}
	if sourceLabel == "" {
		sourceLabel = "ANTIGRAVITY_LLM"
	}

	rawInputJSON, _ := json.Marshal(input)
	now := clock.Now(ctx)

	symbol := strings.TrimSpace(input.Symbol)
	inst, err := s.repo.GetInstrumentBySymbol(ctx, input.Exchange, symbol)
	if err != nil || inst == nil || !inst.IsActive {
		return nil, nil, fmt.Errorf("invalid instrument symbol: %s", symbol)
	}

	rawAction := strings.ToUpper(strings.TrimSpace(input.Action))
	var canonicalAction investment.DecisionType
	switch rawAction {
	case "BUY", "ADD":
		canonicalAction = investment.DecisionBuy
	case "SELL", "REDUCE":
		canonicalAction = investment.DecisionSell
	case "HOLD", "NO_ACTION", "CASH", "MONITOR":
		canonicalAction = investment.DecisionHold
	default:
		return nil, nil, fmt.Errorf("unsupported action: %s", input.Action)
	}

	if enforceMinConfidence && canonicalAction != investment.DecisionHold && input.Confidence < 0.50 {
		// Below actionable threshold -> safely downgrade to HOLD while recording low-confidence rationale
		canonicalAction = investment.DecisionHold
		input.Reasoning = append(input.Reasoning, fmt.Sprintf("Action %s downgraded to HOLD because decision confidence %.2f < 0.50", rawAction, input.Confidence))
	}

	var currentQty int64
	for _, p := range positions {
		if p.InstrumentID == inst.ID {
			currentQty = p.Quantity
			break
		}
	}

	// Default partial sizing for ADD / REDUCE if TargetAmount was omitted
	targetAmt := input.TargetAmount
	if targetAmt <= 0 {
		switch rawAction {
		case "ADD":
			targetAmt = int64(float64(snap.AvailableCash) * 0.25)
		case "BUY":
			targetAmt = int64(float64(snap.AvailableCash) * 0.40)
		case "REDUCE":
			if currentQty > 0 {
				targetAmt = int64(float64(currentQty) * price * 0.50)
			}
		}
	}

	reasoningJSON, _ := json.Marshal(input.Reasoning)
	risksJSON, _ := json.Marshal(input.Risks)
	invalJSON, _ := json.Marshal(input.InvalidationRules)
	actionResJSON, _ := json.Marshal(map[string]any{
		"reference_price": price,
		"raw_action":      rawAction,
		"target_amount":   targetAmt,
	})

	dec := &investment.Decision{
		ID:                         uuid.New().String(),
		PortfolioProfileID:         prof.ID,
		InstrumentID:               inst.ID,
		PortfolioSnapshotID:        snap.ID,
		DecisionType:               canonicalAction,
		Confidence:                 input.Confidence,
		ReasoningJSON:              string(reasoningJSON),
		RisksJSON:                  string(risksJSON),
		InvalidationConditionsJSON: string(invalJSON),
		ExpectedHorizon:            input.ExpectedHorizon,
		StrategyVersion:            input.StrategyVersion,
		ActionResultJSON:           string(actionResJSON),
		RawInputJSON:               string(rawInputJSON),
		Source:                     sourceLabel,
		CreatedAt:                  now,
	}
	if err := s.repo.SaveDecision(ctx, dec); err != nil {
		return nil, nil, fmt.Errorf("save decision: %w", err)
	}

	if canonicalAction == investment.DecisionHold {
		return dec, nil, nil
	}

	constraints, err := s.repo.ListConstraints(ctx, prof.ID)
	if err != nil {
		return dec, nil, err
	}
	sizing := investment.EvaluateTradeLimits(prof, snap, constraints, canonicalAction, targetAmt, price, currentQty, s.costCfg)
	if sizing.Rejected {
		rej := &investment.DecisionRejection{
			ID:                 uuid.New().String(),
			PortfolioProfileID: prof.ID,
			RawOutput:          string(rawInputJSON),
			ErrorsJSON:         fmt.Sprintf(`[%q]`, sizing.RejectionError),
			Source:             "LIMITS_CHECK_GO",
			CreatedAt:          now,
		}
		_ = s.repo.SaveDecisionRejection(ctx, rej)
		return dec, nil, nil
	}

	propPayload, _ := json.Marshal(map[string]any{
		"symbol":           symbol,
		"raw_action":       rawAction,
		"action":           string(canonicalAction),
		"quantity":         sizing.Quantity,
		"estimated_price":  price,
		"gross_amount":     sizing.Cost.GrossAmount,
		"broker_fee":       sizing.Cost.BrokerFee,
		"transaction_tax":  sizing.Cost.TransactionTax,
		"total_cost":       sizing.Cost.TotalCost,
		"confidence":       input.Confidence,
		"strategy_version": input.StrategyVersion,
	})

	proposal := &investment.TradeProposal{
		ID:                   uuid.New().String(),
		PortfolioProfileID:   prof.ID,
		DecisionID:           dec.ID,
		PortfolioSnapshotID:  snap.ID,
		InstrumentID:         inst.ID,
		Action:               canonicalAction,
		Quantity:             sizing.Quantity,
		EstimatedPrice:       price,
		EstimatedGrossAmount: sizing.Cost.GrossAmount,
		EstimatedTotalCost:   sizing.Cost.TotalCost,
		ProposalJSON:         string(propPayload),
		Status:               investment.ProposalPending,
		StatusReason:         sizing.AdjustmentNote,
		CreatedAt:            now,
		ExpiresAt:            now.Add(24 * time.Hour),
		UpdatedAt:            now,
	}
	if err := s.repo.SaveTradeProposal(ctx, proposal); err != nil {
		return dec, nil, err
	}
	return dec, proposal, nil
}

// EvaluatePredictionsForProfile settles due predictions for a specific portfolio profile strictly <= AsOf.
func (s *Service) EvaluatePredictionsForProfile(ctx context.Context, profileID domain.ID) ([]investment.PredictionOutcome, error) {
	pending, err := s.repo.ListPredictionsForProfile(ctx, profileID, investment.PredictionPending, 200)
	if err != nil {
		return nil, err
	}
	now := clock.Now(ctx)
	isReplay := clock.IsReplay(ctx)
	var settled []investment.PredictionOutcome

	for _, pred := range pending {
		if pred.CreatedAt.After(now) {
			continue
		}
		if now.Before(pred.DueAt) {
			continue
		}
		settleCtx := ctx
		if isReplay && pred.DueAt.Before(now) {
			settleCtx = clock.WithAsOf(ctx, pred.DueAt, true)
		}
		md, err := s.repo.GetLatestMarketData(settleCtx, pred.InstrumentID)
		currentPrice := pred.BasePrice
		if err == nil && md != nil {
			var bar investment.MarketBarValue
			if json.Unmarshal([]byte(md.ValueJSON), &bar) == nil && bar.Close > 0 {
				currentPrice = bar.Close
			}
		}
		oc := investment.EvaluateSinglePrediction(pred, currentPrice, "TWSE_PIT_SETTLEMENT")
		oc.EvaluatedAt = now
		if err := s.repo.SavePredictionOutcome(ctx, oc); err == nil {
			settled = append(settled, *oc)
		}
	}
	return settled, nil
}

