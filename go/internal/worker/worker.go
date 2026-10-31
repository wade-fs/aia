package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/orchestrator"
	"github.com/wade/aia/internal/provider"
	invService "github.com/wade/aia/internal/service/investment"
	scheduleService "github.com/wade/aia/internal/service/schedule"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// WorkerConfig defines operating parameters for runtime background workers.
type WorkerConfig struct {
	ScheduleInterval    time.Duration `json:"schedule_interval"`
	TaskInterval        time.Duration `json:"task_interval"`
	MaxConcurrentTasks  int           `json:"max_concurrent_tasks"`
	AutoDispatchEnabled bool          `json:"auto_dispatch_enabled"`
	AutoExecuteEnabled  bool          `json:"auto_execute_enabled"`
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		ScheduleInterval:    15 * time.Second,
		TaskInterval:        5 * time.Second,
		MaxConcurrentTasks:  3,
		AutoDispatchEnabled: true,
		AutoExecuteEnabled:  true,
	}
}

// WorkerStats tracks live runtime performance and counts.
type WorkerStats struct {
	Running             bool       `json:"running"`
	SchedulesChecked    int64      `json:"schedules_checked"`
	TasksDispatched     int64      `json:"tasks_dispatched"`
	TasksCompleted      int64      `json:"tasks_completed"`
	TasksFailed         int64      `json:"tasks_failed"`
	DecisionsPreserved  int64      `json:"decisions_preserved"`
	DecisionsUpdated    int64      `json:"decisions_updated"`
	LastScheduleCheckAt *time.Time `json:"last_schedule_check_at,omitempty"`
	LastTaskExecutionAt *time.Time `json:"last_task_execution_at,omitempty"`
	LastCollectionAt    *time.Time `json:"last_collection_at,omitempty"`
	LastError           string     `json:"last_error,omitempty"`
}

// WorkerRunSummary reports the result of a single run-once pass.
type WorkerRunSummary struct {
	DispatchedSchedules int `json:"dispatched_schedules"`
	ExecutedTasks       int `json:"executed_tasks"`
}

// Manager coordinates all AIA background workers (Schedule Worker and Task Worker).
type Manager struct {
	cfg            WorkerConfig
	scheduleSvc    *scheduleService.Service
	taskSvc        *taskService.Service
	invSvc         *invService.Service
	collectorSvc   *invService.DataCollectorService
	scheduler      *orchestrator.Scheduler
	prov           provider.Provider
	bus            eventbus.Publisher
	auditRepo      *repository.AuditRepository
	scheduleWorker *ScheduleWorker
	taskWorker     *TaskWorker

	mu     sync.RWMutex
	stats  WorkerStats
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager creates a new background worker runtime manager.
func NewManager(
	cfg WorkerConfig,
	schedSvc *scheduleService.Service,
	taskSvc *taskService.Service,
	invSvc *invService.Service,
	collectorSvc *invService.DataCollectorService,
	scheduler *orchestrator.Scheduler,
	prov provider.Provider,
	bus eventbus.Publisher,
	auditRepo *repository.AuditRepository,
) *Manager {
	if cfg.ScheduleInterval <= 0 {
		cfg.ScheduleInterval = 15 * time.Second
	}
	if cfg.TaskInterval <= 0 {
		cfg.TaskInterval = 5 * time.Second
	}
	if cfg.MaxConcurrentTasks <= 0 {
		cfg.MaxConcurrentTasks = 3
	}

	m := &Manager{
		cfg:          cfg,
		scheduleSvc:  schedSvc,
		taskSvc:      taskSvc,
		invSvc:       invSvc,
		collectorSvc: collectorSvc,
		scheduler:    scheduler,
		prov:         prov,
		bus:          bus,
		auditRepo:    auditRepo,
	}

	m.scheduleWorker = newScheduleWorker(m)
	m.taskWorker = newTaskWorker(m)

	return m
}

// Start launches the background worker goroutines.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.stats.Running {
		m.mu.Unlock()
		return fmt.Errorf("worker manager is already running")
	}

	workerCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.stats.Running = true
	m.mu.Unlock()

	slog.Info("starting runtime background workers",
		"schedule_interval", m.cfg.ScheduleInterval,
		"task_interval", m.cfg.TaskInterval,
		"max_concurrent_tasks", m.cfg.MaxConcurrentTasks)

	// Ensure standard schedules are seeded
	_ = m.SeedDefaultSchedules(workerCtx)

	// 1. Start Schedule Worker
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.scheduleWorker.Run(workerCtx)
	}()

	// 2. Start Task Worker
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.taskWorker.Run(workerCtx)
	}()

	return nil
}

// Stop signals background workers to gracefully cease operations.
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.stats.Running {
		m.mu.Unlock()
		return
	}
	m.stats.Running = false
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()

	m.wg.Wait()
	slog.Info("runtime background workers stopped successfully")
}

// Status returns a point-in-time copy of worker statistics.
func (m *Manager) Status() WorkerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// RunOnce executes a single manual pass of schedule dispatching followed by task execution.
func (m *Manager) RunOnce(ctx context.Context) (WorkerRunSummary, error) {
	summary := WorkerRunSummary{}

	// 1. Dispatch due schedules
	dispatched, err := m.DispatchDue(ctx)
	if err != nil {
		return summary, fmt.Errorf("failed to dispatch due schedules: %w", err)
	}
	summary.DispatchedSchedules = dispatched

	// 2. Execute ready tasks
	executed, err := m.ProcessReady(ctx)
	if err != nil {
		return summary, fmt.Errorf("failed to process ready tasks: %w", err)
	}
	summary.ExecutedTasks = executed

	return summary, nil
}

// DispatchDue immediately checks and dispatches due schedules.
func (m *Manager) DispatchDue(ctx context.Context) (int, error) {
	if m.scheduleSvc == nil {
		return 0, nil
	}
	count, err := m.scheduleSvc.ProcessDueSchedules(ctx)
	now := time.Now().UTC()

	m.mu.Lock()
	m.stats.SchedulesChecked++
	m.stats.LastScheduleCheckAt = &now
	m.stats.TasksDispatched += int64(count)
	if err != nil {
		m.stats.LastError = err.Error()
	}
	m.mu.Unlock()

	return count, err
}

// ProcessReady immediately runs all currently READY tasks.
func (m *Manager) ProcessReady(ctx context.Context) (int, error) {
	return m.taskWorker.processPendingTasks(ctx)
}
