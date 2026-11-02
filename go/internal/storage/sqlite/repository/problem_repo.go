package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domainProblem "github.com/wade/aia/internal/domain/problem"
	"github.com/wade/aia/internal/storage/sqlite"
)

// ProblemRepository manages persistent Problem, ProblemModel, and AnalysisRecord entities in SQLite (AIA-22 §6, §12).
type ProblemRepository struct {
	db *sqlite.DB
}

// NewProblemRepository creates a new SQLite problem repository.
func NewProblemRepository(db *sqlite.DB) *ProblemRepository {
	return &ProblemRepository{db: db}
}

var _ domainProblem.Repository = (*ProblemRepository)(nil)

// Create records a new Problem entity.
func (r *ProblemRepository) Create(ctx context.Context, p *domainProblem.Problem) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("%w: %v", domainProblem.ErrInvalidProblem, err)
	}

	query := `
		INSERT INTO problems (
			id, project_id, parent_id, goal_id, title, description,
			objective, scope, status, priority, current_model_version,
			current_plan_version, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		p.ID, p.ProjectID, p.ParentID, p.GoalID, p.Title, p.Description,
		p.Objective, p.Scope, string(p.Status), string(p.Priority),
		p.CurrentModelVersion, p.CurrentPlanVersion,
		p.CreatedAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert problem: %w", err)
	}
	return nil
}

// Get retrieves a Problem by ID.
func (r *ProblemRepository) Get(ctx context.Context, id string) (*domainProblem.Problem, error) {
	query := `
		SELECT id, project_id, parent_id, goal_id, title, description,
		       objective, scope, status, priority, current_model_version,
		       current_plan_version, created_at, updated_at
		FROM problems
		WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)

	var p domainProblem.Problem
	var statusStr, priorityStr, createdAtStr, updatedAtStr string
	err := row.Scan(
		&p.ID, &p.ProjectID, &p.ParentID, &p.GoalID, &p.Title, &p.Description,
		&p.Objective, &p.Scope, &statusStr, &priorityStr, &p.CurrentModelVersion,
		&p.CurrentPlanVersion, &createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domainProblem.ErrProblemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan problem: %w", err)
	}

	p.Status = domainProblem.ProblemStatus(statusStr)
	p.Priority = domainProblem.ProblemPriority(priorityStr)
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &p, nil
}

// Update updates mutable fields of a Problem.
func (r *ProblemRepository) Update(ctx context.Context, p *domainProblem.Problem) error {
	query := `
		UPDATE problems
		SET parent_id = ?, goal_id = ?, title = ?, description = ?,
		    objective = ?, scope = ?, status = ?, priority = ?,
		    current_model_version = ?, current_plan_version = ?, updated_at = ?
		WHERE id = ?
	`
	p.UpdatedAt = time.Now().UTC()
	res, err := r.db.RawDB().ExecContext(
		ctx, query,
		p.ParentID, p.GoalID, p.Title, p.Description,
		p.Objective, p.Scope, string(p.Status), string(p.Priority),
		p.CurrentModelVersion, p.CurrentPlanVersion,
		p.UpdatedAt.Format(time.RFC3339), p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update problem: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domainProblem.ErrProblemNotFound
	}
	return nil
}

// ListByProject returns all problems associated with a project.
func (r *ProblemRepository) ListByProject(ctx context.Context, projectID string) ([]*domainProblem.Problem, error) {
	query := `
		SELECT id, project_id, parent_id, goal_id, title, description,
		       objective, scope, status, priority, current_model_version,
		       current_plan_version, created_at, updated_at
		FROM problems
		WHERE project_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query problems by project: %w", err)
	}
	defer rows.Close()

	var list []*domainProblem.Problem
	for rows.Next() {
		var p domainProblem.Problem
		var statusStr, priorityStr, createdAtStr, updatedAtStr string
		if err := rows.Scan(
			&p.ID, &p.ProjectID, &p.ParentID, &p.GoalID, &p.Title, &p.Description,
			&p.Objective, &p.Scope, &statusStr, &priorityStr, &p.CurrentModelVersion,
			&p.CurrentPlanVersion, &createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan problem row: %w", err)
		}
		p.Status = domainProblem.ProblemStatus(statusStr)
		p.Priority = domainProblem.ProblemPriority(priorityStr)
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		list = append(list, &p)
	}
	return list, nil
}

// ListChildren returns all direct child sub-problems of a parent problem.
func (r *ProblemRepository) ListChildren(ctx context.Context, parentID string) ([]*domainProblem.Problem, error) {
	query := `
		SELECT id, project_id, parent_id, goal_id, title, description,
		       objective, scope, status, priority, current_model_version,
		       current_plan_version, created_at, updated_at
		FROM problems
		WHERE parent_id = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query child problems: %w", err)
	}
	defer rows.Close()

	var list []*domainProblem.Problem
	for rows.Next() {
		var p domainProblem.Problem
		var statusStr, priorityStr, createdAtStr, updatedAtStr string
		if err := rows.Scan(
			&p.ID, &p.ProjectID, &p.ParentID, &p.GoalID, &p.Title, &p.Description,
			&p.Objective, &p.Scope, &statusStr, &priorityStr, &p.CurrentModelVersion,
			&p.CurrentPlanVersion, &createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan child problem row: %w", err)
		}
		p.Status = domainProblem.ProblemStatus(statusStr)
		p.Priority = domainProblem.ProblemPriority(priorityStr)
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		list = append(list, &p)
	}
	return list, nil
}

