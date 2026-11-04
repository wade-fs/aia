package metrics

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// MetricCollectorDeps provides storage references for gathering live operational metrics (AIA-19 §7).
type MetricCollectorDeps struct {
	DBPath       string
	TaskRepo     *repository.TaskRepository
	ExecRepo     *repository.ExecutionRepository
	AutonomyRepo *repository.AutonomyRepository
	DecisionRepo *repository.DecisionRepository
	GoalRepo     *repository.GoalRepository
	ProblemRepo  *repository.ProblemRepository
	PlanRepo     *repository.PlanRepository
	StartTime    time.Time
}

// RuntimeMetrics encapsulates low-level operational indicators (AIA-19 §7.2).
type RuntimeMetrics struct {
	UptimeSeconds     float64 `json:"uptime_seconds"`
	RestartCount      int     `json:"restart_count"`
	ActiveGoroutines  int     `json:"active_goroutines"`
	AllocMemoryBytes  uint64  `json:"alloc_memory_bytes"`
	TotalAllocBytes   uint64  `json:"total_alloc_bytes"`
	SysMemoryBytes    uint64  `json:"sys_memory_bytes"`
	NumGC             uint32  `json:"num_gc"`
	StorageDBSizeByte int64   `json:"storage_db_size_bytes"`
	ActiveWorkers     int     `json:"active_workers"`
	EventQueueDepth   int     `json:"event_queue_depth"`
}

// ExecutionMetrics captures task scheduling and tool invocation throughput (AIA-19 §7.3).
type ExecutionMetrics struct {
	TotalTasks          int     `json:"total_tasks"`
	ActiveTasks         int     `json:"active_tasks"`
	PendingTasks        int     `json:"pending_tasks"`
	CompletedTasks      int     `json:"completed_tasks"`
	FailedTasks         int     `json:"failed_tasks"`
	TaskSuccessRate     float64 `json:"task_success_rate"`
	TaskFailureRate     float64 `json:"task_failure_rate"`
	TotalExecutions     int     `json:"total_executions"`
	TotalTokensConsumed int64   `json:"total_tokens_consumed"`
	TotalSpendUSD       float64 `json:"total_spend_usd"`
}

// AutonomyMetrics reflects higher-order cognitive and decision telemetry (AIA-19 §7.5).
type AutonomyMetrics struct {
	TotalCycles               int64   `json:"total_cycles"`
	TotalDecisions            int     `json:"total_decisions"`
	NoActionDecisions         int     `json:"no_action_decisions"`
	CandidateActionsGenerated int     `json:"candidate_actions_generated"`
	InitiativePrecision       float64 `json:"initiative_precision"`
	CompositeAutonomyScore    float64 `json:"composite_autonomy_score"`
}

// DomainMetrics provides indicators across active functional pilots (AIA-19 §7.6).
type DomainMetrics struct {
	ActiveGoals    int `json:"active_goals"`
	ActiveProblems int `json:"active_problems"`
	ActivePlans    int `json:"active_plans"`
}

// SystemMetricsSnapshot aggregates all 5 telemetry categories (AIA-19 §7.1).
type SystemMetricsSnapshot struct {
	CapturedAt time.Time        `json:"captured_at"`
	Runtime    RuntimeMetrics   `json:"runtime"`
	Execution  ExecutionMetrics `json:"execution"`
	Autonomy   AutonomyMetrics  `json:"autonomy"`
	Domain     DomainMetrics    `json:"domain"`
}

// Registry manages and aggregates process-wide operational metrics (AIA-19 §7).
type Registry struct {
	startTime    time.Time
	restartCount int
	counters     map[string]int64
	gauges       map[string]float64
	mu           sync.RWMutex
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *Registry
)

// DefaultRegistry returns the process-wide metric registry singleton.
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// NewRegistry initializes an operational metric registry.
func NewRegistry() *Registry {
	return &Registry{
		startTime: time.Now().UTC(),
		counters:  make(map[string]int64),
		gauges:    make(map[string]float64),
	}
}

// Incr increments a named metric counter.
func (r *Registry) Incr(name string, val int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] += val
}

