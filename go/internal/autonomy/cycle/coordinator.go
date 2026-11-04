package cycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/autonomy/clock"
	"github.com/wade/aia/internal/autonomy/deliberation"
	"github.com/wade/aia/internal/autonomy/initiative"
	"github.com/wade/aia/internal/autonomy/observation"
	"github.com/wade/aia/internal/autonomy/recall"
	"github.com/wade/aia/internal/domain"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/governance"
	domainPlan "github.com/wade/aia/internal/domain/plan"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/policy"
	approvalService "github.com/wade/aia/internal/service/approval"
	problemService "github.com/wade/aia/internal/service/problem"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var (
	ErrEmergencyStopped = errors.New("cannot run autonomy cycle: runtime is under EMERGENCY_STOP")
)

// DomainPilotExecutor allows domain modules (such as the Investment Agent Pilot)
// to execute governed paper/sandbox actions inline during Step 8 when authorized.
type DomainPilotExecutor interface {
	CanHandle(init *domainAutonomy.CandidateInitiative) bool
	Execute(ctx context.Context, init *domainAutonomy.CandidateInitiative, spawnedTaskID string) (string, error)
}

type RunInput struct {
	ProjectID                string
	TriggerType              continuity.CycleTriggerType
	TriggerRef               string
	GoalID                   *string
	Summary                  string
	MaxConcurrentActiveTasks int
	AsOf                     *time.Time
	ReplayMode               bool
}

type CoordinatorDeps struct {
	ObserverRegistry *observation.Registry
	RecallEngine     *recall.Engine
	Discoverer       *initiative.Discoverer
	Deliberator      *deliberation.Deliberator
	AutonomyRepo     *repository.AutonomyRepository
	DecisionRepo     *repository.DecisionRepository
	ContinuityRepo   *repository.ContinuityRepository
	ExecutionRepo    *repository.ExecutionRepository
	GoalRepo         *repository.GoalRepository
	EventHistoryRepo *repository.EventHistoryRepository
	KnowledgeRepo    *repository.KnowledgeRepository
	ProjectRepo      *repository.ProjectRepository
	GovernanceRepo   *repository.GovernanceRepository
	AuditRepo        *repository.AuditRepository
	TaskSvc          *taskService.Service
	ApprovalSvc      *approvalService.Service
	Evaluator        *policy.Evaluator
	Bus              eventbus.Publisher
	ProblemRepo      *repository.ProblemRepository
	PlanRepo         *repository.PlanRepository
	ProblemSvc       *problemService.Service
}

// Coordinator orchestrates the 11-step closed-loop Autonomy Cycle (AIA-08, AIA-18, Plan §2):
// 1. Observe -> 2. Recall -> 3. Assess -> 4. Discover -> 5. Deliberate ->
// 6. Decide -> 7. Governance Check -> 8. Execute -> 9. Verify -> 10. Learn -> 11. Reassess.
type Coordinator struct {
	deps   CoordinatorDeps
	pilots []DomainPilotExecutor
}

func NewCoordinator(deps CoordinatorDeps) *Coordinator {
	return &Coordinator{deps: deps}
}

func (c *Coordinator) SetProblemService(svc *problemService.Service) {
	c.deps.ProblemSvc = svc
}

func (c *Coordinator) RegisterPilot(p DomainPilotExecutor) {
	if p != nil {
		c.pilots = append(c.pilots, p)
	}
}

