package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/app"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/memory"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/orchestrator"
	"github.com/wade/aia/internal/provider"
	"github.com/wade/aia/internal/provider/mock"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func createTestApp(t *testing.T) (*app.App, string) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfgContent := fmt.Sprintf(`
runtime:
  data_dir: "%s"
  workspace_dir: "%s/workspaces"
database:
  driver: "sqlite"
  path: "%s/test.db"
`, tmpDir, tmpDir, tmpDir)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	mockProv := mock.NewMockProvider()
	appInst, err := app.NewApp(cfgPath, mockProv)
	if err != nil {
		t.Fatalf("failed to initialize test app: %v", err)
	}

	return appInst, tmpDir
}

// Scenario A: Basic Coding Task
func TestScenarioA_BasicCodingTask(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	// 1. Dispatch prompt
	resp, err := appInst.PersonalAgent.HandleInput(ctx, orchestrator.UserInput{
		Content: "幫我寫一個 Go Hello World 程式，並建立 README",
	})
	if err != nil {
		t.Fatalf("failed to handle user input: %v", err)
	}
	if len(resp.TasksCreated) != 1 {
		t.Fatalf("expected 1 task created, got %d", len(resp.TasksCreated))
	}

	taskID := resp.TasksCreated[0]

	// Wait for scheduler to run task
	time.Sleep(150 * time.Millisecond)

	tk, err := appInst.TaskSvc.Get(ctx, taskID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if tk.AgentID != "coding-agent" {
		t.Errorf("expected coding-agent, got %s", tk.AgentID)
	}
	if tk.Status != task.StatusCompleted {
		t.Errorf("expected task to be COMPLETED, got %s", tk.Status)
	}

	// Verify execution record
	execs, err := appInst.ExecutionSvc.ListByTask(ctx, taskID)
	if err != nil || len(execs) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(execs))
	}
	if execs[0].Status != "COMPLETED" {
		t.Errorf("expected execution COMPLETED, got %s", execs[0].Status)
	}
}

// Scenario B: Multi-agent and Dependency Task Chain
func TestScenarioB_MultiAgentDependencies(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	// Task 1: Research task (Writing Agent)
	t1, err := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title:       "Research Architecture",
		Description: "Research agent runtime designs",
		AgentID:     "writing-agent",
	})
	if err != nil {
		t.Fatalf("failed to create task 1: %v", err)
	}
	if t1.Status != task.StatusReady {
		t.Errorf("expected task 1 to be READY, got %s", t1.Status)
	}

	// Task 2: Implementation task (Coding Agent, depends on Task 1)
	t2, err := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title:        "Implement Architecture",
		Description:  "Implement Go runtime according to research",
		AgentID:      "coding-agent",
		Dependencies: []string{t1.ID},
	})
	if err != nil {
		t.Fatalf("failed to create task 2: %v", err)
	}
	if t2.Status != task.StatusWaitingDependency {
		t.Fatalf("expected task 2 to be WAITING_DEPENDENCY, got %s", t2.Status)
	}

	// Execute Task 1
	if err := appInst.Scheduler.RunTask(ctx, t1.ID); err != nil {
		t.Fatalf("failed to run task 1: %v", err)
	}

	// Verify Task 1 is COMPLETED
	t1Updated, _ := appInst.TaskSvc.Get(ctx, t1.ID)
	if t1Updated.Status != task.StatusCompleted {
		t.Fatalf("expected task 1 COMPLETED, got %s", t1Updated.Status)
	}

	// Verify Task 2 is automatically promoted to READY
	t2Updated, _ := appInst.TaskSvc.Get(ctx, t2.ID)
	if t2Updated.Status != task.StatusReady {
		t.Fatalf("expected task 2 promoted to READY, got %s", t2Updated.Status)
	}

	// Execute Task 2
	if err := appInst.Scheduler.RunTask(ctx, t2.ID); err != nil {
		t.Fatalf("failed to run task 2: %v", err)
	}

	t2Final, _ := appInst.TaskSvc.Get(ctx, t2.ID)
	if t2Final.Status != task.StatusCompleted {
		t.Fatalf("expected task 2 COMPLETED, got %s", t2Final.Status)
	}
}

