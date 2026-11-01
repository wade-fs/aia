package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/storage/sqlite"
)

type GovernanceRepository struct {
	db *sqlite.DB
}

func NewGovernanceRepository(db *sqlite.DB) *GovernanceRepository {
	return &GovernanceRepository{db: db}
}

// GetControl returns the single global runtime control record.
func (r *GovernanceRepository) GetControl(ctx context.Context) (*governance.RuntimeControl, error) {
	query := `
		SELECT id, dispatch_state, reason, triggered_by, triggered_at, resumed_by, resumed_at, updated_at
		FROM runtime_controls
		WHERE id = 'global'
	`
	row := r.db.RawDB().QueryRowContext(ctx, query)

	var ctrl governance.RuntimeControl
	var stateStr, updatedAtStr string
	var reason, triggeredBy, triggeredAtStr, resumedBy, resumedAtStr sql.NullString

	if err := row.Scan(
		&ctrl.ID, &stateStr, &reason, &triggeredBy, &triggeredAtStr, &resumedBy, &resumedAtStr, &updatedAtStr,
	); err != nil {
		if err == sql.ErrNoRows {
			// fallback default
			return &governance.RuntimeControl{
				ID:            "global",
				DispatchState: governance.DispatchRunning,
				UpdatedAt:     time.Now().UTC(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get runtime control: %w", err)
	}

	ctrl.DispatchState = governance.DispatchState(stateStr)
	if reason.Valid {
		ctrl.Reason = reason.String
	}
	if triggeredBy.Valid {
		ctrl.TriggeredBy = triggeredBy.String
	}
	if resumedBy.Valid {
		ctrl.ResumedBy = resumedBy.String
	}
	if triggeredAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, triggeredAtStr.String)
		ctrl.TriggeredAt = &t
	}
	if resumedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, resumedAtStr.String)
		ctrl.ResumedAt = &t
	}
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		ctrl.UpdatedAt = t
	}

	return &ctrl, nil
}

// SetEmergencyStop freezes task dispatching in runtime_controls.
func (r *GovernanceRepository) SetEmergencyStop(ctx context.Context, actor, reason string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := `
		INSERT INTO runtime_controls (id, dispatch_state, reason, triggered_by, triggered_at, updated_at)
		VALUES ('global', 'EMERGENCY_STOP', ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			dispatch_state = 'EMERGENCY_STOP',
			reason = excluded.reason,
			triggered_by = excluded.triggered_by,
			triggered_at = excluded.triggered_at,
			updated_at = excluded.updated_at
	`
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, reason, actor, now, now)
		return err
	})
}

// ResumeDispatch restores dispatching in runtime_controls.
func (r *GovernanceRepository) ResumeDispatch(ctx context.Context, actor string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := `
		INSERT INTO runtime_controls (id, dispatch_state, reason, resumed_by, resumed_at, updated_at)
		VALUES ('global', 'RUNNING', '', ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			dispatch_state = 'RUNNING',
			reason = '',
			resumed_by = excluded.resumed_by,
			resumed_at = excluded.resumed_at,
			updated_at = excluded.updated_at
	`
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, actor, now, now)
		return err
	})
}
