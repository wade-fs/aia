package migrations

// EventHistoryAndProvenance defines migration 007 (AIA-ARCH-DATA-01, Phase 1).
// It establishes the immutable event store (hash chain, triggers), multi-dimensional
// sources, evidence provenance, claims with revision history, and memories provenance fields.
const EventHistoryAndProvenance = `
CREATE TABLE IF NOT EXISTS event_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    event_version INTEGER NOT NULL DEFAULT 1,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    aggregate_version INTEGER NOT NULL,
    correlation_id TEXT NOT NULL,
    causation_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    previous_hash TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    recorded_at TEXT NOT NULL,
    UNIQUE(aggregate_type, aggregate_id, aggregate_version)
);

CREATE INDEX IF NOT EXISTS idx_event_history_aggregate ON event_history(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_event_history_correlation ON event_history(correlation_id);
CREATE INDEX IF NOT EXISTS idx_event_history_recorded ON event_history(recorded_at);

CREATE TRIGGER IF NOT EXISTS trg_event_history_no_update
BEFORE UPDATE ON event_history
BEGIN
    SELECT RAISE(ABORT, 'SECURITY ERROR: event_history is immutable and cannot be updated.');
END;

CREATE TRIGGER IF NOT EXISTS trg_event_history_no_delete
BEFORE DELETE ON event_history
BEGIN
    SELECT RAISE(ABORT, 'SECURITY ERROR: event_history is immutable and cannot be deleted.');
END;

CREATE TABLE IF NOT EXISTS sources (
    id TEXT PRIMARY KEY,
    source_type TEXT NOT NULL,
    uri TEXT NOT NULL,
    publisher TEXT NOT NULL DEFAULT '',
    authority_score REAL NOT NULL DEFAULT 0.5,
    directness_score REAL NOT NULL DEFAULT 0.5,
    timeliness_score REAL NOT NULL DEFAULT 0.5,
    independence_score REAL NOT NULL DEFAULT 0.5,
    reproducibility_score REAL NOT NULL DEFAULT 0.5,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS evidence (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES sources(id),
    content_snippet TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    capture_method TEXT NOT NULL,
    data_version TEXT NOT NULL DEFAULT '1.0',
    published_at TEXT,
    captured_at TEXT NOT NULL,
    valid_until TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_evidence_source ON evidence(source_id);
CREATE INDEX IF NOT EXISTS idx_evidence_hash ON evidence(content_hash);

CREATE TABLE IF NOT EXISTS claims (
    id TEXT PRIMARY KEY,
    statement TEXT NOT NULL,
    assertion_status TEXT NOT NULL,
    is_knowledge INTEGER NOT NULL DEFAULT 0,
    current_version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS claim_revisions (
    id TEXT PRIMARY KEY,
    claim_id TEXT NOT NULL REFERENCES claims(id),
    revision_number INTEGER NOT NULL,
    previous_statement TEXT NOT NULL,
    new_statement TEXT NOT NULL,
    reason TEXT NOT NULL,
    revised_decision_id TEXT,
    created_at TEXT NOT NULL,
    UNIQUE(claim_id, revision_number)
);

CREATE TABLE IF NOT EXISTS claim_evidence_links (
    claim_id TEXT NOT NULL REFERENCES claims(id),
    evidence_id TEXT NOT NULL REFERENCES evidence(id),
    support_type TEXT NOT NULL,
    confidence_weight REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (claim_id, evidence_id)
);

ALTER TABLE memories ADD COLUMN provenance_type TEXT NOT NULL DEFAULT 'INFERENCE';
ALTER TABLE memories ADD COLUMN provenance_ref_type TEXT NOT NULL DEFAULT 'NONE';
ALTER TABLE memories ADD COLUMN provenance_ref_id TEXT;
ALTER TABLE memories ADD COLUMN provenance_status TEXT NOT NULL DEFAULT 'UNVERIFIED';
`
