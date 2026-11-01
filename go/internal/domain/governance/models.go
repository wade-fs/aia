package governance

import (
	"encoding/json"
	"strings"
	"time"
)

type ApprovalStatus string

const (
	ApprovalPending   ApprovalStatus = "PENDING"
	ApprovalApproved  ApprovalStatus = "APPROVED"
	ApprovalRejected  ApprovalStatus = "REJECTED"
	ApprovalExpired   ApprovalStatus = "EXPIRED"
	ApprovalCancelled ApprovalStatus = "CANCELLED"
)

type RiskTier string

const (
	RiskTierLow      RiskTier = "LOW"
	RiskTierMedium   RiskTier = "MEDIUM"
	RiskTierHigh     RiskTier = "HIGH"
	RiskTierCritical RiskTier = "CRITICAL"
)

type TimeoutPolicy string

const (
	TimeoutPolicyReject   TimeoutPolicy = "REJECT"
	TimeoutPolicyEscalate TimeoutPolicy = "ESCALATE"
)

type GovernanceAction string

const (
	GovernanceActionAllowed          GovernanceAction = "ALLOWED"
	GovernanceActionRequiredApproval GovernanceAction = "REQUIRED_APPROVAL"
	GovernanceActionBlocked          GovernanceAction = "BLOCKED"
	GovernanceActionEmergencyStop    GovernanceAction = "EMERGENCY_STOP"
	GovernanceActionResumed          GovernanceAction = "RESUMED"
)

// Approval represents an operational gate requiring human or supervisor confirmation.
type Approval struct {
	ID                  string          `json:"id"`
	RequesterEntity     string          `json:"requester_entity"`
	OperationType       string          `json:"operation_type"`
	OperationDetailJSON string          `json:"operation_detail_json"`
	Status              ApprovalStatus  `json:"status"`
	RiskTier            RiskTier        `json:"risk_tier"`
	TimeoutAt           *time.Time      `json:"timeout_at,omitempty"`
	TimeoutPolicy       TimeoutPolicy   `json:"timeout_policy"`
	TaskID              *string         `json:"task_id,omitempty"`
	DecisionID          *string         `json:"decision_id,omitempty"`
	ReviewerID          *string         `json:"reviewer_id,omitempty"`
	ReviewerNote        *string         `json:"reviewer_note,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	ReviewedAt          *time.Time      `json:"reviewed_at,omitempty"`
	ExpiresAt           *time.Time      `json:"expires_at,omitempty"`
}

// Operation types evaluated by the Policy Engine and bound to approvals.
const (
	OperationTaskExecution        = "TASK_EXECUTION"
	OperationFileRead             = "FILE_READ"
	OperationFileWrite            = "FILE_WRITE"
	OperationFileDelete           = "FILE_DELETE"
	OperationFileMutate           = "FILE_MUTATE"
	OperationSystemCommand        = "SYSTEM_COMMAND"
	OperationFinancialTransaction = "FINANCIAL_TRANSACTION"
)

// DispatchState is the global Runtime dispatch gate state (AIA-17 §11.5).
type DispatchState string

const (
	DispatchRunning       DispatchState = "RUNNING"
	DispatchEmergencyStop DispatchState = "EMERGENCY_STOP"
)

// RuntimeControl is the persisted, cross-process Runtime control record.
// Exactly one row (id = "global") exists.
type RuntimeControl struct {
	ID            string        `json:"id"`
	DispatchState DispatchState `json:"dispatch_state"`
	Reason        string        `json:"reason"`
	TriggeredBy   string        `json:"triggered_by"`
	TriggeredAt   *time.Time    `json:"triggered_at,omitempty"`
	ResumedBy     string        `json:"resumed_by"`
	ResumedAt     *time.Time    `json:"resumed_at,omitempty"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// IsEmergencyStopped reports whether dispatch is frozen.
func (c *RuntimeControl) IsEmergencyStopped() bool {
	return c != nil && c.DispatchState == DispatchEmergencyStop
}

// Actor identifies who performs a governance operation.
// Type is one of HUMAN, OWNER, USER, ADMIN, SYSTEM or AGENT.
type Actor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// IsHuman reports whether the actor is a human operator allowed to perform
// privileged governance operations (approve, resume). Agents and the system
// itself are never treated as human (AIA-17 §2.2, Plan §0.3).
func (a Actor) IsHuman() bool {
	switch strings.ToUpper(strings.TrimSpace(a.Type)) {
	case "HUMAN", "OWNER", "USER", "ADMIN":
	default:
		return false
	}
	return !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(a.ID)), "AGENT:")
}

// DetailMap returns the operation detail unmarshaled as a map.
func (a *Approval) DetailMap() map[string]any {
	var m map[string]any
	if a.OperationDetailJSON != "" {
		_ = json.Unmarshal([]byte(a.OperationDetailJSON), &m)
	}
	if m == nil {
		m = make(map[string]any)
	}
	return m
}
