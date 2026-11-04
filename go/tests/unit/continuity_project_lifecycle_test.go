package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/continuity"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	domainContinuity "github.com/wade/aia/internal/domain/continuity"
	domainExp "github.com/wade/aia/internal/domain/experiment"
	"github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/recovery"
	projectService "github.com/wade/aia/internal/service/project"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestProject_LifecycleStates_And_StructuredPendingIssues(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	projSvc := projectService.NewService(projRepo)

	// 1. Create project and verify extended states
	p, err := projSvc.Register(ctx, "AIA-10 Project", "Testing state transitions", "/tmp/aia-10-test", project.DomainCoding)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Test transitions
	if err := projSvc.Block(ctx, domain.ID(p.ID), "Blocked due to dependency missing"); err != nil {
		t.Fatalf("Block failed: %v", err)
	}
	p1, _ := projSvc.Get(ctx, domain.ID(p.ID))
	if p1.Status != project.StatusBlocked {
		t.Errorf("expected StatusBlocked, got %s", p1.Status)
	}

	if err := projSvc.MarkReview(ctx, domain.ID(p.ID), "Code review pending"); err != nil {
		t.Fatalf("MarkReview failed: %v", err)
	}
	p2, _ := projSvc.Get(ctx, domain.ID(p.ID))
	if p2.Status != project.StatusReview {
		t.Errorf("expected StatusReview, got %s", p2.Status)
	}

	if err := projSvc.Complete(ctx, domain.ID(p.ID), "Milestone reached"); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	p3, _ := projSvc.Get(ctx, domain.ID(p.ID))
	if p3.Status != project.StatusCompleted {
		t.Errorf("expected StatusCompleted, got %s", p3.Status)
	}

	// 2. Test structured pending issues
	now := time.Now().UTC()
	issues := []project.PendingIssue{
		{
			ID:            "issue-101",
			Title:         "Memory leak under sustained throughput",
			Severity:      project.IssueSeverityHigh,
			AssignedAgent: "coding-agent",
			Status:        project.IssueStatusInProgress,
			CreatedAt:     now,
		},
		{
			ID:            "issue-102",
			Title:         "Missing audit documentation",
			Severity:      project.IssueSeverityMedium,
			Status:        project.IssueStatusOpen,
			CreatedAt:     now,
		},
	}
	p3.SetPendingIssues(issues)
	if err := projRepo.UpdateContinuity(ctx, domain.ID(p3.ID), nil, nil, p3.PendingIssuesJSON); err != nil {
		t.Fatalf("UpdateContinuity failed: %v", err)
	}

	readBack, err := projRepo.Get(ctx, domain.ID(p3.ID))
	if err != nil {
		t.Fatalf("Get project failed: %v", err)
	}
	parsedIssues := readBack.GetPendingIssues()
	if len(parsedIssues) != 2 {
		t.Fatalf("expected 2 parsed issues, got %d", len(parsedIssues))
	}
	if parsedIssues[0].Severity != project.IssueSeverityHigh || parsedIssues[0].AssignedAgent != "coding-agent" {
		t.Errorf("expected issue 1 severity HIGH and coding-agent, got %+v", parsedIssues[0])
	}
}

func TestContinuity_Reconstruction_CompletenessEvaluation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	contRepo := repository.NewContinuityRepository(db)
	goalRepo := repository.NewGoalRepository(db)
	decRepo := repository.NewDecisionRepository(db)
	eventRepo := repository.NewEventHistoryRepository(db)

	reconstructor := continuity.NewReconstructor(contRepo, goalRepo, decRepo, eventRepo, projRepo)

	// Create project without snapshot, goals, or decisions
	proj := &project.Project{
		ID:        "proj-incomplete",
		Name:      "Incomplete Project",
		Workspace: "/tmp/incomplete",
		Status:    project.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	// Reconstruct state -> should detect missing snapshot, goals, decisions
	recon, err := reconstructor.ReconstructProjectState(ctx, "proj-incomplete")
	if err != nil {
		t.Fatalf("ReconstructProjectState failed: %v", err)
	}
	if recon.Completeness.IsComplete {
		t.Errorf("expected context to be marked incomplete")
	}
	if recon.Completeness.Score >= 0.65 {
		t.Errorf("expected score < 0.65 for totally fresh project, got %.2f", recon.Completeness.Score)
	}
	if !recon.Completeness.PauseHighRiskActions {
		t.Errorf("expected PauseHighRiskActions to be true when score < 0.65")
	}
	if len(recon.Completeness.MissingElements) < 3 {
		t.Errorf("expected at least 3 missing elements, got %v", recon.Completeness.MissingElements)
	}
}

