package goal

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
	domainGoal "github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/task"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type CreateGoalInput struct {
	ID                string
	ParentGoalID      *string
	ProjectID         string
	Name              string
	Description       string
	Level             domainGoal.GoalLevel
	Status            domainGoal.GoalStatus
	TargetMetricJSON  string
	CurrentMetricJSON string
	Priority          int
}

type DecomposeTaskInput struct {
	GoalID      string
	ProjectID   string
	CycleID     *string
	AgentID     string
	Title       string
	Description string
	Priority    int
	Rationale   string
	Acceptance  []string
}

// GoalGapAssessment represents the evaluated progress gap of an active Goal.
type GoalGapAssessment struct {
	GoalID       string   `json:"goal_id"`
	GoalName     string   `json:"goal_name"`
	HasGap       bool     `json:"has_gap"`
	GapRatio     float64  `json:"gap_ratio"` // 0.0 (achieved) .. 1.0 (unstarted)
	GapSummary   string   `json:"gap_summary"`
	MetricGaps   []string `json:"metric_gaps,omitempty"`
	SuggestedDomain string `json:"suggested_domain,omitempty"`
}

// Service manages long-term Goal lifecycle, hierarchy, metric evaluation, and task decomposition (AIA-09, Plan Phase 3 §4.3).
type Service struct {
	goalRepo     *repository.GoalRepository
	autonomyRepo *repository.AutonomyRepository
	eventRepo    *repository.EventHistoryRepository
	taskSvc      *taskService.Service
	evoDeps      EvolutionDeps
}

func NewService(
	goalRepo *repository.GoalRepository,
	autonomyRepo *repository.AutonomyRepository,
	eventRepo *repository.EventHistoryRepository,
	taskSvc *taskService.Service,
) *Service {
	return &Service{
		goalRepo:     goalRepo,
		autonomyRepo: autonomyRepo,
		eventRepo:    eventRepo,
		taskSvc:      taskSvc,
	}
}

