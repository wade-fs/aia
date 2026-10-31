package investment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// MaterialityConfig configures the sensitivity threshold for triggering an AI agent recommendation update.
type MaterialityConfig struct {
	PriceChangeThresholdPct float64 `json:"price_change_threshold_pct"` // Default: 3.0%
	CheckAnnouncements      bool    `json:"check_announcements"`        // Default: true
	Force                   bool    `json:"force"`                      // Bypass check and force re-evaluation
}

func DefaultMaterialityConfig() MaterialityConfig {
	return MaterialityConfig{
		PriceChangeThresholdPct: 3.0,
		CheckAnnouncements:      true,
		Force:                   false,
	}
}

// MaterialityResult captures whether new material information warrants launching an Investment Agent run.
type MaterialityResult struct {
	Symbol            string                     `json:"symbol"`
	InstrumentID      domain.ID                  `json:"instrument_id"`
	IsMaterial        bool                       `json:"is_material"`
	Reason            string                     `json:"reason"`
	PriceChangePct    float64                    `json:"price_change_pct"`
	NewDocumentsCount int                        `json:"new_documents_count"`
	PreviousDecision  *investment.Decision       `json:"previous_decision,omitempty"`
	PreviousProposal  *investment.TradeProposal  `json:"previous_proposal,omitempty"`
	LatestBar         *investment.MarketBarValue `json:"latest_bar,omitempty"`
}

// EvaluateMateriality implements the core principle: "Data update does NOT equal recommendation update."
// Only when new information exceeds the materiality threshold or presents significant events
// will the Investment Agent be scheduled, preserving tokens and maintaining previous theses.
func (s *Service) EvaluateMateriality(ctx context.Context, symbol string, cfg MaterialityConfig) (*MaterialityResult, error) {
	if cfg.PriceChangeThresholdPct <= 0 {
		cfg.PriceChangeThresholdPct = 3.0
	}

	prof, err := s.GetProfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get investment profile: %w", err)
	}

	inst, err := s.repo.GetInstrumentBySymbol(ctx, "", symbol)
	if err != nil {
		return nil, fmt.Errorf("instrument %s not found: %w", symbol, err)
	}

	// 1. Fetch latest decision for instrument
	prevDecision, err := s.repo.GetLatestDecisionForInstrument(ctx, prof.ID, inst.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check prior decision: %w", err)
	}
	prevProposal, _ := s.repo.GetLatestTradeProposalForInstrument(ctx, prof.ID, inst.ID)

	// Fetch current market bar
	bar, err := s.dataProvider.FetchLatestBar(ctx, symbol)
	if err != nil || bar == nil {
		md, mErr := s.repo.GetLatestMarketData(ctx, inst.ID)
		if mErr == nil && md != nil {
			var b investment.MarketBarValue
			if json.Unmarshal([]byte(md.ValueJSON), &b) == nil {
				bar = &b
			}
		}
	}

	res := &MaterialityResult{
		Symbol:           symbol,
		InstrumentID:     inst.ID,
		PreviousDecision: prevDecision,
		PreviousProposal: prevProposal,
		LatestBar:        bar,
	}

	// If forced by caller, trigger update
	if cfg.Force {
		res.IsMaterial = true
		res.Reason = "Manual override: Force re-evaluation requested"
		return res, nil
	}

	// 2. Condition: No prior decision exists -> Initial research required
	if prevDecision == nil {
		res.IsMaterial = true
		res.Reason = fmt.Sprintf("Initial thesis required: No prior investment decision found for %s (%s)", inst.Name, symbol)
		return res, nil
	}

	// 3. Condition: Significant price movement
	var prevPrice float64
	if prevProposal != nil && prevProposal.EstimatedPrice > 0 {
		prevPrice = prevProposal.EstimatedPrice
	} else {
		// Attempt parse from raw decision input
		var rawInput struct {
			TargetAmount int64   `json:"target_amount"`
			Price        float64 `json:"price"`
		}
		if json.Unmarshal([]byte(prevDecision.RawInputJSON), &rawInput) == nil && rawInput.Price > 0 {
			prevPrice = rawInput.Price
		}
	}

	if bar != nil && bar.Close > 0 && prevPrice > 0 {
		pctChange := math.Abs(bar.Close-prevPrice) / prevPrice * 100.0
		res.PriceChangePct = pctChange
		if pctChange >= cfg.PriceChangeThresholdPct {
			res.IsMaterial = true
			res.Reason = fmt.Sprintf("Material price shift: %.2f%% change from prior price %.1f to %.1f (threshold: %.1f%%)",
				pctChange, prevPrice, bar.Close, cfg.PriceChangeThresholdPct)
			return res, nil
		}
	}

	// 4. Condition: New corporate announcements / regulatory filings since last decision
	if cfg.CheckAnnouncements {
		docs, err := s.repo.ListRawDocuments(ctx, inst.ID, 20)
		if err == nil {
			newCount := 0
			for _, d := range docs {
				if d.PublishedAt.After(prevDecision.CreatedAt) {
					newCount++
				}
			}
			res.NewDocumentsCount = newCount
			if newCount > 0 {
				res.IsMaterial = true
				res.Reason = fmt.Sprintf("New corporate disclosures: %d announcements/filings published after previous thesis (%s)",
					newCount, prevDecision.CreatedAt.Format("2006-01-02 15:04"))
				return res, nil
			}
		}
	}

	// 5. No material change detected -> Retain previous recommendation
	res.IsMaterial = false
	decisionLabel := string(prevDecision.DecisionType)
	if prevProposal != nil {
		decisionLabel = fmt.Sprintf("%s (Qty: %d, Total: NT$%d)", prevProposal.Action, prevProposal.Quantity, prevProposal.EstimatedTotalCost)
	}
	res.Reason = fmt.Sprintf("No material change: Price moved %.2f%% (< %.1f%%) with 0 new announcements. Preserved existing recommendation: %s. AI token consumption skipped.",
		res.PriceChangePct, cfg.PriceChangeThresholdPct, decisionLabel)

	return res, nil
}

