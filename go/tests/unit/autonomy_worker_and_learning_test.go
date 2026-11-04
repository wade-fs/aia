package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/worker"
)

func TestAutonomyWorker_AutoWake_And_EventTriggers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	contRepo := repository.NewContinuityRepository(db)
	projRepo := repository.NewProjectRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	decRepo := repository.NewDecisionRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	execRepo := repository.NewExecutionRepository(db)

	_ = projRepo.Create(ctx, &project.Project{
		ID:        "default",
		Name:      "Default Project",
		Workspace: "/tmp/default",
		Status:    project.StatusActive,
	})

	// Setup autonomy engine
	autonomyRepo := repository.NewAutonomyRepository(db)
	govRepo := repository.NewGovernanceRepository(db)
	autonomyEng := autonomy.NewEngineWithDeps(autonomy.Dependencies{
		AutonomyRepo:   autonomyRepo,
		DecisionRepo:   decRepo,
		ContinuityRepo: contRepo,
		ExecutionRepo:  execRepo,
		ProjectRepo:    projRepo,
		TaskRepo:       taskRepo,
		GovernanceRepo: govRepo,
		AuditRepo:      auditRepo,
	})

	// Setup Worker Manager
	mgr := worker.NewManager(
		worker.WorkerConfig{ScheduleInterval: 10 * time.Second, TaskInterval: 5 * time.Second},
		db,
		nil, nil, nil, nil, nil, nil, nil, auditRepo,
	)
	mgr.SetAutonomyEngine(autonomyEng, 50*time.Millisecond)

	// 1. Test event-driven trigger dispatch (AIA-08 §13)
	outcome, err := mgr.TriggerAutonomousCycle(ctx, "default", continuity.TriggerProjectUpdated, "Spec test event trigger")
	if err != nil {
		t.Fatalf("TriggerAutonomousCycle failed: %v", err)
	}
	if outcome == nil || outcome.Decision == nil {
		t.Fatalf("expected non-nil outcome with decision")
	}

	// 2. Verify Decision fields (AIA-08 §7.1, §7.3)
	dec := outcome.Decision
	if dec.ActionType != decision.ActionNoAction {
		t.Errorf("expected calm rational NO_ACTION, got %s", dec.ActionType)
	}
	if dec.ReevaluationConditions == "" {
		t.Errorf("expected non-empty ReevaluationConditions for NO_ACTION decision")
	}
	if dec.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0 for calm state, got %.2f", dec.Confidence)
	}

	// 3. Test background auto-wake worker loop (AIA-08 §12.2)
	runCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()

	if err := mgr.Start(runCtx); err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}

	// Wait for at least 1-2 auto-wake ticks
	time.Sleep(120 * time.Millisecond)
	mgr.Stop()

	// Verify cycles were recorded in SQLite
	cycles, err := contRepo.ListCycles(ctx, 10)
	if err != nil {
		t.Fatalf("ListCycles failed: %v", err)
	}
	if len(cycles) < 2 {
		t.Errorf("expected at least 2 cycles recorded (event trigger + periodic worker), got %d", len(cycles))
	}
}

