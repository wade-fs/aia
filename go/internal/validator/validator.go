package validator

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wade/aia/agent-runtime/internal/git"
	"github.com/wade/aia/agent-runtime/internal/worker"
)

type ValidationResult struct {
	Passed           bool     `json:"passed"`
	WorkerExitOK     bool     `json:"worker_exit_ok"`
	GitDiffExists    bool     `json:"git_diff_exists"`
	FilesInWorkspace bool     `json:"files_in_workspace"`
	UnexpectedFiles  []string `json:"unexpected_files,omitempty"`
	TestPassed       bool     `json:"test_passed"`
	Summary          string   `json:"summary"`
}

type Validator struct {
	testCmd string
}

func NewValidator(testCmd string) *Validator {
	if testCmd == "" {
		testCmd = "go test ./..."
	}
	return &Validator{testCmd: testCmd}
}

func (v *Validator) Validate(ctx context.Context, result *worker.WorkerResult, workspacePath, baseCommit string) (*ValidationResult, error) {
	vr := &ValidationResult{
		FilesInWorkspace: true,
		UnexpectedFiles:  []string{},
	}
	var failureReasons []string

	// 1. Check Worker Exit Code
	if result != nil && result.ExitCode == 0 {
		vr.WorkerExitOK = true
	} else {
		vr.WorkerExitOK = false
		if result == nil {
			failureReasons = append(failureReasons, "Worker execution produced no result")
		} else {
			failureReasons = append(failureReasons, fmt.Sprintf("Worker exit code %d != 0", result.ExitCode))
		}
	}

	// 2. Check Git Diff
	gitClient := git.NewClient(workspacePath)
	diff, err := gitClient.GetDiff(baseCommit)
	if err == nil && len(strings.TrimSpace(diff)) > 0 {
		vr.GitDiffExists = true
	} else {
		// Check porcelain for untracked files
		hasUncommitted, _ := gitClient.HasUncommittedChanges()
		if hasUncommitted {
			vr.GitDiffExists = true
		} else {
			vr.GitDiffExists = false
			failureReasons = append(failureReasons, "No git diff or file changes found")
		}
	}

	// 3. Check modified files (must be inside workspace)
	changedFiles, err := gitClient.GetChangedFiles(baseCommit)
	if err == nil {
		for _, file := range changedFiles {
			cleanPath := filepath.Clean(file)
			if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
				vr.FilesInWorkspace = false
				vr.UnexpectedFiles = append(vr.UnexpectedFiles, file)
			}
		}
	}
	if !vr.FilesInWorkspace || len(vr.UnexpectedFiles) > 0 {
		failureReasons = append(failureReasons, fmt.Sprintf("Unexpected files modified: %s", strings.Join(vr.UnexpectedFiles, ", ")))
	}

	// 4. Run test command in workspace
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	parts := strings.Fields(v.testCmd)
	if len(parts) > 0 {
		cmd := exec.CommandContext(testCtx, parts[0], parts[1:]...)
		cmd.Dir = workspacePath
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		testErr := cmd.Run()
		if testErr == nil {
			vr.TestPassed = true
		} else {
			vr.TestPassed = false
			failureReasons = append(failureReasons, fmt.Sprintf("Test command failed: %s (stdout: %s, stderr: %s)",
				testErr, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String())))
		}
	} else {
		vr.TestPassed = true
	}

	// 5. Final verdict
	vr.Passed = vr.WorkerExitOK && vr.GitDiffExists && vr.FilesInWorkspace && len(vr.UnexpectedFiles) == 0 && vr.TestPassed

	if vr.Passed {
		vr.Summary = "Validation PASSED: Worker succeeded, git diff verified, all files within workspace, and tests passed."
	} else {
		vr.Summary = fmt.Sprintf("Validation FAILED: %s", strings.Join(failureReasons, "; "))
	}

	return vr, nil
}
