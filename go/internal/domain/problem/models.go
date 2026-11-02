package problem

import (
	"errors"
	"time"
)

var (
	ErrProblemNotFound      = errors.New("problem not found")
	ErrInvalidProblem       = errors.New("invalid problem")
	ErrInvalidModelItem     = errors.New("invalid model item")
	ErrAnalysisRecordExists = errors.New("analysis record already exists and is immutable")
)

type ProblemStatus string

const (
	StatusOpen        ProblemStatus = "OPEN"
	StatusAnalyzing   ProblemStatus = "ANALYZING"
	StatusPlanned     ProblemStatus = "PLANNED"
	StatusExecuting   ProblemStatus = "EXECUTING"
	StatusReassessing ProblemStatus = "REASSESSING"
	StatusResolved    ProblemStatus = "RESOLVED"
	StatusClosed      ProblemStatus = "CLOSED"
)

type ProblemPriority string

const (
	PriorityLow      ProblemPriority = "LOW"
	PriorityMedium   ProblemPriority = "MEDIUM"
	PriorityHigh     ProblemPriority = "HIGH"
	PriorityCritical ProblemPriority = "CRITICAL"
)

// ItemCategory defines the epistemic dimension of a model item (AIA-22 §6.2, §7.1).
type ItemCategory string

const (
	CategoryObjective       ItemCategory = "OBJECTIVE"
	CategoryConstraint      ItemCategory = "CONSTRAINT"
	CategoryKnownFact       ItemCategory = "KNOWN_FACT"
	CategoryAssumption      ItemCategory = "ASSUMPTION"
	CategoryUnknown         ItemCategory = "UNKNOWN"
	CategoryDependency      ItemCategory = "DEPENDENCY"
	CategoryRisk            ItemCategory = "RISK"
	CategorySuccessCriteria ItemCategory = "SUCCESS_CRITERIA"
)

// ItemStatus tracks the verification status of a model item (AIA-22 §6.2).
type ItemStatus string

const (
	ItemStatusActive       ItemStatus = "ACTIVE"
	ItemStatusConfirmed    ItemStatus = "CONFIRMED"
	ItemStatusContradicted ItemStatus = "CONTRADICTED"
	ItemStatusResolved     ItemStatus = "RESOLVED"
	ItemStatusRevised      ItemStatus = "REVISED"
)

// ModelItem represents an epistemic finding or assertion in the problem model (AIA-22 §6.2).
type ModelItem struct {
	ID          string       `json:"id"`
	Category    ItemCategory `json:"category"`
	Statement   string       `json:"statement"`
	Source      string       `json:"source"`
	EvidenceRef string       `json:"evidence_ref,omitempty"`
	Confidence  float64      `json:"confidence"` // 0.0 to 1.0
	Status      ItemStatus   `json:"status"`
	CreatedBy   string       `json:"created_by"`
	CreatedAt   time.Time    `json:"created_at"`
}

// Problem represents a structured complex problem entity (AIA-22 §6.1).
type Problem struct {
	ID                  string          `json:"id"`
	ProjectID           string          `json:"project_id"`
	ParentID            string          `json:"parent_id,omitempty"` // For recursive decomposition hierarchy
	GoalID              string          `json:"goal_id,omitempty"`   // Optional link to high-level Goal
	Title               string          `json:"title"`
	Description         string          `json:"description,omitempty"`
	Objective           string          `json:"objective"`
	Scope               string          `json:"scope,omitempty"`
	Status              ProblemStatus   `json:"status"`
	Priority            ProblemPriority `json:"priority"`
	CurrentModelVersion int             `json:"current_model_version"`
	CurrentPlanVersion  int             `json:"current_plan_version"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// ProblemModel represents a versioned structural model of a problem (AIA-22 §6.2).
type ProblemModel struct {
	ID              string      `json:"id"`
	ProblemID       string      `json:"problem_id"`
	Version         int         `json:"version"`
	Objectives      []ModelItem `json:"objectives,omitempty"`
	Constraints     []ModelItem `json:"constraints,omitempty"`
	KnownFacts      []ModelItem `json:"known_facts,omitempty"`
	Assumptions     []ModelItem `json:"assumptions,omitempty"`
	Unknowns        []ModelItem `json:"unknowns,omitempty"`
	Dependencies    []ModelItem `json:"dependencies,omitempty"`
	Risks           []ModelItem `json:"risks,omitempty"`
	SuccessCriteria []ModelItem `json:"success_criteria,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
}

// AnalysisRecord is an immutable log of an individual analysis execution (AIA-22 §6.3).
type AnalysisRecord struct {
	ID            string    `json:"id"`
	ProblemID     string    `json:"problem_id"`
	ModelVersion  int       `json:"model_version"`
	AnalysisType  string    `json:"analysis_type"` // e.g., "INTENT", "CONTEXT", "GAP_CONFLICT", "REASSESSMENT"
	InputRefs     []string  `json:"input_refs,omitempty"`
	Findings      []string  `json:"findings,omitempty"`
	Conclusions   []string  `json:"conclusions,omitempty"`
	OpenQuestions []string  `json:"open_questions,omitempty"`
	AgentID       string    `json:"agent_id,omitempty"`
	ExecutionID   string    `json:"execution_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Validate validates the core Problem entity.
func (p *Problem) Validate() error {
	if p.ID == "" {
		return errors.New("problem ID cannot be empty")
	}
	if p.ProjectID == "" {
		return errors.New("project ID cannot be empty")
	}
	if p.Title == "" {
		return errors.New("problem title cannot be empty")
	}
	if p.Objective == "" {
		return errors.New("problem objective cannot be empty")
	}
	if p.Status == "" {
		p.Status = StatusOpen
	}
	if p.Priority == "" {
		p.Priority = PriorityMedium
	}
	return nil
}

// Validate validates an epistemic ModelItem.
func (m *ModelItem) Validate() error {
	if m.ID == "" {
		return errors.New("model item ID cannot be empty")
	}
	if m.Category == "" {
		return errors.New("model item category cannot be empty")
	}
	if m.Statement == "" {
		return errors.New("model item statement cannot be empty")
	}
	if m.Confidence < 0.0 || m.Confidence > 1.0 {
		return errors.New("model item confidence must be between 0.0 and 1.0")
	}
	if m.Status == "" {
		m.Status = ItemStatusActive
	}
	return nil
}
