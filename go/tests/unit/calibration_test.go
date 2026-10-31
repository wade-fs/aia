package unit

import (
	"testing"
	"time"

	"github.com/wade/aia/internal/domain/investment"
)

func TestCalculateBrierScore(t *testing.T) {
	// If probability is 0.8 and event occurred (actual = 1), BS = (0.8 - 1)^2 = 0.04
	bs1 := investment.CalculateBrierScore(0.8, true)
	if bs1 != 0.04 {
		t.Errorf("expected 0.04, got %f", bs1)
	}

	// If probability is 0.7 and event did NOT occur (actual = 0), BS = (0.7 - 0)^2 = 0.49
	bs2 := investment.CalculateBrierScore(0.7, false)
	if bs2 != 0.49 {
		t.Errorf("expected 0.49, got %f", bs2)
	}

	// Perfect forecast: p = 1.0, occurred = true -> BS = 0
	bs3 := investment.CalculateBrierScore(1.0, true)
	if bs3 != 0.0 {
		t.Errorf("expected 0.0, got %f", bs3)
	}
}

func TestEvaluateSinglePrediction(t *testing.T) {
	pred := investment.Prediction{
		ID:                "pred-1",
		ExpectedDirection: "UP",
		Probability:       0.75,
		BasePrice:         1000.0,
		TargetValue:       1050.0,
	}

	// Current price is 1050 (up 5%) -> Hit!
	outcome := investment.EvaluateSinglePrediction(pred, 1050.0, "TEST")
	if outcome.OutcomeStatus != "HIT" {
		t.Errorf("expected HIT, got %s", outcome.OutcomeStatus)
	}
	if outcome.ActualDirection != "UP" {
		t.Errorf("expected UP direction, got %s", outcome.ActualDirection)
	}
	if outcome.BrierScore != 0.0625 { // (0.75 - 1.0)^2 = 0.0625
		t.Errorf("expected Brier Score 0.0625, got %f", outcome.BrierScore)
	}

	// Current price is 950 (down 5%) -> Miss!
	outcomeMiss := investment.EvaluateSinglePrediction(pred, 950.0, "TEST")
	if outcomeMiss.OutcomeStatus != "MISS" {
		t.Errorf("expected MISS, got %s", outcomeMiss.OutcomeStatus)
	}
	if outcomeMiss.BrierScore != 0.5625 { // (0.75 - 0.0)^2 = 0.5625
		t.Errorf("expected Brier Score 0.5625, got %f", outcomeMiss.BrierScore)
	}
}

func TestComputeCalibrationSummary(t *testing.T) {
	outcomes := []investment.PredictionOutcome{
		{OutcomeStatus: "HIT", BrierScore: 0.04, ActualValue: 5.0},
		{OutcomeStatus: "HIT", BrierScore: 0.09, ActualValue: 3.0},
		{OutcomeStatus: "MISS", BrierScore: 0.49, ActualValue: -2.0},
	}

	metrics := investment.ComputeCalibrationSummary(outcomes)
	if metrics.TotalEvaluated != 3 {
		t.Errorf("expected 3 evaluated, got %d", metrics.TotalEvaluated)
	}
	if metrics.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", metrics.Hits)
	}
	if metrics.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", metrics.Misses)
	}
	// Accuracy = 2/3 = 0.6667
	if metrics.DirectionalAccuracy < 0.66 || metrics.DirectionalAccuracy > 0.67 {
		t.Errorf("expected ~0.6667 accuracy, got %f", metrics.DirectionalAccuracy)
	}

	// Package into EvaluationResult
	now := time.Now()
	res := investment.BuildEvaluationResult(metrics, now.AddDate(0, -1, 0), now, "v1.0")
	if res.SampleCount != 3 {
		t.Errorf("expected sample count 3, got %d", res.SampleCount)
	}
}
