package problem

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	domainClarify "github.com/wade/aia/internal/domain/clarification"
	domainPlan "github.com/wade/aia/internal/domain/plan"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	domainTask "github.com/wade/aia/internal/domain/task"
	clarifyService "github.com/wade/aia/internal/service/clarification"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// ClarificationDispatcher forwards human inquiry questions to the clarification management system (AIA-21, AIA-22 §4.5).
type ClarificationDispatcher interface {
	CreateRequests(ctx context.Context, requests []*domainClarify.ClarificationRequest) error
}

// Service manages the problem analysis and planning lifecycle (AIA-22 §4, §15).
type Service struct {
	problemRepo  domainProblem.Repository
	planRepo     domainPlan.Repository
	projectRepo  *repository.ProjectRepository
	taskSvc      *taskService.Service
	clarifySvc   ClarificationDispatcher
	auditRepo    *repository.AuditRepository
	decompConfig domainProblem.DecompositionConfig
}

// NewService creates a new Problem Service instance.
func NewService(
	problemRepo domainProblem.Repository,
	planRepo domainPlan.Repository,
	projectRepo *repository.ProjectRepository,
	clarifySvc ClarificationDispatcher,
) *Service {
	return &Service{
		problemRepo:  problemRepo,
		planRepo:     planRepo,
		projectRepo:  projectRepo,
		clarifySvc:   clarifySvc,
		decompConfig: domainProblem.DefaultDecompositionConfig(),
	}
}

// SetDecompositionConfig adjusts maximum depth and subproblem limits (AIA-22 §5.2).
func (s *Service) SetDecompositionConfig(cfg domainProblem.DecompositionConfig) {
	s.decompConfig = cfg
}

// SetTaskService attaches the TaskService for work item materialization.
func (s *Service) SetTaskService(ts *taskService.Service) {
	s.taskSvc = ts
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
			model.Assumptions = append(model.Assumptions, item)
		}
	}

	// 3. Save ProblemModel (bumps current_model_version on Problem in SQLite tx)
	if err := s.problemRepo.SaveModel(ctx, model); err != nil {
		return nil, fmt.Errorf("failed to persist problem model v%d: %w", nextVersion, err)
	}

	// 4. Record Analysis Logs (Immutable append-only, AIA-22 §6.3)
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

