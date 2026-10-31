package investment

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CostConfig holds broker fee and tax parameters.
type CostConfig struct {
	FeeRate         float64 // default 0.001425 (0.1425%)
	FeeDiscountRate float64 // default 1.0 (no discount) or e.g. 0.6 (60% discount)
	MinFee          int64   // default 20 TWD
	TaxRate         float64 // default 0.003 (0.3%) for SELL
}

// DefaultTWCSTConfig returns default Taiwan stock transaction cost parameters.
func DefaultTWCostConfig() CostConfig {
	return CostConfig{
		FeeRate:         0.001425,
		FeeDiscountRate: 1.0,
		MinFee:          20,
		TaxRate:         0.003,
	}
}

// CalculateTransactionCost calculates exact fees, taxes and net/gross amounts in Go.
// LLMs must NEVER calculate or finalize these numbers.
func CalculateTransactionCost(action DecisionType, quantity int64, price float64, cfg CostConfig) CostBreakdown {
	if cfg.FeeRate <= 0 {
		cfg.FeeRate = 0.001425
	}
	if cfg.FeeDiscountRate <= 0 {
		cfg.FeeDiscountRate = 1.0
	}
	if cfg.MinFee <= 0 {
		cfg.MinFee = 20
	}
	if cfg.TaxRate <= 0 {
		cfg.TaxRate = 0.003
	}

	gross := int64(math.Round(float64(quantity) * price))
	if gross <= 0 {
		return CostBreakdown{}
	}

	rawFee := float64(gross) * cfg.FeeRate * cfg.FeeDiscountRate
	fee := int64(math.Floor(rawFee))
	if fee < cfg.MinFee {
		fee = cfg.MinFee
	}

	var tax int64
	if action == DecisionSell {
		tax = int64(math.Floor(float64(gross) * cfg.TaxRate))
	}

	var total int64
	if action == DecisionBuy {
		total = gross + fee
	} else {
		// For sell, total cost represents net cash proceeds
		total = gross - fee - tax
	}

	var effectiveRate float64
	if gross > 0 {
		if action == DecisionBuy {
			effectiveRate = float64(fee) / float64(gross)
		} else {
			effectiveRate = float64(fee+tax) / float64(gross)
		}
	}

	return CostBreakdown{
		GrossAmount:    gross,
		BrokerFee:      fee,
		TransactionTax: tax,
		TotalCost:      total,
		EffectiveRate:  effectiveRate,
	}
}

// SizingResult represents the Go-determined trade quantity and adjustments.
type SizingResult struct {
	Quantity       int64
	Cost           CostBreakdown
	WasAdjusted    bool
	AdjustmentNote string
	Rejected       bool
	RejectionError string
}

