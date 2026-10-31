package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/orchestrator"
	"github.com/wade/aia/internal/provider"
	invService "github.com/wade/aia/internal/service/investment"
	scheduleService "github.com/wade/aia/internal/service/schedule"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite"
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

// ActivityMonitorSnapshot presents the unified activity metrics (AIA Activity Monitor).
type ActivityMonitorSnapshot struct {
	Runtime                string `json:"runtime"`                  // "RUNNING" or "STOPPED"
	ScheduleWorker         string `json:"schedule_worker"`          // "RUNNING" or "STOPPED"
	TaskWorker             string `json:"task_worker"`              // "IDLE" or "BUSY" / "RUNNING"
	LastDataSync           string `json:"last_data_sync"`           // "YYYY-MM-DD HH:MM" or "never"
	LastInvestmentResearch string `json:"last_investment_research"` // "YYYY-MM-DD HH:MM" or "never"
	PendingTasks           int    `json:"pending_tasks"`            // e.g. 0
	NextScheduledRun       string `json:"next_scheduled_run"`       // "YYYY-MM-DD HH:MM" or "none"
}

// Manager coordinates all AIA background workers (Schedule Worker and Task Worker).
type Manager struct {
	cfg            WorkerConfig
	db             *sqlite.DB
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
	db *sqlite.DB,
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
		db:           db,
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

// GetActivityMonitor provides real-time status and operational timestamps for the AIA Activity Monitor.
func (m *Manager) GetActivityMonitor(ctx context.Context) (*ActivityMonitorSnapshot, error) {
	m.mu.RLock()
	isRunning := m.stats.Running
	m.mu.RUnlock()

	snap := &ActivityMonitorSnapshot{
		Runtime:                "STOPPED",
		ScheduleWorker:         "STOPPED",
		TaskWorker:             "STOPPED",
		LastDataSync:           "never",
		LastInvestmentResearch: "never",
		PendingTasks:           0,
		NextScheduledRun:       "none",
	}

	if isRunning {
		snap.Runtime = "RUNNING"
		snap.ScheduleWorker = "RUNNING"
		if m.taskWorker != nil && m.taskWorker.ActiveTasks() > 0 {
			snap.TaskWorker = "BUSY"
		} else {
			snap.TaskWorker = "IDLE"
		}
	} else {
		// Probe if local background daemon is listening on port 7777
		port := "7777"
		if envP := os.Getenv("AIA_PORT"); envP != "" {
			port = envP
		}
		host := "127.0.0.1"
		if envH := os.Getenv("AIA_HOST"); envH != "" {
			host = envH
		}
		client := &http.Client{Timeout: 300 * time.Millisecond}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s:%s/api/v1/system/health", host, port), nil)
		if resp, err := client.Do(req); err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			snap.Runtime = "RUNNING"
			snap.ScheduleWorker = "RUNNING"
			snap.TaskWorker = "IDLE"
		} else if m.db != nil && m.db.Ping(ctx) == nil {
			snap.Runtime = "IDLE"
			snap.ScheduleWorker = "IDLE"
			snap.TaskWorker = "IDLE"
		}
	}

	if m.db == nil {
		return snap, nil
	}

	rawDB := m.db.RawDB()

	// Query Last Data Sync (TWSE market bar or raw documents)
	var lastDataSyncRaw sql.NullString
	_ = rawDB.QueryRowContext(ctx, `SELECT MAX(fetched_at) FROM investment_market_data`).Scan(&lastDataSyncRaw)
	if !lastDataSyncRaw.Valid || lastDataSyncRaw.String == "" {
		_ = rawDB.QueryRowContext(ctx, `SELECT MAX(fetched_at) FROM investment_raw_documents`).Scan(&lastDataSyncRaw)
	}
	if lastDataSyncRaw.Valid && lastDataSyncRaw.String != "" {
		if t, err := time.Parse(time.RFC3339, lastDataSyncRaw.String); err == nil {
			snap.LastDataSync = t.Local().Format("2006-01-02 15:04")
		}
	}

	// Query Last Investment Research
	var lastResearchRaw sql.NullString
	_ = rawDB.QueryRowContext(ctx, `SELECT MAX(created_at) FROM investment_decisions`).Scan(&lastResearchRaw)
	if lastResearchRaw.Valid && lastResearchRaw.String != "" {
		if t, err := time.Parse(time.RFC3339, lastResearchRaw.String); err == nil {
			snap.LastInvestmentResearch = t.Local().Format("2006-01-02 15:04")
		}
	}

	// Query Pending Tasks
	var pendingCount int
	_ = rawDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE status = 'READY'`).Scan(&pendingCount)
	snap.PendingTasks = pendingCount

	// Query Next Scheduled Run
	var nextRunRaw sql.NullString
	_ = rawDB.QueryRowContext(ctx, `
		SELECT next_run_at FROM schedules
		WHERE is_active = 1
		ORDER BY next_run_at ASC LIMIT 1
	`).Scan(&nextRunRaw)
	if nextRunRaw.Valid && nextRunRaw.String != "" {
		if t, err := time.Parse(time.RFC3339, nextRunRaw.String); err == nil {
			snap.NextScheduledRun = t.Local().Format("2006-01-02 15:04")
		}
	}

	return snap, nil
}

