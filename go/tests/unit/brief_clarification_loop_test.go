package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/autonomy/cycle"
	"github.com/wade/aia/internal/autonomy/deliberation"
	"github.com/wade/aia/internal/autonomy/initiative"
	"github.com/wade/aia/internal/autonomy/observation"
	"github.com/wade/aia/internal/autonomy/recall"
	domainAgent "github.com/wade/aia/internal/domain/agent"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	domainClarify "github.com/wade/aia/internal/domain/clarification"
	domainDecision "github.com/wade/aia/internal/domain/decision"
	domainProject "github.com/wade/aia/internal/domain/project"
	domainTask "github.com/wade/aia/internal/domain/task"
	briefService "github.com/wade/aia/internal/service/brief"
	clarifyService "github.com/wade/aia/internal/service/clarification"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestAutonomyLoop_BriefGap_GeneratesBlockingClarification_And_Unblocks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	projID := "proj-clarify-loop-01"

	// 1. Repositories
	projRepo := repository.NewProjectRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	genericBriefRepo := repository.NewGenericBriefRepository(db)
	clarifyRepo := repository.NewClarificationRepository(db)
	autoRepo := repository.NewAutonomyRepository(db)
	decRepo := repository.NewDecisionRepository(db)
	contRepo := repository.NewContinuityRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	memRepo := repository.NewMemoryRepository(db)
	evtRepo := repository.NewEventHistoryRepository(db)
	govRepo := repository.NewGovernanceRepository(db)
	agentRepo := repository.NewAgentRepository(db)

	_ = agentRepo.Upsert(ctx, &domainAgent.Agent{
		ID:        "writing-agent",
		Name:      "Writing Agent",
		Domain:    "writing",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})

	// 2. Services
	taskSvc := taskService.NewService(taskRepo, auditRepo)
	briefSvc := briefService.NewService(genericBriefRepo, projRepo)
	clarificationSvc := clarifyService.NewService(clarifyRepo, briefSvc, projRepo, taskRepo, memRepo)

	// 3. Create Project & Running Task
	err := projRepo.Create(ctx, &domainProject.Project{
		ID:         projID,
		Name:       "Autonomous Brief Project",
		DomainType: "writing",
		Workspace:  "/tmp/proj-clarify",
		CreatedAt:  now,
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	createdTask, err := taskSvc.Create(ctx, taskService.CreateTaskRequest{
		ProjectID:   projID,
		AgentID:     "writing-agent",
		Title:       "Draft Chapter 1",
		Description: "Drafting the opening chapter",
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	_ = taskSvc.UpdateStatus(ctx, createdTask.ID, domainTask.StatusRunning)

	// 4. Create Project Brief missing P0 fields (era and core_protagonists in historical_fiction)
	brief := &domainBrief.ProjectBrief{
		ID:           "brief-" + projID,
		ProjectID:    projID,
		Domain:       "historical_fiction",
		BriefVersion: 1,
		Readiness:    domainBrief.ReadinessNotReady,
		Fields:       map[string]*domainBrief.FieldState{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = genericBriefRepo.Save(ctx, brief)

	// 5. Setup Autonomy Core Observer & Coordinator
	obsReg := observation.NewRegistry()
	briefObs := autonomy.NewBriefCompletenessObserver(genericBriefRepo, projRepo)
	obsReg.Register("brief_completeness", briefObs)

	recallEng := recall.NewEngine(evtRepo, nil, memRepo, contRepo, nil, decRepo, projRepo, taskRepo)
	discoverer := initiative.NewDiscoverer(autoRepo)
	deliberator := deliberation.NewDeliberator()

	coord := cycle.NewCoordinator(cycle.CoordinatorDeps{
		ObserverRegistry: obsReg,
		RecallEngine:     recallEng,
		Discoverer:       discoverer,
		Deliberator:      deliberator,
		AutonomyRepo:     autoRepo,
		DecisionRepo:     decRepo,
		ContinuityRepo:   contRepo,
		GovernanceRepo:   govRepo,
		AuditRepo:        auditRepo,
		ProjectRepo:      projRepo,
		TaskSvc:          taskSvc,
		ClarificationSvc: clarificationSvc,
	})

	// 6. Run Autonomy Cycle: Expects ActionAskHuman and structured clarification request
	outcome, err := coord.RunCycle(ctx, cycle.RunInput{
		ProjectID:   projID,
		TriggerType: "SCHEDULED",
		Summary:     "Cycle checking brief completeness",
	})
	if err != nil {
		t.Fatalf("RunCycle failed: %v", err)
	}

	if outcome.Decision == nil || outcome.Decision.ActionType != domainDecision.ActionAskHuman {
		t.Fatalf("expected ActionAskHuman decision, got: %+v", outcome.Decision)
	}

	// 7. Verify ClarificationRequest was created in database
	openReqs, err := clarifyRepo.ListByProject(ctx, projID)
	if err != nil {
		t.Fatalf("failed to list open clarification requests: %v", err)
	}
	if len(openReqs) == 0 {
		t.Fatalf("expected at least 1 open clarification request generated autonomously")
	}

	req := openReqs[0]
	if !req.Blocking {
		t.Errorf("expected clarification request to be blocking")
	}
	if len(req.Options) == 0 {
		t.Errorf("expected clarification request to include recommendation options")
	}

	// 8. Verify the running project task transitioned to WAITING_FOR_INPUT
	curTask, err := taskSvc.Get(ctx, createdTask.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if curTask.Status != domainTask.StatusWaitingInput {
		t.Errorf("expected running task to be suspended to WAITING_FOR_INPUT, got: %s", curTask.Status)
	}

	// 9. Simulate Human Answer: User provides clarification
	answeredReq, err := clarificationSvc.Answer(ctx, req.ID, "歷史傳奇小說 (Historical Fiction)")
	if err != nil {
		t.Fatalf("failed to answer clarification request: %v", err)
	}
	if answeredReq.Status != domainClarify.StatusAnswered {
		t.Errorf("expected status ANSWERED, got: %s", answeredReq.Status)
	}

	// 10. Verify Task is automatically unblocked back to READY
	resumedTask, err := taskSvc.Get(ctx, createdTask.ID)
	if err != nil {
		t.Fatalf("failed to get resumed task: %v", err)
	}
	if resumedTask.Status != domainTask.StatusReady {
		t.Errorf("expected task to be unblocked to READY after resolving clarification, got: %s", resumedTask.Status)
	}

	// 11. Verify Brief was updated with user answer
	updatedBrief, err := genericBriefRepo.GetByProjectID(ctx, projID)
	if err != nil {
		t.Fatalf("failed to get updated brief: %v", err)
	}
	field := updatedBrief.Fields[req.FieldKey]
	if field == nil || field.Status != domainBrief.FieldStatusAnswered {
		t.Errorf("expected field %s to be updated to ANSWERED, got: %+v", req.FieldKey, field)
	}
}
