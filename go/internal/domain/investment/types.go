package investment

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

type DecisionType string

const (
	DecisionBuy  DecisionType = "BUY"
	DecisionSell DecisionType = "SELL"
	DecisionHold DecisionType = "HOLD"
)

type ProposalStatus string

const (
	ProposalPending  ProposalStatus = "PENDING"
	ProposalApproved ProposalStatus = "APPROVED"
	ProposalRejected ProposalStatus = "REJECTED"
	ProposalExpired  ProposalStatus = "EXPIRED"
)

// Instrument represents an investable asset (e.g. TWSE:2330).
type Instrument struct {
	ID        domain.ID `json:"id"`
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	AssetType string    `json:"asset_type"`
	Name      string    `json:"name"`
	Currency  string    `json:"currency"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PortfolioProfile holds investment capital and strategy constraints.
type PortfolioProfile struct {
	ID                 domain.ID `json:"id"`
	Name               string    `json:"name"`
	Market             string    `json:"market"`
	Currency           string    `json:"currency"`
	CapitalLimit       int64     `json:"capital_limit"`
	CapitalLimitBasis  string    `json:"capital_limit_basis"` // e.g. "TOTAL_EQUITY" or "AVAILABLE_CASH"
	AutoTradingEnabled bool      `json:"auto_trading_enabled"` // MUST be false (AIA never auto executes trades)
	ConfigurationJSON  string    `json:"configuration_json"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// WatchlistItem is a target asset watched by a portfolio.
type WatchlistItem struct {
	ID                 domain.ID `json:"id"`
	PortfolioProfileID domain.ID `json:"portfolio_profile_id"`
	InstrumentID       domain.ID `json:"instrument_id"`
	Priority           int       `json:"priority"`
	CreatedAt          time.Time `json:"created_at"`
}

// Constraint is a portfolio risk control (e.g. max_position_pct, min_cash_pct, max_trade_amount).
type Constraint struct {
	ID                 domain.ID `json:"id"`
	PortfolioProfileID domain.ID `json:"portfolio_profile_id"`
	ConstraintType     string    `json:"constraint_type"`
	ConstraintValue    string    `json:"constraint_value"`
	Enabled            bool      `json:"enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// PortfolioSnapshot is a point-in-time state of user assets and cash.
type PortfolioSnapshot struct {
	ID                  domain.ID `json:"id"`
	PortfolioProfileID  domain.ID `json:"portfolio_profile_id"`
	AsOf                time.Time `json:"as_of"`
	Source              string    `json:"source"`
	Currency            string    `json:"currency"`
	AvailableCash       int64     `json:"available_cash"`
	TotalPortfolioValue int64     `json:"total_portfolio_value"`
	ValidationStatus    string    `json:"validation_status"`
	MetadataJSON        string    `json:"metadata_json"`
	CreatedAt           time.Time `json:"created_at"`
}

// Position is an instrument holding within a snapshot.
type Position struct {
	ID                  domain.ID `json:"id"`
	PortfolioSnapshotID domain.ID `json:"portfolio_snapshot_id"`
	InstrumentID        domain.ID `json:"instrument_id"`
	Quantity            int64     `json:"quantity"`
	AverageCost         float64   `json:"average_cost"`
	LastKnownPrice      float64   `json:"last_known_price"`
	MarketValue         int64     `json:"market_value"`
	Currency            string    `json:"currency"`
	MetadataJSON        string    `json:"metadata_json"`
}

// Decision represents the research outcome produced by agy and recorded by Go.
type Decision struct {
	ID                         domain.ID    `json:"id"`
	PortfolioProfileID         domain.ID    `json:"portfolio_profile_id"`
	InstrumentID               domain.ID    `json:"instrument_id"`
	PortfolioSnapshotID        domain.ID    `json:"portfolio_snapshot_id"`
	TaskID                     domain.ID    `json:"task_id"`
	ResearchReportID           string       `json:"research_report_id"`
	DecisionType               DecisionType `json:"decision_type"`
	Confidence                 float64      `json:"confidence"`
	ReasoningJSON              string       `json:"reasoning_json"`
	RisksJSON                  string       `json:"risks_json"`
	InvalidationConditionsJSON string       `json:"invalidation_conditions_json"`
	ExpectedHorizon            string       `json:"expected_horizon"`
	StrategyVersion            string       `json:"strategy_version"`
	ActionResultJSON           string       `json:"action_result_json"`
	RawInputJSON               string       `json:"raw_input_json"`
	Source                     string       `json:"source"`
	CreatedAt                  time.Time    `json:"created_at"`
}

// DecisionRejection records invalid or rejected agy outputs.
type DecisionRejection struct {
	ID                 domain.ID `json:"id"`
	PortfolioProfileID domain.ID `json:"portfolio_profile_id"`
	TaskID             domain.ID `json:"task_id"`
	RawOutput          string    `json:"raw_output"`
	ErrorsJSON         string    `json:"errors_json"`
	Source             string    `json:"source"`
	CreatedAt          time.Time `json:"created_at"`
}

// TradeProposal is the Go-calculated, risk-checked trade recommendation.
// LLM (agy) NEVER produces TradeProposal directly. Go verifies limits,
// calculates quantities, computes fees and taxes, and produces this proposal.
type TradeProposal struct {
	ID                   domain.ID      `json:"id"`
	PortfolioProfileID   domain.ID      `json:"portfolio_profile_id"`
	DecisionID           domain.ID      `json:"decision_id"`
	PortfolioSnapshotID  domain.ID      `json:"portfolio_snapshot_id"`
	AllocationProposalID string         `json:"allocation_proposal_id"`
	InstrumentID         domain.ID      `json:"instrument_id"`
	Action               DecisionType   `json:"action"` // BUY or SELL
	Quantity             int64          `json:"quantity"`
	EstimatedPrice       float64        `json:"estimated_price"`
	EstimatedGrossAmount int64          `json:"estimated_gross_amount"`
	EstimatedTotalCost   int64          `json:"estimated_total_cost"` // Gross + Fee (for BUY) or Gross - Fee - Tax (for SELL proceeds)
	ProposalJSON         string         `json:"proposal_json"`
	Status               ProposalStatus `json:"status"`
	StatusReason         string         `json:"status_reason"`
	CreatedAt            time.Time      `json:"created_at"`
	ExpiresAt            time.Time      `json:"expires_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// CostBreakdown details the Go-calculated transaction fees and taxes.
type CostBreakdown struct {
	GrossAmount    int64   `json:"gross_amount"`
	BrokerFee      int64   `json:"broker_fee"`
	TransactionTax int64   `json:"transaction_tax"`
	TotalCost      int64   `json:"total_cost"` // Required cash for BUY; net proceeds for SELL
	EffectiveRate  float64 `json:"effective_rate"`
}

// AgyDecisionInput is the structured output received from Antigravity CLI / LLM.
type AgyDecisionInput struct {
	Symbol            string   `json:"symbol"`
	Exchange          string   `json:"exchange,omitempty"`
	Action            string   `json:"action"` // BUY, SELL, HOLD
	TargetAmount      int64    `json:"target_amount,omitempty"`
	Confidence        float64  `json:"confidence"`
	Reasoning         []string `json:"reasoning"`
	Risks             []string `json:"risks"`
	InvalidationRules []string `json:"invalidation_rules"`
	ExpectedHorizon   string   `json:"expected_horizon"`
	StrategyVersion   string   `json:"strategy_version"`
}
