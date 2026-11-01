package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/autonomy"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	domainGoal "github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/governance"
	domainInv "github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/domain/knowledge"
	"github.com/wade/aia/internal/domain/task"
	goalService "github.com/wade/aia/internal/service/goal"
	"github.com/wade/aia/internal/web/console"
)

type mockTestObserver struct {
	observations []domainAutonomy.Observation
}

func (m *mockTestObserver) Observe(ctx context.Context, projectID string) ([]domainAutonomy.Observation, error) {
	return m.observations, nil
}

func TestPhase3_AutonomyCore_MilestonesM3AndM4(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// 1. Milestone M3: Rational NO_ACTION & Immutable Event Chain Integrity
	// -------------------------------------------------------------------------
	t.Run("MilestoneM3_RationalNoAction_And_HashChain", func(t *testing.T) {
		outcome, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   "default",
			TriggerType: continuity.TriggerScheduled,
			Summary:     "Idle environment test",
		})
		if err != nil {
			t.Fatalf("failed to run idle cycle: %v", err)
		}

		if outcome.Decision.ActionType != decision.ActionNoAction {
			t.Fatalf("expected NO_ACTION, got %s", outcome.Decision.ActionType)
		}
		if outcome.TaskID != nil {
			t.Errorf("expected zero tasks spawned on NO_ACTION, got %s", *outcome.TaskID)
		}
		if outcome.Deliberation == nil || outcome.Deliberation.RecommendedAction != decision.ActionNoAction {
			t.Fatalf("expected deliberation recommendation NO_ACTION")
		}
		if len(outcome.StepTraces) != 11 {
			t.Errorf("expected all 11 autonomy step traces, got %d", len(outcome.StepTraces))
		}
		if outcome.Verification == nil || !outcome.Verification.Passed {
			t.Errorf("expected Step 9 verification to pass")
		}
		if outcome.PostCycleSnapshotID == nil || *outcome.PostCycleSnapshotID == "" {
			t.Errorf("expected Step 11 POST_CYCLE snapshot to be persisted")
		}

		// Verify L1 immutable event_history hash chain for this cycle
		valid, err := app.EventHistoryRepo.VerifyAggregateChainIntegrity(ctx, "AUTONOMY_CYCLE", outcome.CycleID)
		if err != nil || !valid {
			t.Fatalf("expected valid SHA-256 event chain on AUTONOMY_CYCLE, valid=%v err=%v", valid, err)
		}

		// Verify persisted Decision & Sanitized Snapshot
		persistedDec, err := app.DecisionRepo.Get(ctx, outcome.Decision.ID)
		if err != nil || persistedDec.ActionType != decision.ActionNoAction {
			t.Fatalf("persisted decision mismatch: err=%v, dec=%+v", err, persistedDec)
		}
		snap, err := app.ContinuityRepo.GetSnapshot(ctx, *outcome.PostCycleSnapshotID)
		if err != nil || !strings.Contains(snap.WorkspaceStateJSON, "[REDACTED]") {
			t.Errorf("expected sanitized POST_CYCLE snapshot, got err=%v json=%s", err, snap.WorkspaceStateJSON)
		}
	})

	// -------------------------------------------------------------------------
	// 2. Milestone M3: Unprompted Goal Gap Discovery & Second-Cycle Deduplication
	// -------------------------------------------------------------------------
	t.Run("MilestoneM3_UnpromptedGoalGapDiscovery_And_Deduplication", func(t *testing.T) {
		g, err := app.GoalSvc.Create(ctx, goalService.CreateGoalInput{
			ProjectID:         "default",
			Name:              "Backend Unit Test Coverage >= 85%",
			Description:       "Ensure core Go packages maintain high unit test coverage",
			Level:             domainGoal.LevelTactical,
			Status:            domainGoal.StatusActive,
			TargetMetricJSON:  `{"coverage_pct":85}`,
			CurrentMetricJSON: `{"coverage_pct":55}`,
			Priority:          8,
		})
		if err != nil {
			t.Fatalf("failed to create active goal: %v", err)
		}

		// Cycle 1: Zero external user command or observer event -> discovers GOAL_GAP autonomously
		out1, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   "default",
			TriggerType: continuity.TriggerScheduled,
			Summary:     "Autonomous goal gap review #1",
		})
		if err != nil {
			t.Fatalf("cycle #1 failed: %v", err)
		}
		if out1.Decision.ActionType != decision.ActionAct {
			t.Fatalf("expected ACT on unmet goal gap, got %s", out1.Decision.ActionType)
		}
		if out1.TaskID == nil || *out1.TaskID == "" {
			t.Fatalf("expected decomposed task to be created for goal gap")
		}

		links, err := app.GoalSvc.ListDecomposedTasks(ctx, g.ID)
		if err != nil || len(links) != 1 || links[0].TaskID != *out1.TaskID {
			t.Fatalf("expected goal_task_links to link goal %s -> task %s, got %+v", g.ID, *out1.TaskID, links)
		}

		// Cycle 2: While decomposed task is still READY, Discoverer deduplicates and produces NO_ACTION
		out2, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   "default",
			TriggerType: continuity.TriggerScheduled,
			Summary:     "Autonomous goal gap review #2 (deduplication check)",
		})
		if err != nil {
			t.Fatalf("cycle #2 failed: %v", err)
		}
		if out2.Decision.ActionType != decision.ActionNoAction {
			t.Errorf("expected NO_ACTION due to active decomposed task deduplication, got %s", out2.Decision.ActionType)
		}
		if out2.TaskID != nil {
			t.Errorf("expected zero duplicate tasks spawned, got %s", *out2.TaskID)
		}

		// Complete the task and mark goal ACHIEVED to keep default project clean for subsequent subtests
		_ = app.TaskSvc.UpdateStatus(ctx, *out1.TaskID, task.StatusCompleted)
		_ = app.GoalSvc.TransitionStatus(ctx, g.ID, domainGoal.StatusAchieved, "Coverage target met in test", out1.Decision.ID)
	})

	// -------------------------------------------------------------------------
	// 3. Milestone M3: Diversified Decisions (MONITOR, DEFER, ASK_HUMAN)
	// -------------------------------------------------------------------------
	t.Run("MilestoneM3_DiversifiedDecisions_Monitor_Defer_AskHuman", func(t *testing.T) {
		// 3A. MONITOR on low-confidence early drift
		driftObs := &mockTestObserver{
			observations: []domainAutonomy.Observation{
				{
					Source:     "LATENCY_WATCHER",
					Category:   domainAutonomy.ObsCategoryDrift,
					Confidence: domainAutonomy.ConfidencePotential,
					Severity:   4,
					Summary:    "Minor SQLite WAL checkpoint drift (+3.2ms)",
					Timestamp:  time.Now().UTC(),
				},
			},
		}
		app.AutonomyEngine.RegisterObserver("drift_test", driftObs)
		monOut, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   "default",
			TriggerType: continuity.TriggerDriftDetected,
			Summary:     "Evaluate subtle drift",
		})
		app.AutonomyEngine.UnregisterObserver("drift_test")
		if err != nil {
			t.Fatalf("MONITOR cycle failed: %v", err)
		}
		if monOut.Decision.ActionType != decision.ActionMonitor {
			t.Fatalf("expected MONITOR decision, got %s", monOut.Decision.ActionType)
		}
		if monOut.TaskID != nil {
			t.Errorf("expected zero tasks on MONITOR, got %s", *monOut.TaskID)
		}

		// 3B. DEFER when candidate has a blocked dependency
		deferObs := &mockTestObserver{
			observations: []domainAutonomy.Observation{
				{
					Source:     "DEPENDENCY_SCANNER",
					Category:   domainAutonomy.ObsCategoryOpportunity,
					Confidence: domainAutonomy.ConfidenceConfirmed,
					Severity:   6,
					Summary:    "Upgrade upstream schema adapter once v2 endpoint unlocks",
					Data:       map[string]any{"blocked_dependency": true},
					Timestamp:  time.Now().UTC(),
				},
			},
		}
		app.AutonomyEngine.RegisterObserver("defer_test", deferObs)
		deferOut, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   "default",
			TriggerType: continuity.TriggerScheduled,
			Summary:     "Evaluate blocked opportunity",
		})
		app.AutonomyEngine.UnregisterObserver("defer_test")
		if err != nil {
			t.Fatalf("DEFER cycle failed: %v", err)
		}
		if deferOut.Decision.ActionType != decision.ActionDefer {
			t.Fatalf("expected DEFER decision, got %s", deferOut.Decision.ActionType)
		}

		// 3C. ASK_HUMAN when L3 Knowledge Store contains a CONFLICTED claim
		conflictClaim := &knowledge.Claim{
			ID:              uuid.New().String(),
			Statement:       "Project deployment target is air-gapped vs cloud-hosted",
			AssertionStatus: knowledge.AssertionConflicted,
			CurrentVersion:  1,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		if err := app.KnowledgeRepo.CreateClaim(ctx, conflictClaim); err != nil {
			t.Fatalf("failed to seed conflicted claim: %v", err)
		}

		askOut, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   "default",
			TriggerType: continuity.TriggerScheduled,
			Summary:     "Reconcile conflicted claim",
		})
		if err != nil {
			t.Fatalf("ASK_HUMAN cycle failed: %v", err)
		}
		if askOut.Decision.ActionType != decision.ActionAskHuman {
			t.Fatalf("expected ASK_HUMAN decision on conflicted claim, got %s", askOut.Decision.ActionType)
		}
		if askOut.GovernanceResult != governance.GovernanceActionRequiredApproval {
			t.Errorf("expected REQUIRED_APPROVAL on ASK_HUMAN, got %s", askOut.GovernanceResult)
		}
		if askOut.ApprovalID == nil || *askOut.ApprovalID == "" {
			t.Fatalf("expected scope-bound approval request on ASK_HUMAN")
		}
		// Resolve conflicted claim so subsequent subtests start clean
		_ = app.KnowledgeRepo.UpdateClaimStatus(ctx, conflictClaim.ID, knowledge.AssertionAccepted, true)
	})

	// -------------------------------------------------------------------------
	// 4. Milestone M3: Independent Governance Gate (BLOCKED, REQUIRED_APPROVAL, EMERGENCY_STOP)
	// -------------------------------------------------------------------------
	t.Run("MilestoneM3_IndependentGovernanceGate", func(t *testing.T) {
		// 4A. BLOCKED: Real-money trade attempt is intercepted and blocked by Policy Evaluator
		realMoneyObs := &mockTestObserver{
			observations: []domainAutonomy.Observation{
				{
					Source:        "ROGUE_SIGNAL",
					Category:      domainAutonomy.ObsCategoryOpportunity,
					Confidence:    domainAutonomy.ConfidenceConfirmed,
					Severity:      9,
					TargetAgentID: "investment-agent",
					Summary:       "Attempt live brokerage wire order",
					Data: map[string]any{
						"operation_type": "FINANCIAL_TRANSACTION",
						"is_real_money":  true,
						"amount":         50000.0,
					},
					Timestamp: time.Now().UTC(),
				},
			},
		}
		app.AutonomyEngine.RegisterObserver("real_money_test", realMoneyObs)
		blkOut, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID: "default",
			Summary:   "Governance block test",
		})
		app.AutonomyEngine.UnregisterObserver("real_money_test")
		if err != nil {
			t.Fatalf("blocked cycle returned unexpected error: %v", err)
		}
		if blkOut.GovernanceResult != governance.GovernanceActionBlocked {
			t.Fatalf("expected BLOCKED governance result, got %s", blkOut.GovernanceResult)
		}
		if blkOut.TaskID != nil {
			t.Fatalf("expected zero tasks created when GovernanceResult == BLOCKED, got %s", *blkOut.TaskID)
		}

		// 4B. REQUIRED_APPROVAL: High-risk system command pauses task in WAITING_APPROVAL
		sysCmdObs := &mockTestObserver{
			observations: []domainAutonomy.Observation{
				{
					Source:        "INFRA_WATCH",
					Category:      domainAutonomy.ObsCategoryAnomaly,
					Confidence:    domainAutonomy.ConfidenceConfirmed,
					Severity:      9,
					TargetAgentID: "coding-agent",
					Summary:       "Restart local collector systemd unit",
					Data: map[string]any{
						"operation_type": "SYSTEM_COMMAND",
						"command":        "systemctl restart aia-collector",
					},
					Timestamp: time.Now().UTC(),
				},
			},
		}
		app.AutonomyEngine.RegisterObserver("sys_cmd_test", sysCmdObs)
		apprOut, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID: "default",
			Summary:   "High-risk approval gate test",
		})
		app.AutonomyEngine.UnregisterObserver("sys_cmd_test")
		if err != nil {
			t.Fatalf("approval cycle failed: %v", err)
		}
		if apprOut.GovernanceResult != governance.GovernanceActionRequiredApproval {
			t.Fatalf("expected REQUIRED_APPROVAL, got %s", apprOut.GovernanceResult)
		}
		if apprOut.TaskID == nil || apprOut.ApprovalID == nil {
			t.Fatalf("expected both spawned task and pending approval")
		}
		pausedTask, err := app.TaskSvc.Get(ctx, *apprOut.TaskID)
		if err != nil || pausedTask.Status != task.StatusWaitingApproval {
			t.Fatalf("expected task in WAITING_APPROVAL, got err=%v status=%v", err, pausedTask.Status)
		}
		// Agent self-approval MUST be forbidden; Human approval transitions task to READY
		if err := app.ApprovalSvc.Approve(ctx, *apprOut.ApprovalID, "AGENT:coding-agent", "self", "AGENT"); err == nil {
			t.Fatalf("expected agent self-approval to fail")
		}
		if err := app.ApprovalSvc.Approve(ctx, *apprOut.ApprovalID, "owner_wade", "Approved safe restart", "HUMAN"); err != nil {
			t.Fatalf("human approval failed: %v", err)
		}
		readyTask, _ := app.TaskSvc.Get(ctx, *apprOut.TaskID)
		if readyTask.Status != task.StatusReady {
			t.Errorf("expected task READY after human approval, got %s", readyTask.Status)
		}
		_ = app.TaskSvc.UpdateStatus(ctx, *apprOut.TaskID, task.StatusCompleted)

		// 4C. EMERGENCY_STOP halts Autonomy Cycle execution immediately
		human := governance.Actor{Type: "HUMAN", ID: "owner_wade"}
		if err := app.WorkerMgr.TriggerEmergencyStop(ctx, human, "Phase 3 emergency drill"); err != nil {
			t.Fatalf("TriggerEmergencyStop failed: %v", err)
		}
		_, stopErr := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{ProjectID: "default"})
		if !errors.Is(stopErr, autonomy.ErrAutonomyEmergencyStopped) {
			t.Fatalf("expected ErrAutonomyEmergencyStopped during emergency stop, got %v", stopErr)
		}
		if err := app.WorkerMgr.ResumeDispatch(ctx, human, true); err != nil {
			t.Fatalf("ResumeDispatch failed: %v", err)
		}
	})

	// -------------------------------------------------------------------------
	// 5. Milestone M4: Investment Agent Multi-Cycle Closed-Loop Pilot (Paper Trading)
	//    Cycle 1 (Initial Thesis & Virtual BUY) -> Cycle 2 (Calm Market NO_ACTION) ->
	//    Cycle 3 (Material >=3% Price Shift -> Rebalance Virtual SELL)
	// -------------------------------------------------------------------------
	t.Run("MilestoneM4_InvestmentAgent_MultiCycleClosedLoopPilot", func(t *testing.T) {
		if _, err := app.InvSvc.EnsureDefaultProfile(ctx); err != nil {
			t.Fatalf("EnsureDefaultProfile failed: %v", err)
		}
		_ = app.InvSvc.EnsureDataSources(ctx)

		// Establish baseline HOLD thesis on seeded 2330 @ 1000.0 so we isolate 0050 in Cycle 1
		_, err := app.InvSvc.ProcessAgyDecision(ctx, domainInv.AgyDecisionInput{
			Symbol:          "2330",
			Exchange:        "TWSE",
			Action:          "HOLD",
			Confidence:      0.85,
			Reasoning:       []string{"Core semiconductor holding at fair value"},
			ExpectedHorizon: "20D",
			StrategyVersion: "pilot-seed",
		}, 1000.0)
		if err != nil {
			t.Fatalf("failed to seed baseline 2330 HOLD decision: %v", err)
		}

		// Ingest local TWSE market bar for 0050 at NT$180.0 (no prior decision on 0050 yet)
		inst0050, err := app.InvRepo.GetInstrumentBySymbol(ctx, "TWSE", "0050")
		if err != nil {
			t.Fatalf("failed to get 0050 instrument: %v", err)
		}
		inst2330, err := app.InvRepo.GetInstrumentBySymbol(ctx, "TWSE", "2330")
		if err != nil {
			t.Fatalf("failed to get 2330 instrument: %v", err)
		}

		saveBar := func(instID string, closePrice float64) {
			b, _ := json.Marshal(domainInv.MarketBarValue{
				Open:   closePrice,
				High:   closePrice,
				Low:    closePrice,
				Close:  closePrice,
				Volume: 12000000,
			})
			if err := app.InvRepo.SaveMarketData(ctx, &domainInv.MarketData{
				ID:           uuid.New().String(),
				InstrumentID: instID,
				DataType:     domainInv.MarketDataDailyBar,
				Period:       "1D",
				ValueJSON:    string(b),
				SourceID:     "ds-twse-openapi",
				FetchedAt:    time.Now().UTC(),
				EffectiveAt:  time.Now().UTC(),
			}); err != nil {
				t.Fatalf("SaveMarketData failed: %v", err)
			}
		}

		saveBar(inst0050.ID, 180.0)
		saveBar(inst2330.ID, 1000.0)

		// Cycle 1: Detects missing initial thesis on 0050 -> ACT -> generates & settles virtual BUY of 1,000 shares @ 180.0
		c1, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   "default",
			TriggerType: continuity.TriggerScheduled,
			Summary:     "Investment Pilot Cycle 1: Initial allocation",
		})
		if err != nil {
			t.Fatalf("Investment Cycle 1 failed: %v", err)
		}
		if c1.Decision.ActionType != decision.ActionAct {
			t.Fatalf("Cycle 1: expected ACT for initial 0050 thesis, got %s (%s)", c1.Decision.ActionType, c1.Decision.Justification)
		}

		snapAfterC1, positionsAfterC1, err := app.InvSvc.GetSnapshot(ctx)
		if err != nil {
			t.Fatalf("GetSnapshot after Cycle 1 failed: %v", err)
		}
		if snapAfterC1.AvailableCash >= 600000 {
			t.Errorf("expected virtual cash to be debited after 0050 paper BUY, got %d", snapAfterC1.AvailableCash)
		}
		found0050 := false
		for _, pos := range positionsAfterC1 {
			if pos.InstrumentID == inst0050.ID && pos.Quantity == 1000 {
				found0050 = true
			}
		}
		if !found0050 {
			t.Fatalf("expected virtual position of 1,000 shares in 0050 after Cycle 1, got %+v", positionsAfterC1)
		}

		// Cycle 2: Calm market (< 3% price move, 0 new disclosures) -> Rational NO_ACTION
		saveBar(inst0050.ID, 182.0)  // +1.11% (< 3.0%)
		saveBar(inst2330.ID, 1012.0) // +1.20% (< 3.0%)

		c2, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   "default",
			TriggerType: continuity.TriggerScheduled,
			Summary:     "Investment Pilot Cycle 2: Calm market check",
		})
		if err != nil {
			t.Fatalf("Investment Cycle 2 failed: %v", err)
		}
		if c2.Decision.ActionType != decision.ActionNoAction {
			t.Fatalf("Cycle 2: expected NO_ACTION on <3%% calm market fluctuation, got %s (%s)", c2.Decision.ActionType, c2.Decision.Justification)
		}
		if c2.TaskID != nil {
			t.Errorf("Cycle 2: expected zero tasks spawned on calm market, got %s", *c2.TaskID)
		}

		// Cycle 3: Material price shock on 2330 (+6.5% >= 3.0% threshold) -> ACT -> Rebalance virtual SELL
		saveBar(inst2330.ID, 1065.0) // +6.50% vs 1000.0 reference

		c3, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   "default",
			TriggerType: continuity.TriggerEventTrigger,
			Summary:     "Investment Pilot Cycle 3: Material price surge",
		})
		if err != nil {
			t.Fatalf("Investment Cycle 3 failed: %v", err)
		}
		if c3.Decision.ActionType != decision.ActionAct {
			t.Fatalf("Cycle 3: expected ACT on +6.5%% material shift, got %s (%s)", c3.Decision.ActionType, c3.Decision.Justification)
		}
		snapAfterC3, _, _ := app.InvSvc.GetSnapshot(ctx)
		if snapAfterC3.AvailableCash <= snapAfterC1.AvailableCash {
			t.Errorf("expected virtual cash to increase after overweight 2330 virtual SELL rebalance (before=%d, after=%d)",
				snapAfterC1.AvailableCash, snapAfterC3.AvailableCash)
		}
	})

	// -------------------------------------------------------------------------
	// 6. Strangler Fig Delegation & Web Console Endpoints (/api/v1/goals & /api/v1/cycles)
	// -------------------------------------------------------------------------
	t.Run("StranglerFig_PersonalAgent_And_WebConsoleEndpoints", func(t *testing.T) {
		paOut, err := app.PersonalAgent.RunAutonomousCycle(ctx, "default", "PersonalAgent delegated cycle")
		if err != nil || paOut == nil {
			t.Fatalf("PersonalAgent.RunAutonomousCycle failed: %v", err)
		}

		mux := http.NewServeMux()
		console.RegisterConsoleRoutes(mux, &console.ConsoleServices{
			TaskSvc:        app.TaskSvc,
			AgentSvc:       app.AgentSvc,
			MemorySvc:      app.MemorySvc,
			AuditRepo:      app.AuditRepo,
			InvSvc:         app.InvSvc,
			DB:             app.DB,
			Provider:       app.Provider,
			WorkerMgr:      app.WorkerMgr,
			ApprovalSvc:    app.ApprovalSvc,
			GovernanceSvc:  app.GovernanceSvc,
			GoalSvc:        app.GoalSvc,
			AutonomyEngine: app.AutonomyEngine,
			AutonomyRepo:   app.AutonomyRepo,
			ContinuityRepo: app.ContinuityRepo,
		})

		// POST /api/v1/goals
		reqGoal := httptest.NewRequest(http.MethodPost, "/api/v1/goals",
			strings.NewReader(`{"project_id":"default","name":"Zero Regression Releases","level":"STRATEGIC","priority":9}`))
		wGoal := httptest.NewRecorder()
		mux.ServeHTTP(wGoal, reqGoal)
		if wGoal.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created from POST /api/v1/goals, got %d: %s", wGoal.Code, wGoal.Body.String())
		}

		// GET /api/v1/cycles
		reqCycles := httptest.NewRequest(http.MethodGet, "/api/v1/cycles?limit=10", nil)
		wCycles := httptest.NewRecorder()
		mux.ServeHTTP(wCycles, reqCycles)
		if wCycles.Code != http.StatusOK {
			t.Fatalf("expected 200 OK from GET /api/v1/cycles, got %d: %s", wCycles.Code, wCycles.Body.String())
		}
	})
}
