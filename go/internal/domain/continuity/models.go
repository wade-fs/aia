package continuity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// EventHistory represents an immutable event in the event_history table.
type EventHistory struct {
	ID               int64     `json:"id"`
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	EventVersion     int       `json:"event_version"`
	AggregateType    string    `json:"aggregate_type"`
	AggregateID      string    `json:"aggregate_id"`
	AggregateVersion int       `json:"aggregate_version"`
	CorrelationID    string    `json:"correlation_id"`
	CausationID      string    `json:"causation_id"`
	ActorID          string    `json:"actor_id"`
	PayloadJSON      string    `json:"payload_json"`
	PayloadHash      string    `json:"payload_hash"`
	PreviousHash     string    `json:"previous_hash"`
	OccurredAt       time.Time `json:"occurred_at"`
	RecordedAt       time.Time `json:"recorded_at"`
}

// ComputePayloadHash calculates SHA256 of the payload string.
func ComputePayloadHash(payload string) string {
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}

// ComputeEventHash computes a chain hash linking previous hash, event meta and payload hash.
func ComputeEventHash(prevHash, eventID, eventType, aggregateID string, aggregateVersion int, payloadHash string) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%d|%s", prevHash, eventID, eventType, aggregateID, aggregateVersion, payloadHash)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

type CycleTriggerType string

const (
	TriggerScheduled     CycleTriggerType = "SCHEDULED"
	TriggerEventTrigger  CycleTriggerType = "EVENT_TRIGGERED"
	TriggerManual        CycleTriggerType = "MANUAL"
	TriggerDriftDetected CycleTriggerType = "DRIFT_DETECTED"
)

type CycleStatus string

const (
	CycleStatusStarted      CycleStatus = "STARTED"
	CycleStatusDeliberating CycleStatus = "DELIBERATING"
	CycleStatusExecuting    CycleStatus = "EXECUTING"
	CycleStatusEvaluating   CycleStatus = "EVALUATING"
	CycleStatusCompleted    CycleStatus = "COMPLETED"
	CycleStatusAborted      CycleStatus = "ABORTED"
)

// AutonomyCycle represents a full autonomy closed loop instance.
type AutonomyCycle struct {
	ID                string           `json:"id"`
	CycleNumber       int64            `json:"cycle_number"`
	TriggerType       CycleTriggerType `json:"trigger_type"`
	TriggerRefID      string           `json:"trigger_ref_id,omitempty"`
	Status            CycleStatus      `json:"status"`
	ContextSnapshotID string           `json:"context_snapshot_id,omitempty"`
	Summary           string           `json:"summary,omitempty"`
	StartedAt         time.Time        `json:"started_at"`
	CompletedAt       *time.Time       `json:"completed_at,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
}

type SnapshotScope string

const (
	SnapshotScopeGlobal  SnapshotScope = "GLOBAL"
	SnapshotScopeProject SnapshotScope = "PROJECT"
)

type SnapshotType string

const (
	SnapshotTypeManual        SnapshotType = "MANUAL"
	SnapshotTypePreRestart    SnapshotType = "PRE_RESTART"
	SnapshotTypePostCycle     SnapshotType = "POST_CYCLE"
	SnapshotTypeEmergencyStop SnapshotType = "EMERGENCY_STOP"
)

// ContextSnapshot represents workspace and execution state captured at a point in time.
type ContextSnapshot struct {
	ID                 string        `json:"id"`
	Scope              SnapshotScope `json:"scope"`
	ProjectID          string        `json:"project_id,omitempty"`
	SnapshotType       SnapshotType  `json:"snapshot_type"`
	WorkspaceStateJSON string        `json:"workspace_state_json"`
	ActiveTasksJSON    string        `json:"active_tasks_json"`
	RecentDecisionsJSON string       `json:"recent_decisions_json"`
	PendingIssuesJSON  string        `json:"pending_issues_json"`
	CreatedAt          time.Time     `json:"created_at"`
}

// SanitizeWorkspaceEnv filters out sensitive environment variables like tokens, secrets, keys.
func SanitizeWorkspaceEnv(rawEnv map[string]string) map[string]string {
	sanitized := make(map[string]string, len(rawEnv))
	sensitiveKeys := []string{"KEY", "SECRET", "TOKEN", "PASSWORD", "AUTH", "CREDENTIAL"}

	for k, v := range rawEnv {
		isSensitive := false
		upper := strings.ToUpper(k)
		for _, s := range sensitiveKeys {
			if strings.Contains(upper, s) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}
