package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/app"
	"github.com/wade/aia/internal/autonomy/learning"
	domainExp "github.com/wade/aia/internal/domain/experiment"
	"github.com/wade/aia/internal/domain/investment"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/experiment"
	"github.com/wade/aia/internal/web/console"
)

type barPoint struct {
	AsOf  time.Time
	Close float64
}

func seedTSMCBars(t *testing.T, a *app.App, bars []barPoint) *investment.Instrument {
	t.Helper()
	ctx := context.Background()
	if err := a.InvSvc.EnsureDataSources(ctx); err != nil {
		t.Fatalf("ensure data sources: %v", err)
	}
	inst, err := a.InvRepo.GetInstrumentBySymbol(ctx, "", "2330")
	if err != nil || inst == nil {
		nowStr := time.Now().UTC().Format(time.RFC3339)
		_, execErr := a.DB.RawDB().ExecContext(ctx, `
			INSERT OR IGNORE INTO investment_instruments (id, symbol, exchange, asset_type, name, currency, is_active, created_at, updated_at)
			VALUES ('inst-2330', '2330', 'TWSE', 'STOCK', 'TSMC', 'TWD', 1, ?, ?)
		`, nowStr, nowStr)
		if execErr != nil {
			t.Fatalf("insert instrument 2330: %v", execErr)
		}
		inst, err = a.InvRepo.GetInstrumentBySymbol(ctx, "", "2330")
		if err != nil || inst == nil {
			t.Fatalf("load instrument 2330: %v", err)
		}
	}

	// Clear any pre-seeded bars for 2330 so test trajectory is 100% deterministic
	_, _ = a.DB.RawDB().ExecContext(ctx, `DELETE FROM investment_market_data WHERE instrument_id = ?`, inst.ID)

	for i, bp := range bars {
		valBytes, _ := json.Marshal(investment.MarketBarValue{
			Open:   bp.Close,
			High:   bp.Close * 1.01,
			Low:    bp.Close * 0.99,
			Close:  bp.Close,
			Volume: 25000000,
		})
		md := &investment.MarketData{
			ID:           "md-2330-" + bp.AsOf.Format("200601021504") + "-" + string(rune('a'+i)),
			InstrumentID: inst.ID,
			DataType:     "DAILY_BAR",
			Period:       "1D",
			ValueJSON:    string(valBytes),
			SourceID:     "ds-twse-openapi",
			FetchedAt:    bp.AsOf,
			EffectiveAt:  bp.AsOf,
		}
		if err := a.InvRepo.SaveMarketData(ctx, md); err != nil {
			t.Fatalf("save market bar at %s: %v", bp.AsOf, err)
		}
	}
	return inst
}

