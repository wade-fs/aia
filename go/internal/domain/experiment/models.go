package experiment

import (
	"time"

	"github.com/wade/aia/internal/domain/investment"
)

type Status string

const (
	StatusProposed        Status = "PROPOSED"
	StatusPendingApproval Status = "PENDING_APPROVAL"
	StatusReady           Status = "READY"
	StatusRunning         Status = "RUNNING"
	StatusPaused          Status = "PAUSED"
	StatusEvaluating      Status = "EVALUATING"
	StatusNeedsRevision   Status = "NEEDS_REVISION"
	StatusCompleted       Status = "COMPLETED"
	StatusStopped         Status = "STOPPED"
	StatusFailed          Status = "FAILED"
	StatusArchived        Status = "ARCHIVED"
)

type GenerationRole string

const (
	RoleControlBaseline GenerationRole = "CONTROL_BASELINE"
	RoleExperimental    GenerationRole = "EXPERIMENTAL"
)

type EngineType string

const (
	EngineRuleBased EngineType = "RULE_BASED"
	EngineLLMAgy    EngineType = "LLM_AGY"
	EngineHybrid    EngineType = "HYBRID"
)

type GenerationStatus string

const (
	GenStatusActive     GenerationStatus = "ACTIVE"
	GenStatusPromoted   GenerationStatus = "PROMOTED"
	GenStatusRolledBack GenerationStatus = "ROLLED_BACK"
	GenStatusRetired    GenerationStatus = "RETIRED"
)

// RegimeVerdict encodes the 5 scientific comparison regimes defined in AIA-13 §21.4 & dual.md §4.
type RegimeVerdict string

const (
	RegimeG2ClearAdvantage       RegimeVerdict = "G2_CLEAR_ADVANTAGE"
	RegimeG2UnreliableRiskTaking RegimeVerdict = "G2_UNRELIABLE_RISK_TAKING"
	RegimeG2LearningInProgress   RegimeVerdict = "G2_LEARNING_IN_PROGRESS"
	RegimeG1RulesSuperior        RegimeVerdict = "G1_RULES_SUPERIOR"
	RegimeParityAdaptive         RegimeVerdict = "PARITY_ADAPTIVE"
	RegimeInconclusiveLowSample  RegimeVerdict = "INCONCLUSIVE_LOW_SAMPLE"
)

