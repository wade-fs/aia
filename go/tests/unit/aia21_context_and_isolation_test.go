package unit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/memory"
	domainProject "github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/domain/task"
	domainWriting "github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/orchestrator"
	writingService "github.com/wade/aia/internal/service/writing"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestTaskStateTransition_WaitingForInput(t *testing.T) {
	// 1. RUNNING -> WAITING_FOR_INPUT (allowed)
	if err := task.ValidateTransition(task.StatusRunning, task.StatusWaitingInput); err != nil {
		t.Fatalf("expected transition from RUNNING to WAITING_FOR_INPUT to be valid, got: %v", err)
	}

	// 2. WAITING_FOR_INPUT -> READY (allowed for continuation)
	if err := task.ValidateTransition(task.StatusWaitingInput, task.StatusReady); err != nil {
		t.Fatalf("expected transition from WAITING_FOR_INPUT to READY to be valid, got: %v", err)
	}

	// 3. WAITING_FOR_INPUT -> CANCELLED (allowed)
	if err := task.ValidateTransition(task.StatusWaitingInput, task.StatusCancelled); err != nil {
		t.Fatalf("expected transition from WAITING_FOR_INPUT to CANCELLED to be valid, got: %v", err)
	}

	// 4. WAITING_FOR_INPUT is NOT a terminal state
	tk := &task.Task{Status: task.StatusWaitingInput}
	if tk.IsTerminal() {
		t.Fatalf("WAITING_FOR_INPUT must not be marked as terminal")
	}
}

