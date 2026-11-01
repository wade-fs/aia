package unit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wade/aia/internal/config"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/policy"
	"github.com/wade/aia/internal/provider"
	"github.com/wade/aia/internal/provider/antigravity"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/validation"
)

func TestScrubEnvironment_StripsServiceEnvSecrets(t *testing.T) {
	parentEnv := []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=/home/tester",
		"LANG=en_US.UTF-8",
		"AIA_TELEGRAM_BOT_TOKEN=secret-tg-token",
		"DATABASE_URL=sqlite:///home/tester/.local/share/aia/aia.db",
		"FRED_API_KEY=fred-secret-123",
		"FINMIND_TOKEN=finmind-secret-456",
		"SIGNING_KEY_PASSWORD=super-secret-pass",
		"AWS_SECRET_ACCESS_KEY=aws-secret",
		"XDG_CONFIG_HOME=/home/tester/.config/aia",
		"CUSTOM_HARNESS_FLAG=enabled",
	}

	// Even if FRED_API_KEY is mistakenly listed in extraAllowed, sensitive key prefixes/suffixes MUST block it.
	scrubbed := antigravity.ScrubEnvironment(parentEnv, []string{"CUSTOM_HARNESS_FLAG", "FRED_API_KEY"})
	joined := strings.Join(scrubbed, "\n")

	for _, forbidden := range []string{
		"AIA_TELEGRAM_BOT_TOKEN",
		"DATABASE_URL",
		"FRED_API_KEY",
		"FINMIND_TOKEN",
		"SIGNING_KEY_PASSWORD",
		"AWS_SECRET_ACCESS_KEY",
		".config/aia",
	} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("expected ScrubEnvironment to strip %q, but found in: %s", forbidden, joined)
		}
	}

	for _, expected := range []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=/home/tester",
		"CUSTOM_HARNESS_FLAG=enabled",
	} {
		if !strings.Contains(joined, expected) {
			t.Errorf("expected ScrubEnvironment to retain safe variable %q, got: %s", expected, joined)
		}
	}
}