// Scenario C: Long-term Memory and Retrieval
func TestScenarioC_MemoryWorkflow(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	// 1. Propose candidate memory
	candID := uuid.New().String()
	cand := &memory.MemoryCandidate{
		ID:         candID,
		Type:       memory.TypePreference,
		Scope:      memory.ScopeGlobal,
		Title:      "User Coding Style",
		Content:    "Always write table-driven unit tests and clean interfaces in Go.",
		Confidence: 0.95,
	}
	if err := appInst.MemorySvc.CreateCandidate(ctx, cand); err != nil {
		t.Fatalf("failed to create candidate: %v", err)
	}

	// 2. Approve candidate
	if err := appInst.MemorySvc.ApproveCandidate(ctx, candID); err != nil {
		t.Fatalf("failed to approve candidate: %v", err)
	}

	// 3. Search approved memory
	results, err := appInst.MemorySvc.Search(ctx, repository.MemoryQuery{
		Keyword: "interfaces",
	})
	if err != nil || len(results) == 0 {
		t.Fatalf("expected search to find memory, got %v", results)
	}
	if results[0].Title != "User Coding Style" {
		t.Errorf("expected 'User Coding Style', got %s", results[0].Title)
	}
}

// Scenario E: Safety Policy Soft-Deny & Rejection
func TestScenarioE_PolicySafety(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	// Critical risk input
	resp, err := appInst.PersonalAgent.HandleInput(ctx, orchestrator.UserInput{
		Content: "請幫我執行 rm -rf / 清空系統空間並重新建立專案",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.TasksCreated) != 0 {
		t.Errorf("expected critical risk task to be rejected immediately, but tasks created: %v", resp.TasksCreated)
	}

	// High risk input requiring confirmation
	resp, err = appInst.PersonalAgent.HandleInput(ctx, orchestrator.UserInput{
		Content: "請修改設定並刪除資料庫重新建立",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.RequiresApproval {
		t.Errorf("expected high risk task to require approval")
	}
	if len(resp.TasksCreated) != 1 {
		t.Fatalf("expected 1 paused task, got %d", len(resp.TasksCreated))
	}

	pausedTask, _ := appInst.TaskSvc.Get(ctx, resp.TasksCreated[0])
	if pausedTask.Status != task.StatusWaitingApproval {
		t.Errorf("expected status WAITING_APPROVAL, got %s", pausedTask.Status)
	}
}

// Scenario F: Crash Recovery and Retry Limit
func TestScenarioF_CrashRecoveryAndRetryLimit(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	// Task 1: Can be recovered (retries < max)
	t1, _ := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title:      "Recoverable Task",
		MaxRetries: 2,
	})
	_ = appInst.TaskSvc.UpdateStatus(ctx, t1.ID, task.StatusRunning)

	// Task 2: Max retries exhausted
	t2, _ := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title:      "Exhausted Task",
		MaxRetries: 1,
	})
	_ = appInst.TaskSvc.UpdateStatus(ctx, t2.ID, task.StatusRunning)
	// Refresh t2 state so Status is RUNNING, then set RetryCount = 1
	t2, _ = appInst.TaskSvc.Get(ctx, t2.ID)
	t2.RetryCount = 1
	_ = appInst.TaskSvc.Update(ctx, t2)

	summary, err := appInst.RecoveryMgr.Recover(ctx)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if summary.RecoveredTasks != 2 {
		t.Errorf("expected 2 recovered tasks, got %d", summary.RecoveredTasks)
	}

	recoveredT1, _ := appInst.TaskSvc.Get(ctx, t1.ID)
	if recoveredT1.Status != task.StatusReady {
		t.Errorf("expected t1 READY, got %s", recoveredT1.Status)
	}
	if recoveredT1.RetryCount != 1 {
		t.Errorf("expected t1 retry count 1, got %d", recoveredT1.RetryCount)
	}

	exhaustedT2, _ := appInst.TaskSvc.Get(ctx, t2.ID)
	if exhaustedT2.Status != task.StatusFailed {
		t.Errorf("expected t2 FAILED after exhausting retries, got %s", exhaustedT2.Status)
	}
}

