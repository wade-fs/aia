package clarification

import (
	"time"
)

// ClarificationStatus tracks the formal resolution lifecycle of a request (AIA-21 §5.2).
type ClarificationStatus string

const (
	StatusOpen      ClarificationStatus = "OPEN"      // Awaiting human answer
	StatusAnswered  ClarificationStatus = "ANSWERED"  // Human has provided answer
	StatusDeferred  ClarificationStatus = "DEFERRED"  // Human or agent deferred question
	StatusCancelled ClarificationStatus = "CANCELLED" // Cancelled due to requirements change
	StatusExpired   ClarificationStatus = "EXPIRED"   // Timed out under timeout policy
)

// ClarificationPriority ranks the decision criticality of the inquiry (AIA-21 §5.3).
type ClarificationPriority int

const (
	PriorityP0 ClarificationPriority = 0 // Safety, legal, irreversible decisions (must confirm)
	PriorityP1 ClarificationPriority = 1 // Main direction and overall architecture (priority ask)
	PriorityP2 ClarificationPriority = 2 // Local layout and secondary parameters (can defer)
	PriorityP3 ClarificationPriority = 3 // Non-blocking / informational
)

// ClarificationRequest represents a structured inquiry to resolve requirements unknowns (AIA-21 §5.1, §5.2).
type ClarificationRequest struct {
	ID          string                `json:"id"`
	ProjectID   string                `json:"project_id"`
	TaskID      string                `json:"task_id,omitempty"`
	FieldKey    string                `json:"field_key"`
	Question    string                `json:"question"`
	Options     []string              `json:"options,omitempty"` // Recommended suggestions/options
	Reason      string                `json:"reason"`            // Why this is needed now (決策價值)
	Impact      string                `json:"impact"`            // Downstream consequences on project
	Priority    ClarificationPriority `json:"priority"`          // P0, P1, P2, P3
	Blocking    bool                  `json:"blocking"`          // Does this block the current stage?
	Status      ClarificationStatus   `json:"status"`
	Answer      string                `json:"answer,omitempty"`
	AnsweredAt  *time.Time            `json:"answered_at,omitempty"`
	ExpiresAt   *time.Time            `json:"expires_at,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// IsOpen returns true if the question is currently awaiting input.
func (c *ClarificationRequest) IsOpen() bool {
	return c.Status == StatusOpen
}

// IsResolved returns true if the question is completed.
func (c *ClarificationRequest) IsResolved() bool {
	return c.Status == StatusAnswered || c.Status == StatusCancelled
}

// PriorityString returns a readable representation of the priority.
func (c *ClarificationRequest) PriorityString() string {
	switch c.Priority {
	case PriorityP0:
		return "P0 (Critical)"
	case PriorityP1:
		return "P1 (High)"
	case PriorityP2:
		return "P2 (Medium)"
	case PriorityP3:
		return "P3 (Low)"
	default:
		return "P1 (Default)"
	}
}
