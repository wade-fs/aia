package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/agent-runtime/internal/audit"
	"github.com/wade/aia/agent-runtime/internal/storage"
)

type Manager struct {
	storage *storage.DB
	audit   *audit.Logger
}

func NewManager(db *storage.DB, logger *audit.Logger) *Manager {
	return &Manager{
		storage: db,
		audit:   logger,
	}
}

func (m *Manager) Create(title, description, workerType string) (*Task, error) {
	now := time.Now()
	taskID := uuid.New().String()
	if workerType == "" {
		workerType = "aider"
	}

	t := &Task{
		ID:            taskID,
		Title:         title,
		Description:   description,
		Status:        TaskStatusCreated,
		WorkerType:    workerType,
		CreatedAt:     now,
		LastUpdatedAt: now,
	}

	record := toRecord(t)
	if err := m.storage.SaveTask(context.Background(), record); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	event := &storage.TaskEventRecord{
		ID:        uuid.New().String(),
		TaskID:    t.ID,
		EventType: "TASK_CREATED",
		Detail:    fmt.Sprintf("Task %q created with worker %s", title, workerType),
		CreatedAt: now,
	}
	_ = m.storage.SaveTaskEvent(context.Background(), event)

	if m.audit != nil {
		_ = m.audit.LogInfo("task_created", "task", t.ID, fmt.Sprintf("Title: %s, Worker: %s", title, workerType))
	}

	return t, nil
}

func (m *Manager) Get(id string) (*Task, error) {
	record, err := m.storage.GetTask(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task %s: %w", id, err)
	}
	return fromRecord(record), nil
}

func (m *Manager) List() ([]*Task, error) {
	records, err := m.storage.ListTasks(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	tasks := make([]*Task, len(records))
	for i, r := range records {
		tasks[i] = fromRecord(r)
	}
	return tasks, nil
}

func (m *Manager) ListByStatuses(statuses ...TaskStatus) ([]*Task, error) {
	strStatuses := make([]string, len(statuses))
	for i, s := range statuses {
		strStatuses[i] = string(s)
	}
	records, err := m.storage.ListTasksByStatuses(context.Background(), strStatuses...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by statuses: %w", err)
	}
	tasks := make([]*Task, len(records))
	for i, r := range records {
		tasks[i] = fromRecord(r)
	}
	return tasks, nil
}

func (m *Manager) UpdateStatus(id string, status TaskStatus, detail string) error {
	now := time.Now()
	ctx := context.Background()

	t, err := m.Get(id)
	if err != nil {
		return err
	}

	t.Status = status
	t.LastUpdatedAt = now
	if status == TaskStatusRunning && t.StartedAt == nil {
		t.StartedAt = &now
	}
	if status.IsTerminal() && t.CompletedAt == nil {
		t.CompletedAt = &now
	}

	record := toRecord(t)
	if err := m.storage.SaveTask(ctx, record); err != nil {
		return fmt.Errorf("failed to update task %s status: %w", id, err)
	}

	event := &storage.TaskEventRecord{
		ID:        uuid.New().String(),
		TaskID:    id,
		EventType: string(status),
		Detail:    detail,
		CreatedAt: now,
	}
	_ = m.storage.SaveTaskEvent(ctx, event)

	if m.audit != nil {
		_ = m.audit.LogInfo("task_status_changed", "task", id, fmt.Sprintf("status=%s detail=%s", status, detail))
	}

	return nil
}

func (m *Manager) Update(t *Task) error {
	t.LastUpdatedAt = time.Now()
	record := toRecord(t)
	if err := m.storage.SaveTask(context.Background(), record); err != nil {
		return fmt.Errorf("failed to update task %s: %w", t.ID, err)
	}
	return nil
}

func toRecord(t *Task) *storage.TaskRecord {
	return &storage.TaskRecord{
		ID:                t.ID,
		Title:             t.Title,
		Description:       t.Description,
		Status:            string(t.Status),
		WorkerType:        t.WorkerType,
		WorkspaceID:       t.WorkspaceID,
		CreatedAt:         t.CreatedAt,
		StartedAt:         t.StartedAt,
		CompletedAt:       t.CompletedAt,
		LastUpdatedAt:     t.LastUpdatedAt,
		WorkerExecutionID: t.WorkerExecutionID,
		Result:            t.Result,
		Error:             t.Error,
		GitBaseCommit:     t.GitBaseCommit,
		GitFinalCommit:    t.GitFinalCommit,
		ValidationResult:  t.ValidationResult,
	}
}

func fromRecord(r *storage.TaskRecord) *Task {
	return &Task{
		ID:                r.ID,
		Title:             r.Title,
		Description:       r.Description,
		Status:            TaskStatus(r.Status),
		WorkerType:        r.WorkerType,
		WorkspaceID:       r.WorkspaceID,
		CreatedAt:         r.CreatedAt,
		StartedAt:         r.StartedAt,
		CompletedAt:       r.CompletedAt,
		LastUpdatedAt:     r.LastUpdatedAt,
		WorkerExecutionID: r.WorkerExecutionID,
		Result:            r.Result,
		Error:             r.Error,
		GitBaseCommit:     r.GitBaseCommit,
		GitFinalCommit:    r.GitFinalCommit,
		ValidationResult:  r.ValidationResult,
	}
}