// Experiment represents a longitudinal autonomous research experiment (AIA-15).
type Experiment struct {
	ID                  string    `json:"id"`
	ProjectID           *string   `json:"project_id,omitempty"`
	PrimaryGoalID       *string   `json:"primary_goal_id,omitempty"`
	Domain              string    `json:"domain"`
	OwnerAgentID        string    `json:"owner_agent_id"`
	Name                string    `json:"name"`
	Hypothesis          string    `json:"hypothesis"`
	ExperimentType      string    `json:"experiment_type"`
	Status              Status    `json:"status"`
	BaselineMetricsJSON string    `json:"baseline_metrics_json"`
	CurrentMetricsJSON  string    `json:"current_metrics_json"`
	IterationCount      int       `json:"iteration_count"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ExperimentGeneration binds a versioned cohort (G1, G2, G2.1, ...) to its own isolated portfolio ledger (AIA-05 §20, AIA-15 §21).
type ExperimentGeneration struct {
	ID                 string           `json:"id"`
	ExperimentID       string           `json:"experiment_id"`
	GenerationCode     string           `json:"generation_code"` // e.g. "G1", "G2", "G2.1"
	ParentGenerationID *string          `json:"parent_generation_id,omitempty"`
	Role               GenerationRole   `json:"role"`
	EngineType         EngineType       `json:"engine_type"`
	PortfolioProfileID string           `json:"portfolio_profile_id"`
	IsFrozen           bool             `json:"is_frozen"`
	HarnessVersion     string           `json:"harness_version"`
	ModelID            string           `json:"model_id"`
	PromptVersion      string           `json:"prompt_version"`
	StrategyVersion    string           `json:"strategy_version"`
	StrategyConfigJSON string           `json:"strategy_config_json"`
	SourceCorrectionID *string          `json:"source_correction_id,omitempty"`
	SourceLearningID   *string          `json:"source_learning_id,omitempty"`
	Status             GenerationStatus `json:"status"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

// GenerationalForecastOutput enforces the dual-output contract (AIA-05 §20.2 / dual.md §3):
// separating Event Probability (UpProbability) from overall Decision Confidence (DecisionConfidence).
type GenerationalForecastOutput struct {
	Symbol                 string   `json:"symbol"`
	HorizonDays            int      `json:"horizon_days"`
	UpProbability          float64  `json:"up_probability"`           // Event probability [0,1] -> evaluated via Brier Score & Log Loss
	ExpectedReturnPct      float64  `json:"expected_return_pct"`      // e.g. +2.0 (%)
	ExpectedRangeLowPct    float64  `json:"expected_range_low_pct"`   // e.g. -3.0 (%)
	ExpectedRangeHighPct   float64  `json:"expected_range_high_pct"`  // e.g. +5.0 (%)
	Action                 string   `json:"action"`                   // BUY, SELL, ADD, REDUCE, HOLD
	TargetAmount           int64    `json:"target_amount,omitempty"`  // Optional TWD allocation override
	DecisionConfidence     float64  `json:"decision_confidence"`      // Action choice confidence [0,1] -> evaluated separately from UpProbability
	Reasoning              []string `json:"reasoning"`
	PrimaryRisks           []string `json:"primary_risks"`
	InvalidationConditions []string `json:"invalidation_conditions"`
}

// TrackAPerformance captures portfolio investment outcomes & dual calibration metrics (AIA-13 §21.4 Track A).
type TrackAPerformance struct {
	GenerationCode        string                               `json:"generation_code"`
	PortfolioProfileID    string                               `json:"portfolio_profile_id"`
	InitialCapital        int64                                `json:"initial_capital"`
	TotalAssets           int64                                `json:"total_assets"`
	CashBalance           int64                                `json:"cash_balance"`
	HoldingsValue         int64                                `json:"holdings_value"`
	CashAllocationPct     float64                              `json:"cash_allocation_pct"`
	CumulativeReturnPct   float64                              `json:"cumulative_return_pct"`
	RealizedPnL           int64                                `json:"realized_pnl"`
	UnrealizedPnL         int64                                `json:"unrealized_pnl"`
	MaxDrawdownPct        float64                              `json:"max_drawdown_pct"`
	TotalTransactionCosts int64                                `json:"total_transaction_costs"`
	ExecutedTradeCount    int                                  `json:"executed_trade_count"`
	EventCalibration      investment.CalibrationMetrics        `json:"event_calibration"`
	DecisionConfidence    investment.DecisionConfidenceMetrics `json:"decision_confidence"`
}

// TrackBAutonomy captures autonomous reasoning, restraint, and verifiable learning metrics (AIA-13 §21.4 Track B).
type TrackBAutonomy struct {
	GenerationCode               string  `json:"generation_code"`
	ObservationDecisions         int     `json:"observation_decisions"`
	ActiveTradeDecisions         int     `json:"active_trade_decisions"`
	RationalHoldCount            int     `json:"rational_hold_count"`
	RationalHoldRate             float64 `json:"rational_hold_rate"`
	InvalidationRulesSpecified   int     `json:"invalidation_rules_specified"`
	InvalidationTriggeredCount   int     `json:"invalidation_triggered_count"`
	LineageDepth                 int     `json:"lineage_depth"` // 0 for G1/G2, 1 for G2.1, 2 for G2.2...
	HasVerifiedParentCorrection  bool    `json:"has_verified_parent_correction"`
	VerifiedEffectiveLearnings   int     `json:"verified_effective_learnings"`
	GeneralizedLearnings         int     `json:"generalized_learnings"`
}

// ExperimentComparison persists a point-in-time head-to-head evaluation between G1 and a G2/G2.x cohort.
type ExperimentComparison struct {
	ID                        string           `json:"id"`
	ExperimentID              string           `json:"experiment_id"`
	BaselineGenerationID      string           `json:"baseline_generation_id"`
	CandidateGenerationID     string           `json:"candidate_generation_id"`
	AsOf                      time.Time        `json:"as_of"`
	SampleCount               int              `json:"sample_count"`
	PerformanceComparisonJSON string           `json:"performance_comparison_json"`
	AutonomyComparisonJSON    string           `json:"autonomy_comparison_json"`
	RegimeVerdict             RegimeVerdict    `json:"regime_verdict"`
	Summary                   string           `json:"summary"`
	CreatedAt                 time.Time        `json:"created_at"`
	BaselineTrackA            TrackAPerformance `json:"baseline_track_a"`
	CandidateTrackA           TrackAPerformance `json:"candidate_track_a"`
	BaselineTrackB            TrackBAutonomy    `json:"baseline_track_b"`
	CandidateTrackB           TrackBAutonomy    `json:"candidate_track_b"`
}

// DelegationRequest models cross-domain agent delegation (AIA-12, Migration 012).
type DelegationRequest struct {
	ID                 string    `json:"id"`
	ExperimentID       *string   `json:"experiment_id,omitempty"`
	RequesterAgentID   string    `json:"requester_agent_id"`
	TargetAgentID      string    `json:"target_agent_id"`
	SourceProjectID    *string   `json:"source_project_id,omitempty"`
	GoalID             *string   `json:"goal_id,omitempty"`
	SpawnedTaskID      *string   `json:"spawned_task_id,omitempty"`
	TaskSpecJSON       string    `json:"task_spec_json"`
	AcceptanceGateJSON string    `json:"acceptance_gate_json"`
	DeliverableJSON    string    `json:"deliverable_json"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
