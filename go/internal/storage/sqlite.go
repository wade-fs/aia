package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrNotFound = errors.New("record not found")
)

type TaskRecord struct {
	ID                string
	Title             string
	Description       string
	Status            string
	WorkerType        string
	WorkspaceID       string
	CreatedAt         time.Time
	StartedAt         *time.Time
	CompletedAt       *time.Time
	LastUpdatedAt     time.Time
	WorkerExecutionID string
	Result            string
	Error             string
	GitBaseCommit     string
	GitFinalCommit    string
	ValidationResult  string
}

type WorkerExecutionRecord struct {
	ID            string
	TaskID        string
	Status        string
	PID           int
	ExitCode      *int
	Stdout        string
	Stderr        string
	StartedAt     *time.Time
	CompletedAt   *time.Time
	WorkspacePath string
}

type TaskEventRecord struct {
	ID        string
	TaskID    string
	EventType string
	Detail    string
	CreatedAt time.Time
}

type ValidationResultRecord struct {
	ID              string
	TaskID          string
	Passed          bool
	GitDiffExists   bool
	TestPassed      bool
	UnexpectedFiles string
	Summary         string
	CreatedAt       time.Time
}

type AuditLogRecord struct {
	ID         string
	Level      string
	Action     string
	EntityType string
	EntityID   string
	Detail     string
	CreatedAt  time.Time
}

type DB struct {
	db *sql.DB
}

