package unit

import (
	"context"
	"strings"
	"testing"
	"time"

	domainExp "github.com/wade/aia/internal/domain/experiment"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/knowledge"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/experiment"
	approvalService "github.com/wade/aia/internal/service/approval"
	govService "github.com/wade/aia/internal/service/governance"
	invService "github.com/wade/aia/internal/service/investment"
	knowledgeService "github.com/wade/aia/internal/service/knowledge"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestExperiment_HypothesisValidationAndDecisions(t *testing.T) {
	// 1. Falsifiable Hypothesis Validation (AIA-15 §3.4)
	err := experiment.ValidateHypothesis("")
	if err == nil {
		t.Fatalf("expected error for empty hypothesis")
	}

	err = experiment.ValidateHypothesis("The market exists and stocks trade.")
	if err == nil || !strings.Contains(err.Error(), "lacks falsifiable comparative condition") {
		t.Fatalf("expected error for non-comparative hypothesis, got: %v", err)
	}

	err = experiment.ValidateHypothesis("Generation 2 achieves superior calibrated returns over Generation 1.")
	if err != nil {
		t.Fatalf("expected valid hypothesis to pass, got: %v", err)
	}

	// 2. 4-Way Post-Comparison Decisions (AIA-15 §3.9)
	trackA := domainExp.TrackAPerformance{}
	trackB := domainExp.TrackBAutonomy{}

	dec, _ := experiment.EvaluateNextStepDecision(domainExp.RegimeG2ClearAdvantage, trackA, trackB)
	if dec != domainExp.DecisionContinue {
		t.Errorf("expected CONTINUE for clear advantage, got %v", dec)
	}

	dec, _ = experiment.EvaluateNextStepDecision(domainExp.RegimeG2LearningInProgress, trackA, trackB)
	if dec != domainExp.DecisionModify {
		t.Errorf("expected MODIFY for learning in progress, got %v", dec)
	}

	dec, _ = experiment.EvaluateNextStepDecision(domainExp.RegimeG1RulesSuperior, trackA, trackB)
	if dec != domainExp.DecisionStop {
		t.Errorf("expected STOP for G1 rules superior, got %v", dec)
	}

	dec, _ = experiment.EvaluateNextStepDecision(domainExp.RegimeInconclusiveLowSample, trackA, trackB)
	if dec != domainExp.DecisionDefer {
		t.Errorf("expected DEFER for inconclusive low sample, got %v", dec)
	}
}

func TestExperiment_LifecycleTransitions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	expRepo := repository.NewExperimentRepository(db)
	eventHistoryRepo := repository.NewEventHistoryRepository(db)
	goalRepo := repository.NewGoalRepository(db)
	invRepo := repository.NewInvestmentRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	invSvc := invService.NewService(invRepo, auditRepo, nil)

	ctrl := experiment.NewController(experiment.Dependencies{
		ExpRepo:          expRepo,
		InvSvc:           invSvc,
		EventHistoryRepo: eventHistoryRepo,
		GoalRepo:         goalRepo,
	})

	bundle, err := ctrl.ProposeExperiment(ctx, experiment.CreateDualInput{
		Name:       "Lifecycle Experiment",
		Hypothesis: "G2 outperforms baseline G1 under volatile regimes",
	})
	if err != nil {
		t.Fatalf("failed to propose experiment: %v", err)
	}
	expID := bundle.Experiment.ID

	if bundle.Experiment.Status != domainExp.StatusProposed {
		t.Fatalf("expected PROPOSED status, got %v", bundle.Experiment.Status)
	}

	// Transition: PROPOSED -> PENDING_APPROVAL
	if err := ctrl.SubmitForApproval(ctx, expID); err != nil {
		t.Fatalf("failed to submit for approval: %v", err)
	}
	exp, _ := expRepo.GetExperiment(ctx, expID)
	if exp.Status != domainExp.StatusPendingApproval {
		t.Errorf("expected PENDING_APPROVAL, got %v", exp.Status)
	}

	// Transition: PENDING_APPROVAL -> READY
	if err := ctrl.ApproveAndReady(ctx, expID); err != nil {
		t.Fatalf("failed to approve and ready: %v", err)
	}
	exp, _ = expRepo.GetExperiment(ctx, expID)
	if exp.Status != domainExp.StatusReady {
		t.Errorf("expected READY, got %v", exp.Status)
	}

	// Transition: READY -> RUNNING
	if err := ctrl.StartExperiment(ctx, expID); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	exp, _ = expRepo.GetExperiment(ctx, expID)
	if exp.Status != domainExp.StatusRunning {
		t.Errorf("expected RUNNING, got %v", exp.Status)
	}

	// Transition: RUNNING -> PAUSED
	if err := ctrl.PauseExperiment(ctx, expID, "Market holiday"); err != nil {
		t.Fatalf("failed to pause: %v", err)
	}
	exp, _ = expRepo.GetExperiment(ctx, expID)
	if exp.Status != domainExp.StatusPaused {
		t.Errorf("expected PAUSED, got %v", exp.Status)
	}

	// Transition: PAUSED -> RUNNING
	if err := ctrl.ResumeExperiment(ctx, expID); err != nil {
		t.Fatalf("failed to resume: %v", err)
	}
	exp, _ = expRepo.GetExperiment(ctx, expID)
	if exp.Status != domainExp.StatusRunning {
		t.Errorf("expected RUNNING after resume, got %v", exp.Status)
	}

	// Transition: RUNNING -> EVALUATING
	if err := ctrl.EvaluateExperiment(ctx, expID); err != nil {
		t.Fatalf("failed to evaluate: %v", err)
	}
	exp, _ = expRepo.GetExperiment(ctx, expID)
	if exp.Status != domainExp.StatusEvaluating {
		t.Errorf("expected EVALUATING, got %v", exp.Status)
	}

	// Transition: EVALUATING -> COMPLETED
	if err := ctrl.CompleteExperiment(ctx, expID, "Evaluation finished with positive results"); err != nil {
		t.Fatalf("failed to complete: %v", err)
	}
	exp, _ = expRepo.GetExperiment(ctx, expID)
	if exp.Status != domainExp.StatusCompleted {
		t.Errorf("expected COMPLETED, got %v", exp.Status)
	}
}

