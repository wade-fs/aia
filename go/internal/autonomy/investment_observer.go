package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/wade/aia/internal/autonomy/clock"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	domainInv "github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/service/investment"
	taskService "github.com/wade/aia/internal/service/task"
)

// InvestmentObserver connects the TWSE/TPEx/MOPS local investment state and Materiality Gate
// to Step 1 (Observe) of the Autonomy Core without mutating or seeding the database (AIA-05, Plan Phase 3 §4.4).
type InvestmentObserver struct {
	invSvc       *investment.Service
	thresholdPct float64
}

func NewInvestmentObserver(invSvc *investment.Service) *InvestmentObserver {
	return &InvestmentObserver{
		invSvc:       invSvc,
		thresholdPct: 3.0,
	}
}

func (o *InvestmentObserver) SetThresholdPct(pct float64) {
	if pct > 0 {
		o.thresholdPct = pct
	}
}

func (o *InvestmentObserver) Observe(ctx context.Context, projectID string) ([]domainAutonomy.Observation, error) {
	if o.invSvc == nil {
		return nil, nil
	}

	// Strictly read-only: if no investment portfolio has been initialized, emit zero observations
	prof, snap, positions, err := o.invSvc.GetActiveSnapshotOptional(ctx)
	if err != nil || prof == nil || snap == nil {
		return nil, nil
	}

	now := clock.Now(ctx)
	var observations []domainAutonomy.Observation

	posByInstID := make(map[string]domainInv.Position, len(positions))
	for _, pos := range positions {
		if pos.Quantity > 0 {
			posByInstID[pos.InstrumentID] = pos
		}
	}

	instruments, _ := o.invSvc.ListInstruments(ctx)
	for _, inst := range instruments {
		if !inst.IsActive {
			continue
		}
		pos, hasPos := posByInstID[inst.ID]

		mat, mErr := o.invSvc.EvaluateLocalMateriality(ctx, inst.Symbol, o.thresholdPct)
		if mErr != nil || mat == nil {
			continue
		}

		// Only observe symbols that are actively held OR have persisted market data
		if !hasPos && mat.LatestBar == nil {
			continue
		}

		currentPrice := 0.0
		if mat.LatestBar != nil && mat.LatestBar.Close > 0 {
			currentPrice = mat.LatestBar.Close
		} else if hasPos && pos.LastKnownPrice > 0 {
			currentPrice = pos.LastKnownPrice
		} else if hasPos && pos.AverageCost > 0 {
			currentPrice = pos.AverageCost
		}

		if mat.IsMaterial {
			observations = append(observations, domainAutonomy.Observation{
				Source:        "INVESTMENT_MATERIALITY_GATE",
				Domain:        "investment",
				Category:      domainAutonomy.ObsCategoryMaterialChange,
				Summary:       mat.Reason,
				Confidence:    domainAutonomy.ConfidenceConfirmed,
				Severity:      8,
				TargetAgentID: "investment-agent",
				Data: map[string]any{
					"symbol":              inst.Symbol,
					"exchange":            inst.Exchange,
					"instrument_id":       inst.ID,
					"is_material":         true,
					"price_change_pct":    mat.PriceChangePct,
					"new_documents_count": mat.NewDocumentsCount,
					"held_quantity":       pos.Quantity,
					"current_price":       currentPrice,
					"operation_type":      "FINANCIAL_TRANSACTION",
					"is_real_money":       false,
				},
				Timestamp: now,
			})
		} else {
			// Calm market below materiality threshold and 0 new announcements -> rational NO_ACTION baseline
			observations = append(observations, domainAutonomy.Observation{
				Source:        "INVESTMENT_MATERIALITY_GATE",
				Domain:        "investment",
				Category:      domainAutonomy.ObsCategoryStateSnapshot,
				NonActionable: true,
				Summary:       mat.Reason,
				TargetAgentID: "investment-agent",
				Data: map[string]any{
					"symbol":           inst.Symbol,
					"is_material":      false,
					"price_change_pct": mat.PriceChangePct,
				},
				Timestamp: now,
			})
		}
	}

	// Check if any pending predictions have reached their evaluation horizon
	if pendingPreds, pErr := o.invSvc.ListPredictions(ctx, domainInv.PredictionPending, 50); pErr == nil {
		dueCount := 0
		for _, pred := range pendingPreds {
			if now.After(pred.DueAt) || now.Equal(pred.DueAt) {
				dueCount++
			}
		}
		if dueCount > 0 {
			observations = append(observations, domainAutonomy.Observation{
				Source:        "INVESTMENT_CALIBRATION_WATCH",
				Domain:        "investment",
				Category:      domainAutonomy.ObsCategoryOpportunity,
				Summary:       fmt.Sprintf("%d investment predictions have reached horizon and are ready for Brier Score calibration", dueCount),
				Confidence:    domainAutonomy.ConfidenceConfirmed,
				Severity:      7,
				TargetAgentID: "investment-agent",
				Data: map[string]any{
					"domain":               "investment",
					"evaluate_predictions": true,
					"due_count":            dueCount,
				},
				Timestamp: now,
			})
		}
	}

	return observations, nil
}

