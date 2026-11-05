package integration

// Seven-Day Autonomy Acceptance Scenario
//
// Day 1  User creates a Project with a Goal and initial conditions; AIA builds a Plan and Tasks and starts executing.
// Day 2  The user issues no new command; AIA wakes itself, restores the Project context, checks progress,
//        and finds unfinished work.
// Day 3  A Task fails; AIA must preserve the failure evidence, analyse the cause, and decide to retry or re-plan.
// Day 4  AIA is stopped and restarted; Goal / Decision / Evidence / Task state must be restored, not restarted.
// Day 5  New evidence contradicts the original hypothesis; AIA must re-evaluate the original decision.
// Day 6  AIA corrects the Plan, runs the corrected task, verifies the result and records what it learned.
// Day 7  AIA finds a secondary goal no longer has value and proposes an adjustment; governance decides.
//
// Design notes
//  - The scenario is compressed: "days" are phases, and the autonomy worker ticks every second.
//  - Each day uses t.Errorf (not Fatalf) so one run reports every gap instead of stopping at the first.
//  - Facts that a day needs but an earlier failed day should have produced are SEEDED (and logged), so a
//    failure on Day N does not hide the result of Day N+1.
//  - Assertions define the CONTRACT the scenario asks for. Where a day fails, the message names the missing link.
//  - NOTE: this file was written against the aia_1.tgz sources but has not been compiled or run by its author.

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/app"
	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/knowledge"
	"github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/provider/mock"
	goalService "github.com/wade/aia/internal/service/goal"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

const (
	sdWorkerTick   = 1 * time.Second        // autonomy worker interval (kept >= 1s to stay inside the daily cycle budget)
	sdPollInterval = 150 * time.Millisecond // condition polling step
	sdWaitLong     = 6 * time.Second        // window for Day 1 / Day 2
	sdWaitShort    = 4 * time.Second        // window for Day 3 / 5 / 6 / 7
)

type sdState struct {
	app     *app.App
	cfgPath string
	tmpDir  string

	projID    string
	goalID    string
	subGoalID string

	failedTaskID   string
	inflightTaskID string

	srcID    string
	eviID    string
	claimID  string
	decID    string
	eviNewID string

	learningsAfterDay3 int
	day5At             time.Time
	plansAtDay5        int
	changesAtDay5      int
	learningsAtDay5    int
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func sdPoll(d time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(d)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(sdPollInterval)
	}
}

func sdStartWorker(t *testing.T, ctx context.Context, a *app.App) {
	t.Helper()
	a.WorkerMgr.SetAutonomyEngine(a.AutonomyEngine, sdWorkerTick)
	if err := a.WorkerMgr.Start(ctx); err != nil {
		t.Fatalf("cannot start worker manager: %v", err)
	}
}

// sdForceCycle runs one explicit event-triggered cycle on the scenario project. It is used to separate
// "AIA did not wake up" (Day 2 problem) from "AIA woke up but did not react" (Day 3/5/7 problem).
func sdForceCycle(ctx context.Context, t *testing.T, st *sdState, summary string) {
	t.Helper()
	_, err := st.app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   st.projID,
		TriggerType: continuity.TriggerEventTrigger,
		Summary:     summary,
	})
	if err != nil {
		t.Logf("forced cycle returned: %v", err)
	}
}

func sdTasks(ctx context.Context, st *sdState) []task.Task {
	tasks, err := st.app.TaskSvc.List(ctx, repository.TaskFilter{ProjectID: domain.ID(st.projID)})
	if err != nil {
		return nil
	}
	return tasks
}

func sdPlanStats(ctx context.Context, st *sdState) (plans int, changes int) {
	probs, err := st.app.ProblemSvc.ListProblems(ctx, st.projID)
	if err != nil {
		return 0, 0
	}
	for _, p := range probs {
		pls, err := st.app.ProblemSvc.ListPlans(ctx, p.ID)
		if err != nil {
			continue
		}
		plans += len(pls)
		for _, pl := range pls {
			chs, err := st.app.ProblemSvc.ListPlanChanges(ctx, pl.ID)
			if err == nil {
				changes += len(chs)
			}
		}
	}
	return plans, changes
}

