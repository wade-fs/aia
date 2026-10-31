package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/domain/task"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestRuntimeBackgroundWorker_LifecycleAndDispatch(t *testing.T) {
	application, _ := createTestApp(t)
	defer application.Close()

	ctx := context.Background()

	// 1. Verify default recurring schedules are seeded on startup
	schedules, err := application.ScheduleSvc.ListSchedules(ctx, false)
	if err != nil {
		t.Fatalf("failed to list schedules: %v", err)
	}
	if len(schedules) < 5 {
		t.Fatalf("expected at least 5 seeded default schedules for the 3 frequencies, got %d", len(schedules))
	}

	// Verify standard schedule names exist
	schedNames := make(map[string]bool)
	for _, s := range schedules {
		schedNames[s.Name] = true
	}
	expectedSchedules := []string{
		"TWSE Daily Market Close Sync",
		"MOPS Announcements Check",
		"FRED Macro Indicators Sync",
		"Prediction Verification and Calibration",
		"Investment Research Gated Evaluation",
	}
	for _, expected := range expectedSchedules {
		if !schedNames[expected] {
			t.Errorf("missing expected default schedule: %s", expected)
		}
	}

	// 2. Test Step 2: Schedule Worker auto-dispatch
	// Make a schedule due immediately
	now := time.Now().UTC()
	targetSched := &schedules[0]
	targetSched.NextRunAt = now.Add(-1 * time.Minute)
	if err := application.ScheduleRepo.Save(ctx, targetSched); err != nil {
		t.Fatalf("failed to update schedule: %v", err)
	}

	// Run Worker Manager dispatch
	dispatched, err := application.WorkerMgr.DispatchDue(ctx)
	if err != nil {
		t.Fatalf("failed to dispatch due schedules: %v", err)
	}
	if dispatched == 0 {
		t.Fatalf("expected at least 1 schedule dispatched, got 0")
	}

	// Verify task is now in READY state
	readyTasks, err := application.TaskSvc.List(ctx, repository.TaskFilter{Status: task.StatusReady})
	if err != nil {
		t.Fatalf("failed to list ready tasks: %v", err)
	}
	if len(readyTasks) == 0 {
		t.Fatalf("expected ready tasks after schedule dispatch, got 0")
	}

	// 3. Test Step 3 & 4: Task Worker executes READY Task and connects real data collection
	executed, err := application.WorkerMgr.ProcessReady(ctx)
	if err != nil {
		t.Fatalf("failed to process ready tasks: %v", err)
	}
	if executed == 0 {
		t.Fatalf("expected at least 1 task executed, got 0")
	}

	// Verify the executed task transitioned to COMPLETED
	targetTask, err := application.TaskSvc.Get(ctx, readyTasks[0].ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if targetTask.Status != task.StatusCompleted {
		t.Errorf("expected task status COMPLETED, got %s", targetTask.Status)
	}

	// Verify worker status metrics
	stats := application.WorkerMgr.Status()
	if stats.TasksDispatched == 0 {
		t.Errorf("expected stats.TasksDispatched > 0")
	}
	if stats.TasksCompleted == 0 {
		t.Errorf("expected stats.TasksCompleted > 0")
	}
}

