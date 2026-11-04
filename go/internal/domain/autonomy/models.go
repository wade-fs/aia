package autonomy

import (
	"context"
	"time"

	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/governance"
)

// DiscoveryType classifies how an autonomous candidate item was discovered (AIA-09 §3.2).
type DiscoveryType string

const (
	DiscoveryIssue       DiscoveryType = "ISSUE"
	DiscoveryOpportunity DiscoveryType = "OPPORTUNITY"
	DiscoveryGoalGap     DiscoveryType = "GOAL_GAP"
	DiscoveryContinuity  DiscoveryType = "CONTINUITY"
	DiscoveryImprovement DiscoveryType = "IMPROVEMENT"
	DiscoveryProblem     DiscoveryType = "PROBLEM"
)

// DiscoveryConfidence reflects evidentiary certainty for a discovery (AIA-09 §4.2).
type DiscoveryConfidence string

const (
	ConfidenceConfirmed  DiscoveryConfidence = "CONFIRMED"
	ConfidenceProbable   DiscoveryConfidence = "PROBABLE"
	ConfidencePotential  DiscoveryConfidence = "POTENTIAL"
	ConfidenceUnverified DiscoveryConfidence = "UNVERIFIED"
)

// InitiativeStatus defines the lifecycle of a discovered candidate initiative.
type InitiativeStatus string

const (
	InitiativeCandidate  InitiativeStatus = "CANDIDATE"
	InitiativeAdopted    InitiativeStatus = "ADOPTED"
	InitiativeDiscarded  InitiativeStatus = "DISCARDED"
	InitiativeDeferred   InitiativeStatus = "DEFERRED"
	InitiativeMonitoring InitiativeStatus = "MONITORING"
)

// CandidateInitiative represents an opportunity, issue, or goal gap discovered autonomously.
type CandidateInitiative struct {
	ID                  string              `json:"id"`
	CycleID             string              `json:"cycle_id"`
	ProjectID           *string             `json:"project_id,omitempty"`
	GoalID              *string             `json:"goal_id,omitempty"`
	ProblemID           *string             `json:"problem_id,omitempty"`
	DiscoveryType       DiscoveryType       `json:"discovery_type"`
	ConfidenceLevel     DiscoveryConfidence `json:"confidence_level"`
	TargetAgentID       string              `json:"target_agent_id"`
	TriggerSource       string              `json:"trigger_source"`
	DetectedOpportunity string              `json:"detected_opportunity"`
	Rationale           string              `json:"rationale"`
	ExpectedOutcome     string              `json:"expected_outcome,omitempty"`
	PriorityScore       int                 `json:"priority_score"`
	EstimatedCost       float64             `json:"estimated_cost"`
	EstimatedRisk       governance.RiskTier `json:"estimated_risk"`
	Status              InitiativeStatus    `json:"status"`
	DiscardReason       *string             `json:"discard_reason,omitempty"`
	SpawnedTaskID       *string             `json:"spawned_task_id,omitempty"`
	MetadataJSON        string              `json:"metadata_json,omitempty"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

// DeliberationOption represents a single evaluated alternative in the deliberation matrix.
type DeliberationOption struct {
	Option          string              `json:"option"`
	ActionCode      decision.ActionType `json:"action_code"`
	SuccessProb     float64             `json:"success_prob"`
	ExpectedValue   float64             `json:"expected_value"`
	Cost            float64             `json:"cost"`
	RiskPenalty     float64             `json:"risk_penalty"`
	NetUtility      float64             `json:"net_utility"`
	Reversible      bool                `json:"reversible"`
	Selected        bool                `json:"selected"`
	SelectionReason string              `json:"selection_reason,omitempty"`
}

// DeliberationRecord captures the structured deliberation trade-off matrix (AIA-08 §4, AIA-09 §6).
type DeliberationRecord struct {
	ID                    string              `json:"id"`
	CycleID               string              `json:"cycle_id"`
	DecisionID            *string             `json:"decision_id,omitempty"`
	CandidateInitiativeID *string             `json:"candidate_initiative_id,omitempty"`
	OptionsEvaluatedJSON  string              `json:"options_evaluated_json"`
	CostBenefitAnalysis   string              `json:"cost_benefit_analysis"`
	RiskAssessment        string              `json:"risk_assessment"`
	RecommendedAction     decision.ActionType `json:"recommended_action"`
	CreatedAt             time.Time           `json:"created_at"`
}

// GoalTaskLink connects a long-term Goal to a concrete Task decomposed to advance it.
type GoalTaskLink struct {
	GoalID                 string    `json:"goal_id"`
	TaskID                 string    `json:"task_id"`
	CycleID                *string   `json:"cycle_id,omitempty"`
	DecompositionRationale string    `json:"decomposition_rationale"`
	CreatedAt              time.Time `json:"created_at"`
}

// ObservationCategory distinguishes calm baseline state snapshots from actionable signals.
const (
	ObsCategoryStateSnapshot    = "STATE_SNAPSHOT"
	ObsCategoryMaterialChange   = "MATERIAL_CHANGE"
	ObsCategoryConstraintBreach = "CONSTRAINT_BREACH"
	ObsCategoryAnomaly          = "ANOMALY"
	ObsCategoryOpportunity      = "OPPORTUNITY"
	ObsCategoryDrift            = "DRIFT"
	ObsCategoryConflict         = "CONFLICT"
)

// Observation represents an environmental fact, state snapshot, or signal observed at Step 1.
type Observation struct {
	Source        string              `json:"source"`
	Domain        string              `json:"domain,omitempty"`
	Category      string              `json:"category,omitempty"`
	Summary       string              `json:"summary"`
	NonActionable bool                `json:"non_actionable,omitempty"`
	Confidence    DiscoveryConfidence `json:"confidence,omitempty"`
	Severity      int                 `json:"severity,omitempty"` // 1..10
	TargetAgentID string              `json:"target_agent_id,omitempty"`
	GoalID        *string             `json:"goal_id,omitempty"`
	EvidenceID    *string             `json:"evidence_id,omitempty"`
	Data          map[string]any      `json:"data,omitempty"`
	Timestamp     time.Time           `json:"timestamp"`
}

// IsCalmSnapshot reports whether the observation represents a healthy/calm baseline
// rather than an anomaly, drift, or opportunity requiring action.
func (o Observation) IsCalmSnapshot() bool {
	if o.Category == ObsCategoryDrift || o.Category == ObsCategoryAnomaly ||
		o.Category == ObsCategoryConstraintBreach || o.Category == ObsCategoryConflict ||
		o.Category == ObsCategoryMaterialChange || o.Category == ObsCategoryOpportunity {
		return o.NonActionable
	}
	if o.NonActionable || o.Category == ObsCategoryStateSnapshot {
		return true
	}
	if o.Data != nil {
		if isMat, ok := o.Data["is_material"].(bool); ok && !isMat {
			return true
		}
	}
	return false
}

// StepTrace provides per-step observability across the 11-step Autonomy Cycle (AIA-19, Plan §2).
type StepTrace struct {
	StepNumber int            `json:"step_number"`
	StepName   string         `json:"step_name"`
	Status     string         `json:"status"`
	Summary    string         `json:"summary"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	DurationMs int64          `json:"duration_ms"`
}

