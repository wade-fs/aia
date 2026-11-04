package initiative

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/autonomy/recall"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/governance"
	goalService "github.com/wade/aia/internal/service/goal"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type DiscoverInput struct {
	CycleID      string
	ProjectID    string
	GoalID       *string
	Observations []domainAutonomy.Observation
	Recalled     *recall.RecalledBundle
}

// Discoverer synthesizes autonomous candidate initiatives across 4 independent channels (AIA-09 §3.2):
// 1. Actionable environmental observations (excluding calm baseline snapshots)
// 2. Active long-term Goal metric gaps (GOAL_GAP)
// 3. Unresolved continuity pending issues (CONTINUITY)
// 4. L3 knowledge/memory contradictions requiring reconciliation (ISSUE)
type Discoverer struct {
	autonomyRepo *repository.AutonomyRepository
}

func NewDiscoverer(autonomyRepo *repository.AutonomyRepository) *Discoverer {
	return &Discoverer{autonomyRepo: autonomyRepo}
}

func (d *Discoverer) Discover(ctx context.Context, in DiscoverInput) ([]*domainAutonomy.CandidateInitiative, error) {
	now := time.Now().UTC()
	var candidates []*domainAutonomy.CandidateInitiative

	activeTaskIDs := map[string]bool{}
	var activeTaskTexts []string
	if in.Recalled != nil {
		for _, tk := range in.Recalled.ActiveTasks {
			activeTaskIDs[string(tk.ID)] = true
			activeTaskTexts = append(activeTaskTexts, strings.ToLower(tk.Title+" "+tk.Description))
		}
	}

	seenFingerprints := map[string]bool{}
	isDuplicateOfActiveTask := func(summary string) bool {
		norm := strings.ToLower(strings.TrimSpace(summary))
		if norm == "" {
			return true
		}
		if seenFingerprints[norm] {
			return true
		}
		for _, txt := range activeTaskTexts {
			if strings.Contains(txt, norm) {
				return true
			}
		}
		return false
	}

	var projPtr *string
	if in.ProjectID != "" {
		p := in.ProjectID
		projPtr = &p
	}

	// 1. Environmental Observations (AIA-09 §3.2A, §3.2B)
	for _, obs := range in.Observations {
		if obs.IsCalmSnapshot() {
			continue
		}
		summary := strings.TrimSpace(obs.Summary)
		if isDuplicateOfActiveTask(summary) {
			continue
		}
		seenFingerprints[strings.ToLower(summary)] = true

		goalID := obs.GoalID
		if goalID == nil {
			goalID = in.GoalID
		}

		metaMap := map[string]any{}
		for k, v := range obs.Data {
			metaMap[k] = v
		}
		if obs.Category != "" {
			metaMap["category"] = obs.Category
		}
		if obs.Domain != "" {
			metaMap["domain"] = obs.Domain
		}
		if obs.EvidenceID != nil && *obs.EvidenceID != "" {
			metaMap["evidence_id"] = *obs.EvidenceID
		}
		metaBytes, _ := json.Marshal(metaMap)

		discType := classifyObservationType(obs)
		conf := classifyObservationConfidence(obs)
		prio := classifyObservationPriority(obs)
		cost := classifyObservationCost(obs)
		risk := classifyObservationRisk(obs, prio)
		agentID := obs.TargetAgentID
		if agentID == "" {
			agentID = inferTargetAgent(obs.Domain + " " + obs.Source + " " + summary)
		}

		candidates = append(candidates, &domainAutonomy.CandidateInitiative{
			ID:                  uuid.New().String(),
			CycleID:             in.CycleID,
			ProjectID:           projPtr,
			GoalID:              goalID,
			DiscoveryType:       discType,
			ConfidenceLevel:     conf,
			TargetAgentID:       agentID,
			TriggerSource:       nonEmpty(obs.Source, "ENVIRONMENT_OBSERVER"),
			DetectedOpportunity: summary,
			Rationale:           fmt.Sprintf("Observed %s signal from %s (confidence=%s)", nonEmpty(obs.Category, "ACTIONABLE"), nonEmpty(obs.Source, "observer"), conf),
			ExpectedOutcome:     fmt.Sprintf("Address environmental signal: %s", summary),
			PriorityScore:       prio,
			EstimatedCost:       cost,
			EstimatedRisk:       risk,
			Status:              domainAutonomy.InitiativeCandidate,
			MetadataJSON:        string(metaBytes),
			CreatedAt:           now,
			UpdatedAt:           now,
		})
	}

	// 2. Active Goal Metric Gaps (AIA-09 §3.2C)
	if in.Recalled != nil {
		for _, g := range in.Recalled.ActiveGoals {
			gap := goalService.EvaluateGoalGap(g)
			if !gap.HasGap {
				continue
			}

			// Skip if this goal already has an active decomposed task in progress
			if d.autonomyRepo != nil {
				if links, err := d.autonomyRepo.ListGoalTaskLinks(ctx, g.ID); err == nil {
					hasActive := false
					for _, lk := range links {
						if activeTaskIDs[lk.TaskID] {
							hasActive = true
							break
						}
					}
					if hasActive {
						continue
					}
				}
			}

			if isDuplicateOfActiveTask(gap.GapSummary) {
				continue
			}
			seenFingerprints[strings.ToLower(gap.GapSummary)] = true

			gID := g.ID
			prio := clampPriority(g.Priority + int(math.Round(gap.GapRatio*3.0)))
			if prio < 6 {
				prio = 6
			}
			metaBytes, _ := json.Marshal(map[string]any{
				"goal_id":     g.ID,
				"goal_name":   g.Name,
				"gap_ratio":   gap.GapRatio,
				"metric_gaps": gap.MetricGaps,
			})

			candidates = append(candidates, &domainAutonomy.CandidateInitiative{
				ID:                  uuid.New().String(),
				CycleID:             in.CycleID,
				ProjectID:           projPtr,
				GoalID:              &gID,
				DiscoveryType:       domainAutonomy.DiscoveryGoalGap,
				ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
				TargetAgentID:       inferTargetAgent(g.Name + " " + g.Description),
				TriggerSource:       "GOAL_GAP_EVALUATOR",
				DetectedOpportunity: gap.GapSummary,
				Rationale:           fmt.Sprintf("Active goal %q (%s) has %.1f%% unmet target gap; decomposing concrete task to close gap.", g.Name, g.Level, gap.GapRatio*100),
				ExpectedOutcome:     fmt.Sprintf("Advance goal %q toward target metrics %s", g.Name, g.TargetMetricJSON),
				PriorityScore:       prio,
				EstimatedCost:       1.5,
				EstimatedRisk:       governance.RiskTierLow,
				Status:              domainAutonomy.InitiativeCandidate,
				MetadataJSON:        string(metaBytes),
				CreatedAt:           now,
				UpdatedAt:           now,
			})
		}

		// 3. Continuity Pending Issues (AIA-09 §3.2D)
		for _, rawIssue := range in.Recalled.PendingIssues {
			issue := strings.TrimSpace(rawIssue)
			if issue == "" {
				continue
			}
			// Passive markers recorded by prior MONITOR / DEFER / WAITING_APPROVAL cycles
			// do not auto-escalate without fresh signal
			if strings.HasPrefix(issue, "[MONITOR]") ||
				strings.HasPrefix(issue, "[DEFERRED]") ||
				strings.HasPrefix(issue, "[WAITING_APPROVAL]") {
				continue
			}
			if isDuplicateOfActiveTask(issue) {
				continue
			}
			seenFingerprints[strings.ToLower(issue)] = true

			metaBytes, _ := json.Marshal(map[string]any{
				"continuity_issue": issue,
			})
			candidates = append(candidates, &domainAutonomy.CandidateInitiative{
				ID:                  uuid.New().String(),
				CycleID:             in.CycleID,
				ProjectID:           projPtr,
				GoalID:              in.GoalID,
				DiscoveryType:       domainAutonomy.DiscoveryContinuity,
				ConfidenceLevel:     domainAutonomy.ConfidenceProbable,
				TargetAgentID:       inferTargetAgent(issue),
				TriggerSource:       "CONTINUITY_SNAPSHOT",
				DetectedOpportunity: issue,
				Rationale:           "Unresolved pending issue carried forward in project continuity state",
				ExpectedOutcome:     fmt.Sprintf("Resolve pending continuity issue: %s", issue),
				PriorityScore:       6,
				EstimatedCost:       1.5,
				EstimatedRisk:       governance.RiskTierLow,
				Status:              domainAutonomy.InitiativeCandidate,
				MetadataJSON:        string(metaBytes),
				CreatedAt:           now,
				UpdatedAt:           now,
			})
		}

		// 4. L3 Knowledge / Memory Contradictions (docs/memory.md §4)
		for _, conf := range in.Recalled.Conflicts {
			if isDuplicateOfActiveTask(conf.Description) {
				continue
			}
			// Skip if already flagged as waiting human approval in PendingIssues
			alreadyWaiting := false
			for _, pi := range in.Recalled.PendingIssues {
				if strings.HasPrefix(pi, "[WAITING_APPROVAL]") && strings.Contains(strings.ToLower(pi), strings.ToLower(conf.Topic)) {
					alreadyWaiting = true
					break
				}
			}
			if alreadyWaiting {
				continue
			}
			seenFingerprints[strings.ToLower(conf.Description)] = true

			metaBytes, _ := json.Marshal(map[string]any{
				"requires_human_resolution": true,
				"topic":                     conf.Topic,
				"claim_ids":                 conf.ClaimIDs,
				"memory_ids":                conf.MemoryIDs,
			})
			candidates = append(candidates, &domainAutonomy.CandidateInitiative{
				ID:                  uuid.New().String(),
				CycleID:             in.CycleID,
				ProjectID:           projPtr,
				GoalID:              in.GoalID,
				DiscoveryType:       domainAutonomy.DiscoveryIssue,
				ConfidenceLevel:     domainAutonomy.ConfidenceUnverified,
				TargetAgentID:       "personal-agent",
				TriggerSource:       "MEMORY_CONFLICT_DETECTOR",
				DetectedOpportunity: conf.Description,
				Rationale:           "Conflicting assertions detected in L3 knowledge/memory store; silent overwrite is forbidden, requiring human reconciliation.",
				ExpectedOutcome:     "Human owner reconciles conflicting claims or memories",
				PriorityScore:       7,
				EstimatedCost:       0.5,
				EstimatedRisk:       governance.RiskTierMedium,
				Status:              domainAutonomy.InitiativeCandidate,
				MetadataJSON:        string(metaBytes),
				CreatedAt:           now,
				UpdatedAt:           now,
			})
		}

		// 5. Open Complex Problems requiring modeling or plan decomposition (AIA-22 §10.2)
		for _, prob := range in.Recalled.ActiveProblems {
			if prob == nil {
				continue
			}
			summary := fmt.Sprintf("Analyze and Plan Problem: %s", prob.Title)
			if isDuplicateOfActiveTask(summary) {
				continue
			}
			seenFingerprints[strings.ToLower(summary)] = true

			probID := prob.ID
			metaBytes, _ := json.Marshal(map[string]any{
				"problem_id":            prob.ID,
				"problem_title":         prob.Title,
				"current_model_version": prob.CurrentModelVersion,
				"current_plan_version":  prob.CurrentPlanVersion,
				"status":                prob.Status,
			})

			prio := 7
			if prob.Priority == "HIGH" || prob.Priority == "CRITICAL" {
				prio = 8
			}

			candidates = append(candidates, &domainAutonomy.CandidateInitiative{
				ID:                  uuid.New().String(),
				CycleID:             in.CycleID,
				ProjectID:           projPtr,
				GoalID:              in.GoalID,
				ProblemID:           &probID,
				DiscoveryType:       domainAutonomy.DiscoveryProblem,
				ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
				TargetAgentID:       inferTargetAgent(prob.Title + " " + prob.Objective),
				TriggerSource:       "PROBLEM_ANALYZER",
				DetectedOpportunity: summary,
				Rationale:           fmt.Sprintf("Complex problem %q is active (status=%s, plan_version=%d) and requires epistemic analysis or plan decomposition (AIA-22 §10.2).", prob.Title, prob.Status, prob.CurrentPlanVersion),
				ExpectedOutcome:     fmt.Sprintf("Produce validated ProblemModel or versioned execution Plan for %q", prob.Title),
				PriorityScore:       prio,
				EstimatedCost:       1.5,
				EstimatedRisk:       governance.RiskTierLow,
				Status:              domainAutonomy.InitiativeCandidate,
				MetadataJSON:        string(metaBytes),
				CreatedAt:           now,
				UpdatedAt:           now,
			})
		}

		// 6. Proactive Problem Escalation from Repeated Failures & High Uncertainty (AIA-22 §10.2, §10.4)
		if len(in.Recalled.RecentFailedTasks) >= 2 {
			alreadyHandled := false
			for _, p := range in.Recalled.ActiveProblems {
				if p != nil && (strings.Contains(strings.ToLower(p.Title), "failures") || strings.Contains(strings.ToLower(p.Title), "failure")) {
					alreadyHandled = true
					break
				}
			}
			if !alreadyHandled {
				failCount := len(in.Recalled.RecentFailedTasks)
				var failedTitles []string
				var failedIDs []string
				agentVotes := make(map[string]int)
				for i, ft := range in.Recalled.RecentFailedTasks {
					if i < 3 {
						failedTitles = append(failedTitles, ft.Title)
					}
					failedIDs = append(failedIDs, string(ft.ID))
					if string(ft.AgentID) != "" {
						agentVotes[string(ft.AgentID)]++
					}
				}
				targetAgent := "coding-agent"
				maxVotes := 0
				for ag, v := range agentVotes {
					if v > maxVotes {
						maxVotes = v
						targetAgent = ag
					}
				}

				summary := fmt.Sprintf("Autonomous Escalation: Repeated task failures (%d tasks failed) in project %s", failCount, in.ProjectID)
				if !isDuplicateOfActiveTask(summary) && !seenFingerprints[strings.ToLower(summary)] {
					seenFingerprints[strings.ToLower(summary)] = true
					metaBytes, _ := json.Marshal(map[string]any{
						"auto_create_problem": true,
						"problem_title":       fmt.Sprintf("Resolve Repeated Task Failures: %s", strings.Join(failedTitles, ", ")),
						"problem_objective":   fmt.Sprintf("Systematically analyze root causes of %d failed tasks and execute verified remediation plan", failCount),
						"failed_task_ids":     failedIDs,
						"fail_count":          failCount,
					})

					candidates = append(candidates, &domainAutonomy.CandidateInitiative{
						ID:                  uuid.New().String(),
						CycleID:             in.CycleID,
						ProjectID:           projPtr,
						GoalID:              in.GoalID,
						DiscoveryType:       domainAutonomy.DiscoveryProblem,
						ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
						TargetAgentID:       targetAgent,
						TriggerSource:       "AUTONOMOUS_PROBLEM_ESCALATOR",
						DetectedOpportunity: summary,
						Rationale:           fmt.Sprintf("Multiple consecutive task failures detected (%d failed tasks); autonomy core escalates to complex problem modeling and versioned plan (AIA-22 §10.2).", failCount),
						ExpectedOutcome:     "Establish Problem entity, formulate ProblemModel with epistemic verification, and generate versioned Plan work items",
						PriorityScore:       8,
						EstimatedCost:       2.0,
						EstimatedRisk:       governance.RiskTierLow,
						Status:              domainAutonomy.InitiativeCandidate,
						MetadataJSON:        string(metaBytes),
						CreatedAt:           now,
						UpdatedAt:           now,
					})
				}
			}
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].PriorityScore != candidates[j].PriorityScore {
			return candidates[i].PriorityScore > candidates[j].PriorityScore
		}
		return confidenceRank(candidates[i].ConfidenceLevel) > confidenceRank(candidates[j].ConfidenceLevel)
	})

	return candidates, nil
}

