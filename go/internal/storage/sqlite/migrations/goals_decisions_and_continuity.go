package migrations

// GoalsDecisionsAndContinuity defines migration 008 (AIA-ARCH-DATA-01, Phase 1).
// It establishes the autonomy_cycles temporal hub, goals hierarchy and revisions,
// goal_project_links (M:N), projects context expansion, context_snapshots (global/project),
// and decisions with decision_evidence_links supporting NO_ACTION.
const GoalsDecisionsAndContinuity = `
CREATE TABLE IF NOT EXISTS autonomy_cycles (
    id TEXT PRIMARY KEY,
    cycle_number INTEGER NOT NULL,
    trigger_type TEXT NOT NULL CHECK (trigger_type IN ('SCHEDULED', 'EVENT_TRIGGERED', 'MANUAL', 'DRIFT_DETECTED')),
    trigger_ref_id TEXT,
    status TEXT NOT NULL CHECK (status IN ('STARTED', 'DELIBERATING', 'EXECUTING', 'EVALUATING', 'COMPLETED', 'ABORTED')),
    context_snapshot_id TEXT,
    summary TEXT,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cycles_status ON autonomy_cycles(status);
CREATE INDEX IF NOT EXISTS idx_cycles_started ON autonomy_cycles(started_at);

CREATE TABLE IF NOT EXISTS goals (
    id TEXT PRIMARY KEY,
    parent_goal_id TEXT,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    level TEXT NOT NULL CHECK (level IN ('STRATEGIC', 'TACTICAL', 'OPERATIONAL')),
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'ACTIVE', 'PAUSED', 'ACHIEVED', 'SUPERSEDED', 'ABANDONED')),
    target_metric_json TEXT,
    current_metric_json TEXT,
    priority INTEGER NOT NULL DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (parent_goal_id) REFERENCES goals(id)
);

CREATE INDEX IF NOT EXISTS idx_goals_status ON goals(status);

CREATE TABLE IF NOT EXISTS goal_revisions (
    id TEXT PRIMARY KEY,
    goal_id TEXT NOT NULL,
    revision_number INTEGER NOT NULL,
    previous_state_json TEXT NOT NULL,
    new_state_json TEXT NOT NULL,
    reason TEXT NOT NULL,
    decision_id TEXT,
    created_at TEXT NOT NULL,
    UNIQUE(goal_id, revision_number),
    FOREIGN KEY (goal_id) REFERENCES goals(id)
);

CREATE TABLE IF NOT EXISTS goal_project_links (
    goal_id    TEXT NOT NULL REFERENCES goals(id),
    project_id TEXT NOT NULL REFERENCES projects(id),
    link_type  TEXT NOT NULL DEFAULT 'CONTRIBUTES_TO' CHECK (link_type IN ('CONTRIBUTES_TO', 'VALIDATES', 'DERIVED_FROM')),
    created_at TEXT NOT NULL,
    PRIMARY KEY (goal_id, project_id)
);

CREATE TABLE IF NOT EXISTS context_snapshots (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL CHECK (scope IN ('GLOBAL', 'PROJECT')),
    project_id TEXT REFERENCES projects(id),
    snapshot_type TEXT NOT NULL CHECK (snapshot_type IN ('MANUAL', 'PRE_RESTART', 'POST_CYCLE', 'EMERGENCY_STOP')),
    workspace_state_json TEXT NOT NULL,
    active_tasks_json TEXT NOT NULL,
    recent_decisions_json TEXT NOT NULL,
    pending_issues_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_snapshots_project ON context_snapshots(project_id, created_at);

CREATE TABLE IF NOT EXISTS decisions (
    id TEXT PRIMARY KEY,
    cycle_id TEXT REFERENCES autonomy_cycles(id),
    project_id TEXT REFERENCES projects(id),
    experiment_id TEXT,
    action_type TEXT NOT NULL CHECK (action_type IN ('ACT', 'DEFER', 'MONITOR', 'ASK_HUMAN', 'NO_ACTION')),
    intent_summary TEXT NOT NULL,
    justification TEXT NOT NULL,
    considered_alternatives_json TEXT,
    context_snapshot_id TEXT REFERENCES context_snapshots(id),
    governance_result TEXT NOT NULL CHECK (governance_result IN ('ALLOWED', 'REQUIRED_APPROVAL', 'BLOCKED')),
    action_task_id TEXT REFERENCES tasks(id),
    action_execution_id TEXT REFERENCES executions(id),
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_decisions_cycle ON decisions(cycle_id);
CREATE INDEX IF NOT EXISTS idx_decisions_action ON decisions(action_type);

CREATE TABLE IF NOT EXISTS decision_evidence_links (
    decision_id TEXT NOT NULL REFERENCES decisions(id),
    evidence_id TEXT NOT NULL REFERENCES evidence(id),
    relevance_note TEXT,
    PRIMARY KEY (decision_id, evidence_id)
);

ALTER TABLE projects ADD COLUMN primary_goal_id TEXT REFERENCES goals(id);
ALTER TABLE projects ADD COLUMN latest_snapshot_id TEXT REFERENCES context_snapshots(id);
ALTER TABLE projects ADD COLUMN pending_issues_json TEXT NOT NULL DEFAULT '[]';
`
