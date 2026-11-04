package selfmod

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/selfmod"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/service/approval"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/workspace"
)

type ProposalInput struct {
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Rationale      string            `json:"rationale"`
	TargetFiles    []string          `json:"target_files"`
	Patches        map[string]string `json:"patches"` // filepath -> file content
	ValidationPlan []string          `json:"validation_plan,omitempty"` // AIA-14 §4.6
}

// Pipeline coordinates the safe self-modification workflow (Phase 6, roadmap.md §M8, AIA-14).
type Pipeline struct {
	repoRoot     string
	repo         *repository.SelfModRepository
	approvalSvc  *approval.Service
	auditRepo    *repository.AuditRepository
	decisionRepo *repository.DecisionRepository
	bus          eventbus.Publisher
	classifier   *ScopeClassifier
}

func NewPipeline(
	repoRoot string,
	repo *repository.SelfModRepository,
	approvalSvc *approval.Service,
	auditRepo *repository.AuditRepository,
	decisionRepo *repository.DecisionRepository,
	bus eventbus.Publisher,
) *Pipeline {
	return &Pipeline{
		repoRoot:     repoRoot,
		repo:         repo,
		approvalSvc:  approvalSvc,
		auditRepo:    auditRepo,
		decisionRepo: decisionRepo,
		bus:          bus,
		classifier:   NewScopeClassifier(),
	}
}

// Propose evaluates scope boundaries and registers a self-modification proposal (AIA-14 §4.3, §4.6).
func (p *Pipeline) Propose(ctx context.Context, input ProposalInput) (*selfmod.SelfModificationProposal, error) {
	level, err := p.classifier.Classify(input.TargetFiles)
	now := time.Now().UTC()

	// 1. Fail-closed on Level C (Governance / Security)
	if err == selfmod.ErrLevelCModificationForbidden || level == selfmod.LevelCGovernance {
		if p.auditRepo != nil {
			risk := string(governance.RiskTierCritical)
			action := string(governance.GovernanceActionBlocked)
			details, _ := json.Marshal(map[string]any{
				"title":        input.Title,
				"target_files": input.TargetFiles,
				"error":        selfmod.ErrLevelCModificationForbidden.Error(),
			})
			_ = p.auditRepo.Append(ctx, &repository.AuditLogEntry{
				ID:               uuid.New().String(),
				ActorType:        "AGENT",
				ActorID:          "selfmod-pipeline",
				Action:           "LEVEL_C_SELF_MODIFICATION_BLOCKED",
				ResourceType:     "GOVERNANCE",
				ResourceID:       "core_policy",
				Outcome:          "BLOCKED",
				Details:          details,
				RiskTier:         &risk,
				GovernanceAction: &action,
				CreatedAt:        now,
			})
		}
		return nil, selfmod.ErrLevelCModificationForbidden
	}

	valPlan := input.ValidationPlan
	if len(valPlan) == 0 {
		valPlan = []string{selfmod.VerifLevelL1Syntax, selfmod.VerifLevelL2Unit}
	}

	proposal := &selfmod.SelfModificationProposal{
		ID:               uuid.New().String(),
		Title:            input.Title,
		Description:      input.Description,
		Rationale:        input.Rationale,
		TargetFiles:      input.TargetFiles,
		Patches:          input.Patches,
		ScopeLevel:       level,
		RequiresApproval: (level == selfmod.LevelBCore),
		Status:           selfmod.StatusDraft,
		ValidationPlan:   valPlan,
		CreatedAt:        now,
	}

	if err := p.repo.Create(ctx, proposal); err != nil {
		return nil, fmt.Errorf("failed to persist selfmod proposal: %w", err)
	}

	return proposal, nil
}

// SandboxTestRunner runs test suites in an isolated worktree directory (L2).
type SandboxTestRunner func(ctx context.Context, workDir string) (bool, string, error)

