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

// CalculateLogLoss computes the binary cross-entropy Log Loss: -(y*ln(p) + (1-y)*ln(1-p)).
// Probabilities are clamped to [1e-6, 1-1e-6] to prevent infinity on 0/1 extremes.
func CalculateLogLoss(probability float64, occurred bool) float64 {
	const eps = 1e-6
	p := probability
	if p < eps {
		p = eps
	} else if p > 1.0-eps {
		p = 1.0 - eps
	}
	var loss float64
	if occurred {
		loss = -math.Log(p)
	} else {
		loss = -math.Log(1.0 - p)
	}
	return math.Round(loss*10000) / 10000
}

// CalibrationMetrics holds summary metrics for a batch of evaluated event predictions (AIA-05 §20.2).
type CalibrationMetrics struct {
	TotalEvaluated      int     `json:"total_evaluated"`
	Hits                int     `json:"hits"`
	Misses              int     `json:"misses"`
	Neutrals            int     `json:"neutrals"`
	DirectionalAccuracy float64 `json:"directional_accuracy"` // 0.0 ~ 1.0 (Hits / (Hits + Misses))
	AverageBrierScore   float64 `json:"average_brier_score"`   // Mean squared probability error
	AverageLogLoss      float64 `json:"average_log_loss"`      // Mean binary cross-entropy loss
	AverageReturnPct    float64 `json:"average_return_pct"`
	RangeCoverageRate   float64 `json:"range_coverage_rate"`   // Fraction of outcomes landing inside [range_low, range_high]
}

// DecisionConfidenceMetrics evaluates整體決策信心 (Decision.Confidence) separately from Event Probability (AIA-05 §20.2).
// Instead of treating Decision.Confidence as an event probability, it checks whether higher action confidence
// aligns with positive action edge (e.g. BUY/ADD followed by positive return, SELL/REDUCE avoiding drop, HOLD in flat noise)
// and quantifies overconfidence gap.
type DecisionConfidenceMetrics struct {
	SampleCount            int     `json:"sample_count"`
	AvgDecisionConfidence  float64 `json:"avg_decision_confidence"`
	FavorableActionRate    float64 `json:"favorable_action_rate"`
	OverconfidenceGap      float64 `json:"overconfidence_gap"`       // max(0, AvgDecisionConfidence - FavorableActionRate)
	ConfidenceWeightedEdge float64 `json:"confidence_weighted_edge"` // Sum(conf * signed_action_return_pct) / SampleCount
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
	logLoss := CalculateLogLoss(p.Probability, isHit)

	inRange := false
	hasRange := p.ExpectedRangeHighPct > p.ExpectedRangeLowPct
	if hasRange {
		inRange = priceChangePct >= p.ExpectedRangeLowPct && priceChangePct <= p.ExpectedRangeHighPct
	}

	details, _ := json.Marshal(map[string]any{
		"base_price":              p.BasePrice,
		"current_price":           currentPrice,
		"price_change_pct":        math.Round(priceChangePct*100) / 100,
		"target_value":            p.TargetValue,
		"expected_return_pct":     p.ExpectedReturnPct,
		"expected_range_low_pct":  p.ExpectedRangeLowPct,
		"expected_range_high_pct": p.ExpectedRangeHighPct,
		"has_range":               hasRange,
		"in_expected_range":       inRange,
	})

	return &PredictionOutcome{
		ID:                 uuid.New().String(),
		PredictionID:       p.ID,
		ActualValue:        priceChangePct,
		ActualDirection:    actualDirection,
		OutcomeStatus:      outcomeStatus,
		BrierScore:         brier,
		LogLoss:            logLoss,
		EvaluationSourceID: evalSourceID,
		EvaluatedAt:        time.Now().UTC(),
		DetailsJSON:        string(details),
	}
}

