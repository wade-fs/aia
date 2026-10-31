package execution

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/profile"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/provider"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/validation"
	"github.com/wade/aia/internal/workspace"
)

type Service struct {
	execRepo     *repository.ExecutionRepository
	taskRepo     *repository.TaskRepository
	artifactRepo *repository.ArtifactRepository
	auditRepo    *repository.AuditRepository
	prov         provider.Provider
	wsMgr        *workspace.Manager
	validator    validation.Validator
}

func NewService(
	execRepo *repository.ExecutionRepository,
	taskRepo *repository.TaskRepository,
	artifactRepo *repository.ArtifactRepository,
	auditRepo *repository.AuditRepository,
	prov provider.Provider,
	wsMgr *workspace.Manager,
	validator validation.Validator,
) *Service {
	return &Service{
		execRepo:     execRepo,
		taskRepo:     taskRepo,
		artifactRepo: artifactRepo,
		auditRepo:    auditRepo,
		prov:         prov,
		wsMgr:        wsMgr,
		validator:    validator,
	}
}

func (s *Service) Execute(ctx context.Context, t *task.Task, ag *agent.Agent, prof profile.ExecutionProfile, contextPrompt string) (*execution.Execution, error) {
	execID := uuid.New().String()

	// 1. Prepare Workspace
	wsPath, err := s.wsMgr.PrepareWorkspace(execID)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	// 2. Create Execution in STARTING state
	now := time.Now().UTC()
	execRecord := &execution.Execution{
		ID:             execID,
		TaskID:         t.ID,
		AgentID:        ag.ID,
		Provider:       s.prov.Name(),
		ProfileID:      prof.ID,
		ProfileVersion: prof.Version,
		Workspace:      wsPath,
		Status:         execution.StatusStarting,
		CreatedAt:      now,
	}

	if err := s.execRepo.Create(ctx, execRecord); err != nil {
		return nil, fmt.Errorf("failed to record execution start: %w", err)
	}

	// Update Task to RUNNING
	_ = s.taskRepo.UpdateStatus(ctx, t.ID, task.StatusRunning)

	// 3. Assemble Prompt
	fullPrompt := t.Description
	if contextPrompt != "" {
		fullPrompt = fmt.Sprintf("%s\n\nTask Goal: %s\n%s", contextPrompt, t.Title, t.Description)
	}

	// 4. Start Provider Session
	req := provider.StartRequest{
		ExecutionID: execID,
		TaskID:      t.ID,
		Agent:       *ag,
		Prompt:      fullPrompt,
		WorkDir:     wsPath,
		Profile:     prof,
		Timeout:     prof.Timeout,
	}

	session, err := s.prov.Start(ctx, req)
	if err != nil {
		execRecord.Status = execution.StatusFailed
		execRecord.ErrorMessage = err.Error()
		ended := time.Now().UTC()
		execRecord.EndedAt = &ended
		_ = s.execRepo.Update(ctx, execRecord)
		_ = s.taskRepo.UpdateStatus(ctx, t.ID, task.StatusFailed)
		return execRecord, fmt.Errorf("failed to start provider session: %w", err)
	}

	// 5. Update Execution with process details
	execRecord.Status = execution.StatusRunning
	execRecord.StartedAt = &now
	_ = s.execRepo.Update(ctx, execRecord)

	// 6. Pipe Events asynchronously
	go func() {
		for ev := range session.Events() {
			_ = s.execRepo.AppendEvent(context.Background(), &execution.ExecutionEvent{
				ID:          uuid.New().String(),
				ExecutionID: execID,
				Sequence:    ev.Sequence,
				EventType:   ev.Type,
				Source:      s.prov.Name(),
				Payload:     ev.Payload,
				CreatedAt:   ev.Timestamp,
			})
		}
	}()

	// 7. Wait for Result
	result, err := session.Wait(ctx)
	ended := time.Now().UTC()
	execRecord.EndedAt = &ended

	if err != nil {
		execRecord.Status = execution.StatusFailed
		execRecord.ErrorMessage = err.Error()
		_ = s.execRepo.Update(ctx, execRecord)
		_ = s.taskRepo.UpdateStatus(ctx, t.ID, task.StatusFailed)
		return execRecord, err
	}

	execRecord.ConversationID = result.ConversationID
	execRecord.ResultStatus = result.Status

	// 8. Result Validation
	valReq := validation.ValidationRequest{
		Task:      *t,
		Execution: *execRecord,
		Result:    result,
		Workspace: wsPath,
	}
	valRes, valErr := s.validator.Validate(ctx, valReq)
	if valErr != nil || (valRes != nil && !valRes.Accepted) {
		execRecord.Status = execution.StatusFailed
		if valRes != nil && len(valRes.Findings) > 0 {
			execRecord.ErrorMessage = valRes.Findings[0].Description
		}
		_ = s.execRepo.Update(ctx, execRecord)
		_ = s.taskRepo.UpdateStatus(ctx, t.ID, task.StatusFailed)
		slog.Warn("task validation rejected", "task_id", t.ID, "status", execRecord.Status)
		return execRecord, nil
	}

	// 9. Collect Artifacts
	artifacts, _ := s.wsMgr.CollectArtifacts(t.ID, execID, wsPath)
	for _, art := range artifacts {
		_ = s.artifactRepo.Create(ctx, &art)
	}

	// 10. Mark Execution & Task COMPLETED
	execRecord.Status = execution.StatusCompleted
	_ = s.execRepo.Update(ctx, execRecord)
	_ = s.taskRepo.UpdateStatus(ctx, t.ID, task.StatusCompleted)

	_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
		ID:           uuid.New().String(),
		ActorType:    "PROVIDER",
		ActorID:      s.prov.Name(),
		Action:       "EXECUTION_COMPLETED",
		ResourceType: "EXECUTION",
		ResourceID:   execID,
		Outcome:      "SUCCESS",
	})

	return execRecord, nil
}

func (s *Service) Get(ctx context.Context, id domain.ID) (*execution.Execution, error) {
	return s.execRepo.Get(ctx, id)
}

func (s *Service) ListByTask(ctx context.Context, taskID domain.ID) ([]execution.Execution, error) {
	return s.execRepo.ListByTask(ctx, taskID)
}

func (s *Service) ListEvents(ctx context.Context, id domain.ID) ([]execution.ExecutionEvent, error) {
	return s.execRepo.ListEvents(ctx, id, 0)
}
