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
	domainContinuity "github.com/wade/aia/internal/domain/continuity"
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
	PreservedWaitingInput int
	ExpiredClarifications int
	AbortedStuckCycles    int
	ReconciledDelegations int
}

type Manager struct {
	taskRepo       *repository.TaskRepository
	executionRepo  *repository.ExecutionRepository
	auditRepo      *repository.AuditRepository
	clarifyRepo    *repository.ClarificationRepository
	continuityRepo *repository.ContinuityRepository
	expRepo        *repository.ExperimentRepository
	validator      validation.Validator
	wsMgr          *workspace.Manager
	artRepo        *repository.ArtifactRepository
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

func (m *Manager) SetClarificationRepository(repo *repository.ClarificationRepository) {
	m.clarifyRepo = repo
}

func (m *Manager) SetContinuityRepository(repo *repository.ContinuityRepository) {
	m.continuityRepo = repo
}

func (m *Manager) SetExperimentRepository(repo *repository.ExperimentRepository) {
	m.expRepo = repo
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

	now := time.Now().UTC()

	// 3. Reconcile Expired Clarification Requests (AIA-21 §13.7)
	// CRITICAL: Expiration never automatically assumes user consent or unblocks tasks.
	if m.clarifyRepo != nil {
		expired, err := m.clarifyRepo.ListExpiredOpen(ctx, now)
		if err == nil {
			for _, exp := range expired {
				exp.Status = "EXPIRED"
				exp.UpdatedAt = now
				_ = m.clarifyRepo.Update(ctx, exp)
				summary.ExpiredClarifications++

				if m.auditRepo != nil {
					_ = m.auditRepo.Append(ctx, &repository.AuditLogEntry{
						ID:           domain.ID("audit-exp-" + uuid.New().String()[:8]),
						ActorType:    "SYSTEM",
						ActorID:      "RECOVERY",
						Action:       "CLARIFICATION_EXPIRED",
						ResourceType: "CLARIFICATION",
						ResourceID:   exp.ID,
						Outcome:      "EXPIRED",
						CreatedAt:    now,
					})
				}
			}
		}
	}

	// 4. Reconcile WAITING_FOR_INPUT tasks (AIA-21 §7.2, §7.4, Stage 4)
	waitingInputTasks, err := m.taskRepo.List(ctx, repository.TaskFilter{Status: task.StatusWaitingInput})
	if err == nil {
		for _, wit := range waitingInputTasks {
			if m.clarifyRepo != nil {
				blocking, err := m.clarifyRepo.ListBlockingOpenByProject(ctx, string(wit.ProjectID))
				if err == nil && len(blocking) == 0 {
					// All blocking clarifications resolved while offline -> resume task to READY (AIA-21 §7.4)
					_ = m.taskRepo.UpdateStatus(ctx, wit.ID, task.StatusReady)
					summary.ReadyTasksPromoted++
					slog.Info("recovered and promoted WAITING_FOR_INPUT task to READY as all blocking clarifications are resolved", "task_id", wit.ID)
					continue
				}
			}
			// Retain WAITING_FOR_INPUT: Never crash, never mark failed, never assume consent (AIA-21 §7.2)
			summary.PreservedWaitingInput++
			slog.Info("preserved WAITING_FOR_INPUT task awaiting user input across restart", "task_id", wit.ID, "project_id", wit.ProjectID)
		}
	}

	// 5. Reconcile Stuck Autonomy Cycles (AIA-10 §16.3)
	if m.continuityRepo != nil {
		cycles, err := m.continuityRepo.ListCycles(ctx, 50)
		if err == nil {
			for _, c := range cycles {
				if c.Status == domainContinuity.CycleStatusStarted ||
					c.Status == domainContinuity.CycleStatusDeliberating ||
					c.Status == domainContinuity.CycleStatusExecuting ||
					c.Status == domainContinuity.CycleStatusEvaluating {
					abortedSummary := "Interrupted by runtime restart / crash recovery"
					_ = m.continuityRepo.UpdateCycleStatus(ctx, c.ID, domainContinuity.CycleStatusAborted, abortedSummary, &now)
					summary.AbortedStuckCycles++
					slog.Warn("reconciled stuck autonomy cycle to ABORTED", "cycle_id", c.ID, "previous_status", c.Status)
				}
			}
		}
	}

	// 6. Reconcile Stuck Delegations (AIA-10 §16.3, AIA-12)
	if m.expRepo != nil && m.taskRepo != nil {
		inProgDels, err := m.expRepo.ListDelegationRequests(ctx, "", "", "IN_PROGRESS", 50)
		if err == nil {
			for _, d := range inProgDels {
				if d.SpawnedTaskID != nil {
					tk, err := m.taskRepo.Get(ctx, domain.ID(*d.SpawnedTaskID))
					if err == nil && (tk.Status == task.StatusFailed || tk.Status == task.StatusCancelled) {
						_ = m.expRepo.UpdateDelegationStatus(ctx, d.ID, "FAILED", `{"reason":"Underlying spawned task failed or cancelled"}`)
						summary.ReconciledDelegations++
						slog.Warn("reconciled stuck delegation to FAILED due to task failure", "delegation_id", d.ID, "task_id", *d.SpawnedTaskID)
					}
				}
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