// TestPhase5A_Scenario1_FairnessLedgerIsolationAndZeroLookAhead verifies:
// - Identical market snapshots & fee rules presented to G1 and G2 at each timestamp
// - Complete portfolio ledger isolation (NT$100,000 each, independent cash/holdings/costs)
// - Zero Look-Ahead enforcement (future T5 spike at NT$1,500 is completely invisible at T1..T3)
// - Immutability of frozen G1 baseline
func TestPhase5A_Scenario1_FairnessLedgerIsolationAndZeroLookAhead(t *testing.T) {
	a, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	t0 := time.Date(2026, 5, 1, 6, 0, 0, 0, time.UTC)
	t1 := t0.Add(24 * time.Hour)
	t2 := t0.Add(48 * time.Hour)
	t3 := t0.Add(72 * time.Hour)
	tFuture := t0.Add(120 * time.Hour)

	seedTSMCBars(t, a, []barPoint{
		{AsOf: t0, Close: 1000},
		{AsOf: t1, Close: 960},     // -4.0% pullback -> triggers G1 pullback BUY rule
		{AsOf: t2, Close: 990},     // rebound
		{AsOf: t3, Close: 1020},    // further gain
		{AsOf: tFuture, Close: 1500}, // Future spike that MUST NOT leak into T1..T3
	})

	bundle, err := a.ExperimentCtrl.CreateDualExperiment(ctx, experiment.CreateDualInput{
		ProjectID:      "default",
		Name:           "Fairness & Isolation Verification",
		InitialCapital: 100000,
		StartAsOf:      &t0,
	})
	if err != nil {
		t.Fatalf("CreateDualExperiment failed: %v", err)
	}
	if !bundle.G1.IsFrozen || bundle.G2.IsFrozen || bundle.G1.PortfolioProfileID == bundle.G2.PortfolioProfileID {
		t.Fatalf("expected frozen G1 and unfrozen G2 with distinct portfolio profiles, got G1=%+v G2=%+v", bundle.G1, bundle.G2)
	}

	// Register G2 decider that stays cautious (HOLD at T1, BUY larger size at T2) and checks zero look-ahead
	a.ExperimentCtrl.RegisterGenerationDecider("G2", experiment.FuncGenerationDecider(func(_ context.Context, _ *domainExp.ExperimentGeneration, obs experiment.GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error) {
		if obs.CurrentBar.Close >= 1400 {
			t.Errorf("LOOK-AHEAD BIAS DETECTED at %s: saw future price %.2f", obs.AsOf, obs.CurrentBar.Close)
		}
		action := "HOLD"
		var targetAmt int64
		if obs.AsOf.Equal(t2) {
			action = "BUY"
			targetAmt = 60000
		}
		return &domainExp.GenerationalForecastOutput{
			Symbol:                 obs.Symbol,
			HorizonDays:            1,
			UpProbability:          0.68,
			ExpectedReturnPct:      2.5,
			ExpectedRangeLowPct:    -2.0,
			ExpectedRangeHighPct:   5.0,
			Action:                 action,
			TargetAmount:           targetAmt,
			DecisionConfidence:     0.72,
			Reasoning:              []string{"Wait for rebound confirmation at T2 before allocating NT$60k"},
			PrimaryRisks:           []string{"Short-term pullback continuation"},
			InvalidationConditions: []string{"Close drops below 930"},
		}, nil
	}))

	for _, ts := range []time.Time{t1, t2, t3} {
		stepRes, err := a.ExperimentCtrl.StepObservation(ctx, experiment.StepInput{
			ExperimentID: bundle.Experiment.ID,
			Symbol:       "2330",
			AsOf:         ts,
		})
		if err != nil {
			t.Fatalf("StepObservation at %s failed: %v", ts, err)
		}
		if len(stepRes.Outcomes) != 2 {
			t.Fatalf("expected 2 cohort outcomes at %s, got %d", ts, len(stepRes.Outcomes))
		}
		if stepRes.ReferencePrice >= 1400 {
			t.Fatalf("future price leaked into StepResult at %s: %.2f", ts, stepRes.ReferencePrice)
		}
	}

	cmp, err := a.ExperimentCtrl.CompareGenerations(ctx, bundle.Experiment.ID, "G1", "G2", t3)
	if err != nil {
		t.Fatalf("CompareGenerations failed: %v", err)
	}
	// G1 bought at 960 (T1) while G2 waited and bought at 990 (T2) -> distinct cash, distinct holdings, distinct total assets!
	if cmp.BaselineTrackA.TotalAssets == cmp.CandidateTrackA.TotalAssets {
		t.Fatalf("expected independent portfolio trajectories between G1 (%d) and G2 (%d)", cmp.BaselineTrackA.TotalAssets, cmp.CandidateTrackA.TotalAssets)
	}
	if cmp.BaselineTrackA.TotalTransactionCosts <= 0 || cmp.CandidateTrackA.TotalTransactionCosts <= 0 {
		t.Fatalf("expected both G1 (%d) and G2 (%d) to incur Taiwan broker fees on settled virtual trades",
			cmp.BaselineTrackA.TotalTransactionCosts, cmp.CandidateTrackA.TotalTransactionCosts)
	}

	// Governance check: Attempting to spawn an evolved generation from frozen G1 must be rejected
	_, err = a.ExperimentCtrl.SpawnGenerationFromCorrection(ctx, experiment.SpawnGenerationInput{
		ExperimentID:      bundle.Experiment.ID,
		ParentCode:        "G1",
		NewGenerationCode: "G1.1",
	})
	if err == nil || !strings.Contains(err.Error(), "frozen baseline generation") {
		t.Fatalf("expected SpawnGenerationFromCorrection from G1 to fail with governance error, got %v", err)
	}
}