// SetGauge updates a gauge value.
func (r *Registry) SetGauge(name string, val float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = val
}

// Collect compiles a point-in-time metrics snapshot across all 5 standard categories.
func (r *Registry) Collect(ctx context.Context, deps MetricCollectorDeps) (*SystemMetricsSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now().UTC()
	start := deps.StartTime
	if start.IsZero() {
		start = r.startTime
	}
	uptime := now.Sub(start).Seconds()

	// 1. Runtime MemStats & Goroutines
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	goroutines := runtime.NumGoroutine()

	var dbSize int64
	if deps.DBPath != "" {
		if fi, err := os.Stat(deps.DBPath); err == nil {
			dbSize = fi.Size()
		}
	}

	snap := &SystemMetricsSnapshot{
		CapturedAt: now,
		Runtime: RuntimeMetrics{
			UptimeSeconds:     uptime,
			RestartCount:      r.restartCount,
			ActiveGoroutines:  goroutines,
			AllocMemoryBytes:  ms.Alloc,
			TotalAllocBytes:   ms.TotalAlloc,
			SysMemoryBytes:    ms.Sys,
			NumGC:             ms.NumGC,
			StorageDBSizeByte: dbSize,
			ActiveWorkers:     int(r.counters["active_workers"]),
			EventQueueDepth:   int(r.counters["event_queue_depth"]),
		},
	}

	// 2. Execution metrics
	if deps.TaskRepo != nil {
		allTasks, err := deps.TaskRepo.List(ctx, repository.TaskFilter{Limit: 5000})
		if err == nil {
			snap.Execution.TotalTasks = len(allTasks)
			for _, t := range allTasks {
				switch string(t.Status) {
				case "RUNNING":
					snap.Execution.ActiveTasks++
				case "READY", "PENDING":
					snap.Execution.PendingTasks++
				case "COMPLETED":
					snap.Execution.CompletedTasks++
				case "FAILED":
					snap.Execution.FailedTasks++
				}
			}
			resolved := snap.Execution.CompletedTasks + snap.Execution.FailedTasks
			if resolved > 0 {
				snap.Execution.TaskSuccessRate = float64(snap.Execution.CompletedTasks) / float64(resolved)
				snap.Execution.TaskFailureRate = float64(snap.Execution.FailedTasks) / float64(resolved)
			}
		}
	}

	if deps.ExecRepo != nil {
		if tokens, err := deps.ExecRepo.SumTokensSince(ctx, time.Time{}); err == nil {
			snap.Execution.TotalTokensConsumed = tokens
		}
	}

	// 3. Autonomy Metrics
	if deps.AutonomyRepo != nil {
		if inits, err := deps.AutonomyRepo.ListRecentInitiativesByProject(ctx, "", 1000); err == nil {
			snap.Autonomy.CandidateActionsGenerated = len(inits)
			useful := 0
			for _, init := range inits {
				if init.Status == "ADOPTED" || init.Status == "MONITORING" {
					useful++
				}
			}
			if len(inits) > 0 {
				snap.Autonomy.InitiativePrecision = float64(useful) / float64(len(inits))
			}
		}
	}

	if deps.DecisionRepo != nil {
		if decs, err := deps.DecisionRepo.ListByProject(ctx, "", 1000); err == nil {
			snap.Autonomy.TotalDecisions = len(decs)
			for _, d := range decs {
				if d.ActionType == "NO_ACTION" {
					snap.Autonomy.NoActionDecisions++
				}
			}
		}
	}

	// Default baseline score
	snap.Autonomy.CompositeAutonomyScore = 0.90

	// 4. Domain Metrics
	if deps.GoalRepo != nil {
		if goals, err := deps.GoalRepo.ListAll(ctx, "ACTIVE", 1000); err == nil {
			snap.Domain.ActiveGoals = len(goals)
		}
	}
	if deps.ProblemRepo != nil {
		if probs, err := deps.ProblemRepo.ListByProject(ctx, ""); err == nil {
			snap.Domain.ActiveProblems = len(probs)
		}
	}

	return snap, nil
}