// PreserveExistingDecision explicitly reaffirms the current thesis without LLM token consumption.
func (s *Service) PreserveExistingDecision(ctx context.Context, symbol string, reason string) error {
	prof, err := s.GetProfile(ctx)
	if err != nil {
		return err
	}
	inst, err := s.repo.GetInstrumentBySymbol(ctx, "", symbol)
	if err != nil {
		return err
	}
	prevDecision, err := s.repo.GetLatestDecisionForInstrument(ctx, prof.ID, inst.ID)
	if err != nil || prevDecision == nil {
		return fmt.Errorf("no existing decision to preserve for %s", symbol)
	}

	now := time.Now().UTC()
	if s.auditRepo != nil {
		details, _ := json.Marshal(map[string]any{
			"symbol":           symbol,
			"decision_id":      prevDecision.ID,
			"decision_type":    prevDecision.DecisionType,
			"preservation_why": reason,
			"preserved_at":     now,
		})
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "SYSTEM",
			ActorID:      "materiality-gate",
			Action:       "INVESTMENT_DECISION_PRESERVED",
			ResourceType: "INVESTMENT_DECISION",
			ResourceID:   prevDecision.ID,
			Outcome:      "SUCCESS",
			Details:      details,
			CreatedAt:    now,
		})
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.InvestmentUpdated, map[string]any{
			"type":        "decision_preserved",
			"symbol":      symbol,
			"decision_id": prevDecision.ID,
			"action":      string(prevDecision.DecisionType),
			"reason":      reason,
		})
	}

	slog.Info("preserved existing investment thesis (zero LLM token spend)", "symbol", symbol, "reason", reason)
	return nil
}
