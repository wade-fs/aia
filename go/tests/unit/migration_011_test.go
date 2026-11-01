package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/goal"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestMigration011_SchemaConstraintsAndThreeTierProfile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	ver, err := db.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("failed to read schema version: %v", err)
	}
	if ver < 11 {
		t.Fatalf("expected schema version >= 11, got %d", ver)
	}

	// 1. Verify Table-Driven 3-Tier Profile Mapping (AIA-11 §21.2)
	tier1, auth1, val1 := domainLearning.ResolveCorrectionProfile(domainLearning.ScopeA, domainLearning.CorrectionJudgment)
	if tier1 != domainLearning.Tier1Auto || auth1 != domainLearning.AuthC1 || val1 != domainLearning.ValV2 {
		t.Fatalf("unexpected Tier 1 mapping: %s / %s / %s", tier1, auth1, val1)
	}
	tier2, auth2, val2 := domainLearning.ResolveCorrectionProfile(domainLearning.ScopeB, domainLearning.CorrectionWorkflow)
	if tier2 != domainLearning.Tier2Approval || auth2 != domainLearning.AuthC3 || val2 != domainLearning.ValV4 {
		t.Fatalf("unexpected Tier 2 mapping: %s / %s / %s", tier2, auth2, val2)
	}
	tier3, auth3, val3 := domainLearning.ResolveCorrectionProfile(domainLearning.ScopeC, domainLearning.CorrectionCode)
	if tier3 != domainLearning.Tier3Blocked || auth3 != domainLearning.AuthC5 || val3 != domainLearning.ValV6 {
		t.Fatalf("unexpected Tier 3 mapping: %s / %s / %s", tier3, auth3, val3)
	}

	// 2. Verify Executions Token Columns & SumTokensSince
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO projects (id, name, workspace, description, status, created_at, updated_at)
		VALUES ('proj-m11', 'Migration 011 Project', '/tmp/m11', 'desc', 'ACTIVE', ?, ?)
	`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	agRepo := repository.NewAgentRepository(db)
	if err := agRepo.Upsert(ctx, &agent.Agent{
		ID:          "coding-agent",
		Name:        "Coding Agent",
		Description: "desc",
		Domain:      "coding",
		Enabled:     true,
		Version:     1,
	}); err != nil {
		t.Fatalf("upsert agent: %v", err)
	}

	taskRepo := repository.NewTaskRepository(db)
	tk := &task.Task{
		ID:        "task-m11-1",
		ProjectID: "proj-m11",
		AgentID:   "coding-agent",
		Title:     "Token tracked task",
		Status:    task.StatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := taskRepo.Create(ctx, tk, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}

	execRepo := repository.NewExecutionRepository(db)
	ex := &execution.Execution{
		ID:           "exec-m11-1",
		TaskID:       tk.ID,
		AgentID:      "coding-agent",
		Provider:     "mock",
		Workspace:    "/tmp/m11/ws",
		Status:       execution.StatusCompleted,
		InputTokens:  320,
		OutputTokens: 180,
		TotalTokens:  500,
		CreatedAt:    now,
	}
	if err := execRepo.Create(ctx, ex); err != nil {
		t.Fatalf("create execution with tokens: %v", err)
	}
	totalUsed, err := execRepo.SumTokensSince(ctx, now.Add(-1*time.Hour))
	if err != nil || totalUsed != 500 {
		t.Fatalf("expected SumTokensSince=500, got %d (err=%v)", totalUsed, err)
	}

	// 3. Verify Learning & Correction Persistence + SQLite CHECK Constraints
	learningRepo := repository.NewLearningRepository(db)
	oe := &domainLearning.OutcomeEvaluation{
		ID:                  "oe-m11-1",
		Domain:              "investment",
		ExpectedOutcomeJSON: `{"brier":0.20}`,
		ActualOutcomeJSON:   `{"brier":0.34}`,
		DiscrepancyScore:    0.14,
		IsAnomaly:           true,
		EvaluationStatus:    domainLearning.EvalStatusNeedsRCA,
		EvaluatedAt:         now,
	}
	if err := learningRepo.CreateOutcomeEvaluation(ctx, oe); err != nil {
		t.Fatalf("create outcome_evaluation: %v", err)
	}

	rca := &domainLearning.RootCauseAnalysis{
		ID:                     "rca-m11-1",
		OutcomeEvaluationID:    oe.ID,
		RootCauseCategory:      domainLearning.RCBiasedJudgment,
		AnalysisNarrative:      "Stale revenue growth assumption ignored inventory cycle turn",
		FalsifiableHypothesis:  "Next 5 forecasts using inventory-adjusted prior must improve Brier by >= 0.03",
		CausalEvidenceRefsJSON: `["evi-1"]`,
		CompetingFactorsJSON:   `["sector-wide multiple compression"]`,
		Confidence:             0.75,
		RCAValidityStatus:      domainLearning.RCAProposed,
		CreatedAt:              now,
	}
	if err := learningRepo.CreateRCA(ctx, rca); err != nil {
		t.Fatalf("create rca: %v", err)
	}

	ca := &domainLearning.CorrectionAction{
		ID:                 "ca-m11-1",
		RCAID:              rca.ID,
		ScopeLevel:         domainLearning.ScopeA,
		CorrectionType:     domainLearning.CorrectionJudgment,
		AuthorizationLevel: domainLearning.AuthC1,
		ValidationLevel:    domainLearning.ValV2,
		TierProfile:        domainLearning.Tier1Auto,
		TargetEntityRef:    "claim:inventory_prior",
		BeforeStateJSON:    `{"prior_weight":1.0}`,
		ProposedStateJSON:  `{"prior_weight":0.6}`,
		RegressionStatus:   domainLearning.RegressionPassed,
		VerificationStatus: domainLearning.VerifEffective,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := learningRepo.CreateCorrectionAction(ctx, ca); err != nil {
		t.Fatalf("create correction_action: %v", err)
	}

	rcaID := rca.ID
	caID := ca.ID
	lr := &domainLearning.LearningRecord{
		ID:                      "lr-m11-1",
		Domain:                  "investment",
		OutcomeEvaluationID:     oe.ID,
		RCAID:                   &rcaID,
		CorrectionActionID:      &caID,
		Title:                   "Discount revenue extrapolation during semiconductor inventory build",
		LearningLevel:           domainLearning.LevelL2Verified,
		EffectivenessVerdict:    domainLearning.VerdictEffective,
		ComparisonMethod:        domainLearning.CompControlBenchmark,
		EpistemicConfidence:     0.82,
		BaselineJSON:            `{"brier":0.34}`,
		HypothesisJSON:          `{"expected_delta":-0.05}`,
		InterventionJSON:        `{"target_entity_ref":"claim:inventory_prior"}`,
		VerificationJSON:        `{"post_brier":0.21}`,
		RegressionJSON:          `{"passed":true}`,
		FollowupJSON:            `[]`,
		GeneralizedCycleIDsJSON: `[]`,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := learningRepo.CreateLearningRecord(ctx, lr); err != nil {
		t.Fatalf("create learning_record: %v", err)
	}

	// CHECK constraint: invalid learning_level must fail
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO learning_records (
			id, outcome_evaluation_id, title, learning_level, effectiveness_verdict,
			comparison_method, baseline_json, hypothesis_json, epistemic_confidence, created_at, updated_at
		) VALUES ('lr-bad-lvl', ?, 'bad', 'L4_MAGIC', 'INCONCLUSIVE', 'PRE_POST_ONLY', '{}', '{}', 0.3, ?, ?)
	`, oe.ID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err == nil {
		t.Fatalf("expected CHECK constraint error for invalid learning_level L4_MAGIC")
	}

	// CHECK constraint: out-of-range epistemic_confidence (> 1.0) must fail
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO learning_records (
			id, outcome_evaluation_id, title, learning_level, effectiveness_verdict,
			comparison_method, baseline_json, hypothesis_json, epistemic_confidence, created_at, updated_at
		) VALUES ('lr-bad-conf', ?, 'bad', 'L0_RECORDED', 'INCONCLUSIVE', 'PRE_POST_ONLY', '{}', '{}', 1.45, ?, ?)
	`, oe.ID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err == nil {
		t.Fatalf("expected CHECK constraint error for epistemic_confidence > 1.0")
	}

	// 4. Verify Goal Evolution Proposals table & CHECK constraints
	goalRepo := repository.NewGoalRepository(db)
	g := &goal.Goal{
		ID:               "goal-m11-1",
		Name:             "Maximize Total Assets",
		Level:            goal.LevelStrategic,
		Status:           goal.StatusActive,
		TargetMetricJSON: `{"return_pct":15}`,
		Priority:         9,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := goalRepo.Create(ctx, g); err != nil {
		t.Fatalf("create goal: %v", err)
	}

	prop := &domainLearning.GoalEvolutionProposal{
		ID:                       "gep-m11-1",
		GoalID:                   g.ID,
		EvolutionMode:            domainLearning.EvolutionPersistence,
		ActorType:                "AGENT",
		ActorID:                  "investment-agent",
		PreviousTargetMetricJSON: g.TargetMetricJSON,
		ProposedTargetMetricJSON: g.TargetMetricJSON,
		InvalidatedClaimIDsJSON:  `[]`,
		Rationale:                "Short-term drawdown within expected variance; premises intact",
		Status:                   domainLearning.EvoApplied,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := learningRepo.CreateGoalEvolutionProposal(ctx, prop); err != nil {
		t.Fatalf("create goal_evolution_proposal: %v", err)
	}

	// Invalid evolution_mode must be rejected by SQLite CHECK constraint
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO goal_evolution_proposals (id, goal_id, evolution_mode, actor_type, actor_id, rationale, status, created_at, updated_at)
		VALUES ('gep-bad-mode', ?, 'SELF_REWRITE', 'AGENT', 'bad', 'bad', 'APPLIED', ?, ?)
	`, g.ID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err == nil {
		t.Fatalf("expected CHECK constraint error for invalid evolution_mode SELF_REWRITE")
	}
}
