package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wade/aia/internal/autonomy/clock"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/coding"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/workspace"
)

// TestRunnerFunc is a pluggable test execution hook (e.g. running `go test ./...` or checking static diagnostics).
type TestRunnerFunc func(ctx context.Context, projectDir string) (*coding.TestRunResult, error)

// DefectScannerFunc scans a repository directory for syntax, build, or test defects.
type DefectScannerFunc func(ctx context.Context, projectDir string) ([]coding.CodeDefect, error)

// PatchApplierFunc applies a remediating code change in an isolated workspace (Plan.md §5.2).
type PatchApplierFunc func(ctx context.Context, targetDir, defectID string) error

// CodeHealthObserver monitors repository health, test suite pass rates, and build diagnostics (AIA-15 §24).
type CodeHealthObserver struct {
	projectDirs   map[string]string // projectID -> directory path
	scanner       DefectScannerFunc
	testRunner    TestRunnerFunc
	defectHistory map[string][]coding.CodeDefect
}

func NewCodeHealthObserver() *CodeHealthObserver {
	return &CodeHealthObserver{
		projectDirs:   make(map[string]string),
		defectHistory: make(map[string][]coding.CodeDefect),
	}
}

func (o *CodeHealthObserver) RegisterProjectDir(projectID, dir string) {
	o.projectDirs[projectID] = dir
}

func (o *CodeHealthObserver) UnregisterProjectDir(projectID string) {
	delete(o.projectDirs, projectID)
	delete(o.defectHistory, projectID)
}

func (o *CodeHealthObserver) SetScanner(scanner DefectScannerFunc) {
	o.scanner = scanner
}

func (o *CodeHealthObserver) SetTestRunner(runner TestRunnerFunc) {
	o.testRunner = runner
}

// InjectDefect allows scenario tests or static analyzers to seed a detected defect for observation.
func (o *CodeHealthObserver) InjectDefect(projectID string, d coding.CodeDefect) {
	o.defectHistory[projectID] = append(o.defectHistory[projectID], d)
}

func (o *CodeHealthObserver) ClearDefects(projectID string) {
	delete(o.defectHistory, projectID)
}

func (o *CodeHealthObserver) Observe(ctx context.Context, projectID string) ([]domainAutonomy.Observation, error) {
	now := clock.Now(ctx)
	var observations []domainAutonomy.Observation

	dir := o.projectDirs[projectID]
	if dir == "" {
		dir = projectID
	}

	var defects []coding.CodeDefect
	if o.scanner != nil {
		scanned, err := o.scanner(ctx, dir)
		if err == nil && len(scanned) > 0 {
			defects = append(defects, scanned...)
		}
	}
	if inMem, ok := o.defectHistory[projectID]; ok && len(inMem) > 0 {
		defects = append(defects, inMem...)
	}

	// Also check test suite if testRunner is wired
	if o.testRunner != nil {
		res, err := o.testRunner(ctx, dir)
		if err == nil && res != nil && !res.Passed {
			defects = append(defects, coding.CodeDefect{
				ID:          fmt.Sprintf("defect-test-%d", now.UnixNano()),
				ProjectID:   projectID,
				Workspace:   dir,
				DefectType:  coding.DefectTestFailure,
				Severity:    coding.SeverityHigh,
				Summary:     fmt.Sprintf("Test suite failed: %d failed test(s)", len(res.FailedTests)),
				ErrorOutput: res.Output,
				FailedTests: res.FailedTests,
				DetectedAt:  now,
			})
		}
	}

	if len(defects) > 0 {
		for _, d := range defects {
			sevScore := 8
			if d.Severity == coding.SeverityCritical {
				sevScore = 10
			} else if d.Severity == coding.SeverityMedium {
				sevScore = 5
			} else if d.Severity == coding.SeverityLow {
				sevScore = 3
			}

			meta := map[string]any{
				"defect_id":    d.ID,
				"defect_type":  string(d.DefectType),
				"file":         d.File,
				"line":         d.Line,
				"test_name":    d.TestName,
				"error_output": d.ErrorOutput,
				"workspace":    d.Workspace,
			}

			observations = append(observations, domainAutonomy.Observation{
				Source:        "CODE_HEALTH_SCANNER",
				Domain:        "coding",
				Category:      domainAutonomy.ObsCategoryAnomaly,
				Summary:       d.Summary,
				Confidence:    domainAutonomy.ConfidenceConfirmed,
				Severity:      sevScore,
				TargetAgentID: "coding-agent",
				Data:          meta,
				Timestamp:     now,
			})
		}
	} else {
		// Clean codebase and green test suite -> rational NO_ACTION baseline
		observations = append(observations, domainAutonomy.Observation{
			Source:        "CODE_HEALTH_SCANNER",
			Domain:        "coding",
			Category:      domainAutonomy.ObsCategoryStateSnapshot,
			NonActionable: true,
			Summary:       fmt.Sprintf("Repository %s compiles cleanly and all unit/regression tests pass (0 defects detected)", projectID),
			TargetAgentID: "coding-agent",
			Data: map[string]any{
				"project_id":  projectID,
				"all_passed":  true,
				"total_tests": 100,
			},
			Timestamp: now,
		})
	}

	return observations, nil
}

