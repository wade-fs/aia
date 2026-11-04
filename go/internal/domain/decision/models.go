package decision

import "time"

type ActionType string

const (
	ActionAct            ActionType = "ACT"
	ActionDefer          ActionType = "DEFER"
	ActionMonitor        ActionType = "MONITOR"
	ActionAskHuman       ActionType = "ASK_HUMAN"
	ActionNoAction       ActionType = "NO_ACTION"
	ActionInvestigate    ActionType = "INVESTIGATE"     // AIA-09 §5.2: Autonomous investigation
	ActionValidate       ActionType = "VALIDATE"        // AIA-09 §5.2: Verification of premise or claim
	ActionDecompose      ActionType = "DECOMPOSE"       // AIA-09 §5.2: Goal or complex problem breakdown
	ActionEscalate       ActionType = "ESCALATE"        // AIA-09 §5.2: Human or governance escalation
	ActionSleep          ActionType = "SLEEP"           // AIA-09 §8.7: Quiescent sleep with wake trigger
	ActionResearch       ActionType = "RESEARCH"        // AIA-21 §10.2: Autonomous evidence & data gathering
	ActionProposeDefault ActionType = "PROPOSE_DEFAULT" // AIA-21 §10.2: Propose provisional hypothesis with rationale
	ActionProceed        ActionType = "PROCEED"         // AIA-21 §10.2: Ready for planning or execution
	ActionReassess       ActionType = "REASSESS"        // AIA-21 §10.2: Reassessment triggered by contradiction
)

type GovernanceResult string

const (
	GovernanceAllowed          GovernanceResult = "ALLOWED"
	GovernanceRequiredApproval GovernanceResult = "REQUIRED_APPROVAL"
	GovernanceBlocked          GovernanceResult = "BLOCKED"
)

// Decision records a deliberation outcome, motivation, and alternatives evaluated (AIA-08 §7.1).
type Decision struct {
	ID                         string           `json:"id"`
	CycleID                    *string          `json:"cycle_id,omitempty"`
	ProjectID                  *string          `json:"project_id,omitempty"`
	ExperimentID               *string          `json:"experiment_id,omitempty"`
	Domain                     string           `json:"domain,omitempty"`
	ActionType                 ActionType       `json:"action_type"`
	IntentSummary              string           `json:"intent_summary"`
	Justification              string           `json:"justification"`
	Confidence                 float64          `json:"confidence,omitempty"`
	ExpectedOutcome            string           `json:"expected_outcome,omitempty"`
	RiskAssessment             string           `json:"risk_assessment,omitempty"`
	ApprovalStatus             string           `json:"approval_status,omitempty"`
	ReevaluationConditions     string           `json:"reevaluation_conditions,omitempty"` // AIA-08 §7.3
	ConsideredAlternativesJSON string           `json:"considered_alternatives_json,omitempty"`
	ContextSnapshotID          *string          `json:"context_snapshot_id,omitempty"`
	GovernanceResult           GovernanceResult `json:"governance_result"`
	ActionTaskID               *string          `json:"action_task_id,omitempty"`
	ActionExecutionID          *string          `json:"action_execution_id,omitempty"`
	CreatedAt                  time.Time        `json:"created_at"`
}

// DecisionEvidenceLink connects a Decision to supporting or refuting Evidence.
type DecisionEvidenceLink struct {
	DecisionID    string `json:"decision_id"`
	EvidenceID    string `json:"evidence_id"`
	RelevanceNote string `json:"relevance_note,omitempty"`
}
