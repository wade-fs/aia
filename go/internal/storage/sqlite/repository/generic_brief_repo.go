package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domainBrief "github.com/wade/aia/internal/domain/brief"
	"github.com/wade/aia/internal/storage/sqlite"
)

// GenericBriefRepository persists Project Briefs and immutable Field History (AIA-21 §4, §12).
type GenericBriefRepository struct {
	db *sqlite.DB
}

// NewGenericBriefRepository creates a new generic brief repository.
func NewGenericBriefRepository(db *sqlite.DB) *GenericBriefRepository {
	return &GenericBriefRepository{db: db}
}

// GetByProjectID loads the active Project Brief by project ID.
func (r *GenericBriefRepository) GetByProjectID(ctx context.Context, projectID string) (*domainBrief.ProjectBrief, error) {
	query := `
		SELECT id, project_id, domain, brief_version, readiness_status, raw_intent, brief_yaml, created_at, updated_at
		FROM project_briefs
		WHERE project_id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, projectID)

	var id, projID, domain, readiness, rawIntent, briefYaml, createdAt, updatedAt string
	var version int

	if err := row.Scan(&id, &projID, &domain, &version, &readiness, &rawIntent, &briefYaml, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query project brief for %s: %w", projectID, err)
	}

	brief, err := domainBrief.FromYAML([]byte(briefYaml))
	if err != nil {
		return nil, fmt.Errorf("failed to parse brief yaml for %s: %w", projectID, err)
	}

	brief.ID = id
	brief.ProjectID = projID
	brief.Domain = domain
	brief.BriefVersion = version
	brief.Readiness = domainBrief.ReadinessStatus(readiness)
	brief.RawIntent = rawIntent
	brief.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	brief.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	// Populate full field history from database
	history, err := r.loadAllHistory(ctx, projectID)
	if err == nil && len(history) > 0 {
		brief.History = history
	}

	return brief, nil
}

// Save persists the project brief and any new field history entries atomically.
func (r *GenericBriefRepository) Save(ctx context.Context, b *domainBrief.ProjectBrief) error {
	yamlStr, err := b.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to serialize brief: %w", err)
	}

	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now

	tx, err := r.db.RawDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO project_briefs (
			id, project_id, domain, brief_version, readiness_status, raw_intent, brief_yaml, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE SET
			domain = excluded.domain,
			brief_version = excluded.brief_version,
			readiness_status = excluded.readiness_status,
			raw_intent = excluded.raw_intent,
			brief_yaml = excluded.brief_yaml,
			updated_at = excluded.updated_at
	`
	_, err = tx.ExecContext(
		ctx, query,
		b.ID, b.ProjectID, b.Domain, b.BriefVersion, string(b.Readiness), b.RawIntent, yamlStr,
		b.CreatedAt.Format(time.RFC3339), b.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert project_brief: %w", err)
	}

	// Persist field history entries
	histQuery := `
		INSERT OR IGNORE INTO brief_field_history (
			id, brief_id, project_id, field_key, version,
			old_value_json, new_value_json, old_status, new_status,
			source, rationale, related_task_id, related_clarification_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	for key, entries := range b.History {
		for _, entry := range entries {
			oldJSON, _ := json.Marshal(entry.OldValue)
			newJSON, _ := json.Marshal(entry.NewValue)
			entryID := fmt.Sprintf("hist-%s-%s-v%d", b.ProjectID, key, entry.Version)

			_, err = tx.ExecContext(
				ctx, histQuery,
				entryID, b.ID, b.ProjectID, key, entry.Version,
				string(oldJSON), string(newJSON), string(entry.OldStatus), string(entry.NewStatus),
				entry.Source, entry.Reason, entry.RelatedTaskID, entry.RelatedClarificationID,
				entry.Timestamp.Format(time.RFC3339),
			)
			if err != nil {
				return fmt.Errorf("failed to insert brief field history: %w", err)
			}
		}
	}

	return tx.Commit()
}

// ErrBriefVersionConflict indicates an optimistic concurrency violation (AIA-21 §12.3).
var ErrBriefVersionConflict = errors.New("brief version conflict")

// SaveWithVersionCheck persists the project brief only if current database version matches expectedVersion.
func (r *GenericBriefRepository) SaveWithVersionCheck(ctx context.Context, b *domainBrief.ProjectBrief, expectedVersion int) error {
	if expectedVersion <= 0 {
		return r.Save(ctx, b)
	}

	tx, err := r.db.RawDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	// Check current version in database
	var curVersion int
	row := tx.QueryRowContext(ctx, `SELECT brief_version FROM project_briefs WHERE project_id = ?`, b.ProjectID)
	if err := row.Scan(&curVersion); err == nil {
		if curVersion != expectedVersion {
			return fmt.Errorf("%w: expected v%d, but current database version is v%d (AIA-21 §12.3)", ErrBriefVersionConflict, expectedVersion, curVersion)
		}
	}

	yamlStr, err := b.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to serialize brief: %w", err)
	}

	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now

	query := `
		INSERT INTO project_briefs (
			id, project_id, domain, brief_version, readiness_status, raw_intent, brief_yaml, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE SET
			domain = excluded.domain,
			brief_version = excluded.brief_version,
			readiness_status = excluded.readiness_status,
			raw_intent = excluded.raw_intent,
			brief_yaml = excluded.brief_yaml,
			updated_at = excluded.updated_at
	`
	_, err = tx.ExecContext(
		ctx, query,
		b.ID, b.ProjectID, b.Domain, b.BriefVersion, string(b.Readiness), b.RawIntent, yamlStr,
		b.CreatedAt.Format(time.RFC3339), b.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert project_brief: %w", err)
	}

	// Persist field history entries
	histQuery := `
		INSERT OR IGNORE INTO brief_field_history (
			id, brief_id, project_id, field_key, version,
			old_value_json, new_value_json, old_status, new_status,
			source, rationale, related_task_id, related_clarification_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	for key, entries := range b.History {
		for _, entry := range entries {
			oldJSON, _ := json.Marshal(entry.OldValue)
			newJSON, _ := json.Marshal(entry.NewValue)
			entryID := fmt.Sprintf("hist-%s-%s-v%d", b.ProjectID, key, entry.Version)

			_, err = tx.ExecContext(
				ctx, histQuery,
				entryID, b.ID, b.ProjectID, key, entry.Version,
				string(oldJSON), string(newJSON), string(entry.OldStatus), string(entry.NewStatus),
				entry.Source, entry.Reason, entry.RelatedTaskID, entry.RelatedClarificationID,
				entry.Timestamp.Format(time.RFC3339),
			)
			if err != nil {
				return fmt.Errorf("failed to insert brief field history: %w", err)
			}
		}
	}

	return tx.Commit()
}

// ListFieldHistory retrieves chronological revisions for a specific field.
func (r *GenericBriefRepository) ListFieldHistory(ctx context.Context, projectID string, fieldKey string) ([]domainBrief.FieldHistoryEntry, error) {
	query := `
		SELECT version, old_value_json, new_value_json, old_status, new_status,
		       source, rationale, related_task_id, related_clarification_id, created_at
		FROM brief_field_history
		WHERE project_id = ? AND field_key = ?
		ORDER BY version ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID, fieldKey)
	if err != nil {
		return nil, fmt.Errorf("failed to list history for %s.%s: %w", projectID, fieldKey, err)
	}
	defer rows.Close()

	var result []domainBrief.FieldHistoryEntry
	for rows.Next() {
		var e domainBrief.FieldHistoryEntry
		var oldJSON, newJSON, oldStatus, newStatus, createdAt string
		var taskID, clarifyID sql.NullString

		if err := rows.Scan(
			&e.Version, &oldJSON, &newJSON, &oldStatus, &newStatus,
			&e.Source, &e.Reason, &taskID, &clarifyID, &createdAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal([]byte(oldJSON), &e.OldValue)
		_ = json.Unmarshal([]byte(newJSON), &e.NewValue)
		e.OldStatus = domainBrief.FieldStatus(oldStatus)
		e.NewStatus = domainBrief.FieldStatus(newStatus)
		if taskID.Valid {
			e.RelatedTaskID = taskID.String
		}
		if clarifyID.Valid {
			e.RelatedClarificationID = clarifyID.String
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, createdAt)
		result = append(result, e)
	}
	return result, nil
}

func (r *GenericBriefRepository) loadAllHistory(ctx context.Context, projectID string) (map[string][]domainBrief.FieldHistoryEntry, error) {
	query := `
		SELECT field_key, version, old_value_json, new_value_json, old_status, new_status,
		       source, rationale, related_task_id, related_clarification_id, created_at
		FROM brief_field_history
		WHERE project_id = ?
		ORDER BY field_key ASC, version ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string][]domainBrief.FieldHistoryEntry)
	for rows.Next() {
		var fieldKey string
		var e domainBrief.FieldHistoryEntry
		var oldJSON, newJSON, oldStatus, newStatus, createdAt string
		var taskID, clarifyID sql.NullString

		if err := rows.Scan(
			&fieldKey, &e.Version, &oldJSON, &newJSON, &oldStatus, &newStatus,
			&e.Source, &e.Reason, &taskID, &clarifyID, &createdAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal([]byte(oldJSON), &e.OldValue)
		_ = json.Unmarshal([]byte(newJSON), &e.NewValue)
		e.OldStatus = domainBrief.FieldStatus(oldStatus)
		e.NewStatus = domainBrief.FieldStatus(newStatus)
		if taskID.Valid {
			e.RelatedTaskID = taskID.String
		}
		if clarifyID.Valid {
			e.RelatedClarificationID = clarifyID.String
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, createdAt)
		res[fieldKey] = append(res[fieldKey], e)
	}
	return res, nil
}