func (c *Coordinator) RunCycle(ctx context.Context, in RunInput) (*domainAutonomy.CycleOutcome, error) {
	if in.AsOf != nil && !in.AsOf.IsZero() {
		ctx = clock.WithAsOf(ctx, *in.AsOf, in.ReplayMode)
	}

	// Pre-flight Independent Governance Gate: Global Emergency Stop
	if c.deps.GovernanceRepo != nil {
		if ctrl, err := c.deps.GovernanceRepo.GetControl(ctx); err == nil && ctrl != nil && ctrl.IsEmergencyStopped() {
			return nil, ErrEmergencyStopped
		}
	}

	now := clock.Now(ctx)
	cycleID := uuid.New().String()
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		projectID = "default"
	}

	// Verify whether projectID exists in SQLite projects table to avoid FK violations in standalone unit tests
	projectPersisted := false
	var primaryGoalID *string
	if c.deps.ProjectRepo != nil {
		if proj, err := c.deps.ProjectRepo.Get(ctx, projectID); err == nil && proj != nil {
			projectPersisted = true
			primaryGoalID = proj.PrimaryGoalID
		}
	} else {
		// When ProjectRepo was not wired directly, assume project exists if non-empty
		projectPersisted = projectID != ""
	}
	var persistedProjectPtr *string
	if projectPersisted {
		p := projectID
		persistedProjectPtr = &p
	}

	triggerType := in.TriggerType
	if triggerType == "" {
		triggerType = continuity.TriggerScheduled
	}

	// Pre-flight Hard Resource Budget Gate (Daily Tokens & Daily Autonomy Cycles — AIA-08, AIA-17, AIA-19)
	var budgetDecision *policy.PolicyDecision
	if c.deps.Evaluator != nil {
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		var dailyTokens int64
		if c.deps.ExecutionRepo != nil {
			dailyTokens, _ = c.deps.ExecutionRepo.SumTokensSince(ctx, dayStart)
		}
		var dailyCycles int
		if c.deps.ContinuityRepo != nil {
			dailyCycles, _ = c.deps.ContinuityRepo.CountCyclesSince(ctx, dayStart, &now)
		}
		bDec := c.deps.Evaluator.EvaluateResourceBudget(ctx, "autonomy-coordinator", dailyTokens, dailyCycles)
		if !bDec.Allowed {
			budgetDecision = &bDec
		}
	}

	var cycleNumber int64 = 1
	if c.deps.ContinuityRepo != nil {
		if nextNum, err := c.deps.ContinuityRepo.NextCycleNumber(ctx); err == nil && nextNum > 0 {
			cycleNumber = nextNum
		} else {
			cycleNumber = now.UnixNano()
		}
	}

	cycleRecord := &continuity.AutonomyCycle{
		ID:           cycleID,
		CycleNumber:  cycleNumber,
		TriggerType:  triggerType,
		TriggerRefID: in.TriggerRef,
		Status:       continuity.CycleStatusStarted,
		Summary:      in.Summary,
		StartedAt:    now,
		CreatedAt:    now,
	}
	if c.deps.ContinuityRepo != nil {
		if err := c.deps.ContinuityRepo.CreateCycle(ctx, cycleRecord); err != nil {
			return nil, fmt.Errorf("failed to create autonomy cycle: %w", err)
		}
	}

	if budgetDecision != nil {
		deferDec := &decision.Decision{
			ID:               uuid.New().String(),
			CycleID:          &cycleID,
			ProjectID:        persistedProjectPtr,
			ActionType:       decision.ActionDefer,
			IntentSummary:    fmt.Sprintf("Autonomy Cycle #%d Deferred by Budget Gate", cycleNumber),
			Justification:    budgetDecision.Reason,
			GovernanceResult: decision.GovernanceBlocked,
			CreatedAt:        now,
		}
		if c.deps.DecisionRepo != nil {
			_ = c.deps.DecisionRepo.Create(ctx, deferDec, nil)
		}
		finalSummary := fmt.Sprintf("Cycle #%d deferred by hard resource budget gate: %s", cycleNumber, budgetDecision.Reason)
		if c.deps.ContinuityRepo != nil {
			_ = c.deps.ContinuityRepo.UpdateCycleStatus(ctx, cycleID, continuity.CycleStatusCompleted, finalSummary, &now)
		}
		c.appendCycleEvent(ctx, cycleID, "CYCLE_BUDGET_DEFERRED", map[string]any{
			"cycle_number": cycleNumber,
			"rule_name":    budgetDecision.RuleName,
			"reason":       budgetDecision.Reason,
		})
		return &domainAutonomy.CycleOutcome{
			CycleID:          cycleID,
			CycleNumber:      cycleNumber,
			Decision:         deferDec,
			GovernanceResult: governance.GovernanceActionBlocked,
			Verification: &domainAutonomy.VerificationSummary{
				Passed:        true,
				ActualSummary: "Budget gate short-circuited cycle with zero downstream task/LLM spend",
				VerifiedAt:    now,
			},
			Summary:     finalSummary,
			CompletedAt: now,
		}, nil
	}

	c.appendCycleEvent(ctx, cycleID, "CYCLE_STARTED", map[string]any{
		"cycle_number": cycleNumber,
		"project_id":   projectID,
		"trigger_type": triggerType,
		"summary":      in.Summary,
	})

	var traces []domainAutonomy.StepTrace
	recordStep := func(num int, name, summary string, start time.Time, meta map[string]any) {
		traces = append(traces, domainAutonomy.StepTrace{
			StepNumber: num,
			StepName:   name,
			Status:     "COMPLETED",
			Summary:    summary,
			Metadata:   meta,
			DurationMs: time.Since(start).Milliseconds(),
		})
	}

	// =========================================================================
	// Step 1: Observe (環境與資料訊號收集)
	// =========================================================================
	t1 := time.Now()
	var observations []domainAutonomy.Observation
	var calmObservations []domainAutonomy.Observation
	if c.deps.ObserverRegistry != nil {
		observations = c.deps.ObserverRegistry.ObserveAll(ctx, projectID)
	}
	actionableObsCount := 0
	for _, obs := range observations {
		if obs.IsCalmSnapshot() {
			calmObservations = append(calmObservations, obs)
		} else {
			actionableObsCount++
		}
	}
	c.appendCycleEvent(ctx, cycleID, "ENVIRONMENT_OBSERVED", map[string]any{
		"total_observations":      len(observations),
		"actionable_observations": actionableObsCount,
		"calm_snapshots":          len(calmObservations),
	})
	recordStep(1, "Observe", fmt.Sprintf("Collected %d observations (%d actionable, %d calm snapshots)", len(observations), actionableObsCount, len(calmObservations)), t1, map[string]any{
		"total":      len(observations),
		"actionable": actionableObsCount,
	})

	// =========================================================================
	// Step 2: Recall (L1–L5 聯想檢索記憶、證據與歷史脈絡)
	// =========================================================================
	t2 := time.Now()
	var recalled *recall.RecalledBundle
	if c.deps.RecallEngine != nil {
		b, err := c.deps.RecallEngine.Recall(ctx, recall.RecallInput{
			ProjectID:    projectID,
			GoalID:       in.GoalID,
			Observations: observations,
			MaxMemories:  12,
		})
		if err == nil {
			recalled = b
		}
	}
	if recalled == nil {
		recalled = &recall.RecalledBundle{}
	}
	recordStep(2, "Recall", recalled.ActiveDigest, t2, map[string]any{
		"memories": len(recalled.Memories),
		"evidence": len(recalled.GroundedEvidence),
		"events":   len(recalled.RecentEvents),
	})

	// =========================================================================
	// Step 3: Assess (評估專案、活躍目標差距與待處理問題)
	// =========================================================================
	t3 := time.Now()
	assessSummary := fmt.Sprintf("Assessed %d active goals, %d active tasks, %d active problems, %d pending issues, %d conflicts",
		len(recalled.ActiveGoals), len(recalled.ActiveTasks), len(recalled.ActiveProblems), len(recalled.PendingIssues), len(recalled.Conflicts))
	recordStep(3, "Assess", assessSummary, t3, map[string]any{
		"active_goals":    len(recalled.ActiveGoals),
		"active_tasks":    len(recalled.ActiveTasks),
		"active_problems": len(recalled.ActiveProblems),
		"active_plans":    len(recalled.ActivePlans),
		"pending_issues":  len(recalled.PendingIssues),
		"conflicts":       len(recalled.Conflicts),
	})

	// =========================================================================
	// Step 4: Discover (主動發現候選議題與機會並執行活躍工單去重)
	// =========================================================================
	t4 := time.Now()
	var candidates []*domainAutonomy.CandidateInitiative
	if c.deps.Discoverer != nil {
		discovered, err := c.deps.Discoverer.Discover(ctx, initiative.DiscoverInput{
			CycleID:      cycleID,
			ProjectID:    projectID,
			GoalID:       in.GoalID,
			Observations: observations,
			Recalled:     recalled,
		})
		if err == nil {
			for _, cand := range discovered {
				cand.ProjectID = persistedProjectPtr
				if c.deps.AutonomyRepo != nil {
					if err := c.deps.AutonomyRepo.CreateInitiative(ctx, cand); err != nil {
						continue
					}
				}
				candidates = append(candidates, cand)
			}
		}
	}
	recordStep(4, "Discover", fmt.Sprintf("Discovered %d deduplicated candidate initiatives", len(candidates)), t4, map[string]any{
		"candidate_count": len(candidates),
	})

	// =========================================================================
	// Step 5: Deliberate (深思熟慮、評估期望效用與理性不行動)
	// =========================================================================
	t5 := time.Now()
	if c.deps.ContinuityRepo != nil {
		_ = c.deps.ContinuityRepo.UpdateCycleStatus(ctx, cycleID, continuity.CycleStatusDeliberating, in.Summary, nil)
	}
	delibRes := c.deps.Deliberator.Deliberate(deliberation.DeliberateInput{
		Candidates:               candidates,
		CalmObservations:         calmObservations,
		ActiveTaskCount:          len(recalled.ActiveTasks),
		MaxConcurrentActiveTasks: in.MaxConcurrentActiveTasks,
		VerifiedLearnings:        recalled.VerifiedLearnings,
		NegativeConstraints:      recalled.NegativeConstraints,
	})

	delibRecord := &domainAutonomy.DeliberationRecord{
		ID:                   uuid.New().String(),
		CycleID:              cycleID,
		OptionsEvaluatedJSON: delibRes.OptionsEvaluatedJSON,
		CostBenefitAnalysis:  delibRes.CostBenefitAnalysis,
		RiskAssessment:       delibRes.RiskAssessment,
		RecommendedAction:    delibRes.RecommendedAction,
		CreatedAt:            clock.Now(ctx),
	}
	if delibRes.ChosenInitiative != nil {
		delibRecord.CandidateInitiativeID = &delibRes.ChosenInitiative.ID
	}
	if c.deps.AutonomyRepo != nil {
		_ = c.deps.AutonomyRepo.CreateDeliberation(ctx, delibRecord)
	}
	recordStep(5, "Deliberate", delibRes.CostBenefitAnalysis, t5, map[string]any{
		"recommended_action": delibRes.RecommendedAction,
		"options_evaluated":  len(delibRes.Options),
	})

	// =========================================================================
	// Step 6: Decide (產出正式結構化決策)
	// =========================================================================
	t6 := time.Now()
	chosenAction := delibRes.RecommendedAction
	chosenInitiative := delibRes.ChosenInitiative
	decID := uuid.New().String()

	dec := &decision.Decision{
		ID:                         decID,
		CycleID:                    &cycleID,
		ProjectID:                  persistedProjectPtr,
		ActionType:                 chosenAction,
		IntentSummary:              fmt.Sprintf("Autonomy Cycle #%d Decision: %s", cycleNumber, chosenAction),
		Justification:              delibRes.Justification,
		ConsideredAlternativesJSON: delibRes.OptionsEvaluatedJSON,
		GovernanceResult:           decision.GovernanceAllowed,
		CreatedAt:                  clock.Now(ctx),
	}
	if recalled.LatestSnapshot != nil {
		dec.ContextSnapshotID = &recalled.LatestSnapshot.ID
	}
	recordStep(6, "Decide", fmt.Sprintf("Committed intent %s: %s", chosenAction, delibRes.Justification), t6, map[string]any{
		"decision_id": decID,
		"action_type": chosenAction,
	})

	// =========================================================================
	// Step 7: Governance Check (獨立策略閘門、風險分級與人工審批判定)
	// =========================================================================
	t7 := time.Now()
	govResult := decision.GovernanceAllowed
	govReason := "Within standard autonomous governance boundaries"
	effectiveRiskTier := governance.RiskTierLow

	if chosenAction == decision.ActionAct && chosenInitiative != nil {
		effectiveRiskTier = chosenInitiative.EstimatedRisk
		if effectiveRiskTier == "" {
			effectiveRiskTier = governance.RiskTierLow
		}
		opCtx := buildPolicyOperationContext(chosenInitiative, decID)

		if c.deps.Evaluator != nil {
			pDec := c.deps.Evaluator.EvaluateOperation(ctx, opCtx)
			riskDec := c.deps.Evaluator.EvaluateRisk(ctx, mapRiskTierToDomainRisk(effectiveRiskTier))

			if (!pDec.Allowed && !pDec.RequiresApproval) || riskDec.Decision == policy.DecisionDeny {
				govResult = decision.GovernanceBlocked
				if !pDec.Allowed && pDec.Reason != "" {
					govReason = pDec.Reason
				} else {
					govReason = riskDec.Reason
				}
			} else if pDec.RequiresApproval || riskDec.Decision == policy.DecisionAsk {
				govResult = decision.GovernanceRequiredApproval
				if pDec.RequiresApproval && pDec.Reason != "" {
					govReason = pDec.Reason
				} else {
					govReason = riskDec.Reason
				}
				if pDec.RiskTier == governance.RiskTierHigh || pDec.RiskTier == governance.RiskTierCritical {
					effectiveRiskTier = pDec.RiskTier
				} else if effectiveRiskTier == governance.RiskTierLow || effectiveRiskTier == governance.RiskTierMedium {
					effectiveRiskTier = governance.RiskTierHigh
				}
			}
		}
	} else if chosenAction == decision.ActionAskHuman {
		govResult = decision.GovernanceRequiredApproval
		effectiveRiskTier = governance.RiskTierMedium
		govReason = "Decision requires human owner clarification before autonomous action"
	}

	dec.GovernanceResult = govResult
	if govResult == decision.GovernanceBlocked {
		dec.Justification = fmt.Sprintf("%s [GOVERNANCE BLOCKED: %s]", dec.Justification, govReason)
	}
	recordStep(7, "Governance Check", fmt.Sprintf("Governance=%s (%s)", govResult, govReason), t7, map[string]any{
		"governance_result": govResult,
		"risk_tier":         effectiveRiskTier,
	})

	// =========================================================================
	// Step 8: Execute (依決策與治理結果執行工單派發、審批掛起、Pilot 結算或狀態標記)
	// =========================================================================
	t8 := time.Now()
	if c.deps.ContinuityRepo != nil {
		_ = c.deps.ContinuityRepo.UpdateCycleStatus(ctx, cycleID, continuity.CycleStatusExecuting, dec.IntentSummary, nil)
	}

	var spawnedTaskID *string
	var approvalID *string
	execSummary := "No task execution required"

	// Prepare optional grounded evidence links for Decision
	var eviLinks []*decision.DecisionEvidenceLink
	if chosenInitiative != nil {
		meta := parseMetaMap(chosenInitiative.MetadataJSON)
		if eviID, ok := meta["evidence_id"].(string); ok && eviID != "" {
			if c.deps.KnowledgeRepo == nil {
				eviLinks = append(eviLinks, &decision.DecisionEvidenceLink{
					DecisionID:    decID,
					EvidenceID:    eviID,
					RelevanceNote: "Grounded observation evidence",
				})
			} else if evi, err := c.deps.KnowledgeRepo.GetEvidence(ctx, eviID); err == nil && evi != nil {
				eviLinks = append(eviLinks, &decision.DecisionEvidenceLink{
					DecisionID:    decID,
					EvidenceID:    eviID,
					RelevanceNote: "Grounded observation evidence",
				})
			}
		}
	}

	switch chosenAction {
	case decision.ActionAct:
		if govResult == decision.GovernanceBlocked {
			if chosenInitiative != nil && c.deps.AutonomyRepo != nil {
				reason := "Blocked by governance policy: " + govReason
				_ = c.deps.AutonomyRepo.UpdateInitiativeStatus(ctx, chosenInitiative.ID, domainAutonomy.InitiativeDiscarded, &reason)
				chosenInitiative.Status = domainAutonomy.InitiativeDiscarded
				chosenInitiative.DiscardReason = &reason
			}
			c.persistDecisionAndDeliberationLink(ctx, dec, eviLinks, delibRecord)
			execSummary = "Execution blocked by independent governance policy; zero tasks dispatched"
		} else if chosenInitiative != nil {
			meta := parseMetaMap(chosenInitiative.MetadataJSON)
			isAutoProblem := false
			if v, ok := meta["auto_create_problem"].(bool); ok && v {
				isAutoProblem = true
			}

			// Proactive AIA-22 Problem Escalation & Plan Decomposition (§10.2)
			if (isAutoProblem || chosenInitiative.ProblemID != nil) && c.deps.ProblemSvc != nil {
				pTaskID, pSummary, pErr := c.executeAutonomousProblemOrPlan(ctx, projectID, cycleID, chosenInitiative, effectiveRiskTier, govResult == decision.GovernanceRequiredApproval)
				if pErr == nil && pTaskID != nil {
					spawnedTaskID = pTaskID
					dec.ActionTaskID = pTaskID
					chosenInitiative.SpawnedTaskID = pTaskID
					if c.deps.AutonomyRepo != nil {
						_ = c.deps.AutonomyRepo.MarkInitiativeAdopted(ctx, chosenInitiative.ID, *pTaskID)
						chosenInitiative.Status = domainAutonomy.InitiativeAdopted
					}
					execSummary = pSummary
				} else if pErr != nil {
					execSummary = fmt.Sprintf("Problem escalation error: %v (fallback to direct task)", pErr)
				}
			}

			if spawnedTaskID == nil && c.deps.TaskSvc != nil {
				agentID := chosenInitiative.TargetAgentID
				if agentID == "" {
					agentID = "coding-agent"
				}
				createdTask, err := c.deps.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
					ProjectID:   projectID,
					AgentID:     agentID,
					Title:       "Autonomous Task: " + chosenInitiative.DetectedOpportunity,
					Description: chosenInitiative.Rationale,
					Priority:    chosenInitiative.PriorityScore * 10,
					Risk:        mapRiskTierToDomainRisk(effectiveRiskTier),
				})
				if err == nil && createdTask != nil {
					tID := string(createdTask.ID)
					spawnedTaskID = &tID
					dec.ActionTaskID = &tID

					if c.deps.AutonomyRepo != nil {
						_ = c.deps.AutonomyRepo.MarkInitiativeAdopted(ctx, chosenInitiative.ID, tID)
						chosenInitiative.Status = domainAutonomy.InitiativeAdopted
						chosenInitiative.SpawnedTaskID = &tID

						if chosenInitiative.GoalID != nil && *chosenInitiative.GoalID != "" {
							_ = c.deps.AutonomyRepo.LinkGoalTask(ctx, &domainAutonomy.GoalTaskLink{
								GoalID:                 *chosenInitiative.GoalID,
								TaskID:                 tID,
								CycleID:                &cycleID,
								DecompositionRationale: chosenInitiative.Rationale,
							})
						}
					}
				}
			}

			// Persist Decision BEFORE requesting Approval so approvals.decision_id FK is satisfied
			c.persistDecisionAndDeliberationLink(ctx, dec, eviLinks, delibRecord)

			if govResult == decision.GovernanceRequiredApproval {
				if spawnedTaskID != nil && c.deps.ApprovalSvc != nil {
					detailBytes, _ := json.Marshal(map[string]any{
						"cycle_id":      cycleID,
						"initiative_id": chosenInitiative.ID,
						"opportunity":   chosenInitiative.DetectedOpportunity,
						"reason":        govReason,
					})
					appr, aErr := c.deps.ApprovalSvc.RequestApproval(ctx, approvalService.RequestApprovalInput{
						RequesterEntity:     "AGENT:" + nonEmptyStr(chosenInitiative.TargetAgentID, "autonomy_core"),
						OperationType:       string(governance.OperationTaskExecution),
						OperationDetailJSON: string(detailBytes),
						RiskTier:            effectiveRiskTier,
						TaskID:              spawnedTaskID,
						DecisionID:          &decID,
					})
					if aErr == nil && appr != nil {
						approvalID = &appr.ID
					}
				} else if spawnedTaskID != nil && c.deps.TaskSvc != nil {
					_ = c.deps.TaskSvc.UpdateStatus(ctx, *spawnedTaskID, task.StatusWaitingApproval)
				}
				execSummary = fmt.Sprintf("Task %v created in WAITING_APPROVAL awaiting human authorization", ptrVal(spawnedTaskID))
			} else if spawnedTaskID != nil {
				execSummary = fmt.Sprintf("Dispatched autonomous task %s to %s", *spawnedTaskID, chosenInitiative.TargetAgentID)
				// Run registered domain pilot if applicable (e.g. Investment Agent Paper-Trading Pilot)
				for _, pilot := range c.pilots {
					if pilot.CanHandle(chosenInitiative) {
						if pilotNote, pErr := pilot.Execute(ctx, chosenInitiative, *spawnedTaskID); pErr == nil && pilotNote != "" {
							execSummary = execSummary + " | " + pilotNote
						}
						break
					}
				}
			}
		} else {
			c.persistDecisionAndDeliberationLink(ctx, dec, eviLinks, delibRecord)
		}

	case decision.ActionAskHuman:
		if chosenInitiative != nil && c.deps.AutonomyRepo != nil {
			reason := "Awaiting human owner clarification"
			_ = c.deps.AutonomyRepo.UpdateInitiativeStatus(ctx, chosenInitiative.ID, domainAutonomy.InitiativeDeferred, &reason)
			chosenInitiative.Status = domainAutonomy.InitiativeDeferred
		}
		c.persistDecisionAndDeliberationLink(ctx, dec, eviLinks, delibRecord)
		if c.deps.ApprovalSvc != nil {
			topic := "Ambiguous or conflicted state"
			if chosenInitiative != nil {
				topic = chosenInitiative.DetectedOpportunity
			}
			detailBytes, _ := json.Marshal(map[string]any{
				"cycle_id": cycleID,
				"topic":    topic,
			})
			appr, aErr := c.deps.ApprovalSvc.RequestApproval(ctx, approvalService.RequestApprovalInput{
				RequesterEntity:     "AGENT:autonomy_core",
				OperationType:       "HUMAN_CLARIFICATION",
				OperationDetailJSON: string(detailBytes),
				RiskTier:            governance.RiskTierMedium,
				TaskID:              nil,
				DecisionID:          &decID,
			})
			if aErr == nil && appr != nil {
				approvalID = &appr.ID
			}
		}
		execSummary = "Created scope-bound human clarification request (ASK_HUMAN)"

	case decision.ActionMonitor:
		if chosenInitiative != nil && c.deps.AutonomyRepo != nil {
			_ = c.deps.AutonomyRepo.UpdateInitiativeStatus(ctx, chosenInitiative.ID, domainAutonomy.InitiativeMonitoring, nil)
			chosenInitiative.Status = domainAutonomy.InitiativeMonitoring
		}
		c.persistDecisionAndDeliberationLink(ctx, dec, eviLinks, delibRecord)
		execSummary = "Placed signal under active watch (MONITOR) with zero task dispatch"

	case decision.ActionDefer:
		c.persistDecisionAndDeliberationLink(ctx, dec, eviLinks, delibRecord)
		execSummary = "Deferred candidate initiatives to future cycle (DEFER)"

	default: // decision.ActionNoAction
		c.persistDecisionAndDeliberationLink(ctx, dec, eviLinks, delibRecord)
		execSummary = "Rational NO_ACTION executed: zero tasks created, zero tokens spent"
	}

	// Update statuses of deferred and discarded initiatives so all cycle candidates have terminal/explicit state
	if c.deps.AutonomyRepo != nil {
		deferNote := "Deferred during cycle deliberation"
		for _, di := range delibRes.DeferredInitiatives {
			_ = c.deps.AutonomyRepo.UpdateInitiativeStatus(ctx, di.ID, domainAutonomy.InitiativeDeferred, &deferNote)
			di.Status = domainAutonomy.InitiativeDeferred
		}
		discardNote := "Negative expected net utility"
		for _, dsc := range delibRes.DiscardedInitiatives {
			_ = c.deps.AutonomyRepo.UpdateInitiativeStatus(ctx, dsc.ID, domainAutonomy.InitiativeDiscarded, &discardNote)
			dsc.Status = domainAutonomy.InitiativeDiscarded
			dsc.DiscardReason = &discardNote
		}
	}

	recordStep(8, "Execute", execSummary, t8, map[string]any{
		"spawned_task_id": ptrVal(spawnedTaskID),
		"approval_id":     ptrVal(approvalID),
	})

	// =========================================================================
	// Step 9: Verify (驗證週期不變量與預期產出)
	// =========================================================================
	t9 := time.Now()
	if c.deps.ContinuityRepo != nil {
		_ = c.deps.ContinuityRepo.UpdateCycleStatus(ctx, cycleID, continuity.CycleStatusEvaluating, dec.IntentSummary, nil)
	}
	verification := verifyCycleInvariants(dec, govResult, spawnedTaskID, approvalID, c.deps.ApprovalSvc != nil)
	recordStep(9, "Verify", verification.ActualSummary, t9, map[string]any{
		"passed": verification.Passed,
		"checks": len(verification.Checks),
	})

	// =========================================================================
	// Step 10: Learn (寫入受來源治理保護之決策記憶與不可變事件鏈)
	// =========================================================================
	t10 := time.Now()
	var learnedMemID *string
	if c.deps.RecallEngine != nil {
		if mem, mErr := c.deps.RecallEngine.RecordCycleDecisionMemory(ctx, projectID, dec, cycleNumber); mErr == nil && mem != nil {
			learnedMemID = &mem.ID
		}
	}
	c.appendCycleEvent(ctx, cycleID, "DECISION_COMMITTED", map[string]any{
		"decision_id":       dec.ID,
		"action_type":       dec.ActionType,
		"governance_result": dec.GovernanceResult,
		"task_id":           ptrVal(spawnedTaskID),
		"approval_id":       ptrVal(approvalID),
	})
	if c.deps.Bus != nil {
		c.deps.Bus.Publish("autonomy.cycle_completed", map[string]any{
			"cycle_id":          cycleID,
			"cycle_number":      cycleNumber,
			"project_id":        projectID,
			"action_type":       chosenAction,
			"governance_result": govResult,
			"task_id":           spawnedTaskID,
			"approval_id":       approvalID,
		})
	}
	recordStep(10, "Learn", fmt.Sprintf("Recorded governed AGENT_DECISION memory (%s) and L1 event", ptrVal(learnedMemID)), t10, map[string]any{
		"memory_id": ptrVal(learnedMemID),
	})

	// =========================================================================
	// Step 11: Reassess (產生脫敏 POST_CYCLE 快照、更新專案連續性並收斂週期)
	// =========================================================================
	t11 := time.Now()
	var postSnapID *string
	completedAt := clock.Now(ctx)

	nextPendingIssues := buildNextPendingIssues(recalled.PendingIssues, chosenAction, chosenInitiative, spawnedTaskID != nil)
	if c.deps.ContinuityRepo != nil {
		postSnap := buildPostCycleSnapshot(projectID, projectPersisted, cycleID, cycleNumber, recalled.ActiveTasks, spawnedTaskID, dec, nextPendingIssues, completedAt)
		if err := c.deps.ContinuityRepo.CreateSnapshot(ctx, postSnap); err == nil {
			postSnapID = &postSnap.ID
			_ = c.deps.ContinuityRepo.UpdateCycleSnapshot(ctx, cycleID, postSnap.ID)
			if projectPersisted && c.deps.ProjectRepo != nil {
				_ = c.deps.ProjectRepo.UpdateContinuity(ctx, projectID, primaryGoalID, &postSnap.ID, postSnap.PendingIssuesJSON)
			}
		}
	}

	finalSummary := fmt.Sprintf("Cycle #%d completed with %s (%s)", cycleNumber, chosenAction, govResult)
	if c.deps.ContinuityRepo != nil {
		_ = c.deps.ContinuityRepo.UpdateCycleStatus(ctx, cycleID, continuity.CycleStatusCompleted, finalSummary, &completedAt)
	}
	c.appendCycleEvent(ctx, cycleID, "CYCLE_COMPLETED", map[string]any{
		"cycle_number":     cycleNumber,
		"action_type":      chosenAction,
		"governance":       govResult,
		"post_snapshot_id": ptrVal(postSnapID),
		"verification_ok":  verification.Passed,
	})
	if projectPersisted {
		c.appendAggregateEvent(ctx, "PROJECT", projectID, "AUTONOMY_CYCLE_COMPLETED", map[string]any{
			"cycle_id":         cycleID,
			"cycle_number":     cycleNumber,
			"decision_id":      dec.ID,
			"action_type":      chosenAction,
			"governance":       govResult,
			"spawned_task_id":  ptrVal(spawnedTaskID),
			"post_snapshot_id": ptrVal(postSnapID),
		})
	}
	recordStep(11, "Reassess", fmt.Sprintf("Saved POST_CYCLE snapshot (%s) and marked cycle COMPLETED", ptrVal(postSnapID)), t11, map[string]any{
		"post_cycle_snapshot_id": ptrVal(postSnapID),
		"pending_issues_count":   len(nextPendingIssues),
	})

	return &domainAutonomy.CycleOutcome{
		CycleID:             cycleID,
		CycleNumber:         cycleNumber,
		Observations:        observations,
		Initiatives:         candidates,
		Deliberation:        delibRecord,
		Decision:            dec,
		GovernanceResult:    governance.GovernanceAction(govResult),
		ApprovalID:          approvalID,
		TaskID:              spawnedTaskID,
		Verification:        verification,
		LearnedMemoryID:     learnedMemID,
		PostCycleSnapshotID: postSnapID,
		StepTraces:          traces,
		Summary:             finalSummary,
		CompletedAt:         completedAt,
	}, nil
}

