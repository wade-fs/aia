package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/wade/aia/agent-runtime/internal/audit"
	"github.com/wade/aia/agent-runtime/internal/git"
	"github.com/wade/aia/agent-runtime/internal/recovery"
	"github.com/wade/aia/agent-runtime/internal/state"
	"github.com/wade/aia/agent-runtime/internal/storage"
	"github.com/wade/aia/agent-runtime/internal/task"
	"github.com/wade/aia/agent-runtime/internal/validator"
	"github.com/wade/aia/agent-runtime/internal/worker"
	"github.com/wade/aia/agent-runtime/internal/worker/aider"
	"github.com/wade/aia/agent-runtime/internal/workspace"
)

type Config struct {
	Runtime struct {
		DataDir       string `yaml:"data_dir"`
		WorkspacesDir string `yaml:"workspaces_dir"`
		SourceRepo    string `yaml:"source_repo"`
	} `yaml:"runtime"`
	Worker struct {
		Aider struct {
			BinaryPath   string `yaml:"binary_path"`
			DefaultModel string `yaml:"default_model"`
			Timeout      string `yaml:"timeout"`
		} `yaml:"aider"`
	} `yaml:"worker"`
	Validator struct {
		TestCommand string `yaml:"test_command"`
	} `yaml:"validator"`
	Git struct {
		AutoCommit bool `yaml:"auto_commit"`
	} `yaml:"git"`
}

type Runtime struct {
	config        *Config
	storage       *storage.DB
	taskManager   *task.Manager
	workerManager *worker.Manager
	stateManager  *state.Manager
	workspaceMgr  *workspace.Manager
	validator     *validator.Validator
	recoveryMgr   *recovery.Manager
	audit         *audit.Logger
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = "./configs/config.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	if cfg.Runtime.DataDir == "" {
		cfg.Runtime.DataDir = "./data"
	}
	if cfg.Runtime.WorkspacesDir == "" {
		cfg.Runtime.WorkspacesDir = "./workspaces"
	}
	if cfg.Validator.TestCommand == "" {
		cfg.Validator.TestCommand = "go test ./..."
	}

	return &cfg, nil
}

func New(configPath string) (*Runtime, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// 1. Open Database
	dbPath := filepath.Join(cfg.Runtime.DataDir, "agent.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open runtime db: %w", err)
	}

	// 2. Initialize Audit Logger
	logger := audit.NewLogger(db)

	// 3. Worker Timeout & Aider Adapter
	timeout := 30 * time.Minute
	if cfg.Worker.Aider.Timeout != "" {
		if d, err := time.ParseDuration(cfg.Worker.Aider.Timeout); err == nil {
			timeout = d
		}
	}
	aiderAdapter := aider.NewAdapter(
		cfg.Worker.Aider.BinaryPath,
		cfg.Worker.Aider.DefaultModel,
		timeout,
	)

	// 4. Managers
	workerMgr := worker.NewManager(aiderAdapter, db, logger, cfg.Worker.Aider.DefaultModel, timeout)
	taskMgr := task.NewManager(db, logger)
	stateMgr := state.NewManager(db, logger)
	wsMgr := workspace.NewManager(cfg.Runtime.WorkspacesDir, cfg.Runtime.SourceRepo)
	val := validator.NewValidator(cfg.Validator.TestCommand)
	recMgr := recovery.NewManager(db, workerMgr, taskMgr, logger, val)

	return &Runtime{
		config:        cfg,
		storage:       db,
		taskManager:   taskMgr,
		workerManager: workerMgr,
		stateManager:  stateMgr,
		workspaceMgr:  wsMgr,
		validator:     val,
		recoveryMgr:   recMgr,
		audit:         logger,
	}, nil
}

func (r *Runtime) Close() error {
	if r.storage != nil {
		return r.storage.Close()
	}
	return nil
}

func (r *Runtime) TaskManager() *task.Manager {
	return r.taskManager
}

func (r *Runtime) WorkerManager() *worker.Manager {
	return r.workerManager
}

func (r *Runtime) RecoveryManager() *recovery.Manager {
	return r.recoveryMgr
}

func (r *Runtime) StateManager() *state.Manager {
	return r.stateManager
}

func (r *Runtime) Storage() *storage.DB {
	return r.storage
}

func (r *Runtime) Audit() *audit.Logger {
	return r.audit
}