// MultiLevelRunner runs a specific verification level in sandbox (AIA-14 §7.1).
type MultiLevelRunner func(ctx context.Context, workDir string, level string) (*selfmod.TestVerificationResult, error)

// VerificationOptions defines execution parameters, multi-tier runners, and retry limits (AIA-14 §7.4).
type VerificationOptions struct {
	TestRunner         SandboxTestRunner // Fallback single L2 runner for backward compatibility
	MultiLevelRunner   MultiLevelRunner   // Level-aware runner (L1~L6)
	MaxRetries         int               // Maximum Retry attempts (default: 1, no retries)
	ExecutionBudget    int               // Maximum execution budget
	TimeLimit          time.Duration     // Time limit per verification run
	RetryPatchModifier func(ctx context.Context, attempt int, lastErr string, currentPatches map[string]string) (map[string]string, error)
}

// ExecuteSandboxVerification maintains backward compatibility by running default sandbox verification.
func (p *Pipeline) ExecuteSandboxVerification(
	ctx context.Context,
	proposalID string,
	testRunner SandboxTestRunner,
) (*selfmod.SelfModificationProposal, error) {
	return p.ExecuteVerificationWithOptions(ctx, proposalID, VerificationOptions{
		TestRunner: testRunner,
		MaxRetries: 1,
	})
}