// EvaluateTradeLimits validates whether a proposed trade complies with portfolio cash,
// capital limit, and risk constraints, and sizes the trade accordingly.
func EvaluateTradeLimits(
	profile *PortfolioProfile,
	snapshot *PortfolioSnapshot,
	constraints []Constraint,
	action DecisionType,
	targetAmount int64,
	price float64,
	currentPositionQty int64,
	cfg CostConfig,
) SizingResult {
	if price <= 0 {
		return SizingResult{Rejected: true, RejectionError: "invalid estimated price <= 0"}
	}

	// Read constraints
	var maxPositionPct float64 = 0.30 // default 30% max in single position
	var minCashPct float64 = 0.05      // default 5% min cash reserve
	var maxTradeAmount int64 = 0       // optional ceiling
	var allowOddLots bool = false      // default full lots (1,000 shares)

	for _, c := range constraints {
		if !c.Enabled {
			continue
		}
		switch c.ConstraintType {
		case "max_position_pct":
			if v, err := strconv.ParseFloat(c.ConstraintValue, 64); err == nil && v > 0 {
				maxPositionPct = v
			}
		case "min_cash_pct":
			if v, err := strconv.ParseFloat(c.ConstraintValue, 64); err == nil && v >= 0 {
				minCashPct = v
			}
		case "max_trade_amount":
			if v, err := strconv.ParseInt(c.ConstraintValue, 10, 64); err == nil && v > 0 {
				maxTradeAmount = v
			}
		case "allow_odd_lots":
			allowOddLots = strings.ToLower(c.ConstraintValue) == "true" || c.ConstraintValue == "1"
		}
	}

	lotSize := int64(1000)
	if allowOddLots {
		lotSize = 1
	}

	if action == DecisionHold {
		return SizingResult{Rejected: true, RejectionError: "decision is HOLD; no trade proposal required"}
	}

	if action == DecisionSell {
		if currentPositionQty <= 0 {
			return SizingResult{Rejected: true, RejectionError: "cannot SELL: no active position held"}
		}
		sellQty := currentPositionQty
		if targetAmount > 0 {
			sharesFromTarget := int64(math.Floor(float64(targetAmount) / price))
			if sharesFromTarget < sellQty {
				sellQty = (sharesFromTarget / lotSize) * lotSize
				if sellQty <= 0 && allowOddLots {
					sellQty = sharesFromTarget
				}
			}
		}
		if sellQty <= 0 {
			return SizingResult{Rejected: true, RejectionError: "calculated sell quantity is 0"}
		}
		cost := CalculateTransactionCost(DecisionSell, sellQty, price, cfg)
		return SizingResult{
			Quantity: sellQty,
			Cost:     cost,
		}
	}

	// Decision is BUY
	if snapshot.AvailableCash <= 0 {
		return SizingResult{Rejected: true, RejectionError: "insufficient available cash (cash <= 0)"}
	}

	// 1. Check Portfolio Total Value & Cash Buffer
	totalVal := snapshot.TotalPortfolioValue
	if totalVal <= 0 {
		totalVal = snapshot.AvailableCash
	}

	requiredCashReserve := int64(math.Round(float64(totalVal) * minCashPct))
	spendableCash := snapshot.AvailableCash - requiredCashReserve
	if spendableCash <= 0 {
		return SizingResult{
			Rejected:       true,
			RejectionError: fmt.Sprintf("available cash %d violates min cash reserve requirement (%d)", snapshot.AvailableCash, requiredCashReserve),
		}
	}

	// 2. Capital Limit ceiling
	if profile.CapitalLimit > 0 && spendableCash > profile.CapitalLimit {
		spendableCash = profile.CapitalLimit
	}

	// 3. Max Trade Amount ceiling
	if maxTradeAmount > 0 && spendableCash > maxTradeAmount {
		spendableCash = maxTradeAmount
	}

	// 4. Max Position Pct ceiling
	currentPositionVal := int64(math.Round(float64(currentPositionQty) * price))
	maxAllowedPositionVal := int64(math.Round(float64(totalVal) * maxPositionPct))
	maxBuyValForPosition := maxAllowedPositionVal - currentPositionVal
	if maxBuyValForPosition <= 0 {
		return SizingResult{
			Rejected:       true,
			RejectionError: fmt.Sprintf("position already at or exceeds max_position_pct (%.1f%%)", maxPositionPct*100),
		}
	}
	if spendableCash > maxBuyValForPosition {
		spendableCash = maxBuyValForPosition
	}

	// Desired budget: if targetAmount given, take min of targetAmount and spendableCash
	budget := spendableCash
	adjusted := false
	var adjustNotes []string

	if targetAmount > 0 && targetAmount < budget {
		budget = targetAmount
	} else if targetAmount > 0 && targetAmount > budget {
		adjusted = true
		adjustNotes = append(adjustNotes, fmt.Sprintf("target amount %d reduced to budget %d due to risk/capital limits", targetAmount, budget))
	}

	// 5. Compute quantity based on lotSize and ensure (gross + fee) <= spendableCash
	shares := (int64(math.Floor(float64(budget)/price)) / lotSize) * lotSize
	if shares <= 0 {
		return SizingResult{
			Rejected:       true,
			RejectionError: fmt.Sprintf("budget %d is insufficient for minimum lot size (%d shares at %.2f)", budget, lotSize, price),
		}
	}

	// Iteratively step down if fee pushes total cost over available spendable cash
	cost := CalculateTransactionCost(DecisionBuy, shares, price, cfg)
	for cost.TotalCost > spendableCash && shares >= lotSize {
		shares -= lotSize
		if shares <= 0 {
			break
		}
		cost = CalculateTransactionCost(DecisionBuy, shares, price, cfg)
	}

	if shares <= 0 {
		return SizingResult{
			Rejected:       true,
			RejectionError: "transaction cost and fees exceed spendable budget",
		}
	}

	return SizingResult{
		Quantity:       shares,
		Cost:           cost,
		WasAdjusted:    adjusted,
		AdjustmentNote: strings.Join(adjustNotes, "; "),
	}
}
