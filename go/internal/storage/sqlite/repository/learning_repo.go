package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wade/aia/internal/autonomy/clock"
	"github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/storage/sqlite"
)

// LearningRepository persists Phase 4 outcome evaluations, falsifiable RCAs,
// 3-tier correction actions, L0–L3 learning records, and governed goal evolution proposals.
type LearningRepository struct {
	db *sqlite.DB
}

func NewLearningRepository(db *sqlite.DB) *LearningRepository {
	return &LearningRepository{db: db}
}

// ---------------------------------------------------------------------------
// 1. Outcome Evaluations
// ---------------------------------------------------------------------------

func (r *LearningRepository) CreateOutcomeEvaluation(ctx context.Context, ev *learning.OutcomeEvaluation) error {
	now := clock.Now(ctx)
	if ev.EvaluatedAt.IsZero() {
		ev.EvaluatedAt = now
	}
	if ev.Domain == "" {
		ev.Domain = "general"
	}
	if ev.EvaluationStatus == "" {
		ev.EvaluationStatus = learning.EvalStatusEvaluated
	}
	isAnom := 0
	if ev.IsAnomaly {
		isAnom = 1
	}

	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO outcome_evaluations (
			id, cycle_id, decision_id, task_id, project_id, domain,
			expected_outcome_json, actual_outcome_json, discrepancy_score,
			is_anomaly, evaluation_status, evaluated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		ev.ID,
		nullStrPtr(ev.CycleID),
		nullStrPtr(ev.DecisionID),
		nullStrPtr(ev.TaskID),
		nullStrPtr(ev.ProjectID),
		ev.Domain,
		ev.ExpectedOutcomeJSON,
		ev.ActualOutcomeJSON,
		ev.DiscrepancyScore,
		isAnom,
		string(ev.EvaluationStatus),
		ev.EvaluatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *LearningRepository) GetOutcomeEvaluation(ctx context.Context, id string) (*learning.OutcomeEvaluation, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, cycle_id, decision_id, task_id, project_id, domain,
		       expected_outcome_json, actual_outcome_json, discrepancy_score,
		       is_anomaly, evaluation_status, evaluated_at
		FROM outcome_evaluations WHERE id = ?
	`, id)

	var ev learning.OutcomeEvaluation
	var cycID, decID, tskID, projID sql.NullString
	var isAnom int
	var st, evalAt string
	if err := row.Scan(
		&ev.ID, &cycID, &decID, &tskID, &projID, &ev.Domain,
		&ev.ExpectedOutcomeJSON, &ev.ActualOutcomeJSON, &ev.DiscrepancyScore,
		&isAnom, &st, &evalAt,
	); err != nil {
		return nil, err
	}
	ev.CycleID = strPtrFromNull(cycID)
	ev.DecisionID = strPtrFromNull(decID)
	ev.TaskID = strPtrFromNull(tskID)
	ev.ProjectID = strPtrFromNull(projID)
	ev.IsAnomaly = isAnom == 1
	ev.EvaluationStatus = learning.EvaluationStatus(st)
	ev.EvaluatedAt, _ = time.Parse(time.RFC3339, evalAt)
	return &ev, nil
}

func (r *LearningRepository) UpdateOutcomeEvaluationStatus(ctx context.Context, id string, status learning.EvaluationStatus) error {
	_, err := r.db.RawDB().ExecContext(ctx, `UPDATE outcome_evaluations SET evaluation_status = ? WHERE id = ?`, string(status), id)
	return err
}

// ---------------------------------------------------------------------------
// 2. Root Cause Analyses
// ---------------------------------------------------------------------------

func (r *LearningRepository) CreateRCA(ctx context.Context, rca *learning.RootCauseAnalysis) error {
	if err := rca.Validate(); err != nil {
		return err
	}
	if rca.CreatedAt.IsZero() {
		rca.CreatedAt = clock.Now(ctx)
	}
	if rca.CompetingFactorsJSON == "" {
		rca.CompetingFactorsJSON = "[]"
	}
	if rca.RCAValidityStatus == "" {
		rca.RCAValidityStatus = learning.RCAProposed
	}

	dbCategory := string(rca.RootCauseCategory)
	switch rca.RootCauseCategory {
	case learning.RCIncorrectFact, learning.RCBiasedJudgment, learning.RCFlawedStrategy, learning.RCAmbiguousPrompt, learning.RCWorkflowDefect, learning.RCExternalUnexpected:
	case learning.RCEvaluationError:
		dbCategory = string(learning.RCBiasedJudgment)
	case learning.RCGoalAssumptionError:
		dbCategory = string(learning.RCFlawedStrategy)
	default:
		dbCategory = string(learning.RCFlawedStrategy)
	}

	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO root_cause_analyses (
			id, outcome_evaluation_id, root_cause_category, analysis_narrative,
			falsifiable_hypothesis, causal_evidence_refs_json, competing_factors_json,
			rca_validity_status, confidence, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		rca.ID,
		rca.OutcomeEvaluationID,
		dbCategory,
		rca.AnalysisNarrative,
		rca.FalsifiableHypothesis,
		rca.CausalEvidenceRefsJSON,
		rca.CompetingFactorsJSON,
		string(rca.RCAValidityStatus),
		rca.Confidence,
		rca.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *LearningRepository) GetRCA(ctx context.Context, id string) (*learning.RootCauseAnalysis, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, outcome_evaluation_id, root_cause_category, analysis_narrative,
		       falsifiable_hypothesis, causal_evidence_refs_json, competing_factors_json,
		       rca_validity_status, confidence, created_at
		FROM root_cause_analyses WHERE id = ?
	`, id)
	var rca learning.RootCauseAnalysis
	var cat, valSt, ca string
	if err := row.Scan(
		&rca.ID, &rca.OutcomeEvaluationID, &cat, &rca.AnalysisNarrative,
		&rca.FalsifiableHypothesis, &rca.CausalEvidenceRefsJSON, &rca.CompetingFactorsJSON,
		&valSt, &rca.Confidence, &ca,
	); err != nil {
		return nil, err
	}
	rca.RootCauseCategory = learning.RootCauseCategory(cat)
	rca.RCAValidityStatus = learning.RCAValidityStatus(valSt)
	rca.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	return &rca, nil
}

