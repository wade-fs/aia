package learning

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// MaxPrePostConfidence is the hard epistemic ceiling when verification lacks a control benchmark,
	// Point-in-Time replay counterfactual, or similar-case cohort (AIA-11 §21.2).
	MaxPrePostConfidence = 0.45
)

var (
	ErrUngroundedRCA                    = errors.New("rca rejected: falsifiable_hypothesis and at least one causal_evidence_ref are required (no post-hoc storytelling)")
	ErrTier3FrozenUntilPhase6           = errors.New("correction blocked: TIER3_BLOCKED (Scope C or CODE) is frozen until Phase 6 isolated sandbox")
	ErrSameCycleGeneralizationForbidden = errors.New("learning promotion rejected: L3_GENERALIZED requires independent cross-cycle application (cycle_id != source_cycle_id)")
	ErrGoalpostMovingBlocked            = errors.New("goal revision blocked by Anti-Goalpost-Moving guard: agent cannot revise STRATEGIC goal due to metric shortfall without invalidated premise claims")
	ErrStrategicGoalDirectWriteDenied   = errors.New("governance violation: agent cannot directly mutate STRATEGIC goal without human approval")
)

type ScopeLevel string

const (
	ScopeA ScopeLevel = "A" // Domain tools, thresholds, prompts, local memory/claims
	ScopeB ScopeLevel = "B" // Core runtime / shared services
	ScopeC ScopeLevel = "C" // Governance, policy, approval, audit (locked from agent self-modification)
)

type CorrectionType string

const (
	CorrectionFact             CorrectionType = "FACT"
	CorrectionJudgment         CorrectionType = "JUDGMENT"
	CorrectionStrategy         CorrectionType = "STRATEGY"
	CorrectionPrompt           CorrectionType = "PROMPT"
	CorrectionWorkflow         CorrectionType = "WORKFLOW"
	CorrectionCode             CorrectionType = "CODE"
	CorrectionEvaluationUpdate CorrectionType = "EVALUATION_UPDATE" // AIA-11 §6.1
	CorrectionMemory           CorrectionType = "MEMORY_CORRECTION"  // AIA-11 §6.1
	CorrectionGoalProposal     CorrectionType = "GOAL_PROPOSAL"      // AIA-11 §6.1
)

type AuthorizationLevel string

const (
	AuthC0 AuthorizationLevel = "C0"
	AuthC1 AuthorizationLevel = "C1"
	AuthC2 AuthorizationLevel = "C2"
	AuthC3 AuthorizationLevel = "C3"
	AuthC4 AuthorizationLevel = "C4"
	AuthC5 AuthorizationLevel = "C5"
)

type ValidationLevel string

const (
	ValV1 ValidationLevel = "V1" // Static / schema invariant
	ValV2 ValidationLevel = "V2" // Unit invariant
	ValV3 ValidationLevel = "V3" // Integration verification
	ValV4 ValidationLevel = "V4" // Regression suite
	ValV5 ValidationLevel = "V5" // Behavioral counterfactual
	ValV6 ValidationLevel = "V6" // Longitudinal benchmark
)

type TierProfile string

const (
	Tier1Auto     TierProfile = "TIER1_AUTO"
	Tier2Approval TierProfile = "TIER2_APPROVAL"
	Tier3Blocked  TierProfile = "TIER3_BLOCKED"
)

// ResolveCorrectionProfile collapses the 4 orthogonal dimensions (Scope, Type, Auth, Validation)
// into a deterministic 3-tier runtime profile (Plan Phase 4 §4.1).
func ResolveCorrectionProfile(scope ScopeLevel, cType CorrectionType) (TierProfile, AuthorizationLevel, ValidationLevel) {
	if scope == "" {
		scope = ScopeA
	}
	// Governance core (Scope C) and raw CODE changes are strictly frozen in Phase 4–5
	if scope == ScopeC {
		return Tier3Blocked, AuthC5, ValV6
	}
	if cType == CorrectionCode {
		return Tier3Blocked, AuthC4, ValV6
	}
	if scope == ScopeB {
		return Tier2Approval, AuthC3, ValV4
	}

	switch cType {
	case CorrectionFact, CorrectionMemory:
		return Tier1Auto, AuthC0, ValV2
	case CorrectionJudgment, CorrectionEvaluationUpdate:
		return Tier1Auto, AuthC1, ValV2
	case CorrectionPrompt:
		return Tier2Approval, AuthC2, ValV3
	case CorrectionStrategy, CorrectionWorkflow, CorrectionGoalProposal:
		return Tier2Approval, AuthC3, ValV5
	default:
		return Tier2Approval, AuthC2, ValV3
	}
}

