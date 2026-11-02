package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/coding"
	"github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/service/delegation"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestCodeHealthObserver_CleanState(t *testing.T) {
	ctx := context.Background()
	obs := autonomy.NewCodeHealthObserver()
	obs.RegisterProjectDir("clean-proj", "/tmp/mock-clean")
	obs.SetTestRunner(func(ctx context.Context, dir string) (*coding.TestRunResult, error) {
		return &coding.TestRunResult{
			Passed:     true,
			ExitCode:   0,
			TotalTests: 42,
			ExecutedAt: time.Now(),
		}, nil
	})

	results, err := obs.Observe(ctx, "clean-proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 baseline observation, got %d", len(results))
	}
	if !results[0].IsCalmSnapshot() {
		t.Errorf("expected calm baseline snapshot, got non-calm: %+v", results[0])
	}
	if results[0].TargetAgentID != "coding-agent" {
		t.Errorf("expected target agent coding-agent, got %s", results[0].TargetAgentID)
	}
}

func TestCodeHealthObserver_DefectDetection(t *testing.T) {
	ctx := context.Background()
	obs := autonomy.NewCodeHealthObserver()
	obs.RegisterProjectDir("buggy-proj", "/tmp/mock-buggy")
	obs.SetTestRunner(func(ctx context.Context, dir string) (*coding.TestRunResult, error) {
		return &coding.TestRunResult{
			Passed:      false,
			ExitCode:    1,
			TotalTests:  50,
			FailedTests: []string{"TestTransactionSafety"},
			Output:      "FAIL: TestTransactionSafety (sqlite busy)",
			ExecutedAt:  time.Now(),
		}, nil
	})

	results, err := obs.Observe(ctx, "buggy-proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected at least 1 anomaly observation for failed tests")
	}
	anomaly := results[0]
	if anomaly.Category != domainAutonomy.ObsCategoryAnomaly {
		t.Errorf("expected category %s, got %s", domainAutonomy.ObsCategoryAnomaly, anomaly.Category)
	}
	if anomaly.NonActionable {
		t.Errorf("expected actionable anomaly, got NonActionable=true")
	}
	if anomaly.Confidence != domainAutonomy.ConfidenceConfirmed {
		t.Errorf("expected CONFIRMED confidence, got %s", anomaly.Confidence)
	}
}

func TestManuscriptConsistencyObserver_RealYuanNovel(t *testing.T) {
	ctx := context.Background()
	obs := autonomy.NewManuscriptConsistencyObserver()
	obs.RegisterProjectPath("yuan", "/home/wade/src/github/aia/novel/yuan")

	results, err := obs.Observe(ctx, "yuan")
	if err != nil {
		t.Fatalf("observe yuan novel failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected baseline observations for real yuan manuscript")
	}
	// The real novel should start with a clean baseline snapshot (all verified)
	if !results[0].IsCalmSnapshot() {
		t.Errorf("expected clean baseline for novel, got: %+v", results[0])
	}
	if results[0].TargetAgentID != "writing-agent" {
		t.Errorf("expected target writing-agent, got %s", results[0].TargetAgentID)
	}
}