func (r *LearningRepository) UpdateRCAValidity(ctx context.Context, id string, status learning.RCAValidityStatus) error {
	_, err := r.db.RawDB().ExecContext(ctx, `UPDATE root_cause_analyses SET rca_validity_status = ? WHERE id = ?`, string(status), id)
	return err
}

// ---------------------------------------------------------------------------
// 3. Correction Actions
// ---------------------------------------------------------------------------

func (r *LearningRepository) CreateCorrectionAction(ctx context.Context, ca *learning.CorrectionAction) error {
	now := clock.Now(ctx)
	if ca.CreatedAt.IsZero() {
		ca.CreatedAt = now
	}
	ca.UpdatedAt = now
	if ca.RegressionStatus == "" {
		ca.RegressionStatus = learning.RegressionPending
	}

	var applied sql.NullString
	if ca.AppliedStateJSON != "" {
		applied = sql.NullString{String: ca.AppliedStateJSON, Valid: true}
	}

	dbCorrectionType := string(ca.CorrectionType)
	switch ca.CorrectionType {
	case learning.CorrectionFact, learning.CorrectionJudgment, learning.CorrectionStrategy, learning.CorrectionPrompt, learning.CorrectionWorkflow, learning.CorrectionCode:
	case learning.CorrectionEvaluationUpdate, learning.CorrectionGoalProposal:
		dbCorrectionType = string(learning.CorrectionStrategy)
	case learning.CorrectionMemory:
		dbCorrectionType = string(learning.CorrectionFact)
	default:
		dbCorrectionType = string(learning.CorrectionStrategy)
	}

	dbVerifStatus := string(ca.VerificationStatus)
	if ca.VerificationStatus == learning.VerifPartial {
		dbVerifStatus = string(learning.VerifInconclusive)
	}

	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO correction_actions (
			id, rca_id, scope_level, correction_type, authorization_level,
			validation_level, tier_profile, target_entity_ref, before_state_json,
			proposed_state_json, applied_state_json, approval_id, regression_status,
			verification_status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		ca.ID,
		ca.RCAID,
		string(ca.ScopeLevel),
		dbCorrectionType,
		string(ca.AuthorizationLevel),
		string(ca.ValidationLevel),
		string(ca.TierProfile),
		ca.TargetEntityRef,
		ca.BeforeStateJSON,
		ca.ProposedStateJSON,
		applied,
		nullStrPtr(ca.ApprovalID),
		string(ca.RegressionStatus),
		dbVerifStatus,
		ca.CreatedAt.UTC().Format(time.RFC3339),
		ca.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *LearningRepository) GetCorrectionAction(ctx context.Context, id string) (*learning.CorrectionAction, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, rca_id, scope_level, correction_type, authorization_level,
		       validation_level, tier_profile, target_entity_ref, before_state_json,
		       proposed_state_json, applied_state_json, approval_id, regression_status,
		       verification_status, created_at, updated_at
		FROM correction_actions WHERE id = ?
	`, id)

	var ca learning.CorrectionAction
	var sc, ct, al, vl, tp, regSt, verSt, cAt, uAt string
	var applied, appID sql.NullString
	if err := row.Scan(
		&ca.ID, &ca.RCAID, &sc, &ct, &al,
		&vl, &tp, &ca.TargetEntityRef, &ca.BeforeStateJSON,
		&ca.ProposedStateJSON, &applied, &appID, &regSt,
		&verSt, &cAt, &uAt,
	); err != nil {
		return nil, err
	}
	ca.ScopeLevel = learning.ScopeLevel(sc)
	ca.CorrectionType = learning.CorrectionType(ct)
	ca.AuthorizationLevel = learning.AuthorizationLevel(al)
	ca.ValidationLevel = learning.ValidationLevel(vl)
	ca.TierProfile = learning.TierProfile(tp)
	if applied.Valid {
		ca.AppliedStateJSON = applied.String
	}
	ca.ApprovalID = strPtrFromNull(appID)
	ca.RegressionStatus = learning.RegressionStatus(regSt)
	ca.VerificationStatus = learning.VerificationStatus(verSt)
	ca.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
	ca.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
	return &ca, nil
}

func (r *LearningRepository) UpdateCorrectionAction(ctx context.Context, ca *learning.CorrectionAction) error {
	ca.UpdatedAt = clock.Now(ctx)
	var applied sql.NullString
	if ca.AppliedStateJSON != "" {
		applied = sql.NullString{String: ca.AppliedStateJSON, Valid: true}
	}
	_, err := r.db.RawDB().ExecContext(ctx, `
		UPDATE correction_actions
		SET applied_state_json = ?, approval_id = ?, regression_status = ?, verification_status = ?, updated_at = ?
		WHERE id = ?
	`,
		applied,
		nullStrPtr(ca.ApprovalID),
		string(ca.RegressionStatus),
		string(ca.VerificationStatus),
		ca.UpdatedAt.UTC().Format(time.RFC3339),
		ca.ID,
	)
	return err
}

// ---------------------------------------------------------------------------
// 4. Learning Records (L0–L3 + 7 Mandatory Epistemic Comparison Fields)
// ---------------------------------------------------------------------------

func (r *LearningRepository) CreateLearningRecord(ctx context.Context, rec *learning.LearningRecord) error {
	now := clock.Now(ctx)
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now
	if rec.Domain == "" {
		rec.Domain = "general"
	}
	if rec.InterventionJSON == "" {
		rec.InterventionJSON = "{}"
	}
	if rec.VerificationJSON == "" {
		rec.VerificationJSON = "{}"
	}
	if rec.RegressionJSON == "" {
		rec.RegressionJSON = "{}"
	}
	if rec.FollowupJSON == "" {
		rec.FollowupJSON = "{}"
	}
	if rec.GeneralizedCycleIDsJSON == "" {
		rec.GeneralizedCycleIDsJSON = "[]"
	}

	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO learning_records (
			id, project_id, domain, source_cycle_id, outcome_evaluation_id,
			rca_id, correction_action_id, title, learning_level, effectiveness_verdict,
			comparison_method, baseline_json, hypothesis_json, intervention_json,
			verification_json, regression_json, followup_json, epistemic_confidence,
			generalized_cycle_ids_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		rec.ID,
		nullStrPtr(rec.ProjectID),
		rec.Domain,
		nullStrPtr(rec.SourceCycleID),
		rec.OutcomeEvaluationID,
		nullStrPtr(rec.RCAID),
		nullStrPtr(rec.CorrectionActionID),
		rec.Title,
		string(rec.LearningLevel),
		string(rec.EffectivenessVerdict),
		string(rec.ComparisonMethod),
		rec.BaselineJSON,
		rec.HypothesisJSON,
		rec.InterventionJSON,
		rec.VerificationJSON,
		rec.RegressionJSON,
		rec.FollowupJSON,
		rec.EpistemicConfidence,
		rec.GeneralizedCycleIDsJSON,
		rec.CreatedAt.UTC().Format(time.RFC3339),
		rec.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *LearningRepository) UpdateLearningRecord(ctx context.Context, rec *learning.LearningRecord) error {
	rec.UpdatedAt = clock.Now(ctx)
	_, err := r.db.RawDB().ExecContext(ctx, `
		UPDATE learning_records
		SET rca_id = ?, correction_action_id = ?, title = ?, learning_level = ?,
		    effectiveness_verdict = ?, comparison_method = ?, baseline_json = ?,
		    hypothesis_json = ?, intervention_json = ?, verification_json = ?,
		    regression_json = ?, followup_json = ?, epistemic_confidence = ?,
		    generalized_cycle_ids_json = ?, updated_at = ?
		WHERE id = ?
	`,
		nullStrPtr(rec.RCAID),
		nullStrPtr(rec.CorrectionActionID),
		rec.Title,
		string(rec.LearningLevel),
		string(rec.EffectivenessVerdict),
		string(rec.ComparisonMethod),
		rec.BaselineJSON,
		rec.HypothesisJSON,
		rec.InterventionJSON,
		rec.VerificationJSON,
		rec.RegressionJSON,
		rec.FollowupJSON,
		rec.EpistemicConfidence,
		rec.GeneralizedCycleIDsJSON,
		rec.UpdatedAt.UTC().Format(time.RFC3339),
		rec.ID,
	)
	return err
}

func (r *LearningRepository) GetLearningRecord(ctx context.Context, id string) (*learning.LearningRecord, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, project_id, domain, source_cycle_id, outcome_evaluation_id,
		       rca_id, correction_action_id, title, learning_level, effectiveness_verdict,
		       comparison_method, baseline_json, hypothesis_json, intervention_json,
		       verification_json, regression_json, followup_json, epistemic_confidence,
		       generalized_cycle_ids_json, created_at, updated_at
		FROM learning_records
		WHERE id = ? OR (length(?) >= 6 AND id LIKE ? || '%')
		LIMIT 1
	`, id, id, id)
	return scanLearningRecord(row)
}