func classifyObservationType(obs domainAutonomy.Observation) domainAutonomy.DiscoveryType {
	if obs.Data != nil {
		if dt, ok := obs.Data["discovery_type"].(string); ok && dt != "" {
			return domainAutonomy.DiscoveryType(dt)
		}
	}
	switch obs.Category {
	case domainAutonomy.ObsCategoryOpportunity, domainAutonomy.ObsCategoryMaterialChange:
		return domainAutonomy.DiscoveryOpportunity
	case domainAutonomy.ObsCategoryDrift:
		return domainAutonomy.DiscoveryContinuity
	case domainAutonomy.ObsCategoryAnomaly, domainAutonomy.ObsCategoryConstraintBreach, domainAutonomy.ObsCategoryConflict:
		return domainAutonomy.DiscoveryIssue
	default:
		return domainAutonomy.DiscoveryIssue
	}
}

func classifyObservationConfidence(obs domainAutonomy.Observation) domainAutonomy.DiscoveryConfidence {
	if obs.Confidence != "" {
		return obs.Confidence
	}
	switch obs.Category {
	case domainAutonomy.ObsCategoryMaterialChange, domainAutonomy.ObsCategoryConstraintBreach:
		return domainAutonomy.ConfidenceConfirmed
	case domainAutonomy.ObsCategoryDrift:
		return domainAutonomy.ConfidencePotential
	case domainAutonomy.ObsCategoryConflict:
		return domainAutonomy.ConfidenceUnverified
	default:
		return domainAutonomy.ConfidenceProbable
	}
}

