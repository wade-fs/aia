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

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/autonomy/clock"
	"github.com/wade/aia/internal/autonomy/learning"
	"github.com/wade/aia/internal/autonomy/recall"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/execution"
	domainGoal "github.com/wade/aia/internal/domain/goal"
	domainInv "github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/domain/knowledge"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/domain/profile"
	"github.com/wade/aia/internal/policy"
	goalService "github.com/wade/aia/internal/service/goal"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/web/console"
)

func seedEvidence(t *testing.T, ctx context.Context, repo *repository.KnowledgeRepository, id, uri, snippet string) *knowledge.Evidence {
	t.Helper()
	now := time.Now().UTC()
	src := &knowledge.Source{
		ID:                   "src-" + id,
		SourceType:           knowledge.SourcePrimary,
		URI:                  uri,
		AuthorityScore:       0.9,
		DirectnessScore:      0.9,
		TimelinessScore:      0.9,
		IndependenceScore:    0.9,
		ReproducibilityScore: 0.9,
		CreatedAt:            now,
	}
	if err := repo.CreateSource(ctx, src); err != nil {
		t.Fatalf("create source: %v", err)
	}
	evi := &knowledge.Evidence{
		ID:             id,
		SourceID:       src.ID,
		ContentSnippet: snippet,
		ContentHash:    "sha256:" + id,
		CaptureMethod:  knowledge.CaptureToolStdout,
		CapturedAt:     now,
		CreatedAt:      now,
	}
	if err := repo.CreateEvidence(ctx, evi); err != nil {
		t.Fatalf("create evidence: %v", err)
	}
	return evi
}

