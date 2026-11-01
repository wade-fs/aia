package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/agent"
	domainExp "github.com/wade/aia/internal/domain/experiment"
	"github.com/wade/aia/internal/domain/investment"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/provider"
)

// GenerationObservationContext encapsulates the exact point-in-time market & isolated portfolio state
// presented to a generation at timestamp AsOf (AIA-05 §20.1, Zero Look-Ahead).
type GenerationObservationContext struct {
	AsOf               time.Time
	DataSnapshotID     string
	Symbol             string
	Instrument         investment.Instrument
	CurrentBar         investment.MarketBarValue
	PriorBar           *investment.MarketBarValue
	RecentDisclosures  []investment.RawDocument
	PortfolioProfile   investment.PortfolioProfile
	PortfolioSnapshot  investment.PortfolioSnapshot
	Positions          []investment.Position
	PriorDecisions     []investment.Decision
	VerifiedLearnings  []domainLearning.LearningRecord
	SuppressedPatterns []string
	GoalSummary        string
}

// GenerationDecider produces a structured forecast & trade decision for a specific generation.
type GenerationDecider interface {
	Decide(ctx context.Context, gen *domainExp.ExperimentGeneration, obs GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error)
}

// FuncGenerationDecider adapts a Go function into a GenerationDecider (for deterministic replay & scenarios).
type FuncGenerationDecider func(ctx context.Context, gen *domainExp.ExperimentGeneration, obs GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error)

func (f FuncGenerationDecider) Decide(ctx context.Context, gen *domainExp.ExperimentGeneration, obs GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error) {
	return f(ctx, gen, obs)
}

// FrozenG1RuleDecider implements Generation 1 (Rule-Based Control Baseline, rule-v1.0-frozen).
// Its logic is completely deterministic, version-pinned, and never altered by G2's learning loop.
type FrozenG1RuleDecider struct{}

func NewFrozenG1RuleDecider() *FrozenG1RuleDecider {
	return &FrozenG1RuleDecider{}
}

func (d *FrozenG1RuleDecider) Decide(_ context.Context, _ *domainExp.ExperimentGeneration, obs GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error) {
	price := obs.CurrentBar.Close
	if price <= 0 {
		return nil, fmt.Errorf("invalid current bar close %.2f", price)
	}

	var heldQty int64
	var avgCost float64
	for _, p := range obs.Positions {
		if p.InstrumentID == obs.Instrument.ID {
			heldQty = p.Quantity
			avgCost = p.AverageCost
			break
		}
	}

	barDeltaPct := 0.0
	if obs.PriorBar != nil && obs.PriorBar.Close > 0 {
		barDeltaPct = ((price - obs.PriorBar.Close) / obs.PriorBar.Close) * 100.0
	}

	unrealizedPct := 0.0
	if heldQty > 0 && avgCost > 0 {
		unrealizedPct = ((price - avgCost) / avgCost) * 100.0
	}

	out := &domainExp.GenerationalForecastOutput{
		Symbol:                 obs.Symbol,
		HorizonDays:            5,
		UpProbability:          0.52,
		ExpectedReturnPct:      0.5,
		ExpectedRangeLowPct:    -2.5,
		ExpectedRangeHighPct:   3.0,
		Action:                 "HOLD",
		DecisionConfidence:     0.65,
		Reasoning:              []string{"G1 frozen rule: price within normal band; maintaining current allocation"},
		PrimaryRisks:           []string{"Semiconductor cycle volatility"},
		InvalidationConditions: []string{"Close drops >= 6% below average cost"},
	}

	// Rule 1: Hard stop-loss if unrealized loss <= -6.0%
	if heldQty > 0 && unrealizedPct <= -6.0 {
		out.Action = "SELL"
		out.UpProbability = 0.38
		out.ExpectedReturnPct = -2.0
		out.ExpectedRangeLowPct = -6.0
		out.ExpectedRangeHighPct = 1.5
		out.DecisionConfidence = 0.80
		out.Reasoning = []string{fmt.Sprintf("G1 frozen stop-loss rule triggered: unrealized return %.2f%% <= -6.0%%", unrealizedPct)}
		return out, nil
	}

	// Rule 2: Partial profit-taking if unrealized gain >= +8.0%
	if heldQty > 0 && unrealizedPct >= 8.0 {
		out.Action = "REDUCE"
		out.UpProbability = 0.48
		out.ExpectedReturnPct = 0.0
		out.ExpectedRangeLowPct = -3.5
		out.ExpectedRangeHighPct = 3.0
		out.DecisionConfidence = 0.72
		out.Reasoning = []string{fmt.Sprintf("G1 frozen profit-harvest rule triggered: unrealized return +%.2f%% >= +8.0%%", unrealizedPct)}
		return out, nil
	}

	// Rule 3: Mean-reversion pullback entry/add if daily bar dipped <= -3.0% (or initial 30% core entry when 100% cash and bar >= +1.5% breakout)
	if barDeltaPct <= -3.0 && obs.PortfolioSnapshot.AvailableCash > 15000 {
		if heldQty == 0 {
			out.Action = "BUY"
		} else {
			out.Action = "ADD"
		}
		out.UpProbability = 0.60
		out.ExpectedReturnPct = 2.2
		out.ExpectedRangeLowPct = -2.5
		out.ExpectedRangeHighPct = 5.0
		out.DecisionConfidence = 0.68
		out.Reasoning = []string{fmt.Sprintf("G1 frozen pullback rule triggered: bar change %.2f%% <= -3.0%%", barDeltaPct)}
		return out, nil
	}

	if heldQty == 0 && barDeltaPct >= 1.5 && obs.PortfolioSnapshot.AvailableCash > 20000 {
		out.Action = "BUY"
		out.TargetAmount = int64(math.Round(float64(obs.PortfolioSnapshot.AvailableCash) * 0.35))
		out.UpProbability = 0.58
		out.ExpectedReturnPct = 1.8
		out.ExpectedRangeLowPct = -2.0
		out.ExpectedRangeHighPct = 4.5
		out.DecisionConfidence = 0.64
		out.Reasoning = []string{fmt.Sprintf("G1 frozen momentum-entry rule triggered: flat holdings and bar +%.2f%% >= +1.5%%", barDeltaPct)}
		return out, nil
	}

	return out, nil
}

