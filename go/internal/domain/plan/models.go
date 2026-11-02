package plan

import (
	"errors"
	"time"
)

var (
	ErrPlanNotFound      = errors.New("plan not found")
	ErrInvalidPlan       = errors.New("invalid plan")
	ErrPlanVersionExists = errors.New("plan version already exists")
)

type PlanStatus string

const (
	StatusDraft      PlanStatus = "DRAFT"
	StatusProposed   PlanStatus = "PROPOSED"
	StatusApproved   PlanStatus = "APPROVED"
	StatusActive     PlanStatus = "ACTIVE"
	StatusSuperseded PlanStatus = "SUPERSEDED"
	StatusCompleted  PlanStatus = "COMPLETED"
	StatusFailed     PlanStatus = "FAILED"
)

type WorkItemStatus string

const (
	WorkItemPending    WorkItemStatus = "PENDING"
	WorkItemReady      WorkItemStatus = "READY"
	WorkItemInProgress WorkItemStatus = "IN_PROGRESS"
	WorkItemDone       WorkItemStatus = "DONE"
	WorkItemFailed     WorkItemStatus = "FAILED"
	WorkItemBlocked    WorkItemStatus = "BLOCKED"
)

// WorkItemDef represents a concrete unit of work generated within a Plan (AIA-22 §4.7, §6.4).
type WorkItemDef struct {
	ID                 string         `json:"id"`
	Title              string         `json:"title"`
	Description        string         `json:"description,omitempty"`
	Stage              string         `json:"stage,omitempty"`
	AssignedAgent      string         `json:"assigned_agent,omitempty"`
	Dependencies       []string       `json:"dependencies,omitempty"`
	AcceptanceCriteria []string       `json:"acceptance_criteria,omitempty"`
	Status             WorkItemStatus `json:"status"`
	LinkedTaskID       string         `json:"linked_task_id,omitempty"`
}

// Plan represents a versioned, verifiable execution plan for a problem (AIA-22 §6.4).
type Plan struct {
	ID                 string        `json:"id"`
	ProblemID          string        `json:"problem_id"`
	Version            int           `json:"version"`
	Objective          string        `json:"objective"`
	Strategy           string        `json:"strategy"`
	Status             PlanStatus    `json:"status"`
	WorkItems          []WorkItemDef `json:"work_items,omitempty"`
	Dependencies       []string      `json:"dependencies,omitempty"`
	AcceptanceCriteria []string      `json:"acceptance_criteria,omitempty"`
	SupersedesPlanID   string        `json:"supersedes_plan_id,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// PlanChangeRecord tracks formal modifications to a plan across versions (AIA-22 §6.5).
type PlanChangeRecord struct {
	ID                    string    `json:"id"`
	PlanID                string    `json:"plan_id"`
	FromVersion           int       `json:"from_version"`
	ToVersion             int       `json:"to_version"`
	Reason                string    `json:"reason"`
	EvidenceRefs          []string  `json:"evidence_refs,omitempty"`
	AffectedWorkItems     []string  `json:"affected_work_items,omitempty"`
	CostRiskImpact        string    `json:"cost_risk_impact,omitempty"`
	RequiresHumanApproval bool      `json:"requires_human_approval"`
	ApprovedBy            string    `json:"approved_by,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

// Validate validates the core Plan entity.
func (p *Plan) Validate() error {
	if p.ID == "" {
		return errors.New("plan ID cannot be empty")
	}
	if p.ProblemID == "" {
		return errors.New("problem ID cannot be empty")
	}
	if p.Objective == "" {
		return errors.New("plan objective cannot be empty")
	}
	if p.Status == "" {
		p.Status = StatusDraft
	}
	return nil
}
