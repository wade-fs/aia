package migrations

// Migration is a single versioned schema change.
// Migrations are applied in ascending Version order exactly once and recorded
// in schema_migrations. Existing migrations must never be edited after release;
// add a new version instead.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// All returns every known migration in ascending version order.
func All() []Migration {
	return []Migration{
		{Version: 1, Name: "initial_schema", SQL: InitialSchema},
		{Version: 2, Name: "audit_request_source_metadata", SQL: AuditRequestSourceMetadata},
		{Version: 3, Name: "investment_core", SQL: InvestmentCore},
		{Version: 4, Name: "investment_data_pipeline", SQL: InvestmentDataPipeline},
		{Version: 5, Name: "investment_predictions_and_evaluations", SQL: InvestmentPredictionsAndEvaluations},
		{Version: 6, Name: "schedules_and_automation", SQL: SchedulesAndAutomation},
		{Version: 7, Name: "event_history_and_provenance", SQL: EventHistoryAndProvenance},
		{Version: 8, Name: "goals_decisions_and_continuity", SQL: GoalsDecisionsAndContinuity},
		{Version: 9, Name: "governance_and_approvals", SQL: GovernanceAndApprovals},
		{Version: 10, Name: "initiatives_and_deliberation", SQL: InitiativesAndDeliberation},
		{Version: 11, Name: "learning_and_correction", SQL: LearningAndCorrection},
		{Version: 12, Name: "experiments_and_collaboration", SQL: ExperimentsAndCollaboration},
	}
}

// AuditRequestSourceMetadata adds request tracing, operation source and
// structured metadata to audit_logs (AIA-06 §8, AIA-03 §12).
const AuditRequestSourceMetadata = `
ALTER TABLE audit_logs ADD COLUMN request_id TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN source TEXT NOT NULL DEFAULT 'RUNTIME';
ALTER TABLE audit_logs ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_audit_request ON audit_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_source ON audit_logs(source);
`