// TestPhase4_ScenarioA_TrueEffectiveLearning_L0ToL3 verifies the full L0 -> L1 -> L2 -> L3
// learning lifecycle, falsifiable RCA enforcement, Tier 3 code self-modification freeze,
// same-cycle L3 anti-cheat protection, and cross-cycle Recall/Deliberation reuse.
func TestPhase4_ScenarioA_TrueEffectiveLearning_L0ToL3(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	learnEng := app.AutonomyEngine.LearningEngine()
	if learnEng == nil {
		t.Fatalf("expected LearningEngine to be wired on AutonomyEngine")
	}

	// Run Cycle #1 to establish a real SourceCycleID
	cyc1, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   "default",
		TriggerType: continuity.TriggerScheduled,
		Summary:     "Cycle 1 baseline before discrepancy",
	})
	if err != nil {
		t.Fatalf("cycle 1 failed: %v", err)
	}

	// Seed a grounded Evidence & a stale Claim that caused an error
	now := time.Now().UTC()
	evi := seedEvidence(t, ctx, app.KnowledgeRepo, "evi-tx-race-1", "artifact://sqlite_busy_trace.log",
		"Concurrent writer transactions without IMMEDIATE lock triggered SQLITE_BUSY")

	staleClaim := &knowledge.Claim{
		ID:              "claim-tx-mode-1",
		Statement:       "sqlite_tx_boundary recommended_lock_mode = DEFERRED_DEFAULT",
		AssertionStatus: knowledge.AssertionAccepted,
		IsKnowledge:     true,
		CurrentVersion:  1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := app.KnowledgeRepo.CreateClaim(ctx, staleClaim); err != nil {
		t.Fatalf("create claim: %v", err)
	}

	// 1. L0_RECORDED: Record Outcome Discrepancy with Baseline
	projID := "default"
	oe, rec, err := learnEng.RecordDiscrepancy(ctx, learning.DiscrepancyInput{
		CycleID:         &cyc1.CycleID,
		ProjectID:       &projID,
		Domain:          "coding",
		Title:           "Use BEGIN IMMEDIATE for multi-step write transactions",
		MetricName:      "busy_error_rate",
		ExpectedMetric:  0.00,
		ActualMetric:    0.28,
		LowerIsBetter:   true,
		ActualContext:   map[string]any{"sample_runs": 25},
	})
	if err != nil {
		t.Fatalf("RecordDiscrepancy failed: %v", err)
	}
	if rec.LearningLevel != domainLearning.LevelL0Recorded || rec.EffectivenessVerdict != domainLearning.VerdictPending {
		t.Fatalf("expected L0_RECORDED / PENDING, got %s / %s", rec.LearningLevel, rec.EffectivenessVerdict)
	}

	// 2. Falsifiable RCA Guard: Reject unfalsifiable or evidence-free post-hoc storytelling!
	if _, err := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe.ID,
		LearningRecordID:      rec.ID,
		Category:              domainLearning.RCWorkflowDefect,
		Narrative:             "Maybe CPU was just slow",
		FalsifiableHypothesis: "", // Missing falsifiable hypothesis!
		CausalEvidenceRefs:    []string{evi.ID},
	}); !errors.Is(err, domainLearning.ErrUngroundedRCA) {
		t.Fatalf("expected ErrUngroundedRCA when FalsifiableHypothesis is empty, got %v", err)
	}
	if _, err := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe.ID,
		LearningRecordID:      rec.ID,
		Category:              domainLearning.RCWorkflowDefect,
		Narrative:             "Unbacked claim",
		FalsifiableHypothesis: "Error rate drops below 0.02",
		CausalEvidenceRefs:    nil, // Missing causal evidence!
	}); !errors.Is(err, domainLearning.ErrUngroundedRCA) {
		t.Fatalf("expected ErrUngroundedRCA when CausalEvidenceRefs is empty, got %v", err)
	}

	// Valid grounded & falsifiable RCA succeeds
	rca, err := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe.ID,
		LearningRecordID:      rec.ID,
		Category:              domainLearning.RCIncorrectFact,
		Narrative:             "DEFERRED transaction lock upgrades race under concurrent writers",
		FalsifiableHypothesis: "Under 16-writer stress replay, busy_error_rate must drop from 0.28 to <= 0.02 without p99 latency regression",
		CausalEvidenceRefs:    []string{evi.ID},
		CompetingFactors:      []string{"disk I/O spike"},
		InitialConfidence:     0.78,
	})
	if err != nil {
		t.Fatalf("AnalyzeRootCause failed: %v", err)
	}

	// 3. Governance Tier 3 Freeze Check: Code self-modification in Phase 4 MUST be blocked!
	if _, _, err := learnEng.ProposeAndApplyCorrection(ctx, learning.CorrectionInput{
		RCAID:            rca.ID,
		LearningRecordID: rec.ID,
		Scope:            domainLearning.ScopeC,
		CorrectionType:   domainLearning.CorrectionCode,
		TargetEntityRef:  "go/internal/storage/sqlite/db.go",
	}); !errors.Is(err, domainLearning.ErrTier3FrozenUntilPhase6) {
		t.Fatalf("expected ErrTier3FrozenUntilPhase6 for ScopeC/CODE correction, got %v", err)
	}

	// 4. L1_APPLIED: Apply Tier 1 Fact Correction (superseding stale claim)
	ca, rec, err := learnEng.ProposeAndApplyCorrection(ctx, learning.CorrectionInput{
		RCAID:            rca.ID,
		LearningRecordID: rec.ID,
		Scope:            domainLearning.ScopeA,
		CorrectionType:   domainLearning.CorrectionFact,
		TargetEntityRef:  "claim:" + staleClaim.ID,
		BeforeState:      map[string]any{"assertion_status": "ACCEPTED", "is_knowledge": true},
		ProposedState:    map[string]any{"assertion_status": "SUPERSEDED", "is_knowledge": false, "pattern_key": "sqlite_immediate_tx"},
	})
	if err != nil {
		t.Fatalf("ProposeAndApplyCorrection (Tier 1) failed: %v", err)
	}
	if ca.TierProfile != domainLearning.Tier1Auto || rec.LearningLevel != domainLearning.LevelL1Applied {
		t.Fatalf("expected TIER1_AUTO & L1_APPLIED, got %s & %s", ca.TierProfile, rec.LearningLevel)
	}
	updatedClaim, _ := app.KnowledgeRepo.GetClaim(ctx, staleClaim.ID)
	if updatedClaim.AssertionStatus != knowledge.AssertionSuperseded {
		t.Fatalf("expected claim status SUPERSEDED after Tier 1 correction, got %s", updatedClaim.AssertionStatus)
	}

	// 5. L2_VERIFIED: Controlled Verification (lower busy_error_rate is better)
	rec, err = learnEng.VerifyCorrection(ctx, learning.VerificationInput{
		LearningRecordID:      rec.ID,
		ComparisonMethod:      domainLearning.CompPiTReplay,
		BaselineMetric:        0.28,
		PostMetric:            0.00,
		LowerIsBetter:         true,
		HasControl:            true,
		ControlBaselineMetric: 0.28,
		ControlPostMetric:     0.27,
		SampleSize:            16,
		RegressionPassed:      true,
		RegressionDetails:     map[string]any{"checks": []string{"read_throughput_unchanged", "zero_deadlocks"}},
		ProposedConfidence:    0.85,
	})
	if err != nil {
		t.Fatalf("VerifyCorrection failed: %v", err)
	}
	if rec.LearningLevel != domainLearning.LevelL2Verified || rec.EffectivenessVerdict != domainLearning.VerdictEffective {
		t.Fatalf("expected L2_VERIFIED / EFFECTIVE, got %s / %s", rec.LearningLevel, rec.EffectivenessVerdict)
	}

	// 6. Anti-Cheat: Attempting to promote to L3 within the SAME source cycle MUST be rejected!
	if _, err := learnEng.RecordCrossScenarioApplication(ctx, learning.GeneralizationInput{
		LearningRecordID: rec.ID,
		CycleID:          cyc1.CycleID,
		ScenarioKey:      "same_cycle_cheat",
		OutcomeImproved:  true,
		RegressionPassed: true,
	}); !errors.Is(err, domainLearning.ErrSameCycleGeneralizationForbidden) {
		t.Fatalf("expected ErrSameCycleGeneralizationForbidden on same-cycle L3 attempt, got %v", err)
	}

	// 7. Subsequent Cycle #2: Recall retrieves L2_VERIFIED learning, Deliberator reinforces ACT,
	//    and cross-scenario reuse promotes record to L3_GENERALIZED!
	app.AutonomyEngine.RegisterObserver("tx-followup-obs", &mockTestObserver{
		observations: []domainAutonomy.Observation{
			{
				Source:        "CODE_HEALTH_SCANNER",
				Domain:        "coding",
				Category:      domainAutonomy.ObsCategoryOpportunity,
				Summary:       "Apply sqlite_immediate_tx pattern to portfolio settlement writer",
				Confidence:    domainAutonomy.ConfidenceConfirmed,
				Severity:      7,
				TargetAgentID: "coding-agent",
				Data:          map[string]any{"pattern_key": "sqlite_immediate_tx"},
				Timestamp:     time.Now().UTC(),
			},
		},
	})
	defer app.AutonomyEngine.UnregisterObserver("tx-followup-obs")

	cyc2, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   "default",
		TriggerType: continuity.TriggerScheduled,
		Summary:     "Cycle 2 cross-scenario application",
	})
	if err != nil {
		t.Fatalf("cycle 2 failed: %v", err)
	}
	if cyc2.Decision.ActionType != decision.ActionAct {
		t.Fatalf("expected ACT in cycle 2, got %s (%s)", cyc2.Decision.ActionType, cyc2.Decision.Justification)
	}
	if !strings.Contains(cyc2.Deliberation.CostBenefitAnalysis, "Reinforced by L2_VERIFIED learning") {
		t.Errorf("expected deliberation to cite L2_VERIFIED learning reinforcement, got: %s", cyc2.Deliberation.CostBenefitAnalysis)
	}

	rec, err = learnEng.RecordCrossScenarioApplication(ctx, learning.GeneralizationInput{
		LearningRecordID: rec.ID,
		CycleID:          cyc2.CycleID,
		ScenarioKey:      "portfolio_settlement_writer_concurrency",
		OutcomeImproved:  true,
		RegressionPassed: true,
		Notes:            "Zero lock contention across 32 concurrent settlement updates",
	})
	if err != nil {
		t.Fatalf("RecordCrossScenarioApplication failed: %v", err)
	}
	if rec.LearningLevel != domainLearning.LevelL3Generalized {
		t.Fatalf("expected promotion to L3_GENERALIZED after independent cross-cycle success, got %s", rec.LearningLevel)
	}

	validChain, err := app.EventHistoryRepo.VerifyAggregateChainIntegrity(ctx, "LEARNING", rec.ID)
	if err != nil || !validChain {
		t.Fatalf("expected intact SHA-256 event chain on LEARNING aggregate, valid=%v err=%v", validChain, err)
	}

	// Verify Web Console /api/v1/learnings endpoint returns the L3_GENERALIZED record
	mux := http.NewServeMux()
	console.RegisterConsoleRoutes(mux, &console.ConsoleServices{
		LearningRepo: app.LearningRepo,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/learnings?level=L3_GENERALIZED", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), rec.ID) {
		t.Fatalf("expected /api/v1/learnings?level=L3_GENERALIZED to return %s, status=%d body=%s", rec.ID, w.Code, w.Body.String())
	}
	reqDetail := httptest.NewRequest(http.MethodGet, "/api/v1/learnings/"+rec.ID, nil)
	wDetail := httptest.NewRecorder()
	mux.ServeHTTP(wDetail, reqDetail)
	if wDetail.Code != http.StatusOK || !strings.Contains(wDetail.Body.String(), rca.ID) {
		t.Fatalf("expected /api/v1/learnings/%s to return linked RCA %s, status=%d body=%s", rec.ID, rca.ID, wDetail.Code, wDetail.Body.String())
	}
}

