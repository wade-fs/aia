package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/policy"
	approvalService "github.com/wade/aia/internal/service/approval"
	govService "github.com/wade/aia/internal/service/governance"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestPolicyEvaluator_Boundaries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	auditRepo := repository.NewAuditRepository(db)
	allowedPaths := []string{"/home/wade/src/github/aia/allowed"}
	evaluator := policy.NewEvaluatorWithOptions(auditRepo, allowedPaths, 1000.0)

	// Case 1: Whitelist path check - Disallowed path should be BLOCKED
	opBlocked := policy.OperationContext{
		ActorType:     "AGENT",
		ActorID:       "agent-coding",
		OperationType: governance.OperationFileWrite,
		TargetPath:    "/etc/shadow",
	}
	decBlocked := evaluator.EvaluateOperation(ctx, opBlocked)
	if decBlocked.Allowed {
		t.Errorf("expected /etc/shadow to be blocked, but was allowed")
	}
	if decBlocked.GovernanceAction != governance.GovernanceActionBlocked {
		t.Errorf("expected governance action BLOCKED, got %s", decBlocked.GovernanceAction)
	}

	// Case 2: Real financial transaction forbidden (AIA operates paper trading only)
	opFinance := policy.OperationContext{
		ActorType:     "AGENT",
		ActorID:       "agent-invest",
		OperationType: governance.OperationFinancialTransaction,
		Amount:        50000,
		IsRealMoney:   true,
	}
	decFinance := evaluator.EvaluateOperation(ctx, opFinance)
	if decFinance.Allowed {
		t.Errorf("expected real financial transaction to be denied, but was allowed")
	}

	// Case 3: Whitelisted path should be ALLOWED
	opAllowed := policy.OperationContext{
		ActorType:     "AGENT",
		ActorID:       "agent-coding",
		OperationType: governance.OperationFileWrite,
		TargetPath:    "/home/wade/src/github/aia/allowed/code.go",
	}
	decAllowed := evaluator.EvaluateOperation(ctx, opAllowed)
	if !decAllowed.Allowed {
		t.Errorf("expected whitelisted path to be allowed, but got: %s", decAllowed.Reason)
	}
}

func TestApprovalService_AgentSelfApprovalForbidden(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// Setup task
	_, _ = db.RawDB().ExecContext(ctx, `
		INSERT INTO projects (id, name, workspace, description, status, created_at, updated_at)
		VALUES ('proj-1', 'P1', '/tmp', 'D', 'ACTIVE', ?, ?)
	`, now, now)
	_, _ = db.RawDB().ExecContext(ctx, `
		INSERT INTO tasks (id, project_id, title, description, status, priority, risk, acceptance_json, created_at, updated_at)
		VALUES ('task-1', 'proj-1', 'T1', 'D', 'WAITING_APPROVAL', 5, 'HIGH', '{}', ?, ?)
	`, now, now)

	taskRepo := repository.NewTaskRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	approvalRepo := repository.NewApprovalRepository(db)
	taskSvc := taskService.NewService(taskRepo, auditRepo)
	apprSvc := approvalService.NewService(approvalRepo, taskSvc, auditRepo, nil)

	taskID := "task-1"
	reqInput := approvalService.RequestApprovalInput{
		RequesterEntity:     "AGENT:coding",
		OperationType:       governance.OperationTaskExecution,
		OperationDetailJSON: `{"task_id":"task-1"}`,
		RiskTier:            governance.RiskTierHigh,
		TaskID:              &taskID,
	}

	appr, err := apprSvc.RequestApproval(ctx, reqInput)
	if err != nil {
		t.Fatalf("failed to request approval: %v", err)
	}

	// Agent tries to self-approve - MUST BE REJECTED (ErrAgentSelfApprovalForbidden)
	err = apprSvc.Approve(ctx, appr.ID, "agent-coding", "Self approving", "AGENT")
	if err == nil {
		t.Fatalf("expected error when Agent tries to self-approve, but succeeded")
	}

	// Human approves - MUST SUCCEED
	err = apprSvc.Approve(ctx, appr.ID, "human_owner", "Approved by owner", "HUMAN")
	if err != nil {
		t.Fatalf("human approval failed: %v", err)
	}

	// Verify status updated to APPROVED
	updated, err := apprSvc.Get(ctx, appr.ID)
	if err != nil {
		t.Fatalf("failed to get approval: %v", err)
	}
	if updated.Status != governance.ApprovalApproved {
		t.Errorf("expected status APPROVED, got %s", updated.Status)
	}
}

func TestEmergencyStop_AgentCannotResume(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	govRepo := repository.NewGovernanceRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	govSvc := govService.NewService(govRepo, auditRepo, nil, nil)

	// Trigger emergency stop by anyone/system
	triggerActor := governance.Actor{Type: "AGENT", ID: "safety_monitor"}
	err := govSvc.TriggerEmergencyStop(ctx, triggerActor, "Anomaly detected")
	if err != nil {
		t.Fatalf("failed to trigger emergency stop: %v", err)
	}

	ctrl, err := govSvc.GetRuntimeControl(ctx)
	if err != nil {
		t.Fatalf("failed to get control: %v", err)
	}
	if !ctrl.IsEmergencyStopped() {
		t.Fatalf("expected dispatch_state to be EMERGENCY_STOP")
	}

	// Agent tries to resume - MUST BE FORBIDDEN
	agentActor := governance.Actor{Type: "AGENT", ID: "coding_agent"}
	err = govSvc.ResumeDispatch(ctx, agentActor, true)
	if err == nil {
		t.Fatalf("expected error when agent attempts to resume runtime, but got nil")
	}

	// Human resumes - MUST SUCCEED
	humanActor := governance.Actor{Type: "HUMAN", ID: "admin_user"}
	err = govSvc.ResumeDispatch(ctx, humanActor, true)
	if err != nil {
		t.Fatalf("failed to resume runtime with human actor: %v", err)
	}

	ctrl, err = govSvc.GetRuntimeControl(ctx)
	if err != nil {
		t.Fatalf("failed to get control: %v", err)
	}
	if ctrl.IsEmergencyStopped() {
		t.Fatalf("expected dispatch_state to be RUNNING after human resume")
	}
}
