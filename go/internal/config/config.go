package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	System       SystemConfig       `yaml:"system"`
	Runtime      RuntimeConfig      `yaml:"runtime"`
	Database     DatabaseConfig     `yaml:"database"`
	Provider     ProviderConfig     `yaml:"provider"`
	Budget       BudgetConfig       `yaml:"budget"`
	Memory       MemoryConfig       `yaml:"memory"`
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	Logging      LoggingConfig      `yaml:"logging"`
	DataSources  DataSourcesConfig  `yaml:"data_sources"`
	Domains      DomainsConfig      `yaml:"domains"`
	Policy       PolicyConfig       `yaml:"policy"`
}

// DomainsConfig holds per-domain project registrations that are resolved from
// the YAML config file, eliminating hardcoded absolute paths from source code.
type DomainsConfig struct {
	Writing WritingDomainConfig `yaml:"writing"`
	Coding  CodingDomainConfig  `yaml:"coding"`
}

type WritingDomainConfig struct {
	// Projects lists writing project registrations. Each path supports ~/ expansion
	// and is resolved to an absolute path at load time relative to the config file.
	Projects []ProjectPathConfig `yaml:"projects"`
}

type CodingDomainConfig struct {
	// Projects lists coding repository registrations.
	Projects []ProjectPathConfig `yaml:"projects"`
}

// ProjectPathConfig is a lightweight project registration entry in the config
// file. Full project lifecycle (goals, continuity, tasks) is managed in SQLite;
// this entry only seeds the initial path so the observer can start without a DB
// round-trip at boot.
type ProjectPathConfig struct {
	// ID is the stable project identifier (e.g. "yuan", "aia").
	ID string `yaml:"id"`
	// Name is a human-readable label (e.g. "元朝系列小說").
	Name string `yaml:"name"`
	// Path is the filesystem location of the project root.
	// Supports ~/  prefix. Relative paths are resolved relative to the directory
	// that contains the config file (set by LoadConfig after parsing).
	Path string `yaml:"path"`
}

// PolicyConfig holds runtime governance settings loaded from config.
type PolicyConfig struct {
	// AllowedPaths is the file-write whitelist. Paths support ~/  expansion.
	// When empty the evaluator falls back to safe built-in defaults
	// (/tmp and /var/tmp only — no user-home directories are assumed).
	AllowedPaths []string `yaml:"allowed_paths"`
}

type BudgetConfig struct {
	MaxDailyTokens           int64   `yaml:"max_daily_tokens"`
	MaxDailyCycles           int     `yaml:"max_daily_cycles"`
	MaxSingleExecutionTokens int64   `yaml:"max_single_execution_tokens"`
	MaxDailySpend            float64 `yaml:"max_daily_spend"`
}

type DataSourcesConfig struct {
	FREDKey         string `yaml:"fred_api_key"`
	FinMindToken    string `yaml:"finmind_token"`
	AlphaVantageKey string `yaml:"alpha_vantage_key"`
	TelegramToken   string `yaml:"telegram_bot_token"`
	TelegramChatID  string `yaml:"telegram_chat_id"`
}

type SystemConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	Timezone string `yaml:"timezone"`
}

