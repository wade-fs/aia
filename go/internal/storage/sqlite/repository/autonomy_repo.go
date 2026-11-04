package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/storage/sqlite"
)

type AutonomyRepository struct {
	db *sqlite.DB
}

func NewAutonomyRepository(db *sqlite.DB) *AutonomyRepository {
	return &AutonomyRepository{db: db}
}

// CreateInitiative persists a candidate initiative discovered during Step 4.
func (r *AutonomyRepository) CreateInitiative(ctx context.Context, init *domainAutonomy.CandidateInitiative) error {
	now := time.Now().UTC()
	if init.CreatedAt.IsZero() {
		init.CreatedAt = now
	}
	if init.UpdatedAt.IsZero() {
		init.UpdatedAt = now
	}
	if init.DiscoveryType == "" {
		init.DiscoveryType = domainAutonomy.DiscoveryOpportunity
	}
	if init.ConfidenceLevel == "" {
		init.ConfidenceLevel = domainAutonomy.ConfidenceProbable
	}
	if init.TargetAgentID == "" {
		init.TargetAgentID = "coding-agent"
	}
	if init.Status == "" {
		init.Status = domainAutonomy.InitiativeCandidate
	}
	if init.PriorityScore <= 0 {
		init.PriorityScore = 5
	}
	if init.PriorityScore > 10 {
		init.PriorityScore = 10
	}
	if init.EstimatedCost <= 0 {
		init.EstimatedCost = 1.0
	}
	if init.EstimatedRisk == "" {
		init.EstimatedRisk = governance.RiskTierLow
	}
	if init.MetadataJSON == "" {
		init.MetadataJSON = "{}"
	}

	discTypeStr := string(init.DiscoveryType)
	if discTypeStr == "PROBLEM" {
		discTypeStr = "ISSUE"
	}

	query := `
		INSERT INTO candidate_initiatives (
			id, cycle_id, project_id, goal_id, discovery_type, confidence_level, target_agent_id,
			trigger_source, detected_opportunity, rationale, expected_outcome,
			priority_score, estimated_cost, estimated_risk, status, discard_reason,
			spawned_task_id, metadata_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.RawDB().ExecContext(ctx, query,
		init.ID, init.CycleID, init.ProjectID, init.GoalID,
		discTypeStr, string(init.ConfidenceLevel), init.TargetAgentID,
		init.TriggerSource, init.DetectedOpportunity, init.Rationale, init.ExpectedOutcome,
		init.PriorityScore, init.EstimatedCost, string(init.EstimatedRisk),
		string(init.Status), init.DiscardReason, init.SpawnedTaskID, init.MetadataJSON,
		init.CreatedAt.UTC().Format(time.RFC3339), init.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert candidate_initiative: %w", err)
	}
	return nil
}

// UpdateInitiativeStatus updates the status and optional discard/defer reason of an initiative.
func (r *AutonomyRepository) UpdateInitiativeStatus(ctx context.Context, id string, status domainAutonomy.InitiativeStatus, discardReason *string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := `
		UPDATE candidate_initiatives
		SET status = ?, discard_reason = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.RawDB().ExecContext(ctx, query, string(status), discardReason, now, id)
	return err
}

