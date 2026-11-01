package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/autonomy/learning"
	"github.com/wade/aia/internal/domain/coding"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	domainGoal "github.com/wade/aia/internal/domain/goal"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/service/delegation"
	goalService "github.com/wade/aia/internal/service/goal"
)

// TestPhase5B_EntryGate_UniformAutonomyCore verifies Criterion 1 of Phase 5 Entry Gate (AIA-15 §22):
// Investment, Coding, and Writing domain agents share the exact same 11-step Autonomy Core
// and all produce rational NO_ACTION when operating in a calm baseline state.
func TestPhase5B_EntryGate_UniformAutonomyCore(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	// Verify observers and pilots are wired on app
	if app.CodingObserver == nil || app.WritingObserver == nil {
		t.Fatalf("expected coding and writing observers to be initialized on app")
	}
	if app.DelegationSvc == nil {
		t.Fatalf("expected delegation service to be initialized on app")
	}

	// Run clean autonomy cycle across all domains
	outcome, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   "default",
		TriggerType: continuity.TriggerScheduled,
		Summary:     "Phase 5 Entry Gate - Calm Baseline Observation across Investment, Coding & Writing",
	})
	if err != nil {
		t.Fatalf("clean autonomy cycle failed: %v", err)
	}

	if outcome.Decision.ActionType != decision.ActionNoAction {
		t.Fatalf("expected rational NO_ACTION on calm baseline, got %s", outcome.Decision.ActionType)
	}
	if outcome.TaskID != nil {
		t.Errorf("expected no spawned tasks on NO_ACTION, got %s", *outcome.TaskID)
	}

	// Verify observations were collected across the domains
	if len(outcome.Observations) == 0 {
		t.Errorf("expected baseline observations recorded across registered observers")
	}

	// Verify cycle state in continuity
	cyc, err := app.ContinuityRepo.GetCycle(ctx, outcome.CycleID)
	if err != nil || cyc == nil {
		t.Fatalf("failed to retrieve cycle from continuity repo: %v", err)
	}
	if cyc.Status != continuity.CycleStatusCompleted {
		t.Errorf("expected cycle status COMPLETED, got %s", cyc.Status)
	}
}

