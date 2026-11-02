package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domainPlan "github.com/wade/aia/internal/domain/plan"
	"github.com/wade/aia/internal/storage/sqlite"
)

// PlanRepository manages versioned Plan and PlanChangeRecord entities in SQLite (AIA-22 §6.4, §6.5, §12).
type PlanRepository struct {
	db *sqlite.DB
}

// NewPlanRepository creates a new SQLite plan repository.
func NewPlanRepository(db *sqlite.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

var _ domainPlan.Repository = (*PlanRepository)(nil)

// Create records a new versioned Plan and updates the parent Problem's current_plan_version.
func (r *PlanRepository) Create(ctx context.Context, p *domainPlan.Plan) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("%w: %v", domainPlan.ErrInvalidPlan, err)
	}

	tx, err := r.db.RawDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	wiJSON, _ := json.Marshal(p.WorkItems)
	depJSON, _ := json.Marshal(p.Dependencies)
	acJSON, _ := json.Marshal(p.AcceptanceCriteria)

	query := `
		INSERT INTO plans (
			id, problem_id, version, objective, strategy, status,
			work_items_json, dependencies_json, acceptance_criteria_json,
			supersedes_plan_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.ExecContext(
		ctx, query,
		p.ID, p.ProblemID, p.Version, p.Objective, p.Strategy, string(p.Status),
		string(wiJSON), string(depJSON), string(acJSON),
		p.SupersedesPlanID, p.CreatedAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert plan: %w", err)
	}

	// Bump current_plan_version on Problem
	bumpQuery := `
		UPDATE problems
		SET current_plan_version = CASE WHEN ? > current_plan_version THEN ? ELSE current_plan_version END,
		    updated_at = ?
		WHERE id = ?
	`
	_, err = tx.ExecContext(ctx, bumpQuery, p.Version, p.Version, now.Format(time.RFC3339), p.ProblemID)
	if err != nil {
		return fmt.Errorf("failed to update problem current_plan_version: %w", err)
	}

	return tx.Commit()
}

// Get retrieves a Plan by ID.
func (r *PlanRepository) Get(ctx context.Context, id string) (*domainPlan.Plan, error) {
	query := `
		SELECT id, problem_id, version, objective, strategy, status,
		       work_items_json, dependencies_json, acceptance_criteria_json,
		       supersedes_plan_id, created_at, updated_at
		FROM plans
		WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)
	return r.scanPlanRow(row)
}

// GetByProblemAndVersion retrieves a specific version of a Plan for a problem.
func (r *PlanRepository) GetByProblemAndVersion(ctx context.Context, problemID string, version int) (*domainPlan.Plan, error) {
	query := `
		SELECT id, problem_id, version, objective, strategy, status,
		       work_items_json, dependencies_json, acceptance_criteria_json,
		       supersedes_plan_id, created_at, updated_at
		FROM plans
		WHERE problem_id = ? AND version = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, problemID, version)
	return r.scanPlanRow(row)
}

// GetLatestByProblem retrieves the highest version of Plan for a problem.
func (r *PlanRepository) GetLatestByProblem(ctx context.Context, problemID string) (*domainPlan.Plan, error) {
	query := `
		SELECT id, problem_id, version, objective, strategy, status,
		       work_items_json, dependencies_json, acceptance_criteria_json,
		       supersedes_plan_id, created_at, updated_at
		FROM plans
		WHERE problem_id = ?
		ORDER BY version DESC
		LIMIT 1
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, problemID)
	return r.scanPlanRow(row)
}

// ListByProblem lists all plans for a problem in ascending version order.
func (r *PlanRepository) ListByProblem(ctx context.Context, problemID string) ([]*domainPlan.Plan, error) {
	query := `
		SELECT id, problem_id, version, objective, strategy, status,
		       work_items_json, dependencies_json, acceptance_criteria_json,
		       supersedes_plan_id, created_at, updated_at
		FROM plans
		WHERE problem_id = ?
		ORDER BY version ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, problemID)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans by problem: %w", err)
	}
	defer rows.Close()

	var list []*domainPlan.Plan
	for rows.Next() {
		p, err := r.scanPlanRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

// UpdateStatus changes the status of a Plan.
func (r *PlanRepository) UpdateStatus(ctx context.Context, id string, status domainPlan.PlanStatus) error {
	query := `
		UPDATE plans
		SET status = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now().UTC()
	res, err := r.db.RawDB().ExecContext(ctx, query, string(status), now.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to update plan status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domainPlan.ErrPlanNotFound
	}
	return nil
}

