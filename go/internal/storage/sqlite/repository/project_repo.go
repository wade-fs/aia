package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/storage/sqlite"
)

type ProjectRepository struct {
	db *sqlite.DB
}

func NewProjectRepository(db *sqlite.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, p *project.Project) error {
	query := `
		INSERT INTO projects (
			id, name, description, workspace, domain_type, policy_id, status, created_at, updated_at,
			primary_goal_id, latest_snapshot_id, pending_issues_json
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.PendingIssuesJSON == "" {
		p.PendingIssuesJSON = "[]"
	}
	if p.DomainType == "" {
		p.DomainType = project.DomainGeneral
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		p.ID, p.Name, p.Description, p.Workspace, string(p.DomainType), p.PolicyID, string(p.Status),
		p.CreatedAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339),
		p.PrimaryGoalID, p.LatestSnapshotID, p.PendingIssuesJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	return nil
}

func (r *ProjectRepository) Get(ctx context.Context, id domain.ID) (*project.Project, error) {
	query := `
		SELECT id, name, description, workspace, domain_type, policy_id, status, created_at, updated_at,
		       primary_goal_id, latest_snapshot_id, pending_issues_json
		FROM projects WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)
	p, err := scanProject(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return p, nil
}

func (r *ProjectRepository) List(ctx context.Context) ([]project.Project, error) {
	query := `
		SELECT id, name, description, workspace, domain_type, policy_id, status, created_at, updated_at,
		       primary_goal_id, latest_snapshot_id, pending_issues_json
		FROM projects ORDER BY created_at DESC
	`
	return r.queryProjects(ctx, query)
}

// ListByDomain returns all ACTIVE projects for a given domain type.
// Used by app.go at boot to seed domain observers without hardcoded paths.
func (r *ProjectRepository) ListByDomain(ctx context.Context, dt project.DomainType) ([]project.Project, error) {
	query := `
		SELECT id, name, description, workspace, domain_type, policy_id, status, created_at, updated_at,
		       primary_goal_id, latest_snapshot_id, pending_issues_json
		FROM projects
		WHERE domain_type = ? AND status = 'ACTIVE'
		ORDER BY created_at ASC
	`
	return r.queryProjects(ctx, query, string(dt))
}

func (r *ProjectRepository) queryProjects(ctx context.Context, query string, args ...any) ([]project.Project, error) {
	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	var projects []project.Project
	for rows.Next() {
		p, err := scanProject(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, *p)
	}
	return projects, nil
}

// scanProject is a generic scanner that works for both sql.Row.Scan and sql.Rows.Scan.
func scanProject(scan func(dest ...any) error) (*project.Project, error) {
	var p project.Project
	var domainType, status, createdAt, updatedAt string
	var policyID, primGoal, snapID, issues sql.NullString

	err := scan(&p.ID, &p.Name, &p.Description, &p.Workspace, &domainType, &policyID, &status,
		&createdAt, &updatedAt, &primGoal, &snapID, &issues)
	if err != nil {
		return nil, err
	}

	p.DomainType = project.DomainType(domainType)
	p.Status = project.ProjectStatus(status)
	if policyID.Valid {
		p.PolicyID = policyID.String
	}
	if primGoal.Valid {
		p.PrimaryGoalID = &primGoal.String
	}
	if snapID.Valid {
		p.LatestSnapshotID = &snapID.String
	}
	if issues.Valid {
		p.PendingIssuesJSON = issues.String
	} else {
		p.PendingIssuesJSON = "[]"
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &p, nil
}

func (r *ProjectRepository) UpdateStatus(ctx context.Context, id domain.ID, status project.ProjectStatus) error {
	query := `UPDATE projects SET status = ?, updated_at = ? WHERE id = ?`
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.RawDB().ExecContext(ctx, query, string(status), now, id)
	if err != nil {
		return fmt.Errorf("failed to update project status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	return nil
}

// UpdateContinuity updates the continuity fields of a project.
func (r *ProjectRepository) UpdateContinuity(ctx context.Context, id domain.ID, primaryGoalID *string, latestSnapshotID *string, pendingIssuesJSON string) error {
	query := `UPDATE projects SET primary_goal_id = ?, latest_snapshot_id = ?, pending_issues_json = ?, updated_at = ? WHERE id = ?`
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.RawDB().ExecContext(ctx, query, primaryGoalID, latestSnapshotID, pendingIssuesJSON, now, id)
	if err != nil {
		return fmt.Errorf("failed to update project continuity: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	return nil
}

// Delete removes a project record and safely unlinks all associated references
// so foreign key constraints are preserved while maintaining audit trail entities.
func (r *ProjectRepository) Delete(ctx context.Context, id domain.ID) error {
	tx, err := r.db.RawDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	nullifyQueries := []string{
		`UPDATE tasks SET project_id = NULL WHERE project_id = ?`,
		`UPDATE memories SET project_id = NULL WHERE project_id = ?`,
		`UPDATE experiments SET project_id = NULL WHERE project_id = ?`,
		`UPDATE delegation_requests SET source_project_id = NULL WHERE source_project_id = ?`,
		`UPDATE decisions SET project_id = NULL WHERE project_id = ?`,
		`UPDATE context_snapshots SET project_id = NULL WHERE project_id = ?`,
		`UPDATE outcome_evaluations SET project_id = NULL WHERE project_id = ?`,
		`UPDATE learning_records SET project_id = NULL WHERE project_id = ?`,
		`UPDATE candidate_initiatives SET project_id = NULL WHERE project_id = ?`,
	}
	for _, q := range nullifyQueries {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return fmt.Errorf("failed to nullify project reference: %w", err)
		}
	}

	deleteQueries := []string{
		`DELETE FROM goal_project_links WHERE project_id = ?`,
		`DELETE FROM writing_clarifications WHERE project_id = ?`,
		`DELETE FROM writing_briefs WHERE project_id = ?`,
		`DELETE FROM brief_field_history WHERE project_id = ?`,
		`DELETE FROM project_briefs WHERE project_id = ?`,
	}
	for _, q := range deleteQueries {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return fmt.Errorf("failed to delete project link: %w", err)
		}
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("project not found: %s", id)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit project deletion: %w", err)
	}
	return nil
}
