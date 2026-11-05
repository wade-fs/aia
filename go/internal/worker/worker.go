package worker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wade/aia/internal/autonomy"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/governance"
	domainProject "github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/orchestrator"
	"github.com/wade/aia/internal/provider"
	govService "github.com/wade/aia/internal/service/governance"
	invService "github.com/wade/aia/internal/service/investment"
	scheduleService "github.com/wade/aia/internal/service/schedule"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var (
	ErrEmergencyStopActive = errors.New("dispatch frozen: system is under EMERGENCY_STOP")
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
	EmergencyStopped    bool       `json:"emergency_stopped"`
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
	Runtime                string `json:"runtime"`                  // "RUNNING", "STOPPED", or "EMERGENCY_STOP"
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
	govRepo        *repository.GovernanceRepository
	govSvc         *govService.Service
	scheduleWorker *ScheduleWorker
	taskWorker     *TaskWorker

	mu     sync.RWMutex
	stats  WorkerStats
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// emergencyStop atomic flag for 1-second freeze on new task dispatch
	emergencyStopped atomic.Bool

	// activeTaskCancels tracks cancel funcs for currently running tasks
	activeTaskCancelsMu sync.Mutex
	activeTaskCancels   map[string]context.CancelFunc

	retainDays int
	execRepo   *repository.ExecutionRepository

	autonomyEng      *autonomy.Engine
	autonomyInterval time.Duration
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

	govRepo := repository.NewGovernanceRepository(db)

	m := &Manager{
		cfg:               cfg,
		db:                db,
		scheduleSvc:       schedSvc,
		taskSvc:           taskSvc,
		invSvc:            invSvc,
		collectorSvc:      collectorSvc,
		scheduler:         scheduler,
		prov:              prov,
		bus:               bus,
		auditRepo:         auditRepo,
		govRepo:           govRepo,
		activeTaskCancels: make(map[string]context.CancelFunc),
	}

	m.scheduleWorker = newScheduleWorker(m)
	m.taskWorker = newTaskWorker(m)

	return m
}

// SetGovernanceService connects the governance service.
func (m *Manager) SetGovernanceService(govSvc *govService.Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.govSvc = govSvc
}

// SetLogRetention configures periodic log retention purging (AIA-19 logging.retain_days).
func (m *Manager) SetLogRetention(retainDays int, execRepo *repository.ExecutionRepository) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retainDays = retainDays
	m.execRepo = execRepo
}

// SetAutonomyEngine attaches the autonomy closed-loop engine and configures auto-wake interval (AIA-08 §12.2).
// Passing interval <= 0 activates the standard Taiwan Time schedule (08:30, 13:00, 17:30, 21:00).
func (m *Manager) SetAutonomyEngine(eng *autonomy.Engine, interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autonomyEng = eng
	if interval < 0 {
		interval = 0
	}
	m.autonomyInterval = interval
}

// TriggerAutonomousCycle triggers an autonomous cycle immediately from external events (AIA-08 §13).
func (m *Manager) TriggerAutonomousCycle(ctx context.Context, projectID string, triggerType continuity.CycleTriggerType, summary string) (*domainAutonomy.CycleOutcome, error) {
	if m.IsEmergencyStopped() {
		return nil, ErrEmergencyStopActive
	}
	if m.autonomyEng == nil {
		return nil, fmt.Errorf("autonomy engine is not attached to worker manager")
	}
	if projectID == "" {
		projectID = "default"
	}
	if summary == "" {
		summary = fmt.Sprintf("Event-driven autonomous cycle triggered by %s", triggerType)
	}
	return m.autonomyEng.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   projectID,
		TriggerType: triggerType,
		Summary:     summary,
	})
}

// IsEmergencyStopped checks whether dispatch is frozen.
func (m *Manager) IsEmergencyStopped() bool {
	if m.emergencyStopped.Load() {
		return true
	}
	// Check database persisted state
	if m.govRepo != nil {
		ctrl, err := m.govRepo.GetControl(context.Background())
		if err == nil && ctrl.IsEmergencyStopped() {
			m.emergencyStopped.Store(true)
			return true
		}
	}
	return false
}

