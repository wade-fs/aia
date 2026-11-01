package migrations

// GovernanceAndApprovals defines migration 009 (AIA-ARCH-DATA-01, Phase 2).
//
// It establishes:
//   - approvals: human-in-the-loop operational gates (AIA-17 §7). Every approval
//     must be bound to a concrete Task or Decision (scope binding, AIA-17 §7.4).
//   - audit_logs governance columns (policy_id, decision_id, approval_id,
//     risk_tier, governance_action) — the existing audit trail is expanded
//     instead of introducing a separate policy_audit_logs table.
//   - runtime_controls: the persisted, cross-process Runtime dispatch state used
//     by the Controlled Emergency Stop (AIA-17 §11.5). Persisting the state makes
//     the stop survive process restarts and lets the CLI freeze a running daemon.
const GovernanceAndApprovals = `
CREATE TABLE IF NOT EXISTS approvals (
    id TEXT PRIMARY KEY,
    requester_entity TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    operation_detail_json TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'EXPIRED', 'CANCELLED')),
    risk_tier TEXT NOT NULL CHECK (risk_tier IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    timeout_at TEXT,
    timeout_policy TEXT NOT NULL DEFAULT 'REJECT' CHECK (timeout_policy IN ('REJECT', 'ESCALATE')),
    task_id TEXT REFERENCES tasks(id),
    decision_id TEXT REFERENCES decisions(id),
    reviewer_id TEXT,
    reviewer_note TEXT,
    created_at TEXT NOT NULL,
    reviewed_at TEXT,
    expires_at TEXT,
    CHECK (task_id IS NOT NULL OR decision_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_approvals_status ON approvals(status);
CREATE INDEX IF NOT EXISTS idx_approvals_task ON approvals(task_id);
CREATE INDEX IF NOT EXISTS idx_approvals_decision ON approvals(decision_id);

ALTER TABLE audit_logs ADD COLUMN policy_id TEXT;
ALTER TABLE audit_logs ADD COLUMN decision_id TEXT;
ALTER TABLE audit_logs ADD COLUMN approval_id TEXT;
ALTER TABLE audit_logs ADD COLUMN risk_tier TEXT;
ALTER TABLE audit_logs ADD COLUMN governance_action TEXT;

CREATE INDEX IF NOT EXISTS idx_audit_governance ON audit_logs(governance_action);
CREATE INDEX IF NOT EXISTS idx_audit_decision ON audit_logs(decision_id);
CREATE INDEX IF NOT EXISTS idx_audit_approval ON audit_logs(approval_id);

CREATE TABLE IF NOT EXISTS runtime_controls (
    id TEXT PRIMARY KEY CHECK (id = 'global'),
    dispatch_state TEXT NOT NULL CHECK (dispatch_state IN ('RUNNING', 'EMERGENCY_STOP')),
    reason TEXT NOT NULL DEFAULT '',
    triggered_by TEXT NOT NULL DEFAULT '',
    triggered_at TEXT,
    resumed_by TEXT NOT NULL DEFAULT '',
    resumed_at TEXT,
    updated_at TEXT NOT NULL
);

INSERT OR IGNORE INTO runtime_controls (id, dispatch_state, updated_at)
VALUES ('global', 'RUNNING', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));

CREATE TRIGGER IF NOT EXISTS trg_runtime_controls_no_delete
BEFORE DELETE ON runtime_controls
BEGIN
    SELECT RAISE(ABORT, 'runtime_controls row cannot be deleted');
END;
`