// TestPhase5B_EntryGate_CodingSelfCorrectionWithTestBaseline verifies Criterion 2 (AIA-15 §22, §24):
// Coding Agent detects a test defect, triggers autonomous initiative, executes sandboxed repair,
// and reaches L2_VERIFIED based on deterministic test suite verification.
func TestPhase5B_EntryGate_CodingSelfCorrectionWithTestBaseline(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	testWorkspace := "/tmp/mock-coding-workspace"
	app.CodingObserver.RegisterProjectDir("default", testWorkspace)

	// Step A: Seed a failing test defect in CodingObserver
	now := time.Now().UTC()
	defect := coding.CodeDefect{
		ID:          "defect-sqlite-busy-001",
		ProjectID:   "default",
		Workspace:   testWorkspace,
		DefectType:  coding.DefectTestFailure,
		Severity:    coding.SeverityHigh,
		Summary:     "TestConcurrentTransactions failed: database table is locked",
		ErrorOutput: "FAIL: TestConcurrentTransactions - sqlite busy timeout exceeded",
		FailedTests: []string{"TestConcurrentTransactions"},
		DetectedAt:  now,
	}
	app.CodingObserver.InjectDefect("default", defect)

	// Configure pilot executor test runner to verify fix in sandbox
	patched := false
	pilot := autonomy.NewCodingPilotExecutor(app.TaskSvc, func(ctx context.Context, dir string) (*coding.TestRunResult, error) {
		if !patched {
			return &coding.TestRunResult{Passed: false, ExitCode: 1, Output: "FAIL: TestConcurrentTransactions"}, nil
		}
		return &coding.TestRunResult{Passed: true, ExitCode: 0, TotalTests: 15, Output: "PASS: 15/15 tests OK"}, nil
	})
	app.AutonomyEngine.RegisterPilot(pilot)

	// Step B: Run Autonomy Cycle - should detect anomaly, deliberate ACT, and execute pilot
	cycleOutcome, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   "default",
		TriggerType: continuity.TriggerScheduled,
		Summary:     "Coding Agent Test-Driven Diagnostic Cycle",
	})
	if err != nil {
		t.Fatalf("cycle execution failed: %v", err)
	}

	if cycleOutcome.Decision.ActionType != decision.ActionAct {
		t.Fatalf("expected ActionAct on high severity defect, got %s", cycleOutcome.Decision.ActionType)
	}
	if cycleOutcome.TaskID == nil {
		t.Fatalf("expected autonomous task spawned for defect repair")
	}

	// Step C: Apply patch and run Learning Engine verification (Phase 4 integration)
	patched = true
	app.CodingObserver.ClearDefects("default")

	learnEng := app.AutonomyEngine.LearningEngine()
	projID := "default"
	oe, rec, err := learnEng.RecordDiscrepancy(ctx, learning.DiscrepancyInput{
		CycleID:        &cycleOutcome.CycleID,
		ProjectID:      &projID,
		Domain:         "coding",
		Title:          "SQLite concurrent writer lock timeout under load",
		MetricName:     "test_pass_rate",
		ExpectedMetric: 1.0,
		ActualMetric:   0.0,
		LowerIsBetter:  false,
		ActualContext: map[string]any{
			"failed_tests": 1,
			"total_tests":  15,
		},
	})
	if err != nil {
		t.Fatalf("record discrepancy failed: %v", err)
	}

	evi := seedEvidence(t, ctx, app.KnowledgeRepo, "evi-coding-test-fail-1", "artifact://test_output.log",
		"TestConcurrentTransactions failed with database table is locked error")

	rca, err := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe.ID,
		LearningRecordID:      rec.ID,
		Category:              domainLearning.RCWorkflowDefect,
		Narrative:             "Missing PRAGMA busy_timeout configuration causes immediate SQLITE_BUSY under concurrency",
		FalsifiableHypothesis: "Under 50-thread concurrent write replay, pass rate increases to 1.0 with zero SQLITE_BUSY errors",
		CausalEvidenceRefs:    []string{evi.ID},
	})
	if err != nil {
		t.Fatalf("rca failed: %v", err)
	}

	act, _, err := learnEng.ProposeAndApplyCorrection(ctx, learning.CorrectionInput{
		RCAID:            rca.ID,
		LearningRecordID: rec.ID,
		Scope:            domainLearning.ScopeA,
		CorrectionType:   domainLearning.CorrectionPrompt,
		TargetEntityRef:  "config/sqlite/busy_timeout",
		BeforeState:      map[string]any{"busy_timeout": 0},
		ProposedState:    map[string]any{"busy_timeout": 5000},
	})
	if err != nil {
		t.Fatalf("propose correction failed: %v", err)
	}
	if act == nil {
		t.Fatalf("expected created correction action")
	}

	// Step D: Verify Correction with empirical test suite -> Must reach L2_VERIFIED
	verifiedRecord, err := learnEng.VerifyCorrection(ctx, learning.VerificationInput{
		LearningRecordID:      rec.ID,
		ComparisonMethod:      domainLearning.CompPiTReplay,
		BaselineMetric:        0.0,
		PostMetric:            1.0,
		LowerIsBetter:         false,
		HasControl:            true,
		ControlBaselineMetric: 0.0,
		ControlPostMetric:     0.0,
		MinExcessImprovement:  0.5,
		SampleSize:            15,
		MinSamples:            5,
		RegressionPassed:      true,
		ProposedConfidence:    0.95,
	})
	if err != nil {
		t.Fatalf("verify correction failed: %v", err)
	}

	if verifiedRecord.LearningLevel != domainLearning.LevelL2Verified {
		t.Errorf("expected LearningLevel %s, got %s", domainLearning.LevelL2Verified, verifiedRecord.LearningLevel)
	}
	if verifiedRecord.EffectivenessVerdict != domainLearning.VerdictEffective {
		t.Errorf("expected VerdictEffective, got %s", verifiedRecord.EffectivenessVerdict)
	}
}

