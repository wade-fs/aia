package governance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/policy"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var (
	ErrEmergencyStopActive = errors.New("runtime is under EMERGENCY_STOP: operations are frozen")
	ErrUnauthorizedActor   = errors.New("unauthorized actor: operational governance action requires human operator")
)

type Service struct {
	govRepo   *repository.GovernanceRepository
	auditRepo *repository.AuditRepository
	policy    *policy.Evaluator
	bus       eventbus.Publisher
}

func NewService(
	govRepo *repository.GovernanceRepository,
	auditRepo *repository.AuditRepository,
	policy *policy.Evaluator,
	bus eventbus.Publisher,
) *Service {
	return &Service{
		govRepo:   govRepo,
		auditRepo: auditRepo,
		policy:    policy,
		bus:       bus,
	}
}

// GetRuntimeControl gets the current global runtime control state.
func (s *Service) GetRuntimeControl(ctx context.Context) (*governance.RuntimeControl, error) {
	return s.govRepo.GetControl(ctx)
}

// TriggerEmergencyStop engages the Emergency Stop gate across the system.
// Any actor or automated circuit breaker can initiate an emergency stop.
func (s *Service) TriggerEmergencyStop(ctx context.Context, actor governance.Actor, reason string) error {
	if reason == "" {
		reason = "Emergency stop requested"
	}
	actorStr := fmt.Sprintf("%s:%s", actor.Type, actor.ID)

	if err := s.govRepo.SetEmergencyStop(ctx, actorStr, reason); err != nil {
		return fmt.Errorf("failed to persist emergency stop: %w", err)
	}

	now := time.Now().UTC()
	if s.auditRepo != nil {
		govAction := string(governance.GovernanceActionEmergencyStop)
		risk := string(governance.RiskTierCritical)
		details, _ := json.Marshal(map[string]any{
			"reason":       reason,
			"triggered_by": actorStr,
		})
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:               uuid.New().String(),
			ActorType:        actor.Type,
			ActorID:          actor.ID,
			Action:           "EMERGENCY_STOP_TRIGGERED",
			ResourceType:     "RUNTIME",
			ResourceID:       "global",
			Outcome:          "EMERGENCY_STOP",
			Details:          details,
			RiskTier:         &risk,
			GovernanceAction: &govAction,
			CreatedAt:        now,
		})
	}

	if s.bus != nil {
		s.bus.Publish("runtime.emergency_stop", map[string]any{
			"actor":     actorStr,
			"reason":    reason,
			"timestamp": now,
		})
	}

	return nil
}

// ResumeDispatch resumes dispatching. AIA-17 Rule: Agent CANNOT resume itself.
// The actor MUST be a verified human operator (HUMAN / OWNER / USER / ADMIN).
// If an emergency stop was triggered by a critical safety condition, force=true is required to override.
func (s *Service) ResumeDispatch(ctx context.Context, actor governance.Actor, force bool) error {
	if !actor.IsHuman() {
		return fmt.Errorf("%w: actor %s:%s is not a verified human owner", ErrUnauthorizedActor, actor.Type, actor.ID)
	}

	ctrl, err := s.govRepo.GetControl(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve runtime control state: %w", err)
	}

	if ctrl != nil && ctrl.IsEmergencyStopped() && !force {
		upperReason := strings.ToUpper(ctrl.Reason)
		if strings.Contains(upperReason, "CRITICAL") ||
			strings.Contains(upperReason, "SECURITY") ||
			strings.Contains(upperReason, "BREACH") ||
			strings.Contains(upperReason, "SAFETY_TRIP") {
			return fmt.Errorf("cannot resume dispatch without force flag: active emergency stop was triggered with critical reason (%s)", ctrl.Reason)
		}
	}

	actorStr := fmt.Sprintf("%s:%s", actor.Type, actor.ID)
	if err := s.govRepo.ResumeDispatch(ctx, actorStr); err != nil {
		return fmt.Errorf("failed to resume dispatch: %w", err)
	}

	now := time.Now().UTC()
	if s.auditRepo != nil {
		govAction := string(governance.GovernanceActionResumed)
		risk := string(governance.RiskTierHigh)
		details, _ := json.Marshal(map[string]any{
			"resumed_by": actorStr,
			"force":      force,
		})
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:               uuid.New().String(),
			ActorType:        actor.Type,
			ActorID:          actor.ID,
			Action:           "RUNTIME_RESUMED",
			ResourceType:     "RUNTIME",
			ResourceID:       "global",
			Outcome:          "RESUMED",
			Details:          details,
			RiskTier:         &risk,
			GovernanceAction: &govAction,
			CreatedAt:        now,
		})
	}

	if s.bus != nil {
		s.bus.Publish("runtime.resumed", map[string]any{
			"actor":     actorStr,
			"timestamp": now,
		})
	}

	return nil
}

// CheckDispatchAllowed checks if dispatch is currently allowed.
func (s *Service) CheckDispatchAllowed(ctx context.Context) (bool, error) {
	ctrl, err := s.govRepo.GetControl(ctx)
	if err != nil {
		return false, err
	}
	return !ctrl.IsEmergencyStopped(), nil
}
