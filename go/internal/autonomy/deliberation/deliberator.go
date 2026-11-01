package deliberation

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/governance"
	domainLearning "github.com/wade/aia/internal/domain/learning"
)

type DeliberateInput struct {
	Candidates               []*domainAutonomy.CandidateInitiative
	CalmObservations         []domainAutonomy.Observation
	ActiveTaskCount          int
	MaxConcurrentActiveTasks int
	VerifiedLearnings        []*domainLearning.LearningRecord
	NegativeConstraints      []*domainLearning.LearningRecord
}

type DeliberationResult struct {
	RecommendedAction    decision.ActionType
	ChosenInitiative     *domainAutonomy.CandidateInitiative
	DeferredInitiatives  []*domainAutonomy.CandidateInitiative
	DiscardedInitiatives []*domainAutonomy.CandidateInitiative
	Options              []domainAutonomy.DeliberationOption
	OptionsEvaluatedJSON string
	CostBenefitAnalysis  string
	RiskAssessment       string
	Justification        string
}

// Deliberator evaluates candidate initiatives using Expected Utility (EV = P_s * V - C - R)
// and supports all 5 decision types: ACT, DEFER, MONITOR, ASK_HUMAN, NO_ACTION (AIA-08 §4, AIA-09 §6).
type Deliberator struct{}

func NewDeliberator() *Deliberator {
	return &Deliberator{}
}

