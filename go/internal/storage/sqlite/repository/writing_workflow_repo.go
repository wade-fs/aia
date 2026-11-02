package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/storage/sqlite"
)

type WritingWorkflowRepository struct {
	db *sqlite.DB
}

func NewWritingWorkflowRepository(db *sqlite.DB) *WritingWorkflowRepository {
	return &WritingWorkflowRepository{db: db}
}

// CreateBrief inserts a new CreativeBrief record.
func (r *WritingWorkflowRepository) CreateBrief(ctx context.Context, b *writing.CreativeBrief) error {
	query := `
		INSERT INTO writing_briefs (
			id, project_id, title, raw_intent, genre, era,
			theme_and_philosophy_json, scale_json, core_protagonists_json,
			worldview_settings_json, writing_rules_json, status, pending_questions,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = now
	}

	themeJSON, _ := json.Marshal(b.ThemeAndPhilosophy)
	scaleJSON, _ := json.Marshal(b.Scale)
	protagonistsJSON, _ := json.Marshal(b.CoreProtagonists)
	worldviewJSON, _ := json.Marshal(b.WorldviewSettings)
	rulesJSON, _ := json.Marshal(b.WritingRules)

	_, err := r.db.RawDB().ExecContext(ctx, query,
		b.ID, b.ProjectID, b.Title, b.RawIntent, b.Genre, b.Era,
		string(themeJSON), string(scaleJSON), string(protagonistsJSON),
		string(worldviewJSON), string(rulesJSON), string(b.Status), b.PendingQuestions,
		b.CreatedAt.UTC().Format(time.RFC3339), b.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert writing brief: %w", err)
	}
	return nil
}

// UpdateBrief updates an existing CreativeBrief.
func (r *WritingWorkflowRepository) UpdateBrief(ctx context.Context, b *writing.CreativeBrief) error {
	query := `
		UPDATE writing_briefs SET
			title = ?, raw_intent = ?, genre = ?, era = ?,
			theme_and_philosophy_json = ?, scale_json = ?, core_protagonists_json = ?,
			worldview_settings_json = ?, writing_rules_json = ?, status = ?,
			pending_questions = ?, updated_at = ?
		WHERE id = ?
	`
	b.UpdatedAt = time.Now().UTC()

	themeJSON, _ := json.Marshal(b.ThemeAndPhilosophy)
	scaleJSON, _ := json.Marshal(b.Scale)
	protagonistsJSON, _ := json.Marshal(b.CoreProtagonists)
	worldviewJSON, _ := json.Marshal(b.WorldviewSettings)
	rulesJSON, _ := json.Marshal(b.WritingRules)

	res, err := r.db.RawDB().ExecContext(ctx, query,
		b.Title, b.RawIntent, b.Genre, b.Era,
		string(themeJSON), string(scaleJSON), string(protagonistsJSON),
		string(worldviewJSON), string(rulesJSON), string(b.Status),
		b.PendingQuestions, b.UpdatedAt.UTC().Format(time.RFC3339),
		b.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update writing brief: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("writing brief with id %s not found", b.ID)
	}
	return nil
}

// GetBriefByProjectID retrieves the CreativeBrief associated with a project.
func (r *WritingWorkflowRepository) GetBriefByProjectID(ctx context.Context, projectID string) (*writing.CreativeBrief, error) {
	query := `
		SELECT id, project_id, title, raw_intent, genre, era,
			theme_and_philosophy_json, scale_json, core_protagonists_json,
			worldview_settings_json, writing_rules_json, status, pending_questions,
			created_at, updated_at
		FROM writing_briefs
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, projectID)
	return r.scanBrief(row)
}

// GetBriefByID retrieves a CreativeBrief by its ID.
func (r *WritingWorkflowRepository) GetBriefByID(ctx context.Context, id string) (*writing.CreativeBrief, error) {
	query := `
		SELECT id, project_id, title, raw_intent, genre, era,
			theme_and_philosophy_json, scale_json, core_protagonists_json,
			worldview_settings_json, writing_rules_json, status, pending_questions,
			created_at, updated_at
		FROM writing_briefs
		WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)
	return r.scanBrief(row)
}

func (r *WritingWorkflowRepository) scanBrief(row *sql.Row) (*writing.CreativeBrief, error) {
	var b writing.CreativeBrief
	var themeJSON, scaleJSON, protagonistsJSON, worldviewJSON, rulesJSON string
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&b.ID, &b.ProjectID, &b.Title, &b.RawIntent, &b.Genre, &b.Era,
		&themeJSON, &scaleJSON, &protagonistsJSON,
		&worldviewJSON, &rulesJSON, &b.Status, &b.PendingQuestions,
		&createdAtStr, &updatedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan writing brief: %w", err)
	}

	_ = json.Unmarshal([]byte(themeJSON), &b.ThemeAndPhilosophy)
	_ = json.Unmarshal([]byte(scaleJSON), &b.Scale)
	_ = json.Unmarshal([]byte(protagonistsJSON), &b.CoreProtagonists)
	_ = json.Unmarshal([]byte(worldviewJSON), &b.WorldviewSettings)
	_ = json.Unmarshal([]byte(rulesJSON), &b.WritingRules)

	b.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return &b, nil
}

// CreateClarification inserts a new ClarificationRequest.
func (r *WritingWorkflowRepository) CreateClarification(ctx context.Context, c *writing.ClarificationRequest) error {
	query := `
		INSERT INTO writing_clarifications (
			id, project_id, brief_id, category, question, rationale,
			options_json, user_answer, status, created_at, answered_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}

	optionsJSON, _ := json.Marshal(c.Options)
	var answeredAtStr *string
	if c.AnsweredAt != nil {
		s := c.AnsweredAt.UTC().Format(time.RFC3339)
		answeredAtStr = &s
	}

	_, err := r.db.RawDB().ExecContext(ctx, query,
		c.ID, c.ProjectID, c.BriefID, string(c.Category), c.Question, c.Rationale,
		string(optionsJSON), c.UserAnswer, string(c.Status),
		c.CreatedAt.UTC().Format(time.RFC3339), answeredAtStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert clarification request: %w", err)
	}
	return nil
}