// TestPhase4_ScenarioB_TailwindFalsePositiveAndPrePostCapped verifies that:
//  1. Uncontrolled PRE_POST_ONLY gains are capped at confidence <= 0.45 and remain INCONCLUSIVE (never L2).
//  2. Raw post-correction gains driven purely by market tailwinds (excess_delta <= 0) are marked INEFFECTIVE.
func TestPhase4_ScenarioB_TailwindFalsePositiveAndPrePostCapped(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()
	learnEng := app.AutonomyEngine.LearningEngine()

	evi := seedEvidence(t, ctx, app.KnowledgeRepo, "evi-mkt-1", "twse://2330/monthly", "Monthly return vs TAIEX benchmark")

	// Sub-scenario B1: PRE_POST_ONLY without counterfactual control -> capped at <= 0.45 & INCONCLUSIVE
	oe1, rec1, _ := learnEng.RecordDiscrepancy(ctx, learning.DiscrepancyInput{
		Domain:         "investment",
		Title:          "Uncontrolled pre/post parameter tweak",
		MetricName:     "monthly_return",
		ExpectedMetric: 0.04,
		ActualMetric:   -0.01,
		LowerIsBetter:  false,
	})
	rca1, _ := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe1.ID,
		LearningRecordID:      rec1.ID,
		Category:              domainLearning.RCBiasedJudgment,
		Narrative:             "Moving average window too slow",
		FalsifiableHypothesis: "Outperforms benchmark over >= 10 trades",
		CausalEvidenceRefs:    []string{evi.ID},
	})
	_, _, _ = learnEng.ProposeAndApplyCorrection(ctx, learning.CorrectionInput{
		RCAID:            rca1.ID,
		LearningRecordID: rec1.ID,
		Scope:            domainLearning.ScopeA,
		CorrectionType:   domainLearning.CorrectionJudgment,
		TargetEntityRef:  "judgment:ma_window",
		BeforeState:      map[string]any{"window": 20},
		ProposedState:    map[string]any{"window": 10},
	})

	rec1, err := learnEng.VerifyCorrection(ctx, learning.VerificationInput{
		LearningRecordID:   rec1.ID,
		ComparisonMethod:   domainLearning.CompPrePostOnly,
		BaselineMetric:     -0.01,
		PostMetric:         0.08, // Raw gain looks great (+9%)!
		LowerIsBetter:      false,
		SampleSize:         12,
		RegressionPassed:   true,
		ProposedConfidence: 0.92, // Agent tries to claim 92% confidence without a control!
	})
	if err != nil {
		t.Fatalf("VerifyCorrection B1 failed: %v", err)
	}
	if rec1.EffectivenessVerdict != domainLearning.VerdictInconclusive {
		t.Fatalf("expected PRE_POST_ONLY verdict INCONCLUSIVE, got %s", rec1.EffectivenessVerdict)
	}
	if rec1.LearningLevel != domainLearning.LevelL1Applied {
		t.Fatalf("PRE_POST_ONLY must NOT promote to L2_VERIFIED, got %s", rec1.LearningLevel)
	}
	if rec1.EpistemicConfidence > domainLearning.MaxPrePostConfidence {
		t.Fatalf("expected confidence capped <= %.2f, got %.2f", domainLearning.MaxPrePostConfidence, rec1.EpistemicConfidence)
	}

	// Sub-scenario B2: Benchmark Tailwind Confounder (+5% raw gain, but benchmark rose +7.5% -> excess_delta = -2.5%)
	oe2, rec2, _ := learnEng.RecordDiscrepancy(ctx, learning.DiscrepancyInput{
		Domain:         "investment",
		Title:          "Increase beta exposure",
		MetricName:     "portfolio_return",
		ExpectedMetric: 0.05,
		ActualMetric:   0.02,
		LowerIsBetter:  false,
	})
	rca2, _ := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe2.ID,
		LearningRecordID:      rec2.ID,
		Category:              domainLearning.RCFlawedStrategy,
		Narrative:             "Cash buffer dragged upside capture",
		FalsifiableHypothesis: "Excess return over TAIEX > 0 across >= 10 settlements",
		CausalEvidenceRefs:    []string{evi.ID},
	})
	_, _, _ = learnEng.ProposeAndApplyCorrection(ctx, learning.CorrectionInput{
		RCAID:            rca2.ID,
		LearningRecordID: rec2.ID,
		Scope:            domainLearning.ScopeA,
		CorrectionType:   domainLearning.CorrectionJudgment,
		TargetEntityRef:  "judgment:cash_buffer",
		BeforeState:      map[string]any{"cash_min": 0.25},
		ProposedState:    map[string]any{"cash_min": 0.10},
	})

	rec2, err = learnEng.VerifyCorrection(ctx, learning.VerificationInput{
		LearningRecordID:      rec2.ID,
		ComparisonMethod:      domainLearning.CompControlBenchmark,
		BaselineMetric:        0.02,  // Portfolio pre: +2.0%
		PostMetric:            0.07,  // Portfolio post: +7.0% (rawDelta = +5.0%)
		LowerIsBetter:         false,
		HasControl:            true,
		ControlBaselineMetric: 0.01,  // Benchmark pre: +1.0%
		ControlPostMetric:     0.085, // Benchmark post: +8.5% (controlDelta = +7.5% -> excessDelta = -2.5%!)
		SampleSize:            12,
		RegressionPassed:      true,
		ProposedConfidence:    0.85,
	})
	if err != nil {
		t.Fatalf("VerifyCorrection B2 failed: %v", err)
	}
	if rec2.EffectivenessVerdict != domainLearning.VerdictIneffective {
		t.Fatalf("expected tailwind-confounded gain to be marked INEFFECTIVE, got %s", rec2.EffectivenessVerdict)
	}
	if rec2.LearningLevel == domainLearning.LevelL2Verified {
		t.Fatalf("tailwind-confounded intervention MUST NOT reach L2_VERIFIED")
	}
	if !strings.Contains(rec2.VerificationJSON, `"tailwind_confounded":true`) {
		t.Errorf("expected tailwind_confounded:true in VerificationJSON, got %s", rec2.VerificationJSON)
	}
}