func TestContextAssembler_FullContextInjection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	memRepo := repository.NewMemoryRepository(db)
	workflowRepo := repository.NewWritingWorkflowRepository(db)

	projID := "proj-walk-taiwan"
	p := &domainProject.Project{
		ID:          domain.ID(projID),
		Name:        "徒步環台灣本島計畫書",
		Description: "徒步環台灣本島規劃與旅行紀實",
		Workspace:   "/workspace/walk-taiwan",
		DomainType:  "writing",
		Status:      domainProject.StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := projRepo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Insert brief
	brief := &domainWriting.CreativeBrief{
		ID:                 "brief-walk",
		ProjectID:          projID,
		Title:              "徒步環台灣本島計畫書",
		RawIntent:          "我想寫一本關於計劃徒步環台灣本島一圈的計畫的書",
		Genre:              "nonfiction_plan",
		Era:                "當代 2026",
		ThemeAndPhilosophy: []string{"實用性與旅行文學並重", "每日徒步安全與補給守則"},
		Scale: domainWriting.BookScale{
			TotalVolumes:      2,
			TotalChapters:     30,
			ChaptersPerVolume: 15,
		},
		CoreProtagonists: []string{"徒步規劃者", "在地文史引路人"},
		Status:           domainWriting.BriefStatusUnderClarification,
		Version:          2,
		PendingQuestions: 2,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := workflowRepo.CreateBrief(ctx, brief); err != nil {
		t.Fatalf("failed to create brief: %v", err)
	}

	// Insert memory
	mem := &memory.Memory{
		ID:          "mem-001",
		Type:        memory.TypeDecision,
		Scope:       memory.ScopeProject,
		ProjectID:   domain.ID(projID),
		Title:       "路線方向已確立為逆時針",
		Content:     "使用者確認採逆時針徒步，先走蘇花再下花東",
		Confidence:  1.0,
		Status:      "ACTIVE",
		Version:     1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := memRepo.Create(ctx, mem); err != nil {
		t.Fatalf("failed to create memory: %v", err)
	}

	assembler := orchestrator.NewContextAssembler(projRepo, memRepo, workflowRepo)

	ag := &agent.Agent{
		ID:           "writing-agent",
		Name:         "Writing Agent",
		Domain:       "writing",
		Instructions: "你是一位專精於長篇紀實與長篇小說的專業架構師，確保一致性與事實查證。",
		Enabled:      true,
	}

	tk := &task.Task{
		ID:          "task-123",
		ProjectID:   domain.ID(projID),
		Title:       "規劃書籍第一階段架構",
		Description: "請根據當前已釐清的方向建立前兩章章節綱要",
		Acceptance:  []string{"包含每日步行安全守則", "列出蘇花段大眾運輸替代應變"},
	}

	contextPrompt, err := assembler.Assemble(ctx, tk, ag)
	if err != nil {
		t.Fatalf("assembly failed: %v", err)
	}

	// Verify all 5 layers are present
	if !strings.Contains(contextPrompt, "# AGENT PROFILE & SYSTEM INSTRUCTIONS") || !strings.Contains(contextPrompt, "專業架構師") {
		t.Errorf("expected agent profile in assembled context")
	}
	if !strings.Contains(contextPrompt, "# DOMAIN GOVERNANCE & EXECUTION BOUNDARIES") {
		t.Errorf("expected domain governance in assembled context")
	}
	if !strings.Contains(contextPrompt, "徒步環台灣本島計畫書") || !strings.Contains(contextPrompt, "實用性與旅行文學並重") {
		t.Errorf("expected brief details in assembled context")
	}
	if !strings.Contains(contextPrompt, "# PROJECT MEMORY & PRIOR DECISIONS") || !strings.Contains(contextPrompt, "逆時針") {
		t.Errorf("expected memory item in assembled context")
	}
	if !strings.Contains(contextPrompt, "# ACCEPTANCE CRITERIA") || !strings.Contains(contextPrompt, "蘇花段大眾運輸替代應變") {
		t.Errorf("expected acceptance criteria in assembled context")
	}
}

func TestWorkflowService_WaitingForInput_And_TaskContinuation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	memRepo := repository.NewMemoryRepository(db)
	workflowRepo := repository.NewWritingWorkflowRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	projID := "proj-novel-changan"
	wsDir := t.TempDir()

	p := &domainProject.Project{
		ID:          domain.ID(projID),
		Name:        "長安落日",
		Description: "唐末五代歷史小說",
		Workspace:   wsDir,
		DomainType:  "writing",
		Status:      domainProject.StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := projRepo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Insert agent via repository to satisfy foreign key
	agentRepo := repository.NewAgentRepository(db)
	if err := agentRepo.Upsert(ctx, &agent.Agent{
		ID:           "writing-agent",
		Name:         "Writing Agent",
		Description:  "Novel architect",
		Domain:       "writing",
		Instructions: "Instructions",
		ProviderName: "antigravity",
		Enabled:      true,
		Version:      1,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// 1. Create task currently running
	taskID := "task-initiate-writing"
	now := time.Now().UTC()
	err := taskRepo.Create(ctx, &task.Task{
		ID:          domain.ID(taskID),
		ProjectID:   domain.ID(projID),
		AgentID:     "writing-agent",
		Title:       "Initiate Changan Novel",
		Description: "Analyze intent and setup brief",
		Status:      task.StatusRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	gapAnalyzer := writingService.NewGapAnalyzer(nil)
	workflowSvc := writingService.NewWorkflowService(workflowRepo, projRepo, memRepo, gapAnalyzer)
	workflowSvc.SetTaskRepository(taskRepo)

	// 2. Initiate project with missing scale (triggering clarifications)
	initResp, err := workflowSvc.InitiateProject(ctx, writingService.InitiateProjectRequest{
		ProjectID: projID,
		TaskID:    taskID,
		Title:     "長安落日",
		RawIntent: "以唐末黃巢起義為起點，寫長安城失陷至五代更迭的群像",
		Genre:     "historical_fiction",
		Era:       "唐末五代 (875-960)",
		Workspace: wsDir,
	})
	if err != nil {
		t.Fatalf("initiate failed: %v", err)
	}

	if initResp.Status != domainWriting.BriefStatusUnderClarification {
		t.Fatalf("expected brief status UNDER_CLARIFICATION, got %s", initResp.Status)
	}

	// 3. Verify Task was transitioned to WAITING_FOR_INPUT
	tk, err := taskRepo.Get(ctx, domain.ID(taskID))
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if tk.Status != task.StatusWaitingInput {
		t.Fatalf("expected task status WAITING_FOR_INPUT, got %s", tk.Status)
	}

	// 4. Answer all clarification requests
	pending, err := workflowRepo.ListPendingClarifications(ctx, projID)
	if err != nil || len(pending) == 0 {
		t.Fatalf("expected pending clarifications, got %d", len(pending))
	}

	for _, cr := range pending {
		answer := "16 卷共 1440 章 (每卷 90 章)"
		if cr.Category == domainWriting.CategoryWorldview {
			answer = "遵守正史骨架、合理推演、藝術虛構三層知識結構"
		}
		ansResp, aErr := workflowSvc.AnswerClarification(ctx, cr.ID, answer, wsDir)
		if aErr != nil {
			t.Fatalf("answer failed: %v", aErr)
		}
		if ansResp.AllClarified {
			break
		}
	}

	// 5. Verify Task was resumed to READY for continuation execution!
	tkAfter, err := taskRepo.Get(ctx, domain.ID(taskID))
	if err != nil {
		t.Fatalf("failed to get task after answers: %v", err)
	}
	if tkAfter.Status != task.StatusReady {
		t.Fatalf("expected task status to be resumed to READY, got %s", tkAfter.Status)
	}
}
