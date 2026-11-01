package migrations

// ExperimentsAndCollaboration (Migration 012 — Phase 5) establishes:
// 1. experiments & experiment_goal_links (AIA-15 11-state lifecycle)
// 2. experiment_generations (G1 frozen rule-based baseline vs G2 / G2.x LLM generations)
//    with DB-level immutability triggers protecting frozen baseline cohorts
// 3. experiment_comparisons (Track A portfolio performance + Track B autonomy capability
//    + 5-regime scientific interpretation per AIA-13 §21.4)
// 4. delegation_requests & tasks.experiment_id (AIA-12 cross-domain collaboration)
// 5. Dual forecast / outcome fields on investment_predictions & investment_prediction_outcomes
//    (expected_return_pct, expected_range_low_pct, expected_range_high_pct, log_loss)
const ExperimentsAndCollaboration = `
ALTER TABLE tasks ADD COLUMN experiment_id TEXT REFERENCES experiments(id);

ALTER TABLE investment_predictions ADD COLUMN expected_return_pct REAL NOT NULL DEFAULT 0;
ALTER TABLE investment_predictions ADD COLUMN expected_range_low_pct REAL NOT NULL DEFAULT 0;
ALTER TABLE investment_predictions ADD COLUMN expected_range_high_pct REAL NOT NULL DEFAULT 0;

ALTER TABLE investment_prediction_outcomes ADD COLUMN log_loss REAL NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS experiments (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id),
    primary_goal_id TEXT REFERENCES goals(id),
    domain TEXT NOT NULL DEFAULT 'investment',
    owner_agent_id TEXT NOT NULL DEFAULT 'investment-agent',
    name TEXT NOT NULL,
    hypothesis TEXT NOT NULL,
    experiment_type TEXT NOT NULL DEFAULT 'GENERATIONAL_COMPARISON',
    status TEXT NOT NULL DEFAULT 'PROPOSED' CHECK (status IN (
        'PROPOSED',
        'PENDING_APPROVAL',
        'READY',
        'RUNNING',
        'PAUSED',
        'EVALUATING',
        'NEEDS_REVISION',
        'COMPLETED',
        'STOPPED',
        'FAILED',
        'ARCHIVED'
    )),
    baseline_metrics_json TEXT NOT NULL DEFAULT '{}',
    current_metrics_json TEXT NOT NULL DEFAULT '{}',
    iteration_count INTEGER NOT NULL DEFAULT 0 CHECK (iteration_count >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS experiment_goal_links (
    experiment_id TEXT NOT NULL REFERENCES experiments(id),
    goal_id TEXT NOT NULL REFERENCES goals(id),
    role TEXT NOT NULL DEFAULT 'PRIMARY',
    created_at TEXT NOT NULL,
    PRIMARY KEY (experiment_id, goal_id)
);

CREATE TABLE IF NOT EXISTS experiment_generations (
    id TEXT PRIMARY KEY,
    experiment_id TEXT NOT NULL REFERENCES experiments(id),
    generation_code TEXT NOT NULL,
    parent_generation_id TEXT REFERENCES experiment_generations(id),
    role TEXT NOT NULL CHECK (role IN ('CONTROL_BASELINE', 'EXPERIMENTAL')),
    engine_type TEXT NOT NULL CHECK (engine_type IN ('RULE_BASED', 'LLM_AGY', 'HYBRID')),
    portfolio_profile_id TEXT NOT NULL REFERENCES investment_portfolio_profiles(id),
    is_frozen INTEGER NOT NULL DEFAULT 0 CHECK (is_frozen IN (0, 1)),
    harness_version TEXT NOT NULL DEFAULT '',
    model_id TEXT NOT NULL DEFAULT '',
    prompt_version TEXT NOT NULL DEFAULT '',
    strategy_version TEXT NOT NULL,
    strategy_config_json TEXT NOT NULL DEFAULT '{}',
    source_correction_id TEXT REFERENCES correction_actions(id),
    source_learning_id TEXT REFERENCES learning_records(id),
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'PROMOTED', 'ROLLED_BACK', 'RETIRED')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(experiment_id, generation_code),
    UNIQUE(experiment_id, portfolio_profile_id),
    CHECK (role != 'CONTROL_BASELINE' OR is_frozen = 1)
);

CREATE TRIGGER IF NOT EXISTS trg_exp_gen_freeze_delete
BEFORE DELETE ON experiment_generations
WHEN OLD.is_frozen = 1
BEGIN
    SELECT RAISE(ABORT, 'immutable frozen baseline generation cannot be deleted');
END;

CREATE TRIGGER IF NOT EXISTS trg_exp_gen_freeze_update
BEFORE UPDATE ON experiment_generations
WHEN OLD.is_frozen = 1 AND (
    NEW.is_frozen != 1 OR
    NEW.generation_code != OLD.generation_code OR
    NEW.role != OLD.role OR
    NEW.engine_type != OLD.engine_type OR
    NEW.portfolio_profile_id != OLD.portfolio_profile_id OR
    NEW.strategy_version != OLD.strategy_version OR
    NEW.strategy_config_json != OLD.strategy_config_json OR
    NEW.status != OLD.status
)
BEGIN
    SELECT RAISE(ABORT, 'immutable frozen baseline generation cannot be modified');
END;

CREATE TABLE IF NOT EXISTS experiment_comparisons (
    id TEXT PRIMARY KEY,
    experiment_id TEXT NOT NULL REFERENCES experiments(id),
    baseline_generation_id TEXT NOT NULL REFERENCES experiment_generations(id),
    candidate_generation_id TEXT NOT NULL REFERENCES experiment_generations(id),
    as_of TEXT NOT NULL,
    sample_count INTEGER NOT NULL DEFAULT 0 CHECK (sample_count >= 0),
    performance_comparison_json TEXT NOT NULL,
    autonomy_comparison_json TEXT NOT NULL,
    regime_verdict TEXT NOT NULL CHECK (regime_verdict IN (
        'G2_CLEAR_ADVANTAGE',
        'G2_UNRELIABLE_RISK_TAKING',
        'G2_LEARNING_IN_PROGRESS',
        'G1_RULES_SUPERIOR',
        'PARITY_ADAPTIVE',
        'INCONCLUSIVE_LOW_SAMPLE'
    )),
    summary TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS delegation_requests (
    id TEXT PRIMARY KEY,
    experiment_id TEXT REFERENCES experiments(id),
    requester_agent_id TEXT NOT NULL,
    target_agent_id TEXT NOT NULL,
    source_project_id TEXT REFERENCES projects(id),
    goal_id TEXT REFERENCES goals(id),
    spawned_task_id TEXT REFERENCES tasks(id),
    task_spec_json TEXT NOT NULL,
    acceptance_gate_json TEXT NOT NULL DEFAULT '{}',
    deliverable_json TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'ACCEPTED', 'IN_PROGRESS', 'DELIVERED', 'VERIFIED', 'REJECTED', 'FAILED')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_experiments_project_status ON experiments(project_id, status);
CREATE INDEX IF NOT EXISTS idx_exp_generations_exp ON experiment_generations(experiment_id, generation_code);
CREATE INDEX IF NOT EXISTS idx_exp_comparisons_exp ON experiment_comparisons(experiment_id, created_at);
CREATE INDEX IF NOT EXISTS idx_delegation_status ON delegation_requests(status, target_agent_id);
`
