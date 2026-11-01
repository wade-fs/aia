package antigravity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wade/aia/internal/config"
)

var (
	// ErrOSSandboxUnavailable is returned when RequireOSNamespace is true (e.g. Level B/C or high-risk profiles)
	// and neither bubblewrap nor systemd user mount namespaces are operational on the current host.
	ErrOSSandboxUnavailable = errors.New("os-level mount namespace sandbox is required but unavailable on this host")
)

const (
	IsolationTierBwrap   = "OS_BWRAP_NAMESPACE"
	IsolationTierSystemd = "OS_SYSTEMD_MOUNT_NS"
	IsolationTierEnvOnly = "ENV_SCRUB_AND_WORKDIR"
)

// SandboxPlan describes the isolated OS command and scrubbed environment to execute.
type SandboxPlan struct {
	Executable    string
	Args          []string
	Env           []string
	IsolationTier string
}

var (
	defaultSafeEnvKeys = map[string]bool{
		"PATH":                     true,
		"HOME":                     true,
		"USER":                     true,
		"LOGNAME":                  true,
		"LANG":                     true,
		"LC_ALL":                   true,
		"LC_CTYPE":                 true,
		"TERM":                     true,
		"TMPDIR":                   true,
		"TZ":                       true,
		"XDG_CACHE_HOME":           true,
		"XDG_RUNTIME_DIR":          true,
		"DBUS_SESSION_BUS_ADDRESS": true,
		"SSL_CERT_FILE":            true,
		"SSL_CERT_DIR":             true,
		"HTTP_PROXY":               true,
		"HTTPS_PROXY":              true,
		"NO_PROXY":                 true,
	}

	sensitiveKeyPrefixes = []string{
		"AIA_",
		"DATABASE_",
		"FRED_",
		"FINMIND_",
		"ALPHA_VANTAGE_",
		"TELEGRAM_",
		"AWS_",
		"GCP_",
	}

	sensitiveKeySuffixes = []string{
		"_KEY",
		"_SECRET",
		"_TOKEN",
		"_PASSWORD",
		"_PASS",
		"_CREDENTIAL",
		"_CREDENTIALS",
		"_DSN",
	}

	probeOnce       sync.Once
	cachedBwrapOK   bool
	cachedSystemdOK bool

	// Hooks overridable in unit tests for deterministic sandbox planner verification.
	bwrapProber   = defaultProbeBwrap
	systemdProber = defaultProbeSystemdUser
)

// ScrubEnvironment constructs a strict whitelist environment for child harnesses,
// stripping all secrets from ~/.config/aia/service.env or daemon memory (security-model.md §4.2).
func ScrubEnvironment(parentEnv []string, extraAllowed []string) []string {
	allowed := make(map[string]bool, len(defaultSafeEnvKeys)+len(extraAllowed))
	for k := range defaultSafeEnvKeys {
		allowed[k] = true
	}
	for _, k := range extraAllowed {
		kUpper := strings.ToUpper(strings.TrimSpace(k))
		if kUpper != "" && !isSensitiveEnvKey(kUpper) {
			allowed[kUpper] = true
		}
	}

	var scrubbed []string
	hasPath := false
	for _, entry := range parentEnv {
		idx := strings.IndexByte(entry, '=')
		if idx <= 0 {
			continue
		}
		key := entry[:idx]
		val := entry[idx+1:]
		keyUpper := strings.ToUpper(key)

		if isSensitiveEnvKey(keyUpper) {
			continue
		}
		if keyUpper == "XDG_CONFIG_HOME" {
			// Allow XDG_CONFIG_HOME only if it does not expose ~/.config/aia directly
			if !strings.Contains(filepath.Clean(val), filepath.Join(".config", "aia")) {
				scrubbed = append(scrubbed, entry)
			}
			continue
		}
		if !allowed[keyUpper] {
			continue
		}
		if keyUpper == "PATH" {
			hasPath = true
		}
		scrubbed = append(scrubbed, entry)
	}

	if !hasPath {
		scrubbed = append(scrubbed, "PATH=/usr/local/bin:/usr/bin:/bin")
	}
	return scrubbed
}

func isSensitiveEnvKey(keyUpper string) bool {
	for _, p := range sensitiveKeyPrefixes {
		if strings.HasPrefix(keyUpper, p) {
			return true
		}
	}
	for _, s := range sensitiveKeySuffixes {
		if strings.HasSuffix(keyUpper, s) {
			return true
		}
	}
	return false
}

