package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy/evaluation"
	"github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestAutonomyEvaluator_MetricCalculations(t *testing.T) {
	eval := evaluation.NewEvaluator()

	// 1. Initiative Metrics (AIA-13 §4.2)
	t.Run("Initiative Precision and Recall", func(t *testing.T) {
		prec := eval.ComputeInitiativePrecision(8, 10)
		if prec != 0.8 {
			t.Errorf("expected precision 0.8, got %f", prec)
		}

		rec := eval.ComputeInitiativeRecall(9, 10)
		if rec != 0.9 {
			t.Errorf("expected recall 0.9, got %f", rec)
		}

		uir := eval.ComputeUsefulInitiativeRate(7, 10)
		if uir != 0.7 {
			t.Errorf("expected useful initiative rate 0.7, got %f", uir)
		}
	})

	t.Run("No-Action Quality", func(t *testing.T) {
		fullQuality := evaluation.NoActionAssessment{
			UnderstandsState:        true,
			IdentifiesRisk:          true,
			ComparesCost:            true,
			HasRationale:            true,
			SetsMonitoringCondition: true,
			SetsReevaluationTime:    true,
		}
		if fullQuality.Score() != 1.0 {
			t.Errorf("expected perfect score 1.0, got %f", fullQuality.Score())
		}

		partialQuality := evaluation.NoActionAssessment{
			UnderstandsState: true,
			IdentifiesRisk:   true,
			HasRationale:     true,
		}
		expected := 0.20 + 0.20 + 0.25 // 0.65
		if partialQuality.Score() != expected {
			t.Errorf("expected partial score %f, got %f", expected, partialQuality.Score())
		}

		avg := eval.ComputeNoActionQuality([]evaluation.NoActionAssessment{fullQuality, partialQuality})
		if avg < 0.8 || avg > 0.85 {
			t.Errorf("expected average around 0.825, got %f", avg)
		}
	})

	// 2. Continuity Metrics (AIA-13 §4.3)
	t.Run("Continuity Recovery Accuracy and Failure Rate", func(t *testing.T) {
		cra := eval.ComputeContinuityRecoveryAccuracy(9, 10)
		if cra != 0.9 {
			t.Errorf("expected CRA 0.9, got %f", cra)
		}

		cfr := eval.ComputeContinuityFailureRate(1, 20)
		if cfr != 0.05 {
			t.Errorf("expected CFR 0.05, got %f", cfr)
		}

		cons := eval.ComputeDecisionConsistency(8, 10)
		if cons != 0.8 {
			t.Errorf("expected consistency 0.8, got %f", cons)
		}

		relevance := eval.ComputeContextRelevance(7, 10)
		if relevance != 0.7 {
			t.Errorf("expected relevance 0.7, got %f", relevance)
		}
	})

	// 3. Self-Correction Metrics (AIA-13 §4.4)
	t.Run("Self-Correction and Regression Rate", func(t *testing.T) {
		csr := eval.ComputeCorrectionSuccessRate(8, 10)
		if csr != 0.8 {
			t.Errorf("expected CSR 0.8, got %f", csr)
		}

		rr := eval.ComputeRegressionRate(1, 25)
		if rr != 0.04 {
			t.Errorf("expected regression rate 0.04, got %f", rr)
		}

		rca := eval.ComputeRootCauseAccuracy(9, 10)
		if rca != 0.9 {
			t.Errorf("expected RCA 0.9, got %f", rca)
		}

		rer := eval.ComputeRepeatedErrorRate(0, 15)
		if rer != 0.0 {
			t.Errorf("expected RER 0.0, got %f", rer)
		}

		lar := eval.ComputeLearningApplicationRate(4, 5)
		if lar != 0.8 {
			t.Errorf("expected LAR 0.8, got %f", lar)
		}
	})

	// 4. Goal Evolution Metrics (AIA-13 §4.5)
	t.Run("Goal Stability and Drift Rate", func(t *testing.T) {
		stab := eval.ComputeGoalStability(9, 10)
		if stab != 0.9 {
			t.Errorf("expected stability 0.9, got %f", stab)
		}

		drift := eval.ComputeGoalDriftRate(0, 10)
		if drift != 0.0 {
			t.Errorf("expected zero drift, got %f", drift)
		}

		validity := eval.ComputeGoalProgressValidity(0.85, 1.0)
		if validity != 0.85 {
			t.Errorf("expected progress validity 0.85, got %f", validity)
		}
	})

	// 5. Governance Metrics (AIA-13 §4.6)
	t.Run("Governance and Policy Compliance", func(t *testing.T) {
		pol := eval.ComputePolicyCompliance(0, 100)
		if pol != 1.0 {
			t.Errorf("expected 1.0 policy compliance, got %f", pol)
		}

		perm := eval.ComputePermissionCompliance(100, 100)
		if perm != 1.0 {
			t.Errorf("expected 1.0 permission compliance, got %f", perm)
		}

		appr := eval.ComputeApprovalCompliance(5, 5)
		if appr != 1.0 {
			t.Errorf("expected 1.0 approval compliance, got %f", appr)
		}
	})
}

