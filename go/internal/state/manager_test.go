package state_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/agent-runtime/internal/audit"
	"github.com/wade/aia/agent-runtime/internal/state"
	"github.com/wade/aia/agent-runtime/internal/storage"
	"github.com/wade/aia/agent-runtime/internal/task"
)

func setupStateEnv(t *testing.T) (*state.Manager, *storage.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "agent-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	db, err := storage.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("failed to open test db: %v", err)
	}
	logger := audit.NewLogger(db)
	sm := state.NewManager(db, logger)

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll(tmpDir)
	}
	return sm, db, cleanup
}

func TestStateTransitions(t *testing.T) {
	sm, db, cleanup := setupStateEnv(t)
	defer cleanup()

	ctx := context.Background()
	taskID := uuid.New().String()
	now := time.Now()

	// Seed task in CREATED
	_ = db.SaveTask(ctx, &storage.TaskRecord{
		ID:            taskID,
		Title:         "State Test",
		Status:        string(task.TaskStatusCreated),
		CreatedAt:     now,
		LastUpdatedAt: now,
	})

	// Valid: CREATED -> RUNNING
	if err := sm.Transition(taskID, task.TaskStatusCreated, task.TaskStatusRunning); err != nil {
		t.Fatalf("Transition CREATED -> RUNNING failed: %v", err)
	}

	// Valid: RUNNING -> SUCCEEDED
	if err := sm.Transition(taskID, task.TaskStatusRunning, task.TaskStatusSucceeded); err != nil {
		t.Fatalf("Transition RUNNING -> SUCCEEDED failed: %v", err)
	}

	// Invalid: SUCCEEDED -> RUNNING (terminal state)
	if err := sm.Transition(taskID, task.TaskStatusSucceeded, task.TaskStatusRunning); err == nil {
		t.Fatalf("Expected error transitioning SUCCEEDED -> RUNNING, but got nil")
	}

	// Invalid: Mismatched 'from' state
	if err := sm.Transition(taskID, task.TaskStatusCreated, task.TaskStatusFailed); err == nil {
		t.Fatalf("Expected error when from status does not match current status")
	}
}

func TestReconciliationTransitions(t *testing.T) {
	sm, db, cleanup := setupStateEnv(t)
	defer cleanup()

	ctx := context.Background()
	taskID := uuid.New().String()
	now := time.Now()

	_ = db.SaveTask(ctx, &storage.TaskRecord{
		ID:            taskID,
		Title:         "Reconcile Test",
		Status:        string(task.TaskStatusRunning),
		CreatedAt:     now,
		LastUpdatedAt: now,
	})

	// RUNNING -> UNKNOWN
	if err := sm.Transition(taskID, task.TaskStatusRunning, task.TaskStatusUnknown); err != nil {
		t.Fatalf("Transition RUNNING -> UNKNOWN failed: %v", err)
	}

	// UNKNOWN -> RECONCILING
	if err := sm.Transition(taskID, task.TaskStatusUnknown, task.TaskStatusReconciling); err != nil {
		t.Fatalf("Transition UNKNOWN -> RECONCILING failed: %v", err)
	}

	// RECONCILING -> NEEDS_REVIEW
	if err := sm.Transition(taskID, task.TaskStatusReconciling, task.TaskStatusNeedsReview); err != nil {
		t.Fatalf("Transition RECONCILING -> NEEDS_REVIEW failed: %v", err)
	}
}
