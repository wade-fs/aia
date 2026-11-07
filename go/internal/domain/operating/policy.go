package operating

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

// WorkMode defines the collaborative and autonomy nature of work.
type WorkMode string

const (
	// WorkModeAutonomous represents independent autonomous operation (e.g. Investment, Monitoring).
	WorkModeAutonomous WorkMode = "AUTONOMOUS"
	// WorkModeCollaborative represents human-in-the-loop collaborative operation (e.g. Writing, Coding).
	WorkModeCollaborative WorkMode = "COLLABORATIVE"
	// WorkModeDependencyDriven represents downstream work waiting on upstream artifacts (e.g. Illustration, Video).
	WorkModeDependencyDriven WorkMode = "DEPENDENCY_DRIVEN"
)

// TriggerType defines how an execution or evaluation is initiated.
type TriggerType string

const (
	TriggerTimeBased      TriggerType = "TIME_BASED"
	TriggerEventBased     TriggerType = "EVENT_BASED"
	TriggerConditionBased TriggerType = "CONDITION_BASED"
	TriggerManual         TriggerType = "MANUAL"
)

// ExecutionForm defines the execution form factor in runtime.
type ExecutionForm string

const (
	ExecutionInteractive ExecutionForm = "INTERACTIVE"
	ExecutionBackground  ExecutionForm = "BACKGROUND"
	ExecutionBatch       ExecutionForm = "BATCH"
)

// OperatingPolicy groups the multidimensional dimensions governing an Agent or Task execution.
type OperatingPolicy struct {
	WorkMode      WorkMode      `json:"work_mode"`
	TriggerPolicy TriggerType   `json:"trigger_policy"`
	ExecutionForm ExecutionForm `json:"execution_form"`
	ResourceQuota ResourceQuota `json:"resource_quota"`
}

// ResourceQuota defines hierarchical resource caps.
type ResourceQuota struct {
	MaxTokens        int64 `json:"max_tokens,omitempty"`
	MaxInputTokens   int64 `json:"max_input_tokens,omitempty"`
	MaxOutputTokens  int64 `json:"max_output_tokens,omitempty"`
	MaxAPICalls      int   `json:"max_api_calls,omitempty"`
	MaxDurationSec   int   `json:"max_duration_sec,omitempty"`
	ConcurrencyLimit int   `json:"concurrency_limit,omitempty"`
}

// SessionStatus represents the lifecycle of a discrete work session.
type SessionStatus string

const (
	SessionActive    SessionStatus = "ACTIVE"
	SessionPaused    SessionStatus = "PAUSED"
	SessionCompleted SessionStatus = "COMPLETED"
	SessionFailed    SessionStatus = "FAILED"
)

// Session represents a discrete, session-based unit of execution within a Task.
type Session struct {
	ID             domain.ID     `json:"id"`
	TaskID         domain.ID     `json:"task_id"`
	ProjectID      domain.ID     `json:"project_id"`
	AgentID        domain.ID     `json:"agent_id"`
	Stage          string        `json:"stage"` // e.g. "Requirements", "Specification", "Implementation"
	Status         SessionStatus `json:"status"`
	InputContext   string        `json:"input_context,omitempty"`
	OutputSummary  string        `json:"output_summary,omitempty"`
	TokensConsumed int64         `json:"tokens_consumed"`
	StartedAt      time.Time     `json:"started_at"`
	EndedAt        *time.Time    `json:"ended_at,omitempty"`
}

// Checkpoint captures a resilient recovery snapshot during long-running or multi-stage tasks.
type Checkpoint struct {
	ID                 domain.ID         `json:"id"`
	SessionID          domain.ID         `json:"session_id"`
	TaskID             domain.ID         `json:"task_id"`
	ProjectID          domain.ID         `json:"project_id"`
	Stage              string            `json:"stage"`
	CompletedSteps     []string          `json:"completed_steps"`
	PendingSteps       []string          `json:"pending_steps"`
	ArtifactVersions   map[string]string `json:"artifact_versions,omitempty"` // artifact_name -> version/hash
	ResumeInstructions string            `json:"resume_instructions,omitempty"`
	SnapshotData       string            `json:"snapshot_data,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
}
