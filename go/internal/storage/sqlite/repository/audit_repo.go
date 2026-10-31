package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/storage/sqlite"
)

type AuditLogEntry struct {
	ID           domain.ID       `json:"id"`
	Sequence     int64           `json:"sequence"`
	ActorType    string          `json:"actor_type"`
	ActorID      string          `json:"actor_id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Outcome      string          `json:"outcome"`
	Details      json.RawMessage `json:"details"`
	CreatedAt    time.Time       `json:"created_at"`
}

type AuditRepository struct {
	db *sqlite.DB
}

func NewAuditRepository(db *sqlite.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Append(ctx context.Context, entry *AuditLogEntry) error {
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
				id, sequence, actor_type, actor_id, action, resource_type, resource_id, outcome, details_json, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		details := string(entry.Details)
		if details == "" {
			details = "{}"
		}

		_, err := tx.ExecContext(
			ctx, query,
			entry.ID, entry.Sequence, entry.ActorType, entry.ActorID, entry.Action,
			entry.ResourceType, entry.ResourceID, entry.Outcome, details, entry.CreatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to append audit log: %w", err)
		}
		return nil
	})
}

func (r *AuditRepository) List(ctx context.Context, limit int) ([]AuditLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	query := fmt.Sprintf(`
		SELECT id, sequence, actor_type, actor_id, action, resource_type, resource_id, outcome, details_json, created_at
		FROM audit_logs ORDER BY sequence DESC LIMIT %d
	`, limit)

	rows, err := r.db.RawDB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var list []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var details, createdAt string
		if err := rows.Scan(
			&e.ID, &e.Sequence, &e.ActorType, &e.ActorID, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.Outcome, &details, &createdAt,
		); err != nil {
			return nil, err
		}
		e.Details = json.RawMessage(details)
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		list = append(list, e)
	}
	return list, nil
}
