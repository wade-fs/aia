package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/wade/aia/internal/config"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/memory/retrieval"
	"github.com/wade/aia/internal/orchestrator"
	"github.com/wade/aia/internal/policy"
	"github.com/wade/aia/internal/provider"
	"github.com/wade/aia/internal/provider/antigravity"
	"github.com/wade/aia/internal/recovery"
	agentService "github.com/wade/aia/internal/service/agent"
	execService "github.com/wade/aia/internal/service/execution"
	memoryService "github.com/wade/aia/internal/service/memory"
	projectService "github.com/wade/aia/internal/service/project"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/validation"
	"github.com/wade/aia/internal/workspace"
)

type App struct {
	Config        *config.Config
	DB            *sqlite.DB
	Provider      provider.Provider
	WorkspaceMgr  *workspace.Manager
	ProjectSvc    *projectService.Service
	AgentSvc      *agentService.Service
	TaskSvc       *taskService.Service
	ExecutionSvc  *execService.Service
	MemorySvc     *memoryService.Service
	RecoveryMgr   *recovery.Manager
	Scheduler     *orchestrator.Scheduler
	PersonalAgent *orchestrator.PersonalAgent
}

func NewApp(cfgPath string, overrideProv provider.Provider) (*App, error) {
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// 1. Initialize SQLite Database
	db, err := sqlite.NewDB(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sqlite database: %w", err)
	}

	// 2. Repositories
	projRepo := repository.NewProjectRepository(db)
	agRepo := repository.NewAgentRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	execRepo := repository.NewExecutionRepository(db)
	memRepo := repository.NewMemoryRepository(db)
	artRepo := repository.NewArtifactRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// 3. Provider
	var prov provider.Provider
	if overrideProv != nil {
		prov = overrideProv
	} else {
		prov = antigravity.NewAdapter(cfg.Provider.Antigravity)
	}

	// 4. Workspace & Validation
	wsMgr, err := workspace.NewManager(cfg.Runtime.WorkspaceDir)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	validator := validation.NewDefaultValidator()

	// 5. Services
	projSvc := projectService.NewService(projRepo)
	agSvc := agentService.NewService(agRepo)
	taskSvc := taskService.NewService(taskRepo, auditRepo)
	taskSvc.SetHealthChecker(prov)
	execSvc := execService.NewService(execRepo, taskRepo, artRepo, auditRepo, prov, wsMgr, validator)
	memSvc := memoryService.NewService(memRepo)

	// 6. Recovery
	recMgr := recovery.NewManager(taskRepo, execRepo, auditRepo)
	recMgr.SetValidator(validator)
	recMgr.SetWorkspaceManager(wsMgr)
	recMgr.SetArtifactRepository(artRepo)

	// 7. Orchestrator & Memory Retrieval
	retriever := retrieval.NewRetriever(memRepo, cfg.Memory)
	evaluator := policy.NewEvaluator()
	scheduler := orchestrator.NewScheduler(taskSvc, execSvc, agSvc, cfg.Runtime)
	pa := orchestrator.NewPersonalAgent(prov, retriever, taskSvc, scheduler, evaluator)

	a := &App{
		Config:        cfg,
		DB:            db,
		Provider:      prov,
		WorkspaceMgr:  wsMgr,
		ProjectSvc:    projSvc,
		AgentSvc:      agSvc,
		TaskSvc:       taskSvc,
		ExecutionSvc:  execSvc,
		MemorySvc:     memSvc,
		RecoveryMgr:   recMgr,
		Scheduler:     scheduler,
		PersonalAgent: pa,
	}

	// Seed default domain agents & default project
	a.seedDefaultAgents(context.Background())
	a.seedDefaultProjects(context.Background())

	// Run recovery check
	if summary, err := recMgr.Recover(context.Background()); err == nil {
		if summary.RecoveredTasks > 0 || summary.InterruptedExecutions > 0 {
			slog.Info("startup recovery executed", "interrupted_executions", summary.InterruptedExecutions, "recovered_tasks", summary.RecoveredTasks)
		}
	}

	return a, nil
}

func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}

func (a *App) seedDefaultAgents(ctx context.Context) {
	defaultAgents := []agent.Agent{
		{
			ID:           "personal-agent",
			Name:         "Personal Agent",
			Description:  "Central orchestrator for task planning and routing",
			Domain:       "orchestrator",
			Instructions: "You are the primary AIA personal assistant. Understand user intent and coordinate domain agents.",
			Capabilities: []string{"plan", "chat", "route"},
			Enabled:      true,
			Version:      1,
		},
		{
			ID:           "coding-agent",
			Name:         "Coding Agent",
			Description:  "Specialized in software development, debugging, and testing",
			Domain:       "coding",
			Instructions: "You are a professional software engineer. Write modular, clean, and tested code.",
			Capabilities: []string{"code", "test", "debug", "refactor"},
			Enabled:      true,
			Version:      1,
		},
		{
			ID:           "writing-agent",
			Name:         "Writing Agent",
			Description:  "Specialized in copywriting, documentation, and storytelling",
			Domain:       "writing",
			Instructions: "You are an expert writer. Produce compelling and structured content.",
			Capabilities: []string{"write", "edit", "summarize"},
			Enabled:      true,
			Version:      1,
		},
		{
			ID:           "investment-agent",
			Name:         "Investment Agent",
			Description:  "Specialized in financial analysis, valuation, and market research",
			Domain:       "investment",
			Instructions: "You are a cautious financial analyst. Analyze trends with objective data.",
			Capabilities: []string{"analyze", "research"},
			Enabled:      true,
			Version:      1,
		},
		{
			ID:           "video-agent",
			Name:         "Video Agent",
			Description:  "Specialized in scriptwriting, storyboarding, and video assets",
			Domain:       "video",
			Instructions: "You are a creative video producer. Craft engaging scripts and visual concepts.",
			Capabilities: []string{"script", "storyboard"},
			Enabled:      true,
			Version:      1,
		},
	}

	for _, ag := range defaultAgents {
		_ = a.AgentSvc.Upsert(ctx, &ag)
	}

	// Also attempt to load agents from configs/agents directory
	_ = a.AgentSvc.LoadAgentsFromDir(ctx, "configs/agents")
	_ = a.AgentSvc.LoadAgentsFromDir(ctx, os.ExpandEnv("$HOME/.config/aia/agents"))
}

func (a *App) seedDefaultProjects(ctx context.Context) {
	_, _ = a.ProjectSvc.EnsureDefault(ctx, a.Config.Runtime.WorkspaceDir)
}