type EvaluationStatus string

const (
	EvalStatusEvaluated EvaluationStatus = "EVALUATED"
	EvalStatusNeedsRCA  EvaluationStatus = "NEEDS_RCA"
	EvalStatusResolved  EvaluationStatus = "RESOLVED"
)

type RootCauseCategory string

const (
	RCIncorrectFact        RootCauseCategory = "INCORRECT_FACT"
	RCBiasedJudgment       RootCauseCategory = "BIASED_JUDGMENT"
	RCFlawedStrategy       RootCauseCategory = "FLAWED_STRATEGY"
	RCAmbiguousPrompt      RootCauseCategory = "AMBIGUOUS_PROMPT"
	RCWorkflowDefect       RootCauseCategory = "WORKFLOW_DEFECT"
	RCExternalUnexpected   RootCauseCategory = "EXTERNAL_UNEXPECTED"
	RCEvaluationError      RootCauseCategory = "EVALUATION_ERROR"      // AIA-11 §4
	RCGoalAssumptionError  RootCauseCategory = "GOAL_ASSUMPTION_ERROR" // AIA-11 §4
)

type RCAValidityStatus string

const (
	RCAProposed             RCAValidityStatus = "PROPOSED"
	RCASupported            RCAValidityStatus = "SUPPORTED"
	RCAFalsified            RCAValidityStatus = "FALSIFIED"
	RCAInterventionMismatch RCAValidityStatus = "INTERVENTION_MISMATCH"
	RCAUncertain            RCAValidityStatus = "UNCERTAIN"
)

type RegressionStatus string

const (
	RegressionPending RegressionStatus = "PENDING"
	RegressionPassed  RegressionStatus = "PASSED"
	RegressionFailed  RegressionStatus = "FAILED"
)

type VerificationStatus string

const (
	VerifPending      VerificationStatus = "PENDING"
	VerifApplied      VerificationStatus = "APPLIED"
	VerifEffective    VerificationStatus = "VERIFIED_EFFECTIVE"
	VerifIneffective  VerificationStatus = "VERIFIED_INEFFECTIVE"
	VerifInconclusive VerificationStatus = "INCONCLUSIVE"
	VerifPartial      VerificationStatus = "PARTIAL" // AIA-11 §10
	VerifRegressed    VerificationStatus = "REGRESSED"
	VerifRolledBack   VerificationStatus = "ROLLED_BACK"
	VerifBlocked      VerificationStatus = "BLOCKED"
)

// LearningType distinguishes the epistemic category of the insight (AIA-11 §10.2).
type LearningType string

const (
	LearnTypeFact         LearningType = "FACT"
	LearnTypePattern      LearningType = "PATTERN"
	LearnTypeHeuristic    LearningType = "HEURISTIC"
	LearnTypeErrorPattern LearningType = "ERROR_PATTERN"
)

type LearningLevel string

const (
	LevelL0Recorded    LearningLevel = "L0_RECORDED"
	LevelL1Applied     LearningLevel = "L1_APPLIED"
	LevelL2Verified    LearningLevel = "L2_VERIFIED"
	LevelL3Generalized LearningLevel = "L3_GENERALIZED"
)

type EffectivenessVerdict string

const (
	VerdictPending      EffectivenessVerdict = "PENDING"
	VerdictEffective    EffectivenessVerdict = "EFFECTIVE"
	VerdictIneffective  EffectivenessVerdict = "INEFFECTIVE"
	VerdictRegressed    EffectivenessVerdict = "REGRESSED"
	VerdictInconclusive EffectivenessVerdict = "INCONCLUSIVE"
)

type ComparisonMethod string

