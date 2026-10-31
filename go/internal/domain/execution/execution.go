package execution

import (
	"encoding/json"
	"time"

	"github.com/wade/aia/internal/domain"
)

type ExecutionStatus string

const (
	StatusPending     ExecutionStatus = "PENDING"
	StatusStarting    ExecutionStatus = "STARTING"
	StatusRunning     ExecutionStatus = "RUNNING"
	StatusCompleted   ExecutionStatus = "COMPLETED"
	StatusFailed      ExecutionStatus = "FAILED"
	StatusCancelled   ExecutionStatus = "CANCELLED"
	StatusTimedOut    ExecutionStatus = "TIMED_OUT"
	StatusInterrupted ExecutionStatus = "INTERRUPTED"
)

type Execution struct {
	ID             domain.ID       `json:"id"`
	TaskID         domain.ID       `json:"task_id"`
	AgentID        domain.ID       `json:"agent_id"`
	Provider       string          `json:"provider"`
	ProfileID      domain.ID       `json:"profile_id"`
	ProfileVersion int             `json:"profile_version"`
	Workspace      string          `json:"workspace"`
	ConversationID string          `json:"conversation_id"`
	ProcessID      int             `json:"process_id"`
	Status         ExecutionStatus `json:"status"`
	ExitCode       *int            `json:"exit_code"`
	ResultStatus   string          `json:"result_status"`
	ErrorCode      string          `json:"error_code"`
	ErrorMessage   string          `json:"error_message"`
	StartedAt      *time.Time      `json:"started_at"`
	EndedAt        *time.Time      `json:"ended_at"`
	CreatedAt      time.Time       `json:"created_at"`
}

type ExecutionEvent struct {
	ID          domain.ID       `json:"id"`
	ExecutionID domain.ID       `json:"execution_id"`
	Sequence    uint64          `json:"sequence"`
	EventType   string          `json:"event_type"`
	Source      string          `json:"source"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
}
