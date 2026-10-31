package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/agent-runtime/internal/audit"
	"github.com/wade/aia/agent-runtime/internal/storage"
	"github.com/wade/aia/agent-runtime/internal/task"
)

type Manager struct {
	worker  Worker
	storage *storage.DB
	audit   *audit.Logger
	model   string
	timeout time.Duration
}

func NewManager(w Worker, db *storage.DB, logger *audit.Logger, model string, timeout time.Duration) *Manager {
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	return &Manager{
		worker:  w,
		storage: db,
		audit:   logger,
		model:   model,
		timeout: timeout,
	}
}

func (m *Manager) Run(ctx context.Context, t *task.Task, workspacePath string) (*WorkerResult, error) {
	execID := uuid.New().String()
	startTime := time.Now()

	// 1. Save execution intent before starting worker (status=STARTING)
	execRecord := &storage.WorkerExecutionRecord{
		ID:            execID,
		TaskID:        t.ID,
		Status:        "STARTING",
		WorkspacePath: workspacePath,
		StartedAt:     &startTime,
	}
	if err := m.storage.SaveWorkerExecution(ctx, execRecord); err != nil {
		return nil, fmt.Errorf("failed to save initial worker execution: %w", err)
	}

	if m.audit != nil {
		_ = m.audit.LogInfo("worker_start_requested", "worker_execution", execID,
			fmt.Sprintf("task_id=%s workspace=%s", t.ID, workspacePath))
	}

	// 2. Prepare request with OnStart callback to update PID & RUNNING status
	req := WorkerRequest{
		TaskID:        t.ID,
		ExecutionID:   execID,
		WorkspacePath: workspacePath,
		Description:   t.Description,
		Model:         m.model,
		Timeout:       m.timeout,
		OnStart: func(pid int) {
			execRecord.Status = "RUNNING"
			execRecord.PID = pid
			_ = m.storage.UpdateWorkerExecution(context.Background(), execRecord)
			if m.audit != nil {
				_ = m.audit.LogInfo("worker_running", "worker_execution", execID, fmt.Sprintf("pid=%d", pid))
			}
		},
	}

	// 3. Execute worker
	result, execErr := m.worker.Execute(ctx, req)

	// 4. Update completion status in DB
	endTime := time.Now()
	execRecord.CompletedAt = &endTime

	if result != nil {
		execRecord.ExitCode = &result.ExitCode
		execRecord.Stdout = result.Stdout
		execRecord.Stderr = result.Stderr
	}

	if execErr != nil || (result != nil && result.ExitCode != 0) {
		execRecord.Status = "FAILED"
	} else {
		execRecord.Status = "COMPLETED"
	}

	_ = m.storage.UpdateWorkerExecution(context.Background(), execRecord)

	exitCodeVal := 0
	if execRecord.ExitCode != nil {
		exitCodeVal = *execRecord.ExitCode
	}
	if m.audit != nil {
		_ = m.audit.LogInfo("worker_completed", "worker_execution", execID,
			fmt.Sprintf("status=%s exit_code=%d", execRecord.Status, exitCodeVal))
	}

	return result, execErr
}

func (m *Manager) HealthCheck(ctx context.Context) error {
	if m.worker == nil {
		return fmt.Errorf("no worker adapter configured")
	}
	return m.worker.HealthCheck(ctx)
}

func (m *Manager) GetStatus(ctx context.Context, executionID string) (*WorkerStatus, error) {
	if m.worker == nil {
		return nil, fmt.Errorf("no worker adapter configured")
	}
	return m.worker.GetStatus(ctx, executionID)
}

func (m *Manager) Cancel(ctx context.Context, executionID string) error {
	if m.worker == nil {
		return fmt.Errorf("no worker adapter configured")
	}
	return m.worker.Cancel(ctx, executionID)
}