// TriggerEmergencyStop engages the Controlled Emergency Stop (AIA-17 §11.5).
// 1. Immediately sets atomic flag (within milliseconds, well under 1-second requirement)
// 2. Halts new task dispatch
// 3. Cancels all actively running tasks and marks them CANCELLED
// 4. Persists the state to SQLite runtime_controls table
func (m *Manager) TriggerEmergencyStop(ctx context.Context, actor governance.Actor, reason string) error {
	m.emergencyStopped.Store(true)
	now := time.Now().UTC()

	m.mu.Lock()
	m.stats.EmergencyStopped = true
	m.mu.Unlock()

	slog.Warn("EMERGENCY STOP TRIGGERED: freezing task dispatch immediately", "reason", reason, "actor", actor)

	// Persist to DB
	if m.govRepo != nil {
		actorStr := fmt.Sprintf("%s:%s", actor.Type, actor.ID)
		_ = m.govRepo.SetEmergencyStop(ctx, actorStr, reason)
	}

	// Cancel all actively executing tasks
	m.activeTaskCancelsMu.Lock()
	runningCancels := make(map[string]context.CancelFunc, len(m.activeTaskCancels))
	for tid, cancel := range m.activeTaskCancels {
		runningCancels[tid] = cancel
	}
	m.activeTaskCancelsMu.Unlock()

	for tid, cancel := range runningCancels {
		if cancel != nil {
			cancel()
		}
		// Persist task status as CANCELLED to avoid re-dispatch on restart
		if m.taskSvc != nil {
			_ = m.taskSvc.UpdateStatus(ctx, tid, task.StatusCancelled)
		}
	}

	// Broadcast via eventbus
	if m.bus != nil {
		m.bus.Publish("runtime.emergency_stop", map[string]any{
			"actor":     actor,
			"reason":    reason,
			"timestamp": now,
		})
	}

	return nil
}

// ResumeDispatch restores task dispatching from an Emergency Stop.
// AIA-17 Rule: Only a human operator can resume the system.
func (m *Manager) ResumeDispatch(ctx context.Context, actor governance.Actor, force bool) error {
	if !actor.IsHuman() {
		return fmt.Errorf("unauthorized actor: only a human owner can resume runtime from emergency stop")
	}

	if m.govRepo != nil {
		actorStr := fmt.Sprintf("%s:%s", actor.Type, actor.ID)
		if err := m.govRepo.ResumeDispatch(ctx, actorStr); err != nil {
			return err
		}
	}

	m.emergencyStopped.Store(false)

	m.mu.Lock()
	m.stats.EmergencyStopped = false
	m.mu.Unlock()

	slog.Info("runtime resumed from emergency stop", "actor", actor)

	if m.bus != nil {
		m.bus.Publish("runtime.resumed", map[string]any{
			"actor":     actor,
			"timestamp": time.Now().UTC(),
		})
	}

	return nil
}

func (m *Manager) registerRunningTask(taskID string, cancel context.CancelFunc) {
	m.activeTaskCancelsMu.Lock()
	defer m.activeTaskCancelsMu.Unlock()
	m.activeTaskCancels[taskID] = cancel
}

func (m *Manager) unregisterRunningTask(taskID string) {
	m.activeTaskCancelsMu.Lock()
	defer m.activeTaskCancelsMu.Unlock()
	delete(m.activeTaskCancels, taskID)
}

// Start launches the background worker goroutines.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.stats.Running {
		m.mu.Unlock()
		return fmt.Errorf("worker manager is already running")
	}

	// Sync emergency stop status from DB
	if m.govRepo != nil {
		ctrl, err := m.govRepo.GetControl(ctx)
		if err == nil && ctrl.IsEmergencyStopped() {
			m.emergencyStopped.Store(true)
			m.stats.EmergencyStopped = true
			slog.Warn("starting worker manager in EMERGENCY_STOP state from persisted controls")
		}
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

	// 3. Start Periodic Log Retention Purger (AIA-19 logging.retain_days)
	if m.retainDays > 0 && (m.auditRepo != nil || m.execRepo != nil) {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.runLogRetentionPurger(workerCtx)
		}()
	}

	// 4. Start Autonomy Periodic Worker (AIA-08 §12.2)
	if m.autonomyEng != nil {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.runAutonomyWorker(workerCtx)
		}()
	}

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
	st := m.stats
	st.EmergencyStopped = m.IsEmergencyStopped()
	return st
}