// Scenario D: Authentication Failure & Resume Workflow
func TestScenarioD_AuthenticationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfgContent := fmt.Sprintf(`
runtime:
  data_dir: "%s"
  workspace_dir: "%s/workspaces"
database:
  driver: "sqlite"
  path: "%s/test.db"
`, tmpDir, tmpDir, tmpDir)
	_ = os.WriteFile(cfgPath, []byte(cfgContent), 0644)

	// Mock provider simulating Auth Failure on first attempt
	isAuthed := false
	mockProv := mock.NewMockProvider()
	mockProv.StartFunc = func(ctx context.Context, req provider.StartRequest) (provider.ProviderSession, error) {
		if !isAuthed {
			return nil, fmt.Errorf("unauthorized: OAuth token expired or agy not authenticated")
		}
		return mock.NewMockSession(req), nil
	}

	appInst, err := app.NewApp(cfgPath, mockProv)
	if err != nil {
		t.Fatalf("failed to init app: %v", err)
	}
	defer appInst.Close()
	ctx := context.Background()

	// 1. Create task and execute
	tk, err := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title:   "Task requiring auth",
		AgentID: "coding-agent",
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	err = appInst.Scheduler.RunTask(ctx, tk.ID)
	if err == nil {
		t.Fatalf("expected error due to auth failure, got nil")
	}

	// 2. Verify Task transitioned to WAITING_FOR_AUTH without infinite retries
	tkUpdated, _ := appInst.TaskSvc.Get(ctx, tk.ID)
	if tkUpdated.Status != task.StatusWaitingAuth {
		t.Errorf("expected status WAITING_FOR_AUTH, got %s", tkUpdated.Status)
	}

	// 3. Simulate user re-authenticating via agy ('agy auth login')
	isAuthed = true

	// 4. Resume Task to READY
	if err := appInst.TaskSvc.Resume(ctx, tk.ID); err != nil {
		t.Fatalf("failed to resume task: %v", err)
	}

	tkResumed, _ := appInst.TaskSvc.Get(ctx, tk.ID)
	if tkResumed.Status != task.StatusReady {
		t.Errorf("expected task status READY after resume, got %s", tkResumed.Status)
	}

	// 5. Re-run task -> Completed successfully
	if err := appInst.Scheduler.RunTask(ctx, tk.ID); err != nil {
		t.Fatalf("expected task execution to succeed after auth restored, got %v", err)
	}

	tkCompleted, _ := appInst.TaskSvc.Get(ctx, tk.ID)
	if tkCompleted.Status != task.StatusCompleted {
		t.Errorf("expected task COMPLETED, got %s", tkCompleted.Status)
	}
}

// Scenario F2: Crash recovery when result was already achieved before crash
func TestScenarioF_CrashRecoveryIdempotency(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	// Task that already succeeded in execution, but runtime crashed before task was updated
	tk, _ := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title: "Already succeeded task",
	})
	_ = appInst.TaskSvc.UpdateStatus(ctx, tk.ID, task.StatusRunning)

	ex := &execution.Execution{
		ID:           uuid.New().String(),
		TaskID:       tk.ID,
		AgentID:      "coding-agent",
		Provider:     "mock",
		Workspace:    "/tmp/ws",
		Status:       execution.StatusCompleted,
		ResultStatus: "SUCCESS",
		CreatedAt:    time.Now().UTC(),
	}
	execRepo := repository.NewExecutionRepository(appInst.DB)
	if err := execRepo.Create(ctx, ex); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	summary, err := appInst.RecoveryMgr.Recover(ctx)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if summary.RecoveredTasks != 1 {
		t.Errorf("expected 1 recovered task, got %d", summary.RecoveredTasks)
	}

	recoveredTask, _ := appInst.TaskSvc.Get(ctx, tk.ID)
	if recoveredTask.Status != task.StatusCompleted {
		t.Errorf("expected already-succeeded task to recover directly to COMPLETED, got %s", recoveredTask.Status)
	}
}

