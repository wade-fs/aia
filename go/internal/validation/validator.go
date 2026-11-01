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
	trimmedResp := strings.TrimSpace(req.Result.Response)
	if trimmedResp == "" {
		findings = append(findings, Finding{
			Code:        "EMPTY_RESPONSE",
			Description: "agent returned an empty response",
			Severity:    "ERROR",
		})
	}

	// 2b. Golden Rule (Plan §0.3 Rule 7): Never trust exit code 0 alone.
	// Detect silent provider errors, auth expiration, quota exhaustion, or unhandled panic traces disguised inside Status=SUCCESS.
	if strings.TrimSpace(req.Result.Error) != "" {
		findings = append(findings, Finding{
			Code:        "NON_EMPTY_ERROR_ON_SUCCESS",
			Description: fmt.Sprintf("provider reported error alongside exit status %s: %s", req.Result.Status, req.Result.Error),
			Severity:    "ERROR",
		})
	}
	lowerResp := strings.ToLower(trimmedResp)
	silentFailureMarkers := []string{
		"authentication failed",
		"quota exceeded",
		"quota reached",
		"resource_exhausted",
		"rate limit exceeded: fatal",
		"panic: runtime error",
		"fatal: not logged in",
	}
	for _, marker := range silentFailureMarkers {
		if strings.Contains(lowerResp, marker) {
			findings = append(findings, Finding{
				Code:        "SILENT_HARNESS_FAILURE",
				Description: fmt.Sprintf("exit code 0 response contains fatal harness failure marker %q", marker),
				Severity:    "ERROR",
			})
			break
		}
	}

	// 3. Check acceptance criteria if defined
	for _, ac := range req.Task.Acceptance {
		acTrim := strings.TrimSpace(ac)
		switch {
		case strings.HasPrefix(acTrim, "file_non_empty:"):
			relPath := strings.TrimSpace(strings.TrimPrefix(acTrim, "file_non_empty:"))
			fullPath := filepath.Join(req.Workspace, relPath)
			info, err := os.Stat(fullPath)
			if os.IsNotExist(err) {
				findings = append(findings, Finding{
					Code:        "MISSING_EXPECTED_FILE",
					Description: fmt.Sprintf("acceptance criteria file not found: %s", relPath),
					Severity:    "ERROR",
				})
			} else if err == nil {
				raw, rErr := os.ReadFile(fullPath)
				if info.Size() == 0 || (rErr == nil && strings.TrimSpace(string(raw)) == "") {
					findings = append(findings, Finding{
						Code:        "EMPTY_EXPECTED_FILE",
						Description: fmt.Sprintf("acceptance criteria file exists but is empty: %s", relPath),
						Severity:    "ERROR",
					})
				}
			}
		case strings.HasPrefix(acTrim, "file:"):
			relPath := strings.TrimSpace(strings.TrimPrefix(acTrim, "file:"))
			fullPath := filepath.Join(req.Workspace, relPath)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				findings = append(findings, Finding{
					Code:        "MISSING_EXPECTED_FILE",
					Description: fmt.Sprintf("acceptance criteria file not found: %s", relPath),
					Severity:    "ERROR",
				})
			}
		case strings.HasPrefix(acTrim, "contains:"):
			expectedSub := strings.TrimSpace(strings.TrimPrefix(acTrim, "contains:"))
			if expectedSub != "" && !strings.Contains(req.Result.Response, expectedSub) {
				findings = append(findings, Finding{
					Code:        "MISSING_EXPECTED_CONTENT",
					Description: fmt.Sprintf("response missing required substring %q", expectedSub),
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
