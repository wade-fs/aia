package unit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy/deliberation"
	"github.com/wade/aia/internal/autonomy/recall"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/domain/knowledge"
	"github.com/wade/aia/internal/domain/memory"
	goalService "github.com/wade/aia/internal/service/goal"
	invService "github.com/wade/aia/internal/service/investment"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestRecallAndMemoryGovernance_ProvenanceAndConflicts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	memRepo := repository.NewMemoryRepository(db)
	knowRepo := repository.NewKnowledgeRepository(db)
	eventRepo := repository.NewEventHistoryRepository(db)
	contRepo := repository.NewContinuityRepository(db)
	goalRepo := repository.NewGoalRepository(db)
	decRepo := repository.NewDecisionRepository(db)
	projRepo := repository.NewProjectRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	recallEng := recall.NewEngine(eventRepo, knowRepo, memRepo, contRepo, goalRepo, decRepo, projRepo, taskRepo)

	// 1. Provenance Integrity Matrix: HYPOTHESIS cannot be marked VERIFIED
	hypMem := &memory.Memory{
		Title:             "Speculative Market Hypothesis",
		Content:           "Semiconductor cycle might peak next quarter",
		Type:              memory.TypeFact,
		Scope:             memory.ScopeGlobal,
		Confidence:        0.6,
		ProvenanceType:    "HYPOTHESIS",
		ProvenanceRefType: "NONE",
		ProvenanceStatus:  "VERIFIED",
	}
	err := recallEng.WriteGovernedMemory(ctx, hypMem)
	if !errors.Is(err, recall.ErrHypothesisCannotBeVerified) {
		t.Fatalf("expected ErrHypothesisCannotBeVerified, got %v", err)
	}

	// 2. FACT_BASED cannot be VERIFIED without EVIDENCE or SOURCE reference
	unanchoredFact := &memory.Memory{
		Title:             "Unanchored Fact",
		Content:           "Revenue grew 20%",
		Type:              memory.TypeFact,
		Scope:             memory.ScopeGlobal,
		Confidence:        0.9,
		ProvenanceType:    "FACT_BASED",
		ProvenanceRefType: "NONE",
		ProvenanceStatus:  "VERIFIED",
	}
	err = recallEng.WriteGovernedMemory(ctx, unanchoredFact)
	if !errors.Is(err, recall.ErrFactMissingProvenanceRef) {
		t.Fatalf("expected ErrFactMissingProvenanceRef, got %v", err)
	}

	// 3. Valid FACT_BASED with EVIDENCE ref succeeds
	eviID := "evi-valid-1"
	anchoredFact := &memory.Memory{
		Title:             "TSMC Q3 Revenue",
		Content:           "Consolidated revenue reached NT$759.69 billion",
		Type:              memory.TypeFact,
		Scope:             memory.ScopeProject,
		ProjectID:         "proj-1",
		Confidence:        0.95,
		ProvenanceType:    "FACT_BASED",
		ProvenanceRefType: "EVIDENCE",
		ProvenanceRefID:   &eviID,
		ProvenanceStatus:  "VERIFIED",
	}
	if err := recallEng.WriteGovernedMemory(ctx, anchoredFact); err != nil {
		t.Fatalf("expected anchored FACT_BASED memory to succeed, got %v", err)
	}

	// 4. Temporal expiry: expired memory is filtered out during Recall
	past := now.Add(-2 * time.Hour)
	expiredMem := &memory.Memory{
		Title:            "Stale Ephemeral Cache",
		Content:          "Temporary build lock held",
		Type:             memory.TypeFact,
		Scope:            memory.ScopeProject,
		ProjectID:        "proj-1",
		Confidence:       0.9,
		ExpiresAt:        &past,
		ProvenanceType:   "INFERENCE",
		ProvenanceStatus: "UNVERIFIED",
	}
	if err := recallEng.WriteGovernedMemory(ctx, expiredMem); err != nil {
		t.Fatalf("failed to write expired memory: %v", err)
	}

	// 5. Conflict preservation: second memory on same subject with contradictory assertion
	contradictoryMem := &memory.Memory{
		Title:            "TSMC Q3 Revenue",
		Content:          "Consolidated revenue declined to NT$500 billion",
		Type:             memory.TypeFact,
		Scope:            memory.ScopeProject,
		ProjectID:        "proj-1",
		Confidence:       0.70,
		ProvenanceType:   "INFERENCE",
		ProvenanceStatus: "UNVERIFIED",
	}
	if err := recallEng.WriteGovernedMemory(ctx, contradictoryMem); err != nil {
		t.Fatalf("failed to write contradictory memory: %v", err)
	}

	// Also insert a CONFLICTED claim in L3 knowledge store
	_ = knowRepo.CreateClaim(ctx, &knowledge.Claim{
		ID:              "claim-conflict-1",
		Statement:       "Capex guidance revised upward vs downward",
		AssertionStatus: knowledge.AssertionConflicted,
		CurrentVersion:  1,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	bundle, err := recallEng.Recall(ctx, recall.RecallInput{
		ProjectID: "proj-1",
		Observations: []domainAutonomy.Observation{
			{Source: "TEST", Summary: "Review TSMC Q3 Revenue"},
		},
	})
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	for _, m := range bundle.Memories {
		if m.Title == "Stale Ephemeral Cache" {
			t.Errorf("expired memory should have been excluded from RecalledBundle")
		}
	}
	if len(bundle.Conflicts) < 2 {
		t.Errorf("expected both memory contradiction and conflicted claim in bundle.Conflicts, got %d", len(bundle.Conflicts))
	}
}

func TestDeliberator_AllFiveDecisionTypes(t *testing.T) {
	delib := deliberation.NewDeliberator()

	// 1. NO_ACTION on zero candidates
	resIdle := delib.Deliberate(deliberation.DeliberateInput{})
	if resIdle.RecommendedAction != decision.ActionNoAction {
		t.Errorf("expected NO_ACTION on empty candidates, got %s", resIdle.RecommendedAction)
	}

	// 2. NO_ACTION on negative-EV low-priority noise
	resNoise := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{
			{
				ID:                  "c-noise",
				DiscoveryType:       domainAutonomy.DiscoveryImprovement,
				ConfidenceLevel:     domainAutonomy.ConfidenceProbable,
				DetectedOpportunity: "Cosmetic whitespace tweak",
				PriorityScore:       1,
				EstimatedCost:       4.0,
				EstimatedRisk:       governance.RiskTierLow,
			},
		},
	})
	if resNoise.RecommendedAction != decision.ActionNoAction {
		t.Errorf("expected NO_ACTION for negative net utility candidate, got %s", resNoise.RecommendedAction)
	}

	// 3. ACT on high-EV confirmed opportunity
	resAct := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{
			{
				ID:                  "c-act",
				DiscoveryType:       domainAutonomy.DiscoveryGoalGap,
				ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
				DetectedOpportunity: "Close 40% unit test coverage gap",
				PriorityScore:       8,
				EstimatedCost:       1.5,
				EstimatedRisk:       governance.RiskTierLow,
			},
		},
	})
	if resAct.RecommendedAction != decision.ActionAct {
		t.Errorf("expected ACT, got %s", resAct.RecommendedAction)
	}

	// 4. MONITOR on potential early drift
	resMon := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{
			{
				ID:                  "c-mon",
				DiscoveryType:       domainAutonomy.DiscoveryContinuity,
				ConfidenceLevel:     domainAutonomy.ConfidencePotential,
				DetectedOpportunity: "Subtle latency drift (+4%)",
				PriorityScore:       4,
				EstimatedCost:       2.0,
				EstimatedRisk:       governance.RiskTierMedium,
				MetadataJSON:        `{"category":"DRIFT"}`,
			},
		},
	})
	if resMon.RecommendedAction != decision.ActionMonitor {
		t.Errorf("expected MONITOR on POTENTIAL drift, got %s", resMon.RecommendedAction)
	}

	// 5. DEFER when active task queue is at capacity
	resDefer := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{
			{
				ID:                  "c-defer",
				DiscoveryType:       domainAutonomy.DiscoveryOpportunity,
				ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
				DetectedOpportunity: "Refactor collector retry helper",
				PriorityScore:       6,
				EstimatedCost:       1.5,
				EstimatedRisk:       governance.RiskTierLow,
			},
		},
		ActiveTaskCount:          5,
		MaxConcurrentActiveTasks: 5,
	})
	if resDefer.RecommendedAction != decision.ActionDefer {
		t.Errorf("expected DEFER when queue is at capacity, got %s", resDefer.RecommendedAction)
	}

	// 6. ASK_HUMAN on unresolved knowledge conflict
	resAsk := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{
			{
				ID:                  "c-ask",
				DiscoveryType:       domainAutonomy.DiscoveryIssue,
				ConfidenceLevel:     domainAutonomy.ConfidenceUnverified,
				DetectedOpportunity: "Conflicting portfolio risk tolerance rules",
				PriorityScore:       7,
				EstimatedCost:       0.5,
				EstimatedRisk:       governance.RiskTierMedium,
				MetadataJSON:        `{"requires_human_resolution":true}`,
			},
		},
	})
	if resAsk.RecommendedAction != decision.ActionAskHuman {
		t.Errorf("expected ASK_HUMAN on conflict requiring human resolution, got %s", resAsk.RecommendedAction)
	}
}