// ExploreSolutions processes candidate solution alternatives and records trade-off rationale (AIA-22 §4.6).
func (s *Service) ExploreSolutions(
	ctx context.Context,
	problemID string,
	exp *domainPlan.SolutionExplorationResult,
	agentID string,
) (*domainProblem.AnalysisRecord, error) {
	if err := domainPlan.ValidateSolutionExploration(exp); err != nil {
		return nil, fmt.Errorf("solution exploration validation failed: %w", err)
	}

	prob, err := s.problemRepo.Get(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("problem %s not found: %w", problemID, err)
	}

	var findings []string
	var selectedAlt domainPlan.SolutionAlternative
	for _, alt := range exp.Alternatives {
		isSel := alt.ID == exp.SelectedAlternativeID
		if isSel {
			selectedAlt = alt
		}
		findings = append(findings, fmt.Sprintf(
			"[%s] %s | Feasibility: %s | Reversibility: %s | Cost: %s | Duration: %s | Selected: %t",
			alt.ID, alt.Title, alt.TechnicalFeasibility, alt.Reversibility, alt.EstimatedCost, alt.EstimatedDuration, isSel,
		))
	}

	conclusions := []string{
		fmt.Sprintf("Selected Alternative: %s (%s)", selectedAlt.ID, selectedAlt.Title),
		fmt.Sprintf("Rationale: %s", exp.OverallRationale),
	}
	if exp.RequiresHumanApproval {
		conclusions = append(conclusions, "Governance: Alternative involves irreversible or high-risk actions; requires explicit human approval before execution (AIA-17, AIA-22 §9.2)")
	}

	rec := &domainProblem.AnalysisRecord{
		ID:           fmt.Sprintf("ar-%s-sol-%s", prob.ID, uuid.New().String()[:6]),
		ProblemID:    prob.ID,
		ModelVersion: prob.CurrentModelVersion,
		AnalysisType: "SOLUTION_EXPLORATION",
		Findings:     findings,
		Conclusions:  conclusions,
		AgentID:      agentID,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.problemRepo.CreateAnalysisRecord(ctx, rec); err != nil {
		return nil, fmt.Errorf("failed to record solution exploration: %w", err)
	}
	return rec, nil
}

// GeneratePlanInput defines the specification for generating a new Plan version (AIA-22 §4.7, §6.4).
type GeneratePlanInput struct {
	ProblemID             string
	Objective             string
	Strategy              string
	WorkItems             []domainPlan.WorkItemDef
	Dependencies          []string
	AcceptanceCriteria    []string
	SupersedesPlanID      string
	ChangeReason          string
	RequiresHumanApproval bool
	ApprovedBy            string
}

// GeneratePlan produces a versioned Plan, checks DAG dependency consistency, and records version changes (AIA-22 §4.7, §6.4, §6.5).
func (s *Service) GeneratePlan(ctx context.Context, in GeneratePlanInput) (*domainPlan.Plan, error) {
	if s.planRepo == nil {
		return nil, errors.New("plan repository is not configured")
	}

	prob, err := s.problemRepo.Get(ctx, in.ProblemID)
	if err != nil {
		return nil, fmt.Errorf("failed to load problem %s: %w", in.ProblemID, err)
	}

	var version int
	var prevPlan *domainPlan.Plan
	if in.SupersedesPlanID != "" {
		prevPlan, err = s.planRepo.Get(ctx, in.SupersedesPlanID)
		if err != nil {
			return nil, fmt.Errorf("superseded plan %s not found: %w", in.SupersedesPlanID, err)
		}
		if in.ChangeReason == "" {
			return nil, errors.New("change reason is required when superseding an existing plan")
		}
		version = prevPlan.Version + 1
	} else {
		version = prob.CurrentPlanVersion + 1
	}

	planID := fmt.Sprintf("%s-plan-v%d", prob.ID, version)

	initialStatus := domainPlan.StatusProposed
	if in.RequiresHumanApproval {
		initialStatus = domainPlan.StatusProposed
	}

	p := &domainPlan.Plan{
		ID:                 planID,
		ProblemID:          prob.ID,
		Version:            version,
		Objective:          in.Objective,
		Strategy:           in.Strategy,
		Status:             initialStatus,
		WorkItems:          in.WorkItems,
		Dependencies:       in.Dependencies,
		AcceptanceCriteria: in.AcceptanceCriteria,
		SupersedesPlanID:   in.SupersedesPlanID,
	}

	// Validate Plan and WorkItem DAG
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}

	// Persist Plan (atomic bump of current_plan_version on Problem)
	if err := s.planRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to persist plan: %w", err)
	}

	// If superseding, mark previous plan as SUPERSEDED and log change record (AIA-22 §6.5)
	if in.SupersedesPlanID != "" {
		_ = s.planRepo.UpdateStatus(ctx, in.SupersedesPlanID, domainPlan.StatusSuperseded)

		var affectedWIs []string
		for _, wi := range in.WorkItems {
			affectedWIs = append(affectedWIs, wi.ID)
		}
		changeRec := &domainPlan.PlanChangeRecord{
			ID:                    fmt.Sprintf("change-%s-v%d", prob.ID, version),
			PlanID:                p.ID,
			FromVersion:           prevPlan.Version,
			ToVersion:             version,
			Reason:                in.ChangeReason,
			AffectedWorkItems:     affectedWIs,
			RequiresHumanApproval: in.RequiresHumanApproval,
			ApprovedBy:            in.ApprovedBy,
			CreatedAt:             time.Now().UTC(),
		}
		_ = s.planRepo.RecordChange(ctx, changeRec)
	}

	// Update Problem status to PLANNED
	prob.Status = domainProblem.StatusPlanned
	prob.CurrentPlanVersion = version
	_ = s.problemRepo.Update(ctx, prob)

	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           domain.ID("audit-" + uuid.New().String()[:8]),
			ActorType:    "SYSTEM",
			ActorID:      "AIA-22",
			Action:       "GENERATE_PLAN",
			ResourceType: "PLAN",
			ResourceID:   p.ID,
			Outcome:      "SUCCESS",
			CreatedAt:    time.Now().UTC(),
		})
	}

	return p, nil
}