// TestPhase5B_EntryGate_WritingConsistencyWithHistoricalSkeleton verifies Criterion 3 (AIA-15 §22, §23):
// Writing Agent distinguishes intentional artistic license (rational NO_ACTION) from true
// historical timeline contradictions (ACT + amendment proposal -> L2_VERIFIED).
func TestPhase5B_EntryGate_WritingConsistencyWithHistoricalSkeleton(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	realNovelPath := "/home/wade/src/github/novel/yuan"
	app.WritingObserver.RegisterProjectPath("default", realNovelPath)

	// Part 1: Deliberate fiction license (Qingnangzi folklore medicine) -> NO_ACTION
	app.WritingObserver.InjectAnomaly("default", writing.ConsistencyAnomaly{
		ID:                  "anom-fic-001",
		AnomalyType:         writing.AnomalyFictionDeviation,
		CharacterName:       "青囊子",
		ChapterRef:          "volume1/ch05",
		Description:         "民間隱士傳授奇門醫道（合理小說虛構設定）",
		KnowledgeTier:       writing.TierFiction,
		IsDeliberateFiction: true,
		Severity:            2,
	})

	ficOutcome, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   "default",
		TriggerType: continuity.TriggerScheduled,
		Summary:     "Writing Observation - Intentional Fiction License Evaluation",
	})
	if err != nil {
		t.Fatalf("fiction cycle failed: %v", err)
	}

	if ficOutcome.Decision.ActionType != decision.ActionNoAction {
		t.Errorf("expected rational NO_ACTION for intentional fiction license, got %s", ficOutcome.Decision.ActionType)
	}

	// Part 2: True historical fact contradiction (Tuotuo in 1360 when he died in 1355) -> ACT + Erratum
	app.WritingObserver.ClearInjectedAnomalies("default")
	app.WritingObserver.InjectAnomaly("default", writing.ConsistencyAnomaly{
		ID:                  "anom-hist-tuotuo-1360",
		AnomalyType:         writing.AnomalyPostMortemAppearance,
		CharacterName:       "脫脫",
		ChapterRef:          "volume3/ch18",
		StatedYear:          1360,
		DeathYear:           1355,
		Description:         "脫脫於至正十五年 (1355) 遭流放賜死，不可在 1360 年繼續擔任朝廷統帥",
		KnowledgeTier:       writing.TierFactHistory,
		IsDeliberateFiction: false,
		Severity:            9,
	})

	histOutcome, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   "default",
		TriggerType: continuity.TriggerScheduled,
		Summary:     "Writing Observation - Historical Contradiction Resolution",
	})
	if err != nil {
		t.Fatalf("historical contradiction cycle failed: %v", err)
	}

	if histOutcome.Decision.ActionType != decision.ActionAct {
		t.Fatalf("expected ActionAct on historical fact violation, got %s", histOutcome.Decision.ActionType)
	}
	if histOutcome.TaskID == nil {
		t.Fatalf("expected autonomous task spawned for manuscript errata")
	}

	// Part 3: Verify amendment proposal reaches L2_VERIFIED
	learnEng := app.AutonomyEngine.LearningEngine()
	projID := "default"
	oe, rec, err := learnEng.RecordDiscrepancy(ctx, learning.DiscrepancyInput{
		CycleID:        &histOutcome.CycleID,
		ProjectID:      &projID,
		Domain:         "writing",
		Title:          "Post-mortem appearance contradiction for Tuotuo in 1360",
		MetricName:     "historical_fact_accuracy",
		ExpectedMetric: 1.0,
		ActualMetric:   0.0,
		LowerIsBetter:  false,
		ActualContext: map[string]any{"post_mortem_count": 1},
	})
	if err != nil {
		t.Fatalf("record discrepancy failed: %v", err)
	}

	evi := seedEvidence(t, ctx, app.KnowledgeRepo, "evi-writing-hist-tuotuo", "characters.md#tuotuo",
		"Tuotuo dismissed in 1354 and died in 1355/1356 per Yuanshi Biography 138")

	rca, err := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe.ID,
		LearningRecordID:      rec.ID,
		Category:              domainLearning.RCIncorrectFact,
		Narrative:             "Commander in 1360 was historically Chaghan Temür (察罕帖木兒), not Tuotuo",
		FalsifiableHypothesis: "Replacing Tuotuo with Chaghan Temür in Volume 3 Chapter 18 eliminates chronology contradiction against Yuanshi",
		CausalEvidenceRefs:    []string{evi.ID},
	})
	if err != nil {
		t.Fatalf("rca failed: %v", err)
	}

	_, _, err = learnEng.ProposeAndApplyCorrection(ctx, learning.CorrectionInput{
		RCAID:            rca.ID,
		LearningRecordID: rec.ID,
		Scope:            domainLearning.ScopeA,
		CorrectionType:   domainLearning.CorrectionPrompt,
		TargetEntityRef:  "outline/volume3/chapter18",
		BeforeState:      map[string]any{"commander": "脫脫"},
		ProposedState:    map[string]any{"commander": "察罕帖木兒"},
	})
	if err != nil {
		t.Fatalf("propose correction failed: %v", err)
	}

	verified, err := learnEng.VerifyCorrection(ctx, learning.VerificationInput{
		LearningRecordID:      rec.ID,
		ComparisonMethod:      domainLearning.CompPiTReplay,
		BaselineMetric:        0.0,
		PostMetric:            1.0,
		LowerIsBetter:         false,
		HasControl:            true,
		ControlBaselineMetric: 0.0,
		ControlPostMetric:     0.0,
		MinExcessImprovement:  0.5,
		SampleSize:            10,
		MinSamples:            5,
		RegressionPassed:      true,
		ProposedConfidence:    0.92,
	})
	if err != nil {
		t.Fatalf("verify correction failed: %v", err)
	}

	if verified.LearningLevel != domainLearning.LevelL2Verified {
		t.Errorf("expected L2_VERIFIED, got %s", verified.LearningLevel)
	}
}

