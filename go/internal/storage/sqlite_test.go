package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/agent-runtime/internal/storage"
)

func setupTestDB(t *testing.T) (*storage.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "agent-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("failed to open test db: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll(tmpDir)
	}
	return db, cleanup
}

func TestTaskCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	taskID := uuid.New().String()
	now := time.Now().Truncate(time.Millisecond)

	task := &storage.TaskRecord{
		ID:            taskID,
		Title:         "Unit Test Task",
		Description:   "Testing SQLite CRUD",
		Status:        "CREATED",
		WorkerType:    "aider",
		WorkspaceID:   "ws-123",
		CreatedAt:     now,
		LastUpdatedAt: now,
	}

	// 1. Save
	if err := db.SaveTask(ctx, task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// 2. Get
	fetched, err := db.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if fetched.Title != task.Title {
		t.Errorf("expected title %q, got %q", task.Title, fetched.Title)
	}
	if fetched.Status != "CREATED" {
		t.Errorf("expected status CREATED, got %s", fetched.Status)
	}

	// 3. Update Status
	now2 := time.Now()
	if err := db.UpdateTaskStatus(ctx, taskID, "RUNNING", now2); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	updated, err := db.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask failed after update: %v", err)
	}
	if updated.Status != "RUNNING" {
		t.Errorf("expected status RUNNING, got %s", updated.Status)
	}

	// 4. List
	tasks, err := db.ListTasks(ctx)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in list, got %d", len(tasks))
	}
}

func TestWorkerExecutionCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	execID := uuid.New().String()
	taskID := uuid.New().String()
	start := time.Now()

	exec := &storage.WorkerExecutionRecord{
		ID:            execID,
		TaskID:        taskID,
		Status:        "STARTING",
		WorkspacePath: "/tmp/ws",
		StartedAt:     &start,
	}

	if err := db.SaveWorkerExecution(ctx, exec); err != nil {
		t.Fatalf("SaveWorkerExecution failed: %v", err)
	}

	// Update with PID
	exec.Status = "RUNNING"
	exec.PID = 1234
	if err := db.UpdateWorkerExecution(ctx, exec); err != nil {
		t.Fatalf("UpdateWorkerExecution failed: %v", err)
	}

	fetched, err := db.GetWorkerExecution(ctx, execID)
	if err != nil {
		t.Fatalf("GetWorkerExecution failed: %v", err)
	}
	if fetched.PID != 1234 || fetched.Status != "RUNNING" {
		t.Errorf("unexpected exec record: %+v", fetched)
	}

	// Latest by task
	latest, err := db.GetLatestWorkerExecutionByTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetLatestWorkerExecutionByTask failed: %v", err)
	}
	if latest.ID != execID {
		t.Errorf("expected execID %s, got %s", execID, latest.ID)
	}
}

func TestTaskEventsAndAuditLogs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	taskID := uuid.New().String()

	ev := &storage.TaskEventRecord{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		EventType: "CREATED",
		Detail:    "Task initialized",
		CreatedAt: time.Now(),
	}
	if err := db.SaveTaskEvent(ctx, ev); err != nil {
		t.Fatalf("SaveTaskEvent failed: %v", err)
	}

	events, err := db.ListTaskEvents(ctx, taskID)
	if err != nil || len(events) != 1 {
		t.Fatalf("ListTaskEvents failed: %v", err)
	}

	auditLog := &storage.AuditLogRecord{
		ID:         uuid.New().String(),
		Level:      "INFO",
		Action:     "test_action",
		EntityType: "task",
		EntityID:   taskID,
		Detail:     "test log detail",
		CreatedAt:  time.Now(),
	}
	if err := db.SaveAuditLog(ctx, auditLog); err != nil {
		t.Fatalf("SaveAuditLog failed: %v", err)
	}

	logs, err := db.ListAuditLogs(ctx, "task", taskID)
	if err != nil || len(logs) != 1 {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
}
