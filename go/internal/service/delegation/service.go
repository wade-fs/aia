package delegation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	domainExp "github.com/wade/aia/internal/domain/experiment"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type CreateDelegationInput struct {
	ExperimentID       *string                     `json:"experiment_id,omitempty"`
	RequesterAgentID   string                      `json:"requester_agent_id"`
	TargetAgentID      string                      `json:"target_agent_id"`
	SourceProjectID    *string                     `json:"source_project_id,omitempty"`
	GoalID             *string                     `json:"goal_id,omitempty"`
	ParentTaskID       *string                     `json:"parent_task_id,omitempty"`
	Deadline           *time.Time                  `json:"deadline,omitempty"`
	ResourceBudget     map[string]any              `json:"resource_budget,omitempty"`
	PermissionScope    string                      `json:"permission_scope,omitempty"`
	CollaborationMode  domainExp.CollaborationMode `json:"collaboration_mode,omitempty"`
	TaskSpec           map[string]any              `json:"task_spec"`
	AcceptanceCriteria map[string]any              `json:"acceptance_criteria"`
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

	taskSpec := in.TaskSpec
	if taskSpec == nil {
		taskSpec = make(map[string]any)
	}
	if in.ParentTaskID != nil {
		taskSpec["parent_task_id"] = *in.ParentTaskID
	}
	if in.Deadline != nil {
		taskSpec["deadline"] = in.Deadline.Format(time.RFC3339)
	}
	if in.ResourceBudget != nil {
		taskSpec["resource_budget"] = in.ResourceBudget
	}
	if in.PermissionScope != "" {
		taskSpec["permission_scope"] = in.PermissionScope
	}
	if in.CollaborationMode != "" {
		taskSpec["collaboration_mode"] = string(in.CollaborationMode)
	}

	specJSON, _ := json.Marshal(taskSpec)
	gateJSON, _ := json.Marshal(in.AcceptanceCriteria)

	reqID := uuid.New().String()
	now := time.Now().UTC()

	var spawnedTaskID *string
	if s.taskSvc != nil && in.SourceProjectID != nil {
		title := fmt.Sprintf("Delegated from %s to %s", in.RequesterAgentID, in.TargetAgentID)
		if t, ok := taskSpec["title"].(string); ok && t != "" {
			title = t
		}
		tk, err := s.taskSvc.Create(ctx, taskService.CreateTaskRequest{
			ProjectID:   domain.ID(*in.SourceProjectID),
			AgentID:     domain.ID(in.TargetAgentID),
			Title:       title,
			Description: string(specJSON),
			Priority:    1,
		})
		if err != nil {
			return nil, fmt.Errorf("spawn delegated task: %w", err)
		}
		if tk != nil {
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
		ParentTaskID:       in.ParentTaskID,
		Deadline:           in.Deadline,
		CollaborationMode:  in.CollaborationMode,
		PermissionScope:    in.PermissionScope,
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

// EvaluateAcceptanceGate evaluates the submitted deliverable against the acceptance criteria (AIA-12 §6).
func EvaluateAcceptanceGate(acceptanceGateJSON, deliverableJSON string) (bool, string) {
	if strings.TrimSpace(acceptanceGateJSON) == "" || strings.TrimSpace(acceptanceGateJSON) == "{}" {
		return true, "No acceptance criteria specified"
	}
	var gate map[string]any
	if err := json.Unmarshal([]byte(acceptanceGateJSON), &gate); err != nil {
		return true, "Unparseable acceptance criteria, skipped"
	}
	if len(gate) == 0 {
		return true, "Empty criteria"
	}

	var deliv map[string]any
	if err := json.Unmarshal([]byte(deliverableJSON), &deliv); err != nil {
		return false, "Invalid deliverable JSON format"
	}

	// 1. Check required keys
	if reqKeys, ok := gate["required_keys"].([]any); ok {
		for _, k := range reqKeys {
			keyStr, _ := k.(string)
			if keyStr != "" {
				if _, exists := deliv[keyStr]; !exists {
					return false, fmt.Sprintf("Missing required deliverable key: %s", keyStr)
				}
			}
		}
	}

	// 2. Check min_score
	if minScoreVal, ok := gate["min_score"]; ok {
		minScore := toFloat(minScoreVal)
		actualScore := toFloat(deliv["score"])
		if actualScore == 0 {
			actualScore = toFloat(deliv["evaluation_score"])
		}
		if actualScore < minScore {
			return false, fmt.Sprintf("Score %.2f is below required minimum %.2f", actualScore, minScore)
		}
	}

	// 3. Check min_pass
	if minPassVal, ok := gate["min_pass"]; ok {
		minPass := toFloat(minPassVal)
		actualPass := toFloat(deliv["pass_rate"])
		if actualPass == 0 && deliv["min_pass"] != nil {
			actualPass = toFloat(deliv["min_pass"])
		}
		if actualPass < minPass {
			// If boolean tests_passed is provided and true, treat as passing
			if tp, ok := deliv["tests_passed"].(bool); !ok || !tp {
				return false, fmt.Sprintf("Pass rate %.2f is below required %.2f", actualPass, minPass)
			}
		}
	}

	// 4. Check explicit tests_passed requirement
	if reqTests, ok := gate["require_tests_passed"].(bool); ok && reqTests {
		if tp, ok := deliv["tests_passed"].(bool); !ok || !tp {
			return false, "Tests did not pass as required by acceptance gate"
		}
	}

	return true, "Acceptance criteria satisfied"
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

// VerifyDelivery checks if the deliverable satisfies the acceptance gate.
func (s *Service) VerifyDelivery(ctx context.Context, id string, callerPassed bool, comment string) error {
	d, err := s.repo.GetDelegationRequest(ctx, id)
	if err != nil {
		return err
	}
	if d.Status != "DELIVERED" {
		return fmt.Errorf("delegation %s is not in DELIVERED state (current: %s)", id, d.Status)
	}

	gatePassed, gateReason := EvaluateAcceptanceGate(d.AcceptanceGateJSON, d.DeliverableJSON)
	finalPassed := callerPassed && gatePassed

	newStatus := "VERIFIED"
	finalComment := comment
	if !finalPassed {
		newStatus = "FAILED"
		if !gatePassed {
			if finalComment != "" {
				finalComment = fmt.Sprintf("%s (Acceptance Gate: %s)", finalComment, gateReason)
			} else {
				finalComment = fmt.Sprintf("Acceptance Gate Failed: %s", gateReason)
			}
		}
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
			Details:      json.RawMessage(fmt.Sprintf(`{"passed":%v,"gate_passed":%v,"comment":%q}`, finalPassed, gatePassed, finalComment)),
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
