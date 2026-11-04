package unit

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestLogRetention_PurgeAuditAndExecutionEvents(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test_retention.db")
	db, err := sqlite.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	auditRepo := repository.NewAuditRepository(db)
	agRepo := repository.NewAgentRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	execRepo := repository.NewExecutionRepository(db)

	now := time.Now().UTC()
	oldTime := now.AddDate(0, 0, -45)   // 45 days ago
	recentTime := now.AddDate(0, 0, -5) // 5 days ago

	// 1. Insert audit logs: 1 old, 1 recent
	err = auditRepo.Append(ctx, &repository.AuditLogEntry{
		ID:        uuid.New().String(),
		ActorType: "SYSTEM",
		ActorID:   "system",
		Action:    "OLD_ACTION",
		CreatedAt: oldTime,
	})
	if err != nil {
		t.Fatalf("failed to append old audit log: %v", err)
	}

	err = auditRepo.Append(ctx, &repository.AuditLogEntry{
		ID:        uuid.New().String(),
		ActorType: "SYSTEM",
		ActorID:   "system",
		Action:    "RECENT_ACTION",
		CreatedAt: recentTime,
	})
	if err != nil {
		t.Fatalf("failed to append recent audit log: %v", err)
	}

	// 2. Create Agent, Task, and Execution
	agentID := uuid.New().String()
	err = agRepo.Upsert(ctx, &agent.Agent{
		ID:           agentID,
		Name:         "Test Agent",
		Domain:       "test",
		Enabled:      true,
		ProviderName: "mock",
		CreatedAt:    oldTime,
		UpdatedAt:    oldTime,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	taskID := uuid.New().String()
	err = taskRepo.Create(ctx, &task.Task{
		ID:        taskID,
		AgentID:   agentID,
		Title:     "Test Retention Task",
		Status:    task.StatusCompleted,
		Priority:  50,
		Risk:      "LOW",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
	}, nil)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	execID := uuid.New().String()
	err = execRepo.Create(ctx, &execution.Execution{
		ID:        execID,
		TaskID:    taskID,
		AgentID:   agentID,
		Provider:  "mock",
		Workspace: "/tmp/test",
		Status:    execution.StatusCompleted,
		StartedAt: &oldTime,
		CreatedAt: oldTime,
	})
	if err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	err = execRepo.AppendEvent(ctx, &execution.ExecutionEvent{
		ID:          uuid.New().String(),
		ExecutionID: execID,
		EventType:   "OUTPUT",
		Sequence:    1,
		Payload:     json.RawMessage(`{}`),
		CreatedAt:   oldTime,
	})
	if err != nil {
		t.Fatalf("failed to append old execution event: %v", err)
	}

	err = execRepo.AppendEvent(ctx, &execution.ExecutionEvent{
		ID:          uuid.New().String(),
		ExecutionID: execID,
		EventType:   "OUTPUT",
		Sequence:    2,
		Payload:     json.RawMessage(`{}`),
		CreatedAt:   recentTime,
	})
	if err != nil {
		t.Fatalf("failed to append recent execution event: %v", err)
	}

	// 3. Purge older than 30 days
	cutoff := now.AddDate(0, 0, -30)
	purgedAudit, err := auditRepo.PurgeOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("PurgeOlderThan failed: %v", err)
	}
	if purgedAudit != 1 {
		t.Errorf("expected 1 audit log purged, got %d", purgedAudit)
	}

	purgedEvents, err := execRepo.PurgeEventsOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("PurgeEventsOlderThan failed: %v", err)
	}
	if purgedEvents != 1 {
		t.Errorf("expected 1 execution event purged, got %d", purgedEvents)
	}

	// 4. Verify remaining audit logs
	remainingAudit, err := auditRepo.List(ctx, 10)
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}
	if len(remainingAudit) != 1 || remainingAudit[0].Action != "RECENT_ACTION" {
		t.Errorf("expected only RECENT_ACTION audit log remaining, got %v", remainingAudit)
	}

	// 5. Verify remaining execution events
	remainingEvents, err := execRepo.ListEvents(ctx, execID, 0)
	if err != nil {
		t.Fatalf("failed to list execution events: %v", err)
	}
	if len(remainingEvents) != 1 || remainingEvents[0].Sequence != 2 {
		t.Errorf("expected only event sequence 2 remaining, got %v", remainingEvents)
	}
}
