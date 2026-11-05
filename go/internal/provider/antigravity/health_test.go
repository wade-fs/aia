package antigravity

import (
	"context"
	"os/exec"
	"testing"
)

func TestParseAndCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
		ok       bool
	}{
		{"1.5.0", "1.4.12", 1, true},
		{"1.4.12", "1.5.0", -1, true},
		{"1.5.0", "1.5.0", 0, true},
		{"v1.5.0", "1.5.0", 0, true},
		{"1.5.0-beta", "1.5.0", 0, true},
		{"2.0.0", "1.99.99", 1, true},
		{"1.4", "1.4.12", -1, true},
		{"invalid", "1.0.0", 0, false},
	}

	for _, tc := range tests {
		cmp, ok := compareVersions(tc.v1, tc.v2)
		if ok != tc.ok {
			t.Errorf("compareVersions(%q, %q) ok = %v, want %v", tc.v1, tc.v2, ok, tc.ok)
		}
		if ok && cmp != tc.expected {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.v1, tc.v2, cmp, tc.expected)
		}
	}
}

func TestHealthCheckWithPolicy(t *testing.T) {
	ctx := context.Background()
	agyPath, err := exec.LookPath("agy")
	if err != nil {
		t.Skip("agy not installed, skipping live health check test")
	}

	// 1. Empty required version accepts whatever is installed
	if err := HealthCheckWithPolicy(ctx, agyPath, "", "minimum"); err != nil {
		t.Errorf("expected empty requiredVersion to pass, got: %v", err)
	}

	// 2. Minimum policy with lower version requirement passes (e.g. 1.4.12 passes on 1.5.0)
	if err := HealthCheckWithPolicy(ctx, agyPath, "1.4.12", "minimum"); err != nil {
		t.Errorf("expected 1.4.12 minimum to pass on installed agy, got: %v", err)
	}

	// 3. None policy passes even if required version is high
	if err := HealthCheckWithPolicy(ctx, agyPath, "99.0.0", "none"); err != nil {
		t.Errorf("expected policy 'none' to pass, got: %v", err)
	}

	// 4. Minimum policy with impossible high version fails
	if err := HealthCheckWithPolicy(ctx, agyPath, "99.0.0", "minimum"); err == nil {
		t.Errorf("expected 99.0.0 minimum to fail, but it passed")
	}

	// 5. Exact policy with different version fails
	if err := HealthCheckWithPolicy(ctx, agyPath, "1.4.12", "exact"); err == nil {
		t.Errorf("expected exact 1.4.12 to fail on agy 1.5.0, but it passed")
	}
}

