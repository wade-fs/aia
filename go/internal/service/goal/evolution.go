package goal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/autonomy/clock"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/decision"
	domainGoal "github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/knowledge"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	approvalService "github.com/wade/aia/internal/service/approval"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type EvolutionDeps struct {
	LearningRepo  *repository.LearningRepository
	KnowledgeRepo *repository.KnowledgeRepository
	DecisionRepo  *repository.DecisionRepository
	AuditRepo     *repository.AuditRepository
	ApprovalSvc   *approvalService.Service
}

func (s *Service) SetEvolutionDeps(deps EvolutionDeps) {
	s.evoDeps = deps
}

type EvolutionInput struct {
	GoalID                   string
	CycleID                  *string
	ProjectID                *string
	Mode                     domainLearning.GoalEvolutionMode
	ActorType                string // "AGENT" or "HUMAN"
	ActorID                  string
	Rationale                string
	ProposedName             string
	ProposedDescription      string
	ProposedTargetMetricJSON string
	ProposedSubGoalName      string
	ProposedSubGoalMetric    string
	InvalidatedClaimIDs      []string
	ResourceExhausted        bool
	ResourceConstraintNote   string
}

// ProposeOrApplyEvolution enforces AIA-09 §25 Goal Evolution 3-Way Taxonomy
// (PERSISTENCE | OPTIMIZATION | REVISION) and the Anti-Goalpost-Moving Guard:
//  1. Agents CANNOT revise a STRATEGIC goal purely because performance fell short (e.g., turning
//     "Maximize Total Assets" into "Minimize Volatility" after investment losses).
//  2. Even when premise claims are proven refuted/conflicted in KnowledgeRepository,
//     an AGENT only receives a PENDING_APPROVAL proposal — never direct write access to STRATEGIC goals.
func (s *Service) ProposeOrApplyEvolution(ctx context.Context, in EvolutionInput) (*domainLearning.GoalEvolutionProposal, error) {
	g, err := s.goalRepo.Get(ctx, in.GoalID)
	if err != nil {
		return nil, fmt.Errorf("goal not found: %w", err)
	}

	actorType := strings.ToUpper(strings.TrimSpace(in.ActorType))
	if actorType == "" {
		actorType = "AGENT"
	}
	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		actorID = "autonomous-agent"
	}
	now := clock.Now(ctx)

	claimIDsJSON, _ := json.Marshal(in.InvalidatedClaimIDs)
	prop := &domainLearning.GoalEvolutionProposal{
		ID:                       uuid.New().String(),
		GoalID:                   g.ID,
		CycleID:                  in.CycleID,
		EvolutionMode:            in.Mode,
		ActorType:                actorType,
		ActorID:                  actorID,
		Rationale:                in.Rationale,
		PreviousTargetMetricJSON: g.TargetMetricJSON,
		ProposedTargetMetricJSON: in.ProposedTargetMetricJSON,
		InvalidatedClaimIDsJSON:  string(claimIDsJSON),
		ResourceConstraintNote:   in.ResourceConstraintNote,
		Status:                   domainLearning.EvoApplied,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if prop.ProposedTargetMetricJSON == "" {
		prop.ProposedTargetMetricJSON = g.TargetMetricJSON
	}

	switch in.Mode {
	case domainLearning.EvolutionPersistence:
		// Goal definition & TargetMetricJSON are strictly preserved.
		prop.ProposedTargetMetricJSON = g.TargetMetricJSON
		if in.ResourceExhausted && g.Status == domainGoal.StatusActive {
			_ = s.TransitionStatus(ctx, g.ID, domainGoal.StatusPaused,
				fmt.Sprintf("Goal Persistence resource pause: %s", in.ResourceConstraintNote), "")
		}
		if s.evoDeps.LearningRepo != nil {
			_ = s.evoDeps.LearningRepo.CreateGoalEvolutionProposal(ctx, prop)
		}
		s.appendGoalEvent(ctx, g.ID, "GOAL_EVOLUTION_PERSISTENCE", map[string]any{
			"proposal_id":        prop.ID,
			"resource_exhausted": in.ResourceExhausted,
		})
		return prop, nil

	case domainLearning.EvolutionOptimization:
		// Top-level STRATEGIC goal Name and TargetMetricJSON MUST remain locked!
		if g.Level == domainGoal.LevelStrategic &&
			((in.ProposedTargetMetricJSON != "" && in.ProposedTargetMetricJSON != g.TargetMetricJSON) ||
				(in.ProposedName != "" && in.ProposedName != g.Name)) {
			prop.Status = domainLearning.EvoRejectedGoalpostMoving
			if s.evoDeps.LearningRepo != nil {
				_ = s.evoDeps.LearningRepo.CreateGoalEvolutionProposal(ctx, prop)
			}
			s.recordGoalpostAudit(ctx, actorID, g.ID, "Goal Optimization attempted to mutate STRATEGIC target metric or name")
			return prop, domainLearning.ErrGoalpostMovingBlocked
		}
		prop.ProposedTargetMetricJSON = g.TargetMetricJSON

		// Allow spawning an OPERATIONAL sub-goal to optimize strategy execution without changing the parent goal
		if strings.TrimSpace(in.ProposedSubGoalName) != "" {
			parentID := g.ID
			_, _ = s.Create(ctx, CreateGoalInput{
				ParentGoalID:     &parentID,
				Name:             in.ProposedSubGoalName,
				Description:      in.Rationale,
				Level:            domainGoal.LevelOperational,
				Status:           domainGoal.StatusActive,
				TargetMetricJSON: in.ProposedSubGoalMetric,
				Priority:         g.Priority,
			})
		}
		if s.evoDeps.LearningRepo != nil {
			_ = s.evoDeps.LearningRepo.CreateGoalEvolutionProposal(ctx, prop)
		}
		s.appendGoalEvent(ctx, g.ID, "GOAL_EVOLUTION_OPTIMIZATION", map[string]any{
			"proposal_id": prop.ID,
			"sub_goal":    in.ProposedSubGoalName,
		})
		return prop, nil

	case domainLearning.EvolutionRevision:
		if g.Level == domainGoal.LevelStrategic && actorType != "HUMAN" && actorType != "USER" {
			// Anti-Goalpost-Moving Check #1: Must prove at least one underlying premise claim is invalidated!
			if !s.hasVerifiedInvalidatedPremise(ctx, in.InvalidatedClaimIDs) {
				prop.Status = domainLearning.EvoRejectedGoalpostMoving
				if s.evoDeps.LearningRepo != nil {
					_ = s.evoDeps.LearningRepo.CreateGoalEvolutionProposal(ctx, prop)
				}
				s.recordGoalpostAudit(ctx, actorID, g.ID,
					"Blocked STRATEGIC goal revision lacking invalidated premise evidence (Anti-Goalpost-Moving)")
				s.appendGoalEvent(ctx, g.ID, "GOAL_REVISION_BLOCKED_GOALPOST_MOVING", map[string]any{
					"proposal_id": prop.ID,
					"actor_id":    actorID,
					"rationale":   in.Rationale,
				})
				return prop, domainLearning.ErrGoalpostMovingBlocked
			}

			// Anti-Goalpost-Moving Check #2: Even with valid invalidated premise evidence, AGENT has zero direct write access!
			prop.Status = domainLearning.EvoPendingApproval
			if s.evoDeps.ApprovalSvc != nil && s.evoDeps.DecisionRepo != nil {
				dec := &decision.Decision{
					ID:               uuid.New().String(),
					CycleID:          in.CycleID,
					ProjectID:        in.ProjectID,
					ActionType:       decision.ActionAskHuman,
					IntentSummary:    fmt.Sprintf("Strategic Goal Revision Proposal for %q", g.Name),
					Justification:    in.Rationale,
					GovernanceResult: decision.GovernanceRequiredApproval,
					CreatedAt:        now,
				}
				if dErr := s.evoDeps.DecisionRepo.Create(ctx, dec, nil); dErr == nil {
					detailJSON, _ := json.Marshal(map[string]any{
						"goal_id":                g.ID,
						"previous_name":          g.Name,
						"proposed_name":          in.ProposedName,
						"previous_target_metric": g.TargetMetricJSON,
						"proposed_target_metric": in.ProposedTargetMetricJSON,
						"invalidated_claim_ids":  in.InvalidatedClaimIDs,
						"rationale":              in.Rationale,
					})
					appr, aErr := s.evoDeps.ApprovalSvc.RequestApproval(ctx, approvalService.RequestApprovalInput{
						RequesterEntity:     fmt.Sprintf("%s:%s", actorType, actorID),
						OperationType:       "STRATEGIC_GOAL_REVISION",
						OperationDetailJSON: string(detailJSON),
						RiskTier:            governance.RiskTierHigh,
						DecisionID:          &dec.ID,
					})
					if aErr == nil && appr != nil {
						prop.ApprovalID = &appr.ID
					}
				}
			}
			if s.evoDeps.LearningRepo != nil {
				if err := s.evoDeps.LearningRepo.CreateGoalEvolutionProposal(ctx, prop); err != nil {
					return nil, err
				}
			}
			s.appendGoalEvent(ctx, g.ID, "GOAL_REVISION_PROPOSED", map[string]any{
				"proposal_id": prop.ID,
				"approval_id": prop.ApprovalID,
			})
			return prop, nil
		}

		// Human-initiated revision OR non-strategic (OPERATIONAL/TACTICAL) goal revision
		rev, err := s.goalRepo.ReviseDefinition(
			ctx, g.ID, in.ProposedName, in.ProposedDescription, in.ProposedTargetMetricJSON, 0, in.Rationale, "",
		)
		if err != nil {
			return nil, err
		}
		prop.Status = domainLearning.EvoApplied
		prop.ResultingRevisionID = &rev.ID
		if s.evoDeps.LearningRepo != nil {
			_ = s.evoDeps.LearningRepo.CreateGoalEvolutionProposal(ctx, prop)
		}
		s.appendGoalEvent(ctx, g.ID, "GOAL_REVISION_APPLIED", map[string]any{
			"proposal_id": prop.ID,
			"revision_id": rev.ID,
		})
		return prop, nil

	default:
		return nil, fmt.Errorf("unsupported goal evolution mode: %s", in.Mode)
	}
}

