package unit

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/investment"
	invService "github.com/wade/aia/internal/service/investment"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type mockMarketProvider struct {
	bar *investment.MarketBarValue
}

func (m *mockMarketProvider) FetchLatestBar(ctx context.Context, symbol string) (*investment.MarketBarValue, error) {
	return m.bar, nil
}

func (m *mockMarketProvider) FetchNewsAndAnnouncements(ctx context.Context, symbol string, limit int) ([]investment.RawDocument, error) {
	return nil, nil
}

func TestMaterialityGate_GatingLogic(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	invRepo := repository.NewInvestmentRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	svc := invService.NewService(invRepo, auditRepo, nil)

	ctx := context.Background()
	_ = svc.EnsureDataSources(ctx)
	prof, err := svc.EnsureDefaultProfile(ctx)
	if err != nil {
		t.Fatalf("failed to ensure profile: %v", err)
	}

	mockProv := &mockMarketProvider{
		bar: &investment.MarketBarValue{
			Close: 1000.0,
		},
	}
	svc.SetDataProvider(mockProv)

	// Case 1: Initial state - No prior decision -> MUST be material
	res1, err := svc.EvaluateMateriality(ctx, "2330", invService.DefaultMaterialityConfig())
	if err != nil {
		t.Fatalf("eval materiality err: %v", err)
	}
	if !res1.IsMaterial {
		t.Errorf("expected initial analysis to be material, got false")
	}

	// Now record a Decision at price 1000.0 (SELL to rebalance overweight position)
	prop, err := svc.ProcessAgyDecision(ctx, investment.AgyDecisionInput{
		Symbol:          "2330",
		Action:          "SELL",
		Confidence:      0.80,
		Reasoning:       []string{"Rebalance overweight position"},
		TargetAmount:    0,
		ExpectedHorizon: "20D",
		StrategyVersion: "v1.0",
	}, 1000.0)
	if err != nil {
		t.Fatalf("failed to process initial decision: %v", err)
	}
	if prop == nil {
		t.Fatal("expected proposal created")
	}

	// Case 2: Daily price moved only 1.0% (to 1010.0), no new announcements -> MUST NOT be material
	mockProv.bar = &investment.MarketBarValue{Close: 1010.0}
	res2, err := svc.EvaluateMateriality(ctx, "2330", invService.DefaultMaterialityConfig())
	if err != nil {
		t.Fatalf("eval materiality err: %v", err)
	}
	if res2.IsMaterial {
		t.Errorf("expected minor 1%% move to NOT be material, got is_material=true (reason: %s)", res2.Reason)
	}

	// Preserve previous decision
	if err := svc.PreserveExistingDecision(ctx, "2330", res2.Reason); err != nil {
		t.Fatalf("failed to preserve decision: %v", err)
	}

	// Case 3: Price moves significantly > 3.0% (to 1050.0 = +5.0%) -> MUST be material
	mockProv.bar = &investment.MarketBarValue{Close: 1050.0}
	res3, err := svc.EvaluateMateriality(ctx, "2330", invService.DefaultMaterialityConfig())
	if err != nil {
		t.Fatalf("eval materiality err: %v", err)
	}
	if !res3.IsMaterial {
		t.Errorf("expected 5%% price move to be material, got false")
	}

	// Case 4: Price moved only 0.5%, BUT a new MOPS announcement arrived -> MUST be material
	mockProv.bar = &investment.MarketBarValue{Close: 1005.0}
	inst, _ := invRepo.GetInstrumentBySymbol(ctx, "", "2330")
	now := time.Now().UTC()
	_ = invRepo.SaveRawDocument(ctx, &investment.RawDocument{
		ID:               uuid.New().String(),
		SourceID:         "ds-twse-openapi",
		InstrumentID:     inst.ID,
		SourceRecordID:   "MOPS-TEST-001",
		Title:            "重大訊息：董事會核准新增資本支出",
		Summary:          "因應先進製程需求大增，通過數百億資本支出",
		ContentReference: "mops",
		ContentHash:      "hash-12345",
		ContentType:      "ANNOUNCEMENT",
		PublishedAt:      now.Add(1 * time.Minute), // published AFTER decision
		FetchedAt:        now,
	})

	res4, err := svc.EvaluateMateriality(ctx, "2330", invService.DefaultMaterialityConfig())
	if err != nil {
		t.Fatalf("eval materiality err: %v", err)
	}
	if !res4.IsMaterial {
		t.Errorf("expected new MOPS announcement to be material, got false")
	}
	if res4.NewDocumentsCount != 1 {
		t.Errorf("expected 1 new document, got %d", res4.NewDocumentsCount)
	}
	_ = prof
}