// ApprovePlan transitions a proposed plan to APPROVED status (AIA-22 §9.2).
func (s *Service) ApprovePlan(ctx context.Context, planID string, approver string) (*domainPlan.Plan, error) {
	if s.planRepo == nil {
		return nil, errors.New("plan repository is not configured")
	}

	p, err := s.planRepo.Get(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}

	if err := s.planRepo.UpdateStatus(ctx, planID, domainPlan.StatusApproved); err != nil {
		return nil, fmt.Errorf("failed to approve plan: %w", err)
	}
	p.Status = domainPlan.StatusApproved

	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           domain.ID("audit-" + uuid.New().String()[:8]),
			ActorType:    "HUMAN",
			ActorID:      approver,
			Action:       "APPROVE_PLAN",
			ResourceType: "PLAN",
			ResourceID:   planID,
			Outcome:      "SUCCESS",
			CreatedAt:    time.Now().UTC(),
		})
	}
	return p, nil
}

// MaterializeWorkItemToTask converts a single Plan WorkItem into an executable AIA Task (AIA-22 §4.7).
func (s *Service) MaterializeWorkItemToTask(ctx context.Context, planID string, workItemID string) (*domainTask.Task, error) {
	if s.planRepo == nil {
		return nil, errors.New("plan repository is not configured")
	}
	if s.taskSvc == nil {
		return nil, errors.New("task service is not configured")
	}

	plan, err := s.planRepo.Get(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("plan %s not found: %w", planID, err)
	}

	prob, err := s.problemRepo.Get(ctx, plan.ProblemID)
	if err != nil {
		return nil, fmt.Errorf("problem %s not found: %w", plan.ProblemID, err)
	}

	var targetItem *domainPlan.WorkItemDef
	var targetIdx int
	for i, wi := range plan.WorkItems {
		if wi.ID == workItemID {
			targetItem = &plan.WorkItems[i]
			targetIdx = i
			break
		}
	}
	if targetItem == nil {
		return nil, fmt.Errorf("work item %s not found in plan %s", workItemID, planID)
	}

	// If already materialized, return existing task
	if targetItem.LinkedTaskID != "" {
		return s.taskSvc.Get(ctx, targetItem.LinkedTaskID)
	}

	// Map dependencies to their materialized task IDs
	var depTaskIDs []domain.ID
	for _, depItemID := range targetItem.Dependencies {
		for _, wi := range plan.WorkItems {
			if wi.ID == depItemID && wi.LinkedTaskID != "" {
				depTaskIDs = append(depTaskIDs, domain.ID(wi.LinkedTaskID))
			}
		}
	}

	var descBuilder strings.Builder
	if targetItem.Description != "" {
		descBuilder.WriteString(targetItem.Description)
		descBuilder.WriteString("\n\n")
	}
	descBuilder.WriteString("### Acceptance Criteria (AIA-22):\n")
	for _, ac := range targetItem.AcceptanceCriteria {
		descBuilder.WriteString(fmt.Sprintf("- [ ] %s\n", ac))
	}

	agentID := targetItem.AssignedAgent
	if agentID == "" {
		if prob.ProjectID != "" && s.projectRepo != nil {
			if proj, err := s.projectRepo.Get(ctx, domain.ID(prob.ProjectID)); err == nil && proj != nil {
				switch proj.DomainType {
				case domainProject.DomainWriting:
					agentID = "writing-agent"
				case domainProject.DomainCoding:
					agentID = "coding-agent"
				case domainProject.DomainInvestment:
					agentID = "investment-agent"
				default:
					agentID = "personal-agent"
				}
			}
		}
		if agentID == "" {
			agentID = "personal-agent"
		}
	}

	taskReq := taskService.CreateTaskRequest{
		ProjectID:    domain.ID(prob.ProjectID),
		AgentID:      domain.ID(agentID),
		Title:        targetItem.Title,
		Description:  descBuilder.String(),
		Dependencies: depTaskIDs,
		Priority:     50,
	}

	t, err := s.taskSvc.Create(ctx, taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create task for work item %s: %w", workItemID, err)
	}

	// Update work item with task link
	plan.WorkItems[targetIdx].LinkedTaskID = t.ID
	if len(depTaskIDs) > 0 {
		plan.WorkItems[targetIdx].Status = domainPlan.WorkItemPending
	} else {
		plan.WorkItems[targetIdx].Status = domainPlan.WorkItemReady
	}

	if err := s.planRepo.UpdateWorkItems(ctx, plan.ID, plan.WorkItems); err != nil {
		return nil, fmt.Errorf("failed to update plan work item status: %w", err)
	}

	return t, nil
}

