package migrations

// GenericBriefAndHistory establishes universal project brief storage and immutable field history (AIA-21 §4, §12).
const GenericBriefAndHistory = `
CREATE TABLE IF NOT EXISTS project_briefs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    domain TEXT NOT NULL,
    brief_version INTEGER NOT NULL DEFAULT 1,
    readiness_status TEXT NOT NULL DEFAULT 'NOT_READY',
    raw_intent TEXT NOT NULL DEFAULT '',
    brief_yaml TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(project_id)
);

CREATE TABLE IF NOT EXISTS brief_field_history (
    id TEXT PRIMARY KEY,
    brief_id TEXT NOT NULL REFERENCES project_briefs(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    field_key TEXT NOT NULL,
    version INTEGER NOT NULL,
    old_value_json TEXT,
    new_value_json TEXT,
    old_status TEXT NOT NULL,
    new_status TEXT NOT NULL,
    source TEXT NOT NULL,
    rationale TEXT NOT NULL,
    related_task_id TEXT,
    related_clarification_id TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_brief_proj ON project_briefs(project_id);
CREATE INDEX IF NOT EXISTS idx_brief_hist_key ON brief_field_history(project_id, field_key);
`