// TestPhase5A_Scenario2_FiveRegimeMatrix_UnreliableRiskTakerDetection verifies dual.md §4:
// Higher total return with terrible Brier/LogLoss & overconfidence MUST be classified as G2_UNRELIABLE_RISK_TAKING,
// NOT G2_CLEAR_ADVANTAGE!
func TestPhase5A_Scenario2_FiveRegimeMatrix_UnreliableRiskTakerDetection(t *testing.T) {
	a, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	t0 := time.Date(2026, 6, 1, 6, 0, 0, 0, time.UTC)
	t1 := t0.Add(24 * time.Hour)
	t2 := t0.Add(48 * time.Hour)
	t3 := t0.Add(72 * time.Hour)
	t4 := t0.Add(96 * time.Hour)

	// Market: T0=1000 -> T1=1005 (+0.5%, G1 stays HOLD) -> T2=990 (-1.5% miss!) -> T3=980 (-1.0% miss!) -> T4=1080 (+10.2% lucky jump!)
	seedTSMCBars(t, a, []barPoint{
		{AsOf: t0, Close: 1000},
		{AsOf: t1, Close: 1005},
		{AsOf: t2, Close: 990},
		{AsOf: t3, Close: 980},
		{AsOf: t4, Close: 1080},
	})

	bundle, err := a.ExperimentCtrl.CreateDualExperiment(ctx, experiment.CreateDualInput{
		ProjectID:      "default",
		Name:           "Overconfident Risk-Taker Detection",
		InitialCapital: 100000,
		StartAsOf:      &t0,
	})
	if err != nil {
		t.Fatalf("CreateDualExperiment: %v", err)
	}

	// Reckless G2 goes max-position BUY at T1 (80,000 TWD) and repeatedly predicts 0.97 UP probability with 0.98 confidence even on down days T2 & T3!
	a.ExperimentCtrl.RegisterGenerationDecider("G2", experiment.FuncGenerationDecider(func(_ context.Context, _ *domainExp.ExperimentGeneration, obs experiment.GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error) {
		action := "HOLD"
		var amt int64
		if obs.AsOf.Equal(t1) {
			action = "BUY"
			amt = 80000
		}
		return &domainExp.GenerationalForecastOutput{
			Symbol:                 obs.Symbol,
			HorizonDays:            1,
			UpProbability:          0.97, // Extreme overconfident probability -> crushed on T1->T2 and T2->T3 drops!
			ExpectedReturnPct:      6.0,
			ExpectedRangeLowPct:    3.0,
			ExpectedRangeHighPct:   10.0,
			Action:                 action,
			TargetAmount:           amt,
			DecisionConfidence:     0.98,
			Reasoning:              []string{"All-in high-beta bet"},
			PrimaryRisks:           []string{"Drawdown"},
			InvalidationConditions: []string{"None"},
		}, nil
	}))

	for _, ts := range []time.Time{t1, t2, t3, t4} {
		if _, err := a.ExperimentCtrl.StepObservation(ctx, experiment.StepInput{
			ExperimentID: bundle.Experiment.ID,
			Symbol:       "2330",
			AsOf:         ts,
		}); err != nil {
			t.Fatalf("step %s: %v", ts, err)
		}
	}

	cmp, err := a.ExperimentCtrl.CompareGenerations(ctx, bundle.Experiment.ID, "G1", "G2", t4)
	if err != nil {
		t.Fatalf("CompareGenerations: %v", err)
	}
	if cmp.CandidateTrackA.CumulativeReturnPct <= cmp.BaselineTrackA.CumulativeReturnPct {
		t.Fatalf("expected reckless G2 (%.2f%%) to have higher lucky return than G1 (%.2f%%)",
			cmp.CandidateTrackA.CumulativeReturnPct, cmp.BaselineTrackA.CumulativeReturnPct)
	}
	if cmp.CandidateTrackA.EventCalibration.AverageBrierScore <= 0.30 {
		t.Fatalf("expected poor Brier score (> 0.30) for 0.97 misses, got %.4f", cmp.CandidateTrackA.EventCalibration.AverageBrierScore)
	}
	if cmp.RegimeVerdict != domainExp.RegimeG2UnreliableRiskTaking {
		t.Fatalf("expected RegimeG2UnreliableRiskTaking, got %s (%s)", cmp.RegimeVerdict, cmp.Summary)
	}
}