// ApplyApprovedEvolution commits a PENDING_APPROVAL Strategic Goal Revision ONLY after the linked human Approval is APPROVED.
func (s *Service) ApplyApprovedEvolution(ctx context.Context, proposalID string, newName string, newDescription string) (*domainGoal.GoalRevision, error) {
	if s.evoDeps.LearningRepo == nil {
		return nil, fmt.Errorf("learning repository is not configured")
	}
	prop, err := s.evoDeps.LearningRepo.GetGoalEvolutionProposal(ctx, proposalID)
	if err != nil {
		return nil, err
	}
	if prop.Status != domainLearning.EvoPendingApproval {
		return nil, fmt.Errorf("proposal %s is in status %s, expected PENDING_APPROVAL", proposalID, prop.Status)
	}
	if prop.ApprovalID == nil || *prop.ApprovalID == "" || s.evoDeps.ApprovalSvc == nil {
		return nil, domainLearning.ErrStrategicGoalDirectWriteDenied
	}
	appr, err := s.evoDeps.ApprovalSvc.Get(ctx, *prop.ApprovalID)
	if err != nil || appr == nil || appr.Status != governance.ApprovalApproved {
		return nil, fmt.Errorf("%w: linked approval %s is not APPROVED", domainLearning.ErrStrategicGoalDirectWriteDenied, *prop.ApprovalID)
	}

	decID := ""
	if appr.DecisionID != nil {
		decID = *appr.DecisionID
	}
	rev, err := s.goalRepo.ReviseDefinition(
		ctx, prop.GoalID, newName, newDescription, prop.ProposedTargetMetricJSON, 0, prop.Rationale, decID,
	)
	if err != nil {
		return nil, err
	}

	prop.Status = domainLearning.EvoApproved
	prop.ResultingRevisionID = &rev.ID
	_ = s.evoDeps.LearningRepo.UpdateGoalEvolutionProposal(ctx, prop)

	s.appendGoalEvent(ctx, prop.GoalID, "STRATEGIC_GOAL_REVISION_APPROVED_AND_APPLIED", map[string]any{
		"proposal_id": prop.ID,
		"approval_id": *prop.ApprovalID,
		"revision_id": rev.ID,
	})
	return rev, nil
}

