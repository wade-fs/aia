package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/storage/sqlite"
)

type ApprovalFilter struct {
	Status        governance.ApprovalStatus
	RiskTier      governance.RiskTier
	TaskID        string
	DecisionID    string
	OperationType string
	Limit         int
}

type ApprovalRepository struct {
	db *sqlite.DB
}

func NewApprovalRepository(db *sqlite.DB) *ApprovalRepository {
	return &ApprovalRepository{db: db}
}

func (r *ApprovalRepository) Create(ctx context.Context, a *governance.Approval) error {
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.TimeoutPolicy == "" {
		a.TimeoutPolicy = governance.TimeoutPolicyReject
	}

	var timeoutAtStr, reviewedAtStr, expiresAtStr *string
	if a.TimeoutAt != nil {
		s := a.TimeoutAt.Format(time.RFC3339)
		timeoutAtStr = &s
	}
	if a.ReviewedAt != nil {
		s := a.ReviewedAt.Format(time.RFC3339)
		reviewedAtStr = &s
	}
	if a.ExpiresAt != nil {
		s := a.ExpiresAt.Format(time.RFC3339)
		expiresAtStr = &s
	}

	query := `
		INSERT INTO approvals (
			id, requester_entity, operation_type, operation_detail_json, status, risk_tier,
			timeout_at, timeout_policy, task_id, decision_id, reviewer_id, reviewer_note,
			created_at, reviewed_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(
			ctx, query,
			a.ID, a.RequesterEntity, a.OperationType, a.OperationDetailJSON,
			string(a.Status), string(a.RiskTier),
			timeoutAtStr, string(a.TimeoutPolicy), a.TaskID, a.DecisionID,
			a.ReviewerID, a.ReviewerNote,
			a.CreatedAt.Format(time.RFC3339), reviewedAtStr, expiresAtStr,
		)
		if err != nil {
			return fmt.Errorf("failed to create approval: %w", err)
		}
		return nil
	})
}

func (r *ApprovalRepository) Get(ctx context.Context, id string) (*governance.Approval, error) {
	query := `
		SELECT id, requester_entity, operation_type, operation_detail_json, status, risk_tier,
			timeout_at, timeout_policy, task_id, decision_id, reviewer_id, reviewer_note,
			created_at, reviewed_at, expires_at
		FROM approvals WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)

	var a governance.Approval
	var statusStr, riskStr, timeoutPolStr, createdAtStr string
	var timeoutAtStr, reviewedAtStr, expiresAtStr, taskID, decisionID, reviewerID, reviewerNote sql.NullString

	if err := row.Scan(
		&a.ID, &a.RequesterEntity, &a.OperationType, &a.OperationDetailJSON,
		&statusStr, &riskStr, &timeoutAtStr, &timeoutPolStr,
		&taskID, &decisionID, &reviewerID, &reviewerNote,
		&createdAtStr, &reviewedAtStr, &expiresAtStr,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("approval %s not found: %w", id, err)
		}
		return nil, fmt.Errorf("failed to get approval %s: %w", id, err)
	}

	a.Status = governance.ApprovalStatus(statusStr)
	a.RiskTier = governance.RiskTier(riskStr)
	a.TimeoutPolicy = governance.TimeoutPolicy(timeoutPolStr)
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	if timeoutAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, timeoutAtStr.String)
		a.TimeoutAt = &t
	}
	if reviewedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, reviewedAtStr.String)
		a.ReviewedAt = &t
	}
	if expiresAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAtStr.String)
		a.ExpiresAt = &t
	}
	if taskID.Valid {
		a.TaskID = &taskID.String
	}
	if decisionID.Valid {
		a.DecisionID = &decisionID.String
	}
	if reviewerID.Valid {
		a.ReviewerID = &reviewerID.String
	}
	if reviewerNote.Valid {
		a.ReviewerNote = &reviewerNote.String
	}

	return &a, nil
}

