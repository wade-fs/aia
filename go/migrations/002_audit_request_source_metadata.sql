-- Migration 002: Add request_id, source and metadata_json to audit_logs
ALTER TABLE audit_logs ADD COLUMN request_id TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN source TEXT NOT NULL DEFAULT 'RUNTIME';
ALTER TABLE audit_logs ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_audit_request ON audit_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_source ON audit_logs(source);
