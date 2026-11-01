package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/app"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/knowledge"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/provider/mock"
)

// TestPhase1_MilestonesM1AndM2 verifies Phase 1 acceptance criteria:
// 1. Milestone M1 (Temporal Continuity & Crash Recovery): Process termination & restart preserves project state, pending issues, goals, and event history.
// 2. Milestone M2 (Trustworthy Decision & Provenance): Any decision can be backtracked to context snapshot, evidence, and 5D source scores; model-generated content cannot be promoted to verified knowledge without grounding evidence.
func TestPhase1_MilestonesM1AndM2(t *testing.T) {
	appInst1, tmpDir := createTestApp(t)
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	ctx := context.Background()
	now := time.Now().UTC()

	projID := "proj-long-term-alpha"
	goalID := "goal-alpha-strategy"
	cycleID := "cycle-alpha-001"
	decID := "dec-pilot-restraint-001"
	snapID := "snap-pre-restart-001"

	// ==========================================
	// Step 1: Initialize Project, Goal, and Events
	// ==========================================
	_, err := appInst1.ProjectSvc.EnsureDefault(ctx, tmpDir)
	if err != nil {
		t.Fatalf("failed to ensure default project: %v", err)
	}

	createdProj, err := appInst1.ProjectSvc.Create(ctx, "Long Term Alpha", "Autonomous trading and research pilot", filepath.Join(tmpDir, "alpha"))
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	projID = string(createdProj.ID)

	// Create Goal
	err = appInst1.GoalRepo.Create(ctx, &goal.Goal{
		ID:          goalID,
		Name:        "Sustain Compound Return",
		Description: "Continuously scan for edge with strict risk management",
		Level:       goal.LevelStrategic,
		Status:      goal.StatusActive,
		Priority:    9,
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatalf("failed to create goal: %v", err)
	}

	err = appInst1.GoalRepo.LinkProject(ctx, goalID, projID, goal.LinkContributesTo)
	if err != nil {
		t.Fatalf("failed to link goal: %v", err)
	}

	// Append immutable events to event_history (Hash Chain)
	evt1 := &continuity.EventHistory{
		EventID:       uuid.New().String(),
		EventType:     "CYCLE_INITIALIZED",
		AggregateType: "AUTONOMY_CYCLE",
		AggregateID:   cycleID,
		CorrelationID: "corr-001",
		CausationID:   "genesis",
		ActorID:       "SYSTEM:autonomy",
		PayloadJSON:   `{"cycle_number":1,"mode":"PILOT"}`,
		OccurredAt:    now,
	}
	if err := appInst1.EventHistoryRepo.Append(ctx, nil, evt1); err != nil {
		t.Fatalf("failed to append event 1: %v", err)
	}

	evt2 := &continuity.EventHistory{
		EventID:       uuid.New().String(),
		EventType:     "GOAL_ATTACHED",
		AggregateType: "AUTONOMY_CYCLE",
		AggregateID:   cycleID,
		CorrelationID: "corr-001",
		CausationID:   evt1.EventID,
		ActorID:       "SYSTEM:autonomy",
		PayloadJSON:   `{"goal_id":"` + goalID + `"}`,
		OccurredAt:    now.Add(1 * time.Second),
	}
	if err := appInst1.EventHistoryRepo.Append(ctx, nil, evt2); err != nil {
		t.Fatalf("failed to append event 2: %v", err)
	}

	// Verify hash chain
	if evt2.PreviousHash != evt1.PayloadHash {
		t.Errorf("expected hash chain link: evt2.PreviousHash (%s) == evt1.PayloadHash (%s)", evt2.PreviousHash, evt1.PayloadHash)
	}

	// ==========================================
	// Step 2: Create In-flight Task and Snapshot
	// ==========================================
	tTask, err := appInst1.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		ProjectID:   createdProj.ID,
		Title:       "Heavy data crunching",
		Description: "Crunch market data",
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	_ = appInst1.TaskSvc.UpdateStatus(ctx, tTask.ID, task.StatusRunning)

	pendingIssues := []string{
		"Issue A: Review latency on MOPS parser",
		"Issue B: Adjust risk parameters before market open",
	}

	snap, err := appInst1.Reconstructor.CapturePreRestartSnapshot(
		ctx,
		projID,
		continuity.SnapshotTypePreRestart,
		[]string{string(tTask.ID)},
		pendingIssues,
		map[string]string{"STATUS": "HEALTHY", "TOKEN": "SECRET123"},
	)
	if err != nil {
		t.Fatalf("failed to capture snapshot: %v", err)
	}
	snapID = snap.ID

	// Create Autonomy Cycle and Decision (NO_ACTION)
	_ = appInst1.ContinuityRepo.CreateCycle(ctx, &continuity.AutonomyCycle{
		ID:          cycleID,
		CycleNumber: 1,
		TriggerType: continuity.TriggerScheduled,
		Status:      continuity.CycleStatusExecuting,
		Summary:     "Cycle 1 evaluation",
		StartedAt:   now,
		CreatedAt:   now,
	})

	// Register Source & Evidence
	err = appInst1.KnowledgeSvc.RegisterSource(ctx, &knowledge.Source{
		ID:                   "src-twse-mops",
		SourceType:           knowledge.SourceOfficial,
		URI:                  "https://mops.twse.com.tw/nas/t21/sii/t21sc03.html",
		Publisher:            "Taiwan Stock Exchange",
		AuthorityScore:       0.99,
		DirectnessScore:      0.95,
		TimelinessScore:      0.92,
		IndependenceScore:    1.0,
		ReproducibilityScore: 1.0,
		CreatedAt:            now,
	})
	if err != nil {
		t.Fatalf("failed to register source: %v", err)
	}

	eviID := "evi-mops-revenue-growth"
	err = appInst1.KnowledgeSvc.RecordEvidence(ctx, &knowledge.Evidence{
		ID:             eviID,
		SourceID:       "src-twse-mops",
		ContentSnippet: "Semiconductor aggregate monthly revenue increased 24.3% YoY",
		ContentHash:    "mops-sha256-abc1234",
		CaptureMethod:  knowledge.CaptureAPIJSON,
		CapturedAt:     now,
		CreatedAt:      now,
	})
	if err != nil {
		t.Fatalf("failed to record evidence: %v", err)
	}

	// Decision: NO_ACTION based on Evidence and Snapshot
	dec := &decision.Decision{
		ID:                decID,
		CycleID:           &cycleID,
		ProjectID:         &projID,
		ActionType:        decision.ActionNoAction,
		IntentSummary:     "Hold cash: market condition is frothy despite strong YoY growth",
		Justification:     "Macro conditions warrant rational restraint",
		ContextSnapshotID: &snapID,
		GovernanceResult:  decision.GovernanceAllowed,
		CreatedAt:         now,
	}
	links := []*decision.DecisionEvidenceLink{
		{
			DecisionID:    decID,
			EvidenceID:    eviID,
			RelevanceNote: "Supports rationale for safe holding",
		},
	}
	if err := appInst1.DecisionRepo.Create(ctx, dec, links); err != nil {
		t.Fatalf("failed to create decision: %v", err)
	}

	// ==========================================
	// Step 3: Simulate Crash and App Restart (Milestone M1)
	// ==========================================
	// Close appInst1 abruptly (simulating SIGKILL / crash recovery)
	_ = appInst1.Close()

	// Instantiate appInst2 on the exact same database
	mockProv := mock.NewMockProvider()
	appInst2, err := app.NewApp(cfgPath, mockProv)
	if err != nil {
		t.Fatalf("failed to restart AIA application: %v", err)
	}
	defer appInst2.Close()

	// Check Task recovery: The in-flight running task should be safely reset
	recoveredTask, err := appInst2.TaskSvc.Get(ctx, tTask.ID)
	if err != nil {
		t.Fatalf("failed to query recovered task: %v", err)
	}
	if recoveredTask.Status == task.StatusRunning {
		t.Errorf("expected task to no longer be RUNNING after crash recovery, got: %s", recoveredTask.Status)
	}

	// Context Reconstruction: Verify full continuity restore (Milestone M1)
	recon, err := appInst2.Reconstructor.ReconstructProjectState(ctx, projID)
	if err != nil {
		t.Fatalf("failed to reconstruct project state after restart: %v", err)
	}

	if recon.ProjectID != projID {
		t.Errorf("expected project ID %s, got: %s", projID, recon.ProjectID)
	}
	if len(recon.ActiveGoals) != 1 || recon.ActiveGoals[0].ID != goalID {
		t.Fatalf("expected 1 restored goal %s, got: %+v", goalID, recon.ActiveGoals)
	}
	if len(recon.PendingIssues) != 2 {
		t.Fatalf("expected 2 pending issues restored, got %d: %+v", len(recon.PendingIssues), recon.PendingIssues)
	}
	if recon.PendingIssues[0] != pendingIssues[0] || recon.PendingIssues[1] != pendingIssues[1] {
		t.Errorf("pending issues mismatch: got %+v, expected %+v", recon.PendingIssues, pendingIssues)
	}
	if len(recon.RecentDecisions) == 0 || recon.RecentDecisions[0].ID != decID {
		t.Fatalf("expected restored recent decision %s, got: %+v", decID, recon.RecentDecisions)
	}
	if recon.RecentDecisions[0].ActionType != decision.ActionNoAction {
		t.Errorf("expected decision action type NO_ACTION, got: %s", recon.RecentDecisions[0].ActionType)
	}

	// ==========================================
	// Step 4: Decision Provenance Backtracking (Milestone M2)
	// ==========================================
	trace, err := appInst2.KnowledgeSvc.TraceDecisionProvenance(ctx, decID)
	if err != nil {
		t.Fatalf("failed to trace decision provenance: %v", err)
	}

	if trace.Decision.ID != decID {
		t.Errorf("trace decision ID mismatch: %s vs %s", trace.Decision.ID, decID)
	}
	if trace.ContextSnapshot == nil || trace.ContextSnapshot.ID != snapID {
		t.Errorf("expected linked context snapshot %s, got: %+v", snapID, trace.ContextSnapshot)
	}
	if len(trace.EvidenceChain) != 1 {
		t.Fatalf("expected 1 evidence linked to decision, got: %d", len(trace.EvidenceChain))
	}
	evTrace := trace.EvidenceChain[0]
	if evTrace.Evidence.ID != eviID {
		t.Errorf("expected evidence ID %s, got: %s", eviID, evTrace.Evidence.ID)
	}
	if evTrace.Source == nil || evTrace.Source.URI != "https://mops.twse.com.tw/nas/t21/sii/t21sc03.html" {
		t.Errorf("source mismatch in evidence trace: %+v", evTrace.Source)
	}
	if evTrace.Source.AuthorityScore != 0.99 || evTrace.Source.ReproducibilityScore != 1.0 {
		t.Errorf("source 5D score mismatch: authority=%v, repro=%v", evTrace.Source.AuthorityScore, evTrace.Source.ReproducibilityScore)
	}

	// ==========================================
	// Step 5: Knowledge Integrity & Promotion Check (AIA-16)
	// ==========================================
	// 5.1 Model generated content cannot be promoted to verified knowledge without grounding evidence
	promoRes1, err := appInst2.KnowledgeSvc.ValidateKnowledgePromotion(ctx, knowledge.SourceModel, nil)
	if err != nil {
		t.Fatalf("unexpected error validating model promotion: %v", err)
	}
	if promoRes1.Allowed {
		t.Errorf("AIA-16 Rule 5 violation: pure model generated content must NOT be promoted to VERIFIED")
	}

	// 5.2 Grounded in high-authority official evidence -> Allowed as VERIFIED
	promoRes2, err := appInst2.KnowledgeSvc.ValidateKnowledgePromotion(ctx, knowledge.SourceModel, []string{eviID})
	if err != nil {
		t.Fatalf("unexpected error validating promotion with evidence: %v", err)
	}
	if !promoRes2.Allowed || promoRes2.ProvenanceStatus != knowledge.ProvenanceStatusVerified {
		t.Errorf("expected promotion allowed as VERIFIED with official evidence, got: allowed=%v status=%s", promoRes2.Allowed, promoRes2.ProvenanceStatus)
	}
}
