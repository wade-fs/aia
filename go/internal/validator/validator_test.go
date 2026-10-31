package validator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wade/aia/agent-runtime/internal/git"
	"github.com/wade/aia/agent-runtime/internal/validator"
	"github.com/wade/aia/agent-runtime/internal/worker"
)

func setupTestWorkspace(t *testing.T) (string, string) {
	wsDir, err := os.MkdirTemp("", "agent-validator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create sample go project
	goMod := "module testws\n\ngo 1.21\n"
	_ = os.WriteFile(filepath.Join(wsDir, "go.mod"), []byte(goMod), 0644)

	mainCode := "package main\n\nfunc Answer() int { return 42 }\n"
	_ = os.WriteFile(filepath.Join(wsDir, "main.go"), []byte(mainCode), 0644)

	testCode := "package main\n\nimport \"testing\"\n\nfunc TestAnswer(t *testing.T) { if Answer() != 42 { t.Fail() } }\n"
	_ = os.WriteFile(filepath.Join(wsDir, "main_test.go"), []byte(testCode), 0644)

	gitClient := git.NewClient(wsDir)
	_ = gitClient.Init()
	_ = gitClient.AddAndCommit("initial commit")
	baseCommit, _ := gitClient.GetCurrentCommit()

	return wsDir, baseCommit
}

func TestValidatorSuccess(t *testing.T) {
	wsDir, baseCommit := setupTestWorkspace(t)
	defer os.RemoveAll(wsDir)

	// Simulate worker modifying a file
	extraFile := filepath.Join(wsDir, "extra.go")
	_ = os.WriteFile(extraFile, []byte("package main\n\nfunc Extra() {}\n"), 0644)

	v := validator.NewValidator("go test ./...")
	res := &worker.WorkerResult{
		ExecutionID: "exec-1",
		ExitCode:    0,
		Stdout:      "success",
	}

	valRes, err := v.Validate(context.Background(), res, wsDir, baseCommit)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if !valRes.Passed {
		t.Fatalf("Expected validation to PASS, got summary: %s", valRes.Summary)
	}
	if !valRes.WorkerExitOK || !valRes.GitDiffExists || !valRes.TestPassed {
		t.Errorf("Unexpected validation flags: %+v", valRes)
	}
}

func TestValidatorWorkerFailed(t *testing.T) {
	wsDir, baseCommit := setupTestWorkspace(t)
	defer os.RemoveAll(wsDir)

	v := validator.NewValidator("go test ./...")
	res := &worker.WorkerResult{
		ExecutionID: "exec-2",
		ExitCode:    1,
		Stderr:      "error occurred",
	}

	valRes, err := v.Validate(context.Background(), res, wsDir, baseCommit)
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}

	if valRes.Passed {
		t.Fatalf("Expected validation to FAIL due to worker exit code 1")
	}
	if valRes.WorkerExitOK {
		t.Errorf("WorkerExitOK should be false")
	}
}

func TestValidatorTestFailed(t *testing.T) {
	wsDir, baseCommit := setupTestWorkspace(t)
	defer os.RemoveAll(wsDir)

	// Break the test
	breakCode := "package main\n\nfunc Answer() int { return 999 }\n"
	_ = os.WriteFile(filepath.Join(wsDir, "main.go"), []byte(breakCode), 0644)

	v := validator.NewValidator("go test ./...")
	res := &worker.WorkerResult{
		ExecutionID: "exec-3",
		ExitCode:    0,
	}

	valRes, err := v.Validate(context.Background(), res, wsDir, baseCommit)
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}

	if valRes.Passed {
		t.Fatalf("Expected validation to FAIL due to failing go test")
	}
	if valRes.TestPassed {
		t.Errorf("TestPassed should be false")
	}
}