func TestRuntimeWorker_ThreeFrequencies_DataCollectionAndPrediction(t *testing.T) {
	application, _ := createTestApp(t)
	defer application.Close()

	ctx := context.Background()

	// 1. Ensure profile & instruments exist
	_, err := application.InvSvc.EnsureDefaultProfile(ctx)
	if err != nil {
		t.Fatalf("failed to ensure profile: %v", err)
	}

	// 2. Frequency 1: Low frequency data collection task (TWSE / MOPS)
	colTask, err := application.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title:       "TWSE Daily Market Close Sync",
		Description: "Sync official TWSE/TPEx daily bars and MOPS announcements for watchlist",
		Acceptance:  []string{"twse_sync", "data_collection"},
		Priority:    50,
	})
	if err != nil {
		t.Fatalf("failed to create data collection task: %v", err)
	}

	// Process READY tasks via Worker
	executed, err := application.WorkerMgr.ProcessReady(ctx)
	if err != nil {
		t.Fatalf("failed to process ready task: %v", err)
	}
	if executed == 0 {
		t.Fatalf("expected task executed")
	}

	// Verify task completed
	refreshedTask, _ := application.TaskSvc.Get(ctx, colTask.ID)
	if refreshedTask.Status != task.StatusCompleted {
		t.Errorf("expected task COMPLETED, got %s", refreshedTask.Status)
	}

	// Verify market data and documents were saved
	history, err := application.InvSvc.ListHistoricalData(ctx, "2330", 10)
	if err != nil || len(history) == 0 {
		t.Fatalf("expected market data history saved for 2330")
	}

	// 3. Frequency 3: Low frequency prediction verification
	// Seed a due prediction
	now := time.Now().UTC()
	inst, err := application.InvRepo.GetInstrumentBySymbol(ctx, "", "2330")
	if err != nil {
		t.Fatalf("failed to get instrument: %v", err)
	}

	// First create a valid decision to link the prediction to
	prop, err := application.InvSvc.ProcessAgyDecision(ctx, investment.AgyDecisionInput{
		Symbol:          "2330",
		Action:          "SELL",
		Confidence:      0.80,
		Reasoning:       []string{"Rebalance overweight position"},
		TargetAmount:    0,
		ExpectedHorizon: "5D",
		StrategyVersion: "v1.0",
	}, 1000.0)
	if err != nil {
		t.Fatalf("failed to process agy decision: %v", err)
	}

	pred := &investment.Prediction{
		ID:                uuid.New().String(),
		DecisionID:        prop.DecisionID,
		InstrumentID:      inst.ID,
		PredictionType:    "DIRECTION",
		Target:            "PRICE_CHANGE_PCT",
		ExpectedDirection: "UP",
		Probability:       0.75,
		BasePrice:         1000.0,
		TargetValue:       1050.0,
		EvaluationHorizon: "5D",
		DueAt:             now.Add(-1 * time.Hour), // already due
		StrategyVersion:   "v1.0",
		Status:            investment.PredictionPending,
		CreatedAt:         now.Add(-5 * 24 * time.Hour),
	}
	if err := application.InvSvc.CreatePrediction(ctx, pred); err != nil {
		t.Fatalf("failed to create prediction: %v", err)
	}

	predTask, err := application.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title:       "Prediction Verification and Calibration",
		Description: "Evaluate due forecasts against actual settlement prices",
		Acceptance:  []string{"prediction_eval"},
		Priority:    40,
	})
	if err != nil {
		t.Fatalf("failed to create prediction task: %v", err)
	}

	executed, err = application.WorkerMgr.ProcessReady(ctx)
	if err != nil {
		t.Fatalf("failed to process prediction task: %v", err)
	}
	if executed == 0 {
		t.Fatalf("expected prediction task executed")
	}

	refreshedPredTask, _ := application.TaskSvc.Get(ctx, predTask.ID)
	if refreshedPredTask.Status != task.StatusCompleted {
		t.Errorf("expected prediction task COMPLETED, got %s", refreshedPredTask.Status)
	}

	// Verify prediction outcome was evaluated
	outcomes, err := application.InvRepo.ListPredictionOutcomes(ctx, 10)
	if err != nil || len(outcomes) == 0 {
		t.Fatalf("expected prediction outcome recorded")
	}
}