func (d *Deliberator) Deliberate(in DeliberateInput) DeliberationResult {
	maxConcurrent := in.MaxConcurrentActiveTasks
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	// Case 1: Zero candidate initiatives discovered -> Rational NO_ACTION
	if len(in.Candidates) == 0 {
		calmNotes := make([]string, 0, len(in.CalmObservations))
		for _, co := range in.CalmObservations {
			if strings.TrimSpace(co.Summary) != "" {
				calmNotes = append(calmNotes, co.Summary)
			}
		}

		just := "Environment and project goals are within stable bounds; cost-benefit analysis favors rational NO_ACTION (zero LLM token/task overhead)."
		if len(calmNotes) > 0 {
			just = fmt.Sprintf("Rational NO_ACTION: %s", strings.Join(calmNotes, " | "))
		}

		opts := []domainAutonomy.DeliberationOption{
			{
				Option:          "NO_ACTION: Preserve current state and conserve compute/token budget",
				ActionCode:      decision.ActionNoAction,
				SuccessProb:     1.0,
				ExpectedValue:   0.0,
				Cost:            0.0,
				RiskPenalty:     0.0,
				NetUtility:      0.0,
				Reversible:      true,
				Selected:        true,
				SelectionReason: "Optimal net utility under calm/idle conditions",
			},
			{
				Option:          "ACT: Speculative full-workspace scan without trigger signal",
				ActionCode:      decision.ActionAct,
				SuccessProb:     0.10,
				ExpectedValue:   2.0,
				Cost:            2.0,
				RiskPenalty:     0.5,
				NetUtility:      -2.30,
				Reversible:      true,
				Selected:        false,
				SelectionReason: "Negative expected utility (cost + noise exceed expected gain)",
			},
		}
		optsJSON, _ := json.Marshal(opts)

		return DeliberationResult{
			RecommendedAction:    decision.ActionNoAction,
			Options:              opts,
			OptionsEvaluatedJSON: string(optsJSON),
			CostBenefitAnalysis:  "EV(NO_ACTION)=0.00 > EV(SPECULATIVE_ACT)=-2.30; zero actionable signals or unmet goal gaps.",
			RiskAssessment:       "LOW: Retaining verified baseline carries zero operational risk.",
			Justification:        just,
		}
	}

	top := in.Candidates[0]
	meta := parseMetadata(top.MetadataJSON)

	value := float64(top.PriorityScore) * 2.0
	prob := confidenceProbability(top.ConfidenceLevel)
	cost := top.EstimatedCost
	if cost <= 0 {
		cost = 1.5
	}
	riskPen := riskTierPenalty(top.EstimatedRisk)

	requiresHuman := boolFromMeta(meta, "requires_human_resolution") ||
		(top.ConfidenceLevel == domainAutonomy.ConfidenceUnverified && top.PriorityScore >= 7)
	isBlockedOrOverloaded := boolFromMeta(meta, "blocked_dependency") ||
		strFromMeta(meta, "defer_reason") != "" ||
		in.ActiveTaskCount >= maxConcurrent
	isDrift := top.ConfidenceLevel == domainAutonomy.ConfidencePotential ||
		strings.EqualFold(strFromMeta(meta, "category"), domainAutonomy.ObsCategoryDrift)

	// Evaluate candidate option matrix using EV = P_s * V - C - R
	actProb := prob
	actRisk := riskPen
	if requiresHuman {
		actProb = 0.15
		actRisk += 10.0 // Acting unilaterally on conflicted/ambiguous truth incurs severe epistemic penalty
	} else if isBlockedOrOverloaded {
		actProb = 0.15 // Congestion or blocked dependency degrades immediate execution success probability
	}

	var learningNote string
	matchedNegative := findMatchingLearning(top, meta, in.NegativeConstraints)
	matchedPositive := findMatchingLearning(top, meta, in.VerifiedLearnings)
	if matchedNegative != nil {
		// Suppress repeating a known REGRESSED / INEFFECTIVE intervention (AIA-11 §21)
		actProb = round2(actProb * 0.25)
		actRisk += 6.0
		learningNote = fmt.Sprintf(" [NegativeConstraint %s (%s) penalized ACT utility]", matchedNegative.ID, matchedNegative.EffectivenessVerdict)
	} else if matchedPositive != nil {
		boost := 0.06
		if matchedPositive.LearningLevel == domainLearning.LevelL3Generalized {
			boost = 0.10
		}
		actProb = math.Min(0.98, round2(actProb+boost))
		actRisk = math.Max(0.10, round2(actRisk-0.30))
		learningNote = fmt.Sprintf(" [Reinforced by %s learning %s]", matchedPositive.LearningLevel, matchedPositive.ID)
	}

	actUtility := round2(actProb*value - cost - actRisk)

	var opts []domainAutonomy.DeliberationOption
	opts = append(opts, domainAutonomy.DeliberationOption{
		Option:        fmt.Sprintf("ACT: %s", top.DetectedOpportunity),
		ActionCode:    decision.ActionAct,
		SuccessProb:   actProb,
		ExpectedValue: value,
		Cost:          cost,
		RiskPenalty:   actRisk,
		NetUtility:    actUtility,
		Reversible:    top.EstimatedRisk != governance.RiskTierCritical,
	})

	if requiresHuman {
		askUtil := round2(0.85*(value*0.75) - 0.50 - 0.20)
		opts = append(opts, domainAutonomy.DeliberationOption{
			Option:        fmt.Sprintf("ASK_HUMAN: Request owner clarification for %s", top.DetectedOpportunity),
			ActionCode:    decision.ActionAskHuman,
			SuccessProb:   0.85,
			ExpectedValue: round2(value * 0.75),
			Cost:          0.50,
			RiskPenalty:   0.20,
			NetUtility:    askUtil,
			Reversible:    true,
		})
	}

	if isBlockedOrOverloaded {
		deferUtil := round2(0.75*(value*0.60) - 0.20 - 0.10)
		opts = append(opts, domainAutonomy.DeliberationOption{
			Option:        fmt.Sprintf("DEFER: Queue %s until capacity/dependency clears", top.DetectedOpportunity),
			ActionCode:    decision.ActionDefer,
			SuccessProb:   0.75,
			ExpectedValue: round2(value * 0.60),
			Cost:          0.20,
			RiskPenalty:   0.10,
			NetUtility:    deferUtil,
			Reversible:    true,
		})
	}

	if isDrift {
		monUtil := round2(0.70*(value*0.45) - 0.20 - 0.10)
		opts = append(opts, domainAutonomy.DeliberationOption{
			Option:        fmt.Sprintf("MONITOR: Track %s for stronger confirmation before committing resources", top.DetectedOpportunity),
			ActionCode:    decision.ActionMonitor,
			SuccessProb:   0.70,
			ExpectedValue: round2(value * 0.45),
			Cost:          0.20,
			RiskPenalty:   0.10,
			NetUtility:    monUtil,
			Reversible:    true,
		})
	}

	opts = append(opts, domainAutonomy.DeliberationOption{
		Option:        "NO_ACTION: Do not intervene this cycle",
		ActionCode:    decision.ActionNoAction,
		SuccessProb:   1.0,
		ExpectedValue: 0.0,
		Cost:          0.0,
		RiskPenalty:   0.0,
		NetUtility:    0.0,
		Reversible:    true,
	})

	// Select highest NetUtility option
	bestIdx := 0
	for i := 1; i < len(opts); i++ {
		if opts[i].NetUtility > opts[bestIdx].NetUtility {
			bestIdx = i
		}
	}
	opts[bestIdx].Selected = true
	chosenAction := opts[bestIdx].ActionCode
	opts[bestIdx].SelectionReason = fmt.Sprintf("Highest expected net utility (EV=%.2f)", opts[bestIdx].NetUtility)

	optsJSON, _ := json.Marshal(opts)

	var res DeliberationResult
	res.RecommendedAction = chosenAction
	res.Options = opts
	res.OptionsEvaluatedJSON = string(optsJSON)
	res.CostBenefitAnalysis = fmt.Sprintf(
		"Selected %s with EV=%.2f (P_s=%.2f, V=%.2f, C=%.2f, R=%.2f) across %d evaluated alternatives.%s",
		chosenAction, opts[bestIdx].NetUtility, opts[bestIdx].SuccessProb, opts[bestIdx].ExpectedValue, opts[bestIdx].Cost, opts[bestIdx].RiskPenalty, len(opts), learningNote,
	)
	res.RiskAssessment = fmt.Sprintf("Candidate risk tier=%s, confidence=%s, active_tasks=%d/%d",
		nonEmptyRisk(top.EstimatedRisk), top.ConfidenceLevel, in.ActiveTaskCount, maxConcurrent)

	switch chosenAction {
	case decision.ActionAct:
		res.ChosenInitiative = top
		res.Justification = fmt.Sprintf("Acting on %q (%s, EV=%.2f): positive net utility outweighs execution cost and risk.%s", top.DetectedOpportunity, top.DiscoveryType, opts[bestIdx].NetUtility, learningNote)
		if len(in.Candidates) > 1 {
			res.DeferredInitiatives = in.Candidates[1:]
		}

	case decision.ActionAskHuman:
		res.ChosenInitiative = top
		res.Justification = fmt.Sprintf("Escalating %q to human owner (ASK_HUMAN, EV=%.2f): unresolved conflict or unverified high-impact ambiguity requires human judgment.%s", top.DetectedOpportunity, opts[bestIdx].NetUtility, learningNote)
		if len(in.Candidates) > 1 {
			res.DeferredInitiatives = in.Candidates[1:]
		}

	case decision.ActionDefer:
		res.ChosenInitiative = top
		reason := strFromMeta(meta, "defer_reason")
		if reason == "" {
			if boolFromMeta(meta, "blocked_dependency") {
				reason = "external dependency not yet satisfied"
			} else {
				reason = fmt.Sprintf("active task queue at capacity (%d/%d)", in.ActiveTaskCount, maxConcurrent)
			}
		}
		res.Justification = fmt.Sprintf("Deferring %q (DEFER, EV=%.2f): %s.%s", top.DetectedOpportunity, opts[bestIdx].NetUtility, reason, learningNote)
		res.DeferredInitiatives = in.Candidates

	case decision.ActionMonitor:
		res.ChosenInitiative = top
		res.Justification = fmt.Sprintf("Monitoring %q (MONITOR, EV=%.2f): signal confidence is %s; continued observation beats premature intervention.%s", top.DetectedOpportunity, opts[bestIdx].NetUtility, top.ConfidenceLevel, learningNote)
		if len(in.Candidates) > 1 {
			res.DeferredInitiatives = in.Candidates[1:]
		}

	default: // decision.ActionNoAction
		res.Justification = fmt.Sprintf("Rational NO_ACTION: candidate %q yielded negative net utility (EV=%.2f < 0.00); discarded as non-cost-effective.%s", top.DetectedOpportunity, actUtility, learningNote)
		res.DiscardedInitiatives = in.Candidates
	}

	return res
}