func (r *LearningRepository) ListLearningRecords(ctx context.Context, projectID string, domainFilter string, limit int) ([]*learning.LearningRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, project_id, domain, source_cycle_id, outcome_evaluation_id,
		       rca_id, correction_action_id, title, learning_level, effectiveness_verdict,
		       comparison_method, baseline_json, hypothesis_json, intervention_json,
		       verification_json, regression_json, followup_json, epistemic_confidence,
		       generalized_cycle_ids_json, created_at, updated_at
		FROM learning_records
		WHERE 1 = 1`
	var args []any
	if projectID != "" {
		query += ` AND (project_id = ? OR project_id IS NULL)`
		args = append(args, projectID)
	}
	if domainFilter != "" {
		query += ` AND domain = ?`
		args = append(args, domainFilter)
	}
	if asOf, ok := clock.AsOf(ctx); ok {
		query += ` AND updated_at <= ?`
		args = append(args, asOf.Format(time.RFC3339))
	}
	query += ` ORDER BY updated_at DESC, rowid DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*learning.LearningRecord
	for rows.Next() {
		rec, err := scanLearningRecord(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

type LearningFilter struct {
	ProjectID string
	Domain    string
	Level     learning.LearningLevel
	Verdict   learning.EffectivenessVerdict
	Limit     int
}

func (r *LearningRepository) QueryLearningRecords(ctx context.Context, f LearningFilter) ([]*learning.LearningRecord, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, project_id, domain, source_cycle_id, outcome_evaluation_id,
		       rca_id, correction_action_id, title, learning_level, effectiveness_verdict,
		       comparison_method, baseline_json, hypothesis_json, intervention_json,
		       verification_json, regression_json, followup_json, epistemic_confidence,
		       generalized_cycle_ids_json, created_at, updated_at
		FROM learning_records
		WHERE 1 = 1`
	var args []any
	if f.ProjectID != "" {
		query += ` AND (project_id = ? OR project_id IS NULL)`
		args = append(args, f.ProjectID)
	}
	if f.Domain != "" {
		query += ` AND domain = ?`
		args = append(args, f.Domain)
	}
	if f.Level != "" {
		query += ` AND learning_level = ?`
		args = append(args, string(f.Level))
	}
	if f.Verdict != "" {
		query += ` AND effectiveness_verdict = ?`
		args = append(args, string(f.Verdict))
	}
	if asOf, ok := clock.AsOf(ctx); ok {
		query += ` AND updated_at <= ?`
		args = append(args, asOf.Format(time.RFC3339))
	}
	query += ` ORDER BY updated_at DESC, rowid DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*learning.LearningRecord
	for rows.Next() {
		rec, err := scanLearningRecord(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

func scanLearningRecord(s rowScanner) (*learning.LearningRecord, error) {
	var rec learning.LearningRecord
	var projID, srcCycID, rcaID, corrID sql.NullString
	var lvl, verd, comp, cAt, uAt string
	if err := s.Scan(
		&rec.ID, &projID, &rec.Domain, &srcCycID, &rec.OutcomeEvaluationID,
		&rcaID, &corrID, &rec.Title, &lvl, &verd,
		&comp, &rec.BaselineJSON, &rec.HypothesisJSON, &rec.InterventionJSON,
		&rec.VerificationJSON, &rec.RegressionJSON, &rec.FollowupJSON, &rec.EpistemicConfidence,
		&rec.GeneralizedCycleIDsJSON, &cAt, &uAt,
	); err != nil {
		return nil, err
	}
	rec.ProjectID = strPtrFromNull(projID)
	rec.SourceCycleID = strPtrFromNull(srcCycID)
	rec.RCAID = strPtrFromNull(rcaID)
	rec.CorrectionActionID = strPtrFromNull(corrID)
	rec.LearningLevel = learning.LearningLevel(lvl)
	rec.EffectivenessVerdict = learning.EffectivenessVerdict(verd)
	rec.ComparisonMethod = learning.ComparisonMethod(comp)
	rec.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
	rec.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
	return &rec, nil
}

// ---------------------------------------------------------------------------
// 5. Goal Evolution Proposals
// ---------------------------------------------------------------------------

func (r *LearningRepository) CreateGoalEvolutionProposal(ctx context.Context, p *learning.GoalEvolutionProposal) error {
	now := clock.Now(ctx)
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.PreviousTargetMetricJSON == "" {
		p.PreviousTargetMetricJSON = "{}"
	}
	if p.ProposedTargetMetricJSON == "" {
		p.ProposedTargetMetricJSON = "{}"
	}
	if p.InvalidatedClaimIDsJSON == "" {
		p.InvalidatedClaimIDsJSON = "[]"
	}

	dbEvolutionMode := string(p.EvolutionMode)
	switch p.EvolutionMode {
	case learning.EvolutionPersistence, learning.EvolutionOptimization, learning.EvolutionRevision:
	default:
		dbEvolutionMode = string(learning.EvolutionRevision)
	}

	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO goal_evolution_proposals (
			id, goal_id, cycle_id, evolution_mode, actor_type, actor_id,
			rationale, previous_target_metric_json, proposed_target_metric_json,
			invalidated_claim_ids_json, resource_constraint_note, status,
			approval_id, resulting_revision_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		p.ID,
		p.GoalID,
		nullStrPtr(p.CycleID),
		dbEvolutionMode,
		p.ActorType,
		p.ActorID,
		p.Rationale,
		p.PreviousTargetMetricJSON,
		p.ProposedTargetMetricJSON,
		p.InvalidatedClaimIDsJSON,
		p.ResourceConstraintNote,
		string(p.Status),
		nullStrPtr(p.ApprovalID),
		nullStrPtr(p.ResultingRevisionID),
		p.CreatedAt.UTC().Format(time.RFC3339),
		p.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *LearningRepository) GetGoalEvolutionProposal(ctx context.Context, id string) (*learning.GoalEvolutionProposal, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, goal_id, cycle_id, evolution_mode, actor_type, actor_id,
		       rationale, previous_target_metric_json, proposed_target_metric_json,
		       invalidated_claim_ids_json, resource_constraint_note, status,
		       approval_id, resulting_revision_id, created_at, updated_at
		FROM goal_evolution_proposals WHERE id = ?
	`, id)
	var p learning.GoalEvolutionProposal
	var cycID, appID, revID sql.NullString
	var mode, st, cAt, uAt string
	if err := row.Scan(
		&p.ID, &p.GoalID, &cycID, &mode, &p.ActorType, &p.ActorID,
		&p.Rationale, &p.PreviousTargetMetricJSON, &p.ProposedTargetMetricJSON,
		&p.InvalidatedClaimIDsJSON, &p.ResourceConstraintNote, &st,
		&appID, &revID, &cAt, &uAt,
	); err != nil {
		return nil, err
	}
	p.CycleID = strPtrFromNull(cycID)
	p.EvolutionMode = learning.GoalEvolutionMode(mode)
	p.Status = learning.GoalEvolutionStatus(st)
	p.ApprovalID = strPtrFromNull(appID)
	p.ResultingRevisionID = strPtrFromNull(revID)
	p.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
	return &p, nil
}

func (r *LearningRepository) UpdateGoalEvolutionProposal(ctx context.Context, p *learning.GoalEvolutionProposal) error {
	p.UpdatedAt = clock.Now(ctx)
	_, err := r.db.RawDB().ExecContext(ctx, `
		UPDATE goal_evolution_proposals
		SET status = ?, approval_id = ?, resulting_revision_id = ?, updated_at = ?
		WHERE id = ?
	`,
		string(p.Status),
		nullStrPtr(p.ApprovalID),
		nullStrPtr(p.ResultingRevisionID),
		p.UpdatedAt.UTC().Format(time.RFC3339),
		p.ID,
	)
	return err
}

func (r *LearningRepository) ListGoalEvolutionProposals(ctx context.Context, goalID string) ([]*learning.GoalEvolutionProposal, error) {
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, goal_id, cycle_id, evolution_mode, actor_type, actor_id,
		       rationale, previous_target_metric_json, proposed_target_metric_json,
		       invalidated_claim_ids_json, resource_constraint_note, status,
		       approval_id, resulting_revision_id, created_at, updated_at
		FROM goal_evolution_proposals
		WHERE goal_id = ?
		ORDER BY created_at DESC, rowid DESC
	`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*learning.GoalEvolutionProposal
	for rows.Next() {
		var p learning.GoalEvolutionProposal
		var cycID, appID, revID sql.NullString
		var mode, st, cAt, uAt string
		if err := rows.Scan(
			&p.ID, &p.GoalID, &cycID, &mode, &p.ActorType, &p.ActorID,
			&p.Rationale, &p.PreviousTargetMetricJSON, &p.ProposedTargetMetricJSON,
			&p.InvalidatedClaimIDsJSON, &p.ResourceConstraintNote, &st,
			&appID, &revID, &cAt, &uAt,
		); err != nil {
			return nil, err
		}
		p.CycleID = strPtrFromNull(cycID)
		p.EvolutionMode = learning.GoalEvolutionMode(mode)
		p.Status = learning.GoalEvolutionStatus(st)
		p.ApprovalID = strPtrFromNull(appID)
		p.ResultingRevisionID = strPtrFromNull(revID)
		p.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
		list = append(list, &p)
	}
	return list, rows.Err()
}

func nullStrPtr(p *string) sql.NullString {
	if p == nil || *p == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

func strPtrFromNull(ns sql.NullString) *string {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	v := ns.String
	return &v
}
