package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/provider"
	"github.com/wade/aia/internal/validation"
)

func TestValidatorSuccess(t *testing.T) {
	v := validation.NewDefaultValidator()
	req := validation.ValidationRequest{
		Task: task.Task{
			Title: "Test Task",
		},
		Execution: execution.Execution{
			Status: execution.StatusRunning,
		},
		Result: provider.ProviderResult{
			Status:   "SUCCESS",
			Response: "Code generated successfully",
		},
		Workspace: t.TempDir(),
	}

	res, err := v.Validate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if !res.Accepted {
		t.Errorf("expected validation to be accepted, got: %s", res.Status)
	}
}

func TestValidatorMissingFile(t *testing.T) {
	v := validation.NewDefaultValidator()
	tmpDir := t.TempDir()

	req := validation.ValidationRequest{
		Task: task.Task{
			Title:      "Test Task with file requirement",
			Acceptance: []string{"file: main.go"},
		},
		Execution: execution.Execution{
			Status: execution.StatusRunning,
		},
		Result: provider.ProviderResult{
			Status:   "SUCCESS",
			Response: "I created main.go",
		},
		Workspace: tmpDir,
	}

	// Case 1: file does not exist -> should be rejected
	res, err := v.Validate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Accepted {
		t.Errorf("expected rejection due to missing file")
	}

	// Case 2: create file -> should pass
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	res, err = v.Validate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Accepted {
		t.Errorf("expected acceptance when file exists")
	}
}
