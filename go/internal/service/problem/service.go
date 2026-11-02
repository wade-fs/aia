package problem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	domainClarify "github.com/wade/aia/internal/domain/clarification"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	clarifyService "github.com/wade/aia/internal/service/clarification"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// ClarificationDispatcher forwards human inquiry questions to the clarification management system (AIA-21, AIA-22 §4.5).
type ClarificationDispatcher interface {
	CreateRequests(ctx context.Context, requests []*domainClarify.ClarificationRequest) error
}

// Service manages the problem analysis and planning lifecycle (AIA-22 §4, §15).
type Service struct {
	problemRepo  domainProblem.Repository
	projectRepo  *repository.ProjectRepository
	clarifySvc   ClarificationDispatcher
	auditRepo    *repository.AuditRepository
}

// NewService creates a new Problem Service instance.
func NewService(
	problemRepo domainProblem.Repository,
	projectRepo *repository.ProjectRepository,
	clarifySvc ClarificationDispatcher,
) *Service {
	return &Service{
		problemRepo: problemRepo,
		projectRepo: projectRepo,
		clarifySvc:  clarifySvc,
	}
}

// SetAuditRepo attaches an audit repository for operational logging.
func (s *Service) SetAuditRepo(repo *repository.AuditRepository) {
	s.auditRepo = repo
}

// CreateProblemInput defines input parameters to initialize a Problem.
type CreateProblemInput struct {
	ID        string
	ProjectID string
	ParentID  string
	GoalID    string
	Title     string
	Objective string
	Scope     string
	Priority  domainProblem.ProblemPriority
}

// CreateProblem initializes a new Problem entity.
func (s *Service) CreateProblem(ctx context.Context, in CreateProblemInput) (*domainProblem.Problem, error) {
	id := in.ID
	if id == "" {
		id = fmt.Sprintf("prob-%s", uuid.New().String()[:8])
	}
	p := &domainProblem.Problem{
		ID:        id,
		ProjectID: in.ProjectID,
		ParentID:  in.ParentID,
		GoalID:    in.GoalID,
		Title:     in.Title,
		Objective: in.Objective,
		Scope:     in.Scope,
		Status:    domainProblem.StatusOpen,
		Priority:  in.Priority,
	}
	if p.Priority == "" {
		p.Priority = domainProblem.PriorityMedium
	}

	if err := s.problemRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to create problem: %w", err)
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           domain.ID("audit-" + uuid.New().String()[:8]),
			ActorType:    "SYSTEM",
			ActorID:      "AIA-22",
			Action:       "CREATE_PROBLEM",
			ResourceType: "PROBLEM",
			ResourceID:   id,
			Outcome:      "SUCCESS",
			CreatedAt:    time.Now().UTC(),
		})
	}
	return p, nil
}

