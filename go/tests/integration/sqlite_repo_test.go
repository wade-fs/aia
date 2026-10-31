package integration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/memory"
	"github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func setupTestDB(t *testing.T) *sqlite.DB {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	return db
}

func TestProjectAndAgentCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// 1. Project
	projRepo := repository.NewProjectRepository(db)
	p := &project.Project{
		ID:        uuid.New().String(),
		Name:      "Test Project",
		Workspace: "/tmp/test",
		Status:    project.StatusActive,
	}
	if err := projRepo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	gotP, err := projRepo.Get(ctx, p.ID)
	if err != nil || gotP.Name != "Test Project" {
		t.Errorf("failed to get project correctly: %v", err)
	}

	// 2. Agent
	agRepo := repository.NewAgentRepository(db)
	ag := &agent.Agent{
		ID:           "test-agent",
		Name:         "Test Agent",
		Domain:       "test",
		Instructions: "Run tests",
		Capabilities: []string{"test"},
		ProviderName: "antigravity",
		Enabled:      true,
		Version:      1,
	}
	if err := agRepo.Upsert(ctx, ag); err != nil {
		t.Fatalf("failed to upsert agent: %v", err)
	}

	gotAg, err := agRepo.Get(ctx, ag.ID)
	if err != nil || gotAg.Name != "Test Agent" {
		t.Errorf("failed to get agent correctly: %v", err)
	}
}

func TestTaskAndDependencyCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	p := &project.Project{ID: "p1", Name: "P1", Workspace: "/tmp", Status: project.StatusActive}
	_ = projRepo.Create(ctx, p)

	taskRepo := repository.NewTaskRepository(db)

	// Task 1
	t1 := &task.Task{
		ID:          "t1",
		ProjectID:   "p1",
		Title:       "Task 1",
		Description: "Base task",
		Status:      task.StatusCreated,
		Priority:    50,
		Risk:        domain.RiskLow,
		MaxRetries:  2,
	}
	if err := taskRepo.Create(ctx, t1, nil); err != nil {
		t.Fatalf("failed to create t1: %v", err)
	}

	// Task 2 depends on Task 1
	t2 := &task.Task{
		ID:          "t2",
		ProjectID:   "p1",
		Title:       "Task 2",
		Description: "Dependent task",
		Status:      task.StatusWaitingDependency,
		Priority:    50,
		Risk:        domain.RiskLow,
		MaxRetries:  2,
	}
	if err := taskRepo.Create(ctx, t2, []domain.ID{"t1"}); err != nil {
		t.Fatalf("failed to create t2: %v", err)
	}

	deps, err := taskRepo.GetDependencies(ctx, "t2")
	if err != nil || len(deps) != 1 || deps[0] != "t1" {
		t.Errorf("expected t2 to depend on t1, got: %v", deps)
	}

	completed, _ := taskRepo.AreDependenciesCompleted(ctx, "t2")
	if completed {
		t.Errorf("expected dependencies not completed")
	}

	// Update t1 to COMPLETED
	_ = taskRepo.UpdateStatus(ctx, "t1", task.StatusReady)
	_ = taskRepo.UpdateStatus(ctx, "t1", task.StatusRunning)
	_ = taskRepo.UpdateStatus(ctx, "t1", task.StatusCompleted)

	completed, _ = taskRepo.AreDependenciesCompleted(ctx, "t2")
	if !completed {
		t.Errorf("expected dependencies completed after t1 completed")
	}
}

func TestExecutionAndMonotonicEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	_ = projRepo.Create(ctx, &project.Project{ID: "p1", Name: "P1", Workspace: "/tmp", Status: project.StatusActive})
	taskRepo := repository.NewTaskRepository(db)
	_ = taskRepo.Create(ctx, &task.Task{ID: "t1", ProjectID: "p1", Title: "T1", Status: task.StatusReady, Risk: domain.RiskLow}, nil)
	agRepo := repository.NewAgentRepository(db)
	_ = agRepo.Upsert(ctx, &agent.Agent{ID: "ag1", Name: "A1", Domain: "d", ProviderName: "p", Enabled: true, Version: 1})

	execRepo := repository.NewExecutionRepository(db)
	ex := &execution.Execution{
		ID:        "ex1",
		TaskID:    "t1",
		AgentID:   "ag1",
		Provider:  "mock",
		Workspace: "/tmp/ws",
		Status:    execution.StatusRunning,
	}
	if err := execRepo.Create(ctx, ex); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	// Append monotonic events
	payload := json.RawMessage(`{"text":"hello"}`)
	for i := 0; i < 3; i++ {
		ev := &execution.ExecutionEvent{
			ID:          uuid.New().String(),
			ExecutionID: "ex1",
			EventType:   "step_update",
			Source:      "mock",
			Payload:     payload,
			CreatedAt:   time.Now().UTC(),
		}
		if err := execRepo.AppendEvent(ctx, ev); err != nil {
			t.Fatalf("failed to append event: %v", err)
		}
	}

	events, err := execRepo.ListEvents(ctx, "ex1", 0)
	if err != nil || len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	for idx, ev := range events {
		expectedSeq := uint64(idx + 1)
		if ev.Sequence != expectedSeq {
			t.Errorf("expected sequence %d, got %d", expectedSeq, ev.Sequence)
		}
	}
}

func TestMemoryAndCandidateWorkflow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	memRepo := repository.NewMemoryRepository(db)

	cand := &memory.MemoryCandidate{
		ID:         "c1",
		Type:       memory.TypeFact,
		Scope:      memory.ScopeGlobal,
		Title:      "Go standard convention",
		Content:    "Always check errors",
		Confidence: 0.9,
	}
	if err := memRepo.CreateCandidate(ctx, cand); err != nil {
		t.Fatalf("failed to create candidate: %v", err)
	}

	// List pending
	cands, err := memRepo.ListCandidates(ctx, memory.ReviewPending)
	if err != nil || len(cands) != 1 {
		t.Fatalf("expected 1 pending candidate, got %d", len(cands))
	}

	// Approve
	_ = memRepo.UpdateCandidateStatus(ctx, "c1", memory.ReviewApproved)
	_ = memRepo.Create(ctx, &memory.Memory{
		ID:         "m1",
		Type:       cand.Type,
		Scope:      cand.Scope,
		Title:      cand.Title,
		Content:    cand.Content,
		Confidence: cand.Confidence,
		Status:     "ACTIVE",
		Version:    1,
	})

	// Search
	results, err := memRepo.Search(ctx, repository.MemoryQuery{Keyword: "convention"})
	if err != nil || len(results) != 1 {
		t.Errorf("expected to find approved memory, got %d results", len(results))
	}
}