func TestBuildSandboxPlan_BwrapSystemdAndFailClosed(t *testing.T) {
	ctx := context.Background()
	rootTmp := t.TempDir()
	cfgDir := filepath.Join(rootTmp, ".config", "aia")
	dataDir := filepath.Join(rootTmp, ".local", "share", "aia")
	workDir := filepath.Join(dataDir, "workspaces", "exec-42")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}

	baseCfg := config.AntigravityConfig{
		Executable:  "/usr/bin/agy",
		SandboxMode: "auto",
		HiddenPaths: []string{cfgDir, dataDir},
	}

	// 1. Bwrap mount order: mask ~/.config/aia and ~/.local/share/aia BEFORE binding WorkDir RW
	t.Run("BwrapMasksDataDirBeforeBindingNestedWorkDir", func(t *testing.T) {
		restore := antigravity.SetProbersForTest(true, false)
		defer restore()

		plan, err := antigravity.BuildSandboxPlan(ctx, baseCfg, "/usr/bin/agy", []string{"--output-format", "stream-json"}, workDir, []string{"PATH=/usr/bin"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.IsolationTier != antigravity.IsolationTierBwrap || plan.Executable != "bwrap" {
			t.Fatalf("expected bwrap plan, got tier=%s exe=%s", plan.IsolationTier, plan.Executable)
		}

		argsStr := strings.Join(plan.Args, " ")
		roIdx := strings.Index(argsStr, "--ro-bind / /")
		maskCfgIdx := strings.Index(argsStr, "--tmpfs "+cfgDir)
		maskDataIdx := strings.Index(argsStr, "--tmpfs "+dataDir)
		bindWorkIdx := strings.Index(argsStr, "--bind "+workDir+" "+workDir)

		if roIdx < 0 || maskCfgIdx < 0 || maskDataIdx < 0 || bindWorkIdx < 0 {
			t.Fatalf("missing required bwrap flags in: %s", argsStr)
		}
		if !(roIdx < maskDataIdx && maskDataIdx < bindWorkIdx) {
			t.Fatalf("CRITICAL mount order violation: DataDir tmpfs mask (%d) must precede nested WorkDir RW bind (%d): %s",
				maskDataIdx, bindWorkIdx, argsStr)
		}
	})

	// 2. Systemd fallback when bwrap is unavailable
	t.Run("SystemdMountNamespaceFallback", func(t *testing.T) {
		restore := antigravity.SetProbersForTest(false, true)
		defer restore()

		plan, err := antigravity.BuildSandboxPlan(ctx, baseCfg, "/usr/bin/agy", []string{"--version"}, workDir, []string{"PATH=/usr/bin"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.IsolationTier != antigravity.IsolationTierSystemd || plan.Executable != "systemd-run" {
			t.Fatalf("expected systemd-run plan, got tier=%s exe=%s", plan.IsolationTier, plan.Executable)
		}
		argsStr := strings.Join(plan.Args, " ")
		if !strings.Contains(argsStr, "ReadOnlyPaths=/") || !strings.Contains(argsStr, "TemporaryFileSystem="+dataDir) || !strings.Contains(argsStr, "BindPaths="+workDir) {
			t.Fatalf("missing systemd mount isolation properties in: %s", argsStr)
		}
	})

	// 3. Fail-Closed when RequireOSNamespace is true and no OS namespace backend is operational
	t.Run("FailClosedWhenOSNamespaceRequiredAndUnavailable", func(t *testing.T) {
		restore := antigravity.SetProbersForTest(false, false)
		defer restore()

		strictCfg := baseCfg
		strictCfg.RequireOSNamespace = true
		_, err := antigravity.BuildSandboxPlan(ctx, strictCfg, "/usr/bin/agy", nil, workDir, nil)
		if !errors.Is(err, antigravity.ErrOSSandboxUnavailable) {
			t.Fatalf("expected ErrOSSandboxUnavailable in fail-closed mode, got %v", err)
		}
	})
}

func TestDefaultValidator_NeverTrustsExitCodeZeroAlone(t *testing.T) {
	ctx := context.Background()
	v := validation.NewDefaultValidator()
	ws := t.TempDir()

	hasFinding := func(res *validation.ValidationResult, code string) bool {
		for _, f := range res.Findings {
			if f.Code == code {
				return true
			}
		}
		return false
	}

	// 1. Exit code 0 ("SUCCESS") + non-empty stderr/result error MUST fail
	rep1, err := v.Validate(ctx, validation.ValidationRequest{
		Workspace: ws,
		Result: provider.ProviderResult{
			ConversationID: "conv-1",
			Status:         "SUCCESS",
			Response:       "Completed task.",
			Error:          "Error: SQLITE_BUSY database is locked",
		},
	})
	if err != nil || rep1.Accepted || !hasFinding(rep1, "NON_EMPTY_ERROR_ON_SUCCESS") {
		t.Fatalf("expected NON_EMPTY_ERROR_ON_SUCCESS, got accepted=%v findings=%+v err=%v", rep1.Accepted, rep1.Findings, err)
	}

	// 2. Exit code 0 ("SUCCESS") + silent quota/auth/panic text in response MUST fail
	rep2, err := v.Validate(ctx, validation.ValidationRequest{
		Workspace: ws,
		Result: provider.ProviderResult{
			ConversationID: "conv-2",
			Status:         "SUCCESS",
			Response:       "Sorry, RESOURCE_EXHAUSTED: quota reached for model.",
		},
	})
	if err != nil || rep2.Accepted || !hasFinding(rep2, "SILENT_HARNESS_FAILURE") {
		t.Fatalf("expected SILENT_HARNESS_FAILURE, got accepted=%v findings=%+v err=%v", rep2.Accepted, rep2.Findings, err)
	}

	// 3. file_non_empty:<rel> rejects whitespace-only artifact and accepts non-empty content
	emptyFile := filepath.Join(ws, "report.md")
	_ = os.WriteFile(emptyFile, []byte("   \n"), 0644)
	rep3, err := v.Validate(ctx, validation.ValidationRequest{
		Workspace: ws,
		Task:      task.Task{Acceptance: []string{"file_non_empty:report.md"}},
		Result: provider.ProviderResult{
			ConversationID: "conv-3",
			Status:         "SUCCESS",
			Response:       "Wrote report.md",
		},
	})
	if err != nil || rep3.Accepted || !hasFinding(rep3, "EMPTY_EXPECTED_FILE") {
		t.Fatalf("expected EMPTY_EXPECTED_FILE on whitespace-only file, got accepted=%v findings=%+v err=%v", rep3.Accepted, rep3.Findings, err)
	}

	_ = os.WriteFile(emptyFile, []byte("# Verified Analysis\nBrier=0.14\n"), 0644)
	rep4, err := v.Validate(ctx, validation.ValidationRequest{
		Workspace: ws,
		Task:      task.Task{Acceptance: []string{"file_non_empty:report.md", "contains:PONG"}},
		Result: provider.ProviderResult{
			ConversationID: "conv-4",
			Status:         "SUCCESS",
			Response:       "Wrote verified analysis with PONG token",
		},
	})
	if err != nil || !rep4.Accepted {
		t.Fatalf("expected validation to pass, got accepted=%v findings=%+v err=%v", rep4.Accepted, rep4.Findings, err)
	}
}

func TestHardResourceBudgetGate_PreFlightTokensAndCycles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	auditRepo := repository.NewAuditRepository(db)

	eval := policy.NewEvaluatorWithOptions(auditRepo, nil, 1000)
	eval.SetBudgetLimits(policy.BudgetLimits{
		MaxDailyTokens: 10000,
		MaxDailyCycles: 5,
	})

	okDec := eval.EvaluateResourceBudget(ctx, "coding-agent", 4200, 2)
	if !okDec.Allowed {
		t.Fatalf("expected within-budget check to pass, got %s", okDec.Reason)
	}

	tokBlocked := eval.EvaluateResourceBudget(ctx, "coding-agent", 10000, 2)
	if tokBlocked.Allowed || tokBlocked.GovernanceAction != governance.GovernanceActionBlocked {
		t.Fatalf("expected hard block when dailyTokens == MaxDailyTokens, got %+v", tokBlocked)
	}

	cycBlocked := eval.EvaluateResourceBudget(ctx, "autonomy-coordinator", 1200, 5)
	if cycBlocked.Allowed || cycBlocked.GovernanceAction != governance.GovernanceActionBlocked {
		t.Fatalf("expected hard block when dailyCycles == MaxDailyCycles, got %+v", cycBlocked)
	}

	logs, err := auditRepo.Query(ctx, repository.AuditFilter{Limit: 10})
	if err != nil || len(logs) < 2 {
		t.Fatalf("expected budget gate blocks to write immutable audit entries, got %d (err=%v)", len(logs), err)
	}
}
