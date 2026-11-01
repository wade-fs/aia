package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/storage/sqlite"
)

type EventHistoryRepository struct {
	db *sqlite.DB
}

func NewEventHistoryRepository(db *sqlite.DB) *EventHistoryRepository {
	return &EventHistoryRepository{db: db}
}

// Append inserts an immutable event into event_history, calculating its hash chain link.
func (r *EventHistoryRepository) Append(ctx context.Context, tx *sql.Tx, e *continuity.EventHistory) error {
	var execer interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	}

	if tx != nil {
		execer = tx
	} else {
		execer = r.db.RawDB()
	}

	// 1. Calculate payload hash
	e.PayloadHash = continuity.ComputePayloadHash(e.PayloadJSON)

	// 2. Query previous hash for the same aggregate (or GENESIS if first event)
	var prevHash string
	var maxVer sql.NullInt64
	err := execer.QueryRowContext(ctx, `
		SELECT payload_hash, aggregate_version 
		FROM event_history 
		WHERE aggregate_type = ? AND aggregate_id = ? 
		ORDER BY aggregate_version DESC LIMIT 1`,
		e.AggregateType, e.AggregateID,
	).Scan(&prevHash, &maxVer)

	if err == sql.ErrNoRows {
		e.PreviousHash = "GENESIS"
		e.AggregateVersion = 1
	} else if err != nil {
		return fmt.Errorf("failed to query previous event hash: %w", err)
	} else {
		e.PreviousHash = prevHash
		e.AggregateVersion = int(maxVer.Int64) + 1
	}

	if e.EventVersion == 0 {
		e.EventVersion = 1
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	if e.RecordedAt.IsZero() {
		e.RecordedAt = time.Now().UTC()
	}

	occurredStr := e.OccurredAt.UTC().Format(time.RFC3339Nano)
	recordedStr := e.RecordedAt.UTC().Format(time.RFC3339Nano)

	query := `
	INSERT INTO event_history (
		event_id, event_type, event_version, aggregate_type, aggregate_id, aggregate_version,
		correlation_id, causation_id, actor_id, payload_json, payload_hash, previous_hash,
		occurred_at, recorded_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = execer.ExecContext(ctx, query,
		e.EventID, e.EventType, e.EventVersion, e.AggregateType, e.AggregateID, e.AggregateVersion,
		e.CorrelationID, e.CausationID, e.ActorID, e.PayloadJSON, e.PayloadHash, e.PreviousHash,
		occurredStr, recordedStr,
	)
	if err != nil {
		return fmt.Errorf("failed to append event_history: %w", err)
	}

	return nil
}

// GetByEventID retrieves a single event by its UUID.
func (r *EventHistoryRepository) GetByEventID(ctx context.Context, eventID string) (*continuity.EventHistory, error) {
	query := `
	SELECT id, event_id, event_type, event_version, aggregate_type, aggregate_id, aggregate_version,
	       correlation_id, causation_id, actor_id, payload_json, payload_hash, previous_hash,
	       occurred_at, recorded_at
	FROM event_history WHERE event_id = ?`

	e := &continuity.EventHistory{}
	var occurredStr, recordedStr string

	err := r.db.RawDB().QueryRowContext(ctx, query, eventID).Scan(
		&e.ID, &e.EventID, &e.EventType, &e.EventVersion, &e.AggregateType, &e.AggregateID, &e.AggregateVersion,
		&e.CorrelationID, &e.CausationID, &e.ActorID, &e.PayloadJSON, &e.PayloadHash, &e.PreviousHash,
		&occurredStr, &recordedStr,
	)
	if err != nil {
		return nil, err
	}

	e.OccurredAt, _ = time.Parse(time.RFC3339Nano, occurredStr)
	e.RecordedAt, _ = time.Parse(time.RFC3339Nano, recordedStr)
	return e, nil
}

// ListByAggregate retrieves all events of an aggregate in chronological order.
func (r *EventHistoryRepository) ListByAggregate(ctx context.Context, aggregateType, aggregateID string) ([]*continuity.EventHistory, error) {
	query := `
	SELECT id, event_id, event_type, event_version, aggregate_type, aggregate_id, aggregate_version,
	       correlation_id, causation_id, actor_id, payload_json, payload_hash, previous_hash,
	       occurred_at, recorded_at
	FROM event_history 
	WHERE aggregate_type = ? AND aggregate_id = ?
	ORDER BY aggregate_version ASC`

	rows, err := r.db.RawDB().QueryContext(ctx, query, aggregateType, aggregateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*continuity.EventHistory
	for rows.Next() {
		e := &continuity.EventHistory{}
		var occurredStr, recordedStr string
		err := rows.Scan(
			&e.ID, &e.EventID, &e.EventType, &e.EventVersion, &e.AggregateType, &e.AggregateID, &e.AggregateVersion,
			&e.CorrelationID, &e.CausationID, &e.ActorID, &e.PayloadJSON, &e.PayloadHash, &e.PreviousHash,
			&occurredStr, &recordedStr,
		)
		if err != nil {
			return nil, err
		}
		e.OccurredAt, _ = time.Parse(time.RFC3339Nano, occurredStr)
		e.RecordedAt, _ = time.Parse(time.RFC3339Nano, recordedStr)
		events = append(events, e)
	}

	return events, rows.Err()
}

// VerifyAggregateChainIntegrity checks the SHA256 payload hashes and sequential previous_hash chain.
func (r *EventHistoryRepository) VerifyAggregateChainIntegrity(ctx context.Context, aggregateType, aggregateID string) (bool, error) {
	events, err := r.ListByAggregate(ctx, aggregateType, aggregateID)
	if err != nil {
		return false, err
	}
	if len(events) == 0 {
		return true, nil
	}

	expectedPrevHash := "GENESIS"
	for idx, e := range events {
		// 1. Verify sequence number
		if e.AggregateVersion != idx+1 {
			return false, fmt.Errorf("broken version sequence at event %s: expected %d got %d", e.EventID, idx+1, e.AggregateVersion)
		}

		// 2. Verify payload hash integrity
		actualPayloadHash := continuity.ComputePayloadHash(e.PayloadJSON)
		if actualPayloadHash != e.PayloadHash {
			return false, fmt.Errorf("payload hash mismatch at event %s", e.EventID)
		}

		// 3. Verify previous hash link
		if e.PreviousHash != expectedPrevHash {
			return false, fmt.Errorf("hash chain link broken at event %s: expected %s got %s", e.EventID, expectedPrevHash, e.PreviousHash)
		}

		expectedPrevHash = e.PayloadHash
	}

	return true, nil
}