const (
	CompControlBenchmark ComparisonMethod = "CONTROL_BENCHMARK"
	CompPiTReplay        ComparisonMethod = "PIT_REPLAY"
	CompSimilarCohort    ComparisonMethod = "SIMILAR_COHORT"
	CompPrePostOnly      ComparisonMethod = "PRE_POST_ONLY"
)

type OutcomeEvaluation struct {
	ID                  string           `json:"id"`
	CycleID             *string          `json:"cycle_id,omitempty"`
	DecisionID          *string          `json:"decision_id,omitempty"`
	TaskID              *string          `json:"task_id,omitempty"`
	ProjectID           *string          `json:"project_id,omitempty"`
	Domain              string           `json:"domain"`
	ExpectedOutcomeJSON string           `json:"expected_outcome_json"`
	ActualOutcomeJSON   string           `json:"actual_outcome_json"`
	DiscrepancyScore    float64          `json:"discrepancy_score"`
	IsAnomaly           bool             `json:"is_anomaly"`
	EvaluationStatus    EvaluationStatus `json:"evaluation_status"`
	EvaluatedAt         time.Time        `json:"evaluated_at"`
}

type RootCauseAnalysis struct {
	ID                     string            `json:"id"`
	OutcomeEvaluationID    string            `json:"outcome_evaluation_id"`
	RootCauseCategory      RootCauseCategory `json:"root_cause_category"`
	AnalysisNarrative      string            `json:"analysis_narrative"`
	FalsifiableHypothesis  string            `json:"falsifiable_hypothesis"`
	CausalEvidenceRefsJSON string            `json:"causal_evidence_refs_json"`
	CompetingFactorsJSON   string            `json:"competing_factors_json"`
	RCAValidityStatus      RCAValidityStatus `json:"rca_validity_status"`
	Confidence             float64           `json:"confidence"`
	CreatedAt              time.Time         `json:"created_at"`
}

func (r *RootCauseAnalysis) Validate() error {
	if strings.TrimSpace(r.OutcomeEvaluationID) == "" {
		return fmt.Errorf("outcome_evaluation_id is required")
	}
	if strings.TrimSpace(r.FalsifiableHypothesis) == "" {
		return ErrUngroundedRCA
	}
	if strings.TrimSpace(r.CausalEvidenceRefsJSON) == "" || strings.TrimSpace(r.CausalEvidenceRefsJSON) == "[]" {
		return ErrUngroundedRCA
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("confidence %.2f out of [0,1]", r.Confidence)
	}
	return nil
}

