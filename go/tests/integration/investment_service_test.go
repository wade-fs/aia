package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/eventbus"
	invService "github.com/wade/aia/internal/service/investment"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestInvestmentService_EndToEnd(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "inv_test.db")
	db, err := sqlite.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	bus := eventbus.New(100)
	defer bus.Close()

	invRepo := repository.NewInvestmentRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	svc := invService.NewService(invRepo, auditRepo, bus)

	ctx := context.Background()

	// 1. Ensure default profile & snapshot
	prof, err := svc.EnsureDefaultProfile(ctx)
	if err != nil {
		t.Fatalf("failed to ensure profile: %v", err)
	}
	if prof.Name != "Taiwan Stock Strategy" {
		t.Errorf("expected Taiwan Stock Strategy, got %s", prof.Name)
	}

	snap, positions, err := svc.GetSnapshot(ctx)
	if err != nil {
		t.Fatalf("failed to get snapshot: %v", err)
	}
	if snap.AvailableCash != 600000 {
		t.Errorf("expected 600k cash, got %d", snap.AvailableCash)
	}
	if len(positions) == 0 {
		t.Errorf("expected seeded positions")
	}

	// 2. Process BUY decision for Foxconn (2317) at NT$ 150/share
	// 1 lot (1,000 shares) = NT$ 150,000. Target amount NT$ 160,000 -> 1 lot
	input := investment.AgyDecisionInput{
		Symbol:            "2317",
		Exchange:          "TWSE",
		Action:            "BUY",
		TargetAmount:      160000,
		Confidence:        0.85,
		Reasoning:         []string{"Strong AI server shipment growth"},
		Risks:             []string{"Margin pressure"},
		InvalidationRules: []string{"Revenue drops > 10% YoY"},
		ExpectedHorizon:   "6M",
		StrategyVersion:   "v1.0",
	}

	prop, err := svc.ProcessAgyDecision(ctx, input, 150.0)
	if err != nil {
		t.Fatalf("process decision failed: %v", err)
	}
	if prop == nil {
		t.Fatalf("expected non-nil trade proposal")
	}
	if prop.Action != investment.DecisionBuy {
		t.Errorf("expected action BUY, got %s", prop.Action)
	}
	if prop.Quantity != 1000 {
		t.Errorf("expected 1000 shares (1 lot), got %d", prop.Quantity)
	}
	if prop.EstimatedGrossAmount != 150000 {
		t.Errorf("expected gross 150000, got %d", prop.EstimatedGrossAmount)
	}
	if prop.Status != investment.ProposalPending {
		t.Errorf("expected status PENDING, got %s", prop.Status)
	}

	// 3. User Approves Proposal
	if err := svc.ApproveProposal(ctx, prop.ID); err != nil {
		t.Fatalf("failed to approve proposal: %v", err)
	}
	approved, err := invRepo.GetTradeProposal(ctx, prop.ID)
	if err != nil {
		t.Fatalf("failed to reload proposal: %v", err)
	}
	if approved.Status != investment.ProposalApproved {
		t.Errorf("expected APPROVED, got %s", approved.Status)
	}

	// 4. Low Confidence Decision is rejected
	lowConfInput := investment.AgyDecisionInput{
		Symbol:     "2317",
		Action:     "BUY",
		Confidence: 0.35, // below 0.50 threshold
	}
	if _, err := svc.ProcessAgyDecision(ctx, lowConfInput, 150.0); err == nil {
		t.Errorf("expected low confidence decision to be rejected")
	}

	// Verify rejection recorded in database
	rejections, err := svc.ListRejections(ctx, 10)
	if err != nil || len(rejections) == 0 {
		t.Errorf("expected rejection to be recorded in db")
	}

	// 5. Invalid symbol is rejected
	invalidSymbolInput := investment.AgyDecisionInput{
		Symbol:     "NONEXISTENT",
		Action:     "BUY",
		Confidence: 0.90,
	}
	if _, err := svc.ProcessAgyDecision(ctx, invalidSymbolInput, 100.0); err == nil {
		t.Errorf("expected invalid symbol to be rejected")
	}
}
