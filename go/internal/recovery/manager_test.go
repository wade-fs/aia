package recovery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/agent-runtime/internal/audit"
	"github.com/wade/aia/agent-runtime/internal/git"
	"github.com/wade/aia/agent-runtime/internal/recovery"
	"github.com/wade/aia/agent-runtime/internal/storage"
	"github.com/wade/aia/agent-runtime/internal/task"
	"github.com/wade/aia/agent-runtime/internal/validator"
	"github.com/wade/aia/agent-runtime/internal/worker"
)

type dummyWorker struct{}

func (d *dummyWorker) Initialize(ctx context.Context) error { return nil }
func (d *dummyWorker) Execute(ctx context.Context, req worker.WorkerRequest) (*worker.WorkerResult, error) {
	return nil, nil
}
func (d *dummyWorker) GetStatus(ctx context.Context, id string) (*worker.WorkerStatus, error) {
	return nil, nil
}
func (d *dummyWorker) Cancel(ctx context.Context, id string) error { return nil }
func (d *dummyWorker) HealthCheck(ctx context.Context) error      { return nil }

func setupRecoveryEnv(t *testing.T) (*recovery.Manager, *storage.DB, *task.Manager, func()) {
	tmpDir, err := os.MkdirTemp("", "agent-recovery-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	db, err := storage.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("failed to open test db: %v", err)
	}

	logger := audit.NewLogger(db)
	tm := task.NewManager(db, logger)
	wm := worker.NewManager(&dummyWorker{}, db, logger, "gpt-4o", 5*time.Minute)
	val := validator.NewValidator("go test ./...")
	rm := recovery.NewManager(db, wm, tm, logger, val)

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll(tmpDir)
	}
	return rm, db, tm, cleanup
}

func TestRecoveryMissingWorkspace(t *testing.T) {
	rm, db, tm, cleanup := setupRecoveryEnv(t)
	defer cleanup()

	ctx := context.Background()
	taskID := uuid.New().String()
	now := time.Now()

	// Seed task in RUNNING
	_ = db.SaveTask(ctx, &storage.TaskRecord{
		ID:            taskID,
		Title:         "Crash Task",
		Status:        string(task.TaskStatusRunning),
		CreatedAt:     now,
		LastUpdatedAt: now,
	})

	// Seed execution with non-existent workspace
	_ = db.SaveWorkerExecution(ctx, &storage.WorkerExecutionRecord{
		ID:            uuid.New().String(),
		TaskID:        taskID,
		Status:        "RUNNING",
		PID:           999999, // Dead PID
		WorkspacePath: "/path/that/does/not/exist",
		StartedAt:     &now,
	})

	// Run recovery
	if err := rm.Run(ctx); err != nil {
		t.Fatalf("Recovery Run failed: %v", err)
	}

	// Task must be marked NEEDS_REVIEW
	recoveredTask, err := tm.Get(taskID)
	if err != nil {
		t.Fatalf("Get task failed: %v", err)
	}
	if recoveredTask.Status != task.TaskStatusNeedsReview {
		t.Errorf("Expected status NEEDS_REVIEW, got %s", recoveredTask.Status)
	}
}

func TestRecoverySuccessfulReconciliation(t *testing.T) {
	rm, db, tm, cleanup := setupRecoveryEnv(t)
	defer cleanup()

	ctx := context.Background()
	taskID := uuid.New().String()
	now := time.Now()

	// Create workspace on disk
	wsDir, err := os.MkdirTemp("", "agent-rec-ws-*")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(wsDir)

	_ = os.WriteFile(filepath.Join(wsDir, "go.mod"), []byte("module recws\n\ngo 1.21\n"), 0644)
	_ = os.WriteFile(filepath.Join(wsDir, "main.go"), []byte("package main\n\nfunc Hello() string { return \"hi\" }\n"), 0644)
	_ = os.WriteFile(filepath.Join(wsDir, "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) {}\n"), 0644)

	g := git.NewClient(wsDir)
	_ = g.Init()
	_ = g.AddAndCommit("initial commit")
	baseCommit, _ := g.GetCurrentCommit()

	// Worker modified a file before runtime crashed
	_ = os.WriteFile(filepath.Join(wsDir, "main.go"), []byte("package main\n\nfunc Hello() string { return \"hello\" }\n"), 0644)

	// Seed task in RUNNING
	_ = db.SaveTask(ctx, &storage.TaskRecord{
		ID:            taskID,
		Title:         "Interrupted Task",
		Status:        string(task.TaskStatusRunning),
		GitBaseCommit: baseCommit,
		CreatedAt:     now,
		LastUpdatedAt: now,
	})

	exitZero := 0
	_ = db.SaveWorkerExecution(ctx, &storage.WorkerExecutionRecord{
		ID:            uuid.New().String(),
		TaskID:        taskID,
		Status:        "RUNNING",
		PID:           999999, // Dead PID
		ExitCode:      &exitZero,
		WorkspacePath: wsDir,
		StartedAt:     &now,
	})

	// Run recovery
	if err := rm.Run(ctx); err != nil {
		t.Fatalf("Recovery Run failed: %v", err)
	}

	// Should reconcile and mark SUCCEEDED
	recTask, err := tm.Get(taskID)
	if err != nil {
		t.Fatalf("Get task failed: %v", err)
	}
	if recTask.Status != task.TaskStatusSucceeded {
		t.Errorf("Expected status SUCCEEDED after successful reconciliation, got %s", recTask.Status)
	}
}