func TestGovernance_OversightAndApprovalLifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// Seed project and task
	_, _ = db.RawDB().ExecContext(ctx, `
		INSERT INTO projects (id, name, workspace, description, status, created_at, updated_at)
		VALUES ('p-gov', 'GovProj', '/tmp', 'D', 'ACTIVE', ?, ?)
	`, now, now)
	_, _ = db.RawDB().ExecContext(ctx, `
		INSERT INTO tasks (id, project_id, title, description, status, priority, risk, acceptance_json, created_at, updated_at)
		VALUES ('task-gov-1', 'p-gov', 'T-Gov', 'D', 'READY', 5, 'HIGH', '{}', ?, ?)
	`, now, now)

	taskRepo := repository.NewTaskRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	apprRepo := repository.NewApprovalRepository(db)
	taskSvc := taskService.NewService(taskRepo, auditRepo)
	apprSvc := approvalService.NewService(apprRepo, taskSvc, auditRepo, nil)

	taskID := "task-gov-1"
	reqInput := approvalService.RequestApprovalInput{
		RequesterEntity:     "AGENT:coordinator",
		OperationType:       governance.OperationTaskExecution,
		OperationDetailJSON: `{"action":"critical_mutation"}`,
		RiskTier:            governance.RiskTierHigh,
		TaskID:              &taskID,
		ExpectedBenefit:     "Reduces latency by 40%",
		ResourceImpact:      "Requires 2GB additional RAM",
		Reversibility:       "Immediate rollback available via snapshot",
		Alternatives:        []string{"Keep baseline unchanged", "Scale horizontally"},
	}

	appr, err := apprSvc.RequestApproval(ctx, reqInput)
	if err != nil {
		t.Fatalf("failed to request approval: %v", err)
	}

	// Verify structured fields unpacked
	fetched, err := apprSvc.Get(ctx, appr.ID)
	if err != nil {
		t.Fatalf("failed to get approval: %v", err)
	}
	if fetched.ExpectedBenefit != "Reduces latency by 40%" {
		t.Errorf("expected benefit mismatch: %s", fetched.ExpectedBenefit)
	}
	if len(fetched.Alternatives) != 2 {
		t.Errorf("expected 2 alternatives, got %d", len(fetched.Alternatives))
	}

	// 1. Test RequestRevision
	err = apprSvc.RequestRevision(ctx, appr.ID, "human_reviewer", "Provide rollback contingency benchmarks", "HUMAN")
	if err != nil {
		t.Fatalf("failed to request revision: %v", err)
	}
	fetched, _ = apprSvc.Get(ctx, appr.ID)
	if fetched.Status != governance.ApprovalNeedsRevision {
		t.Errorf("expected status NEEDS_REVISION, got %s", fetched.Status)
	}
	if !strings.Contains(*fetched.ReviewerNote, "Provide rollback contingency") {
		t.Errorf("expected feedback in note, got %v", *fetched.ReviewerNote)
	}

	// 2. Test RevokeApproval
	err = apprSvc.RevokeApproval(ctx, appr.ID, "human_reviewer", "Risk profile changed externally", "HUMAN")
	if err != nil {
		t.Fatalf("failed to revoke approval: %v", err)
	}
	fetched, _ = apprSvc.Get(ctx, appr.ID)
	if fetched.Status != governance.ApprovalRevoked {
		t.Errorf("expected status REVOKED, got %s", fetched.Status)
	}

	// Linked task should be cancelled
	tsk, _ := taskRepo.Get(ctx, taskID)
	if tsk.Status != task.StatusCancelled {
		t.Errorf("expected task CANCELLED after revocation, got %s", tsk.Status)
	}
}