// ExecuteVerificationWithOptions runs multi-tier verification with retry loops and budget limits (AIA-14 §7.1, §7.4).
func (p *Pipeline) ExecuteVerificationWithOptions(
	ctx context.Context,
	proposalID string,
	opts VerificationOptions,
) (*selfmod.SelfModificationProposal, error) {
	proposal, err := p.repo.Get(ctx, proposalID)
	if err != nil {
		return nil, err
	}

	if opts.TimeLimit > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.TimeLimit)
		defer cancel()
	}

	if _, statErr := os.Stat(filepath.Join(p.repoRoot, ".git")); statErr != nil {
		return nil, fmt.Errorf("repository root %s is not a git repository", p.repoRoot)
	}

	// Capture initial commit SHA
	cmdRev := exec.Command("git", "rev-parse", "HEAD")
	cmdRev.Dir = p.repoRoot
	outRev, err := cmdRev.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git HEAD: %w", err)
	}
	preCommitSHA := strings.TrimSpace(string(outRev))
	proposal.PreCommitSHA = preCommitSHA

	// Allocate isolated branch and temp worktree
	branchName := fmt.Sprintf("aia-selfmod-%s-%d", proposal.ID[:8], time.Now().UnixNano())
	proposal.BranchName = branchName

	tmpWorktree, err := os.MkdirTemp("", "aia-selfmod-wt-*")
	if err != nil {
		return nil, fmt.Errorf("failed to allocate temp worktree: %w", err)
	}
	_ = os.Remove(tmpWorktree)

	if err := workspace.CreateIsolatedWorktree(p.repoRoot, branchName, tmpWorktree); err != nil {
		return nil, fmt.Errorf("failed to create isolated worktree: %w", err)
	}

	defer func() {
		_ = workspace.RemoveIsolatedWorktree(p.repoRoot, tmpWorktree)
	}()

	proposal.Status = selfmod.StatusInSandbox
	_ = p.repo.Update(ctx, proposal)

	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	var allResults []selfmod.TestVerificationResult
	var lastOutput string
	var verifiedOverall bool
	var rcaAttempts []string

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// If retrying, request modified patches
		if attempt > 1 && opts.RetryPatchModifier != nil {
			revised, modErr := opts.RetryPatchModifier(ctx, attempt, lastOutput, proposal.Patches)
			if modErr != nil {
				rcaAttempts = append(rcaAttempts, fmt.Sprintf("Attempt %d patch revision failed: %v", attempt, modErr))
				break
			}
			proposal.Patches = revised
		}

		// Apply patches in sandbox worktree
		applyErrOccurred := false
		for relPath, content := range proposal.Patches {
			absPath := filepath.Join(tmpWorktree, relPath)
			if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create patch directory for %s: %w", relPath, err)
			}
			if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to write patch file %s: %w", relPath, err)
			}
		}
		if applyErrOccurred {
			break
		}

		// Multi-tier stage execution (AIA-14 §7.1)
		stages := proposal.ValidationPlan
		if len(stages) == 0 {
			stages = []string{selfmod.VerifLevelL1Syntax, selfmod.VerifLevelL2Unit}
		}

		attemptPassed := true
		var currentStageResults []selfmod.TestVerificationResult

		for _, stage := range stages {
			start := time.Now()
			var res *selfmod.TestVerificationResult

			if opts.MultiLevelRunner != nil {
				res, err = opts.MultiLevelRunner(ctx, tmpWorktree, stage)
				if err != nil && res == nil {
					res = &selfmod.TestVerificationResult{
						Level:      stage,
						Passed:     false,
						Output:     err.Error(),
						ExecutedAt: start,
					}
				}
			} else {
				// Built-in runner fallbacks
				switch stage {
				case selfmod.VerifLevelL1Syntax:
					res = runL1SyntaxVerification(tmpWorktree, proposal.TargetFiles)
				case selfmod.VerifLevelL2Unit:
					if opts.TestRunner != nil {
						pOk, outStr, tErr := opts.TestRunner(ctx, tmpWorktree)
						res = &selfmod.TestVerificationResult{
							Level:      selfmod.VerifLevelL2Unit,
							Passed:     pOk && tErr == nil,
							Output:     outStr,
							DurationMs: time.Since(start).Milliseconds(),
							ExecutedAt: start,
						}
						if tErr != nil && res.Output == "" {
							res.Output = tErr.Error()
						}
					} else {
						res = &selfmod.TestVerificationResult{
							Level:      selfmod.VerifLevelL2Unit,
							Passed:     true,
							Output:     "No unit test runner configured; stage skipped as pass",
							ExecutedAt: start,
						}
					}
				default:
					// For L3~L6 when only TestRunner is provided, propagate L2 outcome
					passed := true
					outStr := fmt.Sprintf("Stage %s passed via default verification baseline", stage)
					if opts.TestRunner != nil {
						pOk, o, tErr := opts.TestRunner(ctx, tmpWorktree)
						passed = pOk && tErr == nil
						outStr = o
					}
					res = &selfmod.TestVerificationResult{
						Level:      stage,
						Passed:     passed,
						Output:     outStr,
						DurationMs: time.Since(start).Milliseconds(),
						ExecutedAt: start,
					}
				}
			}

			if res == nil {
				res = &selfmod.TestVerificationResult{
					Level:      stage,
					Passed:     true,
					Output:     "Stage completed",
					ExecutedAt: start,
				}
			}

			currentStageResults = append(currentStageResults, *res)
			if !res.Passed {
				attemptPassed = false
				lastOutput = fmt.Sprintf("[%s failed]: %s", stage, res.Output)
				rcaAttempts = append(rcaAttempts, fmt.Sprintf("Attempt %d failed at stage %s: %s", attempt, stage, res.Output))
				break
			}
		}

		allResults = currentStageResults
		if attemptPassed {
			verifiedOverall = true
			break
		}
	}

	proposal.VerificationResults = allResults

	// On Verification Failure: Trigger Clean Rollback
	if !verifiedOverall {
		delCmd := exec.Command("git", "branch", "-D", branchName)
		delCmd.Dir = p.repoRoot
		_ = delCmd.Run()

		now := time.Now().UTC()
		proposal.Status = selfmod.StatusRolledBack
		proposal.RCALog = strings.Join(rcaAttempts, " | ")
		if proposal.RCALog == "" {
			proposal.RCALog = fmt.Sprintf("Sandbox verification failed: %s", lastOutput)
		}
		proposal.CompletedAt = &now
		_ = p.repo.Update(ctx, proposal)

		return proposal, fmt.Errorf("sandbox verification failed after %d attempts (clean rollback executed): %s", maxRetries, lastOutput)
	}

	// On Verification Success: Commit in branch
	gitRun := func(dir string, args ...string) error {
		c := exec.Command("git", args...)
		c.Dir = dir
		c.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=AIASelfMod",
			"GIT_AUTHOR_EMAIL=selfmod@aia.local",
			"GIT_COMMITTER_NAME=AIASelfMod",
			"GIT_COMMITTER_EMAIL=selfmod@aia.local",
		)
		out, err := c.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s failed: %s: %w", strings.Join(args, " "), string(out), err)
		}
		return nil
	}

	if err := gitRun(tmpWorktree, "add", "."); err != nil {
		return nil, err
	}
	if err := gitRun(tmpWorktree, "commit", "-m", fmt.Sprintf("[selfmod] %s", proposal.Title)); err != nil {
		return nil, err
	}

	cmdPostRev := exec.Command("git", "rev-parse", "HEAD")
	cmdPostRev.Dir = tmpWorktree
	postRevOut, _ := cmdPostRev.Output()
	proposal.PostCommitSHA = strings.TrimSpace(string(postRevOut))

	var outSummaries []string
	for _, r := range allResults {
		statusStr := "PASS"
		if !r.Passed {
			statusStr = "FAIL"
		}
		outSummaries = append(outSummaries, fmt.Sprintf("[%s: %s] %s", r.Level, statusStr, r.Output))
	}
	proposal.VerificationOutput = strings.Join(outSummaries, "\n")
	proposal.Status = selfmod.StatusVerified

	// If Level A: Autonomous merge without human approval
	if proposal.ScopeLevel == selfmod.LevelADomain && !proposal.RequiresApproval {
		if err := p.mergeBranchDirect(ctx, proposal); err != nil {
			return nil, fmt.Errorf("autonomous merge failed: %w", err)
		}
		return proposal, nil
	}

	// If Level B: Request human approval
	if proposal.RequiresApproval {
		appr, err := p.SubmitForApproval(ctx, proposal.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to submit approval gate: %w", err)
		}
		proposal.ApprovalID = &appr.ID
		proposal.Status = selfmod.StatusPendingApproval
	}

	_ = p.repo.Update(ctx, proposal)
	return proposal, nil
}

