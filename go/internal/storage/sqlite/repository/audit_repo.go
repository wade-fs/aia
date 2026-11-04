package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wade/aia/internal/audit"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/storage/sqlite"
)

type AuditLogEntry struct {
	ID               domain.ID       `json:"id"`
	Sequence         int64           `json:"sequence"`
	ActorType        string          `json:"actor_type"`
	ActorID          string          `json:"actor_id"`
	Action           string          `json:"action"`
	ResourceType     string          `json:"resource_type"`
	ResourceID       string          `json:"resource_id"`
	Outcome          string          `json:"outcome"`
	Details          json.RawMessage `json:"details"`
	RequestID        string          `json:"request_id"`
	Source           string          `json:"source"`
	Metadata         json.RawMessage `json:"metadata"`
	PolicyID         *string         `json:"policy_id,omitempty"`
	DecisionID       *string         `json:"decision_id,omitempty"`
	ApprovalID       *string         `json:"approval_id,omitempty"`
	RiskTier         *string         `json:"risk_tier,omitempty"`
	GovernanceAction *string         `json:"governance_action,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// AuditFilter narrows audit queries.
type AuditFilter struct {
	Limit            int
	Source           string
	ResourceType     string
	ResourceID       string
	RequestID        string
	PolicyID         string
	DecisionID       string
	ApprovalID       string
	RiskTier         string
	GovernanceAction string
}

type AuditRepository struct {
	db *sqlite.DB
}

func NewAuditRepository(db *sqlite.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Append writes an immutable audit record. Request ID, source and actor are
// filled from the context origin when not set explicitly. Sensitive metadata
// keys are redacted before persisting.
func (r *AuditRepository) Append(ctx context.Context, entry *AuditLogEntry) error {
	if o, ok := audit.FromContext(ctx); ok {
		if entry.RequestID == "" {
			entry.RequestID = o.RequestID
		}
		if entry.Source == "" {
			entry.Source = string(o.Source)
		}
		// Human actors (web/cli users) take precedence over the generic
		// SYSTEM actor used by services, so operations remain attributable.
		if o.ActorID != "" && (entry.ActorID == "" || entry.ActorType == "SYSTEM") {
			entry.Metadata = mergeActor(entry.Metadata, entry.ActorType, entry.ActorID)
			entry.ActorType = o.ActorType
			entry.ActorID = o.ActorID
		}
	}
	if entry.Source == "" {
		entry.Source = string(audit.SourceRuntime)
	}
	entry.Metadata = audit.SanitizeRaw(entry.Metadata)
	entry.Details = audit.SanitizeRaw(entry.Details)

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		var maxSeq sql.NullInt64
		_ = tx.QueryRowContext(ctx, "SELECT MAX(sequence) FROM audit_logs").Scan(&maxSeq)

		nextSeq := int64(1)
		if maxSeq.Valid {
			nextSeq = maxSeq.Int64 + 1
		}
		entry.Sequence = nextSeq

		now := time.Now().UTC()
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = now
		}

		query := `
			INSERT INTO audit_logs (
				id, sequence, actor_type, actor_id, action, resource_type, resource_id, outcome, details_json,
				request_id, source, metadata_json, policy_id, decision_id, approval_id, risk_tier, governance_action, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := tx.ExecContext(
			ctx, query,
			entry.ID, entry.Sequence, entry.ActorType, entry.ActorID, entry.Action,
			entry.ResourceType, entry.ResourceID, entry.Outcome, string(entry.Details),
			entry.RequestID, entry.Source, string(entry.Metadata),
			entry.PolicyID, entry.DecisionID, entry.ApprovalID, entry.RiskTier, entry.GovernanceAction,
			entry.CreatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to append audit log: %w", err)
		}
		return nil
	})
}

func mergeActor(meta json.RawMessage, actorType, actorID string) json.RawMessage {
	m := map[string]any{}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &m)
	}
	if actorID != "" {
		m["service_actor"] = actorType + ":" + actorID
	}
	b, _ := json.Marshal(m)
	return b
}

func (r *AuditRepository) List(ctx context.Context, limit int) ([]AuditLogEntry, error) {
	return r.Query(ctx, AuditFilter{Limit: limit})
}

// Query lists audit records newest first using the given filter.
func (r *AuditRepository) Query(ctx context.Context, f AuditFilter) ([]AuditLogEntry, error) {
	if f.Limit <= 0 || f.Limit > 1000 {
		f.Limit = 100
	}
	var where []string
	var args []any
	if f.Source != "" {
		where = append(where, "source = ?")
		args = append(args, f.Source)
	}
	if f.ResourceType != "" {
		where = append(where, "resource_type = ?")
		args = append(args, f.ResourceType)
	}
	if f.ResourceID != "" {
		where = append(where, "resource_id = ?")
		args = append(args, f.ResourceID)
	}
	if f.RequestID != "" {
		where = append(where, "request_id = ?")
		args = append(args, f.RequestID)
	}
	if f.PolicyID != "" {
		where = append(where, "policy_id = ?")
		args = append(args, f.PolicyID)
	}
	if f.DecisionID != "" {
		where = append(where, "decision_id = ?")
		args = append(args, f.DecisionID)
	}
	if f.ApprovalID != "" {
		where = append(where, "approval_id = ?")
		args = append(args, f.ApprovalID)
	}
	if f.RiskTier != "" {
		where = append(where, "risk_tier = ?")
		args = append(args, f.RiskTier)
	}
	if f.GovernanceAction != "" {
		where = append(where, "governance_action = ?")
		args = append(args, f.GovernanceAction)
	}

	query := `SELECT id, sequence, actor_type, actor_id, action, resource_type, resource_id, outcome, details_json,
		request_id, source, metadata_json, policy_id, decision_id, approval_id, risk_tier, governance_action, created_at FROM audit_logs`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY sequence DESC LIMIT ?"
	args = append(args, f.Limit)

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var list []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var details, metadata, createdAt string
		var policyID, decisionID, approvalID, riskTier, govAction sql.NullString
		if err := rows.Scan(
			&e.ID, &e.Sequence, &e.ActorType, &e.ActorID, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.Outcome, &details,
			&e.RequestID, &e.Source, &metadata,
			&policyID, &decisionID, &approvalID, &riskTier, &govAction,
			&createdAt,
		); err != nil {
			return nil, err
		}
		e.Details = json.RawMessage(details)
		e.Metadata = json.RawMessage(metadata)
		if policyID.Valid {
			e.PolicyID = &policyID.String
		}
		if decisionID.Valid {
			e.DecisionID = &decisionID.String
		}
		if approvalID.Valid {
			e.ApprovalID = &approvalID.String
		}
		if riskTier.Valid {
			e.RiskTier = &riskTier.String
		}
		if govAction.Valid {
			e.GovernanceAction = &govAction.String
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		list = append(list, e)
	}
	return list, rows.Err()
}

// PurgeOlderThan removes audit logs created before cutoff (implements AIA-19 logging.retain_days).
func (r *AuditRepository) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	cutoffStr := cutoff.UTC().Format(time.RFC3339)
	res, err := r.db.RawDB().ExecContext(ctx, "DELETE FROM audit_logs WHERE created_at < ?", cutoffStr)
	if err != nil {
		return 0, fmt.Errorf("failed to purge old audit logs: %w", err)
	}
	return res.RowsAffected()
}