func (c *Coordinator) persistDecisionAndDeliberationLink(
	ctx context.Context,
	dec *decision.Decision,
	eviLinks []*decision.DecisionEvidenceLink,
	delib *domainAutonomy.DeliberationRecord,
) {
	if c.deps.DecisionRepo != nil {
		_ = c.deps.DecisionRepo.Create(ctx, dec, eviLinks)
	}
	if delib != nil {
		delib.DecisionID = &dec.ID
		if c.deps.AutonomyRepo != nil {
			_ = c.deps.AutonomyRepo.LinkDeliberationDecision(ctx, delib.ID, dec.ID)
		}
	}
}

func (c *Coordinator) appendCycleEvent(ctx context.Context, cycleID string, eventType string, payload map[string]any) {
	c.appendAggregateEvent(ctx, "AUTONOMY_CYCLE", cycleID, eventType, payload)
}

func (c *Coordinator) appendAggregateEvent(ctx context.Context, aggType, aggID, eventType string, payload map[string]any) {
	if c.deps.EventHistoryRepo == nil || aggID == "" {
		return
	}
	b, _ := json.Marshal(payload)
	evtID := uuid.New().String()
	_ = c.deps.EventHistoryRepo.Append(ctx, nil, &continuity.EventHistory{
		EventID:       evtID,
		EventType:     eventType,
		AggregateType: aggType,
		AggregateID:   aggID,
		CorrelationID: aggID,
		CausationID:   evtID,
		ActorID:       "SYSTEM:autonomy_coordinator",
		PayloadJSON:   string(b),
		OccurredAt:    clock.Now(ctx),
	})
}

