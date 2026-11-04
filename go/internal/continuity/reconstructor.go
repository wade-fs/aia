package continuity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/goal"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// ProjectReconstruction represents the fully reconstructed contextual state of a project.
type ProjectReconstruction struct {
	ProjectID       string                      `json:"project_id"`
	ProjectName     string                      `json:"project_name"`
	Workspace       string                      `json:"workspace"`
	Status          string                      `json:"status"`
	LatestSnapshot  *continuity.ContextSnapshot `json:"latest_snapshot,omitempty"`
	ActiveGoals     []*goal.Goal                `json:"active_goals"`
	RecentDecisions []*decision.Decision        `json:"recent_decisions"`
	PendingIssues   []string                    `json:"pending_issues"`
	ActiveProblems  []*domainProblem.Problem    `json:"active_problems,omitempty"`
	LastCycle       *continuity.AutonomyCycle   `json:"last_cycle,omitempty"`
	UnfinishedTasks []string                    `json:"unfinished_tasks"`
	ReconstructedAt time.Time                   `json:"reconstructed_at"`
}

// Reconstructor coordinates state recovery across SQLite snapshots, goals, decisions, and event streams.
type Reconstructor struct {
	continuityRepo *repository.ContinuityRepository
	goalRepo       *repository.GoalRepository
	decisionRepo   *repository.DecisionRepository
	eventRepo      *repository.EventHistoryRepository
	projectRepo    *repository.ProjectRepository
	problemRepo    *repository.ProblemRepository
}

func NewReconstructor(
	continuityRepo *repository.ContinuityRepository,
	goalRepo *repository.GoalRepository,
	decisionRepo *repository.DecisionRepository,
	eventRepo *repository.EventHistoryRepository,
	projectRepo *repository.ProjectRepository,
) *Reconstructor {
	return &Reconstructor{
		continuityRepo: continuityRepo,
		goalRepo:       goalRepo,
		decisionRepo:   decisionRepo,
		eventRepo:      eventRepo,
		projectRepo:    projectRepo,
	}
}

func (r *Reconstructor) SetProblemRepository(repo *repository.ProblemRepository) {
	r.problemRepo = repo
}

// ReconstructProjectState restores full project continuity after process termination or restart.
func (r *Reconstructor) ReconstructProjectState(ctx context.Context, projectID string) (*ProjectReconstruction, error) {
	// 1. 取得專案基礎資訊
	p, err := r.projectRepo.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve project for reconstruction: %w", err)
	}

	recon := &ProjectReconstruction{
		ProjectID:       p.ID,
		ProjectName:     p.Name,
		Workspace:       p.Workspace,
		Status:          string(p.Status),
		ReconstructedAt: time.Now().UTC(),
	}

	// 2. 取得最新 Context Snapshot
	snap, err := r.continuityRepo.GetLatestSnapshotByProject(ctx, projectID)
	if err == nil && snap != nil {
		recon.LatestSnapshot = snap

		// 解析未完成任務
		var tasks []string
		if err := json.Unmarshal([]byte(snap.ActiveTasksJSON), &tasks); err == nil {
			recon.UnfinishedTasks = tasks
		}

		// 解析待決問題
		var issues []string
		if err := json.Unmarshal([]byte(snap.PendingIssuesJSON), &issues); err == nil {
			recon.PendingIssues = issues
		}
	} else if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("error loading latest snapshot: %w", err)
	}

	// 3. 取得關聯目標清單
	goals, err := r.goalRepo.ListByProject(ctx, projectID)
	if err == nil {
		recon.ActiveGoals = goals
	} else if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("error loading goals: %w", err)
	}

	// 4. 取得近期決策歷程 (包含 NO_ACTION)
	decisions, err := r.decisionRepo.ListByProject(ctx, projectID, 10)
	if err == nil {
		recon.RecentDecisions = decisions
	} else if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("error loading recent decisions: %w", err)
	}

	// 5. 若有決策關聯 cycle，取得最新自主循環資訊
	if len(decisions) > 0 && decisions[0].CycleID != nil {
		cycle, err := r.continuityRepo.GetCycle(ctx, *decisions[0].CycleID)
		if err == nil {
			recon.LastCycle = cycle
		}
	}

	// 6. 取得關聯之活躍複雜問題 (AIA-22 §10.3)
	if r.problemRepo != nil {
		if probs, err := r.problemRepo.ListByProject(ctx, projectID); err == nil {
			for _, prob := range probs {
				if prob.Status != domainProblem.StatusResolved && prob.Status != domainProblem.StatusClosed {
					recon.ActiveProblems = append(recon.ActiveProblems, prob)
				}
			}
		}
	}

	return recon, nil
}

// CapturePreRestartSnapshot records a safe state snapshot before scheduled restart or graceful stop.
func (r *Reconstructor) CapturePreRestartSnapshot(
	ctx context.Context,
	projectID string,
	snapshotType continuity.SnapshotType,
	activeTasks []string,
	pendingIssues []string,
	safeWorkspaceMeta map[string]string,
) (*continuity.ContextSnapshot, error) {
	tasksJSON, _ := json.Marshal(activeTasks)
	issuesJSON, _ := json.Marshal(pendingIssues)
	wsJSON, _ := json.Marshal(safeWorkspaceMeta)

	snap := &continuity.ContextSnapshot{
		ID:                 uuid.New().String(),
		Scope:              continuity.SnapshotScopeProject,
		ProjectID:          projectID,
		SnapshotType:       snapshotType,
		WorkspaceStateJSON: string(wsJSON),
		ActiveTasksJSON:    string(tasksJSON),
		RecentDecisionsJSON: "[]",
		PendingIssuesJSON:  string(issuesJSON),
		CreatedAt:          time.Now().UTC(),
	}

	if err := r.continuityRepo.CreateSnapshot(ctx, snap); err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	return snap, nil
}
