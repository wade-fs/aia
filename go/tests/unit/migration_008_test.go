package unit

import (
	"context"
	"testing"
	"time"
)

func TestMigration008_AutonomyCyclesAndDecisions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// 1. 建立 Autonomy Cycle
	insertCycleSQL := `
	INSERT INTO autonomy_cycles (
		id, cycle_number, trigger_type, trigger_ref_id, status, summary, started_at, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.RawDB().ExecContext(ctx, insertCycleSQL,
		"cycle-001", 1, "SCHEDULED", "sched-daily", "STARTED", "Daily market scan cycle", now, now,
	)
	if err != nil {
		t.Fatalf("failed to insert autonomy_cycle: %v", err)
	}

	// 2. 建立 Goal (包含 DRAFT 狀態)
	insertGoalSQL := `
	INSERT INTO goals (
		id, parent_goal_id, name, description, level, status, target_metric_json, priority, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.RawDB().ExecContext(ctx, insertGoalSQL,
		"goal-001", nil, "Long-term Investment Autonomy", "Achieve safe alpha", "STRATEGIC", "DRAFT", `{"target_return":0.15}`, 8, now, now,
	)
	if err != nil {
		t.Fatalf("failed to insert goal with DRAFT status: %v", err)
	}

	// 3. 建立 Goal Revision (測試 UNIQUE(goal_id, revision_number))
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO goal_revisions (id, goal_id, revision_number, previous_state_json, new_state_json, reason, created_at)
		VALUES ('rev-001', 'goal-001', 1, '{"status":"DRAFT"}', '{"status":"ACTIVE"}', 'Activated after review', ?)`, now)
	if err != nil {
		t.Fatalf("failed to insert goal revision: %v", err)
	}

	// 重複同一個 revision_number 應該報錯
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO goal_revisions (id, goal_id, revision_number, previous_state_json, new_state_json, reason, created_at)
		VALUES ('rev-002', 'goal-001', 1, '{"status":"ACTIVE"}', '{"status":"PAUSED"}', 'Duplicate rev', ?)`, now)
	if err == nil {
		t.Fatalf("expected UNIQUE constraint error on duplicate revision_number, got nil")
	}

	// 4. 建立 Project 與 M:N 關聯
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO projects (id, name, workspace, status, created_at, updated_at, primary_goal_id, pending_issues_json)
		VALUES ('proj-001', 'TWSE Pilot Project', '/tmp/twse', 'ACTIVE', ?, ?, 'goal-001', '["Issue A"]')`, now, now)
	if err != nil {
		t.Fatalf("failed to insert project with primary_goal_id: %v", err)
	}

	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO goal_project_links (goal_id, project_id, link_type, created_at)
		VALUES ('goal-001', 'proj-001', 'CONTRIBUTES_TO', ?)`, now)
	if err != nil {
		t.Fatalf("failed to insert goal_project_link: %v", err)
	}

	// 5. 建立 Context Snapshot (GLOBAL 與 PROJECT 層級)
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO context_snapshots (
			id, scope, project_id, snapshot_type, workspace_state_json, active_tasks_json, recent_decisions_json, pending_issues_json, created_at
		) VALUES (
			'snap-001', 'PROJECT', 'proj-001', 'PRE_RESTART', '{"git_head":"abc1234"}', '[]', '[]', '["Issue A"]', ?
		)`, now)
	if err != nil {
		t.Fatalf("failed to insert context_snapshot: %v", err)
	}

	// 6. 建立 Decision (支援 NO_ACTION 且關聯 Evidence)
	// 先在 007 的表裡建立 source 與 evidence
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO sources (id, source_type, uri, created_at) VALUES ('src-test', 'OFFICIAL', 'https://twse', ?)`, now)
	if err != nil {
		t.Fatalf("failed to insert source: %v", err)
	}
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO evidence (id, source_id, content_snippet, content_hash, capture_method, captured_at, created_at)
		VALUES ('evi-test', 'src-test', 'No market volatility', 'h123', 'API_JSON', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("failed to insert evidence: %v", err)
	}

	insertDecisionSQL := `
	INSERT INTO decisions (
		id, cycle_id, project_id, action_type, intent_summary, justification,
		context_snapshot_id, governance_result, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.RawDB().ExecContext(ctx, insertDecisionSQL,
		"dec-001", "cycle-001", "proj-001", "NO_ACTION", "Hold position", "Market volatility within normal range",
		"snap-001", "ALLOWED", now,
	)
	if err != nil {
		t.Fatalf("failed to insert decision with NO_ACTION: %v", err)
	}

	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO decision_evidence_links (decision_id, evidence_id, relevance_note)
		VALUES ('dec-001', 'evi-test', 'Supported hold decision')`)
	if err != nil {
		t.Fatalf("failed to link decision with evidence: %v", err)
	}

	// 7. 驗證 Decision 讀取
	var actionType, justification string
	err = db.RawDB().QueryRowContext(ctx, "SELECT action_type, justification FROM decisions WHERE id = 'dec-001'").Scan(&actionType, &justification)
	if err != nil {
		t.Fatalf("failed to query decision: %v", err)
	}
	if actionType != "NO_ACTION" {
		t.Errorf("expected action_type NO_ACTION, got: %s", actionType)
	}
}