func classifyObservationPriority(obs domainAutonomy.Observation) int {
	if obs.Severity > 0 {
		return clampPriority(obs.Severity)
	}
	switch obs.Category {
	case domainAutonomy.ObsCategoryConstraintBreach:
		return 9
	case domainAutonomy.ObsCategoryMaterialChange:
		return 8
	case domainAutonomy.ObsCategoryConflict:
		return 7
	case domainAutonomy.ObsCategoryDrift:
		return 4
	default:
		return 6
	}
}

func classifyObservationCost(obs domainAutonomy.Observation) float64 {
	if obs.Data != nil {
		if c, ok := toFloat64(obs.Data["estimated_cost"]); ok && c >= 0 {
			return c
		}
	}
	if obs.Category == domainAutonomy.ObsCategoryDrift {
		return 2.0
	}
	return 1.5
}

func classifyObservationRisk(obs domainAutonomy.Observation, prio int) governance.RiskTier {
	if obs.Data != nil {
		if rt, ok := obs.Data["risk_tier"].(string); ok && rt != "" {
			return governance.RiskTier(strings.ToUpper(rt))
		}
		if realMoney, ok := obs.Data["is_real_money"].(bool); ok && realMoney {
			return governance.RiskTierCritical
		}
		if opType, ok := obs.Data["operation_type"].(string); ok {
			if opType == "SYSTEM_COMMAND" {
				return governance.RiskTierHigh
			}
			if opType == "FINANCIAL_TRANSACTION" {
				if amt, ok := toFloat64(obs.Data["amount"]); ok && amt > 100000 {
					return governance.RiskTierHigh
				}
				return governance.RiskTierMedium
			}
		}
		if reqApp, ok := obs.Data["requires_approval"].(bool); ok && reqApp {
			return governance.RiskTierHigh
		}
	}
	if obs.Domain == "investment" || strings.Contains(strings.ToUpper(obs.Source), "INVESTMENT") {
		return governance.RiskTierMedium
	}
	if prio >= 9 {
		return governance.RiskTierHigh
	}
	return governance.RiskTierLow
}

