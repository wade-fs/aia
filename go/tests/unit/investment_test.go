package unit

import (
	"testing"
	"time"

	"github.com/wade/aia/internal/domain/investment"
)

func TestCalculateTransactionCost_Buy(t *testing.T) {
	cfg := investment.DefaultTWCostConfig()

	// 1,000 shares at NT$ 100 = 100,000
	// fee = 100,000 * 0.001425 = 142.5 -> floor = 142
	// tax = 0
	// total = 100,142
	cost := investment.CalculateTransactionCost(investment.DecisionBuy, 1000, 100.0, cfg)
	if cost.GrossAmount != 100000 {
		t.Errorf("expected gross 100000, got %d", cost.GrossAmount)
	}
	if cost.BrokerFee != 142 {
		t.Errorf("expected broker fee 142, got %d", cost.BrokerFee)
	}
	if cost.TransactionTax != 0 {
		t.Errorf("expected tax 0 for buy, got %d", cost.TransactionTax)
	}
	if cost.TotalCost != 100142 {
		t.Errorf("expected total cost 100142, got %d", cost.TotalCost)
	}
}

func TestCalculateTransactionCost_MinFee(t *testing.T) {
	cfg := investment.DefaultTWCostConfig()

	// 10 shares at NT$ 10 = 100
	// raw fee = 100 * 0.001425 = 0.1425 -> min fee = 20
	cost := investment.CalculateTransactionCost(investment.DecisionBuy, 10, 10.0, cfg)
	if cost.BrokerFee != 20 {
		t.Errorf("expected min fee 20, got %d", cost.BrokerFee)
	}
	if cost.TotalCost != 120 {
		t.Errorf("expected total cost 120, got %d", cost.TotalCost)
	}
}

func TestCalculateTransactionCost_Sell(t *testing.T) {
	cfg := investment.DefaultTWCostConfig()

	// 1,000 shares at NT$ 100 = 100,000
	// fee = 142
	// tax = 100,000 * 0.003 = 300
	// net proceeds = 100,000 - 142 - 300 = 99,558
	cost := investment.CalculateTransactionCost(investment.DecisionSell, 1000, 100.0, cfg)
	if cost.TransactionTax != 300 {
		t.Errorf("expected tax 300, got %d", cost.TransactionTax)
	}
	if cost.TotalCost != 99558 {
		t.Errorf("expected net proceeds 99558, got %d", cost.TotalCost)
	}
}

func TestEvaluateTradeLimits_CashAndLimits(t *testing.T) {
	profile := &investment.PortfolioProfile{
		ID:           "test-prof",
		Name:         "Main Growth",
		Market:       "TW",
		Currency:     "TWD",
		CapitalLimit: 500000,
	}
	snapshot := &investment.PortfolioSnapshot{
		ID:                  "snap-1",
		PortfolioProfileID:  "test-prof",
		AvailableCash:       200000,
		TotalPortfolioValue: 500000,
		AsOf:                time.Now().UTC(),
	}
	constraints := []investment.Constraint{
		{ConstraintType: "max_position_pct", ConstraintValue: "0.25", Enabled: true},
		{ConstraintType: "min_cash_pct", ConstraintValue: "0.10", Enabled: true},
	}
	cfg := investment.DefaultTWCostConfig()

	// Scenario 1: Buy within limits
	// Price = NT$ 50/share. 1 lot = 1,000 shares = NT$ 50,000
	// Total val = 500k, max position 25% = 125k. Cash buffer 10% = 50k. Spendable = 150k.
	// Target amount = 100k -> 2 lots = 2,000 shares = 100k
	res := investment.EvaluateTradeLimits(profile, snapshot, constraints, investment.DecisionBuy, 100000, 50.0, 0, cfg)
	if res.Rejected {
		t.Fatalf("unexpected rejection: %s", res.RejectionError)
	}
	if res.Quantity != 2000 {
		t.Errorf("expected 2000 shares, got %d", res.Quantity)
	}
	if res.Cost.GrossAmount != 100000 {
		t.Errorf("expected gross 100000, got %d", res.Cost.GrossAmount)
	}

	// Scenario 2: Target exceeds max_position_pct -> adjusted downward
	// Target 200k, but max position is 125k -> max 2 lots (100k)
	res2 := investment.EvaluateTradeLimits(profile, snapshot, constraints, investment.DecisionBuy, 200000, 50.0, 0, cfg)
	if res2.Rejected {
		t.Fatalf("unexpected rejection: %s", res2.RejectionError)
	}
	if !res2.WasAdjusted {
		t.Errorf("expected adjustment when target exceeds position limit")
	}
	if res2.Quantity != 2000 {
		t.Errorf("expected 2000 shares adjusted, got %d", res2.Quantity)
	}

	// Scenario 3: Already at max position -> rejected
	res3 := investment.EvaluateTradeLimits(profile, snapshot, constraints, investment.DecisionBuy, 50000, 50.0, 2500, cfg)
	if !res3.Rejected {
		t.Errorf("expected rejection when already at max position limit")
	}

	// Scenario 4: Insufficient cash -> rejected
	brokeSnapshot := &investment.PortfolioSnapshot{
		ID:                  "snap-broke",
		AvailableCash:       1000,
		TotalPortfolioValue: 100000,
	}
	res4 := investment.EvaluateTradeLimits(profile, brokeSnapshot, constraints, investment.DecisionBuy, 50000, 50.0, 0, cfg)
	if !res4.Rejected {
		t.Errorf("expected rejection for insufficient cash")
	}
}