type RuntimeConfig struct {
	DataDir                 string        `yaml:"data_dir"`
	WorkspaceDir            string        `yaml:"workspace_dir"`
	MaxConcurrentExecutions int           `yaml:"max_concurrent_executions"`
	MaxConcurrentPerAgent   int           `yaml:"max_concurrent_per_agent"`
	ShutdownTimeout         time.Duration `yaml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	Path   string `yaml:"path"`
}

type ProviderConfig struct {
	Default     string            `yaml:"default"`
	Antigravity AntigravityConfig `yaml:"antigravity"`
}

type AntigravityConfig struct {
	Executable         string        `yaml:"executable"`
	RequiredVersion    string        `yaml:"required_version"`
	VersionPolicy      string        `yaml:"version_policy"`
	OutputFormat       string        `yaml:"output_format"`
	InputFormat        string        `yaml:"input_format"`
	DefaultTimeout     time.Duration `yaml:"default_timeout"`
	SandboxRequired    bool          `yaml:"sandbox_required"`
	SandboxMode        string        `yaml:"sandbox_mode"` // "auto", "bwrap", "systemd", "strict", "env_only"
	RequireOSNamespace bool          `yaml:"require_os_namespace"`
	HiddenPaths        []string      `yaml:"hidden_paths"`
	ReadOnlyPaths      []string      `yaml:"read_only_paths"`
	AllowedEnvVars     []string      `yaml:"allowed_env_vars"`
	SkipPermissions    bool          `yaml:"skip_permissions"`
}

type MemoryConfig struct {
	MaxContextItems    int  `yaml:"max_context_items"`
	MaxContextBytes    int  `yaml:"max_context_bytes"`
	AutoCreateCandidate bool `yaml:"auto_create_candidates"`
	AutoApproveLowRisk bool `yaml:"auto_approve_low_risk"`
}

type OrchestratorConfig struct {
	MaxPlanRetries     int `yaml:"max_plan_retries"`
	MaxTaskRetries     int `yaml:"max_task_retries"`
	MaxSubtasksPerPlan int `yaml:"max_subtasks_per_plan"`
	MaxDependencyDepth int `yaml:"max_dependency_depth"`
}

type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	RetainDays int    `yaml:"retain_days"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "/tmp"
	}
	dataDir := filepath.Join(home, ".local", "share", "aia")
	return &Config{
		System: SystemConfig{
			Name:     "AIA",
			Version:  "1.0",
			Timezone: "Asia/Taipei",
		},
		Runtime: RuntimeConfig{
			DataDir:                 dataDir,
			WorkspaceDir:            filepath.Join(dataDir, "workspaces"),
			MaxConcurrentExecutions: 2,
			MaxConcurrentPerAgent:   1,
			ShutdownTimeout:         30 * time.Second,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(dataDir, "aia.db"),
		},
		Provider: ProviderConfig{
			Default: "antigravity",
			Antigravity: AntigravityConfig{
				Executable:      "agy",
				RequiredVersion: "1.4.12",
				VersionPolicy:   "minimum",
				OutputFormat:    "stream-json",
				InputFormat:     "stream-json",
				DefaultTimeout:  30 * time.Minute,
				SandboxRequired: true,
				SandboxMode:     "auto",
				HiddenPaths: []string{
					filepath.Join(home, ".config", "aia"),
					dataDir,
				},
				SkipPermissions: false,
			},
		},
		Budget: BudgetConfig{
			MaxDailyTokens:           500000,
			MaxDailyCycles:           48,
			MaxSingleExecutionTokens: 120000,
			MaxDailySpend:            100000,
		},
		Memory: MemoryConfig{
			MaxContextItems:    20,
			MaxContextBytes:    24000,
			AutoCreateCandidate: true,
			AutoApproveLowRisk: false,
		},
		Orchestrator: OrchestratorConfig{
			MaxPlanRetries:     1,
			MaxTaskRetries:     2,
			MaxSubtasksPerPlan: 20,
			MaxDependencyDepth: 8,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			RetainDays: 30,
		},
	}
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// resolveSinglePath expands ~/ and resolves relative paths against baseDir.
func resolveSinglePath(path, baseDir string) string {
	if path == "" {
		return path
	}
	expanded := ExpandPath(path)
	if filepath.IsAbs(expanded) {
		return expanded
	}
	abs, err := filepath.Abs(filepath.Join(baseDir, expanded))
	if err != nil {
		return filepath.Join(baseDir, expanded)
	}
	return abs
}

// resolveProjectPaths resolves every ProjectPathConfig.Path in place and
// returns the updated slice. baseDir is the directory containing aia.yaml.
func resolveProjectPaths(projects []ProjectPathConfig, baseDir string) []ProjectPathConfig {
	for i := range projects {
		projects[i].Path = resolveSinglePath(projects[i].Path, baseDir)
	}
	return projects
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}

	cleanPath := ExpandPath(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", cleanPath, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	cfg.Runtime.DataDir = ExpandPath(cfg.Runtime.DataDir)
	cfg.Runtime.WorkspaceDir = ExpandPath(cfg.Runtime.WorkspaceDir)
	cfg.Database.Path = ExpandPath(cfg.Database.Path)

	// Resolve domain project paths relative to the config file's directory so
	// that users can write short relative paths (e.g. "novel/yuan") instead of
	// hardcoded absolute paths.
	configDir := filepath.Dir(cleanPath)
	cfg.Domains.Writing.Projects = resolveProjectPaths(cfg.Domains.Writing.Projects, configDir)
	cfg.Domains.Coding.Projects = resolveProjectPaths(cfg.Domains.Coding.Projects, configDir)

	// Resolve policy allowed paths.
	for i, p := range cfg.Policy.AllowedPaths {
		cfg.Policy.AllowedPaths[i] = resolveSinglePath(p, configDir)
	}

	// Environment variable overrides for data sources
	if k := os.Getenv("FRED_API_KEY"); k != "" {
		cfg.DataSources.FREDKey = k
	}
	if k := os.Getenv("FINMIND_TOKEN"); k != "" {
		cfg.DataSources.FinMindToken = k
	}
	if k := os.Getenv("ALPHA_VANTAGE_KEY"); k != "" {
		cfg.DataSources.AlphaVantageKey = k
	}
	if k := os.Getenv("TELEGRAM_BOT_TOKEN"); k != "" {
		cfg.DataSources.TelegramToken = k
	}
	if k := os.Getenv("TELEGRAM_CHAT_ID"); k != "" {
		cfg.DataSources.TelegramChatID = k
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Runtime.DataDir == "" {
		return fmt.Errorf("runtime.data_dir must not be empty")
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database.path must not be empty")
	}
	if c.Runtime.MaxConcurrentExecutions <= 0 {
		return fmt.Errorf("runtime.max_concurrent_executions must be > 0")
	}
	if c.Provider.Antigravity.Executable == "" {
		return fmt.Errorf("provider.antigravity.executable must not be empty")
	}
	return nil
}
