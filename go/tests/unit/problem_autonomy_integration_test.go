package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy/initiative"
	"github.com/wade/aia/internal/autonomy/recall"
	"github.com/wade/aia/internal/continuity"
	"github.com/wade/aia/internal/domain"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	domainPlan "github.com/wade/aia/internal/domain/plan"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	domainTask "github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/orchestrator"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestProblemAutonomyCoreIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	pRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)
	cRepo := repository.NewContinuityRepository(db)
	gRepo := repository.NewGoalRepository(db)
	dRepo := repository.NewDecisionRepository(db)
	eRepo := repository.NewEventHistoryRepository(db)
	memRepo := repository.NewMemoryRepository(db)

	now := time.Now().UTC()
	projID := "proj-auto-prob-001"

	err := pRepo.Create(ctx, &domainProject.Project{
		ID:        projID,
		Name:      "Problem Autonomy Integration Test",
		Workspace: "/tmp/prob-test",
		Status:    domainProject.StatusActive,
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	prob := &domainProblem.Problem{
		ID:                  "prob-001",
		ProjectID:           projID,
		Title:               "High Memory Consumption Under Load",
		Description:         "Profiling shows leak in worker pools",
		Objective:           "Identify and plug memory leak in worker pool",
		Status:              domainProblem.StatusOpen,
		Priority:            domainProblem.PriorityHigh,
		CurrentModelVersion: 1,
		CurrentPlanVersion:  1,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := problemRepo.Create(ctx, prob); err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	plan := &domainPlan.Plan{
		ID:        "plan-001",
		ProblemID: prob.ID,
		Version:   1,
		Objective: "Resolve memory leaks in worker pool",
		Status:    domainPlan.StatusDraft,
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-1",
				Title:              "Profile memory allocation",
				Description:        "Run pprof on worker pool",
				AcceptanceCriteria: []string{"profiling data captured"},
				Status:             domainPlan.WorkItemPending,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := planRepo.Create(ctx, plan); err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	t.Run("Reconstructor restores active problems", func(t *testing.T) {
		recon := continuity.NewReconstructor(cRepo, gRepo, dRepo, eRepo, pRepo)
		recon.SetProblemRepository(problemRepo)

		state, err := recon.ReconstructProjectState(ctx, projID)
		if err != nil {
			t.Fatalf("ReconstructProjectState failed: %v", err)
		}
		if len(state.ActiveProblems) != 1 {
			t.Fatalf("expected 1 active problem, got %d", len(state.ActiveProblems))
		}
		if state.ActiveProblems[0].ID != prob.ID {
			t.Errorf("expected problem %s, got %s", prob.ID, state.ActiveProblems[0].ID)
		}
	})

	var bundle *recall.RecalledBundle
	t.Run("Recall engine retrieves active problems and plans", func(t *testing.T) {
		recallEng := recall.NewEngine(eRepo, nil, memRepo, cRepo, gRepo, dRepo, pRepo, nil)
		recallEng.SetProblemRepository(problemRepo)
		recallEng.SetPlanRepository(planRepo)

		var err error
		bundle, err = recallEng.Recall(ctx, recall.RecallInput{ProjectID: projID})
		if err != nil {
			t.Fatalf("Recall failed: %v", err)
		}
		if len(bundle.ActiveProblems) != 1 {
			t.Fatalf("expected 1 active problem in bundle, got %d", len(bundle.ActiveProblems))
		}
		if len(bundle.ActivePlans) != 1 {
			t.Fatalf("expected 1 active plan in bundle, got %d", len(bundle.ActivePlans))
		}
		if bundle.ActivePlans[0].ID != plan.ID {
			t.Errorf("expected plan %s, got %s", plan.ID, bundle.ActivePlans[0].ID)
		}
	})

	t.Run("Discoverer generates initiative from open problem", func(t *testing.T) {
		disc := initiative.NewDiscoverer(nil)

		candidates, err := disc.Discover(ctx, initiative.DiscoverInput{ProjectID: projID, Recalled: bundle})
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}
		var found bool
		for _, c := range candidates {
			if c.DiscoveryType == domainAutonomy.DiscoveryProblem && c.ProblemID != nil && *c.ProblemID == prob.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected candidate initiative from problem %s, got %+v", prob.ID, candidates)
		}
	})

	t.Run("ContextAssembler includes active problem context", func(t *testing.T) {
		ca := orchestrator.NewContextAssembler(pRepo, memRepo, nil)
		ca.SetProblemRepository(problemRepo)

		testTask := &domainTask.Task{
			ID:          "task-prob-test",
			ProjectID:   domain.ID(projID),
			Title:       "Investigate pool leak",
			Description: "Run benchmarks",
		}

		ctxStr, err := ca.Assemble(ctx, testTask, nil)
		if err != nil {
			t.Fatalf("Assemble failed: %v", err)
		}
		if !containsSubstr(ctxStr, "ACTIVE PROBLEMS & PLANNING CONTEXT") || !containsSubstr(ctxStr, prob.Title) {
			t.Errorf("expected assembled context to contain problem info, got:\n%s", ctxStr)
		}
	})
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && searchSubstr(s, substr))
}

func searchSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
