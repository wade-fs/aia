package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/storage/sqlite"
)

type GoalRepository struct {
	db *sqlite.DB
}

func NewGoalRepository(db *sqlite.DB) *GoalRepository {
	return &GoalRepository{db: db}
}

// Create inserts a new goal, preventing circular parent chains.
func (r *GoalRepository) Create(ctx context.Context, g *goal.Goal) error {
	if g.CreatedAt.IsZero() {
		g.CreatedAt = time.Now().UTC()
	}
	if g.UpdatedAt.IsZero() {
		g.UpdatedAt = g.CreatedAt
	}
	if g.Priority == 0 {
		g.Priority = 5
	}
	if g.Status == "" {
		g.Status = goal.StatusDraft
	}

	// 循環參照防護：若有指定 parent_goal_id，確保不可指向自身
	if g.ParentGoalID != nil && *g.ParentGoalID == g.ID {
		return fmt.Errorf("circular reference error: goal cannot be its own parent")
	}

	query := `
	INSERT INTO goals (
		id, parent_goal_id, name, description, level, status, target_metric_json, current_metric_json, priority, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	createdStr := g.CreatedAt.UTC().Format(time.RFC3339)
	updatedStr := g.UpdatedAt.UTC().Format(time.RFC3339)

	_, err := r.db.RawDB().ExecContext(ctx, query,
		g.ID, g.ParentGoalID, g.Name, g.Description, g.Level, g.Status,
		g.TargetMetricJSON, g.CurrentMetricJSON, g.Priority, createdStr, updatedStr,
	)
	return err
}

// Get retrieves a goal by ID.
func (r *GoalRepository) Get(ctx context.Context, id string) (*goal.Goal, error) {
	query := `
	SELECT id, parent_goal_id, name, description, level, status, target_metric_json, current_metric_json, priority, created_at, updated_at
	FROM goals WHERE id = ?`

	g := &goal.Goal{}
	var parentID, targetMetric, currentMetric sql.NullString
	var createdStr, updatedStr string

	err := r.db.RawDB().QueryRowContext(ctx, query, id).Scan(
		&g.ID, &parentID, &g.Name, &g.Description, &g.Level, &g.Status,
		&targetMetric, &currentMetric, &g.Priority, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		g.ParentGoalID = &parentID.String
	}
	g.TargetMetricJSON = targetMetric.String
	g.CurrentMetricJSON = currentMetric.String
	g.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	g.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)

	return g, nil
}

// UpdateStatus atomically updates goal status and appends a goal revision record.
func (r *GoalRepository) UpdateStatus(ctx context.Context, id string, newStatus goal.GoalStatus, reason, decisionID string) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. 取得當前狀態
		var oldStatus string
		var oldMetric sql.NullString
		err := tx.QueryRowContext(ctx, "SELECT status, current_metric_json FROM goals WHERE id = ?", id).Scan(&oldStatus, &oldMetric)
		if err != nil {
			return fmt.Errorf("failed to load goal before status update: %w", err)
		}

		// 2. 終態不可再逆轉保護
		if oldStatus == string(goal.StatusSuperseded) || oldStatus == string(goal.StatusAbandoned) {
			return fmt.Errorf("invalid state transition: cannot transition from terminal status %s", oldStatus)
		}

		// 3. 取得最大 revision_number
		var maxRev sql.NullInt64
		err = tx.QueryRowContext(ctx, "SELECT MAX(revision_number) FROM goal_revisions WHERE goal_id = ?", id).Scan(&maxRev)
		if err != nil {
			return fmt.Errorf("failed to query goal revisions: %w", err)
		}
		nextRev := int(maxRev.Int64) + 1

		now := time.Now().UTC()
		nowStr := now.Format(time.RFC3339)

		// 4. 更新 goals 表
		_, err = tx.ExecContext(ctx, "UPDATE goals SET status = ?, updated_at = ? WHERE id = ?", newStatus, nowStr, id)
		if err != nil {
			return fmt.Errorf("failed to update goal status: %w", err)
		}

		// 5. 插入 goal_revisions
		revID := fmt.Sprintf("%s-rev-%d", id, nextRev)
		prevState := fmt.Sprintf(`{"status":"%s"}`, oldStatus)
		newState := fmt.Sprintf(`{"status":"%s"}`, newStatus)

		var decID *string
		if decisionID != "" {
			decID = &decisionID
		}

		revQuery := `
		INSERT INTO goal_revisions (
			id, goal_id, revision_number, previous_state_json, new_state_json, reason, decision_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

		_, err = tx.ExecContext(ctx, revQuery,
			revID, id, nextRev, prevState, newState, reason, decID, nowStr,
		)
		if err != nil {
			return fmt.Errorf("failed to insert goal revision: %w", err)
		}

		return nil
	})
}

// LinkProject creates an M:N association between a Goal and a Project.
func (r *GoalRepository) LinkProject(ctx context.Context, goalID, projectID string, linkType goal.GoalProjectLinkType) error {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	query := `
	INSERT INTO goal_project_links (goal_id, project_id, link_type, created_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(goal_id, project_id) DO UPDATE SET link_type = excluded.link_type`

	_, err := r.db.RawDB().ExecContext(ctx, query, goalID, projectID, linkType, nowStr)
	return err
}

// ListByProject returns all goals linked to a specific project.
func (r *GoalRepository) ListByProject(ctx context.Context, projectID string) ([]*goal.Goal, error) {
	query := `
	SELECT g.id, g.parent_goal_id, g.name, g.description, g.level, g.status, g.target_metric_json, g.current_metric_json, g.priority, g.created_at, g.updated_at
	FROM goals g
	JOIN goal_project_links l ON g.id = l.goal_id
	WHERE l.project_id = ?
	ORDER BY g.priority DESC`

	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []*goal.Goal
	for rows.Next() {
		g := &goal.Goal{}
		var parentID, targetMetric, currentMetric sql.NullString
		var createdStr, updatedStr string

		err := rows.Scan(
			&g.ID, &parentID, &g.Name, &g.Description, &g.Level, &g.Status,
			&targetMetric, &currentMetric, &g.Priority, &createdStr, &updatedStr,
		)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			g.ParentGoalID = &parentID.String
		}
		g.TargetMetricJSON = targetMetric.String
		g.CurrentMetricJSON = currentMetric.String
		g.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		g.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		goals = append(goals, g)
	}

	return goals, rows.Err()
}