// MarkInitiativeAdopted marks an initiative ADOPTED and links its spawned Task ID.
func (r *AutonomyRepository) MarkInitiativeAdopted(ctx context.Context, id string, spawnedTaskID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var taskRef *string
	if spawnedTaskID != "" {
		taskRef = &spawnedTaskID
	}
	query := `
		UPDATE candidate_initiatives
		SET status = ?, spawned_task_id = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.RawDB().ExecContext(ctx, query, string(domainAutonomy.InitiativeAdopted), taskRef, now, id)
	return err
}

func scanInitiativeRows(rows *sql.Rows) ([]*domainAutonomy.CandidateInitiative, error) {
	var list []*domainAutonomy.CandidateInitiative
	for rows.Next() {
		var item domainAutonomy.CandidateInitiative
		var projID, goalID, discard, spawnedTask sql.NullString
		var discType, confLvl, estRisk, statusStr, createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&item.ID, &item.CycleID, &projID, &goalID,
			&discType, &confLvl, &item.TargetAgentID,
			&item.TriggerSource, &item.DetectedOpportunity, &item.Rationale, &item.ExpectedOutcome,
			&item.PriorityScore, &item.EstimatedCost, &estRisk,
			&statusStr, &discard, &spawnedTask, &item.MetadataJSON,
			&createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, err
		}

		if projID.Valid {
			item.ProjectID = &projID.String
		}
		if goalID.Valid {
			item.GoalID = &goalID.String
		}
		if discard.Valid {
			item.DiscardReason = &discard.String
		}
		if spawnedTask.Valid {
			item.SpawnedTaskID = &spawnedTask.String
		}
		item.DiscoveryType = domainAutonomy.DiscoveryType(discType)
		item.ConfidenceLevel = domainAutonomy.DiscoveryConfidence(confLvl)
		item.EstimatedRisk = governance.RiskTier(estRisk)
		item.Status = domainAutonomy.InitiativeStatus(statusStr)
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		list = append(list, &item)
	}
	return list, rows.Err()
}

// ListInitiativesByCycle returns all candidate initiatives for a cycle ordered by priority.
func (r *AutonomyRepository) ListInitiativesByCycle(ctx context.Context, cycleID string) ([]*domainAutonomy.CandidateInitiative, error) {
	query := `
		SELECT id, cycle_id, project_id, goal_id, discovery_type, confidence_level, target_agent_id,
		       trigger_source, detected_opportunity, rationale, expected_outcome,
		       priority_score, estimated_cost, estimated_risk, status, discard_reason,
		       spawned_task_id, metadata_json, created_at, updated_at
		FROM candidate_initiatives
		WHERE cycle_id = ?
		ORDER BY priority_score DESC, created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, cycleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query candidate_initiatives: %w", err)
	}
	defer rows.Close()
	return scanInitiativeRows(rows)
}

