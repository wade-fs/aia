package migrations

// SelfModificationAndEvolution creates the schema for Phase 6 Safe Self-Modification Proposals (Plan.md §15, roadmap.md §Phase 6).
const SelfModificationAndEvolution = `
CREATE TABLE IF NOT EXISTS self_modification_proposals (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    rationale TEXT NOT NULL,
    target_files_json TEXT NOT NULL DEFAULT '[]',
    patches_json TEXT NOT NULL DEFAULT '{}',
    scope_level TEXT NOT NULL,
    requires_approval BOOLEAN NOT NULL DEFAULT 0,
    approval_id TEXT,
    status TEXT NOT NULL,
    branch_name TEXT NOT NULL DEFAULT '',
    pre_commit_sha TEXT NOT NULL DEFAULT '',
    post_commit_sha TEXT NOT NULL DEFAULT '',
    verification_output TEXT NOT NULL DEFAULT '',
    rca_log TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    completed_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_selfmod_status ON self_modification_proposals(status);
CREATE INDEX IF NOT EXISTS idx_selfmod_scope ON self_modification_proposals(scope_level);
`