func buildPolicyOperationContext(init *domainAutonomy.CandidateInitiative, decisionID string) policy.OperationContext {
	meta := parseMetaMap(init.MetadataJSON)
	opType := string(governance.OperationTaskExecution)
	if v, ok := meta["operation_type"].(string); ok && v != "" {
		opType = v
	}
	targetPath, _ := meta["target_path"].(string)
	command, _ := meta["command"].(string)
	isRealMoney, _ := meta["is_real_money"].(bool)
	var amount float64
	switch n := meta["amount"].(type) {
	case float64:
		amount = n
	case int:
		amount = float64(n)
	case int64:
		amount = float64(n)
	}

	return policy.OperationContext{
		ActorType:     "AGENT",
		ActorID:       nonEmptyStr(init.TargetAgentID, "autonomy-engine"),
		OperationType: opType,
		TargetPath:    targetPath,
		Command:       command,
		Amount:        amount,
		IsRealMoney:   isRealMoney,
		TaskID:        init.ID,
		DecisionID:    decisionID,
	}
}

func mapRiskTierToDomainRisk(rt governance.RiskTier) domain.RiskLevel {
	switch rt {
	case governance.RiskTierCritical:
		return domain.RiskCritical
	case governance.RiskTierHigh:
		return domain.RiskHigh
	case governance.RiskTierMedium:
		return domain.RiskMedium
	default:
		return domain.RiskLow
	}
}