func TestGovernance_ResumeDispatchCriticalSafetyCheck(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	govRepo := repository.NewGovernanceRepository(db)
	govSvc := govService.NewService(govRepo, nil, nil, nil)
	humanActor := governance.Actor{Type: "HUMAN", ID: "admin"}

	// Trigger emergency stop with CRITICAL security breach
	err := govSvc.TriggerEmergencyStop(ctx, governance.Actor{Type: "SYSTEM", ID: "monitor"}, "CRITICAL: Potential privilege escalation detected")
	if err != nil {
		t.Fatalf("failed to trigger stop: %v", err)
	}

	// Resuming without force MUST FAIL due to critical security stop
	err = govSvc.ResumeDispatch(ctx, humanActor, false)
	if err == nil || !strings.Contains(err.Error(), "without force flag") {
		t.Fatalf("expected error resuming critical stop without force, got: %v", err)
	}

	// Resuming with force MUST SUCCEED
	err = govSvc.ResumeDispatch(ctx, humanActor, true)
	if err != nil {
		t.Fatalf("expected forced resume to succeed, got: %v", err)
	}
	ctrl, _ := govSvc.GetRuntimeControl(ctx)
	if ctrl.IsEmergencyStopped() {
		t.Errorf("expected runtime to be RUNNING after forced resume")
	}
}

func TestKnowledge_ProvenanceInvalidationPropagation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	kRepo := repository.NewKnowledgeRepository(db)
	dRepo := repository.NewDecisionRepository(db)
	cRepo := repository.NewContinuityRepository(db)
	svc := knowledgeService.NewService(kRepo, dRepo, cRepo)

	// 1. Register Source
	_ = svc.RegisterSource(ctx, &knowledge.Source{
		ID:                   "src-audit-1",
		SourceType:           knowledge.SourceOfficial,
		URI:                  "https://example.com/audit",
		AuthorityScore:       0.95,
		DirectnessScore:      0.9,
		TimelinessScore:      0.9,
		IndependenceScore:    1.0,
		ReproducibilityScore: 1.0,
		CreatedAt:            now,
	})

	// 2. Record Evidence with Status
	evidenceID := "evi-q3-results"
	err := svc.RecordEvidence(ctx, &knowledge.Evidence{
		ID:             evidenceID,
		SourceID:       "src-audit-1",
		ContentSnippet: "Q3 net income was 500 million",
		ContentHash:    "hash123",
		CaptureMethod:  knowledge.CaptureAPIJSON,
		Status:         knowledge.EvidenceValidated,
		CapturedAt:     now,
	})
	if err != nil {
		t.Fatalf("failed to record evidence: %v", err)
	}

	evi, _ := kRepo.GetEvidence(ctx, evidenceID)
	if evi.Status != knowledge.EvidenceValidated {
		t.Errorf("expected evidence status VALIDATED, got %v", evi.Status)
	}

	// 3. Create Claim of type HISTORICAL_SKELETON supported by evidence
	claimID := "claim-earnings"
	claim := &knowledge.Claim{
		ID:              claimID,
		Statement:       "The company maintained strong profitability in Q3",
		ClaimType:       knowledge.ClaimHistoricalSkeleton,
		AssertionStatus: knowledge.AssertionAccepted,
		IsKnowledge:     true,
	}
	links := []*knowledge.ClaimEvidenceLink{
		{
			ClaimID:          claimID,
			EvidenceID:       evidenceID,
			SupportType:      knowledge.SupportTypeSupports,
			ConfidenceWeight: 1.0,
		},
	}
	if err := svc.CreateClaim(ctx, claim, links); err != nil {
		t.Fatalf("failed to create claim: %v", err)
	}

	retrievedClaim, _ := kRepo.GetClaim(ctx, claimID)
	if retrievedClaim.ClaimType != knowledge.ClaimHistoricalSkeleton {
		t.Errorf("expected claim type HISTORICAL_SKELETON, got %v", retrievedClaim.ClaimType)
	}

	// 4. Propagate Invalidation
	report, err := svc.PropagateEvidenceInvalidation(ctx, evidenceID, "Accounting error discovered; company issued retraction")
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	if report.NewStatus != knowledge.EvidenceDeprecated {
		t.Errorf("expected evidence DEPRECATED, got %v", report.NewStatus)
	}
	if len(report.AffectedClaimIDs) != 1 || report.AffectedClaimIDs[0] != claimID {
		t.Errorf("expected claim %s in affected claims, got %v", claimID, report.AffectedClaimIDs)
	}

	// Verify claim was demoted
	updatedClaim, _ := kRepo.GetClaim(ctx, claimID)
	if updatedClaim.AssertionStatus != knowledge.AssertionUncertain {
		t.Errorf("expected assertion status UNCERTAIN, got %v", updatedClaim.AssertionStatus)
	}
	if updatedClaim.IsKnowledge {
		t.Errorf("expected claim is_knowledge to be demoted to false")
	}
}
