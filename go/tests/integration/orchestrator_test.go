package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wade/aia/internal/app"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/orchestrator"
	"github.com/wade/aia/internal/provider/mock"
	taskService "github.com/wade/aia/internal/service/task"
)

func TestPersonalAgentOrchestration(t *testing.T) {
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
		t.Fatalf("failed to create test app: %v", err)
	}
	defer appInst.Close()

	ctx := context.Background()

	// 1. Test conversational input
	resp, err := appInst.PersonalAgent.HandleInput(ctx, orchestrator.UserInput{
		Content: "你好，你是誰？",
	})
	if err != nil {
		t.Fatalf("unexpected conversational error: %v", err)
	}
	if len(resp.TasksCreated) != 0 {
		t.Errorf("expected 0 tasks created for conversation, got %d", len(resp.TasksCreated))
	}

	// 2. Test actionable input (triggers task planning)
	resp, err = appInst.PersonalAgent.HandleInput(ctx, orchestrator.UserInput{
		Content: "請實作一個計算兩個整數相加的函式 Add 並撰寫單元測試",
	})
	if err != nil {
		t.Fatalf("unexpected task planning error: %v", err)
	}
	if len(resp.TasksCreated) != 1 {
		t.Fatalf("expected 1 task created, got %d", len(resp.TasksCreated))
	}

	taskID := resp.TasksCreated[0]

	// Allow goroutine time to execute
	time.Sleep(100 * time.Millisecond)

	createdTask, err := appInst.TaskSvc.Get(ctx, taskID)
	if err != nil {
		t.Fatalf("failed to get created task: %v", err)
	}
	if createdTask.AgentID != "coding-agent" {
		t.Errorf("expected coding-agent assigned, got %s", createdTask.AgentID)
	}

	// 3. Test Recovery on simulated crash
	// Manually insert a RUNNING task
	dummyTask, err := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title: "Crashed Task",
	})
	if err != nil {
		t.Fatalf("failed to create dummy task: %v", err)
	}
	_ = appInst.TaskSvc.UpdateStatus(ctx, dummyTask.ID, task.StatusRunning)

	summary, err := appInst.RecoveryMgr.Recover(ctx)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if summary.RecoveredTasks == 0 {
		t.Errorf("expected at least 1 recovered task")
	}

	recoveredTask, _ := appInst.TaskSvc.Get(ctx, dummyTask.ID)
	if recoveredTask.Status != task.StatusReady {
		t.Errorf("expected recovered task status READY, got %s", recoveredTask.Status)
	}
}