// RunOnce executes a single manual pass of schedule dispatching followed by task execution.
func (m *Manager) RunOnce(ctx context.Context) (WorkerRunSummary, error) {
	summary := WorkerRunSummary{}

	if m.IsEmergencyStopped() {
		return summary, ErrEmergencyStopActive
	}

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
	if m.IsEmergencyStopped() {
		return 0, ErrEmergencyStopActive
	}

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
	if m.IsEmergencyStopped() {
		return 0, ErrEmergencyStopActive
	}
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

	if m.IsEmergencyStopped() {
		snap.Runtime = "EMERGENCY_STOP"
		snap.ScheduleWorker = "STOPPED"
		snap.TaskWorker = "STOPPED"
	} else if isRunning {
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

func (m *Manager) runLogRetentionPurger(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	// Initial purge on startup
	m.purgeExpiredLogs(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.purgeExpiredLogs(ctx)
		}
	}
}

func (m *Manager) purgeExpiredLogs(ctx context.Context) {
	m.mu.RLock()
	days := m.retainDays
	auditRepo := m.auditRepo
	execRepo := m.execRepo
	m.mu.RUnlock()

	if days <= 0 {
		return
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	if auditRepo != nil {
		if purged, err := auditRepo.PurgeOlderThan(ctx, cutoff); err == nil && purged > 0 {
			slog.Info("periodic log retention purger removed expired audit logs", "count", purged, "retain_days", days)
		}
	}
	if execRepo != nil {
		if purged, err := execRepo.PurgeEventsOlderThan(ctx, cutoff); err == nil && purged > 0 {
			slog.Info("periodic log retention purger removed expired execution events", "count", purged, "retain_days", days)
		}
	}
}

// AutoWakeSlot represents a specific hour and minute in a day.
type AutoWakeSlot struct {
	Hour   int
	Minute int
}

// DefaultTaiwanAutoWakeSlots defines the 4 daily auto-wake target times in Taiwan Time (UTC+8): 08:30, 13:00, 17:30, 21:00.
var DefaultTaiwanAutoWakeSlots = []AutoWakeSlot{
	{8, 30},
	{13, 0},
	{17, 30},
	{21, 0},
}

// NextAutoWakeTime calculates the next scheduled wake-up time and wait duration relative to now in the specified location.
func NextAutoWakeTime(now time.Time, loc *time.Location) (time.Time, time.Duration) {
	if loc == nil {
		taiwanLoc, err := time.LoadLocation("Asia/Taipei")
		if err != nil {
			taiwanLoc = time.FixedZone("CST", 8*3600)
		}
		loc = taiwanLoc
	}
	nowInLoc := now.In(loc)
	year, month, day := nowInLoc.Date()

	// Check today's remaining slots
	for _, slot := range DefaultTaiwanAutoWakeSlots {
		cand := time.Date(year, month, day, slot.Hour, slot.Minute, 0, 0, loc)
		if cand.After(nowInLoc) {
			return cand, cand.Sub(nowInLoc)
		}
	}

	// First slot tomorrow
	firstSlot := DefaultTaiwanAutoWakeSlots[0]
	tomorrowCand := time.Date(year, month, day+1, firstSlot.Hour, firstSlot.Minute, 0, 0, loc)
	return tomorrowCand, tomorrowCand.Sub(nowInLoc)
}

func (m *Manager) runAutonomyWorker(ctx context.Context) {
	m.mu.RLock()
	interval := m.autonomyInterval
	eng := m.autonomyEng
	m.mu.RUnlock()

	if eng == nil {
		return
	}

	// If a custom interval was configured (e.g. in test suites > 0), use ticker mode
	if interval > 0 {
		slog.Info("autonomy worker periodic loop started", "interval", interval.String())
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.executeAutonomyWorkerCycle(ctx, eng)
			}
		}
	}

	// Standard Production Schedule Mode: Taiwan Time 08:30, 13:00, 17:30, 21:00
	taiwanLoc, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		taiwanLoc = time.FixedZone("CST", 8*3600)
	}

	slog.Info("autonomy worker scheduled loop started (Taiwan Time: 08:30, 13:00, 17:30, 21:00)")
	for {
		nextWake, waitDur := NextAutoWakeTime(time.Now(), taiwanLoc)
		slog.Info("autonomy worker scheduled next auto-wake",
			"next_wake_time", nextWake.Format("2006-01-02 15:04:05 MST"),
			"wait_duration", waitDur.Round(time.Second).String())

		timer := time.NewTimer(waitDur)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			m.executeAutonomyWorkerCycle(ctx, eng)
		}
	}
}

func (m *Manager) executeAutonomyWorkerCycle(ctx context.Context, eng *autonomy.Engine) {
	if m.IsEmergencyStopped() {
		slog.Debug("autonomy worker cycle skipped: emergency stop active")
		return
	}
	slog.Info("autonomy worker auto-waking to execute scheduled closed-loop cycle (AIA-08 §12.2)")

	// Enumerate active projects so all active projects are cycled without hardcoding "default"
	var activeProjectIDs []string
	if m.db != nil {
		projRepo := repository.NewProjectRepository(m.db)
		if projs, err := projRepo.List(ctx); err == nil {
			for _, p := range projs {
				if p.Status == domainProject.StatusActive {
					activeProjectIDs = append(activeProjectIDs, string(p.ID))
				}
			}
		}
	}

	// If no projects found, fallback to "default"
	if len(activeProjectIDs) == 0 {
		activeProjectIDs = []string{"default"}
	}

	for _, pid := range activeProjectIDs {
		if ctx.Err() != nil {
			return
		}
		if m.IsEmergencyStopped() {
			slog.Debug("autonomy worker remaining project cycles skipped: emergency stop active")
			return
		}
		_, err := eng.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   pid,
			TriggerType: continuity.TriggerScheduled,
			Summary:     fmt.Sprintf("Scheduled autonomous closed loop executed by background autonomy worker for project %s", pid),
		})
		if err != nil {
			slog.Warn("scheduled autonomy cycle encountered non-fatal notice", "project_id", pid, "err", err)
		}
	}
}
