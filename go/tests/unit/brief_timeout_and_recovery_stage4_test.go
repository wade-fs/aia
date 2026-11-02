package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	domainClarify "github.com/wade/aia/internal/domain/clarification"
	domainProject "github.com/wade/aia/internal/domain/project"
	domainTask "github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/recovery"
	briefService "github.com/wade/aia/internal/service/brief"
	clarifyService "github.com/wade/aia/internal/service/clarification"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestBriefTimeout_NeverAssumesConsent(t *testing.T) {
	// AIA-21 §13.7: Timeout Policy & Anti-Assumption Governance Rule:
	// "不允許因 Timeout 自動取得使用者同意" (Never assume user consent due to timeout).
	// An expired clarification request must transition to EXPIRED, but associated
	// blocking tasks must NOT be unblocked with assumed default values.

	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	briefRepo := repository.NewGenericBriefRepository(db)
	clarifyRepo := repository.NewClarificationRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	agentRepo := repository.NewAgentRepository(db)

	briefSvc := briefService.NewService(briefRepo, projRepo)
	briefSvc.SetAuditRepo(auditRepo)

	clarifySvc := clarifyService.NewService(clarifyRepo, briefSvc, projRepo, taskRepo, nil)
	clarifySvc.SetAuditRepo(auditRepo)

	projID := "proj-timeout-test"
	p := &domainProject.Project{
		ID:          domain.ID(projID),
		Name:        "逾期治理測試專案",
		DomainType:  "general",
		Status:      domainProject.StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := projRepo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	ag := &agent.Agent{
		ID:        "agy-timeout",
		Name:      "Timeout Agent",
		Domain:    "writing",
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = agentRepo.Upsert(ctx, ag)

	taskID := "task-timeout-01"
	tk := &domainTask.Task{
		ID:        domain.ID(taskID),
		ProjectID: domain.ID(projID),
		Title:     "待釐清逾期任務",
		AgentID:   "agy-timeout",
		Status:    domainTask.StatusRunning,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = taskRepo.Create(ctx, tk, nil)

	// Create clarification request with past expiration
	past := time.Now().UTC().Add(-2 * time.Hour)
	req := &domainClarify.ClarificationRequest{
		ID:        "cr-timeout-01",
		ProjectID: projID,
		TaskID:    taskID,
		FieldKey:  "route_scope",
		Question:  "請問路線走向？",
		Reason:    "規劃路線需要",
		Impact:    "章節劃分",
		Priority:  domainClarify.PriorityP0,
		Blocking:  true,
		ExpiresAt: &past,
	}

	if err := clarifySvc.CreateRequests(ctx, []*domainClarify.ClarificationRequest{req}); err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Task should be WAITING_FOR_INPUT
	curTask, _ := taskRepo.Get(ctx, domain.ID(taskID))
	if curTask.Status != domainTask.StatusWaitingInput {
		t.Fatalf("expected task status WAITING_FOR_INPUT, got %s", curTask.Status)
	}

	// 1. Trigger CheckExpirations
	expired, err := clarifySvc.CheckExpirations(ctx)
	if err != nil {
		t.Fatalf("CheckExpirations failed: %v", err)
	}
	if len(expired) != 1 || expired[0].ID != "cr-timeout-01" {
		t.Fatalf("expected 1 expired request cr-timeout-01, got %d", len(expired))
	}

	// 2. Verify status changed to EXPIRED
	fetched, _ := clarifyRepo.Get(ctx, "cr-timeout-01")
	if fetched.Status != domainClarify.StatusExpired {
		t.Fatalf("expected status EXPIRED, got %s", fetched.Status)
	}

	// 3. CRITICAL GOVERNANCE CHECK: Task must NOT be resumed to READY with assumed consent!
	curTask, _ = taskRepo.Get(ctx, domain.ID(taskID))
	if curTask.Status == domainTask.StatusReady {
		t.Fatalf("VIOLATION of AIA-21 §13.7: task was automatically promoted to READY upon timeout!")
	}
	if curTask.Status != domainTask.StatusWaitingInput {
		t.Fatalf("expected task to remain in WAITING_FOR_INPUT, got %s", curTask.Status)
	}

	// 4. Verify Audit Log recorded expiration
	logs, err := auditRepo.List(ctx, 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected audit logs, err: %v", err)
	}
	found := false
	for _, l := range logs {
		if l.Action == "CLARIFICATION_EXPIRED" && l.ResourceID == "cr-timeout-01" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected CLARIFICATION_EXPIRED audit log entry")
	}
}

func TestBriefVersionConflict_OptimisticLocking(t *testing.T) {
	// AIA-21 §12.3: Brief Version Control & Optimistic Concurrency:
	// "每次正式變更都產生新版本。不得讓不同 Execution 在不知情的情況下覆蓋對方的 Brief。"

	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	briefRepo := repository.NewGenericBriefRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	svc := briefService.NewService(briefRepo, projRepo)
	svc.SetAuditRepo(auditRepo)

	projID := "proj-version-lock"
	p := &domainProject.Project{
		ID:         domain.ID(projID),
		Name:       "版本樂觀鎖測試",
		DomainType: "general",
		Status:     domainProject.StatusActive,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	_ = projRepo.Create(ctx, p)

	// 1. Initialize Brief (starts at version 1)
	brief, err := svc.InitializeBrief(ctx, projID, "nonfiction_plan", "徒步環台灣本島", "")
	if err != nil {
		t.Fatalf("init brief failed: %v", err)
	}
	if brief.BriefVersion != 1 {
		t.Fatalf("expected initial brief version 1, got %d", brief.BriefVersion)
	}

	// 2. Successful update with expectedVersion = 1
	updated, err := svc.UpdateFieldWithVersion(
		ctx, projID, 1, "daily_mileage_km", 25,
		domainBrief.FieldStatusAnswered, "user", 1.0, "使用者表示每天走 25 公里",
		false, "user", "task-1", "cr-1", "",
	)
	if err != nil {
		t.Fatalf("update with matching expectedVersion 1 failed: %v", err)
	}
	if updated.BriefVersion != 2 {
		t.Fatalf("expected version to increment to 2, got %d", updated.BriefVersion)
	}

	// 3. Stale update with expectedVersion = 1 must be REJECTED with ErrBriefVersionConflict
	_, err = svc.UpdateFieldWithVersion(
		ctx, projID, 1, "daily_mileage_km", 30,
		domainBrief.FieldStatusAnswered, "user", 1.0, "過時 Execution 試圖覆蓋",
		false, "user", "task-stale", "", "",
	)
	if err == nil {
		t.Fatalf("expected ErrBriefVersionConflict for stale version 1, got nil")
	}
	if !errors.Is(err, repository.ErrBriefVersionConflict) {
		t.Fatalf("expected errors.Is ErrBriefVersionConflict, got: %v", err)
	}

	// 4. Update with expectedVersion = 2 succeeds and increments to 3
	updated3, err := svc.UpdateFieldWithVersion(
		ctx, projID, 2, "daily_mileage_km", 22,
		domainBrief.FieldStatusConfirmed, "research", 0.95, "確認採 22 公里",
		false, "agent", "task-3", "", "",
	)
	if err != nil {
		t.Fatalf("update with matching expectedVersion 2 failed: %v", err)
	}
	if updated3.BriefVersion != 3 {
		t.Fatalf("expected version to increment to 3, got %d", updated3.BriefVersion)
	}
}

func TestStartupRecovery_WaitingForInput_And_OfflineClarification(t *testing.T) {
	// AIA-21 §7.2, §7.4, Stage 4:
	// Verify system recovery preserves WAITING_FOR_INPUT tasks across crashes/restarts,
	// and automatically promotes them to READY when all blocking clarifications have been resolved.

	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	clarifyRepo := repository.NewClarificationRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	execRepo := repository.NewExecutionRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	recMgr := recovery.NewManager(taskRepo, execRepo, auditRepo)
	recMgr.SetClarificationRepository(clarifyRepo)

	projID := "proj-rec-recovery"
	p := &domainProject.Project{
		ID:         domain.ID(projID),
		Name:       "重啟恢復測試專案",
		DomainType: "general",
		Status:     domainProject.StatusActive,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	_ = projRepo.Create(ctx, p)

	// Seed Task in WAITING_FOR_INPUT
	taskID := "task-rec-wait-01"
	tk := &domainTask.Task{
		ID:        domain.ID(taskID),
		ProjectID: domain.ID(projID),
		Title:     "等待使用者回答路線之任務",
		Status:    domainTask.StatusWaitingInput,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = taskRepo.Create(ctx, tk, nil)

	// Seed open blocking clarification
	req := &domainClarify.ClarificationRequest{
		ID:        "cr-rec-01",
		ProjectID: projID,
		TaskID:    taskID,
		FieldKey:  "route_scope",
		Question:  "順時針還是逆時針？",
		Reason:    "行程規劃",
		Impact:    "章節劃分",
		Priority:  domainClarify.PriorityP0,
		Blocking:  true,
		Status:    domainClarify.StatusOpen,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = clarifyRepo.Create(ctx, req)

	// Case 1: Recovery runs while blocking clarification is STILL open
	summary, err := recMgr.Recover(ctx)
	if err != nil {
		t.Fatalf("Recover failed: %v", err)
	}
	if summary.PreservedWaitingInput != 1 {
		t.Fatalf("expected PreservedWaitingInput == 1, got %d", summary.PreservedWaitingInput)
	}

	curTask, _ := taskRepo.Get(ctx, domain.ID(taskID))
	if curTask.Status != domainTask.StatusWaitingInput {
		t.Fatalf("expected task status to remain WAITING_FOR_INPUT, got %s", curTask.Status)
	}

	// Case 2: User answered the clarification while system was restarting / offline
	req.Status = domainClarify.StatusAnswered
	req.Answer = "順時針海線"
	_ = clarifyRepo.Update(ctx, req)

	// Now recovery runs again
	summary2, err := recMgr.Recover(ctx)
	if err != nil {
		t.Fatalf("Recover failed: %v", err)
	}
	if summary2.ReadyTasksPromoted != 1 {
		t.Fatalf("expected ReadyTasksPromoted == 1, got %d", summary2.ReadyTasksPromoted)
	}

	// Task should now be promoted to READY for continuation execution!
	curTask, _ = taskRepo.Get(ctx, domain.ID(taskID))
	if curTask.Status != domainTask.StatusReady {
		t.Fatalf("expected task status to transition to READY, got %s", curTask.Status)
	}
}
