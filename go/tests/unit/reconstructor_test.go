package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/continuity"
	continuityDomain "github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestReconstructor_ReconstructProjectState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	cRepo := repository.NewContinuityRepository(db)
	gRepo := repository.NewGoalRepository(db)
	dRepo := repository.NewDecisionRepository(db)
	eRepo := repository.NewEventHistoryRepository(db)
	pRepo := repository.NewProjectRepository(db)

	recon := continuity.NewReconstructor(cRepo, gRepo, dRepo, eRepo, pRepo)

	now := time.Now().UTC()
	projID := "proj-recon-001"

	// 1. Create Project
	err := pRepo.Create(ctx, &project.Project{
		ID:        projID,
		Name:      "Autonomous Pilot",
		Workspace: "/tmp/pilot",
		Status:    project.StatusActive,
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// 2. Create Goal and link to project
	goalID := "goal-alpha-001"
	err = gRepo.Create(ctx, &goal.Goal{
		ID:          goalID,
		Name:        "Maintain Alpha",
		Description: "Continuously explore safe market opportunities",
		Level:       goal.LevelStrategic,
		Status:      goal.StatusActive,
		Priority:    8,
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatalf("failed to create goal: %v", err)
	}
	err = gRepo.LinkProject(ctx, goalID, projID, goal.LinkContributesTo)
	if err != nil {
		t.Fatalf("failed to link goal to project: %v", err)
	}

	// 3. Create Autonomy Cycle & Decision
	cycleID := "cycle-100"
	err = cRepo.CreateCycle(ctx, &continuityDomain.AutonomyCycle{
		ID:          cycleID,
		CycleNumber: 100,
		TriggerType: continuityDomain.TriggerScheduled,
		Status:      continuityDomain.CycleStatusCompleted,
		Summary:     "Cycle 100 scan completed",
		StartedAt:   now.Add(-10 * time.Minute),
		CreatedAt:   now.Add(-10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("failed to create cycle: %v", err)
	}

	decID := "dec-noaction-100"
	err = dRepo.Create(ctx, &decision.Decision{
		ID:               decID,
		CycleID:          &cycleID,
		ProjectID:        &projID,
		ActionType:       decision.ActionNoAction,
		IntentSummary:    "Rational restraint: no high-probability setup",
		Justification:    "Noise ratio high",
		GovernanceResult: decision.GovernanceAllowed,
		CreatedAt:        now.Add(-5 * time.Minute),
	}, nil)
	if err != nil {
		t.Fatalf("failed to create decision: %v", err)
	}

	// 4. Capture Pre-restart Snapshot
	activeTasks := []string{"task-background-fetch"}
	pendingIssues := []string{"Issue 1: verify data feed latency", "Issue 2: refresh model checkpoint"}
	safeMeta := continuityDomain.SanitizeWorkspaceEnv(map[string]string{
		"PROJECT_NAME": "Pilot",
		"API_KEY":      "secret-super-key",
	})

	snap, err := recon.CapturePreRestartSnapshot(ctx, projID, continuityDomain.SnapshotTypePreRestart, activeTasks, pendingIssues, safeMeta)
	if err != nil {
		t.Fatalf("failed to capture pre-restart snapshot: %v", err)
	}

	// Verify env sanitization
	if safeMeta["API_KEY"] != "[REDACTED]" {
		t.Errorf("expected API_KEY to be [REDACTED], got: %s", safeMeta["API_KEY"])
	}
	if safeMeta["PROJECT_NAME"] != "Pilot" {
		t.Errorf("expected PROJECT_NAME to be Pilot, got: %s", safeMeta["PROJECT_NAME"])
	}

	// 5. Reconstruct State (Milestone M1 verification)
	state, err := recon.ReconstructProjectState(ctx, projID)
	if err != nil {
		t.Fatalf("failed to reconstruct project state: %v", err)
	}

	if state.ProjectID != projID {
		t.Errorf("expected project ID %s, got: %s", projID, state.ProjectID)
	}
	if state.LatestSnapshot == nil || state.LatestSnapshot.ID != snap.ID {
		t.Errorf("expected latest snapshot %s, got: %+v", snap.ID, state.LatestSnapshot)
	}
	if len(state.PendingIssues) != 2 {
		t.Errorf("expected 2 pending issues, got: %d", len(state.PendingIssues))
	}
	if len(state.ActiveGoals) != 1 || state.ActiveGoals[0].ID != goalID {
		t.Errorf("expected 1 active goal %s, got: %+v", goalID, state.ActiveGoals)
	}
	if len(state.RecentDecisions) != 1 || state.RecentDecisions[0].ActionType != decision.ActionNoAction {
		t.Errorf("expected 1 recent decision with NO_ACTION, got: %+v", state.RecentDecisions)
	}
	if state.LastCycle == nil || state.LastCycle.ID != cycleID {
		t.Errorf("expected last cycle %s, got: %+v", cycleID, state.LastCycle)
	}
}