func TestGoalHierarchyCyclePrevention_And_GapEvaluation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	goalRepo := repository.NewGoalRepository(db)

	gA := &goal.Goal{ID: "g-A", Name: "Strategic Goal A", Level: goal.LevelStrategic, Status: goal.StatusActive, Priority: 9}
	if err := goalRepo.Create(ctx, gA); err != nil {
		t.Fatalf("create g-A failed: %v", err)
	}
	idA := gA.ID
	gB := &goal.Goal{ID: "g-B", ParentGoalID: &idA, Name: "Tactical Goal B", Level: goal.LevelTactical, Status: goal.StatusActive, Priority: 7}
	if err := goalRepo.Create(ctx, gB); err != nil {
		t.Fatalf("create g-B failed: %v", err)
	}
	idB := gB.ID
	gC := &goal.Goal{
		ID:                "g-C",
		ParentGoalID:      &idB,
		Name:              "Operational Goal C",
		Level:             goal.LevelOperational,
		Status:            goal.StatusActive,
		TargetMetricJSON:  `{"coverage":80,"pass_rate":100}`,
		CurrentMetricJSON: `{"coverage":60,"pass_rate":100}`,
		Priority:          6,
	}
	if err := goalRepo.Create(ctx, gC); err != nil {
		t.Fatalf("create g-C failed: %v", err)
	}

	// Attempting to make g-A a child of g-C creates cycle A -> C -> B -> A and MUST fail
	idC := gC.ID
	err := goalRepo.UpdateParent(ctx, gA.ID, &idC, "Attempt invalid cycle")
	if err == nil || !strings.Contains(err.Error(), "circular reference error") {
		t.Fatalf("expected circular reference error on A -> C -> B -> A, got %v", err)
	}

	gap := goalService.EvaluateGoalGap(gC)
	if !gap.HasGap {
		t.Fatalf("expected unmet metric gap on g-C")
	}
	if gap.GapRatio < 0.24 || gap.GapRatio > 0.26 {
		t.Errorf("expected gap ratio ~0.25 (60 vs 80), got %.3f", gap.GapRatio)
	}
}

