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
			id, name, description, workspace, policy_id, status, created_at, updated_at,
			primary_goal_id, latest_snapshot_id, pending_issues_json
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.PendingIssuesJSON == "" {
		p.PendingIssuesJSON = "[]"
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		p.ID, p.Name, p.Description, p.Workspace, p.PolicyID, string(p.Status),
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
		SELECT id, name, description, workspace, policy_id, status, created_at, updated_at,
		       primary_goal_id, latest_snapshot_id, pending_issues_json
		FROM projects WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)

	var p project.Project
	var status, createdAt, updatedAt string
	var policyID, primGoal, snapID, issues sql.NullString

	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Workspace, &policyID, &status, &createdAt, &updatedAt, &primGoal, &snapID, &issues)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

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

func (r *ProjectRepository) List(ctx context.Context) ([]project.Project, error) {
	query := `
		SELECT id, name, description, workspace, policy_id, status, created_at, updated_at,
		       primary_goal_id, latest_snapshot_id, pending_issues_json
		FROM projects ORDER BY created_at DESC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []project.Project
	for rows.Next() {
		var p project.Project
		var status, createdAt, updatedAt string
		var policyID, primGoal, snapID, issues sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Workspace, &policyID, &status, &createdAt, &updatedAt, &primGoal, &snapID, &issues); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
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
		projects = append(projects, p)
	}
	return projects, nil
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