func verifyCycleInvariants(
	dec *decision.Decision,
	govResult decision.GovernanceResult,
	taskID *string,
	approvalID *string,
	hasApprovalSvc bool,
) *domainAutonomy.VerificationSummary {
	var checks []string
	passed := true

	if dec != nil && dec.ID != "" {
		checks = append(checks, "decision_record_assigned")
	} else {
		passed = false
		checks = append(checks, "FAIL:missing_decision_record")
	}

	if govResult == decision.GovernanceBlocked {
		if taskID == nil {
			checks = append(checks, "blocked_action_spawned_zero_tasks")
		} else {
			passed = false
			checks = append(checks, "FAIL:blocked_action_spawned_task")
		}
	}

	if dec != nil && dec.ActionType == decision.ActionNoAction {
		if taskID == nil {
			checks = append(checks, "no_action_conserved_resources")
		} else {
			passed = false
			checks = append(checks, "FAIL:no_action_spawned_task")
		}
	}

	if govResult == decision.GovernanceRequiredApproval && hasApprovalSvc {
		if approvalID != nil && *approvalID != "" {
			checks = append(checks, "approval_gate_created")
		} else {
			passed = false
			checks = append(checks, "FAIL:missing_approval_gate")
		}
	}

	return &domainAutonomy.VerificationSummary{
		Passed:          passed,
		ExpectedSummary: fmt.Sprintf("Action=%s consistent with Governance=%s", dec.ActionType, govResult),
		ActualSummary:   fmt.Sprintf("Verified %d invariants (passed=%v)", len(checks), passed),
		Checks:          checks,
		VerifiedAt:      time.Now().UTC(),
	}
}

