package plan

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrPlanNotFound               = errors.New("plan not found")
	ErrInvalidPlan                = errors.New("invalid plan")
	ErrPlanVersionExists          = errors.New("plan version already exists")
	ErrNoAlternatives             = errors.New("at least one solution alternative must be provided")
	ErrSelectedAlternativeNotFound = errors.New("selected alternative ID not found in proposed alternatives")
	ErrCircularDependency         = errors.New("circular dependency detected in plan work items")
	ErrMissingAcceptanceCriteria  = errors.New("work item must have at least one acceptance criterion")
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

// SolutionAlternative describes a candidate resolution pathway for a complex problem (AIA-22 §4.6).
type SolutionAlternative struct {
	ID                   string   `json:"id"`
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	ExpectedBenefits     []string `json:"expected_benefits,omitempty"`
	EstimatedCost        string   `json:"estimated_cost,omitempty"`
	EstimatedDuration    string   `json:"estimated_duration,omitempty"`
	TechnicalFeasibility string   `json:"technical_feasibility,omitempty"` // "HIGH", "MEDIUM", "LOW"
	Risks                []string `json:"risks,omitempty"`
	Reversibility        string   `json:"reversibility,omitempty"` // "HIGH", "MEDIUM", "LOW", "IRREVERSIBLE"
	Prerequisites        []string `json:"prerequisites,omitempty"`
	VerificationMethod   string   `json:"verification_method,omitempty"`
	Selected             bool     `json:"selected"`
	SelectionRationale   string   `json:"selection_rationale,omitempty"`
}

// SolutionExplorationResult captures multi-alternative exploration and trade-off comparison (AIA-22 §4.6).
type SolutionExplorationResult struct {
	ProblemID             string                `json:"problem_id"`
	ModelVersion          int                   `json:"model_version"`
	Alternatives          []SolutionAlternative `json:"alternatives"`
	SelectedAlternativeID string                `json:"selected_alternative_id"`
	OverallRationale      string                `json:"overall_rationale"`
	RequiresHumanApproval bool                  `json:"requires_human_approval"`
}

// ValidateSolutionExploration validates that solution exploration satisfies diversity and rationale requirements (AIA-22 §4.6).
func ValidateSolutionExploration(exp *SolutionExplorationResult) error {
	if exp == nil {
		return errors.New("solution exploration result is nil")
	}
	if len(exp.Alternatives) == 0 {
		return ErrNoAlternatives
	}
	if strings.TrimSpace(exp.SelectedAlternativeID) == "" {
		return errors.New("selected alternative ID cannot be empty")
	}
	if strings.TrimSpace(exp.OverallRationale) == "" {
		return errors.New("overall selection rationale cannot be empty")
	}

	foundSelected := false
	for _, alt := range exp.Alternatives {
		if strings.TrimSpace(alt.ID) == "" {
			return errors.New("solution alternative ID cannot be empty")
		}
		if strings.TrimSpace(alt.Title) == "" {
			return fmt.Errorf("solution alternative %q must have a title", alt.ID)
		}
		if alt.ID == exp.SelectedAlternativeID {
			foundSelected = true
		}
		// If an alternative is irreversible, ensure human approval requirement is flagged (AIA-22 §9.2)
		if alt.Reversibility == "IRREVERSIBLE" && alt.ID == exp.SelectedAlternativeID {
			exp.RequiresHumanApproval = true
		}
	}

	if !foundSelected {
		return fmt.Errorf("%w: %q", ErrSelectedAlternativeNotFound, exp.SelectedAlternativeID)
	}

	return nil
}

// WorkItemDef represents a concrete unit of work generated within a Plan (AIA-22 §4.7, §6.4).
type WorkItemDef struct {
	ID                 string         `json:"id"`
	Title              string         `json:"title"`
	Description        string         `json:"description,omitempty"`
	Stage              string         `json:"stage,omitempty"`
	AssignedAgent      string         `json:"assigned_agent,omitempty"`
	Dependencies       []string       `json:"dependencies,omitempty"`
	AcceptanceCriteria []string       `json:"acceptance_criteria"`
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

// Validate validates the core Plan entity and checks work item DAG consistency (AIA-22 §4.7).
func (p *Plan) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("plan ID cannot be empty")
	}
	if strings.TrimSpace(p.ProblemID) == "" {
		return errors.New("problem ID cannot be empty")
	}
	if strings.TrimSpace(p.Objective) == "" {
		return errors.New("plan objective cannot be empty")
	}
	if p.Status == "" {
		p.Status = StatusDraft
	}

	// Validate WorkItems
	for i, wi := range p.WorkItems {
		if strings.TrimSpace(wi.ID) == "" {
			return fmt.Errorf("work item [%d] ID cannot be empty", i)
		}
		if strings.TrimSpace(wi.Title) == "" {
			return fmt.Errorf("work item %q title cannot be empty", wi.ID)
		}
		if len(wi.AcceptanceCriteria) == 0 {
			return fmt.Errorf("%w: work item %q must define acceptance criteria", ErrMissingAcceptanceCriteria, wi.ID)
		}
		if wi.Status == "" {
			p.WorkItems[i].Status = WorkItemPending
		}
	}

	// Validate WorkItem dependency graph (no dangling refs, no cycles)
	if err := checkCircularDependencies(p.WorkItems); err != nil {
		return err
	}

	return nil
}

func checkCircularDependencies(items []WorkItemDef) error {
	itemMap := make(map[string]bool)
	adj := make(map[string][]string)
	for _, it := range items {
		itemMap[it.ID] = true
		adj[it.ID] = it.Dependencies
	}

	for _, it := range items {
		for _, dep := range it.Dependencies {
			if !itemMap[dep] {
				return fmt.Errorf("work item %q depends on non-existent work item %q", it.ID, dep)
			}
			if dep == it.ID {
				return fmt.Errorf("work item %q cannot depend on itself", it.ID)
			}
		}
	}

	// 0: unvisited, 1: visiting (in stack), 2: visited
	state := make(map[string]int)
	var dfs func(u string) error
	dfs = func(u string) error {
		state[u] = 1
		for _, v := range adj[u] {
			if state[v] == 1 {
				return fmt.Errorf("%w: %q <-> %q", ErrCircularDependency, u, v)
			}
			if state[v] == 0 {
				if err := dfs(v); err != nil {
					return err
				}
			}
		}
		state[u] = 2
		return nil
	}

	for _, it := range items {
		if state[it.ID] == 0 {
			if err := dfs(it.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