type CorrectionAction struct {
	ID                 string             `json:"id"`
	RCAID              string             `json:"rca_id"`
	ScopeLevel         ScopeLevel         `json:"scope_level"`
	CorrectionType     CorrectionType     `json:"correction_type"`
	AuthorizationLevel AuthorizationLevel `json:"authorization_level"`
	ValidationLevel    ValidationLevel    `json:"validation_level"`
	TierProfile        TierProfile        `json:"tier_profile"`
	TargetEntityRef    string             `json:"target_entity_ref"`
	BeforeStateJSON    string             `json:"before_state_json"`
	ProposedStateJSON  string             `json:"proposed_state_json"`
	AppliedStateJSON   string             `json:"applied_state_json,omitempty"`
	ApprovalID         *string            `json:"approval_id,omitempty"`
	RegressionStatus   RegressionStatus   `json:"regression_status"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// LearningRecord captures the 7 mandatory epistemic comparison fields (AIA-11 §21.1):
// 1. Baseline, 2. Hypothesis, 3. Intervention, 4. Verification, 5. Regression, 6. Follow-up, 7. Confidence.
type LearningRecord struct {
	ID                         string               `json:"id"`
	ProjectID                  *string              `json:"project_id,omitempty"`
	Domain                     string               `json:"domain"`
	SourceCycleID              *string              `json:"source_cycle_id,omitempty"`
	OutcomeEvaluationID        string               `json:"outcome_evaluation_id"`
	RCAID                      *string              `json:"rca_id,omitempty"`
	CorrectionActionID         *string              `json:"correction_action_id,omitempty"`
	Title                      string               `json:"title"`
	LearningLevel              LearningLevel        `json:"learning_level"`
	LearningType               LearningType         `json:"learning_type,omitempty"` // AIA-11 §10.2: FACT, PATTERN, HEURISTIC, ERROR_PATTERN
	EffectivenessVerdict       EffectivenessVerdict `json:"effectiveness_verdict"`
	ComparisonMethod           ComparisonMethod     `json:"comparison_method"`
	BaselineJSON               string               `json:"baseline_json"`
	HypothesisJSON             string               `json:"hypothesis_json"`
	InterventionJSON           string               `json:"intervention_json"`
	VerificationJSON           string               `json:"verification_json"`
	RegressionJSON             string               `json:"regression_json"`
	FollowupJSON               string               `json:"followup_json"`
	ApplicabilityConditionsJSON string              `json:"applicability_conditions_json,omitempty"` // AIA-11 §10.2
	RolloutMode                string               `json:"rollout_mode,omitempty"`                // AIA-11 §8.3: DIRECT, CANARY
	VerificationDelayCycles    int                  `json:"verification_delay_cycles,omitempty"`   // AIA-11 §9.3
	EpistemicConfidence        float64              `json:"epistemic_confidence"`
	GeneralizedCycleIDsJSON    string               `json:"generalized_cycle_ids_json"`
	CreatedAt                  time.Time            `json:"created_at"`
	UpdatedAt                  time.Time            `json:"updated_at"`
}

type GoalEvolutionMode string

const (
	EvolutionPersistence   GoalEvolutionMode = "PERSISTENCE"
	EvolutionOptimization  GoalEvolutionMode = "OPTIMIZATION"
	EvolutionRevision      GoalEvolutionMode = "REVISION"
	EvolutionDecomposition GoalEvolutionMode = "DECOMPOSITION" // AIA-09 §17.2
	EvolutionMerge         GoalEvolutionMode = "MERGE"         // AIA-09 §17.2
	EvolutionRetirement    GoalEvolutionMode = "RETIREMENT"    // AIA-09 §17.2
	EvolutionReplacement   GoalEvolutionMode = "REPLACEMENT"   // AIA-09 §17.2
)

type GoalEvolutionStatus string

const (
	EvoApplied                GoalEvolutionStatus = "APPLIED"
	EvoPendingApproval        GoalEvolutionStatus = "PENDING_APPROVAL"
	EvoApproved               GoalEvolutionStatus = "APPROVED"
	EvoRejectedGoalpostMoving GoalEvolutionStatus = "REJECTED_GOALPOST_MOVING"
	EvoRejected               GoalEvolutionStatus = "REJECTED"
)

type GoalEvolutionProposal struct {
	ID                       string              `json:"id"`
	GoalID                   string              `json:"goal_id"`
	CycleID                  *string             `json:"cycle_id,omitempty"`
	EvolutionMode            GoalEvolutionMode   `json:"evolution_mode"`
	ActorType                string              `json:"actor_type"`
	ActorID                  string              `json:"actor_id"`
	Rationale                string              `json:"rationale"`
	PreviousTargetMetricJSON string              `json:"previous_target_metric_json"`
	ProposedTargetMetricJSON string              `json:"proposed_target_metric_json"`
	InvalidatedClaimIDsJSON  string              `json:"invalidated_claim_ids_json"`
	ResourceConstraintNote   string              `json:"resource_constraint_note"`
	ExpectedOutcome          string              `json:"expected_outcome,omitempty"` // AIA-09 §17.1
	RiskAssessment           string              `json:"risk_assessment,omitempty"`  // AIA-09 §17.1
	ApprovedBy               string              `json:"approved_by,omitempty"`      // AIA-09 §17.1
	EffectiveAt              *time.Time          `json:"effective_at,omitempty"`     // AIA-09 §17.1
	ReviewDate               *time.Time          `json:"review_date,omitempty"`      // AIA-09 §17.1
	Status                   GoalEvolutionStatus `json:"status"`
	ApprovalID               *string             `json:"approval_id,omitempty"`
	ResultingRevisionID      *string             `json:"resulting_revision_id,omitempty"`
	CreatedAt                time.Time           `json:"created_at"`
	UpdatedAt                time.Time           `json:"updated_at"`
}
