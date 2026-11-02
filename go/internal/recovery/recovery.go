package recovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/provider"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/validation"
	"github.com/wade/aia/internal/workspace"
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
	validator     validation.Validator
	wsMgr         *workspace.Manager
	artRepo       *repository.ArtifactRepository
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

func (m *Manager) SetValidator(v validation.Validator) {
	m.validator = v
}

func (m *Manager) SetWorkspaceManager(ws *workspace.Manager) {
	m.wsMgr = ws
}

func (m *Manager) SetArtifactRepository(repo *repository.ArtifactRepository) {
	m.artRepo = repo
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
		stillActive := false
		if err == nil {
			for _, ex := range execs {
				// Case A: Execution already fully completed before crash
				if ex.Status == execution.StatusCompleted {
					alreadySucceeded = true
					break
				}

				// Case B: Execution finished with SUCCESS, but crashed before/during validation
				if ex.ResultStatus == "SUCCESS" && ex.Status != execution.StatusCompleted {
					if m.validator != nil {
						valReq := validation.ValidationRequest{
							Task:      t,
							Execution: ex,
							Result: provider.ProviderResult{
								Status:         ex.ResultStatus,
								ConversationID: ex.ConversationID,
								Response:       "Recovered execution result",
							},
							Workspace: ex.Workspace,
						}
						valRes, valErr := m.validator.Validate(ctx, valReq)
						if valErr == nil && valRes != nil && valRes.Accepted {
							ex.Status = execution.StatusCompleted
							now := time.Now().UTC()
							if ex.EndedAt == nil {
								ex.EndedAt = &now
							}
							_ = m.executionRepo.Update(ctx, &ex)
							if m.wsMgr != nil && m.artRepo != nil {
								artifacts, _ := m.wsMgr.CollectArtifacts(t.ID, ex.ID, ex.Workspace)
								for _, art := range artifacts {
									_ = m.artRepo.Create(ctx, &art)
								}
							}
							if m.auditRepo != nil {
								_ = m.auditRepo.Append(ctx, &repository.AuditLogEntry{
									ID:           uuid.New().String(),
									ActorType:    "SYSTEM",
									ActorID:      "RECOVERY",
									Action:       "RECOVERY_VALIDATION_PASSED",
									ResourceType: "TASK",
									ResourceID:   t.ID,
									Outcome:      "SUCCESS",
								})
							}
							alreadySucceeded = true
							break
						} else {
							ex.Status = execution.StatusFailed
							if valRes != nil && len(valRes.Findings) > 0 {
								ex.ErrorMessage = fmt.Sprintf("Validation failed during recovery: %s", valRes.Findings[0].Description)
							} else {
								ex.ErrorMessage = "Validation failed during recovery"
							}
							now := time.Now().UTC()
							if ex.EndedAt == nil {
								ex.EndedAt = &now
							}
							_ = m.executionRepo.Update(ctx, &ex)
							if m.auditRepo != nil {
								_ = m.auditRepo.Append(ctx, &repository.AuditLogEntry{
									ID:           uuid.New().String(),
									ActorType:    "SYSTEM",
									ActorID:      "RECOVERY",
									Action:       "RECOVERY_VALIDATION_FAILED",
									ResourceType: "TASK",
									ResourceID:   t.ID,
									Outcome:      "FAILURE",
								})
							}
							continue
						}
					}
				}

				// Case C: Execution interrupted before completion
				if ex.Status == execution.StatusRunning || ex.Status == execution.StatusStarting {
					if ex.ProcessID > 0 && isProcessAlive(ex.ProcessID) && ex.ProcessID != os.Getpid() {
						slog.Info("execution is actively running in background daemon process, skipping interruption", "task_id", t.ID, "execution_id", ex.ID, "pid", ex.ProcessID)
						stillActive = true
						break
					}

					ex.Status = execution.StatusInterrupted
					ex.ErrorMessage = "Process interrupted by unexpected runtime shutdown"
					now := time.Now().UTC()
					ex.EndedAt = &now
					_ = m.executionRepo.Update(ctx, &ex)
					summary.InterruptedExecutions++
				}
			}
		}

		if stillActive {
			// Task has an active execution in a live daemon process. Do not reconcile or retry.
			continue
		}

		if alreadySucceeded {
			_ = m.taskRepo.UpdateStatus(ctx, t.ID, task.StatusCompleted)
			slog.Info("recovered completed task from validated execution", "task_id", t.ID)
			summary.RecoveredTasks++
			continue
		}

		// Reconcile task state:
		// Check whether task has external side effects before blind retry
		hasSideEffects := t.Risk == domain.RiskHigh || t.Risk == domain.RiskCritical
		if hasSideEffects {
			_ = m.taskRepo.UpdateStatus(ctx, t.ID, task.StatusWaitingApproval)
			if m.auditRepo != nil {
				_ = m.auditRepo.Append(ctx, &repository.AuditLogEntry{
					ID:           uuid.New().String(),
					ActorType:    "SYSTEM",
					ActorID:      "RECOVERY",
					Action:       "RECOVERY_SIDE_EFFECTS_BLOCKED",
					ResourceType: "TASK",
					ResourceID:   t.ID,
					Outcome:      "WAITING_APPROVAL",
				})
			}
			slog.Warn("task may have caused external side effects before crash; marked WAITING_APPROVAL to prevent duplicate side effects", "task_id", t.ID)
		} else if t.RetryCount < t.MaxRetries {
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

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