func Open(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000", dbPath)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	db := &DB{db: conn}
	if err := db.initSchema(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return db, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    worker_type TEXT,
    workspace_id TEXT,
    created_at DATETIME,
    started_at DATETIME,
    completed_at DATETIME,
    last_updated_at DATETIME,
    worker_execution_id TEXT,
    result TEXT,
    error TEXT,
    git_base_commit TEXT,
    git_final_commit TEXT,
    validation_result TEXT
);

CREATE TABLE IF NOT EXISTS worker_executions (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    status TEXT NOT NULL,
    pid INTEGER,
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    workspace_path TEXT
);

CREATE TABLE IF NOT EXISTS task_events (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    detail TEXT,
    created_at DATETIME
);

CREATE TABLE IF NOT EXISTS validation_results (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    passed INTEGER NOT NULL,
    git_diff_exists INTEGER,
    test_passed INTEGER,
    unexpected_files TEXT,
    summary TEXT,
    created_at DATETIME
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    level TEXT NOT NULL,
    action TEXT NOT NULL,
    entity_type TEXT,
    entity_id TEXT,
    detail TEXT,
    created_at DATETIME
);
`
	_, err := d.db.Exec(schema)
	return err
}

// Helper time formatting for SQLite
func formatTime(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

func parseTime(s sql.NullString) *time.Time {
	if !s.Valid || s.String == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s.String); err == nil {
			return &t
		}
	}
	return nil
}

// Task CRUD
func (d *DB) SaveTask(ctx context.Context, task *TaskRecord) error {
	query := `
INSERT INTO tasks (
    id, title, description, status, worker_type, workspace_id,
    created_at, started_at, completed_at, last_updated_at,
    worker_execution_id, result, error, git_base_commit, git_final_commit, validation_result
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    title = excluded.title,
    description = excluded.description,
    status = excluded.status,
    worker_type = excluded.worker_type,
    workspace_id = excluded.workspace_id,
    started_at = excluded.started_at,
    completed_at = excluded.completed_at,
    last_updated_at = excluded.last_updated_at,
    worker_execution_id = excluded.worker_execution_id,
    result = excluded.result,
    error = excluded.error,
    git_base_commit = excluded.git_base_commit,
    git_final_commit = excluded.git_final_commit,
    validation_result = excluded.validation_result;
`
	_, err := d.db.ExecContext(ctx, query,
		task.ID, task.Title, task.Description, task.Status, task.WorkerType, task.WorkspaceID,
		formatTime(&task.CreatedAt), formatTime(task.StartedAt), formatTime(task.CompletedAt), formatTime(&task.LastUpdatedAt),
		task.WorkerExecutionID, task.Result, task.Error, task.GitBaseCommit, task.GitFinalCommit, task.ValidationResult,
	)
	if err != nil {
		return fmt.Errorf("SaveTask failed: %w", err)
	}
	return nil
}

func (d *DB) GetTask(ctx context.Context, id string) (*TaskRecord, error) {
	query := `
SELECT id, title, description, status, worker_type, workspace_id,
       created_at, started_at, completed_at, last_updated_at,
       worker_execution_id, result, error, git_base_commit, git_final_commit, validation_result
FROM tasks WHERE id = ?
`
	row := d.db.QueryRowContext(ctx, query, id)
	return scanTask(row)
}

func (d *DB) ListTasks(ctx context.Context) ([]*TaskRecord, error) {
	query := `
SELECT id, title, description, status, worker_type, workspace_id,
       created_at, started_at, completed_at, last_updated_at,
       worker_execution_id, result, error, git_base_commit, git_final_commit, validation_result
FROM tasks ORDER BY created_at DESC
`
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ListTasks query failed: %w", err)
	}
	defer rows.Close()

	var tasks []*TaskRecord
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("ListTasks scan failed: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (d *DB) ListTasksByStatuses(ctx context.Context, statuses ...string) ([]*TaskRecord, error) {
	if len(statuses) == 0 {
		return nil, nil
	}
	placeholders := ""
	args := make([]interface{}, len(statuses))
	for i, s := range statuses {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args[i] = s
	}

	query := fmt.Sprintf(`
SELECT id, title, description, status, worker_type, workspace_id,
       created_at, started_at, completed_at, last_updated_at,
       worker_execution_id, result, error, git_base_commit, git_final_commit, validation_result
FROM tasks WHERE status IN (%s) ORDER BY created_at ASC
`, placeholders)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListTasksByStatuses query failed: %w", err)
	}
	defer rows.Close()

	var tasks []*TaskRecord
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("ListTasksByStatuses scan failed: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (d *DB) UpdateTaskStatus(ctx context.Context, id string, status string, updatedAt time.Time) error {
	query := `UPDATE tasks SET status = ?, last_updated_at = ? WHERE id = ?`
	res, err := d.db.ExecContext(ctx, query, status, formatTime(&updatedAt), id)
	if err != nil {
		return fmt.Errorf("UpdateTaskStatus failed: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(scanner rowScanner) (*TaskRecord, error) {
	var t TaskRecord
	var (
		desc, workerType, wsID                               sql.NullString
		workerExecID, result, taskErr, baseCommit, finCommit sql.NullString
		valResult                                            sql.NullString
		createdAt, startedAt, completedAt, lastUpdatedAt     sql.NullString
	)

	err := scanner.Scan(
		&t.ID, &t.Title, &desc, &t.Status, &workerType, &wsID,
		&createdAt, &startedAt, &completedAt, &lastUpdatedAt,
		&workerExecID, &result, &taskErr, &baseCommit, &finCommit, &valResult,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	t.Description = desc.String
	t.WorkerType = workerType.String
	t.WorkspaceID = wsID.String
	t.WorkerExecutionID = workerExecID.String
	t.Result = result.String
	t.Error = taskErr.String
	t.GitBaseCommit = baseCommit.String
	t.GitFinalCommit = finCommit.String
	t.ValidationResult = valResult.String

	if parsed := parseTime(createdAt); parsed != nil {
		t.CreatedAt = *parsed
	}
	t.StartedAt = parseTime(startedAt)
	t.CompletedAt = parseTime(completedAt)
	if parsed := parseTime(lastUpdatedAt); parsed != nil {
		t.LastUpdatedAt = *parsed
	}

	return &t, nil
}

// Worker Execution CRUD
func (d *DB) SaveWorkerExecution(ctx context.Context, exec *WorkerExecutionRecord) error {
	query := `
INSERT INTO worker_executions (
    id, task_id, status, pid, exit_code, stdout, stderr, started_at, completed_at, workspace_path
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    status = excluded.status,
    pid = excluded.pid,
    exit_code = excluded.exit_code,
    stdout = excluded.stdout,
    stderr = excluded.stderr,
    completed_at = excluded.completed_at,
    workspace_path = excluded.workspace_path;
`
	var exitCodeVal sql.NullInt64
	if exec.ExitCode != nil {
		exitCodeVal = sql.NullInt64{Int64: int64(*exec.ExitCode), Valid: true}
	}

	_, err := d.db.ExecContext(ctx, query,
		exec.ID, exec.TaskID, exec.Status, exec.PID, exitCodeVal,
		exec.Stdout, exec.Stderr, formatTime(exec.StartedAt), formatTime(exec.CompletedAt), exec.WorkspacePath,
	)
	if err != nil {
		return fmt.Errorf("SaveWorkerExecution failed: %w", err)
	}
	return nil
}

func (d *DB) GetWorkerExecution(ctx context.Context, id string) (*WorkerExecutionRecord, error) {
	query := `
SELECT id, task_id, status, pid, exit_code, stdout, stderr, started_at, completed_at, workspace_path
FROM worker_executions WHERE id = ?
`
	row := d.db.QueryRowContext(ctx, query, id)
	return scanWorkerExecution(row)
}

func (d *DB) GetLatestWorkerExecutionByTask(ctx context.Context, taskID string) (*WorkerExecutionRecord, error) {
	query := `
SELECT id, task_id, status, pid, exit_code, stdout, stderr, started_at, completed_at, workspace_path
FROM worker_executions WHERE task_id = ? ORDER BY started_at DESC LIMIT 1
`
	row := d.db.QueryRowContext(ctx, query, taskID)
	return scanWorkerExecution(row)
}

func (d *DB) UpdateWorkerExecution(ctx context.Context, exec *WorkerExecutionRecord) error {
	return d.SaveWorkerExecution(ctx, exec)
}

func scanWorkerExecution(scanner rowScanner) (*WorkerExecutionRecord, error) {
	var exec WorkerExecutionRecord
	var (
		exitCodeVal             sql.NullInt64
		startedAt, completedAt  sql.NullString
		stdout, stderr, wsPath  sql.NullString
	)

	err := scanner.Scan(
		&exec.ID, &exec.TaskID, &exec.Status, &exec.PID, &exitCodeVal,
		&stdout, &stderr, &startedAt, &completedAt, &wsPath,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if exitCodeVal.Valid {
		val := int(exitCodeVal.Int64)
		exec.ExitCode = &val
	}
	exec.Stdout = stdout.String
	exec.Stderr = stderr.String
	exec.WorkspacePath = wsPath.String
	exec.StartedAt = parseTime(startedAt)
	exec.CompletedAt = parseTime(completedAt)

	return &exec, nil
}

// Task Events CRUD
func (d *DB) SaveTaskEvent(ctx context.Context, event *TaskEventRecord) error {
	query := `INSERT INTO task_events (id, task_id, event_type, detail, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := d.db.ExecContext(ctx, query,
		event.ID, event.TaskID, event.EventType, event.Detail, formatTime(&event.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("SaveTaskEvent failed: %w", err)
	}
	return nil
}

func (d *DB) ListTaskEvents(ctx context.Context, taskID string) ([]*TaskEventRecord, error) {
	query := `SELECT id, task_id, event_type, detail, created_at FROM task_events WHERE task_id = ? ORDER BY created_at ASC`
	rows, err := d.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("ListTaskEvents query failed: %w", err)
	}
	defer rows.Close()

	var events []*TaskEventRecord
	for rows.Next() {
		var e TaskEventRecord
		var detail, createdAt sql.NullString
		if err := rows.Scan(&e.ID, &e.TaskID, &e.EventType, &detail, &createdAt); err != nil {
			return nil, fmt.Errorf("ListTaskEvents scan failed: %w", err)
		}
		e.Detail = detail.String
		if parsed := parseTime(createdAt); parsed != nil {
			e.CreatedAt = *parsed
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

// Validation Results CRUD
func (d *DB) SaveValidationResult(ctx context.Context, vr *ValidationResultRecord) error {
	query := `
INSERT INTO validation_results (id, task_id, passed, git_diff_exists, test_passed, unexpected_files, summary, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
	var passedInt, diffInt, testInt int
	if vr.Passed {
		passedInt = 1
	}
	if vr.GitDiffExists {
		diffInt = 1
	}
	if vr.TestPassed {
		testInt = 1
	}

	_, err := d.db.ExecContext(ctx, query,
		vr.ID, vr.TaskID, passedInt, diffInt, testInt, vr.UnexpectedFiles, vr.Summary, formatTime(&vr.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("SaveValidationResult failed: %w", err)
	}
	return nil
}

func (d *DB) GetValidationResult(ctx context.Context, taskID string) (*ValidationResultRecord, error) {
	query := `
SELECT id, task_id, passed, git_diff_exists, test_passed, unexpected_files, summary, created_at
FROM validation_results WHERE task_id = ? ORDER BY created_at DESC LIMIT 1
`
	row := d.db.QueryRowContext(ctx, query, taskID)
	var vr ValidationResultRecord
	var passedInt, diffInt, testInt int
	var unexpected, summary, createdAt sql.NullString

	err := row.Scan(&vr.ID, &vr.TaskID, &passedInt, &diffInt, &testInt, &unexpected, &summary, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetValidationResult failed: %w", err)
	}

	vr.Passed = passedInt == 1
	vr.GitDiffExists = diffInt == 1
	vr.TestPassed = testInt == 1
	vr.UnexpectedFiles = unexpected.String
	vr.Summary = summary.String
	if parsed := parseTime(createdAt); parsed != nil {
		vr.CreatedAt = *parsed
	}
	return &vr, nil
}

// Audit Logs CRUD
func (d *DB) SaveAuditLog(ctx context.Context, log *AuditLogRecord) error {
	query := `INSERT INTO audit_logs (id, level, action, entity_type, entity_id, detail, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := d.db.ExecContext(ctx, query,
		log.ID, log.Level, log.Action, log.EntityType, log.EntityID, log.Detail, formatTime(&log.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("SaveAuditLog failed: %w", err)
	}
	return nil
}

func (d *DB) ListAuditLogs(ctx context.Context, entityType, entityID string) ([]*AuditLogRecord, error) {
	query := `SELECT id, level, action, entity_type, entity_id, detail, created_at FROM audit_logs WHERE 1=1`
	var args []interface{}
	if entityType != "" {
		query += " AND entity_type = ?"
		args = append(args, entityType)
	}
	if entityID != "" {
		query += " AND entity_id = ?"
		args = append(args, entityID)
	}
	query += " ORDER BY created_at ASC"

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListAuditLogs query failed: %w", err)
	}
	defer rows.Close()

	var logs []*AuditLogRecord
	for rows.Next() {
		var l AuditLogRecord
		var entityT, entityI, detail, createdAt sql.NullString
		if err := rows.Scan(&l.ID, &l.Level, &l.Action, &entityT, &entityI, &detail, &createdAt); err != nil {
			return nil, fmt.Errorf("ListAuditLogs scan failed: %w", err)
		}
		l.EntityType = entityT.String
		l.EntityID = entityI.String
		l.Detail = detail.String
		if parsed := parseTime(createdAt); parsed != nil {
			l.CreatedAt = *parsed
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}