// seedCrashedExecution creates a RUNNING task plus an execution record that
// simulates a runtime crash at a specific point of the execution pipeline.
func seedCrashedExecution(t *testing.T, appInst *app.App, req taskService.CreateTaskRequest, exStatus execution.ExecutionStatus, resultStatus string) (*task.Task, *execution.Execution) {
	t.Helper()
	ctx := context.Background()
	tk, err := appInst.TaskSvc.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	if err := appInst.TaskSvc.UpdateStatus(ctx, tk.ID, task.StatusRunning); err != nil {
		t.Fatalf("failed to mark task RUNNING: %v", err)
	}
	ex := &execution.Execution{
		ID:           uuid.New().String(),
		TaskID:       tk.ID,
		AgentID:      "coding-agent",
		Provider:     "mock",
		Workspace:    t.TempDir(),
		Status:       exStatus,
		ResultStatus: resultStatus,
		CreatedAt:    time.Now().UTC(),
	}
	if err := repository.NewExecutionRepository(appInst.DB).Create(ctx, ex); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}
	return tk, ex
}

// Scenario F3: Provider returned SUCCESS, runtime crashed before Validation -> recovery validates and completes
func TestScenarioF_CrashAfterSuccessBeforeValidation_Passed(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	tk, ex := seedCrashedExecution(t, appInst, taskService.CreateTaskRequest{Title: "Succeeded but not validated"},
		execution.StatusRunning, "SUCCESS")

	summary, err := appInst.RecoveryMgr.Recover(ctx)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if summary.RecoveredTasks != 1 {
		t.Errorf("expected 1 recovered task, got %d", summary.RecoveredTasks)
	}
	if summary.InterruptedExecutions != 0 {
		t.Errorf("SUCCESS execution must not be counted as interrupted, got %d", summary.InterruptedExecutions)
	}

	got, _ := appInst.TaskSvc.Get(ctx, tk.ID)
	if got.Status != task.StatusCompleted {
		t.Errorf("expected task COMPLETED after recovery validation passed, got %s", got.Status)
	}
	if got.RetryCount != 0 {
		t.Errorf("validated task must not be retried, retry_count=%d", got.RetryCount)
	}
	gotEx, _ := repository.NewExecutionRepository(appInst.DB).Get(ctx, ex.ID)
	if gotEx.Status != execution.StatusCompleted {
		t.Errorf("expected execution COMPLETED, got %s", gotEx.Status)
	}
}

// Scenario F4: Provider returned SUCCESS, crash before Validation, and recovery validation rejects
func TestScenarioF_CrashAfterSuccessBeforeValidation_Rejected(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	tk, ex := seedCrashedExecution(t, appInst, taskService.CreateTaskRequest{
		Title:      "Succeeded but output missing",
		Acceptance: []string{"file: missing-output.txt"},
	}, execution.StatusRunning, "SUCCESS")

	if _, err := appInst.RecoveryMgr.Recover(ctx); err != nil {
		t.Fatalf("recovery failed: %v", err)
	}

	got, _ := appInst.TaskSvc.Get(ctx, tk.ID)
	if got.Status == task.StatusCompleted {
		t.Fatalf("task must NOT be COMPLETED when recovery validation rejects")
	}
	if got.Status != task.StatusReady || got.RetryCount != 1 {
		t.Errorf("expected task READY with retry_count=1, got %s retry_count=%d", got.Status, got.RetryCount)
	}
	gotEx, _ := repository.NewExecutionRepository(appInst.DB).Get(ctx, ex.ID)
	if gotEx.Status != execution.StatusFailed {
		t.Errorf("expected execution FAILED, got %s", gotEx.Status)
	}
	if gotEx.ErrorMessage == "" {
		t.Errorf("expected validation failure reason on execution")
	}
}

