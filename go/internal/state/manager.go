package state

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/agent-runtime/internal/audit"
	"github.com/wade/aia/agent-runtime/internal/storage"
	"github.com/wade/aia/agent-runtime/internal/task"
)

var allowedTransitions = map[task.TaskStatus]map[task.TaskStatus]bool{
	task.TaskStatusCreated: {
		task.TaskStatusReady:     true,
		task.TaskStatusRunning:   true,
		task.TaskStatusCancelled: true,
	},
	task.TaskStatusReady: {
		task.TaskStatusRunning:   true,
		task.TaskStatusCancelled: true,
	},
	task.TaskStatusRunning: {
		task.TaskStatusSucceeded: true,
		task.TaskStatusFailed:    true,
		task.TaskStatusCancelled: true,
		task.TaskStatusUnknown:   true,
	},
	task.TaskStatusUnknown: {
		task.TaskStatusReconciling: true,
	},
	task.TaskStatusReconciling: {
		task.TaskStatusSucceeded:   true,
		task.TaskStatusFailed:      true,
		task.TaskStatusNeedsReview: true,
	},
	task.TaskStatusSucceeded:   {},
	task.TaskStatusFailed:      {},
	task.TaskStatusCancelled:   {},
	task.TaskStatusNeedsReview: {},
}

type Manager struct {
	storage *storage.DB
	audit   *audit.Logger
}

func NewManager(db *storage.DB, logger *audit.Logger) *Manager {
	return &Manager{
		storage: db,
		audit:   logger,
	}
}

func (m *Manager) IsAllowed(from, to task.TaskStatus) bool {
	targets, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

func (m *Manager) Transition(taskID string, from, to task.TaskStatus) error {
	ctx := context.Background()
	taskRecord, err := m.storage.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task %s not found: %w", taskID, err)
	}

	currentStatus := task.TaskStatus(taskRecord.Status)

	// If caller specified an expected 'from' state, verify it matches
	if from != "" && currentStatus != from {
		return fmt.Errorf("cannot transition task %s: current status %s does not match expected from status %s",
			taskID, currentStatus, from)
	}

	if !m.IsAllowed(currentStatus, to) {
		return fmt.Errorf("invalid state transition for task %s: %s -> %s is not permitted",
			taskID, currentStatus, to)
	}

	now := time.Now()
	taskRecord.Status = string(to)
	taskRecord.LastUpdatedAt = now
	if to == task.TaskStatusRunning && taskRecord.StartedAt == nil {
		taskRecord.StartedAt = &now
	}
	if to.IsTerminal() && taskRecord.CompletedAt == nil {
		taskRecord.CompletedAt = &now
	}

	if err := m.storage.SaveTask(ctx, taskRecord); err != nil {
		return fmt.Errorf("failed to persist task status update: %w", err)
	}

	// Record task event
	event := &storage.TaskEventRecord{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		EventType: fmt.Sprintf("TRANSITION_%s_TO_%s", currentStatus, to),
		Detail:    fmt.Sprintf("State transitioned from %s to %s", currentStatus, to),
		CreatedAt: now,
	}
	_ = m.storage.SaveTaskEvent(ctx, event)

	if m.audit != nil {
		_ = m.audit.LogInfo("state_transition", "task", taskID, fmt.Sprintf("%s -> %s", currentStatus, to))
	}

	return nil
}
