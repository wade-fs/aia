package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/profile"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/policy"
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
	evaluator    *policy.Evaluator
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

func (s *Service) SetPolicyEvaluator(eval *policy.Evaluator) {
	s.evaluator = eval
}

func (s *Service) Execute(ctx context.Context, t *task.Task, ag *agent.Agent, prof profile.ExecutionProfile, contextPrompt string) (*execution.Execution, error) {
	now := time.Now().UTC()

	// Check if task already has an active (STARTING or RUNNING) execution
	if s.execRepo != nil {
		if activeExecs, err := s.execRepo.ListByTask(ctx, t.ID); err == nil {
			for _, ae := range activeExecs {
				if ae.Status == execution.StatusStarting || ae.Status == execution.StatusRunning {
					slog.Info("task already has active execution, reusing in-flight execution", "task_id", t.ID, "execution_id", ae.ID)
					return &ae, nil
				}
			}
		}
	}

	if s.evaluator != nil && s.execRepo != nil {
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		if usedTokens, tErr := s.execRepo.SumTokensSince(ctx, dayStart); tErr == nil {
			bDec := s.evaluator.EvaluateResourceBudget(ctx, ag.ID, usedTokens, 0)
			if !bDec.Allowed {
				return nil, fmt.Errorf("execution blocked by budget gate: %s", bDec.Reason)
			}
		}
	}

	execID := uuid.New().String()

	// 1. Prepare Workspace
	wsPath, err := s.wsMgr.PrepareWorkspace(execID)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	// 2. Create Execution in STARTING state
	now = time.Now().UTC()
	execRecord := &execution.Execution{
		ID:             execID,
		TaskID:         t.ID,
		AgentID:        ag.ID,
		Provider:       s.prov.Name(),
		ProfileID:      prof.ID,
		ProfileVersion: prof.Version,
		Workspace:      wsPath,
		Status:         execution.StatusStarting,
		ProcessID:      os.Getpid(),
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
		fullPrompt = fmt.Sprintf("%s\n\n---\n# CURRENT TASK GOAL & INSTRUCTION\n**Title:** %s\n**Description:**\n%s", contextPrompt, t.Title, t.Description)
	} else if ag != nil && ag.Instructions != "" {
		fullPrompt = fmt.Sprintf("# AGENT INSTRUCTIONS (%s)\n%s\n\n---\n# CURRENT TASK GOAL & INSTRUCTION\n**Title:** %s\n**Description:**\n%s", ag.Name, ag.Instructions, t.Title, t.Description)
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
		ended := time.Now().UTC()
		execRecord.EndedAt = &ended
		execRecord.ErrorMessage = err.Error()
		if isAuthError(err) {
			execRecord.Status = execution.StatusBlocked
			_ = s.execRepo.Update(ctx, execRecord)
			_ = s.taskRepo.UpdateStatus(ctx, t.ID, task.StatusWaitingAuth)
			slog.Warn("failed to start session due to authentication failure; task set to WAITING_FOR_AUTH", "task_id", t.ID)
			return execRecord, fmt.Errorf("authentication required: %w", err)
		}
		execRecord.Status = execution.StatusFailed
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

			if execRecord.ConversationID == "" && len(ev.Payload) > 0 {
				var generic map[string]any
				if err := json.Unmarshal(ev.Payload, &generic); err == nil {
					if cID, ok := generic["conversation_id"].(string); ok && cID != "" {
						execRecord.ConversationID = cID
						_ = s.execRepo.UpdateConversationID(context.Background(), execID, cID)
					}
				}
			}
		}
	}()

	// 7. Wait for Result
	result, err := session.Wait(ctx)
	ended := time.Now().UTC()
	execRecord.EndedAt = &ended

	if err != nil {
		if isAuthError(err) {
			execRecord.Status = execution.StatusBlocked
			execRecord.ErrorMessage = fmt.Sprintf("Authentication failed: %v", err)
			_ = s.execRepo.Update(ctx, execRecord)
			_ = s.taskRepo.UpdateStatus(ctx, t.ID, task.StatusWaitingAuth)
			slog.Warn("execution stopped due to authentication failure; task set to WAITING_FOR_AUTH", "task_id", t.ID)
			return execRecord, err
		}
		execRecord.Status = execution.StatusFailed
		execRecord.ErrorMessage = err.Error()
		_ = s.execRepo.Update(ctx, execRecord)
		_ = s.taskRepo.UpdateStatus(ctx, t.ID, task.StatusFailed)
		return execRecord, err
	}

	if result.Status == "AUTH_REQUIRED" {
		execRecord.Status = execution.StatusBlocked
		execRecord.ErrorMessage = "Provider session indicates authentication required"
		_ = s.execRepo.Update(ctx, execRecord)
		_ = s.taskRepo.UpdateStatus(ctx, t.ID, task.StatusWaitingAuth)
		slog.Warn("execution stopped due to provider AUTH_REQUIRED; task set to WAITING_FOR_AUTH", "task_id", t.ID)
		return execRecord, nil
	}

	if result.ConversationID != "" {
		execRecord.ConversationID = result.ConversationID
	}
	execRecord.ResultStatus = result.Status
	execRecord.InputTokens = result.Usage.InputTokens
	execRecord.OutputTokens = result.Usage.OutputTokens
	execRecord.TotalTokens = result.Usage.TotalTokens
	if execRecord.TotalTokens == 0 && (execRecord.InputTokens > 0 || execRecord.OutputTokens > 0) {
		execRecord.TotalTokens = execRecord.InputTokens + execRecord.OutputTokens
	}
	_ = s.execRepo.Update(ctx, execRecord)

	if s.evaluator != nil {
		limits := s.evaluator.GetBudgetLimits()
		if limits.MaxSingleExecutionTokens > 0 && execRecord.TotalTokens > limits.MaxSingleExecutionTokens && s.auditRepo != nil {
			_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
				ID:           uuid.New().String(),
				ActorType:    "PROVIDER",
				ActorID:      ag.ID,
				Action:       "SINGLE_EXECUTION_TOKEN_CAP_EXCEEDED",
				ResourceType: "EXECUTION",
				ResourceID:   execID,
				Outcome:      "WARNING",
			})
		}
	}

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

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "auth_required") ||
		strings.Contains(msg, "authentication failed") ||
		strings.Contains(msg, "please login") ||
		strings.Contains(msg, "not logged in")
}
