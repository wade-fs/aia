package migrations

// LearningAndCorrection defines migration 011 (AIA-ARCH-DATA-01 §3.6E, AIA-09 §25, AIA-11 §21, AIA-13 §21).
//
// It establishes:
//   - Token usage columns on executions (input_tokens, output_tokens, total_tokens) for hard budget gating.
//   - outcome_evaluations: Expected vs Actual discrepancy measurement.
//   - root_cause_analyses: Evidence-grounded, falsifiable RCA hypotheses decoupled from intervention success.
//   - correction_actions: Orthogonal Scope (A/B/C), Correction Type, Authorization (C0-C5), Validation (V1-V6),
//     collapsed into runtime 3-tier profiles (TIER1_AUTO, TIER2_APPROVAL, TIER3_BLOCKED) with regression tracking.
//   - learning_records: L0-L3 learning maturity gates with 7 mandatory epistemic comparison fields
//     (Baseline, Hypothesis, Intervention, Verification, Regression, Follow-up, Confidence) and INCONCLUSIVE support.
//   - goal_evolution_proposals: Governed Goal Persistence / Optimization / Revision proposals enforcing
//     Anti-Goalpost-Moving invariants.
const LearningAndCorrection = `
ALTER TABLE executions ADD COLUMN input_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE executions ADD COLUMN output_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE executions ADD COLUMN total_tokens INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_executions_created_tokens ON executions(created_at, total_tokens);

CREATE TABLE IF NOT EXISTS outcome_evaluations (
    id TEXT PRIMARY KEY,
    cycle_id TEXT REFERENCES autonomy_cycles(id),
    decision_id TEXT REFERENCES decisions(id),
    task_id TEXT REFERENCES tasks(id),
    project_id TEXT REFERENCES projects(id),
    domain TEXT NOT NULL DEFAULT 'general',
    expected_outcome_json TEXT NOT NULL,
    actual_outcome_json TEXT NOT NULL,
    discrepancy_score REAL NOT NULL DEFAULT 0.0,
    is_anomaly INTEGER NOT NULL DEFAULT 0 CHECK (is_anomaly IN (0, 1)),
    evaluation_status TEXT NOT NULL DEFAULT 'EVALUATED' CHECK (evaluation_status IN ('EVALUATED', 'NEEDS_RCA', 'RESOLVED')),
    evaluated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_eval_discrepancy ON outcome_evaluations(discrepancy_score);
CREATE INDEX IF NOT EXISTS idx_eval_cycle ON outcome_evaluations(cycle_id);
CREATE INDEX IF NOT EXISTS idx_eval_project ON outcome_evaluations(project_id);

CREATE TABLE IF NOT EXISTS root_cause_analyses (
    id TEXT PRIMARY KEY,
    outcome_evaluation_id TEXT NOT NULL REFERENCES outcome_evaluations(id),
    root_cause_category TEXT NOT NULL CHECK (root_cause_category IN ('INCORRECT_FACT', 'BIASED_JUDGMENT', 'FLAWED_STRATEGY', 'AMBIGUOUS_PROMPT', 'WORKFLOW_DEFECT', 'EXTERNAL_UNEXPECTED')),
    analysis_narrative TEXT NOT NULL,
    falsifiable_hypothesis TEXT NOT NULL,
    causal_evidence_refs_json TEXT NOT NULL DEFAULT '[]',
    competing_factors_json TEXT NOT NULL DEFAULT '[]',
    rca_validity_status TEXT NOT NULL DEFAULT 'PROPOSED' CHECK (rca_validity_status IN ('PROPOSED', 'SUPPORTED', 'FALSIFIED', 'INTERVENTION_MISMATCH', 'UNCERTAIN')),
    confidence REAL NOT NULL CHECK (confidence BETWEEN 0.0 AND 1.0),
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rca_evaluation ON root_cause_analyses(outcome_evaluation_id);

CREATE TABLE IF NOT EXISTS correction_actions (
    id TEXT PRIMARY KEY,
    rca_id TEXT NOT NULL REFERENCES root_cause_analyses(id),
    scope_level TEXT NOT NULL DEFAULT 'A' CHECK (scope_level IN ('A', 'B', 'C')),
    correction_type TEXT NOT NULL CHECK (correction_type IN ('FACT', 'JUDGMENT', 'STRATEGY', 'PROMPT', 'WORKFLOW', 'CODE')),
    authorization_level TEXT NOT NULL CHECK (authorization_level IN ('C0', 'C1', 'C2', 'C3', 'C4', 'C5')),
    validation_level TEXT NOT NULL DEFAULT 'V2' CHECK (validation_level IN ('V1', 'V2', 'V3', 'V4', 'V5', 'V6')),
    tier_profile TEXT NOT NULL DEFAULT 'TIER1_AUTO' CHECK (tier_profile IN ('TIER1_AUTO', 'TIER2_APPROVAL', 'TIER3_BLOCKED')),
    target_entity_ref TEXT NOT NULL,
    before_state_json TEXT NOT NULL,
    proposed_state_json TEXT NOT NULL,
    applied_state_json TEXT,
    approval_id TEXT REFERENCES approvals(id),
    regression_status TEXT NOT NULL DEFAULT 'PENDING' CHECK (regression_status IN ('PENDING', 'PASSED', 'FAILED')),
    verification_status TEXT NOT NULL CHECK (verification_status IN ('PENDING', 'APPLIED', 'VERIFIED_EFFECTIVE', 'VERIFIED_INEFFECTIVE', 'INCONCLUSIVE', 'REGRESSED', 'ROLLED_BACK', 'BLOCKED')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_corrections_rca ON correction_actions(rca_id);
CREATE INDEX IF NOT EXISTS idx_corrections_status ON correction_actions(verification_status);

CREATE TABLE IF NOT EXISTS learning_records (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id),
    domain TEXT NOT NULL DEFAULT 'general',
    source_cycle_id TEXT REFERENCES autonomy_cycles(id),
    outcome_evaluation_id TEXT NOT NULL REFERENCES outcome_evaluations(id),
    rca_id TEXT REFERENCES root_cause_analyses(id),
    correction_action_id TEXT REFERENCES correction_actions(id),
    title TEXT NOT NULL,
    learning_level TEXT NOT NULL CHECK (learning_level IN ('L0_RECORDED', 'L1_APPLIED', 'L2_VERIFIED', 'L3_GENERALIZED')),
    effectiveness_verdict TEXT NOT NULL CHECK (effectiveness_verdict IN ('PENDING', 'EFFECTIVE', 'INEFFECTIVE', 'REGRESSED', 'INCONCLUSIVE')),
    comparison_method TEXT NOT NULL CHECK (comparison_method IN ('CONTROL_BENCHMARK', 'PIT_REPLAY', 'SIMILAR_COHORT', 'PRE_POST_ONLY')),
    baseline_json TEXT NOT NULL,
    hypothesis_json TEXT NOT NULL,
    intervention_json TEXT NOT NULL DEFAULT '{}',
    verification_json TEXT NOT NULL DEFAULT '{}',
    regression_json TEXT NOT NULL DEFAULT '{}',
    followup_json TEXT NOT NULL DEFAULT '{}',
    epistemic_confidence REAL NOT NULL CHECK (epistemic_confidence BETWEEN 0.0 AND 1.0),
    generalized_cycle_ids_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_learning_level_domain ON learning_records(domain, learning_level, effectiveness_verdict);
CREATE INDEX IF NOT EXISTS idx_learning_project ON learning_records(project_id);

CREATE TABLE IF NOT EXISTS goal_evolution_proposals (
    id TEXT PRIMARY KEY,
    goal_id TEXT NOT NULL REFERENCES goals(id),
    cycle_id TEXT REFERENCES autonomy_cycles(id),
    evolution_mode TEXT NOT NULL CHECK (evolution_mode IN ('PERSISTENCE', 'OPTIMIZATION', 'REVISION')),
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    rationale TEXT NOT NULL,
    previous_target_metric_json TEXT NOT NULL DEFAULT '{}',
    proposed_target_metric_json TEXT NOT NULL DEFAULT '{}',
    invalidated_claim_ids_json TEXT NOT NULL DEFAULT '[]',
    resource_constraint_note TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('APPLIED', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED_GOALPOST_MOVING', 'REJECTED')),
    approval_id TEXT REFERENCES approvals(id),
    resulting_revision_id TEXT REFERENCES goal_revisions(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_goal_evolution_goal ON goal_evolution_proposals(goal_id, status);
`