// TestPhase5B_EntryGate_CrossDomainDelegation_Writing_x_Coding verifies Criterion 4 (AIA-12 §21, AIA-15 §22):
// Cross-domain collaboration where Writing Agent delegates creation of an automated chronology scanner
// to Coding Agent, which builds and delivers the tool, followed by acceptance verification.
func TestPhase5B_EntryGate_CrossDomainDelegation_Writing_x_Coding(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	sourceProj := "default"
	goalID := "goal-novel-540-chapters"

	// Seed goal so foreign key reference is valid
	_, err := app.GoalSvc.Create(ctx, goalService.CreateGoalInput{
		ID:          goalID,
		ProjectID:   sourceProj,
		Name:        "Complete and Verify 540-Chapter Historical Novel",
		Description: "Chronological consistency tracking across Yuan dynasty characters",
		Level:       domainGoal.LevelStrategic,
		Status:      domainGoal.StatusActive,
	})
	if err != nil {
		t.Fatalf("seed goal failed: %v", err)
	}

	// 1. Writing Agent submits delegation request to Coding Agent
	taskSpec := map[string]any{
		"title":       "Build Character Timeline Consistency Scanner",
		"description": "Develop a CLI parser that checks characters.md birth/death bounds across chapters",
		"target_path": "tools/novel_scanner",
	}
	gate := map[string]any{
		"verification_type": "CLI_EXECUTION_AND_UNIT_TEST",
		"must_pass_test":    "TestManuscriptConsistencyObserver_ArtisticLicenseVsHistoricalContradiction",
		"zero_false_alerts": true,
	}

	delReq, err := app.DelegationSvc.RequestDelegation(ctx, delegation.CreateDelegationInput{
		RequesterAgentID:   "writing-agent",
		TargetAgentID:      "coding-agent",
		SourceProjectID:    &sourceProj,
		GoalID:             &goalID,
		TaskSpec:           taskSpec,
		AcceptanceCriteria: gate,
	})
	if err != nil {
		t.Fatalf("request delegation failed: %v", err)
	}
	if delReq.Status != "PENDING" {
		t.Errorf("expected status PENDING, got %s", delReq.Status)
	}
	if delReq.SpawnedTaskID == nil {
		t.Errorf("expected delegation to spawn a task in task repository")
	}

	// 2. Coding Agent accepts contract
	if err := app.DelegationSvc.AcceptDelegation(ctx, delReq.ID); err != nil {
		t.Fatalf("accept delegation failed: %v", err)
	}
	accepted, err := app.DelegationSvc.Get(ctx, delReq.ID)
	if err != nil || accepted.Status != "IN_PROGRESS" {
		t.Fatalf("expected IN_PROGRESS, got %s", accepted.Status)
	}

	// 3. Coding Agent delivers work
	deliverable := map[string]any{
		"artifact_type": "TOOL_BINARY",
		"command":       "aia novel verify-timeline --path=/home/wade/src/github/novel/yuan",
		"tests_passed":  true,
		"exit_code":     0,
	}
	if err := app.DelegationSvc.DeliverWork(ctx, delReq.ID, deliverable); err != nil {
		t.Fatalf("deliver work failed: %v", err)
	}
	delivered, err := app.DelegationSvc.Get(ctx, delReq.ID)
	if err != nil || delivered.Status != "DELIVERED" {
		t.Fatalf("expected DELIVERED, got %s", delivered.Status)
	}

	// 4. Writing Agent verifies delivery against criteria (Passed)
	if err := app.DelegationSvc.VerifyDelivery(ctx, delReq.ID, true, "Scanner correctly validated 6 volumes and 540 chapters without false positives on fictional characters"); err != nil {
		t.Fatalf("verify delivery failed: %v", err)
	}

	finalDel, err := app.DelegationSvc.Get(ctx, delReq.ID)
	if err != nil {
		t.Fatalf("get final delegation failed: %v", err)
	}
	if finalDel.Status != "VERIFIED" {
		t.Errorf("expected final status VERIFIED, got %s", finalDel.Status)
	}

	// 5. Verify deliverable payload was stored
	var delMap map[string]any
	if err := json.Unmarshal([]byte(finalDel.DeliverableJSON), &delMap); err != nil {
		t.Fatalf("invalid deliverable JSON: %v", err)
	}
	if delMap["artifact_type"] != "TOOL_BINARY" {
		t.Errorf("expected artifact_type TOOL_BINARY, got %v", delMap["artifact_type"])
	}

	// 6. Verify audit trail was recorded
	audits, err := app.AuditRepo.List(ctx, 10)
	if err != nil {
		t.Fatalf("list audit logs failed: %v", err)
	}
	foundVerifyAudit := false
	for _, a := range audits {
		if a.Action == "DELEGATION_VERIFIED" && a.ResourceID == delReq.ID {
			foundVerifyAudit = true
			break
		}
	}
	if !foundVerifyAudit {
		t.Errorf("expected audit entry for DELEGATION_VERIFIED")
	}
}