// MaterializeAllWorkItems materializes all work items in a plan in dependency order (AIA-22 §4.7).
func (s *Service) MaterializeAllWorkItems(ctx context.Context, planID string) ([]*domainTask.Task, error) {
	plan, err := s.planRepo.Get(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("plan %s not found: %w", planID, err)
	}

	var createdTasks []*domainTask.Task
	for _, wi := range plan.WorkItems {
		t, err := s.MaterializeWorkItemToTask(ctx, planID, wi.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to materialize work item %s: %w", wi.ID, err)
		}
		createdTasks = append(createdTasks, t)
	}

	// Update plan status to ACTIVE
	_ = s.planRepo.UpdateStatus(ctx, planID, domainPlan.StatusActive)

	// Update problem status to EXECUTING
	prob, _ := s.problemRepo.Get(ctx, plan.ProblemID)
	if prob != nil {
		prob.Status = domainProblem.StatusExecuting
		_ = s.problemRepo.Update(ctx, prob)
	}

	return createdTasks, nil
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

// GetPlan retrieves a plan by ID.
func (s *Service) GetPlan(ctx context.Context, id string) (*domainPlan.Plan, error) {
	if s.planRepo == nil {
		return nil, errors.New("plan repository is not configured")
	}
	return s.planRepo.Get(ctx, id)
}

// GetLatestPlan retrieves the latest plan for a problem.
func (s *Service) GetLatestPlan(ctx context.Context, problemID string) (*domainPlan.Plan, error) {
	if s.planRepo == nil {
		return nil, errors.New("plan repository is not configured")
	}
	return s.planRepo.GetLatestByProblem(ctx, problemID)
}

// ListPlans retrieves all plans for a problem in ascending version order.
func (s *Service) ListPlans(ctx context.Context, problemID string) ([]*domainPlan.Plan, error) {
	if s.planRepo == nil {
		return nil, errors.New("plan repository is not configured")
	}
	return s.planRepo.ListByProblem(ctx, problemID)
}

// ListPlanChanges retrieves the formal change history for a plan.
func (s *Service) ListPlanChanges(ctx context.Context, planID string) ([]*domainPlan.PlanChangeRecord, error) {
	if s.planRepo == nil {
		return nil, errors.New("plan repository is not configured")
	}
	return s.planRepo.ListChanges(ctx, planID)
}

// GetProblemDepth calculates the nesting depth of a problem in the hierarchy (root = 1) (AIA-22 §5.1, §5.2).
func (s *Service) GetProblemDepth(ctx context.Context, problemID string) (int, error) {
	depth := 1
	currentID := problemID
	visited := make(map[string]bool)

	for {
		if visited[currentID] {
			return 0, errors.New("cycle detected in problem hierarchy")
		}
		visited[currentID] = true

		p, err := s.problemRepo.Get(ctx, currentID)
		if err != nil {
			return 0, err
		}
		if p.ParentID == "" {
			break
		}
		depth++
		currentID = p.ParentID
	}
	return depth, nil
}

// GetProblemTree recursively retrieves the complete decomposition tree starting from rootID (AIA-22 §5.1).
func (s *Service) GetProblemTree(ctx context.Context, rootID string) (*domainProblem.ProblemTreeNode, error) {
	p, err := s.problemRepo.Get(ctx, rootID)
	if err != nil {
		return nil, err
	}
	depth, err := s.GetProblemDepth(ctx, rootID)
	if err != nil {
		return nil, err
	}
	node := &domainProblem.ProblemTreeNode{
		Problem: p,
		Depth:   depth,
	}
	children, err := s.problemRepo.ListChildren(ctx, rootID)
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		childNode, err := s.GetProblemTree(ctx, child.ID)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, childNode)
	}
	return node, nil
}