// VerificationSummary records Step 9 (Verify) outcome validation.
type VerificationSummary struct {
	Passed          bool      `json:"passed"`
	ExpectedSummary string    `json:"expected_summary"`
	ActualSummary   string    `json:"actual_summary"`
	Checks          []string  `json:"checks"`
	VerifiedAt      time.Time `json:"verified_at"`
}

// CycleOutcome stores the complete result and trace of an executed 11-step autonomy cycle.
type CycleOutcome struct {
	CycleID             string                      `json:"cycle_id"`
	CycleNumber         int64                       `json:"cycle_number"`
	Observations        []Observation               `json:"observations,omitempty"`
	Initiatives         []*CandidateInitiative      `json:"initiatives,omitempty"`
	Deliberation        *DeliberationRecord         `json:"deliberation,omitempty"`
	Decision            *decision.Decision          `json:"decision"`
	GovernanceResult    governance.GovernanceAction `json:"governance_result"`
	ApprovalID          *string                     `json:"approval_id,omitempty"`
	TaskID              *string                     `json:"task_id,omitempty"`
	Verification        *VerificationSummary        `json:"verification,omitempty"`
	LearnedMemoryID     *string                     `json:"learned_memory_id,omitempty"`
	PostCycleSnapshotID *string                     `json:"post_cycle_snapshot_id,omitempty"`
	StepTraces          []StepTrace                 `json:"step_traces,omitempty"`
	Summary             string                      `json:"summary"`
	CompletedAt         time.Time                   `json:"completed_at"`
}

// EnvironmentObserver allows domain modules and subsystems to report environmental observations.
type EnvironmentObserver interface {
	Observe(ctx context.Context, projectID string) ([]Observation, error)
}