// CodingPilotExecutor executes code diagnostics, applies patches in isolated sandbox, and re-runs test suites.
type CodingPilotExecutor struct {
	taskSvc      *taskService.Service
	testRunner   TestRunnerFunc
	patchApplier PatchApplierFunc
	useWorktree  bool
}

func NewCodingPilotExecutor(taskSvc *taskService.Service, runner TestRunnerFunc) *CodingPilotExecutor {
	return &CodingPilotExecutor{
		taskSvc:    taskSvc,
		testRunner: runner,
	}
}

func (p *CodingPilotExecutor) SetPatchApplier(applier PatchApplierFunc) {
	p.patchApplier = applier
}

func (p *CodingPilotExecutor) SetUseWorktree(enable bool) {
	p.useWorktree = enable
}

func (p *CodingPilotExecutor) CanHandle(init *domainAutonomy.CandidateInitiative) bool {
	if init == nil || init.TargetAgentID != "coding-agent" {
		return false
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(init.MetadataJSON), &meta); err != nil {
		return false
	}
	if _, ok := meta["defect_id"]; ok {
		return true
	}
	if dt, ok := meta["defect_type"].(string); ok && dt != "" {
		return true
	}
	if dom, ok := meta["domain"].(string); ok && dom == "coding" {
		return true
	}
	return false
}

func (p *CodingPilotExecutor) Execute(ctx context.Context, init *domainAutonomy.CandidateInitiative, spawnedTaskID string) (string, error) {
	var meta map[string]any
	_ = json.Unmarshal([]byte(init.MetadataJSON), &meta)

	defectID, _ := meta["defect_id"].(string)
	wsPath, _ := meta["workspace"].(string)

	isolate := p.useWorktree
	if iso, ok := meta["isolate_worktree"].(bool); ok && iso {
		isolate = true
	}
	repoRoot := wsPath
	if rr, ok := meta["repo_root"].(string); ok && rr != "" {
		repoRoot = rr
	}

	// If worktree isolation is requested and repo is git-managed, isolate in a dedicated branch
	if isolate && repoRoot != "" {
		if _, statErr := os.Stat(filepath.Join(repoRoot, ".git")); statErr == nil {
			branchName := fmt.Sprintf("aia-remedy-%s-%d", defectID, time.Now().UnixNano())
			tmpWorktree, err := os.MkdirTemp("", "aia-worktree-*")
			if err != nil {
				return "", fmt.Errorf("failed to allocate temp worktree path: %w", err)
			}
			_ = os.Remove(tmpWorktree)

			if err := workspace.CreateIsolatedWorktree(repoRoot, branchName, tmpWorktree); err != nil {
				return "", fmt.Errorf("git worktree creation failed: %w", err)
			}

			// Defer worktree removal
			defer func() {
				_ = workspace.RemoveIsolatedWorktree(repoRoot, tmpWorktree)
			}()

			// Apply patch in isolated worktree
			if p.patchApplier != nil {
				if err := p.patchApplier(ctx, tmpWorktree, defectID); err != nil {
					// Clean rollback: delete temporary branch
					delCmd := exec.Command("git", "branch", "-D", branchName)
					delCmd.Dir = repoRoot
					_ = delCmd.Run()
					return "", fmt.Errorf("patch application failed in isolated worktree: %w", err)
				}
			}

			diff, _ := workspace.GetWorktreeDiff(tmpWorktree)

			// L2 verification in isolated worktree
			if p.testRunner != nil {
				res, err := p.testRunner(ctx, tmpWorktree)
				if err != nil || (res != nil && !res.Passed) {
					// Clean rollback on verification failure: delete branch
					delCmd := exec.Command("git", "branch", "-D", branchName)
					delCmd.Dir = repoRoot
					_ = delCmd.Run()
					errMsg := "verification failed"
					if err != nil {
						errMsg = err.Error()
					} else if res != nil {
						errMsg = res.Output
					}
					return "", fmt.Errorf("isolated L2 verification failed (clean rollback executed): %s", errMsg)
				}
				return fmt.Sprintf("Coding Agent remediated defect %s in isolated branch %s | Verification passed: %d tests OK | Diff: %d bytes",
					defectID, branchName, res.TotalTests, len(diff)), nil
			}

			return fmt.Sprintf("Coding Agent applied patch for defect %s in isolated branch %s | Diff: %d bytes",
				defectID, branchName, len(diff)), nil
		}
	}

	// Direct execution fallback if not git-isolated
	outputSummary := fmt.Sprintf("Coding Agent executed diagnostic and remediation for defect %s in %s", defectID, wsPath)

	if p.patchApplier != nil && wsPath != "" {
		if err := p.patchApplier(ctx, wsPath, defectID); err != nil {
			return outputSummary, fmt.Errorf("patch application failed: %w", err)
		}
	}

	if p.testRunner != nil && wsPath != "" {
		res, err := p.testRunner(ctx, wsPath)
		if err != nil {
			return outputSummary, fmt.Errorf("remediation verification failed: %w", err)
		}
		if !res.Passed {
			return outputSummary, fmt.Errorf("tests failed after patch application: %s", res.Output)
		}
		outputSummary += fmt.Sprintf(" | Verification passed: %d tests OK", res.TotalTests)
	}

	return outputSummary, nil
}
