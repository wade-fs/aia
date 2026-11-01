package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/eventbus"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var (
	ErrAgentSelfApprovalForbidden = errors.New("agent cannot self-approve: operational gates require explicit human owner authorization")
	ErrApprovalNotPending         = errors.New("approval is not in PENDING status")
	ErrApprovalExpired            = errors.New("approval has expired")
)

type RequestApprovalInput struct {
	RequesterEntity     string                   `json:"requester_entity"`
	OperationType       string                   `json:"operation_type"`
	OperationDetailJSON string                   `json:"operation_detail_json"`
	RiskTier            governance.RiskTier      `json:"risk_tier"`
	TimeoutPolicy       governance.TimeoutPolicy `json:"timeout_policy"`
	TimeoutDuration     time.Duration            `json:"timeout_duration"`
	TaskID              *string                  `json:"task_id,omitempty"`
	DecisionID          *string                  `json:"decision_id,omitempty"`
}

type Service struct {
	repo      *repository.ApprovalRepository
	taskSvc   *taskService.Service
	auditRepo *repository.AuditRepository
	bus       eventbus.Publisher
}

func NewService(
	repo *repository.ApprovalRepository,
	taskSvc *taskService.Service,
	auditRepo *repository.AuditRepository,
	bus eventbus.Publisher,
) *Service {
	return &Service{
		repo:      repo,
		taskSvc:   taskSvc,
		auditRepo: auditRepo,
		bus:       bus,
	}
}

// RequestApproval creates a new operational gate requiring human or supervisor confirmation.
func (s *Service) RequestApproval(ctx context.Context, input RequestApprovalInput) (*governance.Approval, error) {
	now := time.Now().UTC()
	var timeoutAt *time.Time
	var expiresAt *time.Time

	dur := input.TimeoutDuration
	if dur <= 0 {
		dur = 24 * time.Hour // Default 24h approval window
	}
	t := now.Add(dur)
	timeoutAt = &t
	expiresAt = &t

	if input.RiskTier == "" {
		input.RiskTier = governance.RiskTierHigh
	}
	if input.TimeoutPolicy == "" {
		input.TimeoutPolicy = governance.TimeoutPolicyReject
	}
	if input.RequesterEntity == "" {
		input.RequesterEntity = "AGENT:autonomous"
	}

	appr := &governance.Approval{
		ID:                  uuid.New().String(),
		RequesterEntity:     input.RequesterEntity,
		OperationType:       input.OperationType,
		OperationDetailJSON: input.OperationDetailJSON,
		Status:              governance.ApprovalPending,
		RiskTier:            input.RiskTier,
		TimeoutAt:           timeoutAt,
		TimeoutPolicy:       input.TimeoutPolicy,
		TaskID:              input.TaskID,
		DecisionID:          input.DecisionID,
		CreatedAt:           now,
		ExpiresAt:           expiresAt,
	}

	if err := s.repo.Create(ctx, appr); err != nil {
		return nil, fmt.Errorf("failed to create approval request: %w", err)
	}

	// Update linked Task to WAITING_APPROVAL if present
	if input.TaskID != nil && *input.TaskID != "" && s.taskSvc != nil {
		_ = s.taskSvc.UpdateStatus(ctx, *input.TaskID, task.StatusWaitingApproval)
	}

	// Log audit record
	if s.auditRepo != nil {
		riskStr := string(appr.RiskTier)
		govActionStr := string(governance.GovernanceActionRequiredApproval)
		apprID := appr.ID
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:               uuid.New().String(),
			ActorType:        "SYSTEM",
			ActorID:          appr.RequesterEntity,
			Action:           "APPROVAL_REQUESTED",
			ResourceType:     "APPROVAL",
			ResourceID:       appr.ID,
			Outcome:          "PENDING",
			Details:          json.RawMessage(fmt.Sprintf(`{"operation_type": %q, "risk_tier": %q}`, appr.OperationType, appr.RiskTier)),
			ApprovalID:       &apprID,
			DecisionID:       appr.DecisionID,
			RiskTier:         &riskStr,
			GovernanceAction: &govActionStr,
			CreatedAt:        now,
		})
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.TaskWaitingApproval, map[string]any{
			"approval_id": appr.ID,
			"risk_tier":   appr.RiskTier,
			"operation":   appr.OperationType,
		})
	}

	return appr, nil
}

