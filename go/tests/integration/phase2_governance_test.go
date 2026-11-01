package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/policy"
	approvalService "github.com/wade/aia/internal/service/approval"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/web/console"
)

func TestPhase2_Governance_EmergencyStopAndApprovalFlow(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	// 1. Verify Policy Evaluator boundary enforcement
	t.Run("PolicyEngine_BoundaryEnforcement", func(t *testing.T) {
		opBlocked := policy.OperationContext{
			ActorType:     "AGENT",
			ActorID:       "agent-coding",
			OperationType: governance.OperationFileWrite,
			TargetPath:    "/etc/shadow",
		}
		dec := app.PolicyEvaluator.EvaluateOperation(ctx, opBlocked)
		if dec.Allowed {
			t.Fatalf("expected /etc/shadow write to be blocked")
		}
		if dec.GovernanceAction != governance.GovernanceActionBlocked {
			t.Errorf("expected BLOCKED action, got %s", dec.GovernanceAction)
		}
	})

	// 2. Controlled Approval Workflow: Agent requests, Agent forbidden to self-approve, Human approves
	t.Run("Approval_AgentSelfApprovalForbidden_HumanApproves", func(t *testing.T) {
		// Create a task
		tsk, err := app.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
			Title:       "High Risk Task",
			Description: "Modifying protected code",
			AgentID:     "coding-agent",
			ProjectID:   "default",
		})
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		// Request approval
		tID := string(tsk.ID)
		appr, err := app.ApprovalSvc.RequestApproval(ctx, approvalService.RequestApprovalInput{
			RequesterEntity:     "AGENT:coding",
			OperationType:       governance.OperationTaskExecution,
			OperationDetailJSON: `{"task_id":"` + tID + `"}`,
			RiskTier:            governance.RiskTierHigh,
			TaskID:              &tID,
		})
		if err != nil {
			t.Fatalf("failed to request approval: %v", err)
		}

		// Verify task status was moved to WAITING_APPROVAL
		retrievedTask, err := app.TaskSvc.Get(ctx, tID)
		if err != nil {
			t.Fatalf("failed to get task: %v", err)
		}
		if retrievedTask.Status != task.StatusWaitingApproval {
			t.Errorf("expected task to be in WAITING_APPROVAL, got %s", retrievedTask.Status)
		}

		// Agent attempts self-approval -> MUST FAIL
		err = app.ApprovalSvc.Approve(ctx, appr.ID, "AGENT:coding", "self-approved", "AGENT")
		if err == nil {
			t.Fatalf("expected agent self-approval to be rejected, but succeeded")
		}

		// Human approves -> MUST SUCCEED
		err = app.ApprovalSvc.Approve(ctx, appr.ID, "human_owner", "Approved by owner", "HUMAN")
		if err != nil {
			t.Fatalf("human approval failed: %v", err)
		}

		// Verify task status returned to READY
		retrievedTask, err = app.TaskSvc.Get(ctx, tID)
		if err != nil {
			t.Fatalf("failed to get task: %v", err)
		}
		if retrievedTask.Status != task.StatusReady {
			t.Errorf("expected task to transition to READY upon approval, got %s", retrievedTask.Status)
		}
	})

	// 3. Emergency Stop Dispatch Freeze (<1s) and Cancellation
	t.Run("EmergencyStop_FreezesDispatchAndCancelsRunning", func(t *testing.T) {
		// Trigger emergency stop via WorkerMgr
		actor := governance.Actor{Type: "HUMAN", ID: "operator"}
		start := time.Now()
		err := app.WorkerMgr.TriggerEmergencyStop(ctx, actor, "Safety Drill: Freeze system")
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("failed to trigger emergency stop: %v", err)
		}

		if elapsed > 1*time.Second {
			t.Errorf("emergency stop took %v, required < 1s", elapsed)
		}

		if !app.WorkerMgr.IsEmergencyStopped() {
			t.Errorf("expected worker manager to be in emergency stopped state")
		}

		// Attempting to dispatch tasks or schedules during EMERGENCY_STOP must be rejected
		count, err := app.WorkerMgr.DispatchDue(ctx)
		if err == nil || count > 0 {
			t.Errorf("expected schedule dispatch to be rejected during emergency stop, got count=%d, err=%v", count, err)
		}

		_, err = app.WorkerMgr.ProcessReady(ctx)
		if err == nil {
			t.Errorf("expected task processing to be rejected during emergency stop")
		}

		// Verify Agent cannot resume runtime
		agentActor := governance.Actor{Type: "AGENT", ID: "coding_agent"}
		err = app.WorkerMgr.ResumeDispatch(ctx, agentActor, true)
		if err == nil {
			t.Fatalf("expected error when Agent attempts to resume runtime from emergency stop")
		}

		// Human operator can resume
		err = app.WorkerMgr.ResumeDispatch(ctx, actor, true)
		if err != nil {
			t.Fatalf("failed to resume runtime with human operator: %v", err)
		}

		if app.WorkerMgr.IsEmergencyStopped() {
			t.Errorf("expected worker manager to be RUNNING after resume")
		}
	})

	// 4. Console Web API Endpoints for Emergency Stop and Approvals
	t.Run("WebConsole_GovernanceEndpoints", func(t *testing.T) {
		mux := http.NewServeMux()
		console.RegisterConsoleRoutes(mux, &console.ConsoleServices{
			TaskSvc:       app.TaskSvc,
			AgentSvc:      app.AgentSvc,
			MemorySvc:     app.MemorySvc,
			AuditRepo:     app.AuditRepo,
			InvSvc:        app.InvSvc,
			DB:            app.DB,
			Provider:      app.Provider,
			Host:          nil,
			WorkerMgr:     app.WorkerMgr,
			ApprovalSvc:   app.ApprovalSvc,
			GovernanceSvc: app.GovernanceSvc,
		})

		// Test POST /api/v1/system/emergency-stop
		reqStop := httptest.NewRequest(http.MethodPost, "/api/v1/system/emergency-stop", strings.NewReader(`{"reason":"web emergency drill"}`))
		wStop := httptest.NewRecorder()
		mux.ServeHTTP(wStop, reqStop)
		if wStop.Code != http.StatusOK {
			t.Fatalf("expected 200 OK from emergency-stop endpoint, got %d: %s", wStop.Code, wStop.Body.String())
		}

		if !app.WorkerMgr.IsEmergencyStopped() {
			t.Errorf("expected emergency stop state after web trigger")
		}

		// Test POST /api/v1/system/resume
		reqResume := httptest.NewRequest(http.MethodPost, "/api/v1/system/resume", strings.NewReader(`{"force":true}`))
		wResume := httptest.NewRecorder()
		mux.ServeHTTP(wResume, reqResume)
		if wResume.Code != http.StatusOK {
			t.Fatalf("expected 200 OK from resume endpoint, got %d: %s", wResume.Code, wResume.Body.String())
		}

		if app.WorkerMgr.IsEmergencyStopped() {
			t.Errorf("expected RUNNING state after web resume")
		}
	})
}
