package task

import (
	"errors"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
)

type TaskStatus string

const (
	StatusCreated           TaskStatus = "CREATED"
	StatusReady             TaskStatus = "READY"
	StatusRunning           TaskStatus = "RUNNING"
	StatusWaitingApproval   TaskStatus = "WAITING_APPROVAL"
	StatusWaitingDependency TaskStatus = "WAITING_DEPENDENCY"
	StatusCompleted         TaskStatus = "COMPLETED"
	StatusFailed            TaskStatus = "FAILED"
	StatusCancelled         TaskStatus = "CANCELLED"
	StatusBlocked           TaskStatus = "BLOCKED"
	StatusWaitingAuth       TaskStatus = "WAITING_FOR_AUTH"
)

var (
	ErrInvalidStateTransition = errors.New("invalid task status transition")
	ErrDependencyCycle        = errors.New("cyclic dependency detected")
	ErrTaskTerminalState      = errors.New("task is already in a terminal state")
)

type Task struct {
	ID           domain.ID        `json:"id"`
	ProjectID    domain.ID        `json:"project_id"`
	ParentTaskID domain.ID        `json:"parent_task_id"`
	AgentID      domain.ID        `json:"agent_id"`
	Title        string           `json:"title"`
	Description  string           `json:"description"`
	Status       TaskStatus       `json:"status"`
	Priority     int              `json:"priority"`
	Risk         domain.RiskLevel `json:"risk"`
	Acceptance   []string         `json:"acceptance"`
	RetryCount   int              `json:"retry_count"`
	MaxRetries   int              `json:"max_retries"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	StartedAt    *time.Time       `json:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at"`
}

func (t *Task) IsTerminal() bool {
	return t.Status == StatusCompleted || t.Status == StatusFailed || t.Status == StatusCancelled
}

func ValidateTransition(from, to TaskStatus) error {
	if from == to {
		return nil
	}
	switch from {
	case StatusCreated:
		if to == StatusReady || to == StatusWaitingApproval || to == StatusWaitingDependency || to == StatusCancelled {
			return nil
		}
	case StatusWaitingApproval:
		if to == StatusReady || to == StatusWaitingDependency || to == StatusCancelled || to == StatusBlocked {
			return nil
		}
	case StatusWaitingDependency:
		if to == StatusReady || to == StatusWaitingApproval || to == StatusCancelled || to == StatusBlocked {
			return nil
		}
	case StatusReady:
		if to == StatusRunning || to == StatusCancelled || to == StatusBlocked || to == StatusWaitingApproval {
			return nil
		}
	case StatusRunning:
		if to == StatusCompleted || to == StatusFailed || to == StatusCancelled || to == StatusWaitingApproval || to == StatusBlocked || to == StatusReady || to == StatusWaitingAuth {
			return nil
		}
	case StatusWaitingAuth:
		if to == StatusReady || to == StatusCancelled {
			return nil
		}
	case StatusBlocked:
		if to == StatusReady || to == StatusFailed || to == StatusCancelled {
			return nil
		}
	case StatusFailed:
		// Retry allows FAILED -> READY
		if to == StatusReady {
			return nil
		}
	case StatusCompleted, StatusCancelled:
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrTaskTerminalState, from, to)
	}

	return fmt.Errorf("%w: from %s to %s", ErrInvalidStateTransition, from, to)
}
