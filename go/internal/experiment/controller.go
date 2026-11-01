package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/autonomy/clock"
	autonomyLearning "github.com/wade/aia/internal/autonomy/learning"
	"github.com/wade/aia/internal/domain/continuity"
	domainExp "github.com/wade/aia/internal/domain/experiment"
	"github.com/wade/aia/internal/domain/investment"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	invService "github.com/wade/aia/internal/service/investment"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type Dependencies struct {
	ExpRepo          *repository.ExperimentRepository
	InvSvc           *invService.Service
	LearningRepo     *repository.LearningRepository
	LearningEngine   *autonomyLearning.Engine
	EventHistoryRepo *repository.EventHistoryRepository
	GoalRepo         *repository.GoalRepository
	DefaultLLMDec    GenerationDecider
}

// Controller implements the AIA Generational Experiment Controller (AIA-05 §20, AIA-13 §21.4, AIA-15 §21).
// It enforces identical market snapshots, isolated NT$100,000 ledgers, frozen G1 baseline immutability,
// dual probability/confidence calibration, and non-destructive G2 -> G2.x lineage evolution.
type Controller struct {
	deps          Dependencies
	g1Decider     GenerationDecider
	customDecider map[string]GenerationDecider // keyed by generation_code or generation_id
}

func NewController(deps Dependencies) *Controller {
	return &Controller{
		deps:          deps,
		g1Decider:     NewFrozenG1RuleDecider(),
		customDecider: make(map[string]GenerationDecider),
	}
}

// RegisterGenerationDecider attaches a specific GenerationDecider to a generation code (e.g. "G2", "G2.1") or ID.
func (c *Controller) RegisterGenerationDecider(key string, dec GenerationDecider) {
	if dec != nil {
		c.customDecider[key] = dec
	}
}

type CreateDualInput struct {
	ProjectID        string
	PrimaryGoalID    string
	Name             string
	Hypothesis       string
	InitialCapital   int64
	MaxPositionPct   float64
	G2HarnessVersion string
	G2ModelID        string
	G2PromptVersion  string
	G2StrategyConfig string
	StartAsOf        *time.Time
}

type DualExperimentBundle struct {
	Experiment *domainExp.Experiment           `json:"experiment"`
	G1         *domainExp.ExperimentGeneration `json:"g1"`
	G2         *domainExp.ExperimentGeneration `json:"g2"`
}

