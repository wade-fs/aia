package antigravity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wade/aia/internal/config"
	"github.com/wade/aia/internal/provider"
)

type Adapter struct {
	cfg config.AntigravityConfig
}

func NewAdapter(cfg config.AntigravityConfig) *Adapter {
	return &Adapter{cfg: cfg}
}

func (a *Adapter) Name() string {
	return "antigravity"
}

func (a *Adapter) Version(ctx context.Context) (string, error) {
	return CheckVersion(ctx, a.cfg.Executable)
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	policy := a.cfg.VersionPolicy
	if policy == "" {
		policy = "minimum"
	}
	return HealthCheckWithPolicy(ctx, a.cfg.Executable, a.cfg.RequiredVersion, policy)
}

func (a *Adapter) Start(ctx context.Context, req provider.StartRequest) (provider.ProviderSession, error) {
	if req.WorkDir != "" {
		if err := os.MkdirAll(req.WorkDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create working directory %s: %w", req.WorkDir, err)
		}
	}

	args := []string{
		"--input-format", "stream-json",
		"--output-format", "stream-json",
	}

	if req.Profile.Sandbox || a.cfg.SandboxRequired {
		args = append(args, "--sandbox")
	}

	if req.Profile.Model != "" {
		args = append(args, "--model", req.Profile.Model)
	}

	if req.Profile.Effort != "" {
		args = append(args, "--effort", req.Profile.Effort)
	}

	if req.Profile.Mode != "" {
		args = append(args, "--mode", req.Profile.Mode)
	}

	if req.ConversationID != "" {
		args = append(args, "--conversation", req.ConversationID)
	}

	// Prepare session settings if configured
	if len(req.Profile.ExtraFlags) > 0 {
		settingsPath, err := a.writeSettingsFile(req)
		if err == nil {
			args = append(args, "--settings", settingsPath)
		}
	}

	plan, err := BuildSandboxPlan(ctx, a.cfg, a.cfg.Executable, args, req.WorkDir, os.Environ())
	if err != nil {
		return nil, fmt.Errorf("failed to construct isolated subprocess sandbox: %w", err)
	}

	return NewSessionWithPlan(ctx, plan, req)
}

func (a *Adapter) writeSettingsFile(req provider.StartRequest) (string, error) {
	settingsDir := filepath.Join(os.TempDir(), "aia", "executions", string(req.ExecutionID))
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "", err
	}

	settingsPath := filepath.Join(settingsDir, "settings.json")
	data, err := json.MarshalIndent(req.Profile.ExtraFlags, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return "", err
	}

	return settingsPath, nil
}