func TestRecovery_StuckCycles_And_Delegations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	taskRepo := repository.NewTaskRepository(db)
	execRepo := repository.NewExecutionRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	contRepo := repository.NewContinuityRepository(db)
	expRepo := repository.NewExperimentRepository(db)

	recMgr := recovery.NewManager(taskRepo, execRepo, auditRepo)
	recMgr.SetContinuityRepository(contRepo)
	recMgr.SetExperimentRepository(expRepo)

	projRepo := repository.NewProjectRepository(db)
	_ = projRepo.Create(ctx, &project.Project{
		ID:        "default",
		Name:      "Default",
		Workspace: "/tmp/default",
		Status:    project.StatusActive,
	})

	// 1. Seed stuck autonomy cycle in STARTED state
	stuckCycle := &domainContinuity.AutonomyCycle{
		ID:          "cycle-stuck-001",
		CycleNumber: 1,
		TriggerType: domainContinuity.TriggerScheduled,
		Status:      domainContinuity.CycleStatusStarted,
		StartedAt:   time.Now().Add(-1 * time.Hour),
	}
	if err := contRepo.CreateCycle(ctx, stuckCycle); err != nil {
		t.Fatalf("CreateCycle failed: %v", err)
	}

	agentRepo := repository.NewAgentRepository(db)
	_ = agentRepo.Upsert(ctx, &agent.Agent{ID: "coding-agent", Name: "Coding Agent", Domain: "coding", Enabled: true})
	_ = agentRepo.Upsert(ctx, &agent.Agent{ID: "writing-agent", Name: "Writing Agent", Domain: "writing", Enabled: true})

	// 2. Seed stuck delegation whose spawned task failed
	failedTaskID := domain.ID("task-failed-del-01")
	failedTask := &task.Task{
		ID:         failedTaskID,
		ProjectID:  "default",
		AgentID:    "coding-agent",
		Title:      "Failed delegation subtask",
		Status:     task.StatusFailed,
		MaxRetries: 1,
		RetryCount: 1,
	}
	if err := taskRepo.Create(ctx, failedTask, nil); err != nil {
		t.Fatalf("taskRepo.Create failed: %v", err)
	}

	stuckDelID := "del-stuck-001"
	taskIDStr := string(failedTaskID)
	if err := expRepo.CreateDelegationRequest(ctx, &domainExp.DelegationRequest{
		ID:               stuckDelID,
		RequesterAgentID: "writing-agent",
		TargetAgentID:    "coding-agent",
		SpawnedTaskID:    &taskIDStr,
		Status:           "IN_PROGRESS",
		CreatedAt:        time.Now().Add(-30 * time.Minute),
	}); err != nil {
		t.Fatalf("expRepo.CreateDelegationRequest failed: %v", err)
	}

	// Run Recovery
	summary, err := recMgr.Recover(ctx)
	if err != nil {
		t.Fatalf("Recover failed: %v", err)
	}

	if summary.AbortedStuckCycles != 1 {
		t.Errorf("expected 1 aborted stuck cycle, got %d", summary.AbortedStuckCycles)
	}
	if summary.ReconciledDelegations != 1 {
		t.Errorf("expected 1 reconciled stuck delegation, got %d", summary.ReconciledDelegations)
	}

	// Verify cycle status updated to ABORTED
	cyc, _ := contRepo.GetCycle(ctx, "cycle-stuck-001")
	if cyc.Status != domainContinuity.CycleStatusAborted {
		t.Errorf("expected cycle status ABORTED, got %s", cyc.Status)
	}

	// Verify delegation status updated to FAILED
	del, _ := expRepo.GetDelegationRequest(ctx, stuckDelID)
	if del.Status != "FAILED" {
		t.Errorf("expected delegation status FAILED, got %s", del.Status)
	}
}