// CreateDualExperiment initializes an Experiment with:
// - G1: Frozen Rule-Based Control Baseline (NT$100,000 isolated portfolio, is_frozen=true)
// - G2: Experimental LLM Cohort via agy (NT$100,000 isolated portfolio, versioned manifest)
func (c *Controller) CreateDualExperiment(ctx context.Context, in CreateDualInput) (*DualExperimentBundle, error) {
	if in.StartAsOf != nil && !in.StartAsOf.IsZero() {
		ctx = clock.WithAsOf(ctx, in.StartAsOf.UTC(), true)
	}
	now := clock.Now(ctx)
	if in.InitialCapital <= 0 {
		in.InitialCapital = 100000
	}
	if in.MaxPositionPct <= 0 {
		in.MaxPositionPct = 0.85
	}
	if strings.TrimSpace(in.Name) == "" {
		in.Name = "Generational Investment Decision Experiment (G1 vs G2)"
	}
	if strings.TrimSpace(in.Hypothesis) == "" {
		in.Hypothesis = "LLM-based Generation 2 achieves superior total portfolio assets and calibrated forecasts over frozen Rule-based Generation 1 under identical Taiwan market conditions."
	}
	if in.G2HarnessVersion == "" {
		in.G2HarnessVersion = "agy-1.4.12"
	}
	if in.G2ModelID == "" {
		in.G2ModelID = "gemini-default"
	}
	if in.G2PromptVersion == "" {
		in.G2PromptVersion = "inv-generational-v1"
	}
	if in.G2StrategyConfig == "" {
		in.G2StrategyConfig = `{"entry_mode":"llm_autonomous","focus_symbol":"2330"}`
	}

	expID := uuid.New().String()
	var projPtr, goalPtr *string
	if in.ProjectID != "" {
		p := in.ProjectID
		projPtr = &p
	}
	if in.PrimaryGoalID != "" {
		g := in.PrimaryGoalID
		goalPtr = &g
	}

	baselineMeta, _ := json.Marshal(map[string]any{
		"initial_capital_twd": in.InitialCapital,
		"market":              "TW",
		"control_cohort":      "G1",
		"experimental_cohort": "G2",
	})

	exp := &domainExp.Experiment{
		ID:                  expID,
		ProjectID:           projPtr,
		PrimaryGoalID:       goalPtr,
		Domain:              "investment",
		OwnerAgentID:        "investment-agent",
		Name:                in.Name,
		Hypothesis:          in.Hypothesis,
		ExperimentType:      "GENERATIONAL_COMPARISON",
		Status:              domainExp.StatusRunning,
		BaselineMetricsJSON: string(baselineMeta),
		CurrentMetricsJSON:  "{}",
		IterationCount:      0,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := c.deps.ExpRepo.CreateExperiment(ctx, exp); err != nil {
		return nil, fmt.Errorf("create experiment: %w", err)
	}

	g1ProfID := fmt.Sprintf("prof-%s-G1", expID[:8])
	if _, _, err := c.deps.InvSvc.CreateIsolatedProfile(ctx, g1ProfID, "G1 Rule-Based Control Portfolio", in.InitialCapital, in.MaxPositionPct); err != nil {
		return nil, fmt.Errorf("create G1 isolated profile: %w", err)
	}

	g2ProfID := fmt.Sprintf("prof-%s-G2", expID[:8])
	if _, _, err := c.deps.InvSvc.CreateIsolatedProfile(ctx, g2ProfID, "G2 LLM Experimental Portfolio", in.InitialCapital, in.MaxPositionPct); err != nil {
		return nil, fmt.Errorf("create G2 isolated profile: %w", err)
	}

	g1 := &domainExp.ExperimentGeneration{
		ID:                 uuid.New().String(),
		ExperimentID:       exp.ID,
		GenerationCode:     "G1",
		Role:               domainExp.RoleControlBaseline,
		EngineType:         domainExp.EngineRuleBased,
		PortfolioProfileID: g1ProfID,
		IsFrozen:           true,
		HarnessVersion:     "go-native",
		ModelID:            "deterministic-rules",
		PromptVersion:      "none",
		StrategyVersion:    "rule-v1.0-frozen",
		StrategyConfigJSON: `{"pullback_buy_pct":-3.0,"take_profit_pct":8.0,"stop_loss_pct":-6.0,"momentum_entry_pct":1.5}`,
		Status:             domainExp.GenStatusActive,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := c.deps.ExpRepo.CreateGeneration(ctx, g1); err != nil {
		return nil, fmt.Errorf("create G1 generation: %w", err)
	}

	g2 := &domainExp.ExperimentGeneration{
		ID:                 uuid.New().String(),
		ExperimentID:       exp.ID,
		GenerationCode:     "G2",
		Role:               domainExp.RoleExperimental,
		EngineType:         domainExp.EngineLLMAgy,
		PortfolioProfileID: g2ProfID,
		IsFrozen:           false,
		HarnessVersion:     in.G2HarnessVersion,
		ModelID:            in.G2ModelID,
		PromptVersion:      in.G2PromptVersion,
		StrategyVersion:    "llm-v2.0-initial",
		StrategyConfigJSON: in.G2StrategyConfig,
		Status:             domainExp.GenStatusActive,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := c.deps.ExpRepo.CreateGeneration(ctx, g2); err != nil {
		return nil, fmt.Errorf("create G2 generation: %w", err)
	}

	c.appendEvent(ctx, exp.ID, "EXPERIMENT_DUAL_CREATED", map[string]any{
		"experiment_id":   exp.ID,
		"g1_id":           g1.ID,
		"g2_id":           g2.ID,
		"initial_capital": in.InitialCapital,
	}, now)

	return &DualExperimentBundle{Experiment: exp, G1: g1, G2: g2}, nil
}

type StepInput struct {
	ExperimentID string
	Symbol       string
	AsOf         time.Time
}

type GenerationStepOutcome struct {
	GenerationCode   string                               `json:"generation_code"`
	Decision         *investment.Decision                 `json:"decision"`
	SettledProposal  *investment.TradeProposal            `json:"settled_proposal,omitempty"`
	Prediction       *investment.Prediction               `json:"prediction"`
	PostStepSnapshot *investment.PortfolioSnapshot        `json:"post_step_snapshot"`
	Forecast         *domainExp.GenerationalForecastOutput `json:"forecast"`
}

type StepResult struct {
	ExperimentID   string                  `json:"experiment_id"`
	DataSnapshotID string                  `json:"data_snapshot_id"`
	AsOf           time.Time               `json:"as_of"`
	Symbol         string                  `json:"symbol"`
	ReferencePrice float64                 `json:"reference_price"`
	Outcomes       []GenerationStepOutcome `json:"outcomes"`
}

// StepObservation advances an experiment at timestamp AsOf under strict Zero-Look-Ahead isolation:
// 1. Pins a single DataSnapshot strictly <= AsOf.
// 2. Settles due predictions for every cohort up to AsOf.
// 3. Feeds the exact same market data & transaction cost rules to G1 and active G2/G2.x cohorts.
func (c *Controller) StepObservation(ctx context.Context, in StepInput) (*StepResult, error) {
	if in.AsOf.IsZero() {
		in.AsOf = clock.Now(ctx)
	}
	asOf := in.AsOf.UTC()
	pitCtx := clock.WithAsOf(ctx, asOf, true)

	if strings.TrimSpace(in.Symbol) == "" {
		in.Symbol = "2330"
	}

	exp, err := c.deps.ExpRepo.GetExperiment(pitCtx, in.ExperimentID)
	if err != nil {
		return nil, fmt.Errorf("load experiment: %w", err)
	}

	inst, err := c.deps.InvSvc.Repo().GetInstrumentBySymbol(pitCtx, "", in.Symbol)
	if err != nil || inst == nil {
		return nil, fmt.Errorf("instrument %s not found: %w", in.Symbol, err)
	}

	bars, err := c.deps.InvSvc.Repo().ListHistoricalMarketData(pitCtx, inst.ID, 5)
	if err != nil || len(bars) == 0 {
		return nil, fmt.Errorf("no market bar available <= %s for %s", asOf.Format(time.RFC3339), in.Symbol)
	}

	var curBar investment.MarketBarValue
	if err := json.Unmarshal([]byte(bars[0].ValueJSON), &curBar); err != nil || curBar.Close <= 0 {
		return nil, fmt.Errorf("invalid market bar JSON at %s", asOf.Format(time.RFC3339))
	}
	var priorBar *investment.MarketBarValue
	if len(bars) > 1 {
		var pb investment.MarketBarValue
		if json.Unmarshal([]byte(bars[1].ValueJSON), &pb) == nil && pb.Close > 0 {
			priorBar = &pb
		}
	}

	disclosures, _ := c.deps.InvSvc.Repo().ListRawDocuments(pitCtx, inst.ID, 10)

	// Record shared Point-in-Time DataSnapshot so all cohorts anchor to the exact same cutoff
	dataSnap := &investment.DataSnapshot{
		ID:                    uuid.New().String(),
		SnapshotType:          "GENERATIONAL_OBSERVATION",
		DataCutoffAt:          asOf,
		InstrumentScopeJSON:   fmt.Sprintf(`[%q]`, in.Symbol),
		SourceVersionsJSON:    fmt.Sprintf(`{"latest_bar_id":%q,"effective_at":%q,"fetched_at":%q}`, bars[0].ID, bars[0].EffectiveAt.Format(time.RFC3339), bars[0].FetchedAt.Format(time.RFC3339)),
		ValidationSummaryJSON: `{"zero_look_ahead":true}`,
		CreatedAt:             asOf,
	}
	_ = c.deps.InvSvc.Repo().SaveDataSnapshot(pitCtx, dataSnap, []investment.SnapshotItem{
		{
			ID:          uuid.New().String(),
			SnapshotID:  dataSnap.ID,
			ItemType:    "DAILY_BAR",
			ReferenceID: bars[0].ID,
			PayloadJSON: bars[0].ValueJSON,
			CreatedAt:   asOf,
		},
	})

	goalSummary := "Maximize Total Portfolio Assets"
	if exp.PrimaryGoalID != nil && c.deps.GoalRepo != nil {
		if g, gErr := c.deps.GoalRepo.Get(pitCtx, *exp.PrimaryGoalID); gErr == nil && g != nil {
			goalSummary = g.Name
		}
	}

	var verifiedLearnings []domainLearning.LearningRecord
	var suppressed []string
	if c.deps.LearningRepo != nil {
		projFilter := ""
		if exp.ProjectID != nil {
			projFilter = *exp.ProjectID
		}
		effRecs, _ := c.deps.LearningRepo.QueryLearningRecords(pitCtx, repository.LearningFilter{
			ProjectID: projFilter,
			Domain:    "investment",
			Verdict:   domainLearning.VerdictEffective,
			Limit:     10,
		})
		for _, r := range effRecs {
			if r != nil {
				verifiedLearnings = append(verifiedLearnings, *r)
			}
		}
		for _, badVerdict := range []domainLearning.EffectivenessVerdict{domainLearning.VerdictRegressed, domainLearning.VerdictIneffective} {
			badRecs, _ := c.deps.LearningRepo.QueryLearningRecords(pitCtx, repository.LearningFilter{
				ProjectID: projFilter,
				Domain:    "investment",
				Verdict:   badVerdict,
				Limit:     10,
			})
			for _, r := range badRecs {
				if r != nil && r.Title != "" {
					suppressed = append(suppressed, r.Title)
				}
			}
		}
	}

	generations, err := c.deps.ExpRepo.ListGenerations(pitCtx, exp.ID)
	if err != nil {
		return nil, fmt.Errorf("list generations: %w", err)
	}

	var stepOutcomes []GenerationStepOutcome
	for i := range generations {
		gen := &generations[i]
		if gen.Status != domainExp.GenStatusActive && gen.Status != domainExp.GenStatusPromoted {
			continue
		}

		// Settle any matured predictions for this cohort up to AsOf first
		_, _ = c.deps.InvSvc.EvaluatePredictionsForProfile(pitCtx, gen.PortfolioProfileID)

		prof, snap, positions, err := c.deps.InvSvc.GetSnapshotForProfile(pitCtx, gen.PortfolioProfileID)
		if err != nil {
			return nil, fmt.Errorf("load snapshot for %s: %w", gen.GenerationCode, err)
		}
		priorDecs, _ := c.deps.InvSvc.Repo().ListDecisions(pitCtx, gen.PortfolioProfileID, 10)

		obs := GenerationObservationContext{
			AsOf:               asOf,
			DataSnapshotID:     dataSnap.ID,
			Symbol:             in.Symbol,
			Instrument:         *inst,
			CurrentBar:         curBar,
			PriorBar:           priorBar,
			RecentDisclosures:  disclosures,
			PortfolioProfile:   *prof,
			PortfolioSnapshot:  *snap,
			Positions:          positions,
			PriorDecisions:     priorDecs,
			VerifiedLearnings:  verifiedLearnings,
			SuppressedPatterns: suppressed,
			GoalSummary:        goalSummary,
		}

		decider, err := c.resolveDecider(gen)
		if err != nil {
			return nil, err
		}
		forecast, err := decider.Decide(pitCtx, gen, obs)
		if err != nil {
			return nil, fmt.Errorf("generation %s decide failed: %w", gen.GenerationCode, err)
		}
		forecast, err = validateForecastOutput(forecast, in.Symbol)
		if err != nil {
			return nil, fmt.Errorf("generation %s invalid forecast: %w", gen.GenerationCode, err)
		}

		sourceLabel := "GENERATION_LLM_AGY"
		enforceMinConf := true
		if gen.Role == domainExp.RoleControlBaseline {
			sourceLabel = "GENERATION_G1_RULE"
			enforceMinConf = false
		}

		horizonStr := fmt.Sprintf("%dD", forecast.HorizonDays)
		dec, proposal, err := c.deps.InvSvc.ProcessDecisionForProfile(pitCtx, gen.PortfolioProfileID, sourceLabel, enforceMinConf, investment.AgyDecisionInput{
			Symbol:            in.Symbol,
			Exchange:          inst.Exchange,
			Action:            forecast.Action,
			TargetAmount:      forecast.TargetAmount,
			Confidence:        forecast.DecisionConfidence,
			Reasoning:         forecast.Reasoning,
			Risks:             forecast.PrimaryRisks,
			InvalidationRules: forecast.InvalidationConditions,
			ExpectedHorizon:   horizonStr,
			StrategyVersion:   gen.StrategyVersion,
		}, curBar.Close)
		if err != nil {
			return nil, fmt.Errorf("process decision for %s: %w", gen.GenerationCode, err)
		}

		var postSnap *investment.PortfolioSnapshot
		if proposal != nil {
			settledSnap, err := c.deps.InvSvc.ExecuteVirtualProposal(pitCtx, proposal.ID)
			if err != nil {
				return nil, fmt.Errorf("settle virtual proposal for %s: %w", gen.GenerationCode, err)
			}
			proposal.Status = investment.ProposalExecuted
			postSnap = settledSnap
		} else {
			// Mark held positions to market so equity curve & drawdown reflect AsOf price identically
			mtmSnap, err := c.deps.InvSvc.Repo().MarkToMarketProfile(pitCtx, gen.PortfolioProfileID, map[string]float64{
				inst.ID: curBar.Close,
			})
			if err != nil {
				postSnap = snap
			} else {
				postSnap = mtmSnap
			}
		}

		// Record event forecast (UpProbability + ExpectedReturnPct + Range) strictly distinct from Decision.Confidence
		expDir := "UP"
		prob := forecast.UpProbability
		if forecast.UpProbability < 0.48 {
			expDir = "DOWN"
			prob = 1.0 - forecast.UpProbability
		} else if forecast.UpProbability >= 0.48 && forecast.UpProbability <= 0.52 && math.Abs(forecast.ExpectedReturnPct) < 0.2 {
			expDir = "FLAT"
		}

		pred := &investment.Prediction{
			ID:                   uuid.New().String(),
			DecisionID:           dec.ID,
			InstrumentID:         inst.ID,
			PredictionType:       "DIRECTION_AND_RANGE",
			Target:               "PRICE_CHANGE_PCT",
			ExpectedDirection:    expDir,
			Probability:          prob,
			TargetValue:          curBar.Close * (1.0 + forecast.ExpectedReturnPct/100.0),
			ExpectedReturnPct:    forecast.ExpectedReturnPct,
			ExpectedRangeLowPct:  forecast.ExpectedRangeLowPct,
			ExpectedRangeHighPct: forecast.ExpectedRangeHighPct,
			BasePrice:            curBar.Close,
			EvaluationHorizon:    horizonStr,
			DueAt:                asOf.Add(time.Duration(forecast.HorizonDays) * 24 * time.Hour),
			StrategyVersion:      gen.StrategyVersion,
			Status:               investment.PredictionStatus(investment.PredictionPending),
			CreatedAt:            asOf,
		}
		if err := c.deps.InvSvc.Repo().SavePrediction(pitCtx, pred); err != nil {
			return nil, fmt.Errorf("save prediction for %s: %w", gen.GenerationCode, err)
		}

		stepOutcomes = append(stepOutcomes, GenerationStepOutcome{
			GenerationCode:   gen.GenerationCode,
			Decision:         dec,
			SettledProposal:  proposal,
			Prediction:       pred,
			PostStepSnapshot: postSnap,
			Forecast:         forecast,
		})
	}

	summaryMetrics, _ := json.Marshal(map[string]any{
		"last_step_as_of": asOf.Format(time.RFC3339),
		"symbol":          in.Symbol,
		"close_price":     curBar.Close,
		"cohorts_stepped": len(stepOutcomes),
	})
	_ = c.deps.ExpRepo.IncrementIterationAndMetrics(pitCtx, exp.ID, string(summaryMetrics))

	return &StepResult{
		ExperimentID:   exp.ID,
		DataSnapshotID: dataSnap.ID,
		AsOf:           asOf,
		Symbol:         in.Symbol,
		ReferencePrice: curBar.Close,
		Outcomes:       stepOutcomes,
	}, nil
}

func (c *Controller) resolveDecider(gen *domainExp.ExperimentGeneration) (GenerationDecider, error) {
	if dec, ok := c.customDecider[gen.ID]; ok {
		return dec, nil
	}
	if dec, ok := c.customDecider[gen.GenerationCode]; ok {
		return dec, nil
	}
	if gen.Role == domainExp.RoleControlBaseline || gen.EngineType == domainExp.EngineRuleBased {
		return c.g1Decider, nil
	}
	if c.deps.DefaultLLMDec != nil {
		return c.deps.DefaultLLMDec, nil
	}
	return nil, fmt.Errorf("no GenerationDecider configured for experimental generation %s", gen.GenerationCode)
}

// CompareGenerations evaluates Track A (Portfolio Performance + Dual Calibration) and Track B (Autonomy Capability)
// between baselineCode (typically "G1") and candidateCode ("G2", "G2.1", ...) as of AsOf, and classifies into the 5 scientific regimes.
func (c *Controller) CompareGenerations(ctx context.Context, experimentID string, baselineCode string, candidateCode string, asOf time.Time) (*domainExp.ExperimentComparison, error) {
	if asOf.IsZero() {
		asOf = clock.Now(ctx)
	}
	asOf = asOf.UTC()
	pitCtx := clock.WithAsOf(ctx, asOf, true)

	exp, err := c.deps.ExpRepo.GetExperiment(pitCtx, experimentID)
	if err != nil {
		return nil, err
	}
	if baselineCode == "" {
		baselineCode = "G1"
	}
	if candidateCode == "" {
		candidateCode = "G2"
	}

	baseGen, err := c.deps.ExpRepo.GetGenerationByCode(pitCtx, experimentID, baselineCode)
	if err != nil {
		return nil, fmt.Errorf("load baseline generation %s: %w", baselineCode, err)
	}
	candGen, err := c.deps.ExpRepo.GetGenerationByCode(pitCtx, experimentID, candidateCode)
	if err != nil {
		return nil, fmt.Errorf("load candidate generation %s: %w", candidateCode, err)
	}

	// Settle any matured predictions up to AsOf before computing calibration
	_, _ = c.deps.InvSvc.EvaluatePredictionsForProfile(pitCtx, baseGen.PortfolioProfileID)
	_, _ = c.deps.InvSvc.EvaluatePredictionsForProfile(pitCtx, candGen.PortfolioProfileID)

	baseA, baseB, err := c.computeCohortMetrics(pitCtx, exp, baseGen)
	if err != nil {
		return nil, err
	}
	candA, candB, err := c.computeCohortMetrics(pitCtx, exp, candGen)
	if err != nil {
		return nil, err
	}

	baseComparisonTrackA := baseA
	if candGen.ParentGenerationID != nil && candGen.CreatedAt.After(baseGen.CreatedAt) {
		baseComparisonTrackA.CumulativeReturnPct = math.Round(c.windowReturnPct(pitCtx, baseGen.PortfolioProfileID, candGen.CreatedAt, asOf)*100) / 100
	}

	verdict, summary := classifyRegime(baseComparisonTrackA, candA, candB)

	perfJSON, _ := json.Marshal(map[string]any{
		"baseline":                      baseA,
		"baseline_matched_window_pct":   baseComparisonTrackA.CumulativeReturnPct,
		"candidate":                     candA,
		"excess_return_pct":             math.Round((candA.CumulativeReturnPct-baseComparisonTrackA.CumulativeReturnPct)*100) / 100,
		"excess_assets_twd":             candA.TotalAssets - baseA.TotalAssets,
		"brier_delta":                   math.Round((candA.EventCalibration.AverageBrierScore-baseA.EventCalibration.AverageBrierScore)*10000) / 10000,
		"log_loss_delta":                math.Round((candA.EventCalibration.AverageLogLoss-baseA.EventCalibration.AverageLogLoss)*10000) / 10000,
	})
	autoJSON, _ := json.Marshal(map[string]any{
		"baseline":  baseB,
		"candidate": candB,
	})

	cmp := &domainExp.ExperimentComparison{
		ID:                        uuid.New().String(),
		ExperimentID:              experimentID,
		BaselineGenerationID:      baseGen.ID,
		CandidateGenerationID:     candGen.ID,
		AsOf:                      asOf,
		SampleCount:               candB.ObservationDecisions,
		PerformanceComparisonJSON: string(perfJSON),
		AutonomyComparisonJSON:    string(autoJSON),
		RegimeVerdict:             verdict,
		Summary:                   summary,
		CreatedAt:                 asOf,
		BaselineTrackA:            baseA,
		CandidateTrackA:           candA,
		BaselineTrackB:            baseB,
		CandidateTrackB:           candB,
	}
	if err := c.deps.ExpRepo.SaveComparison(pitCtx, cmp); err != nil {
		return nil, fmt.Errorf("save comparison: %w", err)
	}
	return cmp, nil
}

func (c *Controller) computeCohortMetrics(ctx context.Context, exp *domainExp.Experiment, gen *domainExp.ExperimentGeneration) (domainExp.TrackAPerformance, domainExp.TrackBAutonomy, error) {
	var trackA domainExp.TrackAPerformance
	var trackB domainExp.TrackBAutonomy

	prof, latestSnap, positions, err := c.deps.InvSvc.GetSnapshotForProfile(ctx, gen.PortfolioProfileID)
	if err != nil {
		return trackA, trackB, err
	}
	snaps, err := c.deps.InvSvc.Repo().ListSnapshotsForProfile(ctx, gen.PortfolioProfileID)
	if err != nil {
		return trackA, trackB, err
	}

	initCap := prof.CapitalLimit
	if len(snaps) > 0 && snaps[0].TotalPortfolioValue > 0 {
		initCap = snaps[0].TotalPortfolioValue
	}

	// Max Drawdown across equity curve
	var peak int64 = initCap
	var maxDD float64
	for _, s := range snaps {
		if s.TotalPortfolioValue > peak {
			peak = s.TotalPortfolioValue
		}
		if peak > 0 {
			dd := float64(peak-s.TotalPortfolioValue) / float64(peak) * 100.0
			if dd > maxDD {
				maxDD = dd
			}
		}
	}

	var holdingsVal int64
	var unrealized float64
	for _, pos := range positions {
		holdingsVal += pos.MarketValue
		unrealized += (pos.LastKnownPrice - pos.AverageCost) * float64(pos.Quantity)
	}
	unrealizedInt := int64(math.Round(unrealized))

	// Executed proposals for transaction costs & trade count
	proposals, _ := c.deps.InvSvc.Repo().ListTradeProposals(ctx, gen.PortfolioProfileID, investment.ProposalExecuted, 500)
	var totalFeesAndTax int64
	for _, pr := range proposals {
		var meta struct {
			BrokerFee int64 `json:"broker_fee"`
			Tax       int64 `json:"transaction_tax"`
		}
		if json.Unmarshal([]byte(pr.ProposalJSON), &meta) == nil {
			totalFeesAndTax += meta.BrokerFee + meta.Tax
		}
	}

	totalNetGain := latestSnap.TotalPortfolioValue - initCap
	realizedInt := totalNetGain - unrealizedInt

	cumReturnPct := 0.0
	if initCap > 0 {
		cumReturnPct = float64(latestSnap.TotalPortfolioValue-initCap) / float64(initCap) * 100.0
	}
	cashPct := 0.0
	if latestSnap.TotalPortfolioValue > 0 {
		cashPct = float64(latestSnap.AvailableCash) / float64(latestSnap.TotalPortfolioValue) * 100.0
	}

	outcomes, outcomesByDec, _ := c.deps.InvSvc.Repo().ListPredictionOutcomesForProfile(ctx, gen.PortfolioProfileID, 500)
	decs, _ := c.deps.InvSvc.Repo().ListDecisions(ctx, gen.PortfolioProfileID, 500)

	calMetrics := investment.ComputeCalibrationSummary(outcomes)
	confMetrics := investment.ComputeDecisionConfidenceSummary(decs, outcomesByDec)

	trackA = domainExp.TrackAPerformance{
		GenerationCode:        gen.GenerationCode,
		PortfolioProfileID:    gen.PortfolioProfileID,
		InitialCapital:        initCap,
		TotalAssets:           latestSnap.TotalPortfolioValue,
		CashBalance:           latestSnap.AvailableCash,
		HoldingsValue:         holdingsVal,
		CashAllocationPct:     math.Round(cashPct*100) / 100,
		CumulativeReturnPct:   math.Round(cumReturnPct*100) / 100,
		RealizedPnL:           realizedInt,
		UnrealizedPnL:         unrealizedInt,
		MaxDrawdownPct:        math.Round(maxDD*100) / 100,
		TotalTransactionCosts: totalFeesAndTax,
		ExecutedTradeCount:    len(proposals),
		EventCalibration:      calMetrics,
		DecisionConfidence:    confMetrics,
	}

	// Track B: Autonomy metrics
	holdCount := 0
	activeCount := 0
	invalRulesCount := 0
	invalTriggered := 0
	for _, d := range decs {
		if d.DecisionType == investment.DecisionHold {
			holdCount++
		} else {
			activeCount++
		}
		if d.InvalidationConditionsJSON != "" && d.InvalidationConditionsJSON != "[]" && d.InvalidationConditionsJSON != "null" {
			invalRulesCount++
		}
		if strings.Contains(strings.ToLower(d.ReasoningJSON), "invalidation") || strings.Contains(strings.ToLower(d.ReasoningJSON), "reassess") || strings.Contains(strings.ToLower(d.ReasoningJSON), "stop") {
			invalTriggered++
		}
	}
	holdRate := 0.0
	if len(decs) > 0 {
		holdRate = float64(holdCount) / float64(len(decs))
	}

	depth := 0
	curParent := gen.ParentGenerationID
	for curParent != nil && *curParent != "" {
		depth++
		parentGen, pErr := c.deps.ExpRepo.GetGeneration(ctx, *curParent)
		if pErr != nil || parentGen == nil {
			break
		}
		curParent = parentGen.ParentGenerationID
	}

	hasVerifiedParent := false
	if gen.SourceLearningID != nil && c.deps.LearningRepo != nil {
		if lr, lErr := c.deps.LearningRepo.GetLearningRecord(ctx, *gen.SourceLearningID); lErr == nil && lr != nil {
			if lr.EffectivenessVerdict == domainLearning.VerdictEffective {
				hasVerifiedParent = true
			}
		}
	}

	verifiedCount := 0
	generalizedCount := 0
	if c.deps.LearningRepo != nil {
		projFilter := ""
		if exp.ProjectID != nil {
			projFilter = *exp.ProjectID
		}
		vList, _ := c.deps.LearningRepo.QueryLearningRecords(ctx, repository.LearningFilter{
			ProjectID: projFilter,
			Domain:    "investment",
			Verdict:   domainLearning.VerdictEffective,
			Limit:     100,
		})
		for _, v := range vList {
			if v == nil {
				continue
			}
			verifiedCount++
			if v.LearningLevel == domainLearning.LevelL3Generalized {
				generalizedCount++
			}
		}
	}

	trackB = domainExp.TrackBAutonomy{
		GenerationCode:              gen.GenerationCode,
		ObservationDecisions:        len(decs),
		ActiveTradeDecisions:        activeCount,
		RationalHoldCount:           holdCount,
		RationalHoldRate:            math.Round(holdRate*10000) / 10000,
		InvalidationRulesSpecified:  invalRulesCount,
		InvalidationTriggeredCount:  invalTriggered,
		LineageDepth:                depth,
		HasVerifiedParentCorrection: hasVerifiedParent,
		VerifiedEffectiveLearnings:  verifiedCount,
		GeneralizedLearnings:        generalizedCount,
	}

	return trackA, trackB, nil
}

func classifyRegime(baseA, candA domainExp.TrackAPerformance, candB domainExp.TrackBAutonomy) (domainExp.RegimeVerdict, string) {
	if candB.ObservationDecisions == 0 {
		return domainExp.RegimeInconclusiveLowSample, "Insufficient observation decisions recorded to compare generations."
	}

	excessReturn := candA.CumulativeReturnPct - baseA.CumulativeReturnPct
	poorEventCalibration := candA.EventCalibration.TotalEvaluated > 0 && (candA.EventCalibration.AverageBrierScore > 0.30 ||
		(baseA.EventCalibration.TotalEvaluated > 0 && candA.EventCalibration.AverageBrierScore > baseA.EventCalibration.AverageBrierScore+0.08))
	severeOverconfidence := candA.DecisionConfidence.SampleCount > 0 && candA.DecisionConfidence.OverconfidenceGap > 0.35

	if excessReturn > 0.25 {
		if poorEventCalibration || severeOverconfidence {
			return domainExp.RegimeG2UnreliableRiskTaking, fmt.Sprintf(
				"%s outperformed %s by +%.2f%% total return, but calibration is poor (Brier=%.4f, LogLoss=%.4f, OverconfGap=%.2f): excess return may stem from higher risk or luck rather than calibrated skill.",
				candA.GenerationCode, baseA.GenerationCode, excessReturn,
				candA.EventCalibration.AverageBrierScore, candA.EventCalibration.AverageLogLoss, candA.DecisionConfidence.OverconfidenceGap,
			)
		}
		return domainExp.RegimeG2ClearAdvantage, fmt.Sprintf(
			"%s outperformed %s by +%.2f%% total return with sound calibration (Brier=%.4f, LogLoss=%.4f).",
			candA.GenerationCode, baseA.GenerationCode, excessReturn,
			candA.EventCalibration.AverageBrierScore, candA.EventCalibration.AverageLogLoss,
		)
	}

	if excessReturn < -0.25 {
		if candB.LineageDepth > 0 && (candB.HasVerifiedParentCorrection || candB.VerifiedEffectiveLearnings > 0) {
			return domainExp.RegimeG2LearningInProgress, fmt.Sprintf(
				"%s currently trails %s by %.2f%%, but lineage correction is verified effective; self-correction capability progressed while long-term return remains under validation.",
				candA.GenerationCode, baseA.GenerationCode, excessReturn,
			)
		}
		return domainExp.RegimeG1RulesSuperior, fmt.Sprintf(
			"Frozen rule baseline %s outperformed %s by +%.2f%% (LLM does not automatically beat simple disciplined rules).",
			baseA.GenerationCode, candA.GenerationCode, -excessReturn,
		)
	}

	// Parity band [-0.25%, +0.25%]
	if candB.InvalidationTriggeredCount > 0 || candB.HasVerifiedParentCorrection || candB.VerifiedEffectiveLearnings > 0 {
		return domainExp.RegimeParityAdaptive, fmt.Sprintf(
			"%s and %s achieved comparable returns (diff %.2f%%), with %s demonstrating active regime adaptation and invalidation handling.",
			candA.GenerationCode, baseA.GenerationCode, excessReturn, candA.GenerationCode,
		)
	}
	return domainExp.RegimeG1RulesSuperior, fmt.Sprintf(
		"%s and %s returns are tied (diff %.2f%%) without verified adaptive advantage; simpler frozen G1 baseline holds priority.",
		candA.GenerationCode, baseA.GenerationCode, excessReturn,
	)
}

type SpawnGenerationInput struct {
	ExperimentID       string
	ParentCode         string // e.g. "G2" -> spawns "G2.1" (NEVER "G1"!)
	NewGenerationCode  string // e.g. "G2.1"
	HarnessVersion     string
	ModelID            string
	PromptVersion      string
	StrategyVersion    string
	StrategyConfigJSON string
	SourceCorrectionID string
	SourceLearningID   string
	InitialCapital     int64
	MaxPositionPct     float64
	SupersedeParent    bool
	AsOf               *time.Time
}

// SpawnGenerationFromCorrection creates a versioned child generation (e.g. G2.1 from G2) after RCA & Correction Proposal,
// strictly preserving both frozen G1 and historical G2 untouched (AIA-15 §21 / dual.md §5).
func (c *Controller) SpawnGenerationFromCorrection(ctx context.Context, in SpawnGenerationInput) (*domainExp.ExperimentGeneration, error) {
	if in.AsOf != nil && !in.AsOf.IsZero() {
		ctx = clock.WithAsOf(ctx, in.AsOf.UTC(), true)
	}
	now := clock.Now(ctx)

	parent, err := c.deps.ExpRepo.GetGenerationByCode(ctx, in.ExperimentID, in.ParentCode)
	if err != nil {
		return nil, fmt.Errorf("load parent generation %s: %w", in.ParentCode, err)
	}
	if parent.IsFrozen || parent.Role == domainExp.RoleControlBaseline {
		return nil, fmt.Errorf("governance violation: frozen baseline generation %s cannot be modified or evolved", parent.GenerationCode)
	}
	if strings.TrimSpace(in.NewGenerationCode) == "" || in.NewGenerationCode == parent.GenerationCode || in.NewGenerationCode == "G1" {
		return nil, fmt.Errorf("invalid child generation code %q: must create a distinct versioned generation (e.g. G2.1) without overwriting %s", in.NewGenerationCode, parent.GenerationCode)
	}

	if in.InitialCapital <= 0 {
		parentProf, _, _, pErr := c.deps.InvSvc.GetSnapshotForProfile(ctx, parent.PortfolioProfileID)
		if pErr == nil && parentProf != nil {
			in.InitialCapital = parentProf.CapitalLimit
		} else {
			in.InitialCapital = 100000
		}
	}
	if in.MaxPositionPct <= 0 {
		in.MaxPositionPct = 0.85
	}
	if in.HarnessVersion == "" {
		in.HarnessVersion = parent.HarnessVersion
	}
	if in.ModelID == "" {
		in.ModelID = parent.ModelID
	}
	if in.PromptVersion == "" {
		in.PromptVersion = parent.PromptVersion
	}
	if in.StrategyVersion == "" {
		in.StrategyVersion = parent.StrategyVersion + "-rev1"
	}
	if in.StrategyConfigJSON == "" {
		in.StrategyConfigJSON = parent.StrategyConfigJSON
	}

	profID := fmt.Sprintf("prof-%s-%s", in.ExperimentID[:8], strings.ReplaceAll(in.NewGenerationCode, ".", "_"))
	if _, _, err := c.deps.InvSvc.CreateIsolatedProfile(ctx, profID, fmt.Sprintf("%s Evolved LLM Portfolio", in.NewGenerationCode), in.InitialCapital, in.MaxPositionPct); err != nil {
		return nil, fmt.Errorf("create isolated profile for %s: %w", in.NewGenerationCode, err)
	}

	parentID := parent.ID
	var corrPtr, learnPtr *string
	if in.SourceCorrectionID != "" {
		v := in.SourceCorrectionID
		corrPtr = &v
	}
	if in.SourceLearningID != "" {
		v := in.SourceLearningID
		learnPtr = &v
	}

	child := &domainExp.ExperimentGeneration{
		ID:                 uuid.New().String(),
		ExperimentID:       in.ExperimentID,
		GenerationCode:     in.NewGenerationCode,
		ParentGenerationID: &parentID,
		Role:               domainExp.RoleExperimental,
		EngineType:         parent.EngineType,
		PortfolioProfileID: profID,
		IsFrozen:           false,
		HarnessVersion:     in.HarnessVersion,
		ModelID:            in.ModelID,
		PromptVersion:      in.PromptVersion,
		StrategyVersion:    in.StrategyVersion,
		StrategyConfigJSON: in.StrategyConfigJSON,
		SourceCorrectionID: corrPtr,
		SourceLearningID:   learnPtr,
		Status:             domainExp.GenStatusActive,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := c.deps.ExpRepo.CreateGeneration(ctx, child); err != nil {
		return nil, fmt.Errorf("create evolved generation %s: %w", in.NewGenerationCode, err)
	}
	if in.SupersedeParent {
		_ = c.deps.ExpRepo.UpdateGenerationStatus(ctx, parent.ID, domainExp.GenStatusRetired)
	}

	c.appendEvent(ctx, in.ExperimentID, "EXPERIMENT_GENERATION_SPAWNED", map[string]any{
		"experiment_id":      in.ExperimentID,
		"parent_generation":  parent.GenerationCode,
		"child_generation":   child.GenerationCode,
		"source_correction":  in.SourceCorrectionID,
		"source_learning_id": in.SourceLearningID,
		"strategy_version":   child.StrategyVersion,
		"superseded_parent":  in.SupersedeParent,
	}, now)

	return child, nil
}

type VerifyGenerationInput struct {
	ExperimentID            string
	CandidateCode           string // e.g. "G2.1"
	AsOf                    time.Time
	ParentBaselineReturnPct *float64 // Optional explicit override; auto-computed from parent ledger if nil
	G1BaselineReturnPct     *float64 // Optional explicit override; auto-computed from G1 ledger if nil
	MinExcessImprovement    float64  // Default 0.25 percentage points
	MinSamples              int      // Default 2 observation decisions
}

// VerifyGenerationWithG1Control bridges AIA-15 Generational Experiment Comparison directly into AIA-11 Phase 4 LearningEngine:
// Frozen G1 serves as the live CONTROL_BENCHMARK (Difference-in-Differences), determining whether G2.x is PROMOTED (L2_VERIFIED) or ROLLED_BACK (REGRESSED / INEFFECTIVE).
func (c *Controller) VerifyGenerationWithG1Control(ctx context.Context, in VerifyGenerationInput) (*domainExp.ExperimentComparison, *domainLearning.LearningRecord, *domainExp.ExperimentGeneration, error) {
	if in.AsOf.IsZero() {
		in.AsOf = clock.Now(ctx)
	}
	asOf := in.AsOf.UTC()
	pitCtx := clock.WithAsOf(ctx, asOf, true)

	cmp, err := c.CompareGenerations(pitCtx, in.ExperimentID, "G1", in.CandidateCode, asOf)
	if err != nil {
		return nil, nil, nil, err
	}
	baseGen, err := c.deps.ExpRepo.GetGenerationByCode(pitCtx, in.ExperimentID, "G1")
	if err != nil {
		return cmp, nil, nil, err
	}
	candGen, err := c.deps.ExpRepo.GetGenerationByCode(pitCtx, in.ExperimentID, in.CandidateCode)
	if err != nil {
		return cmp, nil, nil, err
	}
	if candGen.SourceLearningID == nil || c.deps.LearningEngine == nil {
		return cmp, nil, candGen, nil
	}

	postCandReturn := cmp.CandidateTrackA.CumulativeReturnPct
	postG1Return := c.windowReturnPct(pitCtx, baseGen.PortfolioProfileID, candGen.CreatedAt, asOf)

	var parentBaseReturn, g1BaseReturn float64
	if candGen.ParentGenerationID != nil && *candGen.ParentGenerationID != "" {
		if parentGen, pErr := c.deps.ExpRepo.GetGeneration(pitCtx, *candGen.ParentGenerationID); pErr == nil && parentGen != nil {
			parentBaseReturn = c.windowReturnPct(pitCtx, parentGen.PortfolioProfileID, parentGen.CreatedAt, candGen.CreatedAt)
			g1BaseReturn = c.windowReturnPct(pitCtx, baseGen.PortfolioProfileID, parentGen.CreatedAt, candGen.CreatedAt)
		}
	}
	if in.ParentBaselineReturnPct != nil {
		parentBaseReturn = *in.ParentBaselineReturnPct
	}
	if in.G1BaselineReturnPct != nil {
		g1BaseReturn = *in.G1BaselineReturnPct
	}

	minExcess := in.MinExcessImprovement
	if minExcess <= 0 {
		minExcess = 0.25
	}
	minSamples := in.MinSamples
	if minSamples <= 0 {
		minSamples = 2
	}

	// Check regression: severe drawdown blowup (> 20% AND > 2x G1 drawdown), extreme Brier degradation (> 0.35), or severe overconfidence (> 0.45)
	regressed := false
	regNotes := "No portfolio or calibration regression vs G1 baseline"
	if cmp.CandidateTrackA.MaxDrawdownPct > 20.0 && cmp.CandidateTrackA.MaxDrawdownPct > cmp.BaselineTrackA.MaxDrawdownPct*2.0 {
		regressed = true
		regNotes = fmt.Sprintf("Severe drawdown regression: %.2f%% vs G1 %.2f%%", cmp.CandidateTrackA.MaxDrawdownPct, cmp.BaselineTrackA.MaxDrawdownPct)
	} else if cmp.CandidateTrackA.EventCalibration.TotalEvaluated > 0 && cmp.CandidateTrackA.EventCalibration.AverageBrierScore > 0.35 {
		regressed = true
		regNotes = fmt.Sprintf("Severe forecast calibration regression: Brier %.4f > 0.35", cmp.CandidateTrackA.EventCalibration.AverageBrierScore)
	} else if cmp.CandidateTrackA.DecisionConfidence.SampleCount > 0 && cmp.CandidateTrackA.DecisionConfidence.OverconfidenceGap > 0.45 {
		regressed = true
		regNotes = fmt.Sprintf("Severe decision overconfidence regression: gap %.2f > 0.45", cmp.CandidateTrackA.DecisionConfidence.OverconfidenceGap)
	}

	sampleCount := cmp.CandidateTrackB.ObservationDecisions
	if cmp.CandidateTrackA.EventCalibration.TotalEvaluated > sampleCount {
		sampleCount = cmp.CandidateTrackA.EventCalibration.TotalEvaluated
	}

	lr, err := c.deps.LearningEngine.VerifyCorrection(pitCtx, autonomyLearning.VerificationInput{
		LearningRecordID:      *candGen.SourceLearningID,
		ComparisonMethod:      domainLearning.CompControlBenchmark,
		BaselineMetric:        parentBaseReturn,
		PostMetric:            postCandReturn,
		LowerIsBetter:         false,
		HasControl:            true,
		ControlBaselineMetric: g1BaseReturn,
		ControlPostMetric:     postG1Return,
		MinExcessImprovement:  minExcess,
		SampleSize:            sampleCount,
		MinSamples:            minSamples,
		RegressionPassed:      !regressed,
		RegressionDetails: map[string]any{
			"notes":               regNotes,
			"candidate_max_dd":    cmp.CandidateTrackA.MaxDrawdownPct,
			"g1_max_dd":           cmp.BaselineTrackA.MaxDrawdownPct,
			"candidate_brier":     cmp.CandidateTrackA.EventCalibration.AverageBrierScore,
			"candidate_log_loss":  cmp.CandidateTrackA.EventCalibration.AverageLogLoss,
			"regime_verdict":      cmp.RegimeVerdict,
			"comparison_summary":  cmp.Summary,
		},
		ProposedConfidence: 0.82,
	})
	if err != nil {
		return cmp, nil, candGen, err
	}

	newStatus := candGen.Status
	switch lr.EffectivenessVerdict {
	case domainLearning.VerdictEffective:
		newStatus = domainExp.GenStatusPromoted
	case domainLearning.VerdictRegressed, domainLearning.VerdictIneffective:
		newStatus = domainExp.GenStatusRolledBack
	}
	if newStatus != candGen.Status {
		_ = c.deps.ExpRepo.UpdateGenerationStatus(pitCtx, candGen.ID, newStatus)
		candGen.Status = newStatus
	}

	// Re-run comparison if promoted so HasVerifiedParentCorrection reflects immediately
	if newStatus == domainExp.GenStatusPromoted {
		cmp, _ = c.CompareGenerations(pitCtx, in.ExperimentID, "G1", in.CandidateCode, asOf)
	}

	return cmp, lr, candGen, nil
}

// windowReturnPct computes percentage total-asset return for a profile strictly within [startAt, endAt].
func (c *Controller) windowReturnPct(ctx context.Context, profileID string, startAt time.Time, endAt time.Time) float64 {
	pitCtx := clock.WithAsOf(ctx, endAt.UTC(), true)
	snaps, err := c.deps.InvSvc.Repo().ListSnapshotsForProfile(pitCtx, profileID)
	if err != nil || len(snaps) == 0 {
		return 0.0
	}
	startVal := snaps[0].TotalPortfolioValue
	for _, s := range snaps {
		if !s.AsOf.After(startAt) && s.TotalPortfolioValue > 0 {
			startVal = s.TotalPortfolioValue
		}
	}
	endVal := snaps[len(snaps)-1].TotalPortfolioValue
	if startVal <= 0 {
		return 0.0
	}
	return float64(endVal-startVal) / float64(startVal) * 100.0
}

func (c *Controller) appendEvent(ctx context.Context, aggregateID string, eventType string, payload map[string]any, occurredAt time.Time) {
	if c.deps.EventHistoryRepo == nil {
		return
	}
	b, _ := json.Marshal(payload)
	_ = c.deps.EventHistoryRepo.Append(ctx, nil, &continuity.EventHistory{
		EventID:       uuid.New().String(),
		AggregateType: "EXPERIMENT",
		AggregateID:   aggregateID,
		EventType:     eventType,
		OccurredAt:    occurredAt,
		RecordedAt:    occurredAt,
		ActorID:       "experiment-controller",
		PayloadJSON:   string(b),
	})
}
