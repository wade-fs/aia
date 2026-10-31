package unit

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/domain/schedule"
	taskdomain "github.com/wade/aia/internal/domain/task"
	scheduleService "github.com/wade/aia/internal/service/schedule"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/worker"
)

func TestActivityMonitor_FieldsAndFormatting(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	taskRepo := repository.NewTaskRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	schedRepo := repository.NewScheduleRepository(db)
	invRepo := repository.NewInvestmentRepository(db)

	taskSvc := taskService.NewService(taskRepo, auditRepo)
	schedSvc := scheduleService.NewService(schedRepo, taskSvc, nil)

	// Ensure instruments & profiles
	_ = invRepo.EnsureDefaultDataSources(ctx)
	prof, _ := invRepo.EnsureDefaultProfile(ctx)
	_, _ = db.RawDB().ExecContext(ctx, "INSERT OR IGNORE INTO projects (id, name, workspace, status, created_at, updated_at) VALUES ('default', 'Default', '.', 'ACTIVE', datetime('now'), datetime('now'))")
	_, _ = db.RawDB().ExecContext(ctx, "INSERT OR IGNORE INTO agents (id, name, domain, instructions, capabilities_json, provider_name, enabled, version, created_at, updated_at) VALUES ('system-collector', 'Collector', 'system', 'collect', '[]', 'none', 1, 1, datetime('now'), datetime('now'))")

	mgr := worker.NewManager(
		worker.DefaultWorkerConfig(),
		db,
		schedSvc,
		taskSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		auditRepo,
	)

	// 1. Initial snapshot check
	snap, err := mgr.GetActivityMonitor(ctx)
	if err != nil {
		t.Fatalf("failed to get activity monitor: %v", err)
	}
	if snap == nil {
		t.Fatal("activity monitor snapshot is nil")
	}

	// 2. Insert sample market data and decisions with known timestamps
	now := time.Now().UTC()
	inst, _ := invRepo.GetInstrumentBySymbol(ctx, "", "2330")

	// Save market data
	_ = invRepo.SaveMarketData(ctx, &investment.MarketData{
		ID:           uuid.New().String(),
		InstrumentID: inst.ID,
		DataType:     "DAILY_BAR",
		Period:       "1D",
		ValueJSON:    `{"close": 1050.0}`,
		SourceID:     "ds-twse-openapi",
		FetchedAt:    now,
		EffectiveAt:  now,
	})

	// Save decision
	_ = invRepo.SaveDecision(ctx, &investment.Decision{
		ID:                         uuid.New().String(),
		PortfolioProfileID:         prof.ID,
		InstrumentID:               inst.ID,
		PortfolioSnapshotID:        "snap-1",
		DecisionType:               investment.DecisionBuy,
		Confidence:                 0.85,
		ReasoningJSON:              `["Growth"]`,
		RisksJSON:                  `["Macro"]`,
		InvalidationConditionsJSON: `[]`,
		ExpectedHorizon:            "20D",
		StrategyVersion:            "v1.0",
		Source:                     "ANTIGRAVITY",
		CreatedAt:                  now,
	})

	// Save a schedule with next_run_at tomorrow
	nextRun := now.Add(24 * time.Hour)
	_ = schedRepo.Save(ctx, &schedule.Schedule{
		ID:               uuid.New().String(),
		Name:             "Sample Scheduled Sync",
		CronExpr:         "0 14 * * 1-5",
		AgentID:          "system-collector",
		TaskTemplateJSON: `{}`,
		MisfirePolicy:    schedule.MisfireSkip,
		IsActive:         true,
		NextRunAt:        nextRun,
		CreatedAt:        now,
		UpdatedAt:        now,
	})

	// Save a ready task
	_ = taskRepo.Create(ctx, &taskdomain.Task{
		ID:          uuid.New().String(),
		ProjectID:   "default",
		AgentID:     "system-collector",
		Title:       "Test Pending Ready Task",
		Description: "Pending execution",
		Status:      taskdomain.StatusReady,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil)

	// Fetch updated Activity Monitor snapshot
	updatedSnap, err := mgr.GetActivityMonitor(ctx)
	if err != nil {
		t.Fatalf("failed to get updated activity monitor: %v", err)
	}

	expectedTimeStr := now.Local().Format("2006-01-02 15:04")
	if updatedSnap.LastDataSync != expectedTimeStr {
		t.Errorf("expected LastDataSync %s, got %s", expectedTimeStr, updatedSnap.LastDataSync)
	}
	if updatedSnap.LastInvestmentResearch != expectedTimeStr {
		t.Errorf("expected LastInvestmentResearch %s, got %s", expectedTimeStr, updatedSnap.LastInvestmentResearch)
	}
	if updatedSnap.PendingTasks != 1 {
		t.Errorf("expected PendingTasks 1, got %d", updatedSnap.PendingTasks)
	}
	expectedNextRun := nextRun.Local().Format("2006-01-02 15:04")
	if updatedSnap.NextScheduledRun != expectedNextRun {
		t.Errorf("expected NextScheduledRun %s, got %s", expectedNextRun, updatedSnap.NextScheduledRun)
	}
}
