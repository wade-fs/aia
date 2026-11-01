package unit

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/storage/sqlite"
)

func setupTestDB(t *testing.T) *sqlite.DB {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	return db
}

func TestMigration007_EventHistoryImmutability(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	now := time.Now().UTC().Format(time.RFC3339)

	// 1. 插入合法 Event
	insertSQL := `
	INSERT INTO event_history (
		event_id, event_type, event_version, aggregate_type, aggregate_id, aggregate_version,
		correlation_id, causation_id, actor_id, payload_json, payload_hash, previous_hash,
		occurred_at, recorded_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.RawDB().ExecContext(ctx, insertSQL,
		"evt-001", "GOAL_CREATED", 1, "GOAL", "goal-001", 1,
		"corr-001", "evt-001", "AGENT:test", `{"name":"test goal"}`, "hash1", "GENESIS",
		now, now,
	)
	if err != nil {
		t.Fatalf("failed to insert event_history: %v", err)
	}

	// 2. 驗證防竄改 Trigger: 禁止 UPDATE
	_, err = db.RawDB().ExecContext(ctx, "UPDATE event_history SET event_type = 'TAMPERED' WHERE event_id = 'evt-001'")
	if err == nil {
		t.Fatalf("expected error on UPDATE event_history, but got nil")
	}
	if !strings.Contains(err.Error(), "immutable and cannot be updated") {
		t.Errorf("expected immutability error message, got: %v", err)
	}

	// 3. 驗證防竄改 Trigger: 禁止 DELETE
	_, err = db.RawDB().ExecContext(ctx, "DELETE FROM event_history WHERE event_id = 'evt-001'")
	if err == nil {
		t.Fatalf("expected error on DELETE event_history, but got nil")
	}
	if !strings.Contains(err.Error(), "immutable and cannot be deleted") {
		t.Errorf("expected immutability error message, got: %v", err)
	}
}

func TestMigration007_SourcesMultiDimensionalAndClaims(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// 1. 測試 sources 五維度評估欄位
	insertSourceSQL := `
	INSERT INTO sources (
		id, source_type, uri, publisher,
		authority_score, directness_score, timeliness_score, independence_score, reproducibility_score,
		metadata_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.RawDB().ExecContext(ctx, insertSourceSQL,
		"src-001", "OFFICIAL", "https://mops.twse.com.tw", "TWSE",
		0.95, 0.90, 0.85, 0.99, 1.0,
		`{"market":"TWSE"}`, now,
	)
	if err != nil {
		t.Fatalf("failed to insert source with 5 dimensions: %v", err)
	}

	var auth, direct, timeScore, indep, repro float64
	err = db.RawDB().QueryRowContext(ctx, `
		SELECT authority_score, directness_score, timeliness_score, independence_score, reproducibility_score
		FROM sources WHERE id = 'src-001'`,
	).Scan(&auth, &direct, &timeScore, &indep, &repro)
	if err != nil {
		t.Fatalf("failed to query source scores: %v", err)
	}
	if auth != 0.95 || direct != 0.90 || timeScore != 0.85 || indep != 0.99 || repro != 1.0 {
		t.Errorf("source 5-dimensional scores mismatch: got (%v, %v, %v, %v, %v)", auth, direct, timeScore, indep, repro)
	}

	// 2. 測試 evidence 擷取方式與發布時間
	insertEvidenceSQL := `
	INSERT INTO evidence (
		id, source_id, content_snippet, content_hash, capture_method, data_version, published_at, captured_at, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.RawDB().ExecContext(ctx, insertEvidenceSQL,
		"evi-001", "src-001", "2330 10月營收月增5.2%", "hash-snippet", "API_JSON", "1.0", now, now, now,
	)
	if err != nil {
		t.Fatalf("failed to insert evidence: %v", err)
	}

	// 3. 測試 claims 與 claim_revisions
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO claims (id, statement, assertion_status, is_knowledge, current_version, created_at, updated_at)
		VALUES ('clm-001', 'TSMC Q3 Revenue increased', 'ACCEPTED', 1, 1, ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("failed to insert claim: %v", err)
	}

	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO claim_revisions (id, claim_id, revision_number, previous_statement, new_statement, reason, created_at)
		VALUES ('rev-001', 'clm-001', 1, 'TSMC Q3 Revenue increased', 'TSMC Q3 Revenue increased by 5.2%', 'Added precise figure', ?)`, now)
	if err != nil {
		t.Fatalf("failed to insert claim revision: %v", err)
	}

	// 4. 測試 memories 擴充欄位預設值
	var provType, provRefType, provStatus string
	var provRefID sql.NullString
	err = db.RawDB().QueryRowContext(ctx, `
		INSERT INTO memories (id, type, scope, title, content, source_type, confidence, status, version, created_at, updated_at)
		VALUES ('mem-001', 'FACT', 'GLOBAL', 'Test Memory', 'Content', 'AGENT', 0.8, 'ACTIVE', 1, ?, ?)
		RETURNING provenance_type, provenance_ref_type, provenance_ref_id, provenance_status`,
		now, now,
	).Scan(&provType, &provRefType, &provRefID, &provStatus)
	if err != nil {
		t.Fatalf("failed to insert and return memories: %v", err)
	}

	if provType != "INFERENCE" || provRefType != "NONE" || provStatus != "UNVERIFIED" || provRefID.Valid {
		t.Errorf("unexpected default memory provenance: type=%s, ref_type=%s, status=%s, ref_id=%v",
			provType, provRefType, provStatus, provRefID)
	}
}