// InvestmentPilotExecutor implements closed-loop paper-trading evaluation and virtual ledger settlement
// during Step 8 (Execute) for authorized investment-agent initiatives (Plan Phase 3 §4.4, Milestone M4).
type InvestmentPilotExecutor struct {
	invSvc                  *investment.Service
	taskSvc                 *taskService.Service
	autoSettleVirtualTrades bool
}

func NewInvestmentPilotExecutor(invSvc *investment.Service, taskSvc *taskService.Service) *InvestmentPilotExecutor {
	return &InvestmentPilotExecutor{
		invSvc:                  invSvc,
		taskSvc:                 taskSvc,
		autoSettleVirtualTrades: true,
	}
}

func (p *InvestmentPilotExecutor) CanHandle(init *domainAutonomy.CandidateInitiative) bool {
	if p.invSvc == nil || init == nil {
		return false
	}
	if init.TargetAgentID != "investment-agent" {
		return false
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(init.MetadataJSON), &meta); err != nil {
		return false
	}
	if _, ok := meta["symbol"].(string); ok {
		return true
	}
	if evalPred, ok := meta["evaluate_predictions"].(bool); ok && evalPred {
		return true
	}
	return false
}

func (p *InvestmentPilotExecutor) Execute(ctx context.Context, init *domainAutonomy.CandidateInitiative, spawnedTaskID string) (string, error) {
	var meta map[string]any
	_ = json.Unmarshal([]byte(init.MetadataJSON), &meta)

	if evalPred, ok := meta["evaluate_predictions"].(bool); ok && evalPred {
		outcomes, _, err := p.invSvc.EvaluatePredictions(ctx)
		if err != nil {
			return "", err
		}
		if p.taskSvc != nil && spawnedTaskID != "" {
			_ = p.taskSvc.UpdateStatus(ctx, spawnedTaskID, task.StatusCompleted)
		}
		return fmt.Sprintf("Evaluated %d due predictions", len(outcomes)), nil
	}

	symbol, _ := meta["symbol"].(string)
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return "", nil
	}
	exchange, _ := meta["exchange"].(string)

	prof, snap, positions, err := p.invSvc.GetActiveSnapshotOptional(ctx)
	if err != nil || prof == nil || snap == nil {
		return "", err
	}

	mat, err := p.invSvc.EvaluateLocalMateriality(ctx, symbol, 3.0)
	if err != nil || mat == nil {
		return "", err
	}

	var heldQty int64
	var fallbackPrice float64
	for _, pos := range positions {
		if pos.InstrumentID == mat.InstrumentID {
			heldQty = pos.Quantity
			if pos.LastKnownPrice > 0 {
				fallbackPrice = pos.LastKnownPrice
			} else {
				fallbackPrice = pos.AverageCost
			}
			break
		}
	}

	currentPrice := fallbackPrice
	if mat.LatestBar != nil && mat.LatestBar.Close > 0 {
		currentPrice = mat.LatestBar.Close
	} else if mp, ok := meta["current_price"].(float64); ok && mp > 0 {
		currentPrice = mp
	}
	if currentPrice <= 0 {
		return "", nil
	}

	constraints, _ := p.invSvc.ListConstraints(ctx)
	maxPosPct := 0.25
	minCashPct := 0.10
	maxTradeAmt := int64(250000)
	for _, c := range constraints {
		if !c.Enabled {
			continue
		}
		switch c.ConstraintType {
		case "max_position_pct":
			if v, err := strconv.ParseFloat(c.ConstraintValue, 64); err == nil && v > 0 {
				maxPosPct = v
			}
		case "min_cash_pct":
			if v, err := strconv.ParseFloat(c.ConstraintValue, 64); err == nil && v >= 0 {
				minCashPct = v
			}
		case "max_trade_amount":
			if v, err := strconv.ParseInt(c.ConstraintValue, 10, 64); err == nil && v > 0 {
				maxTradeAmt = v
			}
		}
	}

	totalVal := float64(snap.TotalPortfolioValue)
	if totalVal <= 0 {
		totalVal = float64(snap.AvailableCash)
	}
	posVal := float64(heldQty) * currentPrice
	posRatio := 0.0
	if totalVal > 0 {
		posRatio = posVal / totalVal
	}
	spendableCash := snap.AvailableCash - int64(totalVal*minCashPct)
	oneLotApprox := int64(currentPrice * 1005) // 1,000 shares + broker fee buffer

	action := "HOLD"
	var targetAmount int64
	reasoning := []string{mat.Reason}

	if mat.PreviousDecision == nil {
		// Initial thesis establishment
		if heldQty == 0 && oneLotApprox <= spendableCash && float64(oneLotApprox) <= totalVal*maxPosPct && oneLotApprox <= maxTradeAmt {
			action = "BUY"
			targetAmount = oneLotApprox + 500
			reasoning = append(reasoning, fmt.Sprintf("Initial virtual allocation for %s within %.0f%% position cap and cash reserve", symbol, maxPosPct*100))
		} else {
			action = "HOLD"
			reasoning = append(reasoning, fmt.Sprintf("Established baseline HOLD thesis for %s at %.2f", symbol, currentPrice))
		}
	} else {
		// Material price shift (>= 3%) or new corporate disclosure
		if heldQty > 0 && posRatio > maxPosPct {
			action = "SELL"
			targetAmount = 0 // Rebalance/close overweight position within Go risk limits
			reasoning = append(reasoning, fmt.Sprintf("Material shift triggered risk-managed virtual SELL on overweight position (%.1f%% > %.1f%%)", posRatio*100, maxPosPct*100))
		} else if heldQty == 0 && oneLotApprox <= spendableCash && float64(oneLotApprox) <= totalVal*maxPosPct && oneLotApprox <= maxTradeAmt {
			action = "BUY"
			targetAmount = oneLotApprox + 500
			reasoning = append(reasoning, fmt.Sprintf("Material signal triggered virtual BUY entry on %s within risk limits", symbol))
		} else {
			action = "HOLD"
			reasoning = append(reasoning, fmt.Sprintf("Re-evaluated %s after material event; updated thesis reference price to %.2f", symbol, currentPrice))
		}
	}

	agyInput := domainInv.AgyDecisionInput{
		Symbol:            symbol,
		Exchange:          exchange,
		Action:            action,
		TargetAmount:      targetAmount,
		Confidence:        0.80,
		Reasoning:         reasoning,
		Risks:             []string{"Paper-trading simulation only; subject to market volatility"},
		InvalidationRules: []string{"Re-evaluate if price shifts >= 3.0% or new MOPS material filing arrives"},
		ExpectedHorizon:   "5D",
		StrategyVersion:   "phase3-autonomy-pilot-v1",
	}

	proposal, err := p.invSvc.ProcessAgyDecision(ctx, agyInput, currentPrice)
	if err != nil {
		return "", err
	}

	summaryNote := fmt.Sprintf("Investment Pilot recorded %s thesis for %s @ %.2f", action, symbol, currentPrice)
	if proposal != nil && p.autoSettleVirtualTrades {
		if newSnap, sErr := p.invSvc.ExecuteVirtualProposal(ctx, proposal.ID); sErr == nil && newSnap != nil {
			summaryNote = fmt.Sprintf("Investment Pilot settled virtual %s proposal %s (%d shares @ %.2f, cash=%d)",
				proposal.Action, proposal.ID, proposal.Quantity, proposal.EstimatedPrice, newSnap.AvailableCash)
		}
	}

	if p.taskSvc != nil && spawnedTaskID != "" {
		_ = p.taskSvc.UpdateStatus(ctx, spawnedTaskID, task.StatusCompleted)
	}
	return summaryNote, nil
}
