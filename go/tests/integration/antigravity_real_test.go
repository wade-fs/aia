package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/config"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/profile"
	"github.com/wade/aia/internal/provider"
	"github.com/wade/aia/internal/provider/antigravity"
)

func checkAgyInstalled(t *testing.T) string {
	// 1. Check environment variable for configured agy path
	for _, envKey := range []string{"AGY_PATH", "AIA_AGY_PATH", "AGY_EXECUTABLE"} {
		if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
			if path, err := exec.LookPath(val); err == nil {
				return path
			}
			t.Skipf("Configured agy executable '%s' via %s not found; skipping real agy integration test", val, envKey)
		}
	}

	// 2. Fall back to standard PATH
	path, err := exec.LookPath("agy")
	if err != nil {
		t.Skip("agy CLI not found on PATH or via AGY_PATH / AIA_AGY_PATH; skipping real agy integration test")
	}
	return path
}

func TestAgy_RealIntegration_AG01_Version(t *testing.T) {
	agyPath := checkAgyInstalled(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	adapter := antigravity.NewAdapter(config.AntigravityConfig{
		Executable:      agyPath,
		RequiredVersion: "1.4.12",
	})

	ver, err := adapter.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get agy version: %v", err)
	}
	if ver != "1.4.12" {
		t.Errorf("expected version 1.4.12, got %s", ver)
	}

	if err := adapter.HealthCheck(ctx); err != nil {
		t.Errorf("health check failed: %v", err)
	}
}

func TestAgy_RealIntegration_AG02_AG04_StreamJson(t *testing.T) {
	agyPath := checkAgyInstalled(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "real_ws")

	adapter := antigravity.NewAdapter(config.AntigravityConfig{
		Executable:      agyPath,
		RequiredVersion: "1.4.12",
	})

	req := provider.StartRequest{
		ExecutionID: "real-exec-1",
		TaskID:      "real-task-1",
		WorkDir:     wsDir,
		Prompt:      "Reply strictly with the word PONG without extra explanation",
		Agent: agent.Agent{
			ID:   "coding-agent",
			Name: "Coding Agent",
		},
		Profile: profile.ExecutionProfile{
			Sandbox: false,
		},
	}

	session, err := adapter.Start(ctx, req)
	if err != nil {
		t.Fatalf("failed to start agy session: %v", err)
	}
	defer session.Stop(ctx)

	var receivedInit bool
	var receivedStep bool
	for ev := range session.Events() {
		if ev.Type == "init" {
			receivedInit = true
		} else if ev.Type == "step_update" {
			receivedStep = true
		}
	}

	res, err := session.Wait(ctx)
	if err != nil {
		t.Fatalf("agy execution failed: %v", err)
	}

	if !receivedInit {
		t.Errorf("expected to receive 'init' stream event from agy")
	}
	if !receivedStep {
		t.Errorf("expected to receive 'step_update' stream event from agy")
	}
	if res.Status != "SUCCESS" {
		t.Errorf("expected result status SUCCESS, got %s", res.Status)
	}
	if res.ConversationID == "" {
		t.Errorf("expected non-empty conversation_id from real agy")
	}
}