// TestPhase4_ScenarioC_RegressionRollbackAndNegativeConstraintSuppression verifies that:
//  1. A correction that improves the target metric but fails regression checks is marked REGRESSED,
//  2. The mutated entity is automatically rolled back to BeforeState,
//  3. RCA validity is decoupled as INTERVENTION_MISMATCH,
//  4. Future cycles recall the REGRESSED record as a NegativeConstraint and suppress repeating it!
func TestPhase4_ScenarioC_RegressionRollbackAndNegativeConstraintSuppression(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()
	learnEng := app.AutonomyEngine.LearningEngine()

	now := time.Now().UTC()
	evi := seedEvidence(t, ctx, app.KnowledgeRepo, "evi-reg-1", "metric://query_latency", "Batch cache aggressive TTL test")

	claim := &knowledge.Claim{
		ID:              "claim-cache-ttl",
		Statement:       "market_quote_cache ttl_policy = STRICT_60S",
		AssertionStatus: knowledge.AssertionAccepted,
		IsKnowledge:     true,
		CurrentVersion:  1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := app.KnowledgeRepo.CreateClaim(ctx, claim); err != nil {
		t.Fatalf("create claim: %v", err)
	}

	projID := "default"
	oe, rec, _ := learnEng.RecordDiscrepancy(ctx, learning.DiscrepancyInput{
		ProjectID:      &projID,
		Domain:         "coding",
		Title:          "Aggressive 24h stale quote caching",
		MetricName:     "p95_latency_ms",
		ExpectedMetric: 20.0,
		ActualMetric:   65.0,
		LowerIsBetter:  true,
	})
	rca, _ := learnEng.AnalyzeRootCause(ctx, learning.RCAInput{
		OutcomeEvaluationID:   oe.ID,
		LearningRecordID:      rec.ID,
		Category:              domainLearning.RCWorkflowDefect,
		Narrative:             "Short cache TTL triggers redundant DB reads",
		FalsifiableHypothesis: "p95 drops < 20ms while quote freshness invariant (< 5m staleness) holds",
		CausalEvidenceRefs:    []string{evi.ID},
	})
	_, _, err := learnEng.ProposeAndApplyCorrection(ctx, learning.CorrectionInput{
		RCAID:            rca.ID,
		LearningRecordID: rec.ID,
		Scope:            domainLearning.ScopeA,
		CorrectionType:   domainLearning.CorrectionFact,
		TargetEntityRef:  "claim:" + claim.ID,
		BeforeState:      map[string]any{"assertion_status": "ACCEPTED", "is_knowledge": true},
		ProposedState:    map[string]any{"assertion_status": "SUPERSEDED", "is_knowledge": false, "anti_pattern_key": "stale_24h_quote_cache"},
	})
	if err != nil {
		t.Fatalf("apply correction: %v", err)
	}

	// Verify regression failure triggers auto-rollback!
	rec, err = learnEng.VerifyCorrection(ctx, learning.VerificationInput{
		LearningRecordID:      rec.ID,
		ComparisonMethod:      domainLearning.CompPiTReplay,
		BaselineMetric:        65.0,
		PostMetric:            12.0, // Latency improved...
		LowerIsBetter:         true,
		HasControl:            true,
		ControlBaselineMetric: 65.0,
		ControlPostMetric:     64.0,
		SampleSize:            20,
		RegressionPassed:      false, // ...but broke quote freshness invariant!
		RegressionDetails:     map[string]any{"failed_check": "quote_staleness_exceeded_5m_sla"},
	})
	if err != nil {
		t.Fatalf("VerifyCorrection regression: %v", err)
	}
	if rec.EffectivenessVerdict != domainLearning.VerdictRegressed {
		t.Fatalf("expected REGRESSED verdict, got %s", rec.EffectivenessVerdict)
	}

	// Claim must be rolled back to ACCEPTED / is_knowledge=true!
	rolledBackClaim, _ := app.KnowledgeRepo.GetClaim(ctx, claim.ID)
	if rolledBackClaim.AssertionStatus != knowledge.AssertionAccepted || !rolledBackClaim.IsKnowledge {
		t.Fatalf("expected automatic rollback to ACCEPTED/is_knowledge=true, got %s/%v",
			rolledBackClaim.AssertionStatus, rolledBackClaim.IsKnowledge)
	}

	// RCA validity must be decoupled as INTERVENTION_MISMATCH (not FALSIFIED)
	updatedRCA, _ := app.LearningRepo.GetRCA(ctx, rca.ID)
	if updatedRCA.RCAValidityStatus != domainLearning.RCAInterventionMismatch {
		t.Fatalf("expected RCA validity INTERVENTION_MISMATCH, got %s", updatedRCA.RCAValidityStatus)
	}

	// Next cycle: an initiative proposing the regressed anti_pattern_key ("stale_24h_quote_cache")
	// MUST be penalized by Deliberator using NegativeConstraints -> rational NO_ACTION!
	app.AutonomyEngine.RegisterObserver("bad-repeat-obs", &mockTestObserver{
		observations: []domainAutonomy.Observation{
			{
				Source:        "CACHE_ADVISOR",
				Domain:        "coding",
				Category:      domainAutonomy.ObsCategoryOpportunity,
				Summary:       "Enable stale_24h_quote_cache to cut read load",
				Confidence:    domainAutonomy.ConfidenceProbable,
				Severity:      4, // V = 8.0; with NegativeConstraint penalty (P_s*0.25, R+6.0) -> EV < 0!
				TargetAgentID: "coding-agent",
				Data:          map[string]any{"anti_pattern_key": "stale_24h_quote_cache"},
				Timestamp:     time.Now().UTC(),
			},
		},
	})
	defer app.AutonomyEngine.UnregisterObserver("bad-repeat-obs")

	cycNext, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   "default",
		TriggerType: continuity.TriggerScheduled,
		Summary:     "Cycle testing negative constraint suppression",
	})
	if err != nil {
		t.Fatalf("cycle failed: %v", err)
	}
	if cycNext.Decision.ActionType != decision.ActionNoAction {
		t.Fatalf("expected regressed intervention to be suppressed into NO_ACTION, got %s (%s)",
			cycNext.Decision.ActionType, cycNext.Decision.Justification)
	}
	if !strings.Contains(cycNext.Decision.Justification, "NegativeConstraint") {
		t.Errorf("expected justification to cite NegativeConstraint suppression, got: %s", cycNext.Decision.Justification)
	}
}