func TestInvestmentVirtualLedgerSettlement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	invRepo := repository.NewInvestmentRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	svc := invService.NewService(invRepo, auditRepo, nil)

	if _, err := svc.EnsureDefaultProfile(ctx); err != nil {
		t.Fatalf("EnsureDefaultProfile failed: %v", err)
	}

	// Propose a virtual BUY of 0050 at NT$180.0 (1,000 shares = NT$180,000 + NT$256 fee = NT$180,256)
	prop, err := svc.ProcessAgyDecision(ctx, investment.AgyDecisionInput{
		Symbol:          "0050",
		Exchange:        "TWSE",
		Action:          "BUY",
		TargetAmount:    190000,
		Confidence:      0.82,
		Reasoning:       []string{"Broad market ETF allocation within 25% position cap"},
		ExpectedHorizon: "20D",
		StrategyVersion: "unit-test-v1",
	}, 180.0)
	if err != nil || prop == nil {
		t.Fatalf("ProcessAgyDecision BUY failed: %v", err)
	}
	if prop.Quantity != 1000 {
		t.Fatalf("expected 1,000 shares (1 lot), got %d", prop.Quantity)
	}

	newSnap, err := svc.ExecuteVirtualProposal(ctx, prop.ID)
	if err != nil {
		t.Fatalf("ExecuteVirtualProposal failed: %v", err)
	}

	expectedCash := int64(600000) - prop.EstimatedTotalCost
	if newSnap.AvailableCash != expectedCash {
		t.Errorf("expected cash %d after virtual BUY settlement, got %d", expectedCash, newSnap.AvailableCash)
	}

	updatedProp, err := invRepo.GetTradeProposal(ctx, prop.ID)
	if err != nil {
		t.Fatalf("GetTradeProposal failed: %v", err)
	}
	if updatedProp.Status != investment.ProposalExecuted {
		t.Errorf("expected proposal status EXECUTED, got %s", updatedProp.Status)
	}
}
