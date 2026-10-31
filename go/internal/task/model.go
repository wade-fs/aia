package task

import (
	"time"
)

type TaskStatus string

const (
	TaskStatusCreated     TaskStatus = "CREATED"
	TaskStatusReady       TaskStatus = "READY"
	TaskStatusRunning     TaskStatus = "RUNNING"
	TaskStatusSucceeded   TaskStatus = "SUCCEEDED"
	TaskStatusFailed      TaskStatus = "FAILED"
	TaskStatusCancelled   TaskStatus = "CANCELLED"
	TaskStatusUnknown     TaskStatus = "UNKNOWN"
	TaskStatusReconciling TaskStatus = "RECONCILING"
	TaskStatusNeedsReview TaskStatus = "NEEDS_REVIEW"
)

func (s TaskStatus) String() string {
	return string(s)
}

func (s TaskStatus) IsTerminal() bool {
	switch s {
	case TaskStatusSucceeded, TaskStatusFailed, TaskStatusCancelled, TaskStatusNeedsReview:
		return true
	default:
		return false
	}
}

type Task struct {
	ID                string     `json:"id"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	Status            TaskStatus `json:"status"`
	WorkerType        string     `json:"worker_type"`
	WorkspaceID       string     `json:"workspace_id"`
	CreatedAt         time.Time  `json:"created_at"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	LastUpdatedAt     time.Time  `json:"last_updated_at"`
	WorkerExecutionID string     `json:"worker_execution_id,omitempty"`
	Result            string     `json:"result,omitempty"`
	Error             string     `json:"error,omitempty"`
	GitBaseCommit     string     `json:"git_base_commit,omitempty"`
	GitFinalCommit    string     `json:"git_final_commit,omitempty"`
	ValidationResult  string     `json:"validation_result,omitempty"`
}