func TestManuscriptConsistencyObserver_ArtisticLicenseVsHistoricalContradiction(t *testing.T) {
	ctx := context.Background()
	obs := autonomy.NewManuscriptConsistencyObserver()
	obs.RegisterProjectPath("test-novel", "/tmp/novel")

	// 1. Deliberate fiction (artistic license: folklore character Qingnangzi)
	obs.InjectAnomaly("test-novel", writing.ConsistencyAnomaly{
		ID:                  "anom-fiction-qingnang",
		AnomalyType:         writing.AnomalyFictionDeviation,
		CharacterName:       "青囊子",
		ChapterRef:          "volume1/ch03",
		Description:         "民間隱士傳授奇門醫道（合理小說虛構）",
		KnowledgeTier:       writing.TierFiction,
		IsDeliberateFiction: true,
		Severity:            2,
	})

	// 2. Real historical contradiction (post-mortem contradiction: Tuotuo in 1360)
	obs.InjectAnomaly("test-novel", writing.ConsistencyAnomaly{
		ID:                  "anom-hist-tuotuo",
		AnomalyType:         writing.AnomalyPostMortemAppearance,
		CharacterName:       "脫脫",
		ChapterRef:          "volume3/ch12",
		StatedYear:          1360,
		DeathYear:           1355,
		Description:         "脫脫於 1355 年被賜死，不可在 1360 年繼續統軍出現在前線",
		KnowledgeTier:       writing.TierFactHistory,
		IsDeliberateFiction: false,
		Severity:            9,
	})

	results, err := obs.Observe(ctx, "test-novel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 observations (1 deliberate fiction + 1 contradiction), got %d", len(results))
	}

	var fictionObs, contradictionObs *domainAutonomy.Observation
	for i := range results {
		if results[i].NonActionable {
			fictionObs = &results[i]
		} else {
			contradictionObs = &results[i]
		}
	}

	if fictionObs == nil {
		t.Fatalf("expected non-actionable observation for deliberate fiction")
	}
	if fictionObs.Category != domainAutonomy.ObsCategoryStateSnapshot {
		t.Errorf("expected fiction license to be STATE_SNAPSHOT, got %s", fictionObs.Category)
	}

	if contradictionObs == nil {
		t.Fatalf("expected actionable anomaly for historical fact contradiction")
	}
	if contradictionObs.Category != domainAutonomy.ObsCategoryAnomaly {
		t.Errorf("expected historical contradiction to be ANOMALY, got %s", contradictionObs.Category)
	}
	if contradictionObs.Severity != 9 {
		t.Errorf("expected high severity 9, got %d", contradictionObs.Severity)
	}
}

func TestDelegationService_Lifecycle(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	expRepo := repository.NewExperimentRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	svc := delegation.NewService(expRepo, nil, auditRepo)

	// 1. Self-delegation should fail
	_, err := svc.RequestDelegation(ctx, delegation.CreateDelegationInput{
		RequesterAgentID: "coding-agent",
		TargetAgentID:    "coding-agent",
	})
	if err == nil {
		t.Fatalf("expected error delegating to self, got nil")
	}

	// 2. Writing Agent delegates timeline scanner tool to Coding Agent
	req, err := svc.RequestDelegation(ctx, delegation.CreateDelegationInput{
		RequesterAgentID: "writing-agent",
		TargetAgentID:    "coding-agent",
		TaskSpec: map[string]any{
			"title":       "Build Timeline Anomaly Checker CLI",
			"description": "Parse character dates and assert no post-mortem appearances",
		},
		AcceptanceCriteria: map[string]any{
			"test_suite": "go test ./tests/unit/coding_writing_domain_test.go",
			"min_pass":   100,
		},
	})
	if err != nil {
		t.Fatalf("request delegation failed: %v", err)
	}
	if req.Status != "PENDING" {
		t.Errorf("expected status PENDING, got %s", req.Status)
	}

	// 3. Coding Agent accepts
	if err := svc.AcceptDelegation(ctx, req.ID); err != nil {
		t.Fatalf("accept delegation failed: %v", err)
	}
	del, _ := svc.Get(ctx, req.ID)
	if del.Status != "IN_PROGRESS" {
		t.Errorf("expected status IN_PROGRESS, got %s", del.Status)
	}

	// 4. Coding Agent delivers work
	if err := svc.DeliverWork(ctx, req.ID, map[string]any{
		"artifact_type": "CLI_TOOL",
		"binary":        "verify_timeline_anomaly",
		"tests_passed":  true,
	}); err != nil {
		t.Fatalf("deliver work failed: %v", err)
	}
	del, _ = svc.Get(ctx, req.ID)
	if del.Status != "DELIVERED" {
		t.Errorf("expected status DELIVERED, got %s", del.Status)
	}

	// 5. Writing Agent verifies delivery (Passed)
	if err := svc.VerifyDelivery(ctx, req.ID, true, "Tool correctly flagged Tuotuo anomaly in test suite"); err != nil {
		t.Fatalf("verify delivery failed: %v", err)
	}
	del, _ = svc.Get(ctx, req.ID)
	if del.Status != "VERIFIED" {
		t.Errorf("expected status VERIFIED, got %s", del.Status)
	}

	// 6. List delegations
	list, err := svc.List(ctx, "writing-agent", "coding-agent", "VERIFIED", 10)
	if err != nil {
		t.Fatalf("list delegations failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 verified delegation, got %d", len(list))
	}
}