func (s *Service) hasVerifiedInvalidatedPremise(ctx context.Context, claimIDs []string) bool {
	if len(claimIDs) == 0 || s.evoDeps.KnowledgeRepo == nil {
		return false
	}
	for _, cid := range claimIDs {
		cid = strings.TrimSpace(cid)
		if cid == "" {
			continue
		}
		claim, err := s.evoDeps.KnowledgeRepo.GetClaim(ctx, cid)
		if err != nil || claim == nil {
			continue
		}
		if claim.AssertionStatus == knowledge.AssertionRejected ||
			claim.AssertionStatus == knowledge.AssertionConflicted ||
			claim.AssertionStatus == knowledge.AssertionSuperseded {
			return true
		}
	}
	return false
}

func (s *Service) recordGoalpostAudit(ctx context.Context, actorID string, goalID string, reason string) {
	if s.evoDeps.AuditRepo == nil {
		return
	}
	riskStr := string(governance.RiskTierHigh)
	govAct := string(governance.GovernanceActionBlocked)
	details, _ := json.Marshal(map[string]any{
		"goal_id": goalID,
		"reason":  reason,
	})
	_ = s.evoDeps.AuditRepo.Append(ctx, &repository.AuditLogEntry{
		ID:               domain.ID(uuid.New().String()),
		ActorType:        "AGENT",
		ActorID:          actorID,
		Action:           "ANTI_GOALPOST_MOVING_BLOCKED",
		ResourceType:     "GOAL",
		ResourceID:       goalID,
		Outcome:          govAct,
		Details:          details,
		RiskTier:         &riskStr,
		GovernanceAction: &govAct,
		CreatedAt:        clock.Now(ctx),
	})
}
