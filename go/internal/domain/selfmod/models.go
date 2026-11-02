package selfmod

import (
	"errors"
	"time"
)

var (
	ErrLevelCModificationForbidden = errors.New("Level C governance/security modification is strictly forbidden: target paths are immutable")
	ErrProposalNotFound            = errors.New("self-modification proposal not found")
	ErrInvalidProposalStatus       = errors.New("invalid proposal status for this operation")
	ErrApprovalRequired            = errors.New("operation requires human approval before merge")
	ErrAgentSelfApprovalForbidden  = errors.New("agent cannot self-approve self-modification proposals")
)

// ModificationLevel classifies the risk and governance boundary of a code modification (Phase 6).
type ModificationLevel string

const (
	LevelADomain     ModificationLevel = "LEVEL_A_DOMAIN"     // Prompts, rules, domain profiles, novel/knowledge configs (Autonomous L2 verification)
	LevelBCore       ModificationLevel = "LEVEL_B_CORE"       // AIA core non-governance services, workers, observers (Human Approval Required)
	LevelCGovernance ModificationLevel = "LEVEL_C_GOVERNANCE" // Governance, policy, approval, crypto, migrations (Strictly Immutable / Locked)
)

// ModificationStatus tracks the lifecycle of a self-modification proposal.
type ModificationStatus string

const (
	StatusDraft           ModificationStatus = "DRAFT"
	StatusInSandbox       ModificationStatus = "IN_SANDBOX"
	StatusPendingApproval ModificationStatus = "PENDING_APPROVAL"
	StatusApproved        ModificationStatus = "APPROVED"
	StatusVerified        ModificationStatus = "VERIFIED"
	StatusMerged          ModificationStatus = "MERGED"
	StatusRolledBack      ModificationStatus = "ROLLED_BACK"
	StatusRejected        ModificationStatus = "REJECTED"
)

// SelfModificationProposal represents a proposed change to AIA's codebase or configurations.
type SelfModificationProposal struct {
	ID                 string             `json:"id" yaml:"id"`
	Title              string             `json:"title" yaml:"title"`
	Description        string             `json:"description" yaml:"description"`
	Rationale          string             `json:"rationale" yaml:"rationale"`
	TargetFiles        []string           `json:"target_files" yaml:"target_files"`
	Patches            map[string]string  `json:"patches" yaml:"patches"` // filepath -> file content or unified diff
	ScopeLevel         ModificationLevel  `json:"scope_level" yaml:"scope_level"`
	RequiresApproval   bool               `json:"requires_approval" yaml:"requires_approval"`
	ApprovalID         *string            `json:"approval_id,omitempty" yaml:"approval_id,omitempty"`
	Status             ModificationStatus `json:"status" yaml:"status"`
	BranchName         string             `json:"branch_name,omitempty" yaml:"branch_name,omitempty"`
	PreCommitSHA       string             `json:"pre_commit_sha,omitempty" yaml:"pre_commit_sha,omitempty"`
	PostCommitSHA      string             `json:"post_commit_sha,omitempty" yaml:"post_commit_sha,omitempty"`
	VerificationOutput string             `json:"verification_output,omitempty" yaml:"verification_output,omitempty"`
	RCALog             string             `json:"rca_log,omitempty" yaml:"rca_log,omitempty"`
	CreatedAt          time.Time          `json:"created_at" yaml:"created_at"`
	CompletedAt        *time.Time         `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
}

// TestVerificationResult records execution outcomes of sandbox test suites.
type TestVerificationResult struct {
	Level       string    `json:"level"` // L1_SYNTAX, L2_UNIT_TEST, L3_REGRESSION
	Passed      bool      `json:"passed"`
	Output      string    `json:"output"`
	TotalTests  int       `json:"total_tests"`
	FailedTests []string  `json:"failed_tests,omitempty"`
	DurationMs  int64     `json:"duration_ms"`
	ExecutedAt  time.Time `json:"executed_at"`
}
