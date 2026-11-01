package unit

import (
	"context"
	"testing"
	"time"

	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestMigration010_InitiativesDeliberationAndGoalTaskLinks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	ver, err := db.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("failed to read schema version: %v", err)
	}
	if ver < 10 {
		t.Fatalf("expected schema version >= 10, got %d", ver)
	}

	// 1. Seed prerequisite Cycle, Goal, Task, and Decision
	cycleID := "cycle-m10-1"
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO autonomy_cycles (id, cycle_number, trigger_type, status, summary, started_at, created_at)
		VALUES ('cycle-m10-1', 101, 'SCHEDULED', 'STARTED', 'Test cycle 10', ?, ?)
	`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("failed to insert cycle: %v", err)
	}

	goalRepo := repository.NewGoalRepository(db)
	g := &goal.Goal{
		ID:        "goal-m10-1",
		Name:      "Improve Test Coverage",
		Level:     goal.LevelTactical,
		Status:    goal.StatusActive,
		Priority:  7,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := goalRepo.Create(ctx, g); err != nil {
		t.Fatalf("failed to create goal: %v", err)
	}

	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO projects (id, name, workspace, description, status, created_at, updated_at)
		VALUES ('proj-m10-1', 'Migration 010 Test', '/tmp/m10', 'desc', 'ACTIVE', ?, ?)
	`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("failed to insert project: %v", err)
	}

	taskRepo := repository.NewTaskRepository(db)
	tk := &task.Task{
		ID:        "task-m10-1",
		ProjectID: "proj-m10-1",
		Title:     "Add unit tests for recall",
		Status:    task.StatusReady,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := taskRepo.Create(ctx, tk, nil); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	autonomyRepo := repository.NewAutonomyRepository(db)

	// 2. Create Candidate Initiative with full Phase 3 fields
	goalID := g.ID
	init := &domainAutonomy.CandidateInitiative{
		ID:                  "init-1",
		CycleID:             cycleID,
		GoalID:              &goalID,
		DiscoveryType:       domainAutonomy.DiscoveryGoalGap,
		ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
		TargetAgentID:       "coding-agent",
		TriggerSource:       "GOAL_GAP_EVALUATOR",
		DetectedOpportunity: "Optimize background database query index",
		Rationale:           "Query latency increased by 15%",
		ExpectedOutcome:     "Reduce p95 latency below 10ms",
		PriorityScore:       8,
		EstimatedCost:       1.5,
		EstimatedRisk:       governance.RiskTierLow,
		Status:              domainAutonomy.InitiativeCandidate,
		MetadataJSON:        `{"gap_ratio":0.25}`,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := autonomyRepo.CreateInitiative(ctx, init); err != nil {
		t.Fatalf("failed to create initiative: %v", err)
	}

	// 3. Verify round-trip read and status transitions (MONITORING -> ADOPTED with SpawnedTaskID)
	if err := autonomyRepo.UpdateInitiativeStatus(ctx, init.ID, domainAutonomy.InitiativeMonitoring, nil); err != nil {
		t.Fatalf("failed to set MONITORING status: %v", err)
	}
	if err := autonomyRepo.MarkInitiativeAdopted(ctx, init.ID, string(tk.ID)); err != nil {
		t.Fatalf("failed to mark initiative adopted: %v", err)
	}

	list, err := autonomyRepo.ListInitiativesByCycle(ctx, cycleID)
	if err != nil {
		t.Fatalf("failed to list initiatives: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 initiative, got %d", len(list))
	}
	if list[0].Status != domainAutonomy.InitiativeAdopted {
		t.Errorf("expected ADOPTED, got %s", list[0].Status)
	}
	if list[0].SpawnedTaskID == nil || *list[0].SpawnedTaskID != string(tk.ID) {
		t.Errorf("expected spawned_task_id %s, got %v", tk.ID, list[0].SpawnedTaskID)
	}
	if list[0].DiscoveryType != domainAutonomy.DiscoveryGoalGap {
		t.Errorf("expected discovery_type GOAL_GAP, got %s", list[0].DiscoveryType)
	}

	// 4. Deliberation Record + Decision Linkage
	delib := &domainAutonomy.DeliberationRecord{
		ID:                    "delib-1",
		CycleID:               cycleID,
		CandidateInitiativeID: &init.ID,
		OptionsEvaluatedJSON:  `[{"option":"ACT","selected":true}]`,
		CostBenefitAnalysis:   "High impact, low effort",
		RiskAssessment:        "LOW",
		RecommendedAction:     decision.ActionAct,
		CreatedAt:             now,
	}
	if err := autonomyRepo.CreateDeliberation(ctx, delib); err != nil {
		t.Fatalf("failed to create deliberation: %v", err)
	}

	decRepo := repository.NewDecisionRepository(db)
	tIDStr := string(tk.ID)
	dec := &decision.Decision{
		ID:               "dec-m10-1",
		CycleID:          &cycleID,
		ActionType:       decision.ActionAct,
		IntentSummary:    "Execute goal gap task",
		Justification:    "Positive expected value",
		GovernanceResult: decision.GovernanceAllowed,
		ActionTaskID:     &tIDStr,
		CreatedAt:        now,
	}
	if err := decRepo.Create(ctx, dec, nil); err != nil {
		t.Fatalf("failed to create decision: %v", err)
	}
	if err := autonomyRepo.LinkDeliberationDecision(ctx, delib.ID, dec.ID); err != nil {
		t.Fatalf("failed to link deliberation decision: %v", err)
	}

	gotDelib, err := autonomyRepo.GetDeliberationByDecision(ctx, dec.ID)
	if err != nil {
		t.Fatalf("failed to get deliberation by decision: %v", err)
	}
	if gotDelib.ID != delib.ID {
		t.Errorf("expected deliberation %s, got %s", delib.ID, gotDelib.ID)
	}

	// 5. Goal-to-Task Decomposition Link (goal_task_links)
	if err := autonomyRepo.LinkGoalTask(ctx, &domainAutonomy.GoalTaskLink{
		GoalID:                 g.ID,
		TaskID:                 string(tk.ID),
		CycleID:                &cycleID,
		DecompositionRationale: "Close test coverage gap",
		CreatedAt:              now,
	}); err != nil {
		t.Fatalf("failed to link goal to task: %v", err)
	}

	links, err := autonomyRepo.ListGoalTaskLinks(ctx, g.ID)
	if err != nil {
		t.Fatalf("failed to list goal_task_links: %v", err)
	}
	if len(links) != 1 || links[0].TaskID != string(tk.ID) {
		t.Fatalf("unexpected goal_task_links: %+v", links)
	}

	// 6. Foreign Key enforcement (invalid cycle_id rejected)
	badDelib := &domainAutonomy.DeliberationRecord{
		ID:                   "delib-bad",
		CycleID:              "non-existent-cycle",
		OptionsEvaluatedJSON: `[]`,
		CostBenefitAnalysis:  "-",
		RiskAssessment:       "-",
		RecommendedAction:    decision.ActionNoAction,
		CreatedAt:            now,
	}
	if err := autonomyRepo.CreateDeliberation(ctx, badDelib); err == nil {
		t.Fatalf("expected foreign key failure for invalid cycle_id, but got success")
	}
}
