package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/storage/sqlite"
)

type TaskFilter struct {
	ProjectID    domain.ID
	ParentTaskID domain.ID
	Status       task.TaskStatus
	Limit        int
}

// TaskChangeHook is invoked after a task change is committed. It is used by
// the runtime to publish notifications (event bus); it must not block.
type TaskChangeHook func(ctx context.Context, taskID domain.ID, status task.TaskStatus, created bool)

type TaskRepository struct {
	db   *sqlite.DB
	hook TaskChangeHook
}

// SetChangeHook registers a post-commit change hook.
func (r *TaskRepository) SetChangeHook(h TaskChangeHook) { r.hook = h }

func NewTaskRepository(db *sqlite.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, t *task.Task, dependencies []domain.ID) error {
	if err := r.create(ctx, t, dependencies); err != nil {
		return err
	}
	if r.hook != nil {
		r.hook(ctx, t.ID, t.Status, true)
	}
	return nil
}

func (r *TaskRepository) create(ctx context.Context, t *task.Task, dependencies []domain.ID) error {
	// Check dependency cycles
	if len(dependencies) > 0 {
		graph, err := r.GetFullDependencyGraph(ctx)
		if err == nil {
			for _, dep := range dependencies {
				if dep == t.ID || task.DetectCycle(graph, t.ID, dep) {
					return fmt.Errorf("%w: cannot depend on %s", task.ErrDependencyCycle, dep)
				}
			}
		}
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		query := `
			INSERT INTO tasks (
				id, project_id, parent_task_id, agent_id, title, description,
				status, priority, risk, acceptance_json, retry_count, max_retries,
				created_at, updated_at, started_at, completed_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		acceptJSON, err := json.Marshal(t.Acceptance)
		if err != nil {
			acceptJSON = []byte("[]")
		}

		now := time.Now().UTC()
		if t.CreatedAt.IsZero() {
			t.CreatedAt = now
		}
		t.UpdatedAt = now

		var startedAt, completedAt sql.NullString
		if t.StartedAt != nil {
			startedAt = sql.NullString{String: t.StartedAt.Format(time.RFC3339), Valid: true}
		}
		if t.CompletedAt != nil {
			completedAt = sql.NullString{String: t.CompletedAt.Format(time.RFC3339), Valid: true}
		}

		var projID, parentID, agentID sql.NullString
		if t.ProjectID != "" {
			projID = sql.NullString{String: t.ProjectID, Valid: true}
		}
		if t.ParentTaskID != "" {
			parentID = sql.NullString{String: t.ParentTaskID, Valid: true}
		}
		if t.AgentID != "" {
			agentID = sql.NullString{String: t.AgentID, Valid: true}
		}

		_, err = tx.ExecContext(
			ctx, query,
			t.ID, projID, parentID, agentID, t.Title, t.Description,
			string(t.Status), t.Priority, string(t.Risk), string(acceptJSON), t.RetryCount, t.MaxRetries,
			t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339), startedAt, completedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert task: %w", err)
		}

		depQuery := `INSERT INTO task_dependencies (task_id, depends_on_task_id) VALUES (?, ?)`
		for _, dep := range dependencies {
			if _, err := tx.ExecContext(ctx, depQuery, t.ID, dep); err != nil {
				return fmt.Errorf("failed to insert task dependency %s -> %s: %w", t.ID, dep, err)
			}
		}

		return nil
	})
}

func (r *TaskRepository) Get(ctx context.Context, id domain.ID) (*task.Task, error) {
	query := `
		SELECT id, project_id, parent_task_id, agent_id, title, description,
		       status, priority, risk, acceptance_json, retry_count, max_retries,
		       created_at, updated_at, started_at, completed_at
		FROM tasks WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)

	var t task.Task
	var parentID, agentID, startedAt, completedAt sql.NullString
	var status, risk, acceptJSON, createdAt, updatedAt string

	err := row.Scan(
		&t.ID, &t.ProjectID, &parentID, &agentID, &t.Title, &t.Description,
		&status, &t.Priority, &risk, &acceptJSON, &t.RetryCount, &t.MaxRetries,
		&createdAt, &updatedAt, &startedAt, &completedAt,
	)
	if err == sql.ErrNoRows && len(id) >= 6 {
		prefixQuery := `
			SELECT id, project_id, parent_task_id, agent_id, title, description,
			       status, priority, risk, acceptance_json, retry_count, max_retries,
			       created_at, updated_at, started_at, completed_at
			FROM tasks WHERE id LIKE ? || '%' LIMIT 1
		`
		row = r.db.RawDB().QueryRowContext(ctx, prefixQuery, id)
		err = row.Scan(
			&t.ID, &t.ProjectID, &parentID, &agentID, &t.Title, &t.Description,
			&status, &t.Priority, &risk, &acceptJSON, &t.RetryCount, &t.MaxRetries,
			&createdAt, &updatedAt, &startedAt, &completedAt,
		)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	t.Status = task.TaskStatus(status)
	t.Risk = domain.RiskLevel(risk)
	if parentID.Valid {
		t.ParentTaskID = parentID.String
	}
	if agentID.Valid {
		t.AgentID = agentID.String
	}
	_ = json.Unmarshal([]byte(acceptJSON), &t.Acceptance)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if startedAt.Valid {
		st, _ := time.Parse(time.RFC3339, startedAt.String)
		t.StartedAt = &st
	}
	if completedAt.Valid {
		ct, _ := time.Parse(time.RFC3339, completedAt.String)
		t.CompletedAt = &ct
	}

	return &t, nil
}

func (r *TaskRepository) List(ctx context.Context, filter TaskFilter) ([]task.Task, error) {
	query := `
		SELECT id, project_id, parent_task_id, agent_id, title, description,
		       status, priority, risk, acceptance_json, retry_count, max_retries,
		       created_at, updated_at, started_at, completed_at
		FROM tasks WHERE 1=1
	`
	var args []any
	if filter.ProjectID != "" {
		query += " AND project_id = ?"
		args = append(args, filter.ProjectID)
	}
	if filter.ParentTaskID != "" {
		query += " AND parent_task_id = ?"
		args = append(args, filter.ParentTaskID)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	query += " ORDER BY priority DESC, created_at ASC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		var t task.Task
		var parentID, agentID, startedAt, completedAt sql.NullString
		var status, risk, acceptJSON, createdAt, updatedAt string

		if err := rows.Scan(
			&t.ID, &t.ProjectID, &parentID, &agentID, &t.Title, &t.Description,
			&status, &t.Priority, &risk, &acceptJSON, &t.RetryCount, &t.MaxRetries,
			&createdAt, &updatedAt, &startedAt, &completedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		t.Status = task.TaskStatus(status)
		t.Risk = domain.RiskLevel(risk)
		if parentID.Valid {
			t.ParentTaskID = parentID.String
		}
		if agentID.Valid {
			t.AgentID = agentID.String
		}
		_ = json.Unmarshal([]byte(acceptJSON), &t.Acceptance)
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if startedAt.Valid {
			st, _ := time.Parse(time.RFC3339, startedAt.String)
			t.StartedAt = &st
		}
		if completedAt.Valid {
			ct, _ := time.Parse(time.RFC3339, completedAt.String)
			t.CompletedAt = &ct
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id domain.ID, to task.TaskStatus) error {
	if err := r.updateStatus(ctx, id, to); err != nil {
		return err
	}
	if r.hook != nil {
		r.hook(ctx, id, to, false)
	}
	return nil
}

func (r *TaskRepository) updateStatus(ctx context.Context, id domain.ID, to task.TaskStatus) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		var currStatusStr string
		err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", id).Scan(&currStatusStr)
		if err != nil {
			return fmt.Errorf("failed to query task status: %w", err)
		}

		from := task.TaskStatus(currStatusStr)
		if err := task.ValidateTransition(from, to); err != nil {
			return err
		}

		now := time.Now().UTC()
		nowStr := now.Format(time.RFC3339)

		var updateQuery string
		var args []any
		if to == task.StatusRunning {
			updateQuery = "UPDATE tasks SET status = ?, started_at = COALESCE(started_at, ?), updated_at = ? WHERE id = ?"
			args = []any{string(to), nowStr, nowStr, id}
		} else if to == task.StatusCompleted || to == task.StatusFailed || to == task.StatusCancelled {
			updateQuery = "UPDATE tasks SET status = ?, completed_at = ?, updated_at = ? WHERE id = ?"
			args = []any{string(to), nowStr, nowStr, id}
		} else {
			updateQuery = "UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?"
			args = []any{string(to), nowStr, id}
		}

		_, err = tx.ExecContext(ctx, updateQuery, args...)
		if err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

		if to == task.StatusCompleted {
			promoteQuery := `
				UPDATE tasks
				SET status = 'READY', updated_at = ?
				WHERE status = 'WAITING_DEPENDENCY'
				  AND id IN (SELECT task_id FROM task_dependencies WHERE depends_on_task_id = ?)
				  AND NOT EXISTS (
					  SELECT 1 FROM task_dependencies td
					  JOIN tasks dep ON td.depends_on_task_id = dep.id
					  WHERE td.task_id = tasks.id AND dep.status != 'COMPLETED'
				  )
			`
			_, _ = tx.ExecContext(ctx, promoteQuery, nowStr, id)
		}
		return nil
	})
}

func (r *TaskRepository) Update(ctx context.Context, t *task.Task) error {
	query := `
		UPDATE tasks SET
			agent_id = ?, title = ?, description = ?, status = ?, priority = ?,
			risk = ?, acceptance_json = ?, retry_count = ?, max_retries = ?,
			updated_at = ?
		WHERE id = ?
	`
	acceptJSON, _ := json.Marshal(t.Acceptance)
	now := time.Now().UTC().Format(time.RFC3339)

	var agentID sql.NullString
	if t.AgentID != "" {
		agentID = sql.NullString{String: t.AgentID, Valid: true}
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		agentID, t.Title, t.Description, string(t.Status), t.Priority,
		string(t.Risk), string(acceptJSON), t.RetryCount, t.MaxRetries,
		now, t.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}

func (r *TaskRepository) GetDependencies(ctx context.Context, taskID domain.ID) ([]domain.ID, error) {
	query := `SELECT depends_on_task_id FROM task_dependencies WHERE task_id = ?`
	rows, err := r.db.RawDB().QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}
	defer rows.Close()

	var deps []domain.ID
	for rows.Next() {
		var dep domain.ID
		if err := rows.Scan(&dep); err != nil {
			return nil, err
		}
		deps = append(deps, dep)
	}
	return deps, nil
}

func (r *TaskRepository) AreDependenciesCompleted(ctx context.Context, taskID domain.ID) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM task_dependencies td
		JOIN tasks t ON td.depends_on_task_id = t.id
		WHERE td.task_id = ? AND t.status != 'COMPLETED'
	`
	var count int
	err := r.db.RawDB().QueryRowContext(ctx, query, taskID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check dependencies status: %w", err)
	}
	return count == 0, nil
}

func (r *TaskRepository) GetFullDependencyGraph(ctx context.Context) (map[domain.ID][]domain.ID, error) {
	query := `SELECT task_id, depends_on_task_id FROM task_dependencies`
	rows, err := r.db.RawDB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	graph := make(map[domain.ID][]domain.ID)
	for rows.Next() {
		var from, to domain.ID
		if err := rows.Scan(&from, &to); err != nil {
			return nil, err
		}
		graph[from] = append(graph[from], to)
	}
	return graph, nil
}
