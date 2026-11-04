package unit

import (
	"context"
	"testing"
	"time"

	domainPlan "github.com/wade/aia/internal/domain/plan"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/provider"
	"github.com/wade/aia/internal/provider/mock"
	problemService "github.com/wade/aia/internal/service/problem"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestProblemService_AnalyzeProblemWithProvider_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	probRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)
	projRepo := repository.NewProjectRepository(db)
	svc := problemService.NewService(probRepo, planRepo, projRepo, nil)

	proj := &domainProject.Project{
		ID:         "proj-ana-001",
		Name:       "Go Kernel Refactor",
		DomainType: "coding",
		Workspace:  "/tmp/workspace-kernel",
		CreatedAt:  now,
	}
	_ = projRepo.Create(ctx, proj)

	prob, err := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: proj.ID,
		Title:     "Kernel Memory Leak in Worker Pool",
		Objective: "Isolate memory leak when spawning goroutines under high load",
		Scope:     "internal/worker",
		Priority:  domainProblem.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	// Setup Mock Provider returning structured JSON compliant with AIA-22
	mockProv := mock.NewMockProvider()
	mockProv.StartFunc = func(ctx context.Context, req provider.StartRequest) (provider.ProviderSession, error) {
		sess := mock.NewMockSession(req)
		sess.Response = "```json\n" + `{
  "intent": {
    "user_goal": "Isolate memory leak in worker pool",
    "success_criteria": ["Profile pprof heap", "Fix leaked channels", "Zero leak regression"]
  },
  "context": {
    "found_resources": ["go/internal/worker/task_worker.go"],
    "trusted_data": ["PProf memory heap profile"]
  },
  "model_items": [
    {
      "id": "item-fact-1",
      "category": "KNOWN_FACT",
      "statement": "Goroutine count increases linearly under 100 concurrent tasks",
      "source": "pprof_goroutine_profile",
      "confidence": 1.0,
      "status": "CONFIRMED"
    },
    {
      "id": "item-unknown-1",
      "category": "UNKNOWN",
      "statement": "Which specific channel select case blocks goroutine termination",
      "confidence": 0.0,
      "status": "CONFIRMED"
    },
    {
      "id": "item-assumption-1",
      "category": "ASSUMPTION",
      "statement": "Closing task channel with explicit context cancellation will unblock worker leaks",
      "confidence": 0.85,
      "status": "ACTIVE"
    }
  ]
}` + "\n```"
		return sess, nil
	}
	svc.SetProvider(mockProv)

	// Analyze problem with Provider
	model, err := svc.AnalyzeProblemWithProvider(ctx, prob.ID, "Analyze the memory leak issue", "coding-agent")
	if err != nil {
		t.Fatalf("unexpected error in AnalyzeProblemWithProvider: %v", err)
	}

	if model == nil || model.Version != 1 {
		t.Fatalf("expected problem model version 1, got %+v", model)
	}
	if len(model.KnownFacts) != 1 {
		t.Errorf("expected 1 known fact, got %d", len(model.KnownFacts))
	}
	if len(model.Unknowns) != 1 {
		t.Errorf("expected 1 unknown, got %d", len(model.Unknowns))
	}
	if len(model.Assumptions) != 1 {
		t.Errorf("expected 1 assumption, got %d", len(model.Assumptions))
	}

	// Generate Plan from Analysis
	plan, err := svc.GeneratePlanFromAnalysis(ctx, prob.ID, false)
	if err != nil {
		t.Fatalf("failed to generate plan from analysis: %v", err)
	}
	if plan == nil || plan.Status != domainPlan.StatusProposed {
		t.Fatalf("expected proposed plan, got %+v", plan)
	}
	if len(plan.WorkItems) != 2 {
		t.Errorf("expected 2 work items, got %d", len(plan.WorkItems))
	}
	if plan.WorkItems[0].AssignedAgent != "coding-agent" {
		t.Errorf("expected coding-agent, got %s", plan.WorkItems[0].AssignedAgent)
	}
}

func TestProblemService_AnalyzeProblemWithProvider_DomainFallbackSynthesis(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	probRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)
	projRepo := repository.NewProjectRepository(db)
	svc := problemService.NewService(probRepo, planRepo, projRepo, nil)

	// Test Writing Domain
	projWriting := &domainProject.Project{
		ID:         "proj-writing-domain",
		Name:       "Yuan Dynasty Epic",
		DomainType: "writing",
		Workspace:  "/tmp/novel",
		CreatedAt:  now,
	}
	_ = projRepo.Create(ctx, projWriting)

	probWriting, err := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: projWriting.ID,
		Title:     "Volume 3 Historical Discrepancy",
		Objective: "Reconcile Tuotuo timeline contradiction in Chapter 12",
	})
	if err != nil {
		t.Fatalf("failed to create writing problem: %v", err)
	}

	// No provider configured -> Synthesizes writing domain analysis
	modelW, err := svc.AnalyzeProblemWithProvider(ctx, probWriting.ID, "Check historical timeline", "writing-agent")
	if err != nil {
		t.Fatalf("failed to analyze writing problem: %v", err)
	}
	if modelW.Version != 1 {
		t.Errorf("expected model version 1, got %d", modelW.Version)
	}
	if len(modelW.KnownFacts) == 0 || modelW.KnownFacts[0].Confidence != 1.0 {
		t.Errorf("expected valid known fact with confidence 1.0")
	}

	planW, err := svc.GeneratePlanFromAnalysis(ctx, probWriting.ID, true)
	if err != nil {
		t.Fatalf("failed to generate writing plan: %v", err)
	}
	if planW.WorkItems[0].AssignedAgent != "writing-agent" {
		t.Errorf("expected writing-agent assigned, got %s", planW.WorkItems[0].AssignedAgent)
	}
	if planW.Status != domainPlan.StatusProposed {
		t.Errorf("expected plan status PROPOSED, got %s", planW.Status)
	}

	// Test Investment Domain
	projInv := &domainProject.Project{
		ID:         "proj-inv-domain",
		Name:       "Dual Cohort Portfolio",
		DomainType: "investment",
		Workspace:  "/tmp/inv",
		CreatedAt:  now,
	}
	_ = projRepo.Create(ctx, projInv)

	probInv, err := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: projInv.ID,
		Title:     "TSMC Weight Drift Rebalance",
		Objective: "Evaluate 2330 allocation after quarterly earnings release",
	})
	if err != nil {
		t.Fatalf("failed to create investment problem: %v", err)
	}

	modelI, err := svc.AnalyzeProblemWithProvider(ctx, probInv.ID, "Evaluate earnings rebalance", "investment-agent")
	if err != nil {
		t.Fatalf("failed to analyze investment problem: %v", err)
	}
	if modelI.Version != 1 {
		t.Errorf("expected investment model version 1, got %d", modelI.Version)
	}

	planI, err := svc.GeneratePlanFromAnalysis(ctx, probInv.ID, false)
	if err != nil {
		t.Fatalf("failed to generate investment plan: %v", err)
	}
	if planI.WorkItems[0].AssignedAgent != "investment-agent" {
		t.Errorf("expected investment-agent assigned, got %s", planI.WorkItems[0].AssignedAgent)
	}
	if len(planI.WorkItems[1].Dependencies) == 0 || planI.WorkItems[1].Dependencies[0] != planI.WorkItems[0].ID {
		t.Errorf("expected WorkItem 2 to depend on WorkItem 1 in DAG")
	}
}