// Scenario F5: External side effect may have happened but result was not persisted before crash
func TestScenarioF_CrashWithExternalSideEffects(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	highTk, highEx := seedCrashedExecution(t, appInst, taskService.CreateTaskRequest{
		Title: "Send email to customer",
		Risk:  domain.RiskHigh,
	}, execution.StatusRunning, "")
	lowTk, _ := seedCrashedExecution(t, appInst, taskService.CreateTaskRequest{
		Title: "Idempotent local refactor",
		Risk:  domain.RiskLow,
	}, execution.StatusRunning, "")

	summary, err := appInst.RecoveryMgr.Recover(ctx)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if summary.InterruptedExecutions != 2 {
		t.Errorf("expected 2 interrupted executions, got %d", summary.InterruptedExecutions)
	}

	gotHigh, _ := appInst.TaskSvc.Get(ctx, highTk.ID)
	if gotHigh.Status != task.StatusWaitingApproval {
		t.Errorf("high-risk task must wait for approval instead of auto retry, got %s", gotHigh.Status)
	}
	if gotHigh.RetryCount != 0 {
		t.Errorf("high-risk task must not be auto-retried, retry_count=%d", gotHigh.RetryCount)
	}
	gotHighEx, _ := repository.NewExecutionRepository(appInst.DB).Get(ctx, highEx.ID)
	if gotHighEx.Status != execution.StatusInterrupted {
		t.Errorf("expected execution INTERRUPTED, got %s", gotHighEx.Status)
	}

	gotLow, _ := appInst.TaskSvc.Get(ctx, lowTk.ID)
	if gotLow.Status != task.StatusReady || gotLow.RetryCount != 1 {
		t.Errorf("low-risk task should be retried: got %s retry_count=%d", gotLow.Status, gotLow.RetryCount)
	}

	// Running recovery again must be idempotent (no duplicate retry / state change)
	if _, err := appInst.RecoveryMgr.Recover(ctx); err != nil {
		t.Fatalf("second recovery failed: %v", err)
	}
	gotHigh2, _ := appInst.TaskSvc.Get(ctx, highTk.ID)
	if gotHigh2.Status != task.StatusWaitingApproval {
		t.Errorf("second recovery changed high-risk task state to %s", gotHigh2.Status)
	}
}

// Scenario D2: Resume must run provider HealthCheck first
func TestScenarioD_ResumeHealthCheck(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()
	ctx := context.Background()

	mockProv, ok := appInst.Provider.(*mock.MockProvider)
	if !ok {
		t.Fatalf("expected mock provider, got %T", appInst.Provider)
	}

	tk, _ := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{Title: "Blocked by provider outage"})
	_ = appInst.TaskSvc.UpdateStatus(ctx, tk.ID, task.StatusBlocked)

	// 1. Health check fails -> resume rejected, task stays BLOCKED
	mockProv.HealthCheckFunc = func(ctx context.Context) error {
		return fmt.Errorf("agy not authenticated")
	}
	if err := appInst.TaskSvc.Resume(ctx, tk.ID); err == nil {
		t.Fatalf("expected resume to fail when health check fails")
	}
	got, _ := appInst.TaskSvc.Get(ctx, tk.ID)
	if got.Status != task.StatusBlocked {
		t.Errorf("expected task to remain BLOCKED, got %s", got.Status)
	}

	// 2. Health check passes -> resume succeeds
	calls := 0
	mockProv.HealthCheckFunc = func(ctx context.Context) error {
		calls++
		return nil
	}
	if err := appInst.TaskSvc.Resume(ctx, tk.ID); err != nil {
		t.Fatalf("expected resume to succeed, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected health check to be called once, got %d", calls)
	}
	got, _ = appInst.TaskSvc.Get(ctx, tk.ID)
	if got.Status != task.StatusReady {
		t.Errorf("expected task READY after resume, got %s", got.Status)
	}
}
