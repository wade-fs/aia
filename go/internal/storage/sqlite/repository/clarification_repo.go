package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domainClarify "github.com/wade/aia/internal/domain/clarification"
	"github.com/wade/aia/internal/storage/sqlite"
)

// ClarificationRepository manages universal clarification requests and answers in SQLite (AIA-21 §5.2).
type ClarificationRepository struct {
	db *sqlite.DB
}

// NewClarificationRepository creates a new clarification repository.
func NewClarificationRepository(db *sqlite.DB) *ClarificationRepository {
	return &ClarificationRepository{db: db}
}

// Create records a new clarification request.
func (r *ClarificationRepository) Create(ctx context.Context, c *domainClarify.ClarificationRequest) error {
	query := `
		INSERT INTO clarification_requests (
			id, project_id, task_id, field_key, question, options_json,
			reason, impact, priority, blocking, status, answer,
			answered_at, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	optsJSON, _ := json.Marshal(c.Options)
	blockingInt := 0
	if c.Blocking {
		blockingInt = 1
	}

	var answeredAt, expiresAt sql.NullString
	if c.AnsweredAt != nil {
		answeredAt = sql.NullString{String: c.AnsweredAt.Format(time.RFC3339), Valid: true}
	}
	if c.ExpiresAt != nil {
		expiresAt = sql.NullString{String: c.ExpiresAt.Format(time.RFC3339), Valid: true}
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		c.ID, c.ProjectID, c.TaskID, c.FieldKey, c.Question, string(optsJSON),
		c.Reason, c.Impact, int(c.Priority), blockingInt, string(c.Status), c.Answer,
		answeredAt, expiresAt, c.CreatedAt.Format(time.RFC3339), c.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert clarification request: %w", err)
	}
	return nil
}

// Get retrieves a clarification request by ID.
func (r *ClarificationRepository) Get(ctx context.Context, id string) (*domainClarify.ClarificationRequest, error) {
	query := `
		SELECT id, project_id, task_id, field_key, question, options_json,
		       reason, impact, priority, blocking, status, answer,
		       answered_at, expires_at, created_at, updated_at
		FROM clarification_requests
		WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)
	return r.scanRow(row)
}

// Update saves changes to a clarification request.
func (r *ClarificationRepository) Update(ctx context.Context, c *domainClarify.ClarificationRequest) error {
	query := `
		UPDATE clarification_requests SET
			status = ?,
			answer = ?,
			answered_at = ?,
			updated_at = ?
		WHERE id = ?
	`
	now := time.Now().UTC()
	c.UpdatedAt = now

	var answeredAt sql.NullString
	if c.AnsweredAt != nil {
		answeredAt = sql.NullString{String: c.AnsweredAt.Format(time.RFC3339), Valid: true}
	}

	res, err := r.db.RawDB().ExecContext(
		ctx, query,
		string(c.Status), c.Answer, answeredAt, now.Format(time.RFC3339), c.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update clarification request: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("clarification request not found: %s", c.ID)
	}
	return nil
}

