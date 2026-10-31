package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wade/aia/internal/config"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/profile"
	"github.com/wade/aia/internal/domain/task"
	agentService "github.com/wade/aia/internal/service/agent"
	execService "github.com/wade/aia/internal/service/execution"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type Scheduler struct {
	taskSvc  *taskService.Service
	execSvc  *execService.Service
	agentSvc *agentService.Service
	cfg      config.RuntimeConfig
	mu       sync.Mutex
}

func NewScheduler(
	taskSvc *taskService.Service,
	execSvc *execService.Service,
	agentSvc *agentService.Service,
	cfg config.RuntimeConfig,
) *Scheduler {
	return &Scheduler{
		taskSvc:  taskSvc,
		execSvc:  execSvc,
		agentSvc: agentSvc,
		cfg:      cfg,
	}
}

func (s *Scheduler) RunTask(ctx context.Context, taskID domain.ID) error {
	t, err := s.taskSvc.Get(ctx, taskID)
	if err != nil {
		return err
	}

	if t.Status != task.StatusReady {
		return fmt.Errorf("task %s is not in READY state (current: %s)", taskID, t.Status)
	}

	// Verify agent
	agentID := t.AgentID
	if agentID == "" {
		agentID = "coding-agent" // default domain agent
	}

	ag, err := s.agentSvc.Get(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to load agent %s: %w", agentID, err)
	}
	if !ag.Enabled {
		return fmt.Errorf("agent %s is disabled", agentID)
	}

	prof := profile.ExecutionProfile{
		ID:      "default-profile",
		Name:    "Standard Profile",
		Timeout: 30 * time.Minute,
		Sandbox: true,
	}

	slog.Info("scheduling task execution", "task_id", t.ID, "agent", ag.Name)
	_, err = s.execSvc.Execute(ctx, t, ag, prof, "")
	return err
}

func (s *Scheduler) ProcessReadyTasks(ctx context.Context) (int, error) {
	readyTasks, err := s.taskSvc.List(ctx, repository.TaskFilter{Status: task.StatusReady})
	if err != nil {
		return 0, err
	}

	executed := 0
	for _, t := range readyTasks {
		select {
		case <-ctx.Done():
			return executed, ctx.Err()
		default:
			if err := s.RunTask(ctx, t.ID); err == nil {
				executed++
			}
		}
	}
	return executed, nil
}