// Start triggers recovery and prepares the runtime
func (r *Runtime) Start(ctx context.Context) error {
	_ = r.audit.LogInfo("runtime_start", "system", "runtime", "Agent Runtime starting, running recovery checks...")
	if err := r.recoveryMgr.Run(ctx); err != nil {
		return fmt.Errorf("recovery check failed on startup: %w", err)
	}
	return nil
}

// RunTask executes the full task lifecycle
func (r *Runtime) RunTask(ctx context.Context, taskID string) error {
	// 1. Get task & validate status
	t, err := r.taskManager.Get(taskID)
	if err != nil {
		return fmt.Errorf("task %s not found: %w", taskID, err)
	}

	if t.Status == task.TaskStatusRunning {
		return fmt.Errorf("task %s is already running: cannot re-run concurrent task (T11)", taskID)
	}
	if t.Status == task.TaskStatusSucceeded {
		return fmt.Errorf("task %s has already completed with SUCCEEDED status", taskID)
	}
	if t.Status == task.TaskStatusUnknown {
		return fmt.Errorf("task %s is in UNKNOWN status: requires reconciliation before execution", taskID)
	}
	if t.Status != task.TaskStatusCreated && t.Status != task.TaskStatusReady {
		return fmt.Errorf("cannot run task %s in %s status", taskID, t.Status)
	}

	// 2. Create isolated workspace
	ws, err := r.workspaceMgr.Create(t.ID)
	if err != nil {
		_ = r.audit.LogError("workspace_creation_failed", "task", t.ID, err.Error())
		return fmt.Errorf("failed to create workspace for task %s: %w", taskID, err)
	}
	t.WorkspaceID = ws.ID
	t.GitBaseCommit = ws.BaseCommit
	if err := r.taskManager.Update(t); err != nil {
		return fmt.Errorf("failed to update workspace info on task: %w", err)
	}

	// 3. Transition to RUNNING
	if err := r.stateManager.Transition(t.ID, t.Status, task.TaskStatusRunning); err != nil {
		return fmt.Errorf("failed to transition task status to RUNNING: %w", err)
	}
	t.Status = task.TaskStatusRunning
	now := time.Now()
	t.StartedAt = &now
	_ = r.taskManager.Update(t)

	// 4. Run Worker
	workerResult, execErr := r.workerManager.Run(ctx, t, ws.Path)
	if workerResult != nil {
		t.WorkerExecutionID = workerResult.ExecutionID
	}

	// 5. Result Validation
	valResult, valErr := r.validator.Validate(ctx, workerResult, ws.Path, ws.BaseCommit)
	if valResult != nil {
		vrRecord := &storage.ValidationResultRecord{
			ID:              uuid.New().String(),
			TaskID:          t.ID,
			Passed:          valResult.Passed,
			GitDiffExists:   valResult.GitDiffExists,
			TestPassed:      valResult.TestPassed,
			UnexpectedFiles: strings.Join(valResult.UnexpectedFiles, ","),
			Summary:         valResult.Summary,
			CreatedAt:       time.Now(),
		}
		_ = r.storage.SaveValidationResult(ctx, vrRecord)
		t.ValidationResult = valResult.Summary
	}

	// 6. Transition to final status based on validation
	completedTime := time.Now()
	t.CompletedAt = &completedTime

	if execErr != nil || valErr != nil || valResult == nil || !valResult.Passed {
		var reasons []string
		if execErr != nil {
			reasons = append(reasons, fmt.Sprintf("execution error: %v", execErr))
		}
		if valErr != nil {
			reasons = append(reasons, fmt.Sprintf("validation error: %v", valErr))
		}
		if valResult != nil && !valResult.Passed {
			reasons = append(reasons, valResult.Summary)
		}
		t.Error = strings.Join(reasons, " | ")

		_ = r.stateManager.Transition(t.ID, task.TaskStatusRunning, task.TaskStatusFailed)
		t.Status = task.TaskStatusFailed
		_ = r.taskManager.Update(t)

		return fmt.Errorf("task execution failed: %s", t.Error)
	}

	// Success path
	gitClient := git.NewClient(ws.Path)
	finalCommit, _ := gitClient.GetCurrentCommit()
	t.GitFinalCommit = finalCommit
	t.Result = valResult.Summary
	_ = r.stateManager.Transition(t.ID, task.TaskStatusRunning, task.TaskStatusSucceeded)
	t.Status = task.TaskStatusSucceeded
	_ = r.taskManager.Update(t)

	_ = r.audit.LogInfo("task_succeeded", "task", t.ID, fmt.Sprintf("Result: %s", t.Result))
	return nil
}