func buildNextPendingIssues(
	existing []string,
	action decision.ActionType,
	chosen *domainAutonomy.CandidateInitiative,
	taskSpawned bool,
) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		key := strings.ToLower(s)
		if !seen[key] {
			seen[key] = true
			out = append(out, s)
		}
	}

	for _, iss := range existing {
		// If a continuity pending issue was adopted into a concrete task this cycle, clear it from pending_issues
		if taskSpawned && chosen != nil && strings.EqualFold(strings.TrimSpace(iss), strings.TrimSpace(chosen.DetectedOpportunity)) {
			continue
		}
		add(iss)
	}

	if chosen != nil {
		switch action {
		case decision.ActionMonitor:
			add("[MONITOR] " + chosen.DetectedOpportunity)
		case decision.ActionDefer:
			add("[DEFERRED] " + chosen.DetectedOpportunity)
		case decision.ActionAskHuman:
			add("[WAITING_APPROVAL] " + chosen.DetectedOpportunity)
		}
	}
	return out
}

func buildPostCycleSnapshot(
	projectID string,
	projectPersisted bool,
	cycleID string,
	cycleNumber int64,
	activeTasks []task.Task,
	spawnedTaskID *string,
	dec *decision.Decision,
	pendingIssues []string,
	now time.Time,
) *continuity.ContextSnapshot {
	sanitizedEnv := continuity.SanitizeWorkspaceEnv(map[string]string{
		"CYCLE_ID":     cycleID,
		"PROJECT_ID":   projectID,
		"API_KEY_HINT": "redacted-check",
	})
	wsBytes, _ := json.Marshal(map[string]any{
		"cycle_id":     cycleID,
		"cycle_number": cycleNumber,
		"env":          sanitizedEnv,
	})

	taskIDs := make([]string, 0, len(activeTasks)+1)
	for _, tk := range activeTasks {
		taskIDs = append(taskIDs, string(tk.ID))
	}
	if spawnedTaskID != nil && *spawnedTaskID != "" {
		taskIDs = append(taskIDs, *spawnedTaskID)
	}
	tasksBytes, _ := json.Marshal(taskIDs)

	decBytes, _ := json.Marshal([]map[string]any{
		{
			"decision_id": dec.ID,
			"action_type": dec.ActionType,
			"governance":  dec.GovernanceResult,
		},
	})
	if pendingIssues == nil {
		pendingIssues = []string{}
	}
	issuesBytes, _ := json.Marshal(pendingIssues)

	scope := continuity.SnapshotScopeGlobal
	snapProjID := ""
	if projectPersisted && projectID != "" {
		scope = continuity.SnapshotScopeProject
		snapProjID = projectID
	}

	return &continuity.ContextSnapshot{
		ID:                  uuid.New().String(),
		Scope:               scope,
		ProjectID:           snapProjID,
		SnapshotType:        continuity.SnapshotTypePostCycle,
		WorkspaceStateJSON:  string(wsBytes),
		ActiveTasksJSON:     string(tasksBytes),
		RecentDecisionsJSON: string(decBytes),
		PendingIssuesJSON:   string(issuesBytes),
		CreatedAt:           now,
	}
}

