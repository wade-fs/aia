package autonomy

import (
	"context"
	"errors"
	"time"

	"github.com/wade/aia/internal/autonomy/cycle"
	"github.com/wade/aia/internal/autonomy/deliberation"
	"github.com/wade/aia/internal/autonomy/initiative"
	"github.com/wade/aia/internal/autonomy/learning"
	"github.com/wade/aia/internal/autonomy/observation"
	"github.com/wade/aia/internal/autonomy/recall"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/policy"
	approvalService "github.com/wade/aia/internal/service/approval"
	clarifyService "github.com/wade/aia/internal/service/clarification"
	"github.com/wade/aia/internal/service/investment"
	problemService "github.com/wade/aia/internal/service/problem"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var (
	ErrAutonomyEmergencyStopped = errors.New("cannot run autonomy cycle: runtime is under EMERGENCY_STOP")
)

type EngineConfig struct {
	AutoExecuteSafeTasks     bool
	DefaultPriority          int
	MaxConcurrentActiveTasks int
}

type Dependencies struct {
	AutonomyRepo     *repository.AutonomyRepository
	DecisionRepo     *repository.DecisionRepository
	ContinuityRepo   *repository.ContinuityRepository
	ExecutionRepo    *repository.ExecutionRepository
	LearningRepo     *repository.LearningRepository
	GoalRepo         *repository.GoalRepository
	EventHistoryRepo *repository.EventHistoryRepository
	KnowledgeRepo    *repository.KnowledgeRepository
	MemoryRepo       *repository.MemoryRepository
	ProjectRepo      *repository.ProjectRepository
	TaskRepo         *repository.TaskRepository
	GovernanceRepo   *repository.GovernanceRepository
	AuditRepo        *repository.AuditRepository
	TaskSvc          *taskService.Service
	ApprovalSvc      *approvalService.Service
	InvSvc           *investment.Service
	Evaluator        *policy.Evaluator
	Bus              eventbus.Publisher
	ProblemRepo      *repository.ProblemRepository
	PlanRepo         *repository.PlanRepository
	ProblemSvc       *problemService.Service
	ClarificationSvc *clarifyService.Service
	Config           EngineConfig
}

type RunCycleInput struct {
	ProjectID                string
	TriggerType              continuity.CycleTriggerType
	TriggerRef               string
	GoalID                   *string
	Summary                  string
	MaxConcurrentActiveTasks int
	AsOf                     *time.Time
	ReplayMode               bool
}

// Engine exposes the unified Autonomy Core API while delegating to modular subpackages:
// observation, recall, initiative, deliberation, learning, and cycle (AIA-08, AIA-09, AIA-11, AIA-18).
type Engine struct {
	obsRegistry *observation.Registry
	recallEng   *recall.Engine
	discoverer  *initiative.Discoverer
	deliberator *deliberation.Deliberator
	learningEng *learning.Engine
	coordinator *cycle.Coordinator
	cfg         EngineConfig
}

// NewEngine preserves backwards-compatible construction while wiring all modular subpackages.
func NewEngine(
	autonomyRepo *repository.AutonomyRepository,
	decisionRepo *repository.DecisionRepository,
	contRepo *repository.ContinuityRepository,
	goalRepo *repository.GoalRepository,
	taskSvc *taskService.Service,
	apprSvc *approvalService.Service,
	evaluator *policy.Evaluator,
	govRepo *repository.GovernanceRepository,
	auditRepo *repository.AuditRepository,
	bus eventbus.Publisher,
) *Engine {
	return NewEngineWithDeps(Dependencies{
		AutonomyRepo:   autonomyRepo,
		DecisionRepo:   decisionRepo,
		ContinuityRepo: contRepo,
		GoalRepo:       goalRepo,
		GovernanceRepo: govRepo,
		AuditRepo:      auditRepo,
		TaskSvc:        taskSvc,
		ApprovalSvc:    apprSvc,
		Evaluator:      evaluator,
		Bus:            bus,
	})
}

// NewEngineWithDeps constructs a fully wired Autonomy Core Engine with L1–L5 Recall and Domain Pilots.
func NewEngineWithDeps(deps Dependencies) *Engine {
	obsReg := observation.NewRegistry()
	recallEng := recall.NewEngine(
		deps.EventHistoryRepo,
		deps.KnowledgeRepo,
		deps.MemoryRepo,
		deps.ContinuityRepo,
		deps.GoalRepo,
		deps.DecisionRepo,
		deps.ProjectRepo,
		deps.TaskRepo,
	)
	if deps.LearningRepo != nil {
		recallEng.SetLearningRepository(deps.LearningRepo)
	}
	if deps.ProblemRepo != nil {
		recallEng.SetProblemRepository(deps.ProblemRepo)
	}
	if deps.PlanRepo != nil {
		recallEng.SetPlanRepository(deps.PlanRepo)
	}
	disc := initiative.NewDiscoverer(deps.AutonomyRepo)
	delib := deliberation.NewDeliberator()
	var learnEng *learning.Engine
	if deps.LearningRepo != nil {
		learnEng = learning.NewEngine(learning.Dependencies{
			LearningRepo:     deps.LearningRepo,
			KnowledgeRepo:    deps.KnowledgeRepo,
			MemoryRepo:       deps.MemoryRepo,
			DecisionRepo:     deps.DecisionRepo,
			EventHistoryRepo: deps.EventHistoryRepo,
			AuditRepo:        deps.AuditRepo,
			ApprovalSvc:      deps.ApprovalSvc,
		})
	}

	coord := cycle.NewCoordinator(cycle.CoordinatorDeps{
		ObserverRegistry: obsReg,
		RecallEngine:     recallEng,
		Discoverer:       disc,
		Deliberator:      delib,
		AutonomyRepo:     deps.AutonomyRepo,
		DecisionRepo:     deps.DecisionRepo,
		ContinuityRepo:   deps.ContinuityRepo,
		ExecutionRepo:    deps.ExecutionRepo,
		GoalRepo:         deps.GoalRepo,
		EventHistoryRepo: deps.EventHistoryRepo,
		KnowledgeRepo:    deps.KnowledgeRepo,
		ProjectRepo:      deps.ProjectRepo,
		GovernanceRepo:   deps.GovernanceRepo,
		AuditRepo:        deps.AuditRepo,
		TaskSvc:          deps.TaskSvc,
		ApprovalSvc:      deps.ApprovalSvc,
		Evaluator:        deps.Evaluator,
		Bus:              deps.Bus,
		ProblemRepo:      deps.ProblemRepo,
		PlanRepo:         deps.PlanRepo,
		ProblemSvc:       deps.ProblemSvc,
		ClarificationSvc: deps.ClarificationSvc,
	})

	if deps.InvSvc != nil {
		coord.RegisterPilot(NewInvestmentPilotExecutor(deps.InvSvc, deps.TaskSvc))
	}

	return &Engine{
		obsRegistry: obsReg,
		recallEng:   recallEng,
		discoverer:  disc,
		deliberator: delib,
		learningEng: learnEng,
		coordinator: coord,
		cfg:         deps.Config,
	}
}

func (e *Engine) RegisterObserver(name string, obs domainAutonomy.EnvironmentObserver) {
	if e.obsRegistry != nil {
		e.obsRegistry.Register(name, obs)
	}
}

func (e *Engine) UnregisterObserver(name string) {
	if e.obsRegistry != nil {
		e.obsRegistry.Unregister(name)
	}
}

func (e *Engine) RegisterPilot(p cycle.DomainPilotExecutor) {
	if e.coordinator != nil {
		e.coordinator.RegisterPilot(p)
	}
}

func (e *Engine) SetProblemService(svc *problemService.Service) {
	if e.coordinator != nil {
		e.coordinator.SetProblemService(svc)
	}
}

func (e *Engine) SetClarificationService(svc *clarifyService.Service) {
	if e.coordinator != nil {
		e.coordinator.SetClarificationService(svc)
	}
}

func (e *Engine) RecallEngine() *recall.Engine {
	return e.recallEng
}

func (e *Engine) LearningEngine() *learning.Engine {
	return e.learningEng
}

// RunCycle executes one complete 11-step Autonomy Closed Loop.
func (e *Engine) RunCycle(ctx context.Context, in RunCycleInput) (*domainAutonomy.CycleOutcome, error) {
	maxConc := in.MaxConcurrentActiveTasks
	if maxConc <= 0 {
		maxConc = e.cfg.MaxConcurrentActiveTasks
	}
	outcome, err := e.coordinator.RunCycle(ctx, cycle.RunInput{
		ProjectID:                in.ProjectID,
		TriggerType:              in.TriggerType,
		TriggerRef:               in.TriggerRef,
		GoalID:                   in.GoalID,
		Summary:                  in.Summary,
		MaxConcurrentActiveTasks: maxConc,
		AsOf:                     in.AsOf,
		ReplayMode:               in.ReplayMode,
	})
	if errors.Is(err, cycle.ErrEmergencyStopped) {
		return nil, ErrAutonomyEmergencyStopped
	}
	return outcome, err
}
