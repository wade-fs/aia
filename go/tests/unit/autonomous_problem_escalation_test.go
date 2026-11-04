package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy/cycle"
	"github.com/wade/aia/internal/autonomy/deliberation"
	"github.com/wade/aia/internal/autonomy/initiative"
	"github.com/wade/aia/internal/autonomy/recall"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/decision"
	domainPlan "github.com/wade/aia/internal/domain/plan"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	domainTask "github.com/wade/aia/internal/domain/task"
	problemService "github.com/wade/aia/internal/service/problem"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestAutonomousProblemEscalation_ConsecutiveFailures(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	now := time.Now().UTC()
	projID := "proj-auto-escalate-001"

	// 1. Repositories
	projRepo := repository.NewProjectRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	probRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)
	contRepo := repository.NewContinuityRepository(db)
	autoRepo := repository.NewAutonomyRepository(db)
	decRepo := repository.NewDecisionRepository(db)
	evtRepo := repository.NewEventHistoryRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	govRepo := repository.NewGovernanceRepository(db)
	memRepo := repository.NewMemoryRepository(db)
	agentRepo := repository.NewAgentRepository(db)

	// Seed Agent
	err := agentRepo.Upsert(ctx, &agent.Agent{
		ID:        "coding-agent",
		Name:      "Coding Agent",
		Domain:    "coding",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to seed agent: %v", err)
	}

	// 2. Services
	taskSvc := taskService.NewService(taskRepo, auditRepo)
	probSvc := problemService.NewService(probRepo, planRepo, projRepo, nil)
	probSvc.SetTaskService(taskSvc)
	probSvc.SetAuditRepo(auditRepo)

	// 3. Setup Project
	err = projRepo.Create(ctx, &domainProject.Project{
		ID:        projID,
		Name:      "Repeated Failure Remediation Project",
		Workspace: "/tmp/esc-test",
		Status:    domainProject.StatusActive,
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// 4. Seed 2 consecutive failed tasks in TaskRepo
	failedTask1 := &domainTask.Task{
		ID:          "task-fail-001",
		ProjectID:   domain.ID(projID),
		AgentID:     "coding-agent",
		Title:       "Build Microservice Image",
		Description: "Docker build failed: missing libc dependencies",
		Status:      domainTask.StatusFailed,
		Priority:    7,
		CreatedAt:   now.Add(-10 * time.Minute),
		UpdatedAt:   now.Add(-9 * time.Minute),
	}
	failedTask2 := &domainTask.Task{
		ID:          "task-fail-002",
		ProjectID:   domain.ID(projID),
		AgentID:     "coding-agent",
		Title:       "Deploy Container to Staging",
		Description: "Kubernetes pod crashloopbackoff: image pull failure",
		Status:      domainTask.StatusFailed,
		Priority:    8,
		CreatedAt:   now.Add(-8 * time.Minute),
		UpdatedAt:   now.Add(-7 * time.Minute),
	}
	if err := taskRepo.Create(ctx, failedTask1, nil); err != nil {
		t.Fatalf("failed to seed failed task 1: %v", err)
	}
	if err := taskRepo.Create(ctx, failedTask2, nil); err != nil {
		t.Fatalf("failed to seed failed task 2: %v", err)
	}

	// 5. Build Coordinator with recall engine and discoverer
	recallEng := recall.NewEngine(
		evtRepo,
		nil,
		memRepo,
		contRepo,
		nil,
		decRepo,
		projRepo,
		taskRepo,
	)
	recallEng.SetProblemRepository(probRepo)
	recallEng.SetPlanRepository(planRepo)

	discoverer := initiative.NewDiscoverer(autoRepo)
	deliberator := deliberation.NewDeliberator()

	coord := cycle.NewCoordinator(cycle.CoordinatorDeps{
		RecallEngine:   recallEng,
		Discoverer:     discoverer,
		Deliberator:    deliberator,
		AutonomyRepo:   autoRepo,
		DecisionRepo:   decRepo,
		ContinuityRepo: contRepo,
		ProjectRepo:    projRepo,
		GovernanceRepo: govRepo,
		AuditRepo:      auditRepo,
		TaskSvc:        taskSvc,
		ProblemRepo:    probRepo,
		PlanRepo:       planRepo,
		ProblemSvc:     probSvc,
	})

	// 6. Execute Autonomy Cycle
	outcome, err := coord.RunCycle(ctx, cycle.RunInput{
		ProjectID:   projID,
		TriggerType: "SCHEDULED",
		Summary:     "Cycle checking project health under repeated failures",
	})
	if err != nil {
		t.Fatalf("RunCycle failed: %v", err)
	}

	// 7. Verify Cycle Outcomes
	if outcome == nil {
		t.Fatal("expected non-nil cycle outcome")
	}
	if outcome.Decision == nil {
		t.Fatal("expected non-nil decision")
	}
	if outcome.Decision.ActionType != decision.ActionAct {
		t.Fatalf("expected ActionAct to remediate failures, got %s (justification: %s)",
			outcome.Decision.ActionType, outcome.Decision.Justification)
	}
	if outcome.TaskID == nil || *outcome.TaskID == "" {
		t.Fatal("expected spawned task ID from materialized problem plan, got nil")
	}
	if !outcome.Verification.Passed {
		t.Fatalf("expected cycle verification to pass, got: %s", outcome.Verification.ActualSummary)
	}

	// 8. Verify Problem was automatically created in database
	problems, err := probRepo.ListByProject(ctx, projID)
	if err != nil {
		t.Fatalf("failed to list problems: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem created autonomously, got %d", len(problems))
	}
	prob := problems[0]
	if prob.Status != domainProblem.StatusExecuting {
		t.Errorf("expected problem status EXECUTING, got %s", prob.Status)
	}
	if prob.CurrentModelVersion != 1 {
		t.Errorf("expected problem model version 1, got %d", prob.CurrentModelVersion)
	}
	if prob.CurrentPlanVersion != 1 {
		t.Errorf("expected problem plan version 1, got %d", prob.CurrentPlanVersion)
	}

	// 9. Verify ProblemModel items
	models, err := probRepo.ListModels(ctx, prob.ID)
	if err != nil {
		t.Fatalf("failed to list models: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model version, got %d", len(models))
	}
	mod := models[0]
	if len(mod.KnownFacts) == 0 {
		t.Errorf("expected model to have verified known facts")
	}
	if len(mod.Unknowns) == 0 {
		t.Errorf("expected model to have identified unknowns")
	}
	if len(mod.Assumptions) == 0 {
		t.Errorf("expected model to have assumptions")
	}

	// 10. Verify Plan was generated and approved
	plan, err := probSvc.GetLatestPlan(ctx, prob.ID)
	if err != nil {
		t.Fatalf("failed to get latest plan: %v", err)
	}
	if plan == nil {
		t.Fatal("expected plan to exist")
	}
	if plan.Status != domainPlan.StatusActive {
		t.Errorf("expected plan status ACTIVE, got %s", plan.Status)
	}
	if len(plan.WorkItems) != 2 {
		t.Fatalf("expected 2 work items in plan, got %d", len(plan.WorkItems))
	}
	wi1 := plan.WorkItems[0]
	wi2 := plan.WorkItems[1]
	if wi1.LinkedTaskID == "" {
		t.Errorf("expected work item 1 to be materialized with LinkedTaskID")
	}
	if wi2.LinkedTaskID == "" {
		t.Errorf("expected work item 2 to be materialized with LinkedTaskID")
	}
	if wi1.LinkedTaskID != *outcome.TaskID {
		t.Errorf("expected cycle spawned task %s to match primary work item task %s", *outcome.TaskID, wi1.LinkedTaskID)
	}

	// 11. Verify tasks exist in TaskRepository and have dependency links
	t1, err := taskSvc.Get(ctx, wi1.LinkedTaskID)
	if err != nil {
		t.Fatalf("failed to load materialized task 1: %v", err)
	}
	if t1.Status != domainTask.StatusReady {
		t.Errorf("expected task 1 status READY, got %s", t1.Status)
	}

	t2, err := taskSvc.Get(ctx, wi2.LinkedTaskID)
	if err != nil {
		t.Fatalf("failed to load materialized task 2: %v", err)
	}
	if t2.Status != domainTask.StatusWaitingDependency {
		t.Errorf("expected task 2 to be WAITING_DEPENDENCY on task 1, got %s", t2.Status)
	}
}

func TestAutonomousProblemEscalation_ExistingOpenProblem_AutoPlansAndMaterializes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	now := time.Now().UTC()
	projID := "proj-auto-plan-002"

	projRepo := repository.NewProjectRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	probRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)
	contRepo := repository.NewContinuityRepository(db)
	autoRepo := repository.NewAutonomyRepository(db)
	decRepo := repository.NewDecisionRepository(db)
	evtRepo := repository.NewEventHistoryRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	govRepo := repository.NewGovernanceRepository(db)
	memRepo := repository.NewMemoryRepository(db)
	agentRepo := repository.NewAgentRepository(db)

	_ = agentRepo.Upsert(ctx, &agent.Agent{
		ID:        "coding-agent",
		Name:      "Coding Agent",
		Domain:    "coding",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})

	_ = projRepo.Create(ctx, &domainProject.Project{
		ID:        projID,
		Name:      "Existing Problem Planning Project",
		Workspace: "/tmp/exist-test",
		Status:    domainProject.StatusActive,
		CreatedAt: now,
	})

	// Pre-create an OPEN problem that has no plan yet
	openProb := &domainProblem.Problem{
		ID:        "prob-unplanned-001",
		ProjectID: projID,
		Title:     "Refactor Authentication Architecture",
		Objective: "Migrate legacy tokens to JWT with public-key rotation",
		Status:    domainProblem.StatusOpen,
		Priority:  domainProblem.PriorityHigh,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := probRepo.Create(ctx, openProb); err != nil {
		t.Fatalf("failed to create open problem: %v", err)
	}

	taskSvc := taskService.NewService(taskRepo, auditRepo)
	probSvc := problemService.NewService(probRepo, planRepo, projRepo, nil)
	probSvc.SetTaskService(taskSvc)
	probSvc.SetAuditRepo(auditRepo)

	recallEng := recall.NewEngine(
		evtRepo,
		nil,
		memRepo,
		contRepo,
		nil,
		decRepo,
		projRepo,
		taskRepo,
	)
	recallEng.SetProblemRepository(probRepo)
	recallEng.SetPlanRepository(planRepo)

	discoverer := initiative.NewDiscoverer(autoRepo)
	deliberator := deliberation.NewDeliberator()

	coord := cycle.NewCoordinator(cycle.CoordinatorDeps{
		RecallEngine:   recallEng,
		Discoverer:     discoverer,
		Deliberator:    deliberator,
		AutonomyRepo:   autoRepo,
		DecisionRepo:   decRepo,
		ContinuityRepo: contRepo,
		ProjectRepo:    projRepo,
		GovernanceRepo: govRepo,
		AuditRepo:      auditRepo,
		TaskSvc:        taskSvc,
		ProblemRepo:    probRepo,
		PlanRepo:       planRepo,
		ProblemSvc:     probSvc,
	})

	// Run Cycle
	outcome, err := coord.RunCycle(ctx, cycle.RunInput{
		ProjectID:   projID,
		TriggerType: "SCHEDULED",
		Summary:     "Cycle planning existing open problem",
	})
	if err != nil {
		t.Fatalf("RunCycle failed: %v", err)
	}

	if outcome.Decision.ActionType != decision.ActionAct {
		t.Fatalf("expected ActionAct to plan open problem, got %s", outcome.Decision.ActionType)
	}
	if outcome.TaskID == nil {
		t.Fatal("expected spawned task ID, got nil")
	}

	// Verify problem state is now EXECUTING and planned
	prob, err := probRepo.Get(ctx, openProb.ID)
	if err != nil {
		t.Fatalf("failed to reload problem: %v", err)
	}
	if prob.Status != domainProblem.StatusExecuting {
		t.Errorf("expected problem status EXECUTING, got %s", prob.Status)
	}
	if prob.CurrentPlanVersion != 1 {
		t.Errorf("expected plan version 1, got %d", prob.CurrentPlanVersion)
	}

	// Verify plan and materialized tasks
	plan, err := probSvc.GetLatestPlan(ctx, openProb.ID)
	if err != nil || plan == nil {
		t.Fatalf("failed to get plan: %v", err)
	}
	if plan.Status != domainPlan.StatusActive {
		t.Errorf("expected plan status ACTIVE, got %s", plan.Status)
	}
	if len(plan.WorkItems) < 1 {
		t.Fatalf("expected at least 1 work item, got %d", len(plan.WorkItems))
	}
}