// InvestmentCore creates the Investment Agent phase-1 tables (AIA-05 §18,
// milestones I-0 and I-4). Data collection / prediction tables are added by
// later migrations (P4/P5).
const InvestmentCore = `
CREATE TABLE IF NOT EXISTS investment_instruments (
    id TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    exchange TEXT NOT NULL,
    asset_type TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(exchange, symbol)
);

CREATE TABLE IF NOT EXISTS investment_portfolio_profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    market TEXT NOT NULL,
    currency TEXT NOT NULL,
    capital_limit INTEGER NOT NULL,
    capital_limit_basis TEXT NOT NULL,
    auto_trading_enabled INTEGER NOT NULL DEFAULT 0 CHECK (auto_trading_enabled = 0),
    configuration_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS investment_watchlists (
    id TEXT PRIMARY KEY,
    portfolio_profile_id TEXT NOT NULL REFERENCES investment_portfolio_profiles(id),
    instrument_id TEXT NOT NULL REFERENCES investment_instruments(id),
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    UNIQUE(portfolio_profile_id, instrument_id)
);

CREATE TABLE IF NOT EXISTS investment_portfolio_constraints (
    id TEXT PRIMARY KEY,
    portfolio_profile_id TEXT NOT NULL REFERENCES investment_portfolio_profiles(id),
    constraint_type TEXT NOT NULL,
    constraint_value TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(portfolio_profile_id, constraint_type)
);

CREATE TABLE IF NOT EXISTS investment_portfolio_snapshots (
    id TEXT PRIMARY KEY,
    portfolio_profile_id TEXT NOT NULL REFERENCES investment_portfolio_profiles(id),
    as_of TEXT NOT NULL,
    source TEXT NOT NULL,
    currency TEXT NOT NULL,
    available_cash INTEGER NOT NULL,
    total_portfolio_value INTEGER NOT NULL,
    validation_status TEXT NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS investment_portfolio_positions (
    id TEXT PRIMARY KEY,
    portfolio_snapshot_id TEXT NOT NULL REFERENCES investment_portfolio_snapshots(id),
    instrument_id TEXT NOT NULL REFERENCES investment_instruments(id),
    quantity INTEGER NOT NULL,
    average_cost REAL NOT NULL,
    last_known_price REAL NOT NULL DEFAULT 0,
    market_value INTEGER NOT NULL,
    currency TEXT NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS investment_portfolio_open_orders (
    id TEXT PRIMARY KEY,
    portfolio_snapshot_id TEXT NOT NULL REFERENCES investment_portfolio_snapshots(id),
    instrument_id TEXT NOT NULL REFERENCES investment_instruments(id),
    side TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    limit_price REAL NOT NULL,
    status TEXT NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS investment_decisions (
    id TEXT PRIMARY KEY,
    portfolio_profile_id TEXT NOT NULL REFERENCES investment_portfolio_profiles(id),
    instrument_id TEXT NOT NULL REFERENCES investment_instruments(id),
    portfolio_snapshot_id TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    research_report_id TEXT NOT NULL DEFAULT '',
    decision_type TEXT NOT NULL,
    confidence REAL NOT NULL,
    reasoning_json TEXT NOT NULL,
    risks_json TEXT NOT NULL,
    invalidation_conditions_json TEXT NOT NULL,
    expected_horizon TEXT NOT NULL,
    strategy_version TEXT NOT NULL,
    action_result_json TEXT NOT NULL DEFAULT '{}',
    raw_input_json TEXT NOT NULL DEFAULT '{}',
    source TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS investment_decision_evidence (
    id TEXT PRIMARY KEY,
    decision_id TEXT NOT NULL REFERENCES investment_decisions(id),
    raw_document_id TEXT NOT NULL DEFAULT '',
    snapshot_item_id TEXT NOT NULL DEFAULT '',
    evidence_type TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    relevance REAL NOT NULL DEFAULT 0,
    explanation TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS investment_decision_rejections (
    id TEXT PRIMARY KEY,
    portfolio_profile_id TEXT NOT NULL,
    task_id TEXT NOT NULL DEFAULT '',
    raw_output TEXT NOT NULL,
    errors_json TEXT NOT NULL,
    source TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS investment_trade_proposals (
    id TEXT PRIMARY KEY,
    portfolio_profile_id TEXT NOT NULL REFERENCES investment_portfolio_profiles(id),
    decision_id TEXT NOT NULL REFERENCES investment_decisions(id),
    portfolio_snapshot_id TEXT NOT NULL REFERENCES investment_portfolio_snapshots(id),
    allocation_proposal_id TEXT NOT NULL DEFAULT '',
    instrument_id TEXT NOT NULL REFERENCES investment_instruments(id),
    action TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    estimated_price REAL NOT NULL,
    estimated_gross_amount INTEGER NOT NULL,
    estimated_total_cost INTEGER NOT NULL,
    proposal_json TEXT NOT NULL,
    status TEXT NOT NULL,
    status_reason TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_inv_snapshots_profile ON investment_portfolio_snapshots(portfolio_profile_id, as_of);
CREATE INDEX IF NOT EXISTS idx_inv_decisions_profile ON investment_decisions(portfolio_profile_id, created_at);
CREATE INDEX IF NOT EXISTS idx_inv_proposals_profile ON investment_trade_proposals(portfolio_profile_id, status);
`