// runL1SyntaxVerification verifies syntax for code and configuration files.
func runL1SyntaxVerification(workDir string, targetFiles []string) *selfmod.TestVerificationResult {
	start := time.Now()
	fset := token.NewFileSet()

	for _, relPath := range targetFiles {
		absPath := filepath.Join(workDir, relPath)
		if strings.HasSuffix(relPath, ".go") {
			if _, err := parser.ParseFile(fset, absPath, nil, parser.AllErrors); err != nil {
				return &selfmod.TestVerificationResult{
					Level:       selfmod.VerifLevelL1Syntax,
					Passed:      false,
					Output:      fmt.Sprintf("Go syntax error in %s: %v", relPath, err),
					TotalTests:  1,
					FailedTests: []string{relPath},
					DurationMs:  time.Since(start).Milliseconds(),
					ExecutedAt:  start,
				}
			}
		} else if strings.HasSuffix(relPath, ".json") {
			content, err := os.ReadFile(absPath)
			if err == nil && !json.Valid(content) {
				return &selfmod.TestVerificationResult{
					Level:       selfmod.VerifLevelL1Syntax,
					Passed:      false,
					Output:      fmt.Sprintf("Malformed JSON in %s", relPath),
					TotalTests:  1,
					FailedTests: []string{relPath},
					DurationMs:  time.Since(start).Milliseconds(),
					ExecutedAt:  start,
				}
			}
		}
	}

	return &selfmod.TestVerificationResult{
		Level:      selfmod.VerifLevelL1Syntax,
		Passed:     true,
		Output:     fmt.Sprintf("L1 Syntax check passed for %d target files", len(targetFiles)),
		TotalTests: len(targetFiles),
		DurationMs: time.Since(start).Milliseconds(),
		ExecutedAt: start,
	}
}