// SaveModel saves a new version of ProblemModel and bumps current_model_version on the Problem.
func (r *ProblemRepository) SaveModel(ctx context.Context, model *domainProblem.ProblemModel) error {
	tx, err := r.db.RawDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}

	objJSON, _ := json.Marshal(model.Objectives)
	conJSON, _ := json.Marshal(model.Constraints)
	kfJSON, _ := json.Marshal(model.KnownFacts)
	assJSON, _ := json.Marshal(model.Assumptions)
	unkJSON, _ := json.Marshal(model.Unknowns)
	depJSON, _ := json.Marshal(model.Dependencies)
	rskJSON, _ := json.Marshal(model.Risks)
	scJSON, _ := json.Marshal(model.SuccessCriteria)

	query := `
		INSERT INTO problem_models (
			id, problem_id, version, objectives_json, constraints_json,
			known_facts_json, assumptions_json, unknowns_json,
			dependencies_json, risks_json, success_criteria_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.ExecContext(
		ctx, query,
		model.ID, model.ProblemID, model.Version,
		string(objJSON), string(conJSON), string(kfJSON), string(assJSON),
		string(unkJSON), string(depJSON), string(rskJSON), string(scJSON),
		model.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert problem model: %w", err)
	}

	// Update problem current_model_version if this version is newer
	bumpQuery := `
		UPDATE problems
		SET current_model_version = CASE WHEN ? > current_model_version THEN ? ELSE current_model_version END,
		    updated_at = ?
		WHERE id = ?
	`
	_, err = tx.ExecContext(ctx, bumpQuery, model.Version, model.Version, now.Format(time.RFC3339), model.ProblemID)
	if err != nil {
		return fmt.Errorf("failed to update problem model version: %w", err)
	}

	return tx.Commit()
}

// GetModel retrieves a specific version of a ProblemModel.
func (r *ProblemRepository) GetModel(ctx context.Context, problemID string, version int) (*domainProblem.ProblemModel, error) {
	query := `
		SELECT id, problem_id, version, objectives_json, constraints_json,
		       known_facts_json, assumptions_json, unknowns_json,
		       dependencies_json, risks_json, success_criteria_json, created_at
		FROM problem_models
		WHERE problem_id = ? AND version = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, problemID, version)
	return r.scanModelRow(row)
}

