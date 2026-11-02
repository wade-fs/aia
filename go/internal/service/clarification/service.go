package clarification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	domainClarify "github.com/wade/aia/internal/domain/clarification"
	"github.com/wade/aia/internal/domain/memory"
	domainTask "github.com/wade/aia/internal/domain/task"
	briefService "github.com/wade/aia/internal/service/brief"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// Service coordinates clarification inquiries, validation, user answers, brief updates, and task resumption (AIA-21 §5, §6, §7).
type Service struct {
	clarifyRepo *repository.ClarificationRepository
	briefSvc    *briefService.Service
	projectRepo *repository.ProjectRepository
	taskRepo    *repository.TaskRepository
	memoryRepo  *repository.MemoryRepository
}

// NewService creates a new clarification service.
func NewService(
	clarifyRepo *repository.ClarificationRepository,
	briefSvc *briefService.Service,
	projectRepo *repository.ProjectRepository,
	taskRepo *repository.TaskRepository,
	memoryRepo *repository.MemoryRepository,
) *Service {
	return &Service{
		clarifyRepo: clarifyRepo,
		briefSvc:    briefSvc,
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
		memoryRepo:  memoryRepo,
	}
}

// CreateRequests validates and registers a batch of clarification inquiries (AIA-21 §5.4, §6.3).
func (s *Service) CreateRequests(ctx context.Context, requests []*domainClarify.ClarificationRequest) error {
	if err := domainClarify.ValidateBatch(requests); err != nil {
		return fmt.Errorf("clarification validation failed: %w", err)
	}

	now := time.Now().UTC()
	hasBlocking := false
	var projectID string

	for _, req := range requests {
		if req.ID == "" {
			req.ID = "cr-" + uuid.New().String()[:8]
		}
		req.Status = domainClarify.StatusOpen
		req.CreatedAt = now
		req.UpdatedAt = now
		if req.Blocking {
			hasBlocking = true
		}
		projectID = req.ProjectID

		if err := s.clarifyRepo.Create(ctx, req); err != nil {
			return fmt.Errorf("failed to save clarification request %s: %w", req.ID, err)
		}
	}

	// If there are blocking questions, transition associated project tasks to WAITING_FOR_INPUT (AIA-21 §7.1, §7.2)
	if hasBlocking && s.taskRepo != nil && projectID != "" {
		tasks, _ := s.taskRepo.List(ctx, repository.TaskFilter{
			ProjectID: projectID,
			Status:    domainTask.StatusRunning,
		})
		for _, t := range tasks {
			_ = s.taskRepo.UpdateStatus(ctx, t.ID, domainTask.StatusWaitingInput)
		}
	}

	return nil
}

// Answer processes a human answer, updates the brief, writes project memory, and resumes tasks if all blocking questions are solved (AIA-21 §7.3, §7.4).
func (s *Service) Answer(ctx context.Context, requestID string, answer string) (*domainClarify.ClarificationRequest, error) {
	c, err := s.clarifyRepo.Get(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch clarification request: %w", err)
	}
	if c == nil {
		return nil, fmt.Errorf("clarification request %s not found", requestID)
	}
	if !c.IsOpen() {
		return nil, fmt.Errorf("clarification request %s is not open (status: %s)", requestID, c.Status)
	}

	cleanedAnswer := strings.TrimSpace(answer)
	if cleanedAnswer == "" {
		return nil, fmt.Errorf("answer cannot be empty")
	}

	now := time.Now().UTC()
	c.Answer = cleanedAnswer
	c.Status = domainClarify.StatusAnswered
	c.AnsweredAt = &now
	c.UpdatedAt = now

	if err := s.clarifyRepo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to update clarification request: %w", err)
	}

	// 1. Update Project Brief with the answered field
	var wsRoot string
	if s.projectRepo != nil {
		if p, _ := s.projectRepo.Get(ctx, domain.ID(c.ProjectID)); p != nil {
			wsRoot = p.Workspace
		}
	}

	if s.briefSvc != nil {
		rationale := fmt.Sprintf("使用者回答釐清問題: %s", c.Question)
		_, _ = s.briefSvc.UpdateField(
			ctx,
			c.ProjectID,
			c.FieldKey,
			cleanedAnswer,
			domainBrief.FieldStatusAnswered,
			"user",
			1.0,
			rationale,
			false,
			"user",
			c.TaskID,
			c.ID,
			wsRoot,
		)
	}

	// 2. Record in Project Memory (L1~L5 Episodic Decision)
	if s.memoryRepo != nil {
		_ = s.memoryRepo.Create(ctx, &memory.Memory{
			ID:          domain.ID("mem-cr-" + uuid.New().String()[:8]),
			Type:        memory.TypeDecision,
			Scope:       memory.ScopeProject,
			ProjectID:   domain.ID(c.ProjectID),
			Title:       fmt.Sprintf("需求釐清決策: %s", c.FieldKey),
			Content:     fmt.Sprintf("問題: %s\n原因: %s\n使用者採納答案: %s", c.Question, c.Reason, cleanedAnswer),
			Confidence:  1.0,
			Status:      "ACTIVE",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	// 3. Check if all blocking questions for the project are resolved
	blockingOpen, err := s.clarifyRepo.ListBlockingOpenByProject(ctx, c.ProjectID)
	if err == nil && len(blockingOpen) == 0 && s.taskRepo != nil {
		// All blocking clarifications resolved -> resume paused tasks to READY (AIA-21 §7.4 Continuation Execution)
		pausedTasks, _ := s.taskRepo.List(ctx, repository.TaskFilter{
			ProjectID: c.ProjectID,
			Status:    domainTask.StatusWaitingInput,
		})
		for _, pt := range pausedTasks {
			_ = s.taskRepo.UpdateStatus(ctx, pt.ID, domainTask.StatusReady)
		}
	}

	return c, nil
}

// Defer marks a question as deferred (AIA-21 §5.2).
func (s *Service) Defer(ctx context.Context, requestID string) error {
	c, err := s.clarifyRepo.Get(ctx, requestID)
	if err != nil || c == nil {
		return fmt.Errorf("clarification request not found")
	}
	c.Status = domainClarify.StatusDeferred
	return s.clarifyRepo.Update(ctx, c)
}

// Cancel marks a question as cancelled.
func (s *Service) Cancel(ctx context.Context, requestID string) error {
	c, err := s.clarifyRepo.Get(ctx, requestID)
	if err != nil || c == nil {
		return fmt.Errorf("clarification request not found")
	}
	c.Status = domainClarify.StatusCancelled
	return s.clarifyRepo.Update(ctx, c)
}

// ListByProject returns all clarification requests for a project.
func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*domainClarify.ClarificationRequest, error) {
	return s.clarifyRepo.ListByProject(ctx, projectID)
}

// ListOpenByProject returns open requests for a project.
func (s *Service) ListOpenByProject(ctx context.Context, projectID string) ([]*domainClarify.ClarificationRequest, error) {
	return s.clarifyRepo.ListOpenByProject(ctx, projectID)
}

// Get retrieves a clarification request by ID.
func (s *Service) Get(ctx context.Context, requestID string) (*domainClarify.ClarificationRequest, error) {
	return s.clarifyRepo.Get(ctx, requestID)
}