// BuildSandboxPlan prepares the scrubbed environment and wraps the child process in an OS mount namespace
// (bubblewrap or systemd-run --user) hiding ~/.config/aia and the SQLite data directory while mounting
// the root filesystem (including AIA Level B/C governance code) read-only.
func BuildSandboxPlan(
	ctx context.Context,
	cfg config.AntigravityConfig,
	executable string,
	args []string,
	workDir string,
	parentEnv []string,
) (*SandboxPlan, error) {
	if len(parentEnv) == 0 {
		parentEnv = os.Environ()
	}
	scrubbedEnv := ScrubEnvironment(parentEnv, cfg.AllowedEnvVars)

	mode := strings.ToLower(strings.TrimSpace(cfg.SandboxMode))
	if mode == "" {
		mode = "auto"
	}

	absWorkDir := workDir
	if absWorkDir != "" {
		if resolved, err := filepath.Abs(absWorkDir); err == nil {
			absWorkDir = resolved
		}
	}

	hiddenPaths := resolveHiddenPaths(cfg.HiddenPaths)

	switch mode {
	case "env_only":
		if cfg.RequireOSNamespace {
			return nil, fmt.Errorf("%w: sandbox_mode is set to env_only", ErrOSSandboxUnavailable)
		}
		return &SandboxPlan{
			Executable:    executable,
			Args:          args,
			Env:           scrubbedEnv,
			IsolationTier: IsolationTierEnvOnly,
		}, nil

	case "bwrap":
		if !bwrapProber(ctx) {
			return nil, fmt.Errorf("%w: bubblewrap (bwrap) namespace failed kernel capability probe", ErrOSSandboxUnavailable)
		}
		return buildBwrapPlan(executable, args, absWorkDir, hiddenPaths, cfg.ReadOnlyPaths, scrubbedEnv), nil

	case "systemd":
		if !systemdProber(ctx) {
			return nil, fmt.Errorf("%w: systemd-run --user mount namespace failed capability probe", ErrOSSandboxUnavailable)
		}
		return buildSystemdPlan(executable, args, absWorkDir, hiddenPaths, scrubbedEnv), nil

	case "strict":
		if bwrapProber(ctx) {
			return buildBwrapPlan(executable, args, absWorkDir, hiddenPaths, cfg.ReadOnlyPaths, scrubbedEnv), nil
		}
		if systemdProber(ctx) {
			return buildSystemdPlan(executable, args, absWorkDir, hiddenPaths, scrubbedEnv), nil
		}
		return nil, fmt.Errorf("%w: strict mode enabled and neither bwrap nor systemd-run --user is operational", ErrOSSandboxUnavailable)

	default: // "auto"
		if bwrapProber(ctx) {
			return buildBwrapPlan(executable, args, absWorkDir, hiddenPaths, cfg.ReadOnlyPaths, scrubbedEnv), nil
		}
		if systemdProber(ctx) {
			return buildSystemdPlan(executable, args, absWorkDir, hiddenPaths, scrubbedEnv), nil
		}
		if cfg.RequireOSNamespace {
			return nil, ErrOSSandboxUnavailable
		}
		return &SandboxPlan{
			Executable:    executable,
			Args:          args,
			Env:           scrubbedEnv,
			IsolationTier: IsolationTierEnvOnly,
		}, nil
	}
}

func buildBwrapPlan(
	executable string,
	args []string,
	absWorkDir string,
	hiddenPaths []string,
	extraReadOnly []string,
	env []string,
) *SandboxPlan {
	bwrapArgs := []string{
		"--die-with-parent",
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
	}

	// Mask secret & SQLite directories FIRST with empty tmpfs so service.env and aia.db are invisible
	for _, hp := range hiddenPaths {
		if info, err := os.Stat(hp); err == nil && info.IsDir() {
			bwrapArgs = append(bwrapArgs, "--tmpfs", hp)
		}
	}

	for _, ro := range extraReadOnly {
		if roClean := filepath.Clean(ro); roClean != "" {
			if _, err := os.Stat(roClean); err == nil {
				bwrapArgs = append(bwrapArgs, "--ro-bind", roClean, roClean)
			}
		}
	}

	// Allow Antigravity CLI state/cache directories when "/" is mounted read-only,
	// while keeping ~/.config/aia and ~/.local/share/aia completely masked above.
	for _, rw := range resolveHarnessWritableDirs() {
		bwrapArgs = append(bwrapArgs, "--bind", rw, rw)
	}

	// Re-mount ONLY the task's isolated WorkDir read-write ON TOP OF any masked parent directory
	if absWorkDir != "" {
		bwrapArgs = append(bwrapArgs, "--dir", absWorkDir, "--bind", absWorkDir, absWorkDir, "--chdir", absWorkDir)
	}

	bwrapArgs = append(bwrapArgs, "--", executable)
	bwrapArgs = append(bwrapArgs, args...)

	return &SandboxPlan{
		Executable:    "bwrap",
		Args:          bwrapArgs,
		Env:           env,
		IsolationTier: IsolationTierBwrap,
	}
}