func inferTargetAgent(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "invest") || strings.Contains(lower, "portfolio") ||
		strings.Contains(lower, "twse") || strings.Contains(lower, "mops") ||
		strings.Contains(lower, "stock") || strings.Contains(lower, "股票") ||
		strings.Contains(lower, "投資") || strings.Contains(lower, "2330") || strings.Contains(lower, "2317") || strings.Contains(lower, "0050") {
		return "investment-agent"
	}
	if strings.Contains(lower, "write") || strings.Contains(lower, "doc") ||
		strings.Contains(lower, "article") || strings.Contains(lower, "文章") || strings.Contains(lower, "撰寫") {
		return "writing-agent"
	}
	if strings.Contains(lower, "video") || strings.Contains(lower, "storyboard") || strings.Contains(lower, "影片") {
		return "video-agent"
	}
	return "coding-agent"
}

func clampPriority(p int) int {
	if p < 1 {
		return 1
	}
	if p > 10 {
		return 10
	}
	return p
}

func confidenceRank(c domainAutonomy.DiscoveryConfidence) int {
	switch c {
	case domainAutonomy.ConfidenceConfirmed:
		return 4
	case domainAutonomy.ConfidenceProbable:
		return 3
	case domainAutonomy.ConfidencePotential:
		return 2
	default:
		return 1
	}
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
