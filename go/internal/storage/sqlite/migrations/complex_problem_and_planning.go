package migrations

// ComplexProblemAndPlanning establishes universal problem modeling, versioned plans, and analysis tracking (AIA-22 §6, §12).
const ComplexProblemAndPlanning = `
CREATE TABLE IF NOT EXISTS problems (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_id TEXT NOT NULL DEFAULT '',
    goal_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    objective TEXT NOT NULL DEFAULT '',
    scope TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'OPEN',
    priority TEXT NOT NULL DEFAULT 'MEDIUM',
    current_model_version INTEGER NOT NULL DEFAULT 0,
    current_plan_version INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_problems_project_id ON problems(project_id);
CREATE INDEX IF NOT EXISTS idx_problems_parent_id ON problems(parent_id);
CREATE INDEX IF NOT EXISTS idx_problems_status ON problems(status);

CREATE TABLE IF NOT EXISTS problem_models (
    id TEXT PRIMARY KEY,
    problem_id TEXT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    objectives_json TEXT NOT NULL DEFAULT '[]',
    constraints_json TEXT NOT NULL DEFAULT '[]',
    known_facts_json TEXT NOT NULL DEFAULT '[]',
    assumptions_json TEXT NOT NULL DEFAULT '[]',
    unknowns_json TEXT NOT NULL DEFAULT '[]',
    dependencies_json TEXT NOT NULL DEFAULT '[]',
    risks_json TEXT NOT NULL DEFAULT '[]',
    success_criteria_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    UNIQUE(problem_id, version)
);

CREATE INDEX IF NOT EXISTS idx_problem_models_problem ON problem_models(problem_id);

CREATE TABLE IF NOT EXISTS analysis_records (
    id TEXT PRIMARY KEY,
    problem_id TEXT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    model_version INTEGER NOT NULL,
    analysis_type TEXT NOT NULL,
    input_refs_json TEXT NOT NULL DEFAULT '[]',
    findings_json TEXT NOT NULL DEFAULT '[]',
    conclusions_json TEXT NOT NULL DEFAULT '[]',
    open_questions_json TEXT NOT NULL DEFAULT '[]',
    agent_id TEXT NOT NULL DEFAULT '',
    execution_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_analysis_records_problem ON analysis_records(problem_id);
CREATE INDEX IF NOT EXISTS idx_analysis_records_type ON analysis_records(analysis_type);

CREATE TABLE IF NOT EXISTS plans (
    id TEXT PRIMARY KEY,
    problem_id TEXT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    objective TEXT NOT NULL,
    strategy TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'DRAFT',
    work_items_json TEXT NOT NULL DEFAULT '[]',
    dependencies_json TEXT NOT NULL DEFAULT '[]',
    acceptance_criteria_json TEXT NOT NULL DEFAULT '[]',
    supersedes_plan_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(problem_id, version)
);

CREATE INDEX IF NOT EXISTS idx_plans_problem ON plans(problem_id);
CREATE INDEX IF NOT EXISTS idx_plans_status ON plans(status);

CREATE TABLE IF NOT EXISTS plan_changes (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    from_version INTEGER NOT NULL,
    to_version INTEGER NOT NULL,
    reason TEXT NOT NULL,
    evidence_refs_json TEXT NOT NULL DEFAULT '[]',
    affected_work_items_json TEXT NOT NULL DEFAULT '[]',
    cost_risk_impact TEXT NOT NULL DEFAULT '',
    requires_human_approval INTEGER NOT NULL DEFAULT 0,
    approved_by TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plan_changes_plan ON plan_changes(plan_id);
`
