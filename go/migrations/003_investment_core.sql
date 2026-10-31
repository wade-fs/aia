-- Migration 003: Investment Core tables
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