func (s *Service) Create(ctx context.Context, in CreateGoalInput) (*domainGoal.Goal, error) {
	if in.Name == "" {
		return nil, fmt.Errorf("goal name is required")
	}
	now := time.Now().UTC()
	id := in.ID
	if id == "" {
		id = uuid.New().String()
	}
	lvl := in.Level
	if lvl == "" {
		lvl = domainGoal.LevelOperational
	}
	st := in.Status
	if st == "" {
		st = domainGoal.StatusActive
	}
	prio := in.Priority
	if prio <= 0 {
		prio = 5
	}

	g := &domainGoal.Goal{
		ID:                id,
		ParentGoalID:      in.ParentGoalID,
		Name:              in.Name,
		Description:       in.Description,
		Level:             lvl,
		Status:            st,
		TargetMetricJSON:  in.TargetMetricJSON,
		CurrentMetricJSON: in.CurrentMetricJSON,
		Priority:          prio,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.goalRepo.Create(ctx, g); err != nil {
		return nil, err
	}

	if in.ProjectID != "" {
		_ = s.goalRepo.LinkProject(ctx, g.ID, in.ProjectID, domainGoal.LinkContributesTo)
	}

	s.appendGoalEvent(ctx, g.ID, "GOAL_CREATED", map[string]any{
		"name":   g.Name,
		"level":  g.Level,
		"status": g.Status,
	})

	return g, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domainGoal.Goal, error) {
	return s.goalRepo.Get(ctx, id)
}

func (s *Service) ListAll(ctx context.Context, status domainGoal.GoalStatus, limit int) ([]*domainGoal.Goal, error) {
	return s.goalRepo.ListAll(ctx, status, limit)
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*domainGoal.Goal, error) {
	return s.goalRepo.ListByProject(ctx, projectID)
}

func (s *Service) ListChildren(ctx context.Context, parentGoalID string) ([]*domainGoal.Goal, error) {
	return s.goalRepo.ListChildren(ctx, parentGoalID)
}

func (s *Service) ListRevisions(ctx context.Context, goalID string) ([]*domainGoal.GoalRevision, error) {
	return s.goalRepo.ListRevisions(ctx, goalID)
}

func (s *Service) LinkProject(ctx context.Context, goalID, projectID string, linkType domainGoal.GoalProjectLinkType) error {
	if linkType == "" {
		linkType = domainGoal.LinkContributesTo
	}
	return s.goalRepo.LinkProject(ctx, goalID, projectID, linkType)
}

func (s *Service) TransitionStatus(ctx context.Context, goalID string, status domainGoal.GoalStatus, reason string, decisionID string) error {
	if reason == "" {
		reason = fmt.Sprintf("Status transitioned to %s", status)
	}
	if err := s.goalRepo.UpdateStatus(ctx, goalID, status, reason, decisionID); err != nil {
		return err
	}
	s.appendGoalEvent(ctx, goalID, "GOAL_REVISED", map[string]any{
		"new_status":  status,
		"reason":      reason,
		"decision_id": decisionID,
	})
	return nil
}

func (s *Service) UpdateMetrics(ctx context.Context, goalID string, currentMetricJSON string, reason string, decisionID string) error {
	if err := s.goalRepo.UpdateMetrics(ctx, goalID, currentMetricJSON, reason, decisionID); err != nil {
		return err
	}
	s.appendGoalEvent(ctx, goalID, "GOAL_METRICS_UPDATED", map[string]any{
		"current_metric_json": currentMetricJSON,
		"reason":              reason,
		"decision_id":         decisionID,
	})
	return nil
}

// DecomposeToTask creates a concrete Task linked to a Goal to advance it toward its target metrics.
func (s *Service) DecomposeToTask(ctx context.Context, in DecomposeTaskInput) (*task.Task, error) {
	if s.taskSvc == nil {
		return nil, fmt.Errorf("task service not configured")
	}
	g, err := s.goalRepo.Get(ctx, in.GoalID)
	if err != nil {
		return nil, fmt.Errorf("goal %s not found: %w", in.GoalID, err)
	}
	if g.Status == domainGoal.StatusAbandoned || g.Status == domainGoal.StatusSuperseded {
		return nil, fmt.Errorf("cannot decompose terminal goal %s (status=%s)", g.ID, g.Status)
	}

	agentID := in.AgentID
	if agentID == "" {
		agentID = "coding-agent"
	}
	projID := in.ProjectID
	if projID == "" {
		projID = "default"
	}
	prio := in.Priority
	if prio <= 0 {
		prio = g.Priority * 10
	}

	created, err := s.taskSvc.Create(ctx, taskService.CreateTaskRequest{
		ProjectID:   projID,
		AgentID:     agentID,
		Title:       in.Title,
		Description: in.Description,
		Priority:    prio,
		Acceptance:  in.Acceptance,
	})
	if err != nil {
		return nil, err
	}

	if s.autonomyRepo != nil {
		rationale := in.Rationale
		if rationale == "" {
			rationale = fmt.Sprintf("Decomposed from goal %q (%s)", g.Name, g.ID)
		}
		_ = s.autonomyRepo.LinkGoalTask(ctx, &domainAutonomy.GoalTaskLink{
			GoalID:                 g.ID,
			TaskID:                 string(created.ID),
			CycleID:                in.CycleID,
			DecompositionRationale: rationale,
		})
	}

	s.appendGoalEvent(ctx, g.ID, "GOAL_DECOMPOSED_TO_TASK", map[string]any{
		"task_id":    created.ID,
		"task_title": created.Title,
		"agent_id":   agentID,
		"cycle_id":   in.CycleID,
	})

	return created, nil
}

func (s *Service) ListDecomposedTasks(ctx context.Context, goalID string) ([]*domainAutonomy.GoalTaskLink, error) {
	if s.autonomyRepo == nil {
		return nil, nil
	}
	return s.autonomyRepo.ListGoalTaskLinks(ctx, goalID)
}

// EvaluateGoalGap inspects TargetMetricJSON vs CurrentMetricJSON on an ACTIVE goal
// to determine whether an actionable gap exists (AIA-09 §3.2C Goal Gap Discovery).
func EvaluateGoalGap(g *domainGoal.Goal) GoalGapAssessment {
	out := GoalGapAssessment{
		GoalID:   g.ID,
		GoalName: g.Name,
	}
	if g == nil || g.Status != domainGoal.StatusActive {
		return out
	}

	if g.TargetMetricJSON == "" || g.TargetMetricJSON == "{}" {
		// Active goal with no metrics requires planning and task decomposition
		out.HasGap = true
		out.GapRatio = 1.0
		out.GapSummary = fmt.Sprintf("Goal %q requires planning and execution", g.Name)
		return out
	}

	targetMap := make(map[string]float64)
	var rawTarget any
	if err := json.Unmarshal([]byte(g.TargetMetricJSON), &rawTarget); err == nil && rawTarget != nil {
		extractNumericMetrics(rawTarget, "", targetMap)
	}

	if len(targetMap) == 0 {
		out.HasGap = true
		out.GapRatio = 1.0
		out.GapSummary = fmt.Sprintf("Goal %q requires planning and execution", g.Name)
		return out
	}

	currentMap := make(map[string]float64)
	if g.CurrentMetricJSON != "" {
		var rawCurrent any
		if err := json.Unmarshal([]byte(g.CurrentMetricJSON), &rawCurrent); err == nil && rawCurrent != nil {
			extractNumericMetrics(rawCurrent, "", currentMap)
		}
	}

	var gaps []string
	var maxRatio float64
	for key, targetVal := range targetMap {
		curVal, exists := currentMap[key]
		if !exists || curVal < targetVal {
			ratio := 1.0
			if targetVal > 0 {
				ratio = math.Max(0, math.Min(1.0, (targetVal-curVal)/targetVal))
			}
			if ratio > maxRatio {
				maxRatio = ratio
			}
			gaps = append(gaps, fmt.Sprintf("%s: current=%.2f < target=%.2f (gap %.1f%%)", key, curVal, targetVal, ratio*100))
		}
	}

	if len(gaps) > 0 {
		out.HasGap = true
		out.GapRatio = maxRatio
		out.MetricGaps = gaps
		out.GapSummary = fmt.Sprintf("Goal %q has unmet target metrics (%s)", g.Name, gaps[0])
	}
	return out
}

func extractNumericMetrics(val any, prefix string, out map[string]float64) {
	switch v := val.(type) {
	case float64:
		if prefix != "" {
			out[prefix] = v
		}
	case int:
		if prefix != "" {
			out[prefix] = float64(v)
		}
	case int64:
		if prefix != "" {
			out[prefix] = float64(v)
		}
	case map[string]any:
		for k, inner := range v {
			subKey := k
			if prefix != "" {
				subKey = prefix + "." + k
			}
			extractNumericMetrics(inner, subKey, out)
			// Also add the bare leaf key for easier matching if not conflicting
			if _, exists := out[k]; !exists {
				extractNumericMetrics(inner, k, out)
			}
		}
	}
}

func (s *Service) appendGoalEvent(ctx context.Context, goalID string, eventType string, payload map[string]any) {
	if s.eventRepo == nil {
		return
	}
	b, _ := json.Marshal(payload)
	evtID := uuid.New().String()
	_ = s.eventRepo.Append(ctx, nil, &continuity.EventHistory{
		EventID:       evtID,
		EventType:     eventType,
		AggregateType: "GOAL",
		AggregateID:   goalID,
		CorrelationID: evtID,
		CausationID:   evtID,
		ActorID:       "SYSTEM:goal_service",
		PayloadJSON:   string(b),
		OccurredAt:    time.Now().UTC(),
	})
}