// GetLatestModel retrieves the latest version of ProblemModel for a problem.
func (r *ProblemRepository) GetLatestModel(ctx context.Context, problemID string) (*domainProblem.ProblemModel, error) {
	query := `
		SELECT id, problem_id, version, objectives_json, constraints_json,
		       known_facts_json, assumptions_json, unknowns_json,
		       dependencies_json, risks_json, success_criteria_json, created_at
		FROM problem_models
		WHERE problem_id = ?
		ORDER BY version DESC
		LIMIT 1
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, problemID)
	return r.scanModelRow(row)
}

// ListModels retrieves all versions of ProblemModel for a problem in ascending version order.
func (r *ProblemRepository) ListModels(ctx context.Context, problemID string) ([]*domainProblem.ProblemModel, error) {
	query := `
		SELECT id, problem_id, version, objectives_json, constraints_json,
		       known_facts_json, assumptions_json, unknowns_json,
		       dependencies_json, risks_json, success_criteria_json, created_at
		FROM problem_models
		WHERE problem_id = ?
		ORDER BY version ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, problemID)
	if err != nil {
		return nil, fmt.Errorf("failed to list problem models: %w", err)
	}
	defer rows.Close()

	var models []*domainProblem.ProblemModel
	for rows.Next() {
		m, err := r.scanModelRow(rows)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (r *ProblemRepository) scanModelRow(s rowScanner) (*domainProblem.ProblemModel, error) {
	var m domainProblem.ProblemModel
	var objJSON, conJSON, kfJSON, assJSON, unkJSON, depJSON, rskJSON, scJSON, createdAtStr string
	err := s.Scan(
		&m.ID, &m.ProblemID, &m.Version,
		&objJSON, &conJSON, &kfJSON, &assJSON, &unkJSON, &depJSON, &rskJSON, &scJSON,
		&createdAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domainProblem.ErrProblemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan problem model: %w", err)
	}

	_ = json.Unmarshal([]byte(objJSON), &m.Objectives)
	_ = json.Unmarshal([]byte(conJSON), &m.Constraints)
	_ = json.Unmarshal([]byte(kfJSON), &m.KnownFacts)
	_ = json.Unmarshal([]byte(assJSON), &m.Assumptions)
	_ = json.Unmarshal([]byte(unkJSON), &m.Unknowns)
	_ = json.Unmarshal([]byte(depJSON), &m.Dependencies)
	_ = json.Unmarshal([]byte(rskJSON), &m.Risks)
	_ = json.Unmarshal([]byte(scJSON), &m.SuccessCriteria)
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	return &m, nil
}

// CreateAnalysisRecord writes an immutable analysis record.
func (r *ProblemRepository) CreateAnalysisRecord(ctx context.Context, rec *domainProblem.AnalysisRecord) error {
	query := `
		INSERT INTO analysis_records (
			id, problem_id, model_version, analysis_type, input_refs_json,
			findings_json, conclusions_json, open_questions_json,
			agent_id, execution_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}

	inJSON, _ := json.Marshal(rec.InputRefs)
	findJSON, _ := json.Marshal(rec.Findings)
	concJSON, _ := json.Marshal(rec.Conclusions)
	openJSON, _ := json.Marshal(rec.OpenQuestions)

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		rec.ID, rec.ProblemID, rec.ModelVersion, rec.AnalysisType,
		string(inJSON), string(findJSON), string(concJSON), string(openJSON),
		rec.AgentID, rec.ExecutionID, rec.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert analysis record: %w", err)
	}
	return nil
}

// ListAnalysisRecords returns all analysis records for a problem in chronological order.
func (r *ProblemRepository) ListAnalysisRecords(ctx context.Context, problemID string) ([]*domainProblem.AnalysisRecord, error) {
	query := `
		SELECT id, problem_id, model_version, analysis_type, input_refs_json,
		       findings_json, conclusions_json, open_questions_json,
		       agent_id, execution_id, created_at
		FROM analysis_records
		WHERE problem_id = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, problemID)
	if err != nil {
		return nil, fmt.Errorf("failed to list analysis records: %w", err)
	}
	defer rows.Close()

	var list []*domainProblem.AnalysisRecord
	for rows.Next() {
		var rec domainProblem.AnalysisRecord
		var inJSON, findJSON, concJSON, openJSON, createdAtStr string
		if err := rows.Scan(
			&rec.ID, &rec.ProblemID, &rec.ModelVersion, &rec.AnalysisType,
			&inJSON, &findJSON, &concJSON, &openJSON,
			&rec.AgentID, &rec.ExecutionID, &createdAtStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan analysis record row: %w", err)
		}
		_ = json.Unmarshal([]byte(inJSON), &rec.InputRefs)
		_ = json.Unmarshal([]byte(findJSON), &rec.Findings)
		_ = json.Unmarshal([]byte(concJSON), &rec.Conclusions)
		_ = json.Unmarshal([]byte(openJSON), &rec.OpenQuestions)
		rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		list = append(list, &rec)
	}
	return list, nil
}

// GetAnalysisRecord retrieves a single analysis record by ID.
func (r *ProblemRepository) GetAnalysisRecord(ctx context.Context, id string) (*domainProblem.AnalysisRecord, error) {
	query := `
		SELECT id, problem_id, model_version, analysis_type, input_refs_json,
		       findings_json, conclusions_json, open_questions_json,
		       agent_id, execution_id, created_at
		FROM analysis_records
		WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)

	var rec domainProblem.AnalysisRecord
	var inJSON, findJSON, concJSON, openJSON, createdAtStr string
	err := row.Scan(
		&rec.ID, &rec.ProblemID, &rec.ModelVersion, &rec.AnalysisType,
		&inJSON, &findJSON, &concJSON, &openJSON,
		&rec.AgentID, &rec.ExecutionID, &createdAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("analysis record not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan analysis record: %w", err)
	}
	_ = json.Unmarshal([]byte(inJSON), &rec.InputRefs)
	_ = json.Unmarshal([]byte(findJSON), &rec.Findings)
	_ = json.Unmarshal([]byte(concJSON), &rec.Conclusions)
	_ = json.Unmarshal([]byte(openJSON), &rec.OpenQuestions)
	rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	return &rec, nil
}