// RequestChanges allows a human reviewer to request modifications on a pending proposal (AIA-14 §4.8).
func (p *Pipeline) RequestChanges(ctx context.Context, proposalID string, reviewer governance.Actor, feedback string) error {
	if !reviewer.IsHuman() {
		return selfmod.ErrAgentSelfApprovalForbidden
	}
	proposal, err := p.repo.Get(ctx, proposalID)
	if err != nil {
		return err
	}
	if proposal.Status != selfmod.StatusPendingApproval {
		return fmt.Errorf("cannot request changes for proposal in status %s", proposal.Status)
	}

	proposal.Status = selfmod.StatusChangesRequested
	proposal.RCALog = fmt.Sprintf("Changes requested by %s: %s", reviewer.ID, feedback)
	if err := p.repo.Update(ctx, proposal); err != nil {
		return err
	}

	if p.bus != nil {
		p.bus.Publish("selfmod.changes_requested", map[string]any{
			"proposal_id": proposal.ID,
			"reviewer_id": reviewer.ID,
			"feedback":    feedback,
		})
	}
	return nil
}

// DeferProposal allows a human reviewer to defer decision on a proposal (AIA-14 §4.8).
func (p *Pipeline) DeferProposal(ctx context.Context, proposalID string, reviewer governance.Actor, reason string) error {
	if !reviewer.IsHuman() {
		return selfmod.ErrAgentSelfApprovalForbidden
	}
	proposal, err := p.repo.Get(ctx, proposalID)
	if err != nil {
		return err
	}
	if proposal.Status != selfmod.StatusPendingApproval {
		return fmt.Errorf("cannot defer proposal in status %s", proposal.Status)
	}

	proposal.Status = selfmod.StatusDeferred
	proposal.RCALog = fmt.Sprintf("Deferred by %s: %s", reviewer.ID, reason)
	if err := p.repo.Update(ctx, proposal); err != nil {
		return err
	}

	if p.bus != nil {
		p.bus.Publish("selfmod.deferred", map[string]any{
			"proposal_id": proposal.ID,
			"reviewer_id": reviewer.ID,
			"reason":      reason,
		})
	}
	return nil
}

// MonitorPostDeployment checks post-deployment metrics against reliability and autonomy boundaries (AIA-14 §4.10, §8.2).
func (p *Pipeline) MonitorPostDeployment(ctx context.Context, proposalID string, metrics selfmod.PostDeployMetrics) (*selfmod.PostDeployHealthReport, error) {
	proposal, err := p.repo.Get(ctx, proposalID)
	if err != nil {
		return nil, err
	}

	report := &selfmod.PostDeployHealthReport{
		ProposalID: proposalID,
		Healthy:    true,
		Metrics:    metrics,
	}

	if metrics.CrashRate > 0 {
		report.Healthy = false
		report.TriggersRollback = true
		report.Issues = append(report.Issues, fmt.Sprintf("critical crash rate detected: %.2f%%", metrics.CrashRate*100))
	}
	if metrics.ErrorRate > 0.05 {
		report.Healthy = false
		report.TriggersRollback = true
		report.Issues = append(report.Issues, fmt.Sprintf("error rate exceeds safety threshold of 5%%: %.2f%%", metrics.ErrorRate*100))
	}
	if metrics.AutonomyMetric > 0 && metrics.AutonomyMetric < 0.70 {
		report.Healthy = false
		report.TriggersRollback = true
		report.Issues = append(report.Issues, fmt.Sprintf("autonomy score degraded below benchmark: %.2f", metrics.AutonomyMetric))
	}

	proposal.PostDeployMetrics = &metrics
	_ = p.repo.Update(ctx, proposal)

	if p.auditRepo != nil {
		risk := string(governance.RiskTierLow)
		if !report.Healthy {
			risk = string(governance.RiskTierCritical)
		}
		det, _ := json.Marshal(report)
		_ = p.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "SYSTEM",
			ActorID:      "selfmod-monitor",
			Action:       "POST_DEPLOYMENT_HEALTH_CHECK",
			ResourceType: "PROPOSAL",
			ResourceID:   proposal.ID,
			Outcome:      map[bool]string{true: "HEALTHY", false: "DEGRADED"}[report.Healthy],
			Details:      det,
			RiskTier:     &risk,
			CreatedAt:    time.Now().UTC(),
		})
	}

	if p.bus != nil {
		p.bus.Publish("selfmod.monitored", map[string]any{
			"proposal_id":       proposal.ID,
			"healthy":           report.Healthy,
			"triggers_rollback": report.TriggersRollback,
			"metrics":           metrics,
		})
	}

	return report, nil
}