func parseMetaMap(raw string) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

func nonEmptyStr(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func ptrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (c *Coordinator) executeAutonomousProblemOrPlan(
	ctx context.Context,
	projectID string,
	cycleID string,
	chosenInitiative *domainAutonomy.CandidateInitiative,
	effectiveRiskTier governance.RiskTier,
	requiresHumanApproval bool,
) (*string, string, error) {
	if c.deps.ProblemSvc == nil {
		return nil, "", errors.New("problem service not available")
	}

	meta := parseMetaMap(chosenInitiative.MetadataJSON)
	var prob *domainProblem.Problem
	var err error
	now := clock.Now(ctx)

	if chosenInitiative.ProblemID != nil && *chosenInitiative.ProblemID != "" {
		prob, err = c.deps.ProblemSvc.GetProblem(ctx, *chosenInitiative.ProblemID)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load existing problem: %w", err)
		}
	} else {
		// Auto-create new Problem entity from initiative metadata
		probTitle := chosenInitiative.DetectedOpportunity
		if t, ok := meta["problem_title"].(string); ok && t != "" {
			probTitle = t
		}
		probObj := chosenInitiative.Rationale
		if o, ok := meta["problem_objective"].(string); ok && o != "" {
			probObj = o
		}
		goalIDStr := ""
		if chosenInitiative.GoalID != nil {
			goalIDStr = *chosenInitiative.GoalID
		}

		prob, err = c.deps.ProblemSvc.CreateProblem(ctx, problemService.CreateProblemInput{
			ProjectID: projectID,
			GoalID:    goalIDStr,
			Title:     probTitle,
			Objective: probObj,
			Scope:     "AUTONOMY_CORE",
			Priority:  domainProblem.PriorityHigh,
		})
		if err != nil {
			return nil, "", fmt.Errorf("failed to create problem: %w", err)
		}
		chosenInitiative.ProblemID = &prob.ID
	}

	// 1. Initial Epistemic Analysis (if not already modeled)
	if prob.CurrentModelVersion == 0 {
		analysis := &domainProblem.AgentProblemAnalysis{
			Intent: domainProblem.IntentSummary{
				UserGoal:        prob.Objective,
				SuccessCriteria: []string{"Restore operational stability", "Remediate failure root causes"},
			},
			Context: domainProblem.ContextSummary{
				FoundResources: []string{fmt.Sprintf("Project:%s", projectID)},
				TrustedData:    []string{"Execution audit logs", "Continuity snapshot"},
			},
			ModelItems: []domainProblem.ModelItem{
				{
					ID:         fmt.Sprintf("fact-%s-1", prob.ID),
					Category:   domainProblem.CategoryKnownFact,
					Statement:  fmt.Sprintf("Problem %s triggered autonomously under project %s", prob.Title, projectID),
					Source:     "AUTONOMY_CYCLE_COORDINATOR",
					Confidence: 1.0,
					Status:     domainProblem.ItemStatusConfirmed,
					CreatedBy:  "autonomy_core",
					CreatedAt:  now,
				},
				{
					ID:         fmt.Sprintf("unknown-%s-1", prob.ID),
					Category:   domainProblem.CategoryUnknown,
					Statement:  "Exact environmental or logic edge case leading to failure",
					Confidence: 0.0,
					Status:     domainProblem.ItemStatusConfirmed,
					CreatedBy:  "autonomy_core",
					CreatedAt:  now,
				},
				{
					ID:         fmt.Sprintf("assumption-%s-1", prob.ID),
					Category:   domainProblem.CategoryAssumption,
					Statement:  "Decomposing remediation into root cause isolation and verified fix will restore project health",
					Confidence: 0.80,
					Status:     domainProblem.ItemStatusConfirmed,
					CreatedBy:  "autonomy_core",
					CreatedAt:  now,
				},
			},
		}
		if _, aErr := c.deps.ProblemSvc.ApplyAnalysis(ctx, prob.ID, analysis, "autonomy_core", ""); aErr != nil {
			return nil, "", fmt.Errorf("failed to apply problem analysis: %w", aErr)
		}
	}

	// 2. Generate Plan (if not already planned)
	wiAgent := chosenInitiative.TargetAgentID
	if wiAgent == "" {
		wiAgent = "coding-agent"
	}

	var activePlan *domainPlan.Plan
	if prob.CurrentPlanVersion == 0 {
		wi1ID := fmt.Sprintf("wi-%s-1", prob.ID)
		wi2ID := fmt.Sprintf("wi-%s-2", prob.ID)
		planIn := problemService.GeneratePlanInput{
			ProblemID: prob.ID,
			Objective: prob.Objective,
			Strategy:  "Two-stage remediation: 1. Diagnosis and Root-Cause Isolation -> 2. Verified Implementation",
			WorkItems: []domainPlan.WorkItemDef{
				{
					ID:            wi1ID,
					Title:         fmt.Sprintf("Diagnosis: %s", prob.Title),
					Description:   "Extract error logs, reproduce failure, and document isolated root cause",
					Stage:         "DIAGNOSIS",
					AssignedAgent: wiAgent,
					AcceptanceCriteria: []string{
						"Root cause identified and documented",
					},
					Status: domainPlan.WorkItemPending,
				},
				{
					ID:            wi2ID,
					Title:         fmt.Sprintf("Remediate: %s", prob.Title),
					Description:   "Implement fix, verify unit/integration tests, and ensure zero regression",
					Stage:         "REMEDIATION",
					AssignedAgent: wiAgent,
					Dependencies:  []string{wi1ID},
					AcceptanceCriteria: []string{
						"Regression validation passes cleanly",
					},
					Status: domainPlan.WorkItemPending,
				},
			},
			RequiresHumanApproval: requiresHumanApproval,
		}
		p, pErr := c.deps.ProblemSvc.GeneratePlan(ctx, planIn)
		if pErr != nil {
			return nil, "", fmt.Errorf("failed to generate plan: %w", pErr)
		}
		activePlan = p
	} else {
		p, pErr := c.deps.ProblemSvc.GetLatestPlan(ctx, prob.ID)
		if pErr != nil {
			return nil, "", fmt.Errorf("failed to get latest plan: %w", pErr)
		}
		activePlan = p
	}

	// 3. Approve and Materialize Work Items into Tasks
	if activePlan.Status == domainPlan.StatusProposed && !requiresHumanApproval {
		if _, appErr := c.deps.ProblemSvc.ApprovePlan(ctx, activePlan.ID, "AGENT:autonomy_core"); appErr == nil {
			activePlan.Status = domainPlan.StatusApproved
		}
	}

	if activePlan.Status == domainPlan.StatusApproved || activePlan.Status == domainPlan.StatusActive {
		tasks, mErr := c.deps.ProblemSvc.MaterializeAllWorkItems(ctx, activePlan.ID)
		if mErr != nil {
			return nil, "", fmt.Errorf("failed to materialize plan work items: %w", mErr)
		}
		if len(tasks) > 0 {
			firstTaskID := string(tasks[0].ID)
			summary := fmt.Sprintf("Autonomous Problem Escalation: Problem %s (%s) planned (v%d) and materialized %d task(s), dispatching %s",
				prob.ID, prob.Title, activePlan.Version, len(tasks), firstTaskID)
			return &firstTaskID, summary, nil
		}
	}

	// If waiting human approval
	summary := fmt.Sprintf("Autonomous Problem Escalation: Problem %s planned (v%d) awaiting approval", prob.ID, activePlan.Version)
	return nil, summary, nil
}
