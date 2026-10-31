package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type CreateTaskRequest struct {
	ProjectID    domain.ID
	ParentTaskID domain.ID
	AgentID      domain.ID
	Title        string
	Description  string
	Priority     int
	Risk         domain.RiskLevel
	Acceptance   []string
	Dependencies []domain.ID
	MaxRetries   int
}

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type Service struct {
	repo          *repository.TaskRepository
	auditRepo     *repository.AuditRepository
	healthChecker HealthChecker
}

func NewService(repo *repository.TaskRepository, auditRepo *repository.AuditRepository) *Service {
	return &Service{repo: repo, auditRepo: auditRepo}
}

func (s *Service) SetHealthChecker(hc HealthChecker) {
	s.healthChecker = hc
}

func (s *Service) Create(ctx context.Context, req CreateTaskRequest) (*task.Task, error) {
	if req.ProjectID == "" {
		req.ProjectID = "default"
	}
	if req.Title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}
	if req.Risk == "" {
		req.Risk = domain.RiskLow
	}
	if req.Priority <= 0 {
		req.Priority = 50
	}
	if req.MaxRetries <= 0 {
		req.MaxRetries = 2
	}

	initialStatus := task.StatusReady
	if len(req.Dependencies) > 0 {
		initialStatus = task.StatusWaitingDependency
	}

	t := &task.Task{
		ID:           uuid.New().String(),
		ProjectID:    req.ProjectID,
		ParentTaskID: req.ParentTaskID,
		AgentID:      req.AgentID,
		Title:        req.Title,
		Description:  req.Description,
		Status:       initialStatus,
		Priority:     req.Priority,
		Risk:         req.Risk,
		Acceptance:   req.Acceptance,
		MaxRetries:   req.MaxRetries,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, t, req.Dependencies); err != nil {
		return nil, err
	}

	_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
		ID:           uuid.New().String(),
		ActorType:    "SYSTEM",
		ActorID:      "task-service",
		Action:       "CREATE_TASK",
		ResourceType: "TASK",
		ResourceID:   t.ID,
		Outcome:      "SUCCESS",
	})

	return t, nil
}

func (s *Service) Get(ctx context.Context, id domain.ID) (*task.Task, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, filter repository.TaskFilter) ([]task.Task, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Update(ctx context.Context, t *task.Task) error {
	return s.repo.Update(ctx, t)
}

func (s *Service) UpdateStatus(ctx context.Context, id domain.ID, to task.TaskStatus) error {
	if err := s.repo.UpdateStatus(ctx, id, to); err != nil {
		return err
	}

	// If task completed, check if any dependent tasks can now be moved to READY
	if to == task.StatusCompleted {
		_ = s.promoteDependents(ctx, id)
	}

	_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
		ID:           uuid.New().String(),
		ActorType:    "SYSTEM",
		ActorID:      "task-service",
		Action:       "UPDATE_STATUS",
		ResourceType: "TASK",
		ResourceID:   id,
		Outcome:      string(to),
	})

	return nil
}

func (s *Service) Cancel(ctx context.Context, id domain.ID) error {
	return s.UpdateStatus(ctx, id, task.StatusCancelled)
}

func (s *Service) Resume(ctx context.Context, id domain.ID) error {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if t.Status == task.StatusBlocked || t.Status == task.StatusFailed || t.Status == task.StatusWaitingAuth || t.Status == task.StatusWaitingApproval {
		if s.healthChecker != nil {
			if err := s.healthChecker.HealthCheck(ctx); err != nil {
				return fmt.Errorf("health check failed before resume: %w", err)
			}
		}
		return s.UpdateStatus(ctx, id, task.StatusReady)
	}
	return fmt.Errorf("task status %s cannot be resumed", t.Status)
}

func (s *Service) promoteDependents(ctx context.Context, completedTaskID domain.ID) error {
	waitingTasks, err := s.repo.List(ctx, repository.TaskFilter{Status: task.StatusWaitingDependency})
	if err != nil {
		return err
	}

	for _, wt := range waitingTasks {
		completed, err := s.repo.AreDependenciesCompleted(ctx, wt.ID)
		if err == nil && completed {
			_ = s.repo.UpdateStatus(ctx, wt.ID, task.StatusReady)
		}
	}
	return nil
}

func (s *Service) GetDependencies(ctx context.Context, id domain.ID) ([]domain.ID, error) {
	return s.repo.GetDependencies(ctx, id)
}