func buildSystemdPlan(
	executable string,
	args []string,
	absWorkDir string,
	hiddenPaths []string,
	env []string,
) *SandboxPlan {
	sysArgs := []string{
		"--user",
		"--pipe",
		"--wait",
		"--collect",
		"--quiet",
		"-p", "NoNewPrivileges=yes",
		"-p", "ReadOnlyPaths=/",
		"-p", "ReadWritePaths=/tmp",
	}

	for _, hp := range hiddenPaths {
		if info, err := os.Stat(hp); err == nil && info.IsDir() {
			// Use TemporaryFileSystem so nested WorkDir bind mounts still succeed cleanly
			sysArgs = append(sysArgs, "-p", fmt.Sprintf("TemporaryFileSystem=%s", hp))
		}
	}

	for _, rw := range resolveHarnessWritableDirs() {
		sysArgs = append(sysArgs, "-p", fmt.Sprintf("ReadWritePaths=%s", rw))
	}

	if absWorkDir != "" {
		sysArgs = append(sysArgs,
			"-p", fmt.Sprintf("BindPaths=%s", absWorkDir),
			"-p", fmt.Sprintf("ReadWritePaths=%s", absWorkDir),
			fmt.Sprintf("--working-directory=%s", absWorkDir),
		)
	}

	sysArgs = append(sysArgs, "--", executable)
	sysArgs = append(sysArgs, args...)

	return &SandboxPlan{
		Executable:    "systemd-run",
		Args:          sysArgs,
		Env:           env,
		IsolationTier: IsolationTierSystemd,
	}
}

func resolveHiddenPaths(configured []string) []string {
	if len(configured) > 0 {
		out := make([]string, 0, len(configured))
		for _, p := range configured {
			exp := config.ExpandPath(strings.TrimSpace(p))
			if exp != "" {
				if abs, err := filepath.Abs(exp); err == nil {
					out = append(out, abs)
				} else {
					out = append(out, filepath.Clean(exp))
				}
			}
		}
		return out
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "aia"),
		filepath.Join(home, ".local", "share", "aia"),
	}
}

func resolveHarnessWritableDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	var out []string
	for _, rel := range []string{".gemini", ".cache"} {
		cand := filepath.Join(home, rel)
		if info, sErr := os.Stat(cand); sErr == nil && info.IsDir() {
			out = append(out, cand)
		}
	}
	return out
}

func defaultProbeBwrap(ctx context.Context) bool {
	initProbes()
	return cachedBwrapOK
}

func defaultProbeSystemdUser(ctx context.Context) bool {
	initProbes()
	return cachedSystemdOK
}

func initProbes() {
	probeOnce.Do(func() {
		pCtx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()

		if bwrapBin, err := exec.LookPath("bwrap"); err == nil && bwrapBin != "" {
			if trueBin, tErr := exec.LookPath("true"); tErr == nil {
				cmd := exec.CommandContext(pCtx, bwrapBin, "--ro-bind", "/", "/", "--dev", "/dev", "--proc", "/proc", "--tmpfs", "/tmp", "--", trueBin)
				if cmd.Run() == nil {
					cachedBwrapOK = true
				}
			}
		}

		if sysBin, err := exec.LookPath("systemd-run"); err == nil && sysBin != "" {
			if trueBin, tErr := exec.LookPath("true"); tErr == nil {
				// Must probe mount-namespace flags (ReadOnlyPaths=/, TemporaryFileSystem=/tmp)
				// because unprivileged containers allow systemd-run --user without mount namespaces
				// while rejecting CLONE_NEWNS / ReadOnlyPaths with exit status 226/NAMESPACE.
				cmd := exec.CommandContext(
					pCtx, sysBin,
					"--user", "--pipe", "--wait", "--collect", "--quiet",
					"-p", "NoNewPrivileges=yes",
					"-p", "ReadOnlyPaths=/",
					"-p", "TemporaryFileSystem=/tmp",
					"--", trueBin,
				)
				if cmd.Run() == nil {
					cachedSystemdOK = true
				}
			}
		}
	})
}

// SetProbersForTest overrides kernel sandbox probers during unit tests and returns a restore function.
func SetProbersForTest(bwrapOK, systemdOK bool) func() {
	prevBwrap := bwrapProber
	prevSys := systemdProber
	bwrapProber = func(context.Context) bool { return bwrapOK }
	systemdProber = func(context.Context) bool { return systemdOK }
	return func() {
		bwrapProber = prevBwrap
		systemdProber = prevSys
	}
}
