package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/domain/schedule"
)

func TestInvestmentDataPipelineAndCalibration(t *testing.T) {
	application, _ := createTestApp(t)
	defer application.Close()

	ctx := context.Background()

	// 1. Ensure profile & instruments exist
	prof, err := application.InvSvc.EnsureDefaultProfile(ctx)
	if err != nil {
		t.Fatalf("failed to ensure profile: %v", err)
	}
	if prof == nil {
		t.Fatal("profile is nil")
	}

	// 2. Test Market Data Ingestion (P4)
	md, err := application.InvSvc.SyncMarketData(ctx, "2330")
	if err != nil {
		t.Fatalf("failed to sync market data: %v", err)
	}
	if md.DataType != "DAILY_BAR" {
		t.Errorf("expected DAILY_BAR, got %s", md.DataType)
	}

	// Fetch historical
	history, err := application.InvSvc.ListHistoricalData(ctx, "2330", 10)
	if err != nil {
		t.Fatalf("failed to list history: %v", err)
	}
	if len(history) == 0 {
		t.Fatal("expected at least 1 history bar")
	}

	// 3. Test Raw Document Ingestion (MOPS / News) (P4)
	docs, err := application.InvSvc.SyncAnnouncements(ctx, "2330")
	if err != nil {
		t.Fatalf("failed to sync announcements: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("expected ingested announcements")
	}

	// 4. Test Prediction Creation & Calibration (P5)
	now := time.Now().UTC()
	inst, err := application.InvRepo.GetInstrumentBySymbol(ctx, "", "2330")
	if err != nil {
		t.Fatalf("failed to get instrument: %v", err)
	}

	// First create a valid Decision via ProcessAgyDecision (SELL to rebalance overweight position)
	prop, err := application.InvSvc.ProcessAgyDecision(ctx, investment.AgyDecisionInput{
		Symbol:          "2330",
		Action:          "SELL",
		Confidence:      0.85,
		Reasoning:       []string{"Rebalance overweight position", "Take profit on current holdings"},
		TargetAmount:    0, // sell all current position
		ExpectedHorizon: "5D",
		StrategyVersion: "v1.0-alpha",
	}, 1050.0)
	if err != nil {
		t.Fatalf("failed to process agy decision: %v", err)
	}
	if prop == nil {
		t.Fatal("expected trade proposal created")
	}

	// Create a prediction linked to this decision that is due immediately
	pred := &investment.Prediction{
		ID:                uuid.New().String(),
		DecisionID:        prop.DecisionID,
		InstrumentID:      inst.ID,
		PredictionType:    "DIRECTION",
		Target:            "PRICE_CHANGE_PCT",
		ExpectedDirection: "UP",
		Probability:       0.80,
		BasePrice:         1000.0,
		TargetValue:       1050.0,
		EvaluationHorizon: "5D",
		DueAt:             now.Add(-1 * time.Hour), // already due
		StrategyVersion:   "v1.0-alpha",
		Status:            investment.PredictionPending,
		CreatedAt:         now.Add(-5 * 24 * time.Hour),
	}
	if err := application.InvSvc.CreatePrediction(ctx, pred); err != nil {
		t.Fatalf("failed to create prediction: %v", err)
	}

	// Verify it shows up in list
	pendingList, err := application.InvSvc.ListPredictions(ctx, investment.PredictionPending, 10)
	if err != nil {
		t.Fatalf("failed to list predictions: %v", err)
	}
	if len(pendingList) == 0 {
		t.Fatal("expected at least 1 pending prediction")
	}

	// 5. Evaluate predictions
	outcomes, evalRes, err := application.InvSvc.EvaluatePredictions(ctx)
	if err != nil {
		t.Fatalf("failed to evaluate predictions: %v", err)
	}
	if len(outcomes) == 0 {
		t.Fatal("expected at least 1 outcome evaluated")
	}
	if evalRes == nil || evalRes.SampleCount == 0 {
		t.Fatal("expected evaluation result summary generated")
	}
}

func TestScheduleManagementAndExecution(t *testing.T) {
	application, _ := createTestApp(t)
	defer application.Close()

	ctx := context.Background()

	// 1. Create a schedule
	sch, err := application.ScheduleSvc.CreateSchedule(
		ctx,
		"Daily TWSE Data Sync",
		"0 14 * * 1-5", // Weekdays at 14:00
		"investment-agent",
		`{"title": "Sync closing prices", "project_id": "default"}`,
		schedule.MisfireSkip,
	)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}
	if !sch.IsActive {
		t.Error("expected schedule to be active")
	}

	// 2. Trigger the schedule immediately
	dispatchedTask, err := application.ScheduleSvc.TriggerSchedule(ctx, sch.ID)
	if err != nil {
		t.Fatalf("failed to trigger schedule: %v", err)
	}
	if dispatchedTask == nil {
		t.Fatal("expected task to be created")
	}
	if dispatchedTask.AgentID != "investment-agent" {
		t.Errorf("expected agent investment-agent, got %s", dispatchedTask.AgentID)
	}

	// 3. Verify schedule updated last_run_at
	updatedSch, err := application.ScheduleSvc.GetSchedule(ctx, sch.ID)
	if err != nil {
		t.Fatalf("failed to get schedule: %v", err)
	}
	if updatedSch.LastRunAt == nil {
		t.Error("expected LastRunAt to be updated")
	}
}