// UpdateClarification updates an existing ClarificationRequest (e.g. after human response).
func (r *WritingWorkflowRepository) UpdateClarification(ctx context.Context, c *writing.ClarificationRequest) error {
	query := `
		UPDATE writing_clarifications SET
			user_answer = ?, status = ?, answered_at = ?
		WHERE id = ?
	`
	var answeredAtStr *string
	if c.AnsweredAt != nil {
		s := c.AnsweredAt.UTC().Format(time.RFC3339)
		answeredAtStr = &s
	}

	res, err := r.db.RawDB().ExecContext(ctx, query,
		c.UserAnswer, string(c.Status), answeredAtStr, c.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update clarification request: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("clarification request with id %s not found", c.ID)
	}
	return nil
}

// GetClarificationByID retrieves a single ClarificationRequest.
func (r *WritingWorkflowRepository) GetClarificationByID(ctx context.Context, id string) (*writing.ClarificationRequest, error) {
	query := `
		SELECT id, project_id, brief_id, category, question, rationale,
			options_json, user_answer, status, created_at, answered_at
		FROM writing_clarifications
		WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)
	return r.scanClarification(row)
}

// ListClarificationsByProjectID lists all clarification requests for a given project.
func (r *WritingWorkflowRepository) ListClarificationsByProjectID(ctx context.Context, projectID string) ([]*writing.ClarificationRequest, error) {
	query := `
		SELECT id, project_id, brief_id, category, question, rationale,
			options_json, user_answer, status, created_at, answered_at
		FROM writing_clarifications
		WHERE project_id = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query clarifications: %w", err)
	}
	defer rows.Close()

	var results []*writing.ClarificationRequest
	for rows.Next() {
		c, err := r.scanClarificationRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, c)
	}
	return results, nil
}

// ListPendingClarifications lists all unanswered clarification requests for a project.
func (r *WritingWorkflowRepository) ListPendingClarifications(ctx context.Context, projectID string) ([]*writing.ClarificationRequest, error) {
	query := `
		SELECT id, project_id, brief_id, category, question, rationale,
			options_json, user_answer, status, created_at, answered_at
		FROM writing_clarifications
		WHERE project_id = ? AND status = 'PENDING'
		ORDER BY created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending clarifications: %w", err)
	}
	defer rows.Close()

	var results []*writing.ClarificationRequest
	for rows.Next() {
		c, err := r.scanClarificationRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, c)
	}
	return results, nil
}

func (r *WritingWorkflowRepository) scanClarification(row *sql.Row) (*writing.ClarificationRequest, error) {
	var c writing.ClarificationRequest
	var optionsJSON string
	var createdAtStr string
	var answeredAtStr *string

	err := row.Scan(
		&c.ID, &c.ProjectID, &c.BriefID, &c.Category, &c.Question, &c.Rationale,
		&optionsJSON, &c.UserAnswer, &c.Status, &createdAtStr, &answeredAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan clarification request: %w", err)
	}

	_ = json.Unmarshal([]byte(optionsJSON), &c.Options)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	if answeredAtStr != nil {
		t, _ := time.Parse(time.RFC3339, *answeredAtStr)
		c.AnsweredAt = &t
	}
	return &c, nil
}

func (r *WritingWorkflowRepository) scanClarificationRow(rows *sql.Rows) (*writing.ClarificationRequest, error) {
	var c writing.ClarificationRequest
	var optionsJSON string
	var createdAtStr string
	var answeredAtStr *string

	err := rows.Scan(
		&c.ID, &c.ProjectID, &c.BriefID, &c.Category, &c.Question, &c.Rationale,
		&optionsJSON, &c.UserAnswer, &c.Status, &createdAtStr, &answeredAtStr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan clarification row: %w", err)
	}

	_ = json.Unmarshal([]byte(optionsJSON), &c.Options)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	if answeredAtStr != nil {
		t, _ := time.Parse(time.RFC3339, *answeredAtStr)
		c.AnsweredAt = &t
	}
	return &c, nil
}