// ComputeCalibrationSummary calculates directional accuracy, Brier score, Log Loss, and range coverage across outcomes.
func ComputeCalibrationSummary(outcomes []PredictionOutcome) CalibrationMetrics {
	if len(outcomes) == 0 {
		return CalibrationMetrics{}
	}

	var totalBrier float64
	var totalLogLoss float64
	var totalReturn float64
	hits := 0
	misses := 0
	neutrals := 0
	rangeSamples := 0
	rangeHits := 0

	for _, oc := range outcomes {
		totalBrier += oc.BrierScore
		totalLogLoss += oc.LogLoss
		totalReturn += oc.ActualValue
		switch oc.OutcomeStatus {
		case "HIT":
			hits++
		case "MISS":
			misses++
		default:
			neutrals++
		}
		var det struct {
			HasRange bool `json:"has_range"`
			InRange  bool `json:"in_expected_range"`
		}
		if oc.DetailsJSON != "" && json.Unmarshal([]byte(oc.DetailsJSON), &det) == nil && det.HasRange {
			rangeSamples++
			if det.InRange {
				rangeHits++
			}
		}
	}

	evaluatedCount := len(outcomes)
	decisiveCount := hits + misses
	accuracy := 0.0
	if decisiveCount > 0 {
		accuracy = float64(hits) / float64(decisiveCount)
	}

	avgBrier := totalBrier / float64(evaluatedCount)
	avgLogLoss := totalLogLoss / float64(evaluatedCount)
	avgReturn := totalReturn / float64(evaluatedCount)
	coverage := 0.0
	if rangeSamples > 0 {
		coverage = float64(rangeHits) / float64(rangeSamples)
	}

	return CalibrationMetrics{
		TotalEvaluated:      evaluatedCount,
		Hits:                hits,
		Misses:              misses,
		Neutrals:            neutrals,
		DirectionalAccuracy: math.Round(accuracy*10000) / 10000,
		AverageBrierScore:   math.Round(avgBrier*10000) / 10000,
		AverageLogLoss:      math.Round(avgLogLoss*10000) / 10000,
		AverageReturnPct:    math.Round(avgReturn*100) / 100,
		RangeCoverageRate:   math.Round(coverage*10000) / 10000,
	}
}

// ComputeDecisionConfidenceSummary evaluates Decision.Confidence alignment against realized forward return
// without conflating action confidence with event forecast probability.
func ComputeDecisionConfidenceSummary(decisions []Decision, outcomesByDecision map[string]PredictionOutcome) DecisionConfidenceMetrics {
	var count int
	var sumConf float64
	var favorable int
	var sumWeightedEdge float64

	for _, d := range decisions {
		oc, ok := outcomesByDecision[d.ID]
		if !ok {
			continue
		}
		count++
		sumConf += d.Confidence
		var signedEdge float64
		isFav := false
		switch d.DecisionType {
		case DecisionBuy:
			signedEdge = oc.ActualValue
			isFav = oc.ActualValue > 0.15 // exceeds round-trip buy fee
		case DecisionSell:
			signedEdge = -oc.ActualValue
			isFav = oc.ActualValue < -0.15 // avoided drawdown
		default: // HOLD
			signedEdge = -math.Abs(oc.ActualValue) * 0.25
			isFav = math.Abs(oc.ActualValue) <= 3.0 // calm band where inaction conserved fees
		}
		if isFav {
			favorable++
		}
		sumWeightedEdge += d.Confidence * signedEdge
	}

	if count == 0 {
		return DecisionConfidenceMetrics{}
	}
	avgConf := sumConf / float64(count)
	favRate := float64(favorable) / float64(count)
	overconf := 0.0
	if avgConf > favRate {
		overconf = avgConf - favRate
	}

	return DecisionConfidenceMetrics{
		SampleCount:            count,
		AvgDecisionConfidence:  math.Round(avgConf*10000) / 10000,
		FavorableActionRate:    math.Round(favRate*10000) / 10000,
		OverconfidenceGap:      math.Round(overconf*10000) / 10000,
		ConfidenceWeightedEdge: math.Round((sumWeightedEdge/float64(count))*10000) / 10000,
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
