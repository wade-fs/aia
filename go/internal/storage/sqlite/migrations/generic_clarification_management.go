package migrations

// GenericClarificationManagement establishes universal clarification request tracking (AIA-21 §5, §12).
const GenericClarificationManagement = `
CREATE TABLE IF NOT EXISTS clarification_requests (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id TEXT,
    field_key TEXT NOT NULL,
    question TEXT NOT NULL,
    options_json TEXT NOT NULL DEFAULT '[]',
    reason TEXT NOT NULL,
    impact TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 1,
    blocking INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'OPEN',
    answer TEXT,
    answered_at TEXT,
    expires_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_clarify_proj_status ON clarification_requests(project_id, status);
CREATE INDEX IF NOT EXISTS idx_clarify_proj_field ON clarification_requests(project_id, field_key);
`
