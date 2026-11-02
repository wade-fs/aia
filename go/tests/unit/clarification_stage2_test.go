package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	domainClarify "github.com/wade/aia/internal/domain/clarification"
	domainProject "github.com/wade/aia/internal/domain/project"
	domainTask "github.com/wade/aia/internal/domain/task"
	briefService "github.com/wade/aia/internal/service/brief"
	clarifyService "github.com/wade/aia/internal/service/clarification"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestClarificationValidation_DisciplineLimitAndRequiredFields(t *testing.T) {
	// 1. Single validation: empty reason rejected (AIA-21 §5.4)
	reqBad := &domainClarify.ClarificationRequest{
		ProjectID: "proj-1",
		FieldKey:  "route_scope",
		Question:  "請問預計徒步路線是順時針還是逆時針？",
		Reason:    "", // Missing reason
		Impact:    "決定沿途補給點規劃順序",
	}
	if err := domainClarify.Validate(reqBad); err == nil {
		t.Errorf("expected error for empty reason, got nil")
	}

	// 2. Single validation: empty impact rejected
	reqBad2 := &domainClarify.ClarificationRequest{
		ProjectID: "proj-1",
		FieldKey:  "route_scope",
		Question:  "請問預計徒步路線是順時針還是逆時針？",
		Reason:    "規劃路線需要",
		Impact:    "", // Missing impact
	}
	if err := domainClarify.Validate(reqBad2); err == nil {
		t.Errorf("expected error for empty impact, got nil")
	}

	// 3. Batch validation: batch limit <= 5 (AIA-21 §6.3)
	var overBatch []*domainClarify.ClarificationRequest
	for i := 0; i < 6; i++ {
		overBatch = append(overBatch, &domainClarify.ClarificationRequest{
			ProjectID: "proj-1",
			FieldKey:  string(rune('a' + i)),
			Question:  "問題內容？",
			Reason:    "釐清需求",
			Impact:    "影響範圍",
		})
	}
	if err := domainClarify.ValidateBatch(overBatch); err == nil {
		t.Errorf("expected error for exceeding 5 questions limit, got nil")
	}

	// 4. Batch validation: duplicate field key rejected
	dupBatch := []*domainClarify.ClarificationRequest{
		{
			ProjectID: "proj-1",
			FieldKey:  "target_audience",
			Question:  "請問目標讀者是誰？",
			Reason:    "確定文風",
			Impact:    "章節語氣與難易度",
		},
		{
			ProjectID: "proj-1",
			FieldKey:  "target_audience",
			Question:  "重複提問目標讀者？",
			Reason:    "確定文風",
			Impact:    "章節語氣與難易度",
		},
	}
	if err := domainClarify.ValidateBatch(dupBatch); err == nil {
		t.Errorf("expected error for duplicate field key, got nil")
	}
}

