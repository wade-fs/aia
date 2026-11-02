package problem

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrMaxDepthExceeded       = errors.New("maximum problem decomposition depth exceeded")
	ErrMaxSubProblemsExceeded = errors.New("maximum number of sub-problems exceeded")
	ErrMissingTriggerEvidence = errors.New("reassessment requires explicit trigger evidence")
	ErrMissingRootCause       = errors.New("reassessment requires root-cause attribution (AIA-22 §8.3)")
)

// ReassessmentTriggerType defines the event that triggered plan re-evaluation (AIA-22 §8.1).
type ReassessmentTriggerType string

const (
	TriggerHypothesisContradicted ReassessmentTriggerType = "HYPOTHESIS_CONTRADICTED"
	TriggerRuleConflict           ReassessmentTriggerType = "RULE_CONFLICT"
	TriggerTaskFailure            ReassessmentTriggerType = "TASK_FAILURE"
	TriggerMissingDependency      ReassessmentTriggerType = "MISSING_DEPENDENCY"
	TriggerConstraintChanged      ReassessmentTriggerType = "CONSTRAINT_CHANGED"
	TriggerAcceptanceUnachievable ReassessmentTriggerType = "ACCEPTANCE_UNACHIEVABLE"
	TriggerHumanIntervention      ReassessmentTriggerType = "HUMAN_INTERVENTION"
)

// RootCauseCategory distinguishes the true source of failure (AIA-22 §8.3).
type RootCauseCategory string

const (
	RootCauseModelError            RootCauseCategory = "MODEL_ERROR"
	RootCauseAssumptionInvalid     RootCauseCategory = "ASSUMPTION_INVALID"
	RootCauseDataShortage          RootCauseCategory = "DATA_SHORTAGE"
	RootCausePlanUnreasonable      RootCauseCategory = "PLAN_UNREASONABLE"
	RootCauseExecutionFailure      RootCauseCategory = "EXECUTION_FAILURE"
	RootCauseEnvironmentFault      RootCauseCategory = "ENVIRONMENT_FAULT"
	RootCauseAcceptanceUnrealistic RootCauseCategory = "ACCEPTANCE_UNREALISTIC"
	RootCauseExternalChanged       RootCauseCategory = "EXTERNAL_CHANGED"
)

// ReassessmentAction defines the decided course of action following re-evaluation (AIA-22 §8.2).
type ReassessmentAction string

const (
	ActionContinue    ReassessmentAction = "CONTINUE"
	ActionModifyPlan  ReassessmentAction = "MODIFY_PLAN"
	ActionRebuildPlan ReassessmentAction = "REBUILD_PLAN"
	ActionClarify     ReassessmentAction = "REQUEST_CLARIFICATION"
	ActionPause       ReassessmentAction = "PAUSE"
)

// ReassessmentRequest encapsulates the input for dynamic plan re-evaluation (AIA-22 §8.2).
type ReassessmentRequest struct {
	ProblemID                string                  `json:"problem_id"`
	Trigger                  ReassessmentTriggerType `json:"trigger"`
	TriggerEvidence          string                  `json:"trigger_evidence"`
	RootCause                RootCauseCategory       `json:"root_cause"`
	NewFindings              []string                `json:"new_findings,omitempty"`
	ContradictedHypotheses   []string                `json:"contradicted_hypotheses,omitempty"`
	SuggestedAction          ReassessmentAction      `json:"suggested_action"`
	Rationale                string                  `json:"rationale"`
	AgentID                  string                  `json:"agent_id,omitempty"`
}

// ReassessmentResult details the outcome of the re-evaluation and model adjustment (AIA-22 §8.2).
type ReassessmentResult struct {
	ProblemID         string             `json:"problem_id"`
	PreviousModelVer  int                `json:"previous_model_ver"`
	NewModelVer       int                `json:"new_model_ver"`
	ActionTaken       ReassessmentAction `json:"action_taken"`
	AnalysisRecordID  string             `json:"analysis_record_id"`
	ContradictedCount int                `json:"contradicted_count"`
	RequiresNewPlan   bool               `json:"requires_new_plan"`
	ExecutedAt        time.Time          `json:"executed_at"`
}

// DecompositionConfig sets bounded depth and resource quotas to prevent infinite subproblem expansion (AIA-22 §5.2).
type DecompositionConfig struct {
	MaxDepth       int `json:"max_depth"`        // Default 4
	MaxSubProblems int `json:"max_sub_problems"` // Default 20
}

// DefaultDecompositionConfig returns production defaults for problem decomposition limits.
func DefaultDecompositionConfig() DecompositionConfig {
	return DecompositionConfig{
		MaxDepth:       4,
		MaxSubProblems: 20,
	}
}

// ProblemTreeNode represents a node in the hierarchical problem decomposition tree (AIA-22 §5.1).
type ProblemTreeNode struct {
	Problem  *Problem           `json:"problem"`
	Depth    int                `json:"depth"`
	Children []*ProblemTreeNode `json:"children,omitempty"`
}

// ValidateReassessmentRequest ensures reassessments have verified triggers and root causes (AIA-22 §8.1, §8.3).
func ValidateReassessmentRequest(req *ReassessmentRequest) error {
	if req == nil {
		return errors.New("reassessment request cannot be nil")
	}
	if strings.TrimSpace(req.ProblemID) == "" {
		return errors.New("problem ID cannot be empty")
	}
	if req.Trigger == "" {
		return errors.New("reassessment trigger type cannot be empty")
	}
	if strings.TrimSpace(req.TriggerEvidence) == "" {
		return ErrMissingTriggerEvidence
	}
	if req.RootCause == "" {
		return ErrMissingRootCause
	}
	if req.SuggestedAction == "" {
		req.SuggestedAction = ActionModifyPlan
	}
	if strings.TrimSpace(req.Rationale) == "" {
		return errors.New("reassessment rationale cannot be empty")
	}
	return nil
}
