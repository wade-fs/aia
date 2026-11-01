package coding

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

type DefectType string

const (
	DefectTestFailure   DefectType = "TEST_FAILURE"
	DefectBuildError    DefectType = "BUILD_ERROR"
	DefectRegression    DefectType = "REGRESSION"
	DefectTechnicalDebt DefectType = "TECHNICAL_DEBT"
)

type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "CRITICAL"
	SeverityHigh     SeverityLevel = "HIGH"
	SeverityMedium   SeverityLevel = "MEDIUM"
	SeverityLow      SeverityLevel = "LOW"
)

// CodeDefect represents a detected flaw, compilation error, or broken test in a repository.
type CodeDefect struct {
	ID          string        `json:"id"`
	ProjectID   string        `json:"project_id"`
	Workspace   string        `json:"workspace"`
	DefectType  DefectType    `json:"defect_type"`
	Severity    SeverityLevel `json:"severity"`
	File        string        `json:"file,omitempty"`
	Line        int           `json:"line,omitempty"`
	Summary     string        `json:"summary"`
	ErrorOutput string        `json:"error_output,omitempty"`
	TestName    string        `json:"test_name,omitempty"`
	FailedTests []string      `json:"failed_tests,omitempty"`
	DetectedAt  time.Time     `json:"detected_at"`
}

// TestRunResult captures the empirical output of running a test suite (e.g. `go test`).
type TestRunResult struct {
	Passed       bool          `json:"passed"`
	ExitCode     int           `json:"exit_code"`
	TotalTests   int           `json:"total_tests"`
	FailedTests  []string      `json:"failed_tests,omitempty"`
	Duration     time.Duration `json:"duration"`
	Output       string        `json:"output"`
	ExecutedAt   time.Time     `json:"executed_at"`
}

// CodePatchProposal represents a proposed fix to be evaluated and applied in sandbox.
type CodePatchProposal struct {
	ID                 domain.ID `json:"id"`
	DefectID           string    `json:"defect_id"`
	TargetFile         string    `json:"target_file"`
	Diff               string    `json:"diff"`
	Rationale          string    `json:"rationale"`
	ExpectedOutcome    string    `json:"expected_outcome"`
	RegressionCheckCmd string    `json:"regression_check_cmd"`
	CreatedAt          time.Time `json:"created_at"`
}
