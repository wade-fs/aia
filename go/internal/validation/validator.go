package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wade/aia/internal/domain/execution"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/provider"
)

type ValidationRequest struct {
	Task      task.Task
	Execution execution.Execution
	Result    provider.ProviderResult
	Workspace string
}

type Finding struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // "INFO", "WARNING", "ERROR"
}

type ValidationResult struct {
	Accepted bool      `json:"accepted"`
	Status   string    `json:"status"`
	Findings []Finding `json:"findings"`
}

type Validator interface {
	Validate(ctx context.Context, req ValidationRequest) (*ValidationResult, error)
}

type DefaultValidator struct{}

func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{}
}

func (v *DefaultValidator) Validate(ctx context.Context, req ValidationRequest) (*ValidationResult, error) {
	var findings []Finding

	// 1. Check provider result status
	if req.Result.Status != "SUCCESS" {
		findings = append(findings, Finding{
			Code:        "EXECUTION_FAILED",
			Description: fmt.Sprintf("provider returned status: %s (err: %s)", req.Result.Status, req.Result.Error),
			Severity:    "ERROR",
		})
	}

	// 2. Check if response is empty
	if strings.TrimSpace(req.Result.Response) == "" {
		findings = append(findings, Finding{
			Code:        "EMPTY_RESPONSE",
			Description: "agent returned an empty response",
			Severity:    "ERROR",
		})
	}

	// 3. Check acceptance criteria if defined
	for _, ac := range req.Task.Acceptance {
		acTrim := strings.TrimSpace(ac)
		if strings.HasPrefix(acTrim, "file:") {
			relPath := strings.TrimSpace(strings.TrimPrefix(acTrim, "file:"))
			fullPath := filepath.Join(req.Workspace, relPath)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				findings = append(findings, Finding{
					Code:        "MISSING_EXPECTED_FILE",
					Description: fmt.Sprintf("acceptance criteria file not found: %s", relPath),
					Severity:    "ERROR",
				})
			}
		}
	}

	// 4. Summarize acceptance
	hasError := false
	for _, f := range findings {
		if f.Severity == "ERROR" {
			hasError = true
			break
		}
	}

	status := "PASSED"
	if hasError {
		status = "REJECTED"
	}

	return &ValidationResult{
		Accepted: !hasError,
		Status:   status,
		Findings: findings,
	}, nil
}