// ApplyAnalysis validates and incorporates agent problem analysis findings into the ProblemModel (AIA-22 §4.4, §4.5).
func (s *Service) ApplyAnalysis(
	ctx context.Context,
	problemID string,
	analysis *domainProblem.AgentProblemAnalysis,
	agentID string,
	executionID string,
) (*domainProblem.ProblemModel, error) {
	if analysis == nil {
		return nil, errors.New("analysis cannot be nil")
	}

	// 1. Validate Epistemic Integrity and Discipline Limits (AIA-22 §7, AIA-21 §5.4)
	if err := domainProblem.ValidateProblemAnalysis(analysis); err != nil {
		return nil, fmt.Errorf("problem analysis validation failed: %w", err)
	}

	prob, err := s.problemRepo.Get(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("failed to load problem %s: %w", problemID, err)
	}

	nextVersion := prob.CurrentModelVersion + 1
	modelID := fmt.Sprintf("%s-model-v%d", prob.ID, nextVersion)

	// 2. Classify ModelItems into respective epistemic categories
	model := &domainProblem.ProblemModel{
		ID:        modelID,
		ProblemID: prob.ID,
		Version:   nextVersion,
		CreatedAt: time.Now().UTC(),
	}

	for _, item := range analysis.ModelItems {
		switch item.Category {
		case domainProblem.CategoryObjective:
			model.Objectives = append(model.Objectives, item)
		case domainProblem.CategoryConstraint:
			model.Constraints = append(model.Constraints, item)
		case domainProblem.CategoryKnownFact:
			model.KnownFacts = append(model.KnownFacts, item)
		case domainProblem.CategoryAssumption:
			model.Assumptions = append(model.Assumptions, item)
		case domainProblem.CategoryUnknown:
			model.Unknowns = append(model.Unknowns, item)
		case domainProblem.CategoryDependency:
			model.Dependencies = append(model.Dependencies, item)
		case domainProblem.CategoryRisk:
			model.Risks = append(model.Risks, item)
		case domainProblem.CategorySuccessCriteria:
			model.SuccessCriteria = append(model.SuccessCriteria, item)
		default:
			// Fallback to assumption if uncategorized
			model.Assumptions = append(model.Assumptions, item)
		}
	}

	// 3. Save ProblemModel (bumps current_model_version on Problem in SQLite tx)
	if err := s.problemRepo.SaveModel(ctx, model); err != nil {
		return nil, fmt.Errorf("failed to persist problem model v%d: %w", nextVersion, err)
	}

	// 4. Record Analysis Logs (Immutable append-only, AIA-22 §6.3)
	// Phase A/B/C Record
	var conclusions []string
	conclusions = append(conclusions, fmt.Sprintf("Goal: %s", analysis.Intent.UserGoal))
	if analysis.Intent.AnalysisOnly {
		conclusions = append(conclusions, "Scope Constraint: Analysis-only requested by user (execution prohibited without explicit approval)")
	}

	var openQuestions []string
	for _, q := range analysis.ClarificationQuestions {
		openQuestions = append(openQuestions, fmt.Sprintf("[%s] %s (Reason: %s)", q.FieldKey, q.Question, q.Reason))
	}

	analysisRec := &domainProblem.AnalysisRecord{
		ID:            fmt.Sprintf("ar-%s-v%d-%s", prob.ID, nextVersion, uuid.New().String()[:6]),
		ProblemID:     prob.ID,
		ModelVersion:  nextVersion,
		AnalysisType:  "INTENT_AND_MODEL",
		InputRefs:     analysis.Context.FoundResources,
		Findings:      analysis.Context.NeedsVerification,
		Conclusions:   conclusions,
		OpenQuestions: openQuestions,
		AgentID:       agentID,
		ExecutionID:   executionID,
		CreatedAt:     time.Now().UTC(),
	}
	_ = s.problemRepo.CreateAnalysisRecord(ctx, analysisRec)

	// If Gaps or Conflicts were discovered, record explicit GAP_CONFLICT record (AIA-22 §4.5)
	if len(analysis.GapsAndConflicts) > 0 {
		var gapFindings []string
		for _, g := range analysis.GapsAndConflicts {
			gapFindings = append(gapFindings, fmt.Sprintf("[%s][%s] %s -> Handling: %s", g.Category, g.Severity, g.Description, g.SuggestedHandling))
		}
		gapRec := &domainProblem.AnalysisRecord{
			ID:           fmt.Sprintf("ar-%s-v%d-gaps-%s", prob.ID, nextVersion, uuid.New().String()[:6]),
			ProblemID:    prob.ID,
			ModelVersion: nextVersion,
			AnalysisType: "GAP_CONFLICT",
			InputRefs:    analysis.Context.ConflictingData,
			Findings:     gapFindings,
			Conclusions:  []string{fmt.Sprintf("Identified %d gaps/conflicts requiring resolution", len(analysis.GapsAndConflicts))},
			AgentID:      agentID,
			ExecutionID:  executionID,
			CreatedAt:    time.Now().UTC(),
		}
		_ = s.problemRepo.CreateAnalysisRecord(ctx, gapRec)
	}

	// 5. Dispatch Clarification Questions if human input is needed (AIA-21, AIA-22 §4.5)
	if len(analysis.ClarificationQuestions) > 0 && s.clarifySvc != nil {
		for _, q := range analysis.ClarificationQuestions {
			if q.ProjectID == "" {
				q.ProjectID = prob.ProjectID
			}
			if q.ID == "" {
				q.ID = fmt.Sprintf("clarify-%s", uuid.New().String()[:8])
			}
		}
		if err := s.clarifySvc.CreateRequests(ctx, analysis.ClarificationQuestions); err != nil {
			return nil, fmt.Errorf("failed to dispatch clarification requests: %w", err)
		}
	}

	// 6. Update Problem state
	prob.CurrentModelVersion = nextVersion
	if prob.Objective == "" || prob.Objective == prob.Title {
		prob.Objective = analysis.Intent.UserGoal
	}
	if len(analysis.Intent.Constraints) > 0 && prob.Scope == "" {
		prob.Scope = fmt.Sprintf("Constraints: %v", analysis.Intent.Constraints)
	}
	prob.Status = domainProblem.StatusAnalyzing
	_ = s.problemRepo.Update(ctx, prob)

	return model, nil
}

// BuildAnalysisPrompt generates a structured prompt for an Agent given problem and project context.
func (s *Service) BuildAnalysisPrompt(ctx context.Context, problemID string, userPrompt string) (string, error) {
	prob, err := s.problemRepo.Get(ctx, problemID)
	if err != nil {
		return "", fmt.Errorf("problem %s not found: %w", problemID, err)
	}

	var proj *domainProject.Project
	if s.projectRepo != nil {
		proj, _ = s.projectRepo.Get(ctx, domain.ID(prob.ProjectID))
	}

	var findings []string
	if prob.Scope != "" {
		findings = append(findings, fmt.Sprintf("Initial Scope: %s", prob.Scope))
	}
	if prob.ParentID != "" {
		findings = append(findings, fmt.Sprintf("Sub-problem of parent problem %s", prob.ParentID))
	}

	return domainProblem.BuildProblemAnalysisPrompt(prob, proj, userPrompt, findings, nil), nil
}

// GetProblem retrieves a problem by ID.
func (s *Service) GetProblem(ctx context.Context, id string) (*domainProblem.Problem, error) {
	return s.problemRepo.Get(ctx, id)
}

// ListProblems retrieves all problems for a project.
func (s *Service) ListProblems(ctx context.Context, projectID string) ([]*domainProblem.Problem, error) {
	return s.problemRepo.ListByProject(ctx, projectID)
}

// GetLatestModel retrieves the current ProblemModel for a problem.
func (s *Service) GetLatestModel(ctx context.Context, problemID string) (*domainProblem.ProblemModel, error) {
	return s.problemRepo.GetLatestModel(ctx, problemID)
}

// ListModels retrieves all versions of ProblemModel for a problem.
func (s *Service) ListModels(ctx context.Context, problemID string) ([]*domainProblem.ProblemModel, error) {
	return s.problemRepo.ListModels(ctx, problemID)
}

// ListAnalysisRecords retrieves all immutable analysis records for a problem.
func (s *Service) ListAnalysisRecords(ctx context.Context, problemID string) ([]*domainProblem.AnalysisRecord, error) {
	return s.problemRepo.ListAnalysisRecords(ctx, problemID)
}

// Ensure interface compatibility
var _ ClarificationDispatcher = (*clarifyService.Service)(nil)