func sdLearnings(ctx context.Context, st *sdState) []*learning.LearningRecord {
	recs, err := st.app.LearningRepo.ListLearningRecords(ctx, st.projID, "", 50)
	if err != nil {
		return nil
	}
	return recs
}

func sdDecisions(ctx context.Context, st *sdState, projectID string) []*decision.Decision {
	decs, err := st.app.DecisionRepo.ListByProject(ctx, projectID, 100)
	if err != nil {
		return nil
	}
	return decs
}

// sdSeedHypothesis records the "original hypothesis": a source, supporting evidence, an accepted claim and a
// decision that relies on that evidence. It stands in for what Days 1-3 should have produced on their own.
func sdSeedHypothesis(t *testing.T, ctx context.Context, st *sdState) {
	t.Helper()
	now := time.Now().UTC()

	st.srcID = "src-sd-official"
	st.eviID = "evi-sd-original"
	st.claimID = "claim-sd-hypothesis"
	st.decID = "dec-sd-original"

	if err := st.app.KnowledgeSvc.RegisterSource(ctx, &knowledge.Source{
		ID:                   st.srcID,
		SourceType:           knowledge.SourceOfficial,
		URI:                  "https://example.com/sd/official",
		AuthorityScore:       0.95,
		DirectnessScore:      0.9,
		TimelinessScore:      0.9,
		IndependenceScore:    1.0,
		ReproducibilityScore: 1.0,
		CreatedAt:            now,
	}); err != nil {
		t.Fatalf("seed: register source: %v", err)
	}

	if err := st.app.KnowledgeSvc.RecordEvidence(ctx, &knowledge.Evidence{
		ID:             st.eviID,
		SourceID:       st.srcID,
		ContentSnippet: "Benchmark latency is stable at 40ms across 30 days",
		ContentHash:    "sd-hash-original-001",
		CaptureMethod:  knowledge.CaptureAPIJSON,
		Status:         knowledge.EvidenceValidated,
		CapturedAt:     now,
		CreatedAt:      now,
	}); err != nil {
		t.Fatalf("seed: record evidence: %v", err)
	}

	if err := st.app.KnowledgeSvc.CreateClaim(ctx, &knowledge.Claim{
		ID:              st.claimID,
		Statement:       "Latency is stable, so plan A (no re-architecture) is sufficient",
		ClaimType:       knowledge.ClaimHistoricalSkeleton,
		AssertionStatus: knowledge.AssertionAccepted,
		IsKnowledge:     true,
	}, []*knowledge.ClaimEvidenceLink{{
		ClaimID:          st.claimID,
		EvidenceID:       st.eviID,
		SupportType:      knowledge.SupportTypeSupports,
		ConfidenceWeight: 1.0,
	}}); err != nil {
		t.Fatalf("seed: create claim: %v", err)
	}

	pid := st.projID
	if err := st.app.DecisionRepo.Create(ctx, &decision.Decision{
		ID:               st.decID,
		ProjectID:        &pid,
		ActionType:       decision.ActionNoAction,
		IntentSummary:    "Keep plan A: no re-architecture needed because latency is stable",
		Justification:    "Relies on claim " + st.claimID + " supported by evidence " + st.eviID,
		GovernanceResult: decision.GovernanceAllowed,
		CreatedAt:        now,
	}, []*decision.DecisionEvidenceLink{{
		DecisionID:    st.decID,
		EvidenceID:    st.eviID,
		RelevanceNote: "Original hypothesis evidence",
	}}); err != nil {
		t.Fatalf("seed: create decision: %v", err)
	}
	t.Logf("SEED: original hypothesis (claim %s, evidence %s, decision %s) recorded manually", st.claimID, st.eviID, st.decID)
}

// ---------------------------------------------------------------------------
// the scenario
// ---------------------------------------------------------------------------

func TestSevenDayAutonomyScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("seven-day scenario takes ~40s; skipped in -short mode")
	}

	a, tmpDir := createTestApp(t)
	ctx := context.Background()
	st := &sdState{app: a, cfgPath: filepath.Join(tmpDir, "config.yaml"), tmpDir: tmpDir}
	defer func() {
		st.app.WorkerMgr.Stop()
		_ = st.app.Close()
	}()

	// ----------------------------------------------------------------- Day 1
	t.Run("Day1_CreateGoal_AutoPlanAndTasks", func(t *testing.T) {
		proj, err := st.app.ProjectSvc.Create(ctx, "Seven Day Pilot", "7-day autonomy acceptance scenario", filepath.Join(st.tmpDir, "pilot"))
		if err != nil {
			t.Fatalf("create project: %v", err)
		}
		st.projID = string(proj.ID)

		g, err := st.app.GoalSvc.Create(ctx, goalService.CreateGoalInput{
			ProjectID:        st.projID,
			Name:             "Deliver pilot report",
			Description:      "Produce the pilot benchmark report within the budget",
			Level:            goal.LevelTactical,
			Status:           goal.StatusActive,
			Priority:         8,
			TargetMetricJSON: `{"initial_conditions":{"budget_hours":20,"deadline_days":7}}`,
		})
		if err != nil {
			t.Fatalf("create goal: %v", err)
		}
		st.goalID = g.ID

		parent := st.goalID
		sub, err := st.app.GoalSvc.Create(ctx, goalService.CreateGoalInput{
			ProjectID:    st.projID,
			ParentGoalID: &parent,
			Name:         "Collect supplementary benchmark data",
			Description:  "Secondary goal; only valuable while plan A (latency is stable) holds",
			Level:        goal.LevelOperational,
			Status:       goal.StatusActive,
			Priority:     4,
		})
		if err != nil {
			t.Fatalf("create secondary goal: %v", err)
		}
		st.subGoalID = sub.ID

		// From here on the user gives no further commands. AIA is expected to plan and start working.
		sdStartWorker(t, ctx, st.app)

		sdPoll(sdWaitLong, func() bool {
			plans, _ := sdPlanStats(ctx, st)
			return len(sdTasks(ctx, st)) > 0 && plans > 0
		})

		plans, _ := sdPlanStats(ctx, st)
		if plans == 0 {
			t.Errorf("Day1: no Plan exists for the project; nothing turned the new Goal into a Plan without further user commands")
		}
		if len(sdTasks(ctx, st)) == 0 {
			t.Errorf("Day1: no Task exists for the project; nothing decomposed the new Goal into Tasks (goal decompose is manual)")
		}

		sdSeedHypothesis(t, ctx, st)
	})

	// ----------------------------------------------------------------- Day 2
	t.Run("Day2_AutonomousWake_FindUnfinishedWork", func(t *testing.T) {
		start := time.Now().UTC()

		woke := sdPoll(sdWaitLong, func() bool {
			for _, d := range sdDecisions(ctx, st, st.projID) {
				if d.CreatedAt.After(start.Add(-1*time.Second)) && d.ID != st.decID {
					return true
				}
			}
			return false
		})
		if !woke {
			defaultCount := len(sdDecisions(ctx, st, "default"))
			t.Errorf("Day2: no autonomous cycle ran for project %s without a user command "+
				"(decisions recorded for project \"default\": %d; the autonomy worker is alive but only cycles \"default\")",
				st.projID, defaultCount)
		}

		inits, err := st.app.AutonomyRepo.ListRecentInitiativesByProject(ctx, st.projID, 20)
		if err != nil || len(inits) == 0 {
			t.Errorf("Day2: no initiative recorded for project %s; unfinished work was not discovered (err=%v)", st.projID, err)
		}
	})

	// ----------------------------------------------------------------- Day 3
	t.Run("Day3_TaskFailure_EvidenceRCA_RetryOrReplan", func(t *testing.T) {
		// Stop workers while creating the failing task so the task worker cannot grab it first.
		st.app.WorkerMgr.Stop()

		tk, err := st.app.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
			ProjectID:   domain.ID(st.projID),
			AgentID:     "coding-agent",
			Title:       "Day3 data export",
			Description: "Export benchmark data for the pilot report",
		})
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		st.failedTaskID = tk.ID
		if err := st.app.TaskSvc.UpdateStatus(ctx, domain.ID(tk.ID), task.StatusRunning); err != nil {
			t.Fatalf("task -> RUNNING: %v", err)
		}

		_, changesBefore := sdPlanStats(ctx, st)
		learningsBefore := len(sdLearnings(ctx, st))
		failAt := time.Now().UTC()

		if err := st.app.TaskSvc.UpdateStatus(ctx, domain.ID(tk.ID), task.StatusFailed); err != nil {
			t.Fatalf("task -> FAILED: %v", err)
		}
		sdStartWorker(t, ctx, st.app)

		analysed := func() bool {
			for _, r := range sdLearnings(ctx, st) {
				if r.RCAID != nil && r.OutcomeEvaluationID != "" {
					return true
				}
			}
			return false
		}

		natural := sdPoll(sdWaitShort, analysed)
		onlyAfterManual := false
		if !natural {
			t.Logf("Day3: no reaction within the window; forcing one event-triggered cycle to isolate the gap")
			sdForceCycle(ctx, t, st, "task failed: "+st.failedTaskID)
			onlyAfterManual = analysed()
		}

		switch {
		case natural:
			// ok
		case onlyAfterManual:
			t.Errorf("Day3: failure was analysed only after a manual cycle; nothing reacts to the TaskFailed event on its own")
		default:
			t.Errorf("Day3: failure of task %s produced no analysis record (outcome evaluation + root cause) for the project", st.failedTaskID)
		}

		recs := sdLearnings(ctx, st)
		st.learningsAfterDay3 = len(recs)
		if len(recs) > learningsBefore {
			rec := recs[0]
			if rec.OutcomeEvaluationID != "" {
				if _, err := st.app.LearningRepo.GetOutcomeEvaluation(ctx, rec.OutcomeEvaluationID); err != nil {
					t.Errorf("Day3: analysis record points to outcome evaluation %s that cannot be loaded: %v", rec.OutcomeEvaluationID, err)
				}
			}
		}

		retried := false
		for _, other := range sdTasks(ctx, st) {
			if other.ID != st.failedTaskID && other.Status != task.StatusFailed && other.CreatedAt.After(failAt) {
				retried = true
			}
		}
		_, changesAfter := sdPlanStats(ctx, st)
		if !retried && changesAfter <= changesBefore {
			t.Errorf("Day3: AIA neither retried (new task after the failure) nor changed the Plan (plan changes %d -> %d)", changesBefore, changesAfter)
		}
	})

	// ----------------------------------------------------------------- Day 4
	t.Run("Day4_Restart_RestoreInsteadOfRestart", func(t *testing.T) {
		st.app.WorkerMgr.Stop()

		inflight, err := st.app.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
			ProjectID: domain.ID(st.projID),
			AgentID:   "coding-agent",
			Title:     "Day4 in-flight analysis",
		})
		if err != nil {
			t.Fatalf("create in-flight task: %v", err)
		}
		st.inflightTaskID = inflight.ID
		if err := st.app.TaskSvc.UpdateStatus(ctx, domain.ID(inflight.ID), task.StatusRunning); err != nil {
			t.Fatalf("in-flight task -> RUNNING: %v", err)
		}

		goalsBefore, _ := st.app.GoalSvc.ListByProject(ctx, st.projID)

		_ = st.app.Close()
		a2, err := app.NewApp(st.cfgPath, mock.NewMockProvider())
		if err != nil {
			t.Fatalf("restart AIA: %v", err)
		}
		st.app = a2

		recon, err := st.app.Reconstructor.ReconstructProjectState(ctx, st.projID)
		if err != nil {
			t.Fatalf("Day4: cannot reconstruct project state after a plain stop/start: %v "+
				"(a graceful shutdown may need to capture a pre-restart snapshot)", err)
		}

		hasGoal := false
		for _, g := range recon.ActiveGoals {
			if g.ID == st.goalID {
				hasGoal = true
			}
		}
		if !hasGoal {
			t.Errorf("Day4: goal %s not restored (active goals: %d)", st.goalID, len(recon.ActiveGoals))
		}

		hasDecision := false
		for _, d := range recon.RecentDecisions {
			if d.ID == st.decID {
				hasDecision = true
			}
		}
		if !hasDecision {
			t.Errorf("Day4: decision %s not restored (recent decisions: %d)", st.decID, len(recon.RecentDecisions))
		}

		if evi, err := st.app.KnowledgeRepo.GetEvidence(ctx, st.eviID); err != nil || evi.Status != knowledge.EvidenceValidated {
			t.Errorf("Day4: evidence %s not restored intact (err=%v)", st.eviID, err)
		}

		if tk, err := st.app.TaskSvc.Get(ctx, domain.ID(st.failedTaskID)); err != nil || tk.Status != task.StatusFailed {
			t.Errorf("Day4: failed task %s did not keep its FAILED state (err=%v)", st.failedTaskID, err)
		}
		if tk, err := st.app.TaskSvc.Get(ctx, domain.ID(st.inflightTaskID)); err != nil {
			t.Errorf("Day4: in-flight task %s lost: %v", st.inflightTaskID, err)
		} else if tk.Status == task.StatusRunning {
			t.Errorf("Day4: in-flight task %s still RUNNING after restart; it should be recovered or re-queued", st.inflightTaskID)
		}

		goalsAfter, _ := st.app.GoalSvc.ListByProject(ctx, st.projID)
		if len(goalsAfter) != len(goalsBefore) {
			t.Errorf("Day4: goal count changed across restart (%d -> %d); state was rebuilt instead of restored", len(goalsBefore), len(goalsAfter))
		}
	})

	// ----------------------------------------------------------------- Day 5
	t.Run("Day5_ContradictoryEvidence_ReevaluateDecision", func(t *testing.T) {
		st.day5At = time.Now().UTC()
		st.plansAtDay5, st.changesAtDay5 = sdPlanStats(ctx, st)
		st.learningsAtDay5 = len(sdLearnings(ctx, st))
		sdStartWorker(t, ctx, st.app)

		now := time.Now().UTC()
		st.eviNewID = "evi-sd-contradiction"
		if err := st.app.KnowledgeSvc.RecordEvidence(ctx, &knowledge.Evidence{
			ID:             st.eviNewID,
			SourceID:       st.srcID,
			ContentSnippet: "Benchmark latency regressed to 400ms in the latest week",
			ContentHash:    "sd-hash-contradiction-002",
			CaptureMethod:  knowledge.CaptureAPIJSON,
			Status:         knowledge.EvidenceValidated,
			CapturedAt:     now,
			CreatedAt:      now,
		}); err != nil {
			t.Fatalf("record contradicting evidence: %v", err)
		}
		if err := st.app.KnowledgeRepo.LinkClaimEvidence(ctx, &knowledge.ClaimEvidenceLink{
			ClaimID:          st.claimID,
			EvidenceID:       st.eviNewID,
			SupportType:      knowledge.SupportTypeRefutes,
			ConfidenceWeight: 0.9,
		}); err != nil {
			t.Fatalf("link contradicting evidence to the original claim: %v", err)
		}

		claimDemoted := func() bool {
			c, err := st.app.KnowledgeRepo.GetClaim(ctx, st.claimID)
			return err == nil && c.AssertionStatus != knowledge.AssertionAccepted
		}
		reevaluated := func() bool {
			for _, d := range sdDecisions(ctx, st, st.projID) {
				if d.ID == st.decID || d.CreatedAt.Before(st.day5At) {
					continue
				}
				blob := d.IntentSummary + " " + d.Justification + " " + d.ConsideredAlternativesJSON + " " + d.ReevaluationConditions
				if strings.Contains(blob, st.decID) || strings.Contains(blob, st.eviNewID) {
					return true
				}
				if links, err := st.app.DecisionRepo.GetEvidenceLinks(ctx, d.ID); err == nil {
					for _, l := range links {
						if l.EvidenceID == st.eviNewID {
							return true
						}
					}
				}
			}
			return false
		}

		sdPoll(sdWaitShort, func() bool { return claimDemoted() && reevaluated() })
		if !claimDemoted() || !reevaluated() {
			t.Logf("Day5: no complete reaction within the window; forcing one event-triggered cycle to isolate the gap")
			sdForceCycle(ctx, t, st, "new evidence contradicts claim "+st.claimID)
		}

		if !claimDemoted() {
			t.Errorf("Day5: claim %s is still ACCEPTED after a REFUTES evidence link was added; adding refuting evidence does not demote it "+
				"(PropagateEvidenceInvalidation exists but nothing calls it)", st.claimID)
		}
		if !reevaluated() {
			t.Errorf("Day5: decision %s was not re-evaluated; no new decision references it or the contradicting evidence %s",
				st.decID, st.eviNewID)
		}
	})

	// ----------------------------------------------------------------- Day 6
	t.Run("Day6_CorrectPlan_VerifyAndLearn", func(t *testing.T) {
		correctiveDone := func() bool {
			for _, tk := range sdTasks(ctx, st) {
				if tk.ID == st.failedTaskID || tk.ID == st.inflightTaskID {
					continue
				}
				if tk.CreatedAt.After(st.day5At) && tk.Status == task.StatusCompleted {
					return true
				}
			}
			return false
		}
		sdPoll(sdWaitShort, correctiveDone)

		plans, changes := sdPlanStats(ctx, st)
		if plans == 0 {
			t.Errorf("Day6: no Plan exists to be corrected")
		} else if changes <= st.changesAtDay5 {
			t.Errorf("Day6: Plan was not modified after the contradicting evidence (plan changes %d -> %d)", st.changesAtDay5, changes)
		}

		if !correctiveDone() {
			t.Errorf("Day6: no corrective task created after the contradicting evidence reached COMPLETED")
		}

		recs := sdLearnings(ctx, st)
		verified := false
		for _, r := range recs {
			v := strings.TrimSpace(r.VerificationJSON)
			if v != "" && v != "{}" && v != "null" {
				verified = true
			}
		}
		if len(recs) <= st.learningsAtDay5 {
			t.Errorf("Day6: no new learning record after the correction (records %d -> %d)", st.learningsAtDay5, len(recs))
		} else if !verified {
			t.Errorf("Day6: learning records exist but none carries a verification result")
		}
	})

	// ----------------------------------------------------------------- Day 7
	t.Run("Day7_GoalReassessment_GovernanceDecides", func(t *testing.T) {
		proposals := func() int {
			ps, err := st.app.LearningRepo.ListGoalEvolutionProposals(ctx, st.subGoalID)
			if err != nil {
				return 0
			}
			return len(ps)
		}

		sdPoll(sdWaitShort, func() bool { return proposals() > 0 })
		if proposals() == 0 {
			t.Logf("Day7: no proposal within the window; forcing one event-triggered cycle to isolate the gap")
			sdForceCycle(ctx, t, st, "reassess goals after claim "+st.claimID+" was contradicted")
		}

		ps, err := st.app.LearningRepo.ListGoalEvolutionProposals(ctx, st.subGoalID)
		if err != nil || len(ps) == 0 {
			t.Errorf("Day7: no evolution proposal for secondary goal %s although its premise (claim %s) was contradicted (err=%v)",
				st.subGoalID, st.claimID, err)
			return
		}

		for _, p := range ps {
			if strings.TrimSpace(p.Rationale) == "" {
				t.Errorf("Day7: proposal %s has no rationale", p.ID)
			}
			if p.Status == "" {
				t.Errorf("Day7: proposal %s has no governance status", p.ID)
			}
			// Governance: a change that was actually applied must carry an approval trail.
			if p.ResultingRevisionID != nil && p.ApprovalID == nil && p.ApprovedBy == "" {
				t.Errorf("Day7: proposal %s was applied (revision %s) without an approval trail", p.ID, *p.ResultingRevisionID)
			}
		}

		if g, err := st.app.GoalSvc.Get(ctx, st.subGoalID); err == nil {
			t.Logf("Day7: secondary goal status after governance: %s (%s)", g.Status, fmt.Sprintf("%d proposal(s)", len(ps)))
		}
	})
}