func (s *Service) countSubTreeProblems(ctx context.Context, rootID string) (int, error) {
	children, err := s.problemRepo.ListChildren(ctx, rootID)
	if err != nil {
		return 0, err
	}
	count := len(children)
	for _, ch := range children {
		subCount, err := s.countSubTreeProblems(ctx, ch.ID)
		if err != nil {
			return 0, err
		}
		count += subCount
	}
	return count, nil
}

// DecomposeProblem decomposes a problem into child sub-problems with depth and budget enforcement (AIA-22 §5.1, §5.2).
func (s *Service) DecomposeProblem(ctx context.Context, parentID string, subProblems []CreateProblemInput) ([]*domainProblem.Problem, error) {
	parent, err := s.problemRepo.Get(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("parent problem not found: %w", err)
	}
	parentDepth, err := s.GetProblemDepth(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if parentDepth+1 > s.decompConfig.MaxDepth {
		return nil, fmt.Errorf("%w: depth %d exceeds max %d", domainProblem.ErrMaxDepthExceeded, parentDepth+1, s.decompConfig.MaxDepth)
	}

	// Trace root problem of this tree
	rootID := parentID
	cur := parent
	for cur.ParentID != "" {
		p, err := s.problemRepo.Get(ctx, cur.ParentID)
		if err != nil {
			break
		}
		rootID = p.ID
		cur = p
	}

	totalExisting, err := s.countSubTreeProblems(ctx, rootID)
	if err != nil {
		return nil, err
	}
	if totalExisting+len(subProblems) > s.decompConfig.MaxSubProblems {
		return nil, fmt.Errorf("%w: total %d would exceed max limit %d", domainProblem.ErrMaxSubProblemsExceeded, totalExisting+len(subProblems), s.decompConfig.MaxSubProblems)
	}

	var created []*domainProblem.Problem
	for _, in := range subProblems {
		in.ParentID = parentID
		if in.ProjectID == "" {
			in.ProjectID = parent.ProjectID
		}
		p, err := s.CreateProblem(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("failed to create subproblem: %w", err)
		}
		created = append(created, p)
	}
	return created, nil
}

// TriggerReassessment initiates dynamic plan re-evaluation under new evidence or failure triggers (AIA-22 §8.1, §8.2, §8.3).
func (s *Service) TriggerReassessment(ctx context.Context, req domainProblem.ReassessmentRequest) (*domainProblem.ReassessmentResult, error) {
	if err := domainProblem.ValidateReassessmentRequest(&req); err != nil {
		return nil, fmt.Errorf("invalid reassessment request: %w", err)
	}

	prob, err := s.problemRepo.Get(ctx, req.ProblemID)
	if err != nil {
		return nil, fmt.Errorf("problem %s not found: %w", req.ProblemID, err)
	}

	prevModelVer := prob.CurrentModelVersion
	newModelVer := prevModelVer
	contradictedCount := 0

	// 1. Inspect current model and update contradicted hypotheses (AIA-22 §8.2)
	if prevModelVer > 0 {
		currentModel, err := s.problemRepo.GetModel(ctx, prob.ID, prevModelVer)
		if err == nil && currentModel != nil {
			newModelVer = prevModelVer + 1
			newModel := &domainProblem.ProblemModel{
				ID:              fmt.Sprintf("%s-model-v%d", prob.ID, newModelVer),
				ProblemID:       prob.ID,
				Version:         newModelVer,
				Objectives:      currentModel.Objectives,
				Constraints:     currentModel.Constraints,
				KnownFacts:      currentModel.KnownFacts,
				Assumptions:     make([]domainProblem.ModelItem, len(currentModel.Assumptions)),
				Unknowns:        currentModel.Unknowns,
				Dependencies:    currentModel.Dependencies,
				Risks:           currentModel.Risks,
				SuccessCriteria: currentModel.SuccessCriteria,
				CreatedAt:       time.Now().UTC(),
			}

			// Copy assumptions and mark contradicted ones
			copy(newModel.Assumptions, currentModel.Assumptions)
			contraMap := make(map[string]bool)
			for _, cid := range req.ContradictedHypotheses {
				contraMap[cid] = true
			}

			for i, a := range newModel.Assumptions {
				if contraMap[a.ID] {
					newModel.Assumptions[i].Status = domainProblem.ItemStatusContradicted
					contradictedCount++
				}
			}

			// Integrate new findings into KnownFacts
			for i, f := range req.NewFindings {
				newModel.KnownFacts = append(newModel.KnownFacts, domainProblem.ModelItem{
					ID:          fmt.Sprintf("fact-reassess-v%d-%d", newModelVer, i+1),
					Category:    domainProblem.CategoryKnownFact,
					Statement:   f,
					Source:      fmt.Sprintf("reassessment_trigger:%s", req.Trigger),
					Confidence:  1.0,
					Status:      domainProblem.ItemStatusConfirmed,
					CreatedBy:   req.AgentID,
					CreatedAt:   time.Now().UTC(),
				})
			}

			// Persist updated model
			_ = s.problemRepo.SaveModel(ctx, newModel)
		}
	}

	// 2. Create immutable AnalysisRecord for Reassessment (AIA-22 §6.3, §8.2)
	conclusions := []string{
		fmt.Sprintf("Trigger: %s", req.Trigger),
		fmt.Sprintf("Trigger Evidence: %s", req.TriggerEvidence),
		fmt.Sprintf("Root Cause Attribution: %s", req.RootCause),
		fmt.Sprintf("Decided Action: %s", req.SuggestedAction),
		fmt.Sprintf("Rationale: %s", req.Rationale),
	}
	recordID := fmt.Sprintf("ar-%s-reassess-%s", prob.ID, uuid.New().String()[:6])
	analysisRec := &domainProblem.AnalysisRecord{
		ID:           recordID,
		ProblemID:    prob.ID,
		ModelVersion: newModelVer,
		AnalysisType: "REASSESSMENT",
		InputRefs:    []string{fmt.Sprintf("trigger:%s", req.Trigger), fmt.Sprintf("cause:%s", req.RootCause)},
		Findings:     req.NewFindings,
		Conclusions:  conclusions,
		AgentID:      req.AgentID,
		CreatedAt:    time.Now().UTC(),
	}
	_ = s.problemRepo.CreateAnalysisRecord(ctx, analysisRec)

	// 3. Update Problem status
	prob.CurrentModelVersion = newModelVer
	requiresNewPlan := false
	switch req.SuggestedAction {
	case domainProblem.ActionModifyPlan, domainProblem.ActionRebuildPlan:
		prob.Status = domainProblem.StatusReassessing
		requiresNewPlan = true
	case domainProblem.ActionPause:
		prob.Status = domainProblem.StatusClosed
	default:
		prob.Status = domainProblem.StatusExecuting
	}
	_ = s.problemRepo.Update(ctx, prob)

	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           domain.ID("audit-" + uuid.New().String()[:8]),
			ActorType:    "SYSTEM",
			ActorID:      req.AgentID,
			Action:       "TRIGGER_REASSESSMENT",
			ResourceType: "PROBLEM",
			ResourceID:   prob.ID,
			Outcome:      "SUCCESS",
			CreatedAt:    time.Now().UTC(),
		})
	}

	return &domainProblem.ReassessmentResult{
		ProblemID:         prob.ID,
		PreviousModelVer:  prevModelVer,
		NewModelVer:       newModelVer,
		ActionTaken:       req.SuggestedAction,
		AnalysisRecordID:  recordID,
		ContradictedCount: contradictedCount,
		RequiresNewPlan:   requiresNewPlan,
		ExecutedAt:        time.Now().UTC(),
	}, nil
}

// Ensure interface compatibility
var _ ClarificationDispatcher = (*clarifyService.Service)(nil)