func TestLearning_ExtendedCorrectionTypes_And_GoalEvolutionModes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	learningRepo := repository.NewLearningRepository(db)
	goalRepo := repository.NewGoalRepository(db)

	// 1. Test Extended Correction Types & Root Causes (AIA-11 §4, §6.1)
	evalID := "eval-test-101"
	_ = learningRepo.CreateOutcomeEvaluation(ctx, &learning.OutcomeEvaluation{
		ID:                  evalID,
		Domain:              "general",
		ExpectedOutcomeJSON: "{}",
		ActualOutcomeJSON:   "{}",
		EvaluationStatus:    learning.EvalStatusNeedsRCA,
	})

	rca := &learning.RootCauseAnalysis{
		ID:                     "rca-test-101",
		OutcomeEvaluationID:    evalID,
		RootCauseCategory:      learning.RCEvaluationError, // AIA-11 §4
		AnalysisNarrative:      "Metric benchmark target was improperly calibrated",
		FalsifiableHypothesis:  "Re-evaluating with normalized weights will correct discrepancy",
		CausalEvidenceRefsJSON: `["evidence-101"]`,
		RCAValidityStatus:      learning.RCASupported,
		Confidence:             0.90,
	}
	if err := learningRepo.CreateRCA(ctx, rca); err != nil {
		t.Fatalf("CreateRCA failed: %v", err)
	}

	corr := &learning.CorrectionAction{
		ID:                 "corr-test-101",
		RCAID:              rca.ID,
		ScopeLevel:         learning.ScopeA,
		CorrectionType:     learning.CorrectionEvaluationUpdate, // AIA-11 §6.1
		AuthorizationLevel: learning.AuthC1,
		ValidationLevel:    learning.ValV2,
		TierProfile:        learning.Tier1Auto,
		TargetEntityRef:    "metric_calibration",
		BeforeStateJSON:    `{"threshold": 0.5}`,
		ProposedStateJSON:  `{"threshold": 0.8}`,
		VerificationStatus: learning.VerifPartial, // AIA-11 §10
	}
	if err := learningRepo.CreateCorrectionAction(ctx, corr); err != nil {
		t.Fatalf("CreateCorrectionAction failed: %v", err)
	}

	// 2. Test Learning Record with LearningType & Applicability Conditions (AIA-11 §10.2)
	rec := &learning.LearningRecord{
		ID:                          "lr-test-101",
		Domain:                      "coding",
		OutcomeEvaluationID:         evalID,
		Title:                       "Compiler error cache invalidation pattern",
		LearningLevel:               learning.LevelL2Verified,
		LearningType:                learning.LearnTypePattern, // AIA-11 §10.2
		EffectivenessVerdict:        learning.VerdictEffective,
		ComparisonMethod:            learning.CompPrePostOnly,
		BaselineJSON:                "{}",
		HypothesisJSON:              "{}",
		InterventionJSON:            "{}",
		VerificationJSON:            "{}",
		ApplicabilityConditionsJSON: `{"language": "go", "worktree_mode": "isolated"}`,
		RolloutMode:                 "CANARY",
		VerificationDelayCycles:     3,
		EpistemicConfidence:         0.85,
	}
	if err := learningRepo.CreateLearningRecord(ctx, rec); err != nil {
		t.Fatalf("CreateLearningRecord failed: %v", err)
	}

	// 3. Test Goal Evolution Proposals with DECOMPOSITION / MERGE / RETIREMENT (AIA-09 §17.1, §17.2)
	g := &learning.GoalEvolutionProposal{
		ID:                       "gep-test-101",
		GoalID:                   "goal-primary-001",
		EvolutionMode:            learning.EvolutionDecomposition, // AIA-09 §17.2
		ActorType:                "AGENT",
		ActorID:                  "personal-agent",
		Rationale:                "Breaking strategic objective into three tactical sub-goals",
		PreviousTargetMetricJSON: `{"milestone": "v1.0"}`,
		ProposedTargetMetricJSON: `{"milestone": "v1.0-sub1"}`,
		ExpectedOutcome:          "Clear parallelism across sub-tasks",
		RiskAssessment:           "LOW",
		ApprovedBy:               "system_auto",
		Status:                   learning.EvoApplied,
	}
	// Seed goal first to satisfy foreign key
	if err := goalRepo.Create(ctx, &goal.Goal{
		ID:          "goal-primary-001",
		Name:        "Primary Strategic Goal",
		Description: "Scale system capabilities",
		Level:       goal.LevelStrategic,
		Status:      goal.StatusActive,
	}); err != nil {
		t.Fatalf("goalRepo.Create failed: %v", err)
	}

	if err := learningRepo.CreateGoalEvolutionProposal(ctx, g); err != nil {
		t.Fatalf("CreateGoalEvolutionProposal failed: %v", err)
	}
}