// TestPhase4_ScenarioD_AntiGoalpostMovingAndGovernedGoalEvolution verifies that:
//  1. An Agent CANNOT rewrite a STRATEGIC goal ("Maximize Total Assets" -> "Minimize Volatility") after losses without invalidated premise claims.
//  2. OPTIMIZATION cannot alter a parent STRATEGIC goal's target metric.
//  3. Even with a proven invalidated premise claim, an Agent STRATEGIC REVISION stays PENDING_APPROVAL until a HUMAN approves it.
func TestPhase4_ScenarioD_AntiGoalpostMovingAndGovernedGoalEvolution(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	stratGoal, err := app.GoalSvc.Create(ctx, goalService.CreateGoalInput{
		ProjectID:         "default",
		Name:              "Maximize Risk-Adjusted Total Portfolio Assets",
		Description:       "Grow NT$100,000 virtual capital at >= 12% annualized return",
		Level:             domainGoal.LevelStrategic,
		TargetMetricJSON:  `{"annualized_return_pct":12.0}`,
		CurrentMetricJSON: `{"annualized_return_pct":-4.5}`,
		Priority:          9,
	})
	if err != nil {
		t.Fatalf("create strategic goal: %v", err)
	}

	// 1. Adversarial Goalpost-Moving Attempt: Agent tries to change target to "Minimize Volatility" due to -4.5% drawdown!
	badProp, err := app.GoalSvc.ProposeOrApplyEvolution(ctx, goalService.EvolutionInput{
		GoalID:                   stratGoal.ID,
		Mode:                     domainLearning.EvolutionRevision,
		ActorType:                "AGENT",
		ActorID:                  "investment-agent",
		ProposedName:             "Minimize Portfolio Volatility",
		ProposedTargetMetricJSON: `{"max_volatility_pct":6.0}`,
		Rationale:                "Recent portfolio drawdown of -4.5% makes asset maximization look bad; switching metric to low volatility.",
	})
	if !errors.Is(err, domainLearning.ErrGoalpostMovingBlocked) {
		t.Fatalf("expected ErrGoalpostMovingBlocked, got %v", err)
	}
	if badProp == nil || badProp.Status != domainLearning.EvoRejectedGoalpostMoving {
		t.Fatalf("expected proposal persisted as REJECTED_GOALPOST_MOVING, got %+v", badProp)
	}

	// Verify goal definition in DB is 100% untouched and immutable audit log recorded the violation
	unchangedGoal, _ := app.GoalSvc.Get(ctx, stratGoal.ID)
	if unchangedGoal.Name != "Maximize Risk-Adjusted Total Portfolio Assets" || unchangedGoal.TargetMetricJSON != `{"annualized_return_pct":12.0}` {
		t.Fatalf("STRATEGIC goal was illegally mutated! got name=%q metric=%s", unchangedGoal.Name, unchangedGoal.TargetMetricJSON)
	}
	audits, _ := app.AuditRepo.Query(ctx, repository.AuditFilter{ResourceType: "GOAL", ResourceID: stratGoal.ID, Limit: 10})
	if len(audits) == 0 || audits[0].Action != "ANTI_GOALPOST_MOVING_BLOCKED" {
		t.Fatalf("expected ANTI_GOALPOST_MOVING_BLOCKED audit entry, got %+v", audits)
	}

	// 2. OPTIMIZATION smuggling attempt (trying to alter parent STRATEGIC metric under OPTIMIZATION label) -> Blocked!
	_, err = app.GoalSvc.ProposeOrApplyEvolution(ctx, goalService.EvolutionInput{
		GoalID:                   stratGoal.ID,
		Mode:                     domainLearning.EvolutionOptimization,
		ActorType:                "AGENT",
		ActorID:                  "investment-agent",
		ProposedTargetMetricJSON: `{"annualized_return_pct":4.0}`,
		Rationale:                "Quietly lowering return bar via optimization mode",
	})
	if !errors.Is(err, domainLearning.ErrGoalpostMovingBlocked) {
		t.Fatalf("expected OPTIMIZATION altering strategic target metric to fail with ErrGoalpostMovingBlocked, got %v", err)
	}

	// Legitimate OPTIMIZATION keeping parent target metric locked succeeds (spawning TACTICAL sub-goal)
	projID := "default"
	optProp, err := app.GoalSvc.ProposeOrApplyEvolution(ctx, goalService.EvolutionInput{
		GoalID:                   stratGoal.ID,
		ProjectID:                &projID,
		Mode:                     domainLearning.EvolutionOptimization,
		ActorType:                "AGENT",
		ActorID:                  "investment-agent",
		ProposedTargetMetricJSON: `{"annualized_return_pct":12.0}`,
		ProposedSubGoalName:      "Cash-Flow Screen Rebalance",
		Rationale:                "Keep 12% target locked; rebalance sector weights using fundamental cash-flow screen",
	})
	if err != nil || optProp.Status != domainLearning.EvoApplied {
		t.Fatalf("expected valid OPTIMIZATION to apply, err=%v prop=%+v", err, optProp)
	}

	// 3. Legitimate Strategic REVISION backed by a CONFLICTED macro premise claim -> PENDING_APPROVAL -> HUMAN Approve!
	now := time.Now().UTC()
	_ = seedEvidence(t, ctx, app.KnowledgeRepo, "evi-regime-shift", "fred://FEDFUNDS", "Risk-free rate regime shifted structurally")
	invalidatedPremise := &knowledge.Claim{
		ID:              "claim-zero-rate-regime",
		Statement:       "macro_rate_regime zero_interest_rate_bound = TRUE",
		AssertionStatus: knowledge.AssertionConflicted,
		IsKnowledge:     false,
		CurrentVersion:  1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_ = app.KnowledgeRepo.CreateClaim(ctx, invalidatedPremise)

	revProp, err := app.GoalSvc.ProposeOrApplyEvolution(ctx, goalService.EvolutionInput{
		GoalID:                   stratGoal.ID,
		ProjectID:                &projID,
		Mode:                     domainLearning.EvolutionRevision,
		ActorType:                "AGENT",
		ActorID:                  "investment-agent",
		ProposedName:             "Maximize Total Assets Above Risk-Free + 600bps",
		ProposedTargetMetricJSON: `{"annualized_return_pct":10.5}`,
		InvalidatedClaimIDs:      []string{invalidatedPremise.ID},
		Rationale:                "Original ZIRP assumption invalidated by documented rate regime shift",
	})
	if err != nil {
		t.Fatalf("expected premise-backed revision proposal to enter PENDING_APPROVAL, got err=%v", err)
	}
	if revProp.Status != domainLearning.EvoPendingApproval || revProp.ApprovalID == nil {
		t.Fatalf("expected PENDING_APPROVAL with ApprovalID, got status=%s appr=%v", revProp.Status, revProp.ApprovalID)
	}

	// Attempting to apply BEFORE human approval MUST be denied!
	if _, err := app.GoalSvc.ApplyApprovedEvolution(ctx, revProp.ID, "Maximize Total Assets Above Risk-Free + 600bps", ""); !errors.Is(err, domainLearning.ErrStrategicGoalDirectWriteDenied) {
		t.Fatalf("expected ErrStrategicGoalDirectWriteDenied before human approval, got %v", err)
	}

	// Human owner approves via ApprovalSvc -> ApplyApprovedEvolution commits revision!
	if err := app.ApprovalSvc.Approve(ctx, *revProp.ApprovalID, "owner_wade", "Verified macro premise change", "HUMAN"); err != nil {
		t.Fatalf("human approve failed: %v", err)
	}
	appliedRev, err := app.GoalSvc.ApplyApprovedEvolution(ctx, revProp.ID, "Maximize Total Assets Above Risk-Free + 600bps", "Updated benchmark spread")
	if err != nil || appliedRev == nil {
		t.Fatalf("ApplyApprovedEvolution after human approval failed: %v", err)
	}

	updatedGoal, _ := app.GoalSvc.Get(ctx, stratGoal.ID)
	if updatedGoal.TargetMetricJSON != `{"annualized_return_pct":10.5}` {
		t.Fatalf("expected approved target metric to update to 10.5, got %s", updatedGoal.TargetMetricJSON)
	}
}

// TestPhase4_ScenarioE_PointInTimeReplayZeroLookAheadAndBudgetGate verifies:
//  1. Zero Look-Ahead Bias during Point-in-Time (AsOf) historical replay + accelerated prediction calibration.
//  2. Hard Daily Cycle & Daily Token Budget Gates short-circuiting execution before runaway cost.
func TestPhase4_ScenarioE_PointInTimeReplayZeroLookAheadAndBudgetGate(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Part 1: Point-in-Time Historical Replay (Zero Look-Ahead Bias)
	// -------------------------------------------------------------------------
	t1 := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC)
	t3Future := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)

	prof, err := app.InvSvc.EnsureDefaultProfile(clock.WithAsOf(ctx, t1, true))
	if err != nil {
		t.Fatalf("EnsureDefaultProfile: %v", err)
	}
	_ = app.InvSvc.EnsureDataSources(ctx)
	inst, err := app.InvRepo.GetInstrumentBySymbol(ctx, "TWSE", "2330")
	if err != nil {
		t.Fatalf("GetInstrumentBySymbol 2330: %v", err)
	}

	saveBarAt := func(id string, closePrice float64, ts time.Time) {
		b, _ := json.Marshal(domainInv.MarketBarValue{
			Open: closePrice, High: closePrice, Low: closePrice, Close: closePrice, Volume: 10000000,
		})
		if err := app.InvRepo.SaveMarketData(ctx, &domainInv.MarketData{
			ID:           id,
			InstrumentID: inst.ID,
			DataType:     domainInv.MarketDataDailyBar,
			Period:       "1D",
			ValueJSON:    string(b),
			SourceID:     "ds-twse-openapi",
			FetchedAt:    ts,
			EffectiveAt:  ts,
		}); err != nil {
			t.Fatalf("SaveMarketData(%s): %v", id, err)
		}
	}

	// Seed T1 (price 900), T2 (price 920), and FUTURE T3 (price 1180 — huge +28% rally!)
	saveBarAt("md-t1", 900, t1)
	saveBarAt("md-t2", 920, t2)
	saveBarAt("md-t3-future", 1180, t3Future)

	// Seed a T2 announcement and a FUTURE T3 announcement + FUTURE T3 memory
	_ = app.InvRepo.SaveRawDocument(ctx, &domainInv.RawDocument{
		ID:             "doc-t2",
		SourceID:       "ds-mops-announcements",
		InstrumentID:   inst.ID,
		SourceRecordID: "mops-t2",
		Title:          "Jan Revenue +4% YoY",
		ContentType:    domainInv.ContentTypeAnnouncement,
		PublishedAt:    t2,
		FetchedAt:      t2,
	})
	_ = app.InvRepo.SaveRawDocument(ctx, &domainInv.RawDocument{
		ID:             "doc-t3-future",
		SourceID:       "ds-mops-announcements",
		InstrumentID:   inst.ID,
		SourceRecordID: "mops-t3",
		Title:          "FUTURE MARCH MEGA CAPEX SURPRISE",
		ContentType:    domainInv.ContentTypeAnnouncement,
		PublishedAt:    t3Future,
		FetchedAt:      t3Future,
	})
	_, _ = app.DB.RawDB().ExecContext(ctx, `
		INSERT INTO memories (id, type, scope, scope_id, title, content, summary, source_task_id, provenance_type, status, confidence, created_at, updated_at)
		VALUES ('mem-t3-future', 'DECISION', 'PROJECT', 'default', 'Future March Leak', 'TSMC surged to 1180 in March', 'Future leak', '', 'USER_EXPLICIT', 'ACTIVE', 0.95, ?, ?)
	`, t3Future.Format(time.RFC3339), t3Future.Format(time.RFC3339))

	// Construct replay context pinned at T2
	replayCtxT2 := clock.WithAsOf(ctx, t2, true)

	// Live network sync MUST be blocked during replay!
	if _, err := app.InvSvc.SyncMarketData(replayCtxT2, "2330"); err == nil || !strings.Contains(err.Error(), "prohibited during Point-in-Time replay") {
		t.Fatalf("expected SyncMarketData to fail in replay mode, got %v", err)
	}
	if _, err := app.InvSvc.SyncAnnouncements(replayCtxT2, "2330"); err == nil || !strings.Contains(err.Error(), "prohibited during Point-in-Time replay") {
		t.Fatalf("expected SyncAnnouncements to fail in replay mode, got %v", err)
	}

	// Latest market data at T2 MUST be 920 (never 1180 from T3!)
	latestAtT2, err := app.InvRepo.GetLatestMarketData(replayCtxT2, inst.ID)
	if err != nil || latestAtT2 == nil {
		t.Fatalf("GetLatestMarketData at T2 failed: %v", err)
	}
	var barAtT2 domainInv.MarketBarValue
	_ = json.Unmarshal([]byte(latestAtT2.ValueJSON), &barAtT2)
	if barAtT2.Close != 920 {
		t.Fatalf("look-ahead leak! expected close 920 at T2, got %.2f", barAtT2.Close)
	}

	docsAtT2, err := app.InvRepo.ListRawDocuments(replayCtxT2, inst.ID, 10)
	if err != nil || len(docsAtT2) != 1 || docsAtT2[0].ID != "doc-t2" {
		t.Fatalf("look-ahead leak in raw_documents! expected only doc-t2 at T2, got %+v (err=%v)", docsAtT2, err)
	}

	// Recall at T2 MUST NOT see mem-t3-future!
	recalledAtT2, err := app.AutonomyEngine.RecallEngine().Recall(replayCtxT2, recall.RecallInput{
		ProjectID:   "default",
		MaxMemories: 20,
	})
	if err != nil {
		t.Fatalf("Recall at T2 failed: %v", err)
	}
	for _, m := range recalledAtT2.Memories {
		if m.ID == "mem-t3-future" {
			t.Fatalf("look-ahead leak in RecallEngine! future memory %s leaked at T2", m.ID)
		}
	}

	// Record a directional prediction at T2 due at T3, then fast-forward virtual clock to T3 to settle & calibrate!
	_, err = app.InvSvc.ProcessAgyDecision(replayCtxT2, domainInv.AgyDecisionInput{
		Symbol:          "2330",
		Exchange:        "TWSE",
		Action:          "HOLD",
		Confidence:      0.80,
		Reasoning:       []string{"T2 thesis based strictly on <= T2 data"},
		ExpectedHorizon: "28D",
		StrategyVersion: "pit-replay-v1",
	}, 920.0)
	if err != nil {
		t.Fatalf("ProcessAgyDecision at T2: %v", err)
	}
	invDecT2, err := app.InvRepo.GetLatestDecisionForInstrument(replayCtxT2, prof.ID, inst.ID)
	if err != nil || invDecT2 == nil {
		t.Fatalf("GetLatestDecisionForInstrument at T2: %v", err)
	}
	pred := &domainInv.Prediction{
		ID:                "pred-pit-2330",
		DecisionID:        invDecT2.ID,
		InstrumentID:      inst.ID,
		PredictionType:    "DIRECTION",
		Target:            "PRICE_CHANGE_PCT",
		ExpectedDirection: "UP",
		Probability:       0.80,
		BasePrice:         920,
		TargetValue:       980,
		EvaluationHorizon: "28D",
		StrategyVersion:   "pit-replay-v1",
		Status:            domainInv.PredictionPending,
		CreatedAt:         t2,
		DueAt:             t3Future,
	}
	if err := app.InvSvc.CreatePrediction(replayCtxT2, pred); err != nil {
		t.Fatalf("CreatePrediction at T2: %v", err)
	}

	// At T2, prediction is NOT due yet -> 0 evaluated
	outcomesAtT2, _, err := app.InvSvc.EvaluatePredictions(replayCtxT2)
	if err != nil || len(outcomesAtT2) != 0 {
		t.Fatalf("expected 0 predictions settled at T2 before DueAt, got %d (err=%v)", len(outcomesAtT2), err)
	}

	// Advance virtual clock to T3 -> settles against T3 price (1180 > 920 -> UP correct, Brier = (0.80 - 1.0)^2 = 0.04)!
	replayCtxT3 := clock.WithAsOf(ctx, t3Future.Add(1*time.Hour), true)
	outcomesAtT3, evalRes, err := app.InvSvc.EvaluatePredictions(replayCtxT3)
	if err != nil || len(outcomesAtT3) != 1 || evalRes == nil {
		t.Fatalf("expected 1 prediction settled at T3, got %d (err=%v)", len(outcomesAtT3), err)
	}
	if outcomesAtT3[0].BrierScore < 0.039 || outcomesAtT3[0].BrierScore > 0.041 {
		t.Fatalf("expected settled Brier score ~0.04, got %.4f", outcomesAtT3[0].BrierScore)
	}

	// -------------------------------------------------------------------------
	// Part 2: Hard Daily Cycle & Token Budget Gate Enforcement
	// -------------------------------------------------------------------------
	budgetDay := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	app.PolicyEvaluator.SetBudgetLimits(policy.BudgetLimits{
		MaxDailyTokens: 500,
		MaxDailyCycles: 1,
	})

	// 1st cycle on 2026-05-10 is within MaxDailyCycles=1 -> allowed to execute normally (not blocked by budget gate)
	firstCyc, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   "default",
		TriggerType: continuity.TriggerScheduled,
		Summary:     "First cycle within daily limit",
		AsOf:        &budgetDay,
		ReplayMode:  true,
	})
	if err != nil || firstCyc.Decision.GovernanceResult != decision.GovernanceAllowed || firstCyc.Decision.ActionType == decision.ActionDefer {
		t.Fatalf("expected 1st cycle on budgetDay to succeed as ALLOWED (not deferred), got err=%v dec=%+v", err, firstCyc)
	}

	// 2nd cycle on 2026-05-10 exceeds MaxDailyCycles=1 -> short-circuited into DEFER (BLOCKED)!
	laterSameDay := budgetDay.Add(2 * time.Hour)
	secondCyc, err := app.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
		ProjectID:   "default",
		TriggerType: continuity.TriggerScheduled,
		Summary:     "Second cycle exceeding MaxDailyCycles=1",
		AsOf:        &laterSameDay,
		ReplayMode:  true,
	})
	if err != nil {
		t.Fatalf("expected governed DEFER outcome rather than unhandled crash, got %v", err)
	}
	if secondCyc.Decision.ActionType != decision.ActionDefer || secondCyc.Decision.GovernanceResult != decision.GovernanceBlocked {
		t.Fatalf("expected DEFER + BLOCKED on 2nd cycle, got %s + %s",
			secondCyc.Decision.ActionType, secondCyc.Decision.GovernanceResult)
	}

	// Seed 500 used tokens today and verify ExecutionSvc.Execute blocks pre-flight!
	tk, err := app.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		ProjectID: "default",
		AgentID:   "coding-agent",
		Title:     "Budget test task",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	execRepo := repository.NewExecutionRepository(app.DB)
	_ = execRepo.Create(ctx, &execution.Execution{
		ID:          "exec-budget-exhaust",
		TaskID:      tk.ID,
		AgentID:     "coding-agent",
		Provider:    "mock",
		Workspace:   tmpDir,
		Status:      execution.StatusCompleted,
		TotalTokens: 500,
		CreatedAt:   time.Now().UTC(),
	})
	ag, _ := app.AgentSvc.Get(ctx, "coding-agent")
	_, execErr := app.ExecutionSvc.Execute(ctx, tk, ag, profile.ExecutionProfile{ID: "default", Version: 1}, "")
	if execErr == nil || !strings.Contains(execErr.Error(), "execution blocked by budget gate") {
		t.Fatalf("expected ExecutionSvc.Execute to be blocked by token budget gate, got %v", execErr)
	}

	// Verify JSON serialization of CycleOutcome cleanly includes Verification
	rawOut, _ := json.Marshal(secondCyc)
	if !strings.Contains(string(rawOut), "Daily autonomy cycle count") {
		t.Errorf("expected budget reason in serialized CycleOutcome, got %s", string(rawOut))
	}
}
