package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/autonomy/clock"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/knowledge"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	approvalService "github.com/wade/aia/internal/service/approval"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type Dependencies struct {
	LearningRepo     *repository.LearningRepository
	KnowledgeRepo    *repository.KnowledgeRepository
	MemoryRepo       *repository.MemoryRepository
	DecisionRepo     *repository.DecisionRepository
	EventHistoryRepo *repository.EventHistoryRepository
	AuditRepo        *repository.AuditRepository
	ApprovalSvc      *approvalService.Service
}

// Engine implements Phase 4 Verifiable Self-Correction & Learning (AIA-11 §21, AIA-13 §21.3, Plan Phase 4):
//   - Discrepancy Detection -> L0_RECORDED (with Baseline)
//   - Evidence-Grounded Falsifiable RCA (decoupled from intervention outcome)
//   - Table-Driven 3-Tier Correction Profile -> L1_APPLIED / Approval / Blocked
//   - Baseline + Control/Replay + Regression Verification -> L2_VERIFIED | INEFFECTIVE | REGRESSED | INCONCLUSIVE
//   - Independent Cross-Cycle / Cross-Scenario Generalization -> L3_GENERALIZED
type Engine struct {
	deps                   Dependencies
	minSamplesForEffective int
	minGeneralizations     int
}

func NewEngine(deps Dependencies) *Engine {
	return &Engine{
		deps:                   deps,
		minSamplesForEffective: 3,
		minGeneralizations:     1,
	}
}

func (e *Engine) SetMinSamplesForEffective(n int) {
	if n > 0 {
		e.minSamplesForEffective = n
	}
}

func (e *Engine) SetMinGeneralizations(n int) {
	if n > 0 {
		e.minGeneralizations = n
	}
}

type DiscrepancyInput struct {
	CycleID         *string
	DecisionID      *string
	TaskID          *string
	ProjectID       *string
	Domain          string
	Title           string
	MetricName      string
	ExpectedMetric  float64
	ActualMetric    float64
	LowerIsBetter   bool
	AnomalyThreshold float64
	ExpectedContext map[string]any
	ActualContext   map[string]any
}

