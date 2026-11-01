package delegation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	domainExp "github.com/wade/aia/internal/domain/experiment"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type CreateDelegationInput struct {
	ExperimentID       *string        `json:"experiment_id,omitempty"`
	RequesterAgentID   string         `json:"requester_agent_id"`
	TargetAgentID      string         `json:"target_agent_id"`
	SourceProjectID    *string        `json:"source_project_id,omitempty"`
	GoalID             *string        `json:"goal_id,omitempty"`
	TaskSpec           map[string]any `json:"task_spec"`
	AcceptanceCriteria map[string]any `json:"acceptance_criteria"`
}

type Service struct {
	repo      *repository.ExperimentRepository
	taskSvc   *taskService.Service
	auditRepo *repository.AuditRepository
}

func NewService(repo *repository.ExperimentRepository, taskSvc *taskService.Service, auditRepo *repository.AuditRepository) *Service {
	return &Service{
		repo:      repo,
		taskSvc:   taskSvc,
		auditRepo: auditRepo,
	}
}

// RequestDelegation initiates a cross-domain contract (AIA-12 §21).
func (s *Service) RequestDelegation(ctx context.Context, in CreateDelegationInput) (*domainExp.DelegationRequest, error) {
	if in.RequesterAgentID == "" || in.TargetAgentID == "" {
		return nil, fmt.Errorf("requester_agent_id and target_agent_id are required")
	}
	if in.RequesterAgentID == in.TargetAgentID {
		return nil, fmt.Errorf("cannot delegate to self: requester and target must be distinct agents")
	}

	specJSON, _ := json.Marshal(in.TaskSpec)
	gateJSON, _ := json.Marshal(in.AcceptanceCriteria)

	reqID := uuid.New().String()
	now := time.Now().UTC()

	var spawnedTaskID *string
	if s.taskSvc != nil && in.SourceProjectID != nil {
		title := fmt.Sprintf("Delegated from %s to %s", in.RequesterAgentID, in.TargetAgentID)
		if t, ok := in.TaskSpec["title"].(string); ok && t != "" {
			title = t
		}
		tk, err := s.taskSvc.Create(ctx, taskService.CreateTaskRequest{
			ProjectID:   domain.ID(*in.SourceProjectID),
			AgentID:     domain.ID(in.TargetAgentID),
			Title:       title,
			Description: string(specJSON),
			Priority:    1,
		})
		if err == nil && tk != nil {
			idStr := string(tk.ID)
			spawnedTaskID = &idStr
		}
	}

	d := &domainExp.DelegationRequest{
		ID:                 reqID,
		ExperimentID:       in.ExperimentID,
		RequesterAgentID:   in.RequesterAgentID,
		TargetAgentID:      in.TargetAgentID,
		SourceProjectID:    in.SourceProjectID,
		GoalID:             in.GoalID,
		SpawnedTaskID:      spawnedTaskID,
		TaskSpecJSON:       string(specJSON),
		AcceptanceGateJSON: string(gateJSON),
		DeliverableJSON:    "{}",
		Status:             "PENDING",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.repo.CreateDelegationRequest(ctx, d); err != nil {
		return nil, fmt.Errorf("create delegation request: %w", err)
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "AGENT",
			ActorID:      in.RequesterAgentID,
			Action:       "DELEGATION_REQUESTED",
			ResourceType: "DELEGATION_REQUEST",
			ResourceID:   d.ID,
			Outcome:      "SUCCESS",
			Details:      json.RawMessage(string(specJSON)),
			CreatedAt:    now,
		})
	}

	return d, nil
}

// AcceptDelegation transitions status from PENDING to IN_PROGRESS.
func (s *Service) AcceptDelegation(ctx context.Context, id string) error {
	d, err := s.repo.GetDelegationRequest(ctx, id)
	if err != nil {
		return err
	}
	if d.Status != "PENDING" && d.Status != "ACCEPTED" {
		return fmt.Errorf("cannot accept delegation in status %s", d.Status)
	}
	return s.repo.UpdateDelegationStatus(ctx, id, "IN_PROGRESS", "")
}

// DeliverWork submits the deliverable artifact/code from TargetAgent.
func (s *Service) DeliverWork(ctx context.Context, id string, deliverable map[string]any) error {
	d, err := s.repo.GetDelegationRequest(ctx, id)
	if err != nil {
		return err
	}
	if d.Status != "IN_PROGRESS" && d.Status != "PENDING" && d.Status != "ACCEPTED" {
		return fmt.Errorf("cannot deliver delegation in status %s", d.Status)
	}
	b, _ := json.Marshal(deliverable)
	return s.repo.UpdateDelegationStatus(ctx, id, "DELIVERED", string(b))
}

// VerifyDelivery checks if the deliverable satisfies the acceptance gate.
func (s *Service) VerifyDelivery(ctx context.Context, id string, passed bool, comment string) error {
	d, err := s.repo.GetDelegationRequest(ctx, id)
	if err != nil {
		return err
	}
	if d.Status != "DELIVERED" {
		return fmt.Errorf("delegation %s is not in DELIVERED state (current: %s)", id, d.Status)
	}

	newStatus := "VERIFIED"
	if !passed {
		newStatus = "FAILED"
	}
	now := time.Now().UTC()
	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "AGENT",
			ActorID:      d.RequesterAgentID,
			Action:       "DELEGATION_VERIFIED",
			ResourceType: "DELEGATION_REQUEST",
			ResourceID:   d.ID,
			Outcome:      newStatus,
			Details:      json.RawMessage(fmt.Sprintf(`{"passed":%v,"comment":%q}`, passed, comment)),
			CreatedAt:    now,
		})
	}

	return s.repo.UpdateDelegationStatus(ctx, id, newStatus, "")
}

func (s *Service) Get(ctx context.Context, id string) (*domainExp.DelegationRequest, error) {
	return s.repo.GetDelegationRequest(ctx, id)
}

func (s *Service) List(ctx context.Context, requester, target, status string, limit int) ([]domainExp.DelegationRequest, error) {
	return s.repo.ListDelegationRequests(ctx, requester, target, status, limit)
}