// ListByProject returns all clarification requests for a project, sorted by priority ASC, created_at ASC.
// If projectID is empty, it returns requests across all projects.
func (r *ClarificationRepository) ListByProject(ctx context.Context, projectID string) ([]*domainClarify.ClarificationRequest, error) {
	query := `
		SELECT id, project_id, task_id, field_key, question, options_json,
		       reason, impact, priority, blocking, status, answer,
		       answered_at, expires_at, created_at, updated_at
		FROM clarification_requests
		WHERE (? = '' OR project_id = ?)
		ORDER BY priority ASC, created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clarifications: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// ListOpenByProject returns open clarification requests for a project.
// If projectID is empty, it returns open requests across all projects.
func (r *ClarificationRepository) ListOpenByProject(ctx context.Context, projectID string) ([]*domainClarify.ClarificationRequest, error) {
	query := `
		SELECT id, project_id, task_id, field_key, question, options_json,
		       reason, impact, priority, blocking, status, answer,
		       answered_at, expires_at, created_at, updated_at
		FROM clarification_requests
		WHERE (? = '' OR project_id = ?) AND status = 'OPEN'
		ORDER BY priority ASC, created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list open clarifications: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// ListBlockingOpenByProject returns open requests that are marked blocking.
func (r *ClarificationRepository) ListBlockingOpenByProject(ctx context.Context, projectID string) ([]*domainClarify.ClarificationRequest, error) {
	query := `
		SELECT id, project_id, task_id, field_key, question, options_json,
		       reason, impact, priority, blocking, status, answer,
		       answered_at, expires_at, created_at, updated_at
		FROM clarification_requests
		WHERE project_id = ? AND status = 'OPEN' AND blocking = 1
		ORDER BY priority ASC, created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list blocking open clarifications: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// ListExpiredOpen returns open requests whose expires_at is non-null and <= now.
func (r *ClarificationRepository) ListExpiredOpen(ctx context.Context, now time.Time) ([]*domainClarify.ClarificationRequest, error) {
	query := `
		SELECT id, project_id, task_id, field_key, question, options_json,
		       reason, impact, priority, blocking, status, answer,
		       answered_at, expires_at, created_at, updated_at
		FROM clarification_requests
		WHERE status = 'OPEN' AND expires_at IS NOT NULL AND expires_at <= ?
		ORDER BY expires_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to list expired clarifications: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

func (r *ClarificationRepository) scanRow(row *sql.Row) (*domainClarify.ClarificationRequest, error) {
	var c domainClarify.ClarificationRequest
	var optsJSON, statusStr, createdAt, updatedAt string
	var priorityInt, blockingInt int
	var taskID, answer, answeredAt, expiresAt sql.NullString

	if err := row.Scan(
		&c.ID, &c.ProjectID, &taskID, &c.FieldKey, &c.Question, &optsJSON,
		&c.Reason, &c.Impact, &priorityInt, &blockingInt, &statusStr, &answer,
		&answeredAt, &expiresAt, &createdAt, &updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	_ = json.Unmarshal([]byte(optsJSON), &c.Options)
	c.Priority = domainClarify.ClarificationPriority(priorityInt)
	c.Blocking = blockingInt == 1
	c.Status = domainClarify.ClarificationStatus(statusStr)
	if taskID.Valid {
		c.TaskID = taskID.String
	}
	if answer.Valid {
		c.Answer = answer.String
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if answeredAt.Valid {
		t, _ := time.Parse(time.RFC3339, answeredAt.String)
		c.AnsweredAt = &t
	}
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		c.ExpiresAt = &t
	}
	return &c, nil
}

func (r *ClarificationRepository) scanRows(rows *sql.Rows) ([]*domainClarify.ClarificationRequest, error) {
	var list []*domainClarify.ClarificationRequest
	for rows.Next() {
		var c domainClarify.ClarificationRequest
		var optsJSON, statusStr, createdAt, updatedAt string
		var priorityInt, blockingInt int
		var taskID, answer, answeredAt, expiresAt sql.NullString

		if err := rows.Scan(
			&c.ID, &c.ProjectID, &taskID, &c.FieldKey, &c.Question, &optsJSON,
			&c.Reason, &c.Impact, &priorityInt, &blockingInt, &statusStr, &answer,
			&answeredAt, &expiresAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal([]byte(optsJSON), &c.Options)
		c.Priority = domainClarify.ClarificationPriority(priorityInt)
		c.Blocking = blockingInt == 1
		c.Status = domainClarify.ClarificationStatus(statusStr)
		if taskID.Valid {
			c.TaskID = taskID.String
		}
		if answer.Valid {
			c.Answer = answer.String
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if answeredAt.Valid {
			t, _ := time.Parse(time.RFC3339, answeredAt.String)
			c.AnsweredAt = &t
		}
		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			c.ExpiresAt = &t
		}
		list = append(list, &c)
	}
	return list, nil
}