// RecordDiscrepancy measures Expected vs Actual outcome discrepancy and initializes an L0_RECORDED LearningRecord with Baseline.
func (e *Engine) RecordDiscrepancy(ctx context.Context, in DiscrepancyInput) (*domainLearning.OutcomeEvaluation, *domainLearning.LearningRecord, error) {
	if e.deps.LearningRepo == nil {
		return nil, nil, fmt.Errorf("learning repository is not configured")
	}
	now := clock.Now(ctx)
	domainName := strings.TrimSpace(in.Domain)
	if domainName == "" {
		domainName = "general"
	}

	rawDiff := math.Abs(in.ActualMetric - in.ExpectedMetric)
	threshold := in.AnomalyThreshold
	if threshold <= 0 {
		threshold = 0.05
	}
	isAnomaly := rawDiff >= threshold
	status := domainLearning.EvalStatusEvaluated
	if isAnomaly {
		status = domainLearning.EvalStatusNeedsRCA
	}

	expPayload := map[string]any{
		"metric_name":     in.MetricName,
		"expected_metric": in.ExpectedMetric,
		"lower_is_better": in.LowerIsBetter,
	}
	for k, v := range in.ExpectedContext {
		expPayload[k] = v
	}
	actPayload := map[string]any{
		"metric_name":   in.MetricName,
		"actual_metric": in.ActualMetric,
	}
	for k, v := range in.ActualContext {
		actPayload[k] = v
	}
	expJSON, _ := json.Marshal(expPayload)
	actJSON, _ := json.Marshal(actPayload)

	eval := &domainLearning.OutcomeEvaluation{
		ID:                  uuid.New().String(),
		CycleID:             in.CycleID,
		DecisionID:          in.DecisionID,
		TaskID:              in.TaskID,
		ProjectID:           in.ProjectID,
		Domain:              domainName,
		ExpectedOutcomeJSON: string(expJSON),
		ActualOutcomeJSON:   string(actJSON),
		DiscrepancyScore:    rawDiff,
		IsAnomaly:           isAnomaly,
		EvaluationStatus:    status,
		EvaluatedAt:         now,
	}
	if err := e.deps.LearningRepo.CreateOutcomeEvaluation(ctx, eval); err != nil {
		return nil, nil, err
	}

	baselineJSON, _ := json.Marshal(map[string]any{
		"metric_name":       in.MetricName,
		"expected_metric":   in.ExpectedMetric,
		"baseline_metric":   in.ActualMetric,
		"discrepancy_score": rawDiff,
		"lower_is_better":   in.LowerIsBetter,
		"recorded_at":       now.Format(time.RFC3339),
	})

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = fmt.Sprintf("[%s] Discrepancy on %s (%.4f vs %.4f)", domainName, in.MetricName, in.ActualMetric, in.ExpectedMetric)
	}

	rec := &domainLearning.LearningRecord{
		ID:                      uuid.New().String(),
		ProjectID:               in.ProjectID,
		Domain:                  domainName,
		SourceCycleID:           in.CycleID,
		OutcomeEvaluationID:     eval.ID,
		Title:                   title,
		LearningLevel:           domainLearning.LevelL0Recorded,
		EffectivenessVerdict:    domainLearning.VerdictPending,
		ComparisonMethod:        domainLearning.CompPrePostOnly,
		BaselineJSON:            string(baselineJSON),
		HypothesisJSON:          `{"status":"AWAITING_RCA"}`,
		InterventionJSON:        `{}`,
		VerificationJSON:        `{}`,
		RegressionJSON:          `{}`,
		FollowupJSON:            `[]`,
		EpistemicConfidence:     0.20,
		GeneralizedCycleIDsJSON: `[]`,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := e.deps.LearningRepo.CreateLearningRecord(ctx, rec); err != nil {
		return eval, nil, err
	}

	e.appendEvent(ctx, eval.ID, "OUTCOME_DISCREPANCY_RECORDED", map[string]any{
		"learning_record_id": rec.ID,
		"discrepancy_score":  rawDiff,
		"is_anomaly":         isAnomaly,
		"learning_level":     rec.LearningLevel,
	}, now)

	return eval, rec, nil
}

type RCAInput struct {
	OutcomeEvaluationID   string
	LearningRecordID      string
	Category              domainLearning.RootCauseCategory
	Narrative             string
	FalsifiableHypothesis string
	CausalEvidenceRefs    []string
	CompetingFactors      []string
	InitialConfidence     float64
}

// AnalyzeRootCause enforces evidence-grounded, falsifiable RCA hypotheses and links them to the L0 LearningRecord.
func (e *Engine) AnalyzeRootCause(ctx context.Context, in RCAInput) (*domainLearning.RootCauseAnalysis, error) {
	if len(in.CausalEvidenceRefs) == 0 || strings.TrimSpace(in.FalsifiableHypothesis) == "" {
		return nil, domainLearning.ErrUngroundedRCA
	}
	cleanRefs := make([]string, 0, len(in.CausalEvidenceRefs))
	for _, ref := range in.CausalEvidenceRefs {
		if r := strings.TrimSpace(ref); r != "" {
			cleanRefs = append(cleanRefs, r)
		}
	}
	if len(cleanRefs) == 0 {
		return nil, domainLearning.ErrUngroundedRCA
	}

	refsJSON, _ := json.Marshal(cleanRefs)
	compJSON, _ := json.Marshal(in.CompetingFactors)
	conf := in.InitialConfidence
	if conf <= 0 {
		conf = 0.60
	}
	if conf > 1.0 {
		conf = 1.0
	}

	now := clock.Now(ctx)
	rca := &domainLearning.RootCauseAnalysis{
		ID:                     uuid.New().String(),
		OutcomeEvaluationID:    in.OutcomeEvaluationID,
		RootCauseCategory:      in.Category,
		AnalysisNarrative:      in.Narrative,
		FalsifiableHypothesis:  in.FalsifiableHypothesis,
		CausalEvidenceRefsJSON: string(refsJSON),
		CompetingFactorsJSON:   string(compJSON),
		RCAValidityStatus:      domainLearning.RCAProposed,
		Confidence:             conf,
		CreatedAt:              now,
	}
	if err := e.deps.LearningRepo.CreateRCA(ctx, rca); err != nil {
		return nil, err
	}

	if in.LearningRecordID != "" {
		if rec, err := e.deps.LearningRepo.GetLearningRecord(ctx, in.LearningRecordID); err == nil && rec != nil {
			rcaID := rca.ID
			rec.RCAID = &rcaID
			hypJSON, _ := json.Marshal(map[string]any{
				"rca_id":                 rca.ID,
				"category":               rca.RootCauseCategory,
				"falsifiable_hypothesis": rca.FalsifiableHypothesis,
				"causal_evidence_refs":   cleanRefs,
				"competing_factors":      in.CompetingFactors,
			})
			rec.HypothesisJSON = string(hypJSON)
			_ = e.deps.LearningRepo.UpdateLearningRecord(ctx, rec)
		}
	}

	return rca, nil
}

