package recovery

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/wade/aia/agent-runtime/internal/audit"
	"github.com/wade/aia/agent-runtime/internal/git"
	"github.com/wade/aia/agent-runtime/internal/storage"
	"github.com/wade/aia/agent-runtime/internal/task"
	"github.com/wade/aia/agent-runtime/internal/validator"
	"github.com/wade/aia/agent-runtime/internal/worker"
)

type Manager struct {
	storage       *storage.DB
	workerManager *worker.Manager
	taskManager   *task.Manager
	audit         *audit.Logger
	validator     *validator.Validator
}

func NewManager(
	db *storage.DB,
	wm *worker.Manager,
	tm *task.Manager,
	logger *audit.Logger,
	v *validator.Validator,
) *Manager {
	return &Manager{
		storage:       db,
		workerManager: wm,
		taskManager:   tm,
		audit:         logger,
		validator:     v,
	}
}

func (m *Manager) Run(ctx context.Context) error {
	if m.audit != nil {
		_ = m.audit.LogInfo("recovery_cycle_start", "system", "runtime", "Scanning for orphaned or interrupted tasks")
	}

	// 1. Query all tasks with status RUNNING or STARTING (and UNKNOWN for reconciliation)
	tasks, err := m.taskManager.ListByStatuses(
		task.TaskStatusRunning,
		task.TaskStatus("STARTING"),
	)
	if err != nil {
		return fmt.Errorf("failed to query tasks for recovery: %w", err)
	}

	for _, t := range tasks {
		if err := m.recoverTask(ctx, t); err != nil {
			if m.audit != nil {
				_ = m.audit.LogError("recovery_task_error", "task", t.ID, err.Error())
			}
		}
	}

	return nil
}

func (m *Manager) recoverTask(ctx context.Context, t *task.Task) error {
	exec, err := m.storage.GetLatestWorkerExecutionByTask(ctx, t.ID)
	if err != nil {
		// No execution record found
		if t.WorkspaceID != "" {
			_ = m.taskManager.UpdateStatus(t.ID, task.TaskStatusUnknown, "Recovery: No worker execution record found")
		} else {
			_ = m.taskManager.UpdateStatus(t.ID, task.TaskStatusNeedsReview, "Recovery: Neither execution nor workspace exists")
		}
		return nil
	}

	// Check if workspace exists
	if exec.WorkspacePath == "" || !isDir(exec.WorkspacePath) {
		_ = m.taskManager.UpdateStatus(t.ID, task.TaskStatusNeedsReview,
			fmt.Sprintf("Recovery: Workspace path %q does not exist", exec.WorkspacePath))
		if m.audit != nil {
			_ = m.audit.LogWarn("recovery_workspace_missing", "task", t.ID, "Workspace not found, marked NEEDS_REVIEW")
		}
		return nil
	}

	// Check PID if worker is still running
	if exec.PID > 0 {
		proc, err := os.FindProcess(exec.PID)
		if err == nil {
			// Signal(0) tests if process exists
			if sigErr := proc.Signal(syscall.Signal(0)); sigErr == nil {
				if m.audit != nil {
					_ = m.audit.LogInfo("recovery_worker_alive", "worker_execution", exec.ID,
						fmt.Sprintf("worker still running (PID %d), monitoring", exec.PID))
				}
				// Worker is still executing: keep RUNNING status, do not re-run
				return nil
			}
		}
	}

	// Worker process is not running. It terminated during runtime downtime.
	gitClient := git.NewClient(exec.WorkspacePath)
	hasChanges, _ := gitClient.HasUncommittedChanges()
	diff, _ := gitClient.GetDiff(t.GitBaseCommit)
	hasGitChanges := hasChanges || len(diff) > 0

	// Move to RECONCILING first before finalizing
	_ = m.taskManager.UpdateStatus(t.ID, task.TaskStatusReconciling, "Recovery: Worker process not running, reconciling results")

	var workerRes *worker.WorkerResult
	if exec.ExitCode != nil {
		workerRes = &worker.WorkerResult{
			ExecutionID: exec.ID,
			ExitCode:    *exec.ExitCode,
			Stdout:      exec.Stdout,
			Stderr:      exec.Stderr,
		}
	} else if hasGitChanges {
		// Assume exit code 0 if changes exist and no error recorded
		workerRes = &worker.WorkerResult{
			ExecutionID: exec.ID,
			ExitCode:    0,
			Stdout:      exec.Stdout,
			Stderr:      exec.Stderr,
		}
	}

	if workerRes != nil && m.validator != nil {
		valResult, valErr := m.validator.Validate(ctx, workerRes, exec.WorkspacePath, t.GitBaseCommit)
		if valErr == nil && valResult != nil && valResult.Passed {
			t.ValidationResult = valResult.Summary
			t.Result = "Successfully recovered and validated"
			finalCommit, _ := gitClient.GetCurrentCommit()
			t.GitFinalCommit = finalCommit
			_ = m.taskManager.Update(t)
			_ = m.taskManager.UpdateStatus(t.ID, task.TaskStatusSucceeded, "Recovery: Validation passed")
			if m.audit != nil {
				_ = m.audit.LogInfo("recovery_succeeded", "task", t.ID, "Interrupted task successfully recovered and verified")
			}
			return nil
		} else if workerRes.ExitCode != 0 {
			_ = m.taskManager.UpdateStatus(t.ID, task.TaskStatusFailed, "Recovery: Worker process exited with failure")
			return nil
		}
	}

	// If cannot determine state reliably, mark as UNKNOWN (do not restart blindly!)
	_ = m.taskManager.UpdateStatus(t.ID, task.TaskStatusUnknown, "Recovery: Cannot determine execution outcome, marked UNKNOWN")
	if m.audit != nil {
		_ = m.audit.LogWarn("recovery_unknown", "task", t.ID, "Execution state indeterminate, marked UNKNOWN for manual review")
	}

	return nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (m *Manager) ReconcileTask(ctx context.Context, taskID string) error {
	t, err := m.taskManager.Get(taskID)
	if err != nil {
		return err
	}
	return m.recoverTask(ctx, t)
}
