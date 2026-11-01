package unit

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	domainExp "github.com/wade/aia/internal/domain/experiment"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestMigration012_ExperimentSchemaTriggersAndDualCalibration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	ver, err := db.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("failed to read schema version: %v", err)
	}
	if ver < 12 {
		t.Fatalf("expected schema version >= 12, got %d", ver)
	}

	expRepo := repository.NewExperimentRepository(db)
	invRepo := repository.NewInvestmentRepository(db)

	// 1. Verify AIA-15 11-state CHECK constraint on experiments
	validExp := &domainExp.Experiment{
		ID:                  "exp-m12-1",
		Domain:              "investment",
		OwnerAgentID:        "investment-agent",
		Name:                "G1 vs G2 Dual Experiment",
		Hypothesis:          "Calibrated G2 beats frozen G1",
		ExperimentType:      "GENERATIONAL_COMPARISON",
		Status:              domainExp.StatusRunning,
		BaselineMetricsJSON: `{"capital":100000}`,
		CurrentMetricsJSON:  `{}`,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := expRepo.CreateExperiment(ctx, validExp); err != nil {
		t.Fatalf("create valid experiment: %v", err)
	}

	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO experiments (id, domain, owner_agent_id, name, hypothesis, status, created_at, updated_at)
		VALUES ('exp-bad-status', 'investment', 'investment-agent', 'Bad', 'Hyp', 'BOGUS_STATUS', ?, ?)
	`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err == nil {
		t.Fatalf("expected SQLite CHECK failure on invalid experiment status")
	}

	// 2. Seed isolated profiles & verify CONTROL_BASELINE must be frozen + SQLite immutability triggers
	if _, _, err := invRepo.CreateIsolatedProfile(ctx, "prof-m12-g1", "G1 Profile", 100000, 0.85); err != nil {
		t.Fatalf("create G1 profile: %v", err)
	}
	if _, _, err := invRepo.CreateIsolatedProfile(ctx, "prof-m12-g2", "G2 Profile", 100000, 0.85); err != nil {
		t.Fatalf("create G2 profile: %v", err)
	}

	// Unfrozen CONTROL_BASELINE must be rejected by DB CHECK constraint
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO experiment_generations (
			id, experiment_id, generation_code, role, engine_type, portfolio_profile_id,
			is_frozen, strategy_version, status, created_at, updated_at
		) VALUES ('gen-unfrozen-g1', 'exp-m12-1', 'G1_BAD', 'CONTROL_BASELINE', 'RULE_BASED', 'prof-m12-g2', 0, 'v1', 'ACTIVE', ?, ?)
	`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err == nil {
		t.Fatalf("expected CHECK failure when inserting CONTROL_BASELINE with is_frozen=0")
	}

	g1 := &domainExp.ExperimentGeneration{
		ID:                 "gen-m12-g1",
		ExperimentID:       validExp.ID,
		GenerationCode:     "G1",
		Role:               domainExp.RoleControlBaseline,
		EngineType:         domainExp.EngineRuleBased,
		PortfolioProfileID: "prof-m12-g1",
		IsFrozen:           true,
		HarnessVersion:     "go-native",
		ModelID:            "deterministic-rules",
		PromptVersion:      "none",
		StrategyVersion:    "rule-v1.0-frozen",
		StrategyConfigJSON: `{"stop_loss_pct":-6.0}`,
		Status:             domainExp.GenStatusActive,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := expRepo.CreateGeneration(ctx, g1); err != nil {
		t.Fatalf("create frozen G1 generation: %v", err)
	}

	// Trigger trg_exp_gen_freeze_update must block any mutation of frozen G1 strategy/prompt/unfreeze
	_, err = db.RawDB().ExecContext(ctx, `
		UPDATE experiment_generations SET strategy_version = 'tampered-v2' WHERE id = 'gen-m12-g1'
	`)
	if err == nil || !strings.Contains(err.Error(), "frozen baseline generation cannot be modified") {
		t.Fatalf("expected trg_exp_gen_freeze_update to block G1 mutation, got err=%v", err)
	}

	_, err = db.RawDB().ExecContext(ctx, `
		UPDATE experiment_generations SET is_frozen = 0 WHERE id = 'gen-m12-g1'
	`)
	if err == nil || !strings.Contains(err.Error(), "frozen baseline generation cannot be modified") {
		t.Fatalf("expected trg_exp_gen_freeze_update to block unfreezing G1, got err=%v", err)
	}

	// Trigger trg_exp_gen_freeze_delete must block deleting frozen G1
	_, err = db.RawDB().ExecContext(ctx, `DELETE FROM experiment_generations WHERE id = 'gen-m12-g1'`)
	if err == nil || !strings.Contains(err.Error(), "frozen baseline generation cannot be deleted") {
		t.Fatalf("expected trg_exp_gen_freeze_delete to block G1 deletion, got err=%v", err)
	}

	// 3. Verify Dual Calibration Separation: Event Probability (Brier + LogLoss + Range) vs Decision Confidence
	llGood := investment.CalculateLogLoss(0.80, true)
	llBad := investment.CalculateLogLoss(0.95, false)
	if llGood >= 0.30 || llBad <= 2.50 {
		t.Fatalf("unexpected log loss values: llGood=%.4f, llBad=%.4f", llGood, llBad)
	}

	predMiss := investment.Prediction{
		ID:                   "pred-1",
		DecisionID:           "dec-1",
		InstrumentID:         "inst-2330",
		ExpectedDirection:    "UP",
		Probability:          0.95,
		ExpectedReturnPct:    5.0,
		ExpectedRangeLowPct:  1.0,
		ExpectedRangeHighPct: 8.0,
		BasePrice:            1000,
	}
	ocMiss := investment.EvaluateSinglePrediction(predMiss, 940, "TWSE") // -6% drop -> miss & outside [1%, 8%]
	if ocMiss.OutcomeStatus != "MISS" || ocMiss.BrierScore < 0.85 || ocMiss.LogLoss < 2.5 {
		t.Fatalf("expected high Brier & LogLoss on 0.95 confident miss, got %+v", ocMiss)
	}

	predHit := investment.Prediction{
		ID:                   "pred-2",
		DecisionID:           "dec-2",
		InstrumentID:         "inst-2330",
		ExpectedDirection:    "UP",
		Probability:          0.75,
		ExpectedReturnPct:    3.0,
		ExpectedRangeLowPct:  -1.0,
		ExpectedRangeHighPct: 5.0,
		BasePrice:            1000,
	}
	ocHit := investment.EvaluateSinglePrediction(predHit, 1030, "TWSE") // +3% -> hit & inside [-1%, 5%]
	if ocHit.OutcomeStatus != "HIT" || ocHit.BrierScore > 0.07 {
		t.Fatalf("expected low Brier on calibrated hit, got %+v", ocHit)
	}

	calSummary := investment.ComputeCalibrationSummary([]investment.PredictionOutcome{*ocMiss, *ocHit})
	if math.Abs(calSummary.RangeCoverageRate-0.5) > 0.001 {
		t.Fatalf("expected RangeCoverageRate=0.50 (1 of 2 within range), got %.4f", calSummary.RangeCoverageRate)
	}

	// Decision Confidence calibration evaluated separately from Event Probability
	decisions := []investment.Decision{
		{ID: "dec-1", DecisionType: investment.DecisionBuy, Confidence: 0.96},
		{ID: "dec-2", DecisionType: investment.DecisionBuy, Confidence: 0.92},
	}
	outcomesByDec := map[string]investment.PredictionOutcome{
		"dec-1": *ocMiss,
		"dec-2": *ocMiss,
	}
	confSummary := investment.ComputeDecisionConfidenceSummary(decisions, outcomesByDec)
	if confSummary.SampleCount != 2 || confSummary.FavorableActionRate != 0 || confSummary.OverconfidenceGap < 0.90 {
		t.Fatalf("expected severe OverconfidenceGap (>0.90) & FavorableActionRate=0, got %+v", confSummary)
	}
}