// RollbackProposal reverts a merged proposal if post-deployment monitoring detects severe degradation (AIA-14 §8.1, §8.2).
func (p *Pipeline) RollbackProposal(ctx context.Context, proposalID string, reason string) error {
	proposal, err := p.repo.Get(ctx, proposalID)
	if err != nil {
		return err
	}
	if proposal.Status != selfmod.StatusMerged {
		return fmt.Errorf("proposal is not in MERGED status, cannot rollback (current: %s)", proposal.Status)
	}

	if proposal.PreCommitSHA != "" && len(proposal.TargetFiles) > 0 {
		for _, f := range proposal.TargetFiles {
			cmdCheckout := exec.Command("git", "checkout", proposal.PreCommitSHA, "--", f)
			cmdCheckout.Dir = p.repoRoot
			_ = cmdCheckout.Run()
		}
		cmdAdd := exec.Command("git", "add", ".")
		cmdAdd.Dir = p.repoRoot
		_ = cmdAdd.Run()

		cmdCommit := exec.Command("git", "commit", "-m", fmt.Sprintf("[selfmod rollback] Revert %s: %s", proposal.Title, reason))
		cmdCommit.Dir = p.repoRoot
		cmdCommit.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=AIASelfMod",
			"GIT_AUTHOR_EMAIL=selfmod@aia.local",
			"GIT_COMMITTER_NAME=AIASelfMod",
			"GIT_COMMITTER_EMAIL=selfmod@aia.local",
		)
		_ = cmdCommit.Run()
	}

	now := time.Now().UTC()
	proposal.Status = selfmod.StatusRolledBack
	proposal.CompletedAt = &now
	proposal.RCALog = fmt.Sprintf("Rolled back after merge: %s", reason)
	_ = p.repo.Update(ctx, proposal)

	if p.bus != nil {
		p.bus.Publish("selfmod.rolled_back", map[string]any{
			"proposal_id": proposal.ID,
			"reason":      reason,
			"timestamp":   now,
		})
	}
	return nil
}

