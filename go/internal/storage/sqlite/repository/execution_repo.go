package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/storage/sqlite"
)

type ExecutionRepository struct {
	db *sqlite.DB
}

func NewExecutionRepository(db *sqlite.DB) *ExecutionRepository {
	return &ExecutionRepository{db: db}
}

func (r *ExecutionRepository) Create(ctx context.Context, e *execution.Execution) error {
	query := `
		INSERT INTO executions (
			id, task_id, agent_id, provider, profile_id, profile_version,
			workspace, conversation_id, process_id, status, exit_code,
			result_status, error_code, error_message,
			input_tokens, output_tokens, total_tokens,
			started_at, ended_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}

	var startedAt, endedAt sql.NullString
	if e.StartedAt != nil {
		startedAt = sql.NullString{String: e.StartedAt.Format(time.RFC3339), Valid: true}
	}
	if e.EndedAt != nil {
		endedAt = sql.NullString{String: e.EndedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		e.ID, e.TaskID, e.AgentID, e.Provider, e.ProfileID, e.ProfileVersion,
		e.Workspace, e.ConversationID, e.ProcessID, string(e.Status), e.ExitCode,
		e.ResultStatus, e.ErrorCode, e.ErrorMessage,
		e.InputTokens, e.OutputTokens, e.TotalTokens,
		startedAt, endedAt, e.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}
	return nil
}

func (r *ExecutionRepository) Get(ctx context.Context, id domain.ID) (*execution.Execution, error) {
	query := `
		SELECT id, task_id, agent_id, provider, profile_id, profile_version,
		       workspace, conversation_id, process_id, status, exit_code,
		       result_status, error_code, error_message,
		       input_tokens, output_tokens, total_tokens,
		       started_at, ended_at, created_at
		FROM executions WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)

	var e execution.Execution
	var profileID, convID, startedAt, endedAt, resStatus, errCode, errMsg sql.NullString
	var status, createdAt string
	var exitCode sql.NullInt64

	err := row.Scan(
		&e.ID, &e.TaskID, &e.AgentID, &e.Provider, &profileID, &e.ProfileVersion,
		&e.Workspace, &convID, &e.ProcessID, &status, &exitCode,
		&resStatus, &errCode, &errMsg,
		&e.InputTokens, &e.OutputTokens, &e.TotalTokens,
		&startedAt, &endedAt, &createdAt,
	)
	if err == sql.ErrNoRows && len(id) >= 6 {
		prefixQuery := `
			SELECT id, task_id, agent_id, provider, profile_id, profile_version,
			       workspace, conversation_id, process_id, status, exit_code,
			       result_status, error_code, error_message,
			       input_tokens, output_tokens, total_tokens,
			       started_at, ended_at, created_at
			FROM executions WHERE id LIKE ? || '%' LIMIT 1
		`
		row = r.db.RawDB().QueryRowContext(ctx, prefixQuery, id)
		err = row.Scan(
			&e.ID, &e.TaskID, &e.AgentID, &e.Provider, &profileID, &e.ProfileVersion,
			&e.Workspace, &convID, &e.ProcessID, &status, &exitCode,
			&resStatus, &errCode, &errMsg,
			&e.InputTokens, &e.OutputTokens, &e.TotalTokens,
			&startedAt, &endedAt, &createdAt,
		)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("execution not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	e.Status = execution.ExecutionStatus(status)
	if profileID.Valid {
		e.ProfileID = profileID.String
	}
	if convID.Valid {
		e.ConversationID = convID.String
	}
	if exitCode.Valid {
		ec := int(exitCode.Int64)
		e.ExitCode = &ec
	}
	if resStatus.Valid {
		e.ResultStatus = resStatus.String
	}
	if errCode.Valid {
		e.ErrorCode = errCode.String
	}
	if errMsg.Valid {
		e.ErrorMessage = errMsg.String
	}
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if startedAt.Valid {
		st, _ := time.Parse(time.RFC3339, startedAt.String)
		e.StartedAt = &st
	}
	if endedAt.Valid {
		et, _ := time.Parse(time.RFC3339, endedAt.String)
		e.EndedAt = &et
	}

	return &e, nil
}

func (r *ExecutionRepository) Update(ctx context.Context, e *execution.Execution) error {
	query := `
		UPDATE executions SET
			conversation_id = ?, process_id = ?, status = ?, exit_code = ?,
			result_status = ?, error_code = ?, error_message = ?,
			input_tokens = ?, output_tokens = ?, total_tokens = ?,
			started_at = ?, ended_at = ?
		WHERE id = ?
	`
	var startedAt, endedAt sql.NullString
	if e.StartedAt != nil {
		startedAt = sql.NullString{String: e.StartedAt.Format(time.RFC3339), Valid: true}
	}
	if e.EndedAt != nil {
		endedAt = sql.NullString{String: e.EndedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		e.ConversationID, e.ProcessID, string(e.Status), e.ExitCode,
		e.ResultStatus, e.ErrorCode, e.ErrorMessage,
		e.InputTokens, e.OutputTokens, e.TotalTokens,
		startedAt, endedAt, e.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}
	return nil
}

func (r *ExecutionRepository) SumTokensSince(ctx context.Context, since time.Time) (int64, error) {
	var sum sql.NullInt64
	err := r.db.RawDB().QueryRowContext(ctx,
		`SELECT COALESCE(SUM(total_tokens), 0) FROM executions WHERE created_at >= ?`,
		since.UTC().Format(time.RFC3339),
	).Scan(&sum)
	if err != nil {
		return 0, err
	}
	if !sum.Valid {
		return 0, nil
	}
	return sum.Int64, nil
}

func (r *ExecutionRepository) ListByTask(ctx context.Context, taskID domain.ID) ([]execution.Execution, error) {
	query := `
		SELECT id, task_id, agent_id, provider, profile_id, profile_version,
		       workspace, conversation_id, process_id, status, exit_code,
		       result_status, error_code, error_message,
		       input_tokens, output_tokens, total_tokens,
		       started_at, ended_at, created_at
		FROM executions WHERE task_id = ? ORDER BY created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var list []execution.Execution
	for rows.Next() {
		var e execution.Execution
		var profileID, convID, startedAt, endedAt, resStatus, errCode, errMsg sql.NullString
		var status, createdAt string
		var exitCode sql.NullInt64

		if err := rows.Scan(
			&e.ID, &e.TaskID, &e.AgentID, &e.Provider, &profileID, &e.ProfileVersion,
			&e.Workspace, &convID, &e.ProcessID, &status, &exitCode,
			&resStatus, &errCode, &errMsg,
			&e.InputTokens, &e.OutputTokens, &e.TotalTokens,
			&startedAt, &endedAt, &createdAt,
		); err != nil {
			return nil, err
		}

		e.Status = execution.ExecutionStatus(status)
		if profileID.Valid {
			e.ProfileID = profileID.String
		}
		if convID.Valid {
			e.ConversationID = convID.String
		}
		if exitCode.Valid {
			ec := int(exitCode.Int64)
			e.ExitCode = &ec
		}
		if resStatus.Valid {
			e.ResultStatus = resStatus.String
		}
		if errCode.Valid {
			e.ErrorCode = errCode.String
		}
		if errMsg.Valid {
			e.ErrorMessage = errMsg.String
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if startedAt.Valid {
			st, _ := time.Parse(time.RFC3339, startedAt.String)
			e.StartedAt = &st
		}
		if endedAt.Valid {
			et, _ := time.Parse(time.RFC3339, endedAt.String)
			e.EndedAt = &et
		}
		list = append(list, e)
	}

	return list, nil
}

func (r *ExecutionRepository) AppendEvent(ctx context.Context, event *execution.ExecutionEvent) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		var maxSeq sql.NullInt64
		_ = tx.QueryRowContext(ctx, "SELECT MAX(sequence) FROM execution_events WHERE execution_id = ?", event.ExecutionID).Scan(&maxSeq)

		nextSeq := uint64(1)
		if maxSeq.Valid {
			nextSeq = uint64(maxSeq.Int64 + 1)
		}
		event.Sequence = nextSeq

		now := time.Now().UTC()
		if event.CreatedAt.IsZero() {
			event.CreatedAt = now
		}

		query := `
			INSERT INTO execution_events (id, execution_id, sequence, event_type, source, payload_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err := tx.ExecContext(
			ctx, query,
			event.ID, event.ExecutionID, event.Sequence, event.EventType, event.Source,
			string(event.Payload), event.CreatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to append execution event: %w", err)
		}
		return nil
	})
}

func (r *ExecutionRepository) ListEvents(ctx context.Context, executionID domain.ID, afterSeq uint64) ([]execution.ExecutionEvent, error) {
	if len(executionID) >= 6 && len(executionID) < 36 {
		var fullID string
		_ = r.db.RawDB().QueryRowContext(ctx, "SELECT id FROM executions WHERE id LIKE ? || '%' LIMIT 1", executionID).Scan(&fullID)
		if fullID != "" {
			executionID = fullID
		}
	}
	query := `
		SELECT id, execution_id, sequence, event_type, source, payload_json, created_at
		FROM execution_events
		WHERE execution_id = ? AND sequence > ?
		ORDER BY sequence ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, executionID, afterSeq)
	if err != nil {
		return nil, fmt.Errorf("failed to list execution events: %w", err)
	}
	defer rows.Close()

	var events []execution.ExecutionEvent
	for rows.Next() {
		var ev execution.ExecutionEvent
		var payload, createdAt string
		if err := rows.Scan(&ev.ID, &ev.ExecutionID, &ev.Sequence, &ev.EventType, &ev.Source, &payload, &createdAt); err != nil {
			return nil, err
		}
		ev.Payload = json.RawMessage(payload)
		ev.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		events = append(events, ev)
	}
	return events, nil
}