func TestClarificationService_Lifecycle_And_Continuation(t *testing.T) {
	// Canonical Continuation Scenario (AIA-21 §7, §17):
	// A running task hits a requirement gap, creates clarification requests with blocking flag.
	// Task transitions to WAITING_FOR_INPUT.
	// User answers questions via clarification service.
	// Once all blocking questions are answered, brief is updated, memory recorded, and task resumes to READY.

	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	wsDir := t.TempDir()
	projRepo := repository.NewProjectRepository(db)
	briefRepo := repository.NewGenericBriefRepository(db)
	clarifyRepo := repository.NewClarificationRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	memRepo := repository.NewMemoryRepository(db)
	agentRepo := repository.NewAgentRepository(db)

	briefSvc := briefService.NewService(briefRepo, projRepo)
	clarifySvc := clarifyService.NewService(clarifyRepo, briefSvc, projRepo, taskRepo, memRepo)

	projID := "proj-taiwan-walk"
	p := &domainProject.Project{
		ID:          domain.ID(projID),
		Name:        "徒步環島計畫",
		Workspace:   wsDir,
		DomainType:  "general",
		Status:      domainProject.StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := projRepo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Seed Agent
	ag := &agent.Agent{
		ID:           "agy-writer",
		Name:         "Lead Writer",
		Domain:       "writing",
		Enabled:      true,
		Instructions: "Write comprehensive travel plans",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := agentRepo.Upsert(ctx, ag); err != nil {
		t.Fatalf("failed to upsert agent: %v", err)
	}

	// Create running task
	taskID := "task-walk-01"
	tk := &domainTask.Task{
		ID:          domain.ID(taskID),
		ProjectID:   domain.ID(projID),
		Title:       "規劃徒步環島全書架構",
		AgentID:     "agy-writer",
		Status:      domainTask.StatusRunning,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := taskRepo.Create(ctx, tk, nil); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Initialize Brief
	_, err := briefSvc.InitializeBrief(ctx, projID, "nonfiction_plan", "徒步環台灣本島一圈計畫", wsDir)
	if err != nil {
		t.Fatalf("failed to init brief: %v", err)
	}

	// 1. Submit Clarification Batch with 1 blocking (P0) and 1 non-blocking (P1)
	reqs := []*domainClarify.ClarificationRequest{
		{
			ID:        "cr-01",
			ProjectID: projID,
			TaskID:    taskID,
			FieldKey:  "route_scope",
			Question:  "徒步路線主要走省道海線、山線，還是規劃特殊縱走步道？",
			Reason:    "不同路線的地形難度、距離與安全考量差異極大，直接決定全書分卷結構。",
			Impact:    "無法確定各章節路段劃分與補給章節篇幅。",
			Priority:  domainClarify.PriorityP0,
			Blocking:  true,
		},
		{
			ID:        "cr-02",
			ProjectID: projID,
			TaskID:    taskID,
			FieldKey:  "accommodation_preference",
			Question:  "沿途住宿以露營、香客大樓、青年旅館或商務旅館為主？",
			Reason:    "影響裝備篇建議的負重清單與預算估算。",
			Impact:    "影響第 3 章裝備推薦清單，但不阻礙前兩章總覽規劃。",
			Priority:  domainClarify.PriorityP1,
			Blocking:  false,
		},
	}

	if err := clarifySvc.CreateRequests(ctx, reqs); err != nil {
		t.Fatalf("CreateRequests failed: %v", err)
	}

	// Verify task transitioned to WAITING_FOR_INPUT due to blocking cr-01
	curTask, err := taskRepo.Get(ctx, domain.ID(taskID))
	if err != nil || curTask.Status != domainTask.StatusWaitingInput {
		t.Fatalf("expected task status WAITING_FOR_INPUT, got %v (err: %v)", curTask.Status, err)
	}

	// 2. User answers non-blocking cr-02 first
	ans2, err := clarifySvc.Answer(ctx, "cr-02", "以青年旅館與平價民宿為主，緊急情況搭帳露營。")
	if err != nil {
		t.Fatalf("Answer cr-02 failed: %v", err)
	}
	if ans2.Status != domainClarify.StatusAnswered {
		t.Fatalf("expected cr-02 to be ANSWERED, got %s", ans2.Status)
	}

	// Verify task is STILL WAITING_FOR_INPUT because blocking cr-01 is still open
	curTask, _ = taskRepo.Get(ctx, domain.ID(taskID))
	if curTask.Status != domainTask.StatusWaitingInput {
		t.Fatalf("expected task status to remain WAITING_FOR_INPUT while P0 is open, got %s", curTask.Status)
	}

	// 3. User answers blocking cr-01
	ans1, err := clarifySvc.Answer(ctx, "cr-01", "以省道一號及省道九號環島公路為主線，兼走台 11 海線。")
	if err != nil {
		t.Fatalf("Answer cr-01 failed: %v", err)
	}
	if ans1.Status != domainClarify.StatusAnswered {
		t.Fatalf("expected cr-01 to be ANSWERED, got %s", ans1.Status)
	}

	// Verify task has automatically resumed to READY (AIA-21 §7.4 Continuation Execution)
	curTask, _ = taskRepo.Get(ctx, domain.ID(taskID))
	if curTask.Status != domainTask.StatusReady {
		t.Fatalf("expected task status to transition to READY, got %s", curTask.Status)
	}

	// 4. Verify Brief updated and reflects answered values
	brief, err := briefSvc.GetBrief(ctx, projID)
	if err != nil {
		t.Fatalf("failed to get brief: %v", err)
	}
	f1, ok1 := brief.GetField("route_scope")
	if !ok1 || f1.Status != domainBrief.FieldStatusAnswered || f1.Value != "以省道一號及省道九號環島公路為主線，兼走台 11 海線。" {
		t.Fatalf("expected route_scope to be answered, got %+v", f1)
	}

	// 5. Verify Project Memory recorded decisions
	mems, err := memRepo.Search(ctx, repository.MemoryQuery{ProjectID: domain.ID(projID)})
	if err != nil || len(mems) != 2 {
		t.Fatalf("expected 2 memory entries recorded for clarifications, got %d (err: %v)", len(mems), err)
	}
}

func TestClarificationService_DeferAndCancel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	p := &domainProject.Project{
		ID:          domain.ID("p-test"),
		Name:        "Test Project",
		DomainType:  "general",
		Status:      domainProject.StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := projRepo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	clarifyRepo := repository.NewClarificationRepository(db)
	clarifySvc := clarifyService.NewService(clarifyRepo, nil, projRepo, nil, nil)

	req := &domainClarify.ClarificationRequest{
		ID:        "cr-defer-01",
		ProjectID: "p-test",
		FieldKey:  "publisher",
		Question:  "出版社是否已簽約？",
		Reason:    "版權宣告",
		Impact:    "版權頁資訊",
		Priority:  domainClarify.PriorityP2,
		Blocking:  false,
	}
	if err := clarifySvc.CreateRequests(ctx, []*domainClarify.ClarificationRequest{req}); err != nil {
		t.Fatalf("CreateRequests failed: %v", err)
	}

	if err := clarifySvc.Defer(ctx, "cr-defer-01"); err != nil {
		t.Fatalf("Defer failed: %v", err)
	}
	c, _ := clarifySvc.Get(ctx, "cr-defer-01")
	if c.Status != domainClarify.StatusDeferred {
		t.Fatalf("expected status DEFERRED, got %s", c.Status)
	}

	if err := clarifySvc.Cancel(ctx, "cr-defer-01"); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}
	c, _ = clarifySvc.Get(ctx, "cr-defer-01")
	if c.Status != domainClarify.StatusCancelled {
		t.Fatalf("expected status CANCELLED, got %s", c.Status)
	}
}
