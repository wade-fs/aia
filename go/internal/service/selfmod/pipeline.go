package selfmod

import (
	"context"
	"encoding/json"
	"fmt"
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
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Rationale   string            `json:"rationale"`
	TargetFiles []string          `json:"target_files"`
	Patches     map[string]string `json:"patches"` // filepath -> file content
}

// Pipeline coordinates the safe self-modification workflow (Phase 6, roadmap.md §M8).
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

// Propose evaluates scope boundaries and registers a self-modification proposal.
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
		CreatedAt:        now,
	}

	if err := p.repo.Create(ctx, proposal); err != nil {
		return nil, fmt.Errorf("failed to persist selfmod proposal: %w", err)
	}

	return proposal, nil
}

// SandboxTestRunner runs test suites in an isolated worktree directory.
type SandboxTestRunner func(ctx context.Context, workDir string) (bool, string, error)

// ExecuteSandboxVerification runs patches inside an isolated Git worktree and verifies L2 tests.
func (p *Pipeline) ExecuteSandboxVerification(
	ctx context.Context,
	proposalID string,
	testRunner SandboxTestRunner,
) (*selfmod.SelfModificationProposal, error) {
	proposal, err := p.repo.Get(ctx, proposalID)
	if err != nil {
		return nil, err
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

	// Apply patches in sandbox worktree
	for relPath, content := range proposal.Patches {
		absPath := filepath.Join(tmpWorktree, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create patch directory for %s: %w", relPath, err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write patch file %s: %w", relPath, err)
		}
	}

	// Run verification in worktree (L2)
	passed := true
	var testOutput string
	if testRunner != nil {
		pOk, outStr, tErr := testRunner(ctx, tmpWorktree)
		passed = pOk && tErr == nil
		testOutput = outStr
		if tErr != nil && testOutput == "" {
			testOutput = tErr.Error()
		}
	}

	// On Verification Failure: Trigger Clean Rollback
	if !passed {
		// Delete branch
		delCmd := exec.Command("git", "branch", "-D", branchName)
		delCmd.Dir = p.repoRoot
		_ = delCmd.Run()

		now := time.Now().UTC()
		proposal.Status = selfmod.StatusRolledBack
		proposal.RCALog = fmt.Sprintf("Sandbox verification failed: %s", testOutput)
		proposal.CompletedAt = &now
		_ = p.repo.Update(ctx, proposal)

		return proposal, fmt.Errorf("sandbox verification failed (clean rollback executed): %s", testOutput)
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
	proposal.VerificationOutput = testOutput
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