func TestRuntimeWorker_MaterialityGatedInvestmentResearch(t *testing.T) {
	application, _ := createTestApp(t)
	defer application.Close()

	ctx := context.Background()

	// 1. Ensure profile & instruments exist
	_, err := application.InvSvc.EnsureDefaultProfile(ctx)
	if err != nil {
		t.Fatalf("failed to ensure profile: %v", err)
	}

	// First pass: Initial research (No prior decision exists -> IsMaterial = true)
	researchTask1, err := application.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		AgentID:     "investment-agent",
		Title:       "Investment Research Gated Evaluation (Pass 1)",
		Description: "Initial evaluation of watchlist instruments",
		Acceptance:  []string{"materiality_gated_research"},
		Priority:    50,
	})
	if err != nil {
		t.Fatalf("failed to create research task: %v", err)
	}

	executed, err := application.WorkerMgr.ProcessReady(ctx)
	if err != nil {
		t.Fatalf("failed to execute research task: %v", err)
	}
	if executed != 1 {
		t.Fatalf("expected 1 task executed, got %d", executed)
	}

	t1, _ := application.TaskSvc.Get(ctx, researchTask1.ID)
	if t1.Status != task.StatusCompleted {
		t.Errorf("expected task COMPLETED, got %s", t1.Status)
	}

	// Verify initial decisions were created
	stats1 := application.WorkerMgr.Status()
	if stats1.DecisionsUpdated == 0 {
		t.Errorf("expected initial decisions updated on first pass, got %d", stats1.DecisionsUpdated)
	}

	// Second pass: Daily data synced, BUT price hasn't moved significantly and no new disclosures
	// Under the core principle: "Data update != Recommendation update"
	// Worker MUST preserve existing decision and NOT generate unnecessary new versions (0 AI tokens)
	researchTask2, err := application.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		AgentID:     "investment-agent",
		Title:       "Investment Research Gated Evaluation (Pass 2 - No Material Change)",
		Description: "Routine periodic check",
		Acceptance:  []string{"materiality_gated_research"},
		Priority:    50,
	})
	if err != nil {
		t.Fatalf("failed to create second research task: %v", err)
	}

	executed, err = application.WorkerMgr.ProcessReady(ctx)
	if err != nil {
		t.Fatalf("failed to execute second research task: %v", err)
	}
	if executed != 1 {
		t.Fatalf("expected 1 task executed, got %d", executed)
	}

	t2, _ := application.TaskSvc.Get(ctx, researchTask2.ID)
	if t2.Status != task.StatusCompleted {
		t.Errorf("expected task COMPLETED, got %s", t2.Status)
	}

	// Verify decisions were preserved without re-generating proposals
	stats2 := application.WorkerMgr.Status()
	if stats2.DecisionsPreserved == 0 {
		t.Errorf("expected decisions to be preserved under materiality gate, got 0")
	}

	// 3. Test RunOnce helper
	summary, err := application.WorkerMgr.RunOnce(ctx)
	if err != nil {
		t.Fatalf("failed to RunOnce: %v", err)
	}
	_ = summary
}

func TestActivityMonitor_EndpointAndSnapshot(t *testing.T) {
	application, _ := createTestApp(t)
	defer application.Close()

	ctx := context.Background()
	// Start worker loop
	_ = application.WorkerMgr.Start(ctx)

	// Fetch via Go API
	snap, err := application.WorkerMgr.GetActivityMonitor(ctx)
	if err != nil {
		t.Fatalf("failed to get activity monitor: %v", err)
	}
	if snap.Runtime != "RUNNING" {
		t.Errorf("expected Runtime RUNNING, got %s", snap.Runtime)
	}
	if snap.ScheduleWorker != "RUNNING" {
		t.Errorf("expected ScheduleWorker RUNNING, got %s", snap.ScheduleWorker)
	}
	if snap.TaskWorker != "IDLE" {
		t.Errorf("expected TaskWorker IDLE, got %s", snap.TaskWorker)
	}
	if snap.NextScheduledRun == "none" || snap.NextScheduledRun == "" {
		t.Errorf("expected valid NextScheduledRun from seeded schedules, got %s", snap.NextScheduledRun)
	}
}
