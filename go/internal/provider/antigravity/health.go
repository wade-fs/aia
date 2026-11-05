package antigravity

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
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

func parseVersion(v string) []int {
	start := -1
	for i, r := range v {
		if r >= '0' && r <= '9' {
			start = i
			break
		}
	}
	if start == -1 {
		return nil
	}
	v = v[start:]
	if idx := strings.IndexAny(v, " \t\r\n+-~"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			break
		}
		nums = append(nums, n)
	}
	return nums
}

func compareVersions(v1, v2 string) (int, bool) {
	p1 := parseVersion(v1)
	p2 := parseVersion(v2)
	if len(p1) == 0 || len(p2) == 0 {
		return 0, false
	}
	l := len(p1)
	if len(p2) > l {
		l = len(p2)
	}
	for i := 0; i < l; i++ {
		var n1, n2 int
		if i < len(p1) {
			n1 = p1[i]
		}
		if i < len(p2) {
			n2 = p2[i]
		}
		if n1 < n2 {
			return -1, true
		} else if n1 > n2 {
			return 1, true
		}
	}
	return 0, true
}

func HealthCheck(ctx context.Context, executable, requiredVersion string) error {
	return HealthCheckWithPolicy(ctx, executable, requiredVersion, "minimum")
}

func HealthCheckWithPolicy(ctx context.Context, executable, requiredVersion, versionPolicy string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ver, err := CheckVersion(ctx, executable)
	if err != nil {
		return fmt.Errorf("antigravity executable check failed: %w", err)
	}

	req := strings.TrimSpace(requiredVersion)
	if req == "" || versionPolicy == "none" {
		return nil
	}

	policy := strings.ToLower(strings.TrimSpace(versionPolicy))
	if policy == "" {
		policy = "minimum"
	}

	cmp, ok := compareVersions(ver, req)
	if ok {
		if policy == "exact" {
			if cmp != 0 {
				return fmt.Errorf("antigravity version mismatch: got %q, required %q", ver, req)
			}
			return nil
		}
		// Default to minimum: got must be >= req
		if cmp < 0 {
			return fmt.Errorf("antigravity version mismatch: got %q, required minimum %q", ver, req)
		}
		return nil
	}

	// Fallback to substring match if semver parsing is unavailable
	if !strings.Contains(ver, req) {
		return fmt.Errorf("antigravity version mismatch: got %q, required %q", ver, req)
	}

	return nil
}

