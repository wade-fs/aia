package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/storage/sqlite"
)

type ContinuityRepository struct {
	db *sqlite.DB
}

func NewContinuityRepository(db *sqlite.DB) *ContinuityRepository {
	return &ContinuityRepository{db: db}
}

// CreateCycle inserts a new autonomy cycle.
func (r *ContinuityRepository) CreateCycle(ctx context.Context, c *continuity.AutonomyCycle) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	if c.StartedAt.IsZero() {
		c.StartedAt = time.Now().UTC()
	}

	query := `
	INSERT INTO autonomy_cycles (
		id, cycle_number, trigger_type, trigger_ref_id, status, context_snapshot_id, summary, started_at, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	startedStr := c.StartedAt.UTC().Format(time.RFC3339)
	createdStr := c.CreatedAt.UTC().Format(time.RFC3339)

	_, err := r.db.RawDB().ExecContext(ctx, query,
		c.ID, c.CycleNumber, c.TriggerType, c.TriggerRefID, c.Status, c.ContextSnapshotID, c.Summary,
		startedStr, createdStr,
	)
	return err
}

// UpdateCycleStatus updates an active cycle's status, summary, and optional completion timestamp.
func (r *ContinuityRepository) UpdateCycleStatus(ctx context.Context, id string, status continuity.CycleStatus, summary string, completedAt *time.Time) error {
	var completedStr *string
	if completedAt != nil {
		s := completedAt.UTC().Format(time.RFC3339)
		completedStr = &s
	}

	query := `
	UPDATE autonomy_cycles 
	SET status = ?, summary = ?, completed_at = ?
	WHERE id = ?`

	_, err := r.db.RawDB().ExecContext(ctx, query, status, summary, completedStr, id)
	return err
}

// GetCycle retrieves an autonomy cycle by ID.
func (r *ContinuityRepository) GetCycle(ctx context.Context, id string) (*continuity.AutonomyCycle, error) {
	query := `
	SELECT id, cycle_number, trigger_type, trigger_ref_id, status, context_snapshot_id, summary, started_at, completed_at, created_at
	FROM autonomy_cycles WHERE id = ?`

	c := &continuity.AutonomyCycle{}
	var startedStr, createdStr string
	var completedStr, triggerRef, snapshotID, summary sql.NullString

	err := r.db.RawDB().QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.CycleNumber, &c.TriggerType, &triggerRef, &c.Status, &snapshotID, &summary,
		&startedStr, &completedStr, &createdStr,
	)
	if err != nil {
		return nil, err
	}

	c.StartedAt, _ = time.Parse(time.RFC3339, startedStr)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if completedStr.Valid {
		t, _ := time.Parse(time.RFC3339, completedStr.String)
		c.CompletedAt = &t
	}
	c.TriggerRefID = triggerRef.String
	c.ContextSnapshotID = snapshotID.String
	c.Summary = summary.String

	return c, nil
}

// CreateSnapshot saves a workspace context snapshot.
func (r *ContinuityRepository) CreateSnapshot(ctx context.Context, s *continuity.ContextSnapshot) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}

	query := `
	INSERT INTO context_snapshots (
		id, scope, project_id, snapshot_type, workspace_state_json, active_tasks_json, recent_decisions_json, pending_issues_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	createdStr := s.CreatedAt.UTC().Format(time.RFC3339)

	var projID *string
	if s.ProjectID != "" {
		projID = &s.ProjectID
	}

	_, err := r.db.RawDB().ExecContext(ctx, query,
		s.ID, s.Scope, projID, s.SnapshotType, s.WorkspaceStateJSON, s.ActiveTasksJSON, s.RecentDecisionsJSON, s.PendingIssuesJSON, createdStr,
	)
	return err
}

// GetSnapshot retrieves a context snapshot by ID.
func (r *ContinuityRepository) GetSnapshot(ctx context.Context, id string) (*continuity.ContextSnapshot, error) {
	query := `
	SELECT id, scope, project_id, snapshot_type, workspace_state_json, active_tasks_json, recent_decisions_json, pending_issues_json, created_at
	FROM context_snapshots WHERE id = ?`

	s := &continuity.ContextSnapshot{}
	var createdStr string
	var projID sql.NullString

	err := r.db.RawDB().QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Scope, &projID, &s.SnapshotType, &s.WorkspaceStateJSON, &s.ActiveTasksJSON, &s.RecentDecisionsJSON, &s.PendingIssuesJSON, &createdStr,
	)
	if err != nil {
		return nil, err
	}

	s.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	s.ProjectID = projID.String
	return s, nil
}

// GetLatestSnapshotByProject returns the newest snapshot for a specific project.
func (r *ContinuityRepository) GetLatestSnapshotByProject(ctx context.Context, projectID string) (*continuity.ContextSnapshot, error) {
	query := `
	SELECT id, scope, project_id, snapshot_type, workspace_state_json, active_tasks_json, recent_decisions_json, pending_issues_json, created_at
	FROM context_snapshots 
	WHERE project_id = ? 
	ORDER BY created_at DESC LIMIT 1`

	s := &continuity.ContextSnapshot{}
	var createdStr string
	var projID sql.NullString

	err := r.db.RawDB().QueryRowContext(ctx, query, projectID).Scan(
		&s.ID, &s.Scope, &projID, &s.SnapshotType, &s.WorkspaceStateJSON, &s.ActiveTasksJSON, &s.RecentDecisionsJSON, &s.PendingIssuesJSON, &createdStr,
	)
	if err != nil {
		return nil, err
	}

	s.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	s.ProjectID = projID.String
	return s, nil
}
