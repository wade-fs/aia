package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/schedule"
	taskdomain "github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type TaskCreator interface {
	Create(ctx context.Context, req task.CreateTaskRequest) (*taskdomain.Task, error)
}

type Service struct {
	repo        *repository.ScheduleRepository
	taskCreator TaskCreator
	bus         eventbus.Publisher
}

func NewService(repo *repository.ScheduleRepository, taskCreator TaskCreator, bus eventbus.Publisher) *Service {
	return &Service{
		repo:        repo,
		taskCreator: taskCreator,
		bus:         bus,
	}
}

// CreateSchedule sets up a new recurring automated task schedule.
func (s *Service) CreateSchedule(ctx context.Context, name, cronExpr string, agentID domain.ID, templateJSON string, misfirePolicy schedule.MisfirePolicy) (*schedule.Schedule, error) {
	if name == "" {
		return nil, fmt.Errorf("schedule name cannot be empty")
	}
	now := time.Now().UTC()
	nextRun, err := schedule.ComputeNextRun(cronExpr, now)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	if misfirePolicy == "" {
		misfirePolicy = schedule.MisfireSkip
	}

	sch := &schedule.Schedule{
		ID:               uuid.New().String(),
		Name:             name,
		CronExpr:         cronExpr,
		AgentID:          agentID,
		TaskTemplateJSON: templateJSON,
		MisfirePolicy:    misfirePolicy,
		IsActive:         true,
		NextRunAt:        nextRun,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.Save(ctx, sch); err != nil {
		return nil, err
	}

	return sch, nil
}

func (s *Service) GetSchedule(ctx context.Context, id domain.ID) (*schedule.Schedule, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) ListSchedules(ctx context.Context, activeOnly bool) ([]schedule.Schedule, error) {
	return s.repo.List(ctx, activeOnly)
}

func (s *Service) ToggleActive(ctx context.Context, id domain.ID, active bool) error {
	sch, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	sch.IsActive = active
	sch.UpdatedAt = time.Now().UTC()
	return s.repo.Save(ctx, sch)
}

func (s *Service) DeleteSchedule(ctx context.Context, id domain.ID) error {
	return s.repo.Delete(ctx, id)
}

// TriggerSchedule immediately executes the scheduled task template and updates next run time.
func (s *Service) TriggerSchedule(ctx context.Context, id domain.ID) (*taskdomain.Task, error) {
	sch, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	var req task.CreateTaskRequest
	if sch.TaskTemplateJSON != "" {
		_ = json.Unmarshal([]byte(sch.TaskTemplateJSON), &req)
	}
	if req.Title == "" {
		req.Title = fmt.Sprintf("Scheduled: %s", sch.Name)
	}
	if req.AgentID == "" {
		req.AgentID = sch.AgentID
	}

	createdTask, err := s.taskCreator.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to dispatch scheduled task: %w", err)
	}

	now := time.Now().UTC()
	sch.LastRunAt = &now
	next, err := schedule.ComputeNextRun(sch.CronExpr, now)
	if err == nil {
		sch.NextRunAt = next
	}
	sch.UpdatedAt = now
	_ = s.repo.Save(ctx, sch)

	if s.bus != nil {
		s.bus.Publish(eventbus.TaskCreated, map[string]any{
			"schedule_id": sch.ID,
			"task_id":     createdTask.ID,
		})
	}

	return createdTask, nil
}

// ProcessDueSchedules checks for schedules whose next_run_at has arrived and fires them.
func (s *Service) ProcessDueSchedules(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	due, err := s.repo.ListDue(ctx, now)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, sch := range due {
		// Check misfire condition
		timeLag := now.Sub(sch.NextRunAt)
		if timeLag > 10*time.Minute && sch.MisfirePolicy == schedule.MisfireSkip {
			// Skip and simply advance to next interval
			next, err := schedule.ComputeNextRun(sch.CronExpr, now)
			if err == nil {
				sch.NextRunAt = next
			}
			sch.UpdatedAt = now
			_ = s.repo.Save(ctx, &sch)
			continue
		}

		if _, err := s.TriggerSchedule(ctx, sch.ID); err == nil {
			count++
		}
	}
	return count, nil
}