// ListRecentInitiativesByProject returns recent initiatives for deduplication & continuity checks.
func (r *AutonomyRepository) ListRecentInitiativesByProject(ctx context.Context, projectID string, limit int) ([]*domainAutonomy.CandidateInitiative, error) {
	if limit <= 0 {
		limit = 30
	}
	query := `
		SELECT id, cycle_id, project_id, goal_id, discovery_type, confidence_level, target_agent_id,
		       trigger_source, detected_opportunity, rationale, expected_outcome,
		       priority_score, estimated_cost, estimated_risk, status, discard_reason,
		       spawned_task_id, metadata_json, created_at, updated_at
		FROM candidate_initiatives
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent candidate_initiatives: %w", err)
	}
	defer rows.Close()
	return scanInitiativeRows(rows)
}

// CreateDeliberation persists a structured deliberation record.
func (r *AutonomyRepository) CreateDeliberation(ctx context.Context, delib *domainAutonomy.DeliberationRecord) error {
	now := time.Now().UTC()
	if delib.CreatedAt.IsZero() {
		delib.CreatedAt = now
	}

	query := `
		INSERT INTO deliberation_records (
			id, cycle_id, decision_id, candidate_initiative_id, options_evaluated_json,
			cost_benefit_analysis, risk_assessment, recommended_action, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.RawDB().ExecContext(ctx, query,
		delib.ID, delib.CycleID, delib.DecisionID, delib.CandidateInitiativeID,
		delib.OptionsEvaluatedJSON, delib.CostBenefitAnalysis, delib.RiskAssessment,
		string(delib.RecommendedAction), delib.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert deliberation_record: %w", err)
	}
	return nil
}

// LinkDeliberationDecision attaches the finalized Decision ID to a persisted DeliberationRecord.
func (r *AutonomyRepository) LinkDeliberationDecision(ctx context.Context, deliberationID string, decisionID string) error {
	_, err := r.db.RawDB().ExecContext(
		ctx,
		`UPDATE deliberation_records SET decision_id = ? WHERE id = ?`,
		decisionID, deliberationID,
	)
	return err
}

func scanDeliberationRow(row *sql.Row) (*domainAutonomy.DeliberationRecord, error) {
	var rec domainAutonomy.DeliberationRecord
	var decID, initID sql.NullString
	var actionStr, createdAtStr string

	if err := row.Scan(
		&rec.ID, &rec.CycleID, &decID, &initID, &rec.OptionsEvaluatedJSON,
		&rec.CostBenefitAnalysis, &rec.RiskAssessment, &actionStr, &createdAtStr,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if decID.Valid {
		rec.DecisionID = &decID.String
	}
	if initID.Valid {
		rec.CandidateInitiativeID = &initID.String
	}
	rec.RecommendedAction = decision.ActionType(actionStr)
	rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	return &rec, nil
}

// GetDeliberationByDecision fetches the deliberation record associated with a decision.
func (r *AutonomyRepository) GetDeliberationByDecision(ctx context.Context, decisionID string) (*domainAutonomy.DeliberationRecord, error) {
	query := `
		SELECT id, cycle_id, decision_id, candidate_initiative_id, options_evaluated_json,
		       cost_benefit_analysis, risk_assessment, recommended_action, created_at
		FROM deliberation_records
		WHERE decision_id = ?
		LIMIT 1
	`
	return scanDeliberationRow(r.db.RawDB().QueryRowContext(ctx, query, decisionID))
}

// GetDeliberationByCycle fetches the deliberation record for an autonomy cycle.
func (r *AutonomyRepository) GetDeliberationByCycle(ctx context.Context, cycleID string) (*domainAutonomy.DeliberationRecord, error) {
	query := `
		SELECT id, cycle_id, decision_id, candidate_initiative_id, options_evaluated_json,
		       cost_benefit_analysis, risk_assessment, recommended_action, created_at
		FROM deliberation_records
		WHERE cycle_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	return scanDeliberationRow(r.db.RawDB().QueryRowContext(ctx, query, cycleID))
}

// LinkGoalTask records a Goal-to-Task decomposition link (Plan Phase 3 §4.3).
func (r *AutonomyRepository) LinkGoalTask(ctx context.Context, link *domainAutonomy.GoalTaskLink) error {
	now := time.Now().UTC()
	if link.CreatedAt.IsZero() {
		link.CreatedAt = now
	}
	query := `
		INSERT INTO goal_task_links (goal_id, task_id, cycle_id, decomposition_rationale, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(goal_id, task_id) DO UPDATE SET
			cycle_id = excluded.cycle_id,
			decomposition_rationale = excluded.decomposition_rationale
	`
	_, err := r.db.RawDB().ExecContext(ctx, query,
		link.GoalID, link.TaskID, link.CycleID, link.DecompositionRationale,
		link.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to link goal to task: %w", err)
	}
	return nil
}

// ListGoalTaskLinks returns all decomposed task links for a specific Goal.
func (r *AutonomyRepository) ListGoalTaskLinks(ctx context.Context, goalID string) ([]*domainAutonomy.GoalTaskLink, error) {
	query := `
		SELECT goal_id, task_id, cycle_id, decomposition_rationale, created_at
		FROM goal_task_links
		WHERE goal_id = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*domainAutonomy.GoalTaskLink
	for rows.Next() {
		var l domainAutonomy.GoalTaskLink
		var cycleID sql.NullString
		var createdStr string
		if err := rows.Scan(&l.GoalID, &l.TaskID, &cycleID, &l.DecompositionRationale, &createdStr); err != nil {
			return nil, err
		}
		if cycleID.Valid {
			l.CycleID = &cycleID.String
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		links = append(links, &l)
	}
	return links, rows.Err()
}