// UpdateWorkItems updates the serialized work items of an existing Plan.
func (r *PlanRepository) UpdateWorkItems(ctx context.Context, id string, workItems []domainPlan.WorkItemDef) error {
	wiJSON, err := json.Marshal(workItems)
	if err != nil {
		return fmt.Errorf("failed to marshal work items: %w", err)
	}
	query := `
		UPDATE plans
		SET work_items_json = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now().UTC()
	res, err := r.db.RawDB().ExecContext(ctx, query, string(wiJSON), now.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to update plan work items: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domainPlan.ErrPlanNotFound
	}
	return nil
}

// RecordChange records a formal change log between plan versions (AIA-22 §6.5).
func (r *PlanRepository) RecordChange(ctx context.Context, change *domainPlan.PlanChangeRecord) error {
	query := `
		INSERT INTO plan_changes (
			id, plan_id, from_version, to_version, reason,
			evidence_refs_json, affected_work_items_json, cost_risk_impact,
			requires_human_approval, approved_by, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if change.CreatedAt.IsZero() {
		change.CreatedAt = now
	}

	evJSON, _ := json.Marshal(change.EvidenceRefs)
	wiJSON, _ := json.Marshal(change.AffectedWorkItems)
	reqApprInt := 0
	if change.RequiresHumanApproval {
		reqApprInt = 1
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		change.ID, change.PlanID, change.FromVersion, change.ToVersion, change.Reason,
		string(evJSON), string(wiJSON), change.CostRiskImpact,
		reqApprInt, change.ApprovedBy, change.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert plan change: %w", err)
	}
	return nil
}

// ListChanges lists all change records for a plan.
func (r *PlanRepository) ListChanges(ctx context.Context, planID string) ([]*domainPlan.PlanChangeRecord, error) {
	query := `
		SELECT id, plan_id, from_version, to_version, reason,
		       evidence_refs_json, affected_work_items_json, cost_risk_impact,
		       requires_human_approval, approved_by, created_at
		FROM plan_changes
		WHERE plan_id = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan changes: %w", err)
	}
	defer rows.Close()

	var list []*domainPlan.PlanChangeRecord
	for rows.Next() {
		var c domainPlan.PlanChangeRecord
		var evJSON, wiJSON, createdAtStr string
		var reqApprInt int
		if err := rows.Scan(
			&c.ID, &c.PlanID, &c.FromVersion, &c.ToVersion, &c.Reason,
			&evJSON, &wiJSON, &c.CostRiskImpact,
			&reqApprInt, &c.ApprovedBy, &createdAtStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan plan change row: %w", err)
		}
		_ = json.Unmarshal([]byte(evJSON), &c.EvidenceRefs)
		_ = json.Unmarshal([]byte(wiJSON), &c.AffectedWorkItems)
		c.RequiresHumanApproval = (reqApprInt == 1)
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		list = append(list, &c)
	}
	return list, nil
}

func (r *PlanRepository) scanPlanRow(s rowScanner) (*domainPlan.Plan, error) {
	var p domainPlan.Plan
	var statusStr, wiJSON, depJSON, acJSON, createdAtStr, updatedAtStr string
	err := s.Scan(
		&p.ID, &p.ProblemID, &p.Version, &p.Objective, &p.Strategy, &statusStr,
		&wiJSON, &depJSON, &acJSON,
		&p.SupersedesPlanID, &createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domainPlan.ErrPlanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan plan: %w", err)
	}

	p.Status = domainPlan.PlanStatus(statusStr)
	_ = json.Unmarshal([]byte(wiJSON), &p.WorkItems)
	_ = json.Unmarshal([]byte(depJSON), &p.Dependencies)
	_ = json.Unmarshal([]byte(acJSON), &p.AcceptanceCriteria)
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &p, nil
}