type CorrectionInput struct {
	RCAID             string
	LearningRecordID  string
	Scope             domainLearning.ScopeLevel
	CorrectionType    domainLearning.CorrectionType
	TargetEntityRef   string // e.g. "claim:<id>" or "strategy:<name>" or "policy:core"
	BeforeState       map[string]any
	ProposedState     map[string]any
	DecisionID        *string
	TaskID            *string
	ProjectID         *string
}

// ProposeAndApplyCorrection routes the proposal through the Table-Driven 3-Tier Profile:
//   - TIER1_AUTO: applies immediately and promotes LearningRecord to L1_APPLIED.
//   - TIER2_APPROVAL: creates a human Approval gate and holds in PENDING until approved.
//   - TIER3_BLOCKED: rejects immediately (Scope C / CODE frozen until Phase 6).
func (e *Engine) ProposeAndApplyCorrection(ctx context.Context, in CorrectionInput) (*domainLearning.CorrectionAction, *domainLearning.LearningRecord, error) {
	tier, authLvl, valLvl := domainLearning.ResolveCorrectionProfile(in.Scope, in.CorrectionType)
	beforeJSON, _ := json.Marshal(in.BeforeState)
	proposedJSON, _ := json.Marshal(in.ProposedState)
	now := clock.Now(ctx)

	ca := &domainLearning.CorrectionAction{
		ID:                 uuid.New().String(),
		RCAID:              in.RCAID,
		ScopeLevel:         in.Scope,
		CorrectionType:     in.CorrectionType,
		AuthorizationLevel: authLvl,
		ValidationLevel:    valLvl,
		TierProfile:        tier,
		TargetEntityRef:    in.TargetEntityRef,
		BeforeStateJSON:    string(beforeJSON),
		ProposedStateJSON:  string(proposedJSON),
		RegressionStatus:   domainLearning.RegressionPending,
		VerificationStatus: domainLearning.VerifPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if ca.ScopeLevel == "" {
		ca.ScopeLevel = domainLearning.ScopeA
	}

	switch tier {
	case domainLearning.Tier3Blocked:
		ca.VerificationStatus = domainLearning.VerifBlocked
		_ = e.deps.LearningRepo.CreateCorrectionAction(ctx, ca)
		if e.deps.AuditRepo != nil {
			riskStr := string(governance.RiskTierCritical)
			govAct := string(governance.GovernanceActionBlocked)
			_ = e.deps.AuditRepo.Append(ctx, &repository.AuditLogEntry{
				ID:               domain.ID(uuid.New().String()),
				ActorType:        "AGENT",
				ActorID:          "self-correction-engine",
				Action:           "CORRECTION_TIER3_BLOCKED",
				ResourceType:     "CORRECTION_ACTION",
				ResourceID:       ca.ID,
				Outcome:          govAct,
				RiskTier:         &riskStr,
				GovernanceAction: &govAct,
				CreatedAt:        now,
			})
		}
		return ca, nil, domainLearning.ErrTier3FrozenUntilPhase6

	case domainLearning.Tier2Approval:
		decID, err := e.ensureBoundDecision(ctx, in.DecisionID, in.TaskID, in.ProjectID,
			fmt.Sprintf("Tier 2 Correction Approval (%s on %s)", in.CorrectionType, in.TargetEntityRef))
		if err == nil && e.deps.ApprovalSvc != nil {
			detailBytes, _ := json.Marshal(map[string]any{
				"correction_type":   in.CorrectionType,
				"scope_level":       ca.ScopeLevel,
				"authorization":     authLvl,
				"target_entity_ref": in.TargetEntityRef,
				"proposed_state":    in.ProposedState,
			})
			appr, aErr := e.deps.ApprovalSvc.RequestApproval(ctx, approvalService.RequestApprovalInput{
				RequesterEntity:     "AGENT:self-correction",
				OperationType:       fmt.Sprintf("SELF_CORRECTION_%s", in.CorrectionType),
				OperationDetailJSON: string(detailBytes),
				RiskTier:            governance.RiskTierHigh,
				TaskID:              in.TaskID,
				DecisionID:          decID,
			})
			if aErr == nil && appr != nil {
				ca.ApprovalID = &appr.ID
			}
		}
		if err := e.deps.LearningRepo.CreateCorrectionAction(ctx, ca); err != nil {
			return nil, nil, err
		}
		var rec *domainLearning.LearningRecord
		if in.LearningRecordID != "" {
			rec, _ = e.deps.LearningRepo.GetLearningRecord(ctx, in.LearningRecordID)
			if rec != nil {
				caID := ca.ID
				rec.CorrectionActionID = &caID
				intervJSON, _ := json.Marshal(map[string]any{
					"correction_action_id": ca.ID,
					"tier_profile":         tier,
					"status":               "WAITING_HUMAN_APPROVAL",
					"target_entity_ref":    ca.TargetEntityRef,
					"proposed_state":       in.ProposedState,
				})
				rec.InterventionJSON = string(intervJSON)
				_ = e.deps.LearningRepo.UpdateLearningRecord(ctx, rec)
			}
		}
		return ca, rec, nil

	default: // Tier1Auto
		ca.AppliedStateJSON = string(proposedJSON)
		ca.VerificationStatus = domainLearning.VerifApplied
		if err := e.deps.LearningRepo.CreateCorrectionAction(ctx, ca); err != nil {
			return nil, nil, err
		}
		e.applyEntityMutation(ctx, ca.TargetEntityRef, in.ProposedState)

		var rec *domainLearning.LearningRecord
		if in.LearningRecordID != "" {
			rec, _ = e.deps.LearningRepo.GetLearningRecord(ctx, in.LearningRecordID)
			if rec != nil {
				caID := ca.ID
				rec.CorrectionActionID = &caID
				rec.LearningLevel = domainLearning.LevelL1Applied
				intervJSON, _ := json.Marshal(map[string]any{
					"correction_action_id": ca.ID,
					"tier_profile":         tier,
					"authorization_level":  authLvl,
					"target_entity_ref":    ca.TargetEntityRef,
					"before_state":         in.BeforeState,
					"applied_state":        in.ProposedState,
					"applied_at":           now.Format(time.RFC3339),
				})
				rec.InterventionJSON = string(intervJSON)
				_ = e.deps.LearningRepo.UpdateLearningRecord(ctx, rec)
			}
		}
		return ca, rec, nil
	}
}

// ApplyApprovedCorrection executes a Tier 2 correction after human approval and promotes the LearningRecord to L1_APPLIED.
func (e *Engine) ApplyApprovedCorrection(ctx context.Context, correctionID string, learningRecordID string) (*domainLearning.CorrectionAction, *domainLearning.LearningRecord, error) {
	ca, err := e.deps.LearningRepo.GetCorrectionAction(ctx, correctionID)
	if err != nil {
		return nil, nil, err
	}
	ca.AppliedStateJSON = ca.ProposedStateJSON
	ca.VerificationStatus = domainLearning.VerifApplied
	if err := e.deps.LearningRepo.UpdateCorrectionAction(ctx, ca); err != nil {
		return nil, nil, err
	}

	var proposed map[string]any
	if json.Unmarshal([]byte(ca.ProposedStateJSON), &proposed) == nil {
		e.applyEntityMutation(ctx, ca.TargetEntityRef, proposed)
	}

	var rec *domainLearning.LearningRecord
	if learningRecordID != "" {
		rec, _ = e.deps.LearningRepo.GetLearningRecord(ctx, learningRecordID)
		if rec != nil {
			rec.LearningLevel = domainLearning.LevelL1Applied
			intervJSON, _ := json.Marshal(map[string]any{
				"correction_action_id": ca.ID,
				"tier_profile":         ca.TierProfile,
				"target_entity_ref":    ca.TargetEntityRef,
				"applied_state_json":   ca.AppliedStateJSON,
				"status":               "APPLIED_AFTER_APPROVAL",
			})
			rec.InterventionJSON = string(intervJSON)
			_ = e.deps.LearningRepo.UpdateLearningRecord(ctx, rec)
		}
	}
	return ca, rec, nil
}

type VerificationInput struct {
	LearningRecordID      string
	ComparisonMethod      domainLearning.ComparisonMethod
	BaselineMetric        float64
	PostMetric            float64
	LowerIsBetter         bool
	HasControl            bool
	ControlBaselineMetric float64
	ControlPostMetric     float64
	MinExcessImprovement  float64
	SampleSize            int
	MinSamples            int
	RegressionPassed      bool
	RegressionDetails     map[string]any
	HypothesisFalsified   bool
	ProposedConfidence    float64
}

// VerifyCorrection enforces Baseline + Control/Replay + Regression checks and determines whether
// the correction reaches L2_VERIFIED (EFFECTIVE), INEFFECTIVE (including tailwind false positives),
// REGRESSED (with auto-rollback), or INCONCLUSIVE (with confidence cap <= 0.45).
func (e *Engine) VerifyCorrection(ctx context.Context, in VerificationInput) (*domainLearning.LearningRecord, error) {
	rec, err := e.deps.LearningRepo.GetLearningRecord(ctx, in.LearningRecordID)
	if err != nil {
		return nil, err
	}

	var ca *domainLearning.CorrectionAction
	if rec.CorrectionActionID != nil && *rec.CorrectionActionID != "" {
		ca, _ = e.deps.LearningRepo.GetCorrectionAction(ctx, *rec.CorrectionActionID)
	}

	compMethod := in.ComparisonMethod
	if compMethod == "" {
		compMethod = domainLearning.CompPrePostOnly
	}
	rec.ComparisonMethod = compMethod

	minSamples := in.MinSamples
	if minSamples <= 0 {
		minSamples = e.minSamplesForEffective
	}

	rawDelta := in.PostMetric - in.BaselineMetric
	if in.LowerIsBetter {
		rawDelta = in.BaselineMetric - in.PostMetric
	}

	controlDelta := 0.0
	if in.HasControl {
		controlDelta = in.ControlPostMetric - in.ControlBaselineMetric
		if in.LowerIsBetter {
			controlDelta = in.ControlBaselineMetric - in.ControlPostMetric
		}
	}
	excessDelta := rawDelta - controlDelta

	regPayload := map[string]any{
		"passed":     in.RegressionPassed,
		"checked_at": clock.Now(ctx).Format(time.RFC3339),
	}
	for k, v := range in.RegressionDetails {
		regPayload[k] = v
	}
	regJSON, _ := json.Marshal(regPayload)
	rec.RegressionJSON = string(regJSON)

	// 1. Hard Non-Regression Gate: Any regression triggers immediate rollback & anti-pattern preservation
	if !in.RegressionPassed {
		rec.EffectivenessVerdict = domainLearning.VerdictRegressed
		if in.ProposedConfidence > 0 {
			rec.EpistemicConfidence = math.Min(in.ProposedConfidence, 0.85)
		} else {
			rec.EpistemicConfidence = 0.80 // High certainty that this intervention caused regression
		}

		if ca != nil {
			ca.RegressionStatus = domainLearning.RegressionFailed
			ca.VerificationStatus = domainLearning.VerifRegressed
			ca.AppliedStateJSON = ca.BeforeStateJSON // Roll back to pre-correction state
			_ = e.deps.LearningRepo.UpdateCorrectionAction(ctx, ca)

			var before map[string]any
			if json.Unmarshal([]byte(ca.BeforeStateJSON), &before) == nil {
				e.applyEntityMutation(ctx, ca.TargetEntityRef, before)
			}
		}

		// Decouple RCA truth from intervention failure
		if rec.RCAID != nil && *rec.RCAID != "" {
			rcaStatus := domainLearning.RCAInterventionMismatch
			if in.HypothesisFalsified {
				rcaStatus = domainLearning.RCAFalsified
			}
			_ = e.deps.LearningRepo.UpdateRCAValidity(ctx, *rec.RCAID, rcaStatus)
		}

		verifJSON, _ := json.Marshal(map[string]any{
			"verdict":          rec.EffectivenessVerdict,
			"raw_improvement":  rawDelta,
			"excess_delta":     excessDelta,
			"sample_size":      in.SampleSize,
			"rolled_back":      true,
			"anti_pattern_set": true,
		})
		rec.VerificationJSON = string(verifJSON)
		if err := e.deps.LearningRepo.UpdateLearningRecord(ctx, rec); err != nil {
			return nil, err
		}
		return rec, nil
	}

	if ca != nil {
		ca.RegressionStatus = domainLearning.RegressionPassed
	}

	// 2. Epistemic Confidence Ceiling & Minimum Sample Gate:
	// Without a control group / PiT replay / cohort OR with insufficient samples, verdict CANNOT be EFFECTIVE.
	if compMethod == domainLearning.CompPrePostOnly || in.SampleSize < minSamples {
		rec.EffectivenessVerdict = domainLearning.VerdictInconclusive
		conf := in.ProposedConfidence
		if conf <= 0 {
			conf = 0.35
		}
		if conf > domainLearning.MaxPrePostConfidence {
			conf = domainLearning.MaxPrePostConfidence
		}
		rec.EpistemicConfidence = conf

		if ca != nil {
			ca.VerificationStatus = domainLearning.VerifInconclusive
			_ = e.deps.LearningRepo.UpdateCorrectionAction(ctx, ca)
		}
		if rec.RCAID != nil && *rec.RCAID != "" {
			_ = e.deps.LearningRepo.UpdateRCAValidity(ctx, *rec.RCAID, domainLearning.RCAUncertain)
		}

		verifJSON, _ := json.Marshal(map[string]any{
			"verdict":           rec.EffectivenessVerdict,
			"comparison_method": compMethod,
			"raw_improvement":   rawDelta,
			"excess_delta":      excessDelta,
			"sample_size":       in.SampleSize,
			"min_samples":       minSamples,
			"confidence_capped": true,
			"reason":            "Insufficient counterfactual control or sample size; refusing to claim EFFECTIVE learning",
		})
		rec.VerificationJSON = string(verifJSON)
		if err := e.deps.LearningRepo.UpdateLearningRecord(ctx, rec); err != nil {
			return nil, err
		}
		return rec, nil
	}

	// 3. Counterfactual / Tailwind Filter ("修正後變好，不一定是修正造成的"):
	// Must outperform control/replay baseline by > MinExcessImprovement.
	minExcess := in.MinExcessImprovement
	if minExcess <= 0 {
		minExcess = 1e-6
	}
	if excessDelta < minExcess {
		rec.EffectivenessVerdict = domainLearning.VerdictIneffective
		conf := in.ProposedConfidence
		if conf <= 0 {
			conf = 0.70
		}
		rec.EpistemicConfidence = math.Min(conf, 0.90)

		if ca != nil {
			ca.VerificationStatus = domainLearning.VerifIneffective
			_ = e.deps.LearningRepo.UpdateCorrectionAction(ctx, ca)
		}
		if rec.RCAID != nil && *rec.RCAID != "" {
			rcaSt := domainLearning.RCAInterventionMismatch
			if in.HypothesisFalsified {
				rcaSt = domainLearning.RCAFalsified
			}
			_ = e.deps.LearningRepo.UpdateRCAValidity(ctx, *rec.RCAID, rcaSt)
		}

		verifJSON, _ := json.Marshal(map[string]any{
			"verdict":               rec.EffectivenessVerdict,
			"comparison_method":     compMethod,
			"raw_improvement":       rawDelta,
			"control_improvement":   controlDelta,
			"excess_delta":          excessDelta,
			"sample_size":           in.SampleSize,
			"tailwind_confounded":   rawDelta > 0 && excessDelta <= 0,
			"reason":                "Post-intervention gain did not beat counterfactual/control benchmark",
		})
		rec.VerificationJSON = string(verifJSON)
		if err := e.deps.LearningRepo.UpdateLearningRecord(ctx, rec); err != nil {
			return nil, err
		}
		return rec, nil
	}

	// 4. Verified Effective Improvement -> Promote to L2_VERIFIED!
	rec.EffectivenessVerdict = domainLearning.VerdictEffective
	rec.LearningLevel = domainLearning.LevelL2Verified
	conf := in.ProposedConfidence
	if conf <= 0 {
		conf = 0.80
	}
	if conf > 0.99 {
		conf = 0.99
	}
	rec.EpistemicConfidence = conf

	if ca != nil {
		ca.VerificationStatus = domainLearning.VerifEffective
		_ = e.deps.LearningRepo.UpdateCorrectionAction(ctx, ca)
	}
	if rec.RCAID != nil && *rec.RCAID != "" {
		_ = e.deps.LearningRepo.UpdateRCAValidity(ctx, *rec.RCAID, domainLearning.RCASupported)
	}
	_ = e.deps.LearningRepo.UpdateOutcomeEvaluationStatus(ctx, rec.OutcomeEvaluationID, domainLearning.EvalStatusResolved)

	verifJSON, _ := json.Marshal(map[string]any{
		"verdict":             rec.EffectivenessVerdict,
		"comparison_method":   compMethod,
		"raw_improvement":     rawDelta,
		"control_improvement": controlDelta,
		"excess_delta":        excessDelta,
		"sample_size":         in.SampleSize,
		"promoted_to":         rec.LearningLevel,
	})
	rec.VerificationJSON = string(verifJSON)
	if err := e.deps.LearningRepo.UpdateLearningRecord(ctx, rec); err != nil {
		return nil, err
	}

	e.appendEvent(ctx, rec.ID, "LEARNING_VERIFIED_L2", map[string]any{
		"learning_level": rec.LearningLevel,
		"verdict":        rec.EffectivenessVerdict,
		"excess_delta":   excessDelta,
		"sample_size":    in.SampleSize,
	}, clock.Now(ctx))

	return rec, nil
}

type GeneralizationInput struct {
	LearningRecordID string
	CycleID          string
	ScenarioKey      string
	OutcomeImproved  bool
	RegressionPassed bool
	Notes            string
}

// RecordCrossScenarioApplication records reuse of an L2_VERIFIED learning in a subsequent, distinct cycle/scenario.
// Only independent cross-cycle success without regression promotes the record to L3_GENERALIZED (AIA-11 §21).
func (e *Engine) RecordCrossScenarioApplication(ctx context.Context, in GeneralizationInput) (*domainLearning.LearningRecord, error) {
	rec, err := e.deps.LearningRepo.GetLearningRecord(ctx, in.LearningRecordID)
	if err != nil {
		return nil, err
	}
	if rec.EffectivenessVerdict != domainLearning.VerdictEffective ||
		(rec.LearningLevel != domainLearning.LevelL2Verified && rec.LearningLevel != domainLearning.LevelL3Generalized) {
		return nil, fmt.Errorf("only EFFECTIVE L2_VERIFIED/L3_GENERALIZED records can be generalized (current: %s / %s)",
			rec.LearningLevel, rec.EffectivenessVerdict)
	}

	cycleID := strings.TrimSpace(in.CycleID)
	if cycleID == "" || (rec.SourceCycleID != nil && *rec.SourceCycleID == cycleID) {
		return nil, domainLearning.ErrSameCycleGeneralizationForbidden
	}

	now := clock.Now(ctx)
	var followups []map[string]any
	_ = json.Unmarshal([]byte(rec.FollowupJSON), &followups)
	followups = append(followups, map[string]any{
		"cycle_id":          cycleID,
		"scenario_key":      in.ScenarioKey,
		"outcome_improved":  in.OutcomeImproved,
		"regression_passed": in.RegressionPassed,
		"notes":             in.Notes,
		"recorded_at":       now.Format(time.RFC3339),
	})
	followupBytes, _ := json.Marshal(followups)
	rec.FollowupJSON = string(followupBytes)

	if !in.RegressionPassed {
		rec.EffectivenessVerdict = domainLearning.VerdictRegressed
		rec.LearningLevel = domainLearning.LevelL1Applied
		_ = e.deps.LearningRepo.UpdateLearningRecord(ctx, rec)
		return rec, nil
	}

	if !in.OutcomeImproved {
		rec.EpistemicConfidence = math.Max(0.30, rec.EpistemicConfidence-0.15)
		_ = e.deps.LearningRepo.UpdateLearningRecord(ctx, rec)
		return rec, nil
	}

	var genCycles []string
	_ = json.Unmarshal([]byte(rec.GeneralizedCycleIDsJSON), &genCycles)
	exists := false
	for _, cid := range genCycles {
		if cid == cycleID {
			exists = true
			break
		}
	}
	if !exists {
		genCycles = append(genCycles, cycleID)
	}
	genBytes, _ := json.Marshal(genCycles)
	rec.GeneralizedCycleIDsJSON = string(genBytes)

	if len(genCycles) >= e.minGeneralizations {
		rec.LearningLevel = domainLearning.LevelL3Generalized
		rec.EpistemicConfidence = math.Min(0.98, rec.EpistemicConfidence+0.08)
	}

	if err := e.deps.LearningRepo.UpdateLearningRecord(ctx, rec); err != nil {
		return nil, err
	}

	if rec.LearningLevel == domainLearning.LevelL3Generalized {
		e.appendEvent(ctx, rec.ID, "LEARNING_GENERALIZED_L3", map[string]any{
			"cycle_id":               cycleID,
			"scenario_key":           in.ScenarioKey,
			"generalized_cycle_count": len(genCycles),
		}, now)
	}

	return rec, nil
}

func (e *Engine) applyEntityMutation(ctx context.Context, targetRef string, state map[string]any) {
	if strings.HasPrefix(targetRef, "claim:") && e.deps.KnowledgeRepo != nil {
		claimID := strings.TrimPrefix(targetRef, "claim:")
		if stStr, ok := state["assertion_status"].(string); ok && stStr != "" {
			isKnow, _ := state["is_knowledge"].(bool)
			_ = e.deps.KnowledgeRepo.UpdateClaimStatus(ctx, claimID, knowledge.AssertionStatus(stStr), isKnow)
		}
	}
}

func (e *Engine) ensureBoundDecision(ctx context.Context, existingDecID *string, taskID *string, projectID *string, summary string) (*string, error) {
	if existingDecID != nil && *existingDecID != "" {
		return existingDecID, nil
	}
	if taskID != nil && *taskID != "" {
		return nil, nil // approvals table allows task_id OR decision_id
	}
	if e.deps.DecisionRepo == nil {
		return nil, fmt.Errorf("decision repository unavailable")
	}
	dec := &decision.Decision{
		ID:               uuid.New().String(),
		ProjectID:        projectID,
		ActionType:       decision.ActionAskHuman,
		IntentSummary:    summary,
		Justification:    "Governed Tier 2 correction requires explicit human review",
		GovernanceResult: decision.GovernanceRequiredApproval,
		CreatedAt:        clock.Now(ctx),
	}
	if err := e.deps.DecisionRepo.Create(ctx, dec, nil); err != nil {
		return nil, err
	}
	return &dec.ID, nil
}

func (e *Engine) appendEvent(ctx context.Context, aggregateID string, eventType string, payload map[string]any, occurredAt time.Time) {
	if e.deps.EventHistoryRepo == nil || aggregateID == "" {
		return
	}
	raw, _ := json.Marshal(payload)
	evtID := uuid.New().String()
	_ = e.deps.EventHistoryRepo.Append(ctx, nil, &continuity.EventHistory{
		EventID:       evtID,
		EventType:     eventType,
		AggregateType: "LEARNING",
		AggregateID:   aggregateID,
		CorrelationID: aggregateID,
		CausationID:   evtID,
		ActorID:       "SYSTEM:learning_engine",
		PayloadJSON:   string(raw),
		OccurredAt:    occurredAt,
	})
}
