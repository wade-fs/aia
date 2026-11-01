package migrations

// InitiativesAndDeliberation defines migration 010 (AIA-ARCH-DATA-01, Phase 3).
//
// It establishes:
//   - candidate_initiatives: autonomous discovery of issues, opportunities, goal gaps,
//     and continuity items without direct human prompts (AIA-09 §3–5, Plan Phase 3).
//   - deliberation_records: structured deliberation option matrix, expected-value
//     cost-benefit trade-offs, risk assessment, and justified recommendation across
//     ACT, DEFER, MONITOR, ASK_HUMAN, and NO_ACTION (AIA-08 §4–5, AIA-09 §6–8).
//   - goal_task_links: traceable Goal-to-Task decomposition mapping (Plan Phase 3 §4.3).
const InitiativesAndDeliberation = `
CREATE TABLE IF NOT EXISTS candidate_initiatives (
    id TEXT PRIMARY KEY,
    cycle_id TEXT NOT NULL REFERENCES autonomy_cycles(id),
    project_id TEXT REFERENCES projects(id),
    goal_id TEXT REFERENCES goals(id),
    discovery_type TEXT NOT NULL DEFAULT 'OPPORTUNITY' CHECK (discovery_type IN ('ISSUE', 'OPPORTUNITY', 'GOAL_GAP', 'CONTINUITY', 'IMPROVEMENT')),
    confidence_level TEXT NOT NULL DEFAULT 'PROBABLE' CHECK (confidence_level IN ('CONFIRMED', 'PROBABLE', 'POTENTIAL', 'UNVERIFIED')),
    target_agent_id TEXT NOT NULL DEFAULT 'coding-agent',
    trigger_source TEXT NOT NULL,
    detected_opportunity TEXT NOT NULL,
    rationale TEXT NOT NULL,
    expected_outcome TEXT NOT NULL DEFAULT '',
    priority_score INTEGER NOT NULL DEFAULT 5 CHECK (priority_score BETWEEN 1 AND 10),
    estimated_cost REAL NOT NULL DEFAULT 1.0,
    estimated_risk TEXT NOT NULL DEFAULT 'LOW' CHECK (estimated_risk IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    status TEXT NOT NULL DEFAULT 'CANDIDATE' CHECK (status IN ('CANDIDATE', 'ADOPTED', 'DISCARDED', 'DEFERRED', 'MONITORING')),
    discard_reason TEXT,
    spawned_task_id TEXT REFERENCES tasks(id),
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_initiatives_cycle ON candidate_initiatives(cycle_id);
CREATE INDEX IF NOT EXISTS idx_initiatives_status ON candidate_initiatives(status);
CREATE INDEX IF NOT EXISTS idx_initiatives_project ON candidate_initiatives(project_id);
CREATE INDEX IF NOT EXISTS idx_initiatives_goal ON candidate_initiatives(goal_id);

CREATE TABLE IF NOT EXISTS deliberation_records (
    id TEXT PRIMARY KEY,
    cycle_id TEXT NOT NULL REFERENCES autonomy_cycles(id),
    decision_id TEXT REFERENCES decisions(id),
    candidate_initiative_id TEXT REFERENCES candidate_initiatives(id),
    options_evaluated_json TEXT NOT NULL,
    cost_benefit_analysis TEXT NOT NULL,
    risk_assessment TEXT NOT NULL,
    recommended_action TEXT NOT NULL CHECK (recommended_action IN ('ACT', 'DEFER', 'MONITOR', 'ASK_HUMAN', 'NO_ACTION')),
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_deliberation_cycle ON deliberation_records(cycle_id);
CREATE INDEX IF NOT EXISTS idx_deliberation_decision ON deliberation_records(decision_id);
CREATE INDEX IF NOT EXISTS idx_deliberation_initiative ON deliberation_records(candidate_initiative_id);

CREATE TABLE IF NOT EXISTS goal_task_links (
    goal_id TEXT NOT NULL REFERENCES goals(id),
    task_id TEXT NOT NULL REFERENCES tasks(id),
    cycle_id TEXT REFERENCES autonomy_cycles(id),
    decomposition_rationale TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    PRIMARY KEY (goal_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_goal_task_links_task ON goal_task_links(task_id);
`