// AgyLLMGenerationDecider invokes Antigravity CLI (or configured Provider) to produce a G2 / G2.x decision.
type AgyLLMGenerationDecider struct {
	prov    provider.Provider
	workDir string
}

func NewAgyLLMGenerationDecider(prov provider.Provider, workDir string) *AgyLLMGenerationDecider {
	return &AgyLLMGenerationDecider{prov: prov, workDir: workDir}
}

func (d *AgyLLMGenerationDecider) Decide(ctx context.Context, gen *domainExp.ExperimentGeneration, obs GenerationObservationContext) (*domainExp.GenerationalForecastOutput, error) {
	if d.prov == nil {
		return nil, fmt.Errorf("LLM provider is nil for generation %s", gen.GenerationCode)
	}

	promptPayload, _ := json.MarshalIndent(map[string]any{
		"generation_code":     gen.GenerationCode,
		"strategy_version":    gen.StrategyVersion,
		"strategy_config":     gen.StrategyConfigJSON,
		"as_of":               obs.AsOf.Format(time.RFC3339),
		"primary_goal":        obs.GoalSummary,
		"symbol":              obs.Symbol,
		"current_bar":         obs.CurrentBar,
		"prior_bar":           obs.PriorBar,
		"recent_disclosures":  obs.RecentDisclosures,
		"available_cash":      obs.PortfolioSnapshot.AvailableCash,
		"total_portfolio_val": obs.PortfolioSnapshot.TotalPortfolioValue,
		"positions":           obs.Positions,
		"verified_learnings":  obs.VerifiedLearnings,
		"suppressed_patterns": obs.SuppressedPatterns,
	}, "", "  ")

	prompt := fmt.Sprintf(`You are AIA Investment Experimental Generation %s (%s).
Primary Strategic Objective: Maximize Total Portfolio Assets (NT$ virtual portfolio).
Strictly base your judgment ONLY on the Point-in-Time data as of %s below.
Distinguish strictly between:
1. "up_probability" (0.0-1.0 probability that %s closes higher over horizon_days, calibrated by Brier/LogLoss)
2. "decision_confidence" (0.0-1.0 overall confidence in your action choice BUY|SELL|ADD|REDUCE|HOLD).

Return ONLY a single JSON object matching GenerationalForecastOutput:
{"symbol":%q,"horizon_days":5,"up_probability":0.65,"expected_return_pct":2.0,"expected_range_low_pct":-3.0,"expected_range_high_pct":5.0,"action":"HOLD","decision_confidence":0.70,"reasoning":["..."],"primary_risks":["..."],"invalidation_conditions":["..."]}

Context:
%s`, gen.GenerationCode, gen.StrategyVersion, obs.AsOf.Format(time.RFC3339), obs.Symbol, obs.Symbol, string(promptPayload))

	sess, err := d.prov.Start(ctx, provider.StartRequest{
		ExecutionID: uuid.New().String(),
		TaskID:      uuid.New().String(),
		Agent:       agent.Agent{ID: "investment-agent", Name: "Investment Agent"},
		Prompt:      prompt,
		WorkDir:     d.workDir,
		Timeout:     60 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("start provider for %s: %w", gen.GenerationCode, err)
	}
	res, err := sess.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait provider for %s: %w", gen.GenerationCode, err)
	}

	raw := extractJSONObject(res.Response)
	var parsed domainExp.GenerationalForecastOutput
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse %s JSON output: %w (raw=%s)", gen.GenerationCode, err, res.Response)
	}
	return validateForecastOutput(&parsed, obs.Symbol)
}

func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return strings.TrimSpace(s)
}

func validateForecastOutput(out *domainExp.GenerationalForecastOutput, defaultSymbol string) (*domainExp.GenerationalForecastOutput, error) {
	if out == nil {
		return nil, fmt.Errorf("forecast output is nil")
	}
	if strings.TrimSpace(out.Symbol) == "" {
		out.Symbol = defaultSymbol
	}
	if out.HorizonDays <= 0 {
		out.HorizonDays = 5
	}
	if out.UpProbability < 0 || out.UpProbability > 1 {
		return nil, fmt.Errorf("up_probability %.4f must be in [0, 1]", out.UpProbability)
	}
	if out.DecisionConfidence < 0 || out.DecisionConfidence > 1 {
		return nil, fmt.Errorf("decision_confidence %.4f must be in [0, 1]", out.DecisionConfidence)
	}
	if out.ExpectedRangeHighPct < out.ExpectedRangeLowPct {
		out.ExpectedRangeLowPct, out.ExpectedRangeHighPct = out.ExpectedRangeHighPct, out.ExpectedRangeLowPct
	}
	if len(out.PrimaryRisks) == 0 {
		out.PrimaryRisks = []string{"Unspecified market volatility"}
	}
	if len(out.InvalidationConditions) == 0 {
		out.InvalidationConditions = []string{"Reassess on >5% adverse price move"}
	}
	out.Action = strings.ToUpper(strings.TrimSpace(out.Action))
	return out, nil
}