// TestPhase5A_Scenario3_GenerationalEvolution_G2ToG2_1_LiveG1ControlAndRollback verifies the full loop:
// 1. Window 1 (T1..T3): Flawed G2 trails frozen G1 -> G1_RULES_SUPERIOR
// 2. Phase 4 RCA + Tier 2 Strategy Correction (human approved) -> Spawn G2.1 (historical G2 untouched!)
// 3. Window 2 (T4..T6): Disciplined G2.1 beats G1 with calibrated forecasts -> VerifyGenerationWithG1Control promotes G2.1 to PROMOTED & LearningRecord to L2_VERIFIED (EFFECTIVE, confidence > 0.45)
// 4. Window 3 (T7..T8): Flawed G2.2 causes severe Brier regression -> auto-rolled back to ROLLED_BACK (REGRESSED)
// 5. Web Console /api/v1/experiments surfaces the full lineage & comparisons.
func TestPhase5A_Scenario3_GenerationalEvolution_G2ToG2_1_LiveG1ControlAndRollback(t *testing.T) {
	a, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	t0 := time.Date(2026, 7, 1, 6, 0, 0, 0, time.UTC)
	days := make([]time.Time, 9)
	for i := 0; i <= 8; i++ {
		days[i] = t0.Add(time.Duration(i*24) * time.Hour)
	}

	// Price trajectory:
	// Window 1 (days[1..3]): 1000 -> 1020 (+2%, G2 chases top!) -> 975 (-4.4%, G2 hurt while G1 buys pullback!) -> 980
	// Window 2 (days[4..6]): 980 -> 965 (-1.5%, disciplined G2.1 buys value dip) -> 1005 (+4.1%) -> 1045 (+4.0%)
	// Window 3 (days[7..8]): 1045 -> 1015 (-2.9%) -> 990 (-2.5%)
	seedTSMCBars(t, a, []barPoint{
		{AsOf: days[0], Close: 1000},
		{AsOf: days[1], Close: 1020},
		{AsOf: days[2], Close: 975},
		{AsOf: days[3], Close: 980},
		{AsOf: days[4], Close: 965},
		{AsOf: days[5], Close: 1005},
		{AsOf: days[6], Close: 1045},
		{AsOf: days[7], Close: 1015},
		{AsOf: days[8], Close: 990},
	})

	bundle, err := a.ExperimentCtrl.CreateDualExperiment(ctx, experiment.CreateDualInput{
		ProjectID:      "default",
		Name:           "Generational Lineage Evolution G2 -> G2.1 -> G2.2",
		InitialCapital: 100000,
		StartAsOf:      &days[0],
	})
	if err != nil {
		t.Fatalf("CreateDualExperiment: %v", err)
	}

	// G2 (initial flawed strategy): buys 80k on day 1 top (1020) and panic-sells on day 2 dip (975)!
	a.ExperimentCtrl.RegisterGenerationDecider("G2", experiment.FuncGenerationDecider(func(_ context.Context, _ *domainExp.ExperimentGeneration, obs experiment.GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error) {
		act := "HOLD"
		var amt int64
		if obs.AsOf.Equal(days[1]) {
			act = "BUY"
			amt = 80000
		} else if obs.AsOf.Equal(days[2]) {
			act = "SELL"
		}
		return &domainExp.GenerationalForecastOutput{
			Symbol:                 obs.Symbol,
			HorizonDays:            1,
			UpProbability:          0.70,
			ExpectedReturnPct:      2.0,
			ExpectedRangeLowPct:    -2.0,
			ExpectedRangeHighPct:   5.0,
			Action:                 act,
			TargetAmount:           amt,
			DecisionConfidence:     0.75,
			Reasoning:              []string{"Momentum chase without pullback filter"},
			PrimaryRisks:           []string{"False breakout"},
			InvalidationConditions: []string{"Drop below 990"},
		}, nil
	}))

	for _, ts := range []time.Time{days[1], days[2], days[3]} {
		if _, err := a.ExperimentCtrl.StepObservation(ctx, experiment.StepInput{
			ExperimentID: bundle.Experiment.ID,
			Symbol:       "2330",
			AsOf:         ts,
		}); err != nil {
			t.Fatalf("Window 1 step %s: %v", ts, err)
		}
	}

	cmpW1, err := a.ExperimentCtrl.CompareGenerations(ctx, bundle.Experiment.ID, "G1", "G2", days[3])
	if err != nil {
		t.Fatalf("CompareGenerations W1: %v", err)
	}
	if cmpW1.RegimeVerdict != domainExp.RegimeG1RulesSuperior {
		t.Fatalf("expected G1_RULES_SUPERIOR in Window 1 after G2 bought top & panic-sold, got %s (%s)", cmpW1.RegimeVerdict, cmpW1.Summary)
	}
	g2HistoricalAssets := cmpW1.CandidateTrackA.TotalAssets

	// Phase 4 Closed-Loop RCA + Approved Tier 2 Strategy Correction
	learnEng := a.AutonomyEngine.LearningEngine()
	evi := seedEvidence(t, ctx, a.KnowledgeRepo, "evi-g2-whipsaw", "artifact://g2_window1_ledger.json",
		"G2 chased +2% breakout at 1020 with 80% capital and panic-sold at 975, losing -3.8% while frozen G1 preserved capital")
	projID := "default"
	oe, lr1, err := learnEng.RecordDiscrepancy(ctx, learning.DiscrepancyInput{
		ProjectID:      &projID,
		Domain:         "investment",
		Title:          "G2 whipsaw loss from uncalibrated breakout chasing",
		MetricName:     "portfolio_return_pct",
		ExpectedMetric: cmpW1.BaselineTrackA.CumulativeReturnPct,
		ActualMetric:   cmpW1.CandidateTrackA.CumulativeReturnPct,
		LowerIsBetter:  false,
		ActualContext:  map[string]any{"g2_return_pct": cmpW1.CandidateTrackA.CumulativeReturnPct, "g1_return_pct": cmpW1.BaselineTrackA.CumulativeReturnPct},
	})
	if err != nil {
		t.Fatalf("RecordDiscrepancy: %v", err)
	}

	rca1, err := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe.ID,
		LearningRecordID:      lr1.ID,
		Category:              domainLearning.RCFlawedStrategy,
		Narrative:             "Breakout entry lacked pullback confirmation and over-allocated 80% on single bar",
		FalsifiableHypothesis: "Replacing breakout chase with pullback entry <= -1.2% and calibrated horizon probabilities improves excess return over frozen G1 by >= +0.50%",
		CausalEvidenceRefs:    []string{evi.ID},
		CompetingFactors:      []string{"Short-term semiconductor pullback"},
		InitialConfidence:     0.80,
	})
	if err != nil {
		t.Fatalf("AnalyzeRootCause: %v", err)
	}

	ca1, lr1, err := learnEng.ProposeAndApplyCorrection(ctx, learning.CorrectionInput{
		RCAID:            rca1.ID,
		LearningRecordID: lr1.ID,
		Scope:            domainLearning.ScopeA,
		CorrectionType:   domainLearning.CorrectionStrategy,
		TargetEntityRef:  "strategy:g2-entry-filter",
		BeforeState:      map[string]any{"entry": "chase_breakout"},
		ProposedState:    map[string]any{"entry": "pullback_with_calibrated_prob"},
	})
	if err != nil || ca1.ApprovalID == nil {
		t.Fatalf("expected TIER2_APPROVAL strategy correction with ApprovalID, got ca=%+v err=%v", ca1, err)
	}
	if err := a.ApprovalSvc.Approve(ctx, *ca1.ApprovalID, "human-researcher", "Approved for G2.1 lineage spawn", "HUMAN"); err != nil {
		t.Fatalf("approve strategy correction: %v", err)
	}
	ca1, lr1, err = learnEng.ApplyApprovedCorrection(ctx, ca1.ID, lr1.ID)
	if err != nil || lr1.LearningLevel != domainLearning.LevelL1Applied {
		t.Fatalf("expected L1_APPLIED after approval, got level=%s err=%v", lr1.LearningLevel, err)
	}

	// Spawn G2.1 at days[3], retiring G2 from future steps while keeping G2's historical ledger 100% intact!
	g21, err := a.ExperimentCtrl.SpawnGenerationFromCorrection(ctx, experiment.SpawnGenerationInput{
		ExperimentID:       bundle.Experiment.ID,
		ParentCode:         "G2",
		NewGenerationCode:  "G2.1",
		PromptVersion:      "inv-generational-v2",
		StrategyVersion:    "llm-v2.1-pullback-calibrated",
		StrategyConfigJSON: `{"entry":"pullback_with_calibrated_prob","max_tranche_twd":75000}`,
		SourceCorrectionID: ca1.ID,
		SourceLearningID:   lr1.ID,
		InitialCapital:     100000,
		SupersedeParent:    true,
		AsOf:               &days[3],
	})
	if err != nil || g21.GenerationCode != "G2.1" {
		t.Fatalf("SpawnGenerationFromCorrection G2.1: %v", err)
	}

	// Disciplined G2.1 buys value dip at days[4] (965 -> -1.53% dip, where G1's -3.0% rule stays flat) with well-calibrated probabilities!
	a.ExperimentCtrl.RegisterGenerationDecider("G2.1", experiment.FuncGenerationDecider(func(_ context.Context, _ *domainExp.ExperimentGeneration, obs experiment.GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error) {
		act := "HOLD"
		var amt int64
		prob := 0.78
		if obs.AsOf.Equal(days[4]) {
			act = "BUY"
			amt = 75000
			prob = 0.82 // Price rises 965 -> 1005 -> 1045 -> well-calibrated!
		} else if obs.AsOf.Equal(days[6]) {
			act = "SELL" // Harvest profit before Window 3 dip!
			prob = 0.25  // Correctly anticipates pullback 1045 -> 1015!
		}
		return &domainExp.GenerationalForecastOutput{
			Symbol:                 obs.Symbol,
			HorizonDays:            1,
			UpProbability:          prob,
			ExpectedReturnPct:      3.5,
			ExpectedRangeLowPct:    -2.0,
			ExpectedRangeHighPct:   5.5,
			Action:                 act,
			TargetAmount:           amt,
			DecisionConfidence:     0.80,
			Reasoning:              []string{"Pullback entry with calibrated regime probability (learned from G2 RCA)"},
			PrimaryRisks:           []string{"Macro rate volatility"},
			InvalidationConditions: []string{"Reassess if close breaks below 945"},
		}, nil
	}))

	for _, ts := range []time.Time{days[4], days[5], days[6]} {
		if _, err := a.ExperimentCtrl.StepObservation(ctx, experiment.StepInput{
			ExperimentID: bundle.Experiment.ID,
			Symbol:       "2330",
			AsOf:         ts,
		}); err != nil {
			t.Fatalf("Window 2 step %s: %v", ts, err)
		}
	}

	// Verify G2's historical ledger was NEVER overwritten or mutated during Window 2!
	_, g2SnapAfterW2, _, err := a.InvSvc.GetSnapshotForProfile(ctx, bundle.G2.PortfolioProfileID)
	if err != nil || g2SnapAfterW2.TotalPortfolioValue != g2HistoricalAssets {
		t.Fatalf("expected retired G2 ledger (%d) to remain 100%% untouched, got %d (err=%v)",
			g2HistoricalAssets, g2SnapAfterW2.TotalPortfolioValue, err)
	}

	// Verify G2.1 against live G1 Control Benchmark -> must reach L2_VERIFIED (EFFECTIVE, confidence > 0.45) and PROMOTED!
	cmpW2, verifiedLR, updatedG21, err := a.ExperimentCtrl.VerifyGenerationWithG1Control(ctx, experiment.VerifyGenerationInput{
		ExperimentID:  bundle.Experiment.ID,
		CandidateCode: "G2.1",
		AsOf:          days[6],
	})
	if err != nil {
		t.Fatalf("VerifyGenerationWithG1Control G2.1: %v", err)
	}
	if verifiedLR.LearningLevel != domainLearning.LevelL2Verified || verifiedLR.EffectivenessVerdict != domainLearning.VerdictEffective {
		t.Fatalf("expected G2.1 LearningRecord to reach L2_VERIFIED / EFFECTIVE via G1 control, got level=%s verdict=%s",
			verifiedLR.LearningLevel, verifiedLR.EffectivenessVerdict)
	}
	if verifiedLR.ComparisonMethod != domainLearning.CompControlBenchmark || verifiedLR.EpistemicConfidence <= 0.45 {
		t.Fatalf("expected CompControlBenchmark to lift confidence above 0.45 cap, got method=%s conf=%.2f",
			verifiedLR.ComparisonMethod, verifiedLR.EpistemicConfidence)
	}
	if updatedG21.Status != domainExp.GenStatusPromoted {
		t.Fatalf("expected G2.1 status PROMOTED, got %s", updatedG21.Status)
	}
	if cmpW2.RegimeVerdict != domainExp.RegimeG2ClearAdvantage {
		t.Fatalf("expected G2_CLEAR_ADVANTAGE for calibrated G2.1, got %s (%s)", cmpW2.RegimeVerdict, cmpW2.Summary)
	}

	// Now test Regression & Automatic Rollback with G2.2 in Window 3 (days[7..8])!
	oe2, lr2, err := learnEng.RecordDiscrepancy(ctx, learning.DiscrepancyInput{
		ProjectID:      &projID,
		Domain:         "investment",
		Title:          "Attempt aggressive 0.99 probability override in G2.2",
		MetricName:     "brier_score",
		ExpectedMetric: 0.10,
		ActualMetric:   0.22,
		LowerIsBetter:  true,
	})
	if err != nil {
		t.Fatalf("RecordDiscrepancy 2: %v", err)
	}
	rca2, err := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe2.ID,
		LearningRecordID:      lr2.ID,
		Category:              domainLearning.RCBiasedJudgment,
		Narrative:             "Test aggressive probability sharpening",
		FalsifiableHypothesis: "Forcing 0.99 UP probability sharpens Brier without calibration blowup",
		CausalEvidenceRefs:    []string{evi.ID},
		InitialConfidence:     0.65,
	})
	if err != nil {
		t.Fatalf("AnalyzeRootCause 2: %v", err)
	}
	ca2, lr2, err := learnEng.ProposeAndApplyCorrection(ctx, learning.CorrectionInput{
		RCAID:            rca2.ID,
		LearningRecordID: lr2.ID,
		Scope:            domainLearning.ScopeA,
		CorrectionType:   domainLearning.CorrectionJudgment, // TIER1_AUTO
		TargetEntityRef:  "judgment:prob-sharpening",
		BeforeState:      map[string]any{"prob_cap": 0.85},
		ProposedState:    map[string]any{"prob_cap": 0.99},
	})
	if err != nil {
		t.Fatalf("ProposeAndApplyCorrection 2: %v", err)
	}

	_, err = a.ExperimentCtrl.SpawnGenerationFromCorrection(ctx, experiment.SpawnGenerationInput{
		ExperimentID:       bundle.Experiment.ID,
		ParentCode:         "G2.1",
		NewGenerationCode:  "G2.2",
		StrategyVersion:    "llm-v2.2-overconfident-regression",
		SourceCorrectionID: ca2.ID,
		SourceLearningID:   lr2.ID,
		InitialCapital:     100000,
		SupersedeParent:    false,
		AsOf:               &days[6],
	})
	if err != nil {
		t.Fatalf("Spawn G2.2: %v", err)
	}

	a.ExperimentCtrl.RegisterGenerationDecider("G2.2", experiment.FuncGenerationDecider(func(_ context.Context, _ *domainExp.ExperimentGeneration, obs experiment.GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error) {
		return &domainExp.GenerationalForecastOutput{
			Symbol:                 obs.Symbol,
			HorizonDays:            1,
			UpProbability:          0.99, // Blows up Brier (>0.95!) when days[7]->days[8] drops 1015->990!
			ExpectedReturnPct:      5.0,
			ExpectedRangeLowPct:    2.0,
			ExpectedRangeHighPct:   8.0,
			Action:                 "HOLD",
			DecisionConfidence:     0.99,
			Reasoning:              []string{"Over-sharpened probability"},
			PrimaryRisks:           []string{"Calibration blowup"},
			InvalidationConditions: []string{"None"},
		}, nil
	}))

	for _, ts := range []time.Time{days[7], days[8]} {
		if _, err := a.ExperimentCtrl.StepObservation(ctx, experiment.StepInput{
			ExperimentID: bundle.Experiment.ID,
			Symbol:       "2330",
			AsOf:         ts,
		}); err != nil {
			t.Fatalf("Window 3 step %s: %v", ts, err)
		}
	}

	_, regLR, updatedG22, err := a.ExperimentCtrl.VerifyGenerationWithG1Control(ctx, experiment.VerifyGenerationInput{
		ExperimentID:  bundle.Experiment.ID,
		CandidateCode: "G2.2",
		AsOf:          days[8],
	})
	if err != nil {
		t.Fatalf("VerifyGenerationWithG1Control G2.2: %v", err)
	}
	if regLR.EffectivenessVerdict != domainLearning.VerdictRegressed || updatedG22.Status != domainExp.GenStatusRolledBack {
		t.Fatalf("expected G2.2 regression to mark REGRESSED and ROLLED_BACK, got verdict=%s status=%s",
			regLR.EffectivenessVerdict, updatedG22.Status)
	}

	// Verify Web Console /api/v1/experiments endpoint
	mux := http.NewServeMux()
	console.RegisterConsoleRoutes(mux, &console.ConsoleServices{
		ExperimentRepo: a.ExperimentRepo,
		ExperimentCtrl: a.ExperimentCtrl,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/experiments/"+bundle.Experiment.ID, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from GET /api/v1/experiments/<id>, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"G2.1"`) || !strings.Contains(rec.Body.String(), `"PROMOTED"`) || !strings.Contains(rec.Body.String(), `"ROLLED_BACK"`) {
		t.Fatalf("expected experiment detail JSON to include G2.1 PROMOTED and G2.2 ROLLED_BACK, got %s", rec.Body.String())
	}
}
