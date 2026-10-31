package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type Summary struct {
	InterruptedExecutions int
	RecoveredTasks        int
	ReadyTasksPromoted    int
}

type Manager struct {
	taskRepo      *repository.TaskRepository
	executionRepo *repository.ExecutionRepository
	auditRepo     *repository.AuditRepository
}

func NewManager(
	taskRepo *repository.TaskRepository,
	executionRepo *repository.ExecutionRepository,
	auditRepo *repository.AuditRepository,
) *Manager {
	return &Manager{
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		auditRepo:     auditRepo,
	}
}

func (m *Manager) Recover(ctx context.Context) (*Summary, error) {
	summary := &Summary{}

	// 1. Find all running tasks
	runningTasks, err := m.taskRepo.List(ctx, repository.TaskFilter{Status: task.StatusRunning})
	if err != nil {
		return nil, fmt.Errorf("failed to query running tasks for recovery: %w", err)
	}

	for _, t := range runningTasks {
		execs, err := m.executionRepo.ListByTask(ctx, t.ID)
		alreadySucceeded := false
		if err == nil {
			for _, ex := range execs {
				// Check if result was already achieved before crash
				if ex.Status == execution.StatusCompleted || ex.ResultStatus == "SUCCESS" {
					alreadySucceeded = true
					break
				}
				if ex.Status == execution.StatusRunning || ex.Status == execution.StatusStarting {
					ex.Status = execution.StatusInterrupted
					ex.ErrorMessage = "Process interrupted by unexpected runtime shutdown"
					now := time.Now().UTC()
					ex.EndedAt = &now
					_ = m.executionRepo.Update(ctx, &ex)
					summary.InterruptedExecutions++
				}
			}
		}

		if alreadySucceeded {
			_ = m.taskRepo.UpdateStatus(ctx, t.ID, task.StatusCompleted)
			slog.Info("recovered completed task from already succeeded execution", "task_id", t.ID)
			summary.RecoveredTasks++
			continue
		}

		// Reconcile task state
		if t.RetryCount < t.MaxRetries {
			t.RetryCount++
			_ = m.taskRepo.Update(ctx, &t)
			_ = m.taskRepo.UpdateStatus(ctx, t.ID, task.StatusReady)
			slog.Info("recovered interrupted task to READY", "task_id", t.ID)
		} else {
			_ = m.taskRepo.UpdateStatus(ctx, t.ID, task.StatusFailed)
			slog.Warn("max retries exceeded for interrupted task, marked FAILED", "task_id", t.ID)
		}
		summary.RecoveredTasks++
	}

	// 2. Check WAITING_DEPENDENCY tasks
	waitingTasks, err := m.taskRepo.List(ctx, repository.TaskFilter{Status: task.StatusWaitingDependency})
	if err == nil {
		for _, wt := range waitingTasks {
			completed, err := m.taskRepo.AreDependenciesCompleted(ctx, wt.ID)
			if err == nil && completed {
				_ = m.taskRepo.UpdateStatus(ctx, wt.ID, task.StatusReady)
				summary.ReadyTasksPromoted++
			}
		}
	}

	return summary, nil
}
