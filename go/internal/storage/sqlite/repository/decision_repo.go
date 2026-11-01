package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/storage/sqlite"
)

type DecisionRepository struct {
	db *sqlite.DB
}

func NewDecisionRepository(db *sqlite.DB) *DecisionRepository {
	return &DecisionRepository{db: db}
}

// Create inserts a decision and its associated evidence links within a transaction.
func (r *DecisionRepository) Create(ctx context.Context, d *decision.Decision, evidenceLinks []*decision.DecisionEvidenceLink) error {
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		query := `
		INSERT INTO decisions (
			id, cycle_id, project_id, experiment_id, action_type, intent_summary, justification,
			considered_alternatives_json, context_snapshot_id, governance_result, action_task_id,
			action_execution_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		createdStr := d.CreatedAt.UTC().Format(time.RFC3339)

		_, err := tx.ExecContext(ctx, query,
			d.ID, d.CycleID, d.ProjectID, d.ExperimentID, d.ActionType, d.IntentSummary, d.Justification,
			d.ConsideredAlternativesJSON, d.ContextSnapshotID, d.GovernanceResult, d.ActionTaskID,
			d.ActionExecutionID, createdStr,
		)
		if err != nil {
			return err
		}

		linkQuery := `
		INSERT INTO decision_evidence_links (decision_id, evidence_id, relevance_note)
		VALUES (?, ?, ?)
		ON CONFLICT(decision_id, evidence_id) DO UPDATE SET relevance_note = excluded.relevance_note`

		for _, l := range evidenceLinks {
			_, err = tx.ExecContext(ctx, linkQuery, d.ID, l.EvidenceID, l.RelevanceNote)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// Get retrieves a decision by ID.
func (r *DecisionRepository) Get(ctx context.Context, id string) (*decision.Decision, error) {
	query := `
	SELECT id, cycle_id, project_id, experiment_id, action_type, intent_summary, justification,
	       considered_alternatives_json, context_snapshot_id, governance_result, action_task_id,
	       action_execution_id, created_at
	FROM decisions WHERE id = ?`

	d := &decision.Decision{}
	var cycleID, projID, expID, alts, snapID, taskID, execID sql.NullString
	var createdStr string

	err := r.db.RawDB().QueryRowContext(ctx, query, id).Scan(
		&d.ID, &cycleID, &projID, &expID, &d.ActionType, &d.IntentSummary, &d.Justification,
		&alts, &snapID, &d.GovernanceResult, &taskID, &execID, &createdStr,
	)
	if err != nil {
		return nil, err
	}

	if cycleID.Valid {
		d.CycleID = &cycleID.String
	}
	if projID.Valid {
		d.ProjectID = &projID.String
	}
	if expID.Valid {
		d.ExperimentID = &expID.String
	}
	if snapID.Valid {
		d.ContextSnapshotID = &snapID.String
	}
	if taskID.Valid {
		d.ActionTaskID = &taskID.String
	}
	if execID.Valid {
		d.ActionExecutionID = &execID.String
	}
	d.ConsideredAlternativesJSON = alts.String
	d.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)

	return d, nil
}

// ListByProject returns decisions for a project in reverse chronological order.
func (r *DecisionRepository) ListByProject(ctx context.Context, projectID string, limit int) ([]*decision.Decision, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
	SELECT id, cycle_id, project_id, experiment_id, action_type, intent_summary, justification,
	       considered_alternatives_json, context_snapshot_id, governance_result, action_task_id,
	       action_execution_id, created_at
	FROM decisions
	WHERE project_id = ?
	ORDER BY created_at DESC LIMIT ?`

	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []*decision.Decision
	for rows.Next() {
		d := &decision.Decision{}
		var cycleID, projID, expID, alts, snapID, taskID, execID sql.NullString
		var createdStr string

		err := rows.Scan(
			&d.ID, &cycleID, &projID, &expID, &d.ActionType, &d.IntentSummary, &d.Justification,
			&alts, &snapID, &d.GovernanceResult, &taskID, &execID, &createdStr,
		)
		if err != nil {
			return nil, err
		}
		if cycleID.Valid {
			d.CycleID = &cycleID.String
		}
		if projID.Valid {
			d.ProjectID = &projID.String
		}
		if expID.Valid {
			d.ExperimentID = &expID.String
		}
		if snapID.Valid {
			d.ContextSnapshotID = &snapID.String
		}
		if taskID.Valid {
			d.ActionTaskID = &taskID.String
		}
		if execID.Valid {
			d.ActionExecutionID = &execID.String
		}
		d.ConsideredAlternativesJSON = alts.String
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)

		decisions = append(decisions, d)
	}

	return decisions, rows.Err()
}

// GetEvidenceLinks returns all evidence associated with a decision.
func (r *DecisionRepository) GetEvidenceLinks(ctx context.Context, decisionID string) ([]*decision.DecisionEvidenceLink, error) {
	query := `SELECT decision_id, evidence_id, relevance_note FROM decision_evidence_links WHERE decision_id = ?`
	rows, err := r.db.RawDB().QueryContext(ctx, query, decisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*decision.DecisionEvidenceLink
	for rows.Next() {
		l := &decision.DecisionEvidenceLink{}
		var note sql.NullString
		if err := rows.Scan(&l.DecisionID, &l.EvidenceID, &note); err != nil {
			return nil, err
		}
		l.RelevanceNote = note.String
		links = append(links, l)
	}
	return links, rows.Err()
}