// Approve reviews and authorizes a pending approval.
// AIA-17 Rule: Agent cannot self-approve. Reviwer must be HUMAN / OWNER / USER.
func (s *Service) Approve(ctx context.Context, approvalID string, reviewerID, reviewerNote string, actorType string) error {
	appr, err := s.repo.Get(ctx, approvalID)
	if err != nil {
		return err
	}

	if appr.Status != governance.ApprovalPending {
		return fmt.Errorf("%w: current status is %s", ErrApprovalNotPending, appr.Status)
	}

	// Strict security check: Agent cannot self-approve
	normActor := strings.ToUpper(strings.TrimSpace(actorType))
	if normActor == "AGENT" || strings.HasPrefix(strings.ToUpper(reviewerID), "AGENT:") || reviewerID == appr.RequesterEntity {
		return ErrAgentSelfApprovalForbidden
	}
	if normActor != "HUMAN" && normActor != "OWNER" && normActor != "USER" && normActor != "ADMIN" {
		return ErrAgentSelfApprovalForbidden
	}

	// Check if already expired
	now := time.Now().UTC()
	if appr.ExpiresAt != nil && now.After(*appr.ExpiresAt) {
		_ = s.repo.UpdateStatus(ctx, approvalID, governance.ApprovalExpired, &reviewerID, &reviewerNote)
		return ErrApprovalExpired
	}

	if err := s.repo.UpdateStatus(ctx, approvalID, governance.ApprovalApproved, &reviewerID, &reviewerNote); err != nil {
		return err
	}

	// If linked to a task, transition task back to READY for execution
	if appr.TaskID != nil && *appr.TaskID != "" && s.taskSvc != nil {
		_ = s.taskSvc.UpdateStatus(ctx, *appr.TaskID, task.StatusReady)
	}

	// Audit record
	if s.auditRepo != nil {
		riskStr := string(appr.RiskTier)
		govActionStr := string(governance.GovernanceActionAllowed)
		apprID := appr.ID
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:               uuid.New().String(),
			ActorType:        normActor,
			ActorID:          reviewerID,
			Action:           "APPROVAL_GRANTED",
			ResourceType:     "APPROVAL",
			ResourceID:       approvalID,
			Outcome:          "APPROVED",
			Details:          json.RawMessage(fmt.Sprintf(`{"reviewer_note": %q}`, reviewerNote)),
			ApprovalID:       &apprID,
			DecisionID:       appr.DecisionID,
			RiskTier:         &riskStr,
			GovernanceAction: &govActionStr,
			CreatedAt:        now,
		})
	}

	return nil
}

// Reject reviews and denies a pending approval.
func (s *Service) Reject(ctx context.Context, approvalID string, reviewerID, reviewerNote string, actorType string) error {
	appr, err := s.repo.Get(ctx, approvalID)
	if err != nil {
		return err
	}

	if appr.Status != governance.ApprovalPending {
		return fmt.Errorf("%w: current status is %s", ErrApprovalNotPending, appr.Status)
	}

	normActor := strings.ToUpper(strings.TrimSpace(actorType))
	if normActor == "" {
		normActor = "HUMAN"
	}

	if err := s.repo.UpdateStatus(ctx, approvalID, governance.ApprovalRejected, &reviewerID, &reviewerNote); err != nil {
		return err
	}

	// If linked to a task, transition task to CANCELLED
	if appr.TaskID != nil && *appr.TaskID != "" && s.taskSvc != nil {
		_ = s.taskSvc.UpdateStatus(ctx, *appr.TaskID, task.StatusCancelled)
	}

	now := time.Now().UTC()
	if s.auditRepo != nil {
		riskStr := string(appr.RiskTier)
		govActionStr := string(governance.GovernanceActionBlocked)
		apprID := appr.ID
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:               uuid.New().String(),
			ActorType:        normActor,
			ActorID:          reviewerID,
			Action:           "APPROVAL_REJECTED",
			ResourceType:     "APPROVAL",
			ResourceID:       approvalID,
			Outcome:          "REJECTED",
			Details:          json.RawMessage(fmt.Sprintf(`{"reviewer_note": %q}`, reviewerNote)),
			ApprovalID:       &apprID,
			DecisionID:       appr.DecisionID,
			RiskTier:         &riskStr,
			GovernanceAction: &govActionStr,
			CreatedAt:        now,
		})
	}

	return nil
}

func (s *Service) Get(ctx context.Context, id string) (*governance.Approval, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) ListPending(ctx context.Context) ([]*governance.Approval, error) {
	return s.repo.ListPending(ctx)
}

func (s *Service) List(ctx context.Context, filter repository.ApprovalFilter) ([]*governance.Approval, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) CheckExpired(ctx context.Context) (int, error) {
	return s.repo.CheckExpired(ctx, time.Now().UTC())
}