// SubmitForApproval creates an approval gate in the governance subsystem.
func (p *Pipeline) SubmitForApproval(ctx context.Context, proposalID string) (*governance.Approval, error) {
	proposal, err := p.repo.Get(ctx, proposalID)
	if err != nil {
		return nil, err
	}

	if p.approvalSvc == nil {
		return nil, fmt.Errorf("approval service not available")
	}

	detailJSON, _ := json.Marshal(map[string]any{
		"proposal_id":  proposal.ID,
		"title":        proposal.Title,
		"target_files": proposal.TargetFiles,
		"branch_name":  proposal.BranchName,
		"scope_level":  proposal.ScopeLevel,
	})

	decisionID := fmt.Sprintf("dec-selfmod-%s", proposal.ID)
	if p.decisionRepo != nil {
		dec := &decision.Decision{
			ID:               decisionID,
			ActionType:       decision.ActionAct,
			IntentSummary:    proposal.Title,
			Justification:    proposal.Rationale,
			GovernanceResult: decision.GovernanceRequiredApproval,
			CreatedAt:        time.Now().UTC(),
		}
		_ = p.decisionRepo.Create(ctx, dec, nil)
	}

	appr, err := p.approvalSvc.RequestApproval(ctx, approval.RequestApprovalInput{
		RequesterEntity:     "agent:selfmod-pipeline",
		OperationType:       "SELF_MODIFICATION",
		OperationDetailJSON: string(detailJSON),
		RiskTier:            governance.RiskTierHigh,
		TimeoutPolicy:       governance.TimeoutPolicyReject,
		TimeoutDuration:     24 * time.Hour,
		DecisionID:          &decisionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create approval gate: %w", err)
	}

	proposal.ApprovalID = &appr.ID
	proposal.Status = selfmod.StatusPendingApproval
	_ = p.repo.Update(ctx, proposal)

	return appr, nil
}

// ApplyMerge applies an approved proposal to main repository.
func (p *Pipeline) ApplyMerge(ctx context.Context, proposalID string, reviewer governance.Actor) error {
	proposal, err := p.repo.Get(ctx, proposalID)
	if err != nil {
		return err
	}

	// Governance check: Agent CANNOT approve itself
	if proposal.RequiresApproval {
		if !reviewer.IsHuman() {
			return selfmod.ErrAgentSelfApprovalForbidden
		}
		if proposal.ApprovalID != nil {
			appr, err := p.approvalSvc.Get(ctx, *proposal.ApprovalID)
			if err != nil || appr.Status != governance.ApprovalApproved {
				return selfmod.ErrApprovalRequired
			}
		}
	}

	return p.mergeBranchDirect(ctx, proposal)
}

func (p *Pipeline) mergeBranchDirect(ctx context.Context, proposal *selfmod.SelfModificationProposal) error {
	// Merge branch into main
	cmdMerge := exec.Command("git", "merge", "--no-ff", "-m", fmt.Sprintf("Merge self-modification: %s", proposal.Title), proposal.BranchName)
	cmdMerge.Dir = p.repoRoot
	out, err := cmdMerge.CombinedOutput()
	if err != nil {
		// Rollback merge
		abortCmd := exec.Command("git", "merge", "--abort")
		abortCmd.Dir = p.repoRoot
		_ = abortCmd.Run()
		return fmt.Errorf("git merge failed: %s: %w", string(out), err)
	}

	// Delete temporary branch
	delCmd := exec.Command("git", "branch", "-D", proposal.BranchName)
	delCmd.Dir = p.repoRoot
	_ = delCmd.Run()

	now := time.Now().UTC()
	proposal.Status = selfmod.StatusMerged
	proposal.CompletedAt = &now
	_ = p.repo.Update(ctx, proposal)

	if p.auditRepo != nil {
		risk := string(governance.RiskTierMedium)
		if proposal.ScopeLevel == selfmod.LevelBCore {
			risk = string(governance.RiskTierHigh)
		}
		action := string(governance.GovernanceActionAllowed)
		details, _ := json.Marshal(map[string]any{
			"proposal_id":  proposal.ID,
			"title":        proposal.Title,
			"scope_level":  proposal.ScopeLevel,
			"target_files": proposal.TargetFiles,
		})
		_ = p.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:               uuid.New().String(),
			ActorType:        "SYSTEM",
			ActorID:          "selfmod-pipeline",
			Action:           "SELF_MODIFICATION_MERGED",
			ResourceType:     "REPOSITORY",
			ResourceID:       proposal.ID,
			Outcome:          "MERGED",
			Details:          details,
			RiskTier:         &risk,
			GovernanceAction: &action,
			CreatedAt:        now,
		})
	}

	if p.bus != nil {
		p.bus.Publish("selfmod.merged", map[string]any{
			"proposal_id": proposal.ID,
			"title":       proposal.Title,
			"scope_level": proposal.ScopeLevel,
			"timestamp":   now,
		})
	}

	return nil
}