// InvestmentDataPipeline adds data sources, market bar data, and raw research documents (AIA-05 §18.1, §18.2).
const InvestmentDataPipeline = `
CREATE TABLE IF NOT EXISTS investment_data_sources (
    id TEXT PRIMARY KEY,
    source_name TEXT NOT NULL UNIQUE,
    source_type TEXT NOT NULL,
    base_url TEXT NOT NULL DEFAULT '',
    documentation_url TEXT NOT NULL DEFAULT '',
    authentication_type TEXT NOT NULL DEFAULT 'NONE',
    configuration_json TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    health_status TEXT NOT NULL DEFAULT 'HEALTHY',
    last_success_at TEXT,
    last_failure_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS investment_market_data (
    id TEXT PRIMARY KEY,
    instrument_id TEXT NOT NULL REFERENCES investment_instruments(id),
    data_type TEXT NOT NULL,
    period TEXT NOT NULL,
    value_json TEXT NOT NULL,
    source_id TEXT NOT NULL REFERENCES investment_data_sources(id),
    fetched_at TEXT NOT NULL,
    effective_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS investment_raw_documents (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES investment_data_sources(id),
    instrument_id TEXT NOT NULL DEFAULT '',
    source_record_id TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    content_reference TEXT NOT NULL DEFAULT '',
    content_hash TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT 'NEWS',
    published_at TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_inv_market_data_inst ON investment_market_data(instrument_id, effective_at);
CREATE INDEX IF NOT EXISTS idx_inv_docs_inst ON investment_raw_documents(instrument_id, published_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_docs_hash ON investment_raw_documents(source_id, content_hash);

CREATE TABLE IF NOT EXISTS investment_data_snapshots (
    id TEXT PRIMARY KEY,
    snapshot_type TEXT NOT NULL,
    data_cutoff_at TEXT NOT NULL,
    instrument_scope_json TEXT NOT NULL DEFAULT '[]',
    source_versions_json TEXT NOT NULL DEFAULT '{}',
    validation_summary_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS investment_snapshot_items (
    id TEXT PRIMARY KEY,
    snapshot_id TEXT NOT NULL REFERENCES investment_data_snapshots(id),
    item_type TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_inv_snapshots_cutoff ON investment_data_snapshots(data_cutoff_at);
CREATE INDEX IF NOT EXISTS idx_inv_snapshot_items_ref ON investment_snapshot_items(snapshot_id, item_type);
`

// InvestmentPredictionsAndEvaluations adds predictions, outcome verification, and calibration metrics (AIA-05 §18.5).
const InvestmentPredictionsAndEvaluations = `
CREATE TABLE IF NOT EXISTS investment_predictions (
    id TEXT PRIMARY KEY,
    decision_id TEXT NOT NULL REFERENCES investment_decisions(id),
    instrument_id TEXT NOT NULL REFERENCES investment_instruments(id),
    prediction_type TEXT NOT NULL,
    target TEXT NOT NULL,
    expected_direction TEXT NOT NULL,
    probability REAL NOT NULL,
    target_value REAL NOT NULL DEFAULT 0,
    base_price REAL NOT NULL DEFAULT 0,
    evaluation_horizon TEXT NOT NULL,
    due_at TEXT NOT NULL,
    strategy_version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS investment_prediction_outcomes (
    id TEXT PRIMARY KEY,
    prediction_id TEXT NOT NULL REFERENCES investment_predictions(id),
    actual_value REAL NOT NULL,
    actual_direction TEXT NOT NULL,
    outcome_status TEXT NOT NULL,
    brier_score REAL NOT NULL,
    evaluation_source_id TEXT NOT NULL DEFAULT '',
    evaluated_at TEXT NOT NULL,
    details_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS investment_evaluation_results (
    id TEXT PRIMARY KEY,
    evaluation_type TEXT NOT NULL,
    period_start TEXT NOT NULL,
    period_end TEXT NOT NULL,
    sample_count INTEGER NOT NULL,
    metrics_json TEXT NOT NULL,
    strategy_version TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_inv_pred_due ON investment_predictions(status, due_at);
CREATE INDEX IF NOT EXISTS idx_inv_pred_inst ON investment_predictions(instrument_id, created_at);
`

// SchedulesAndAutomation supports cron-scheduled recurring tasks and misfire handling (AIA-03 §10, AIA-07 §14).
const SchedulesAndAutomation = `
CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    task_template_json TEXT NOT NULL,
    misfire_policy TEXT NOT NULL DEFAULT 'SKIP',
    is_active INTEGER NOT NULL DEFAULT 1,
    last_run_at TEXT,
    next_run_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_schedules_next ON schedules(is_active, next_run_at);
`