func (r *ApprovalRepository) UpdateStatus(ctx context.Context, id string, status governance.ApprovalStatus, reviewerID, reviewerNote *string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := `
		UPDATE approvals
		SET status = ?, reviewer_id = COALESCE(?, reviewer_id), reviewer_note = COALESCE(?, reviewer_note), reviewed_at = ?
		WHERE id = ?
	`
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, string(status), reviewerID, reviewerNote, now, id)
		if err != nil {
			return fmt.Errorf("failed to update approval status: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("approval %s not found", id)
		}
		return nil
	})
}

func (r *ApprovalRepository) List(ctx context.Context, filter ApprovalFilter) ([]*governance.Approval, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 100
	}

	var where []string
	var args []any

	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, string(filter.Status))
	}
	if filter.RiskTier != "" {
		where = append(where, "risk_tier = ?")
		args = append(args, string(filter.RiskTier))
	}
	if filter.TaskID != "" {
		where = append(where, "task_id = ?")
		args = append(args, filter.TaskID)
	}
	if filter.DecisionID != "" {
		where = append(where, "decision_id = ?")
		args = append(args, filter.DecisionID)
	}
	if filter.OperationType != "" {
		where = append(where, "operation_type = ?")
		args = append(args, filter.OperationType)
	}

	query := `
		SELECT id, requester_entity, operation_type, operation_detail_json, status, risk_tier,
			timeout_at, timeout_policy, task_id, decision_id, reviewer_id, reviewer_note,
			created_at, reviewed_at, expires_at
		FROM approvals
	`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, filter.Limit)

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list approvals: %w", err)
	}
	defer rows.Close()

	var list []*governance.Approval
	for rows.Next() {
		var a governance.Approval
		var statusStr, riskStr, timeoutPolStr, createdAtStr string
		var timeoutAtStr, reviewedAtStr, expiresAtStr, taskID, decisionID, reviewerID, reviewerNote sql.NullString

		if err := rows.Scan(
			&a.ID, &a.RequesterEntity, &a.OperationType, &a.OperationDetailJSON,
			&statusStr, &riskStr, &timeoutAtStr, &timeoutPolStr,
			&taskID, &decisionID, &reviewerID, &reviewerNote,
			&createdAtStr, &reviewedAtStr, &expiresAtStr,
		); err != nil {
			return nil, err
		}

		a.Status = governance.ApprovalStatus(statusStr)
		a.RiskTier = governance.RiskTier(riskStr)
		a.TimeoutPolicy = governance.TimeoutPolicy(timeoutPolStr)
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

		if timeoutAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, timeoutAtStr.String)
			a.TimeoutAt = &t
		}
		if reviewedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, reviewedAtStr.String)
			a.ReviewedAt = &t
		}
		if expiresAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAtStr.String)
			a.ExpiresAt = &t
		}
		if taskID.Valid {
			a.TaskID = &taskID.String
		}
		if decisionID.Valid {
			a.DecisionID = &decisionID.String
		}
		if reviewerID.Valid {
			a.ReviewerID = &reviewerID.String
		}
		if reviewerNote.Valid {
			a.ReviewerNote = &reviewerNote.String
		}

		list = append(list, &a)
	}

	return list, rows.Err()
}

func (r *ApprovalRepository) ListPending(ctx context.Context) ([]*governance.Approval, error) {
	return r.List(ctx, ApprovalFilter{Status: governance.ApprovalPending, Limit: 100})
}

func (r *ApprovalRepository) CheckExpired(ctx context.Context, now time.Time) (int, error) {
	nowStr := now.UTC().Format(time.RFC3339)
	query := `
		UPDATE approvals
		SET status = 'EXPIRED', reviewed_at = ?
		WHERE status = 'PENDING'
		  AND ((timeout_at IS NOT NULL AND timeout_at <= ?)
		    OR (expires_at IS NOT NULL AND expires_at <= ?))
	`
	var affected int64
	err := r.db.WithTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, nowStr, nowStr, nowStr)
		if err != nil {
			return err
		}
		affected, err = res.RowsAffected()
		return err
	})
	return int(affected), err
}