func findMatchingLearning(top *domainAutonomy.CandidateInitiative, meta map[string]any, records []*domainLearning.LearningRecord) *domainLearning.LearningRecord {
	if top == nil || len(records) == 0 {
		return nil
	}
	candKeys := []string{
		strFromMeta(meta, "learning_record_id"),
		strFromMeta(meta, "anti_pattern_key"),
		strFromMeta(meta, "pattern_key"),
		strFromMeta(meta, "target_entity_ref"),
		strFromMeta(meta, "learning_key"),
	}
	oppLower := strings.ToLower(top.DetectedOpportunity + " " + top.Rationale + " " + top.TriggerSource)

	for _, rec := range records {
		if rec == nil {
			continue
		}
		for _, k := range candKeys {
			if k != "" && k == rec.ID {
				return rec
			}
		}
		interv := parseMetadata(rec.InterventionJSON)
		hyp := parseMetadata(rec.HypothesisJSON)
		recKeys := []string{
			strFromMeta(interv, "anti_pattern_key"),
			strFromMeta(interv, "pattern_key"),
			strFromMeta(interv, "target_entity_ref"),
			strFromMeta(hyp, "pattern_key"),
		}
		if ps, ok := interv["proposed_state"].(map[string]any); ok {
			recKeys = append(recKeys,
				strFromMeta(ps, "anti_pattern_key"),
				strFromMeta(ps, "pattern_key"),
			)
		}
		if as, ok := interv["applied_state"].(map[string]any); ok {
			recKeys = append(recKeys,
				strFromMeta(as, "anti_pattern_key"),
				strFromMeta(as, "pattern_key"),
			)
		}
		if asRaw := strFromMeta(interv, "applied_state_json"); asRaw != "" {
			asMap := parseMetadata(asRaw)
			recKeys = append(recKeys,
				strFromMeta(asMap, "anti_pattern_key"),
				strFromMeta(asMap, "pattern_key"),
			)
		}
		for _, rk := range recKeys {
			rk = strings.TrimSpace(rk)
			if rk == "" {
				continue
			}
			for _, ck := range candKeys {
				if ck != "" && strings.EqualFold(ck, rk) {
					return rec
				}
			}
			if len(rk) >= 4 && strings.Contains(oppLower, strings.ToLower(rk)) {
				return rec
			}
		}
	}
	return nil
}

func confidenceProbability(c domainAutonomy.DiscoveryConfidence) float64 {
	switch c {
	case domainAutonomy.ConfidenceConfirmed:
		return 0.90
	case domainAutonomy.ConfidenceProbable:
		return 0.75
	case domainAutonomy.ConfidencePotential:
		return 0.35
	default:
		return 0.20
	}
}

func riskTierPenalty(r governance.RiskTier) float64 {
	switch r {
	case governance.RiskTierCritical:
		return 4.5
	case governance.RiskTierHigh:
		return 3.0
	case governance.RiskTierMedium:
		return 1.5
	default:
		return 0.5
	}
}

func parseMetadata(raw string) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

func boolFromMeta(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func strFromMeta(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func nonEmptyRisk(r governance.RiskTier) governance.RiskTier {
	if r == "" {
		return governance.RiskTierLow
	}
	return r
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
