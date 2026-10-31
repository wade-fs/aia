package investment

import (
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
)

// CalculateBrierScore computes the standard Brier Score for a probabilistic forecast: (probability - actual)^2.
// Lower scores indicate higher calibration accuracy (0 = perfect calibration, 1 = worst).
func CalculateBrierScore(probability float64, occurred bool) float64 {
	var actual float64
	if occurred {
		actual = 1.0
	}
	diff := probability - actual
	return math.Round(diff*diff*10000) / 10000
}

// CalibrationMetrics holds summary metrics for a batch of evaluated predictions.
type CalibrationMetrics struct {
	TotalEvaluated      int     `json:"total_evaluated"`
	Hits                int     `json:"hits"`
	Misses              int     `json:"misses"`
	Neutrals            int     `json:"neutrals"`
	DirectionalAccuracy float64 `json:"directional_accuracy"` // 0.0 ~ 1.0 (Hits / (Hits + Misses))
	AverageBrierScore   float64 `json:"average_brier_score"`   // Mean squared probability error
	AverageReturnPct    float64 `json:"average_return_pct"`
}

// EvaluateSinglePrediction evaluates a prediction against current/closing market price.
func EvaluateSinglePrediction(p Prediction, currentPrice float64, evalSourceID string) *PredictionOutcome {
	if p.BasePrice <= 0 {
		p.BasePrice = currentPrice
	}

	priceChangePct := 0.0
	if p.BasePrice > 0 {
		priceChangePct = ((currentPrice - p.BasePrice) / p.BasePrice) * 100
	}

	actualDirection := "FLAT"
	if priceChangePct > 0.1 {
		actualDirection = "UP"
	} else if priceChangePct < -0.1 {
		actualDirection = "DOWN"
	}

	isHit := false
	outcomeStatus := "MISS"

	switch p.ExpectedDirection {
	case "UP":
		isHit = (actualDirection == "UP")
	case "DOWN":
		isHit = (actualDirection == "DOWN")
	case "FLAT":
		isHit = (actualDirection == "FLAT")
	default:
		isHit = (actualDirection == p.ExpectedDirection)
	}

	if isHit {
		outcomeStatus = "HIT"
	} else if actualDirection == "FLAT" && p.ExpectedDirection != "FLAT" {
		outcomeStatus = "NEUTRAL"
	}

	brier := CalculateBrierScore(p.Probability, isHit)

	details, _ := json.Marshal(map[string]any{
		"base_price":       p.BasePrice,
		"current_price":    currentPrice,
		"price_change_pct": math.Round(priceChangePct*100) / 100,
		"target_value":     p.TargetValue,
	})

	return &PredictionOutcome{
		ID:                 uuid.New().String(),
		PredictionID:       p.ID,
		ActualValue:        priceChangePct,
		ActualDirection:    actualDirection,
		OutcomeStatus:      outcomeStatus,
		BrierScore:         brier,
		EvaluationSourceID: evalSourceID,
		EvaluatedAt:        time.Now().UTC(),
		DetailsJSON:        string(details),
	}
}

// ComputeCalibrationSummary calculates directional accuracy and average Brier score across outcomes.
func ComputeCalibrationSummary(outcomes []PredictionOutcome) CalibrationMetrics {
	if len(outcomes) == 0 {
		return CalibrationMetrics{}
	}

	var totalBrier float64
	var totalReturn float64
	hits := 0
	misses := 0
	neutrals := 0

	for _, oc := range outcomes {
		totalBrier += oc.BrierScore
		totalReturn += oc.ActualValue
		switch oc.OutcomeStatus {
		case "HIT":
			hits++
		case "MISS":
			misses++
		default:
			neutrals++
		}
	}

	evaluatedCount := len(outcomes)
	decisiveCount := hits + misses
	accuracy := 0.0
	if decisiveCount > 0 {
		accuracy = float64(hits) / float64(decisiveCount)
	}

	avgBrier := totalBrier / float64(evaluatedCount)
	avgReturn := totalReturn / float64(evaluatedCount)

	return CalibrationMetrics{
		TotalEvaluated:      evaluatedCount,
		Hits:                hits,
		Misses:              misses,
		Neutrals:            neutrals,
		DirectionalAccuracy: math.Round(accuracy*10000) / 10000,
		AverageBrierScore:   math.Round(avgBrier*10000) / 10000,
		AverageReturnPct:    math.Round(avgReturn*100) / 100,
	}
}

// BuildEvaluationResult packages summary metrics into an EvaluationResult model for persistence.
func BuildEvaluationResult(metrics CalibrationMetrics, periodStart, periodEnd time.Time, strategyVersion string) *EvaluationResult {
	metricsJSON, _ := json.Marshal(metrics)
	return &EvaluationResult{
		ID:              uuid.New().String(),
		EvaluationType:  "CALIBRATION",
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		SampleCount:     metrics.TotalEvaluated,
		MetricsJSON:     string(metricsJSON),
		StrategyVersion: strategyVersion,
		CreatedAt:       time.Now().UTC(),
	}
}
