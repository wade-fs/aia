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
	"github.com/wade/aia/internal/domain/memory"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/orchestrator"
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
