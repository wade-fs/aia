package migrations

// WritingBriefAndClarification creates schemas for CreativeBrief and ClarificationRequest (Writing Domain workflow).
const WritingBriefAndClarification = `
CREATE TABLE IF NOT EXISTS writing_briefs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    title TEXT NOT NULL,
    raw_intent TEXT NOT NULL,
    genre TEXT NOT NULL,
    era TEXT NOT NULL DEFAULT '',
    theme_and_philosophy_json TEXT NOT NULL DEFAULT '[]',
    scale_json TEXT NOT NULL DEFAULT '{}',
    core_protagonists_json TEXT NOT NULL DEFAULT '[]',
    worldview_settings_json TEXT NOT NULL DEFAULT '{}',
    writing_rules_json TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL,
    pending_questions INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_writing_briefs_project ON writing_briefs(project_id);
CREATE INDEX IF NOT EXISTS idx_writing_briefs_status ON writing_briefs(status);

CREATE TABLE IF NOT EXISTS writing_clarifications (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    brief_id TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL,
    question TEXT NOT NULL,
    rationale TEXT NOT NULL,
    options_json TEXT NOT NULL DEFAULT '[]',
    user_answer TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    answered_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_writing_clarifications_project ON writing_clarifications(project_id);
CREATE INDEX IF NOT EXISTS idx_writing_clarifications_status ON writing_clarifications(status);
`