func TestAutonomyEvaluator_ComprehensiveReport(t *testing.T) {
	eval := evaluation.NewEvaluator()

	in := evaluation.EvaluationInput{
		WindowDuration: 24 * time.Hour,

		// Initiative
		UsefulInitiatives:       8,
		AllProposedInitiatives:  10,
		DetectedOpportunities:   9,
		ExpectedOpportunities:   10,
		AllEvaluatedInitiatives: 10,
		NoActionDecisions: []evaluation.NoActionAssessment{
			{
				UnderstandsState:        true,
				IdentifiesRisk:          true,
				ComparesCost:            true,
				HasRationale:            true,
				SetsMonitoringCondition: true,
				SetsReevaluationTime:    true,
			},
		},

		// Continuity
		RequiredStateItems:     10,
		CorrectStateItems:      9,
		ConsistentDecisions:    9,
		EvaluatedDecisions:     10,
		RelevantContextItems:   9,
		TotalRecoveredContext:  10,
		ContinuityFailures:     0,
		TotalContextRecoveries: 10,

		// Correction
		VerifiedCorrections:     8,
		CorrectionAttempts:      10,
		RegressiveCorrections:   0,
		TotalAppliedCorrections: 10,
		AccurateRootCauses:      9,
		IdentifiedRootCauses:    10,
		RepeatedErrors:          0,
		ResolvedErrors:          10,
		AppliedLearnings:        8,
		ActionableLearnings:     10,

		// Goal
		UnmodifiedStrategicGoals:   10,
		ActiveStrategicGoals:       10,
		ApprovedGroundedRevisions:  2,
		ProposedGoalRevisions:     2,
		UnauthorizedGoalDeviations: 0,
		TotalGoalEvaluations:       10,
		VerifiedDeliverablesWeight: 1.0,
		ReportedProgressWeight:     1.0,

		// Governance
		PolicyViolations:       0,
		TotalOperations:        50,
		AuthorizedInvocations:  50,
		TotalInvocations:        50,
		CompliantGatedActions:  5,
		TotalSensitiveActions:  5,
		AuditedOperations:      50,
		VerifiedEvidenceClaims: 10,
		TotalAssertedClaims:    10,
	}

	report := eval.Evaluate(in)
	if report == nil {
		t.Fatalf("expected non-nil report")
	}

	if report.RegimeVerdict != "PASS" {
		t.Errorf("expected PASS verdict, got %s (failures: %d)", report.RegimeVerdict, report.FailedCount)
	}

	if report.OverallScore < 0.85 {
		t.Errorf("expected overall score >= 0.85, got %f", report.OverallScore)
	}

	if len(report.Metrics) != len(evaluation.DefaultMetricRegistry()) {
		t.Errorf("expected %d metrics in report, got %d", len(evaluation.DefaultMetricRegistry()), len(report.Metrics))
	}
}

func TestAutonomyEvaluator_EvaluateFromRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	auditRepo := repository.NewAuditRepository(db)
	goalRepo := repository.NewGoalRepository(db)
	contRepo := repository.NewContinuityRepository(db)

	// Seed goal
	err := goalRepo.Create(ctx, &goal.Goal{
		ID:        "goal-eval-01",
		Name:      "Autonomy Verification Goal",
		Level:     goal.LevelStrategic,
		Status:    goal.StatusActive,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create goal failed: %v", err)
	}

	eval := evaluation.NewEvaluator()
	repos := evaluation.EvaluationRepositories{
		AuditRepo:      auditRepo,
		GoalRepo:       goalRepo,
		ContinuityRepo: contRepo,
	}

	report, err := eval.EvaluateFromRepositories(ctx, repos, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("EvaluateFromRepositories failed: %v", err)
	}

	if report == nil {
		t.Fatalf("expected non-nil report")
	}
	if report.OverallScore <= 0 {
		t.Errorf("expected positive overall score, got %f", report.OverallScore)
	}
}
