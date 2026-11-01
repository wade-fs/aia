package unit

import (
	"context"
	"testing"
	"time"
)

func TestMigration009_GovernanceAndApprovals(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// 1. 先建立基礎 project 與 task
	_, err := db.RawDB().ExecContext(ctx, `
		INSERT INTO projects (id, name, workspace, description, status, created_at, updated_at)
		VALUES ('proj-gov-1', 'Governance Test', '/tmp/gov', 'desc', 'ACTIVE', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("failed to insert project: %v", err)
	}

	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO tasks (id, project_id, title, description, status, priority, risk, acceptance_json, created_at, updated_at)
		VALUES ('task-gov-1', 'proj-gov-1', 'Controlled Operation', 'High risk operation', 'WAITING_APPROVAL', 9, 'HIGH', '{}', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}

	// 2. 建立 Approval（關聯 task_id）
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO approvals (
			id, requester_entity, operation_type, operation_detail_json, status, risk_tier,
			timeout_policy, task_id, created_at
		) VALUES (
			'appr-001', 'AGENT:coding', 'FILE_MUTATE', '{"file":"/etc/passwd"}', 'PENDING', 'HIGH',
			'REJECT', 'task-gov-1', ?
		)
	`, now)
	if err != nil {
		t.Fatalf("failed to insert approval: %v", err)
	}

	// 3. 測試 CHECK 約束: task_id 與 decision_id 不可同時為空
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO approvals (
			id, requester_entity, operation_type, operation_detail_json, status, risk_tier,
			timeout_policy, created_at
		) VALUES (
			'appr-bad', 'AGENT:coding', 'FILE_MUTATE', '{}', 'PENDING', 'HIGH', 'REJECT', ?
		)
	`, now)
	if err == nil {
		t.Fatalf("expected error inserting approval without task_id or decision_id")
	}

	// 4. 測試 audit_logs 擴充欄位 (policy_id, decision_id, approval_id, risk_tier, governance_action)
	_, err = db.RawDB().ExecContext(ctx, `
		INSERT INTO audit_logs (
			id, sequence, actor_type, actor_id, action, resource_type, resource_id, outcome,
			details_json, policy_id, approval_id, risk_tier, governance_action, created_at
		) VALUES (
			'audit-gov-1', 1, 'HUMAN', 'owner', 'APPROVAL_GRANTED', 'APPROVAL', 'appr-001', 'APPROVED',
			'{"note":"approved"}', 'policy-01', 'appr-001', 'HIGH', 'REQUIRED_APPROVAL', ?
		)
	`, now)
	if err != nil {
		t.Fatalf("failed to insert audit log with governance columns: %v", err)
	}

	// 5. 測試 runtime_controls 表與防刪除 Trigger
	var dispatchState string
	err = db.RawDB().QueryRowContext(ctx, `SELECT dispatch_state FROM runtime_controls WHERE id = 'global'`).Scan(&dispatchState)
	if err != nil {
		t.Fatalf("failed to select runtime_controls: %v", err)
	}
	if dispatchState != "RUNNING" {
		t.Errorf("expected initial dispatch_state to be RUNNING, got %s", dispatchState)
	}

	// 嘗試刪除 runtime_controls global 紀錄應被 trigger 攔截
	_, err = db.RawDB().ExecContext(ctx, `DELETE FROM runtime_controls WHERE id = 'global'`)
	if err == nil {
		t.Fatalf("expected error deleting runtime_controls global row due to trigger")
	}
}
