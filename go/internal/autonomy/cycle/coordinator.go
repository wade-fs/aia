package cycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/autonomy/deliberation"
	"github.com/wade/aia/internal/autonomy/initiative"
	"github.com/wade/aia/internal/autonomy/observation"
	"github.com/wade/aia/internal/autonomy/recall"
	"github.com/wade/aia/internal/domain"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/policy"
	approvalService "github.com/wade/aia/internal/service/approval"
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
}

type CoordinatorDeps struct {
	ObserverRegistry *observation.Registry
	RecallEngine     *recall.Engine
	Discoverer       *initiative.Discoverer
	Deliberator      *deliberation.Deliberator
	AutonomyRepo     *repository.AutonomyRepository
	DecisionRepo     *repository.DecisionRepository
	ContinuityRepo   *repository.ContinuityRepository
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

func (c *Coordinator) RegisterPilot(p DomainPilotExecutor) {
	if p != nil {
		c.pilots = append(c.pilots, p)
	}
}

func (c *Coordinator) RunCycle(ctx context.Context, in RunInput) (*domainAutonomy.CycleOutcome, error) {
	// Pre-flight Independent Governance Gate: Global Emergency Stop
	if c.deps.GovernanceRepo != nil {
		if ctrl, err := c.deps.GovernanceRepo.GetControl(ctx); err == nil && ctrl != nil && ctrl.IsEmergencyStopped() {
			return nil, ErrEmergencyStopped
		}
	}

	now := time.Now().UTC()
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
	assessSummary := fmt.Sprintf("Assessed %d active goals, %d active tasks, %d pending issues, %d conflicts",
		len(recalled.ActiveGoals), len(recalled.ActiveTasks), len(recalled.PendingIssues), len(recalled.Conflicts))
	recordStep(3, "Assess", assessSummary, t3, map[string]any{
		"active_goals":   len(recalled.ActiveGoals),
		"active_tasks":   len(recalled.ActiveTasks),
		"pending_issues": len(recalled.PendingIssues),
		"conflicts":      len(recalled.Conflicts),
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
	})

	delibRecord := &domainAutonomy.DeliberationRecord{
		ID:                   uuid.New().String(),
		CycleID:              cycleID,
		OptionsEvaluatedJSON: delibRes.OptionsEvaluatedJSON,
		CostBenefitAnalysis:  delibRes.CostBenefitAnalysis,
		RiskAssessment:       delibRes.RiskAssessment,
		RecommendedAction:    delibRes.RecommendedAction,
		CreatedAt:            time.Now().UTC(),
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
		CreatedAt:                  time.Now().UTC(),
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
		} else if chosenInitiative != nil && c.deps.TaskSvc != nil {
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
	completedAt := time.Now().UTC()

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
		OccurredAt:    time.Now().UTC(),
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
