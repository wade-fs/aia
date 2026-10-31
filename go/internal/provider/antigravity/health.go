package antigravity

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func CheckVersion(ctx context.Context, executable string) (string, error) {
	cmd := exec.CommandContext(ctx, executable, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run %s --version: %w (%s)", executable, err, string(out))
	}
	ver := strings.TrimSpace(string(out))
	return ver, nil
}

func HealthCheck(ctx context.Context, executable, requiredVersion string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ver, err := CheckVersion(ctx, executable)
	if err != nil {
		return fmt.Errorf("antigravity executable check failed: %w", err)
	}

	if requiredVersion != "" && !strings.Contains(ver, requiredVersion) {
		return fmt.Errorf("antigravity version mismatch: got %q, required %q", ver, requiredVersion)
	}

	return nil
}
