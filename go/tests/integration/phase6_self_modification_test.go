package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/selfmod"
	selfmodSvc "github.com/wade/aia/internal/service/selfmod"
)

// setupSelfModTestGitRepo initializes a local git repository in a temp directory.
func setupSelfModTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	gitRun := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=AIATest",
			"GIT_AUTHOR_EMAIL=aia@example.com",
			"GIT_COMMITTER_NAME=AIATest",
			"GIT_COMMITTER_EMAIL=aia@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %s (%v)", strings.Join(args, " "), string(out), err)
		}
	}

	gitRun("init", "-b", "main")
	gitRun("config", "user.name", "AIATest")
	gitRun("config", "user.email", "aia@example.com")
	gitRun("config", "commit.gpgsign", "false")

	// Create initial file
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# AIA Self-Mod Repository\nInitial baseline commit\n"), 0644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}
	gitRun("add", "README.md")
	gitRun("commit", "-m", "Initial baseline commit")

	return dir
}

// 1. Level C: Governance and Policy modification is strictly forbidden and fails closed (Phase 6 M8).
func TestPhase6_ScopeIsolation_LevelC_GovernanceStrictlyForbidden(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	gitRepo := setupSelfModTestGitRepo(t)
	pipeline := selfmodSvc.NewPipeline(gitRepo, app.SelfModRepo, app.ApprovalSvc, app.AuditRepo, app.DecisionRepo, app.EventBus)

	// Attempt to self-modify governance evaluator
	input := selfmodSvc.ProposalInput{
		Title:       "Bypass policy checks",
		Description: "Attempt to modify governance evaluator directly",
		Rationale:   "Relaxing safety constraints for agent execution",
		TargetFiles: []string{"internal/policy/evaluator.go"},
		Patches: map[string]string{
			"internal/policy/evaluator.go": "// hacked bypass code",
		},
	}

	proposal, err := pipeline.Propose(ctx, input)
	if err != selfmod.ErrLevelCModificationForbidden {
		t.Fatalf("expected ErrLevelCModificationForbidden, got %v (proposal: %+v)", err, proposal)
	}

	// Verify security audit log entry was recorded
	entries, err := app.AuditRepo.List(ctx, 10)
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	foundBlocked := false
	for _, entry := range entries {
		if entry.Action == "LEVEL_C_SELF_MODIFICATION_BLOCKED" && *entry.RiskTier == string(governance.RiskTierCritical) {
			foundBlocked = true
			break
		}
	}
	if !foundBlocked {
		t.Errorf("expected audit entry for LEVEL_C_SELF_MODIFICATION_BLOCKED with CRITICAL risk tier")
	}
}

// 2. Level A: Domain rules/prompts modification executes in sandbox and auto-merges without human approval.
func TestPhase6_LevelA_AutonomousModificationAndL2Verification(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	gitRepo := setupSelfModTestGitRepo(t)
	pipeline := selfmodSvc.NewPipeline(gitRepo, app.SelfModRepo, app.ApprovalSvc, app.AuditRepo, app.DecisionRepo, app.EventBus)

	input := selfmodSvc.ProposalInput{
		Title:       "Update Domain Writing Consistency Rules",
		Description: "Extend rules to support celestial body spatial constraints",
		Rationale:   "Plan2.md space opera research profile expansion",
		TargetFiles: []string{"rules/space_rules.yaml"},
		Patches: map[string]string{
			"rules/space_rules.yaml": "rules:\n  - celestial_orbit_consistency\n  - speed_of_light_limit\n",
		},
	}

	proposal, err := pipeline.Propose(ctx, input)
	if err != nil {
		t.Fatalf("propose failed: %v", err)
	}
	if proposal.ScopeLevel != selfmod.LevelADomain || proposal.RequiresApproval {
		t.Fatalf("expected LevelADomain and RequiresApproval=false, got %+v", proposal)
	}

	// Execute sandbox verification with a passing test runner
	verified, err := pipeline.ExecuteSandboxVerification(ctx, proposal.ID, func(ctx context.Context, workDir string) (bool, string, error) {
		// Verify patch exists in worktree
		target := filepath.Join(workDir, "rules", "space_rules.yaml")
		data, err := os.ReadFile(target)
		if err != nil || !strings.Contains(string(data), "speed_of_light_limit") {
			return false, "rule file missing or invalid", nil
		}
		return true, "L2 verification passed (syntax valid, schema verified)", nil
	})

	if err != nil {
		t.Fatalf("sandbox execution failed: %v", err)
	}
	if verified.Status != selfmod.StatusMerged {
		t.Fatalf("expected Level A proposal to be automatically merged, got %s", verified.Status)
	}

	// Verify main repository now contains the merged file
	mergedFile := filepath.Join(gitRepo, "rules", "space_rules.yaml")
	content, err := os.ReadFile(mergedFile)
	if err != nil || !strings.Contains(string(content), "celestial_orbit_consistency") {
		t.Fatalf("merged file missing or invalid in main repo: %v", err)
	}

	// Verify main repo git status is clean
	cmdStatus := exec.Command("git", "status", "--porcelain")
	cmdStatus.Dir = gitRepo
	statusOut, _ := cmdStatus.Output()
	if strings.TrimSpace(string(statusOut)) != "" {
		t.Errorf("main repo dirty after autonomous merge: %s", string(statusOut))
	}
}

// 3. Level B: AIA Core modification executes in sandbox, passes L2, but requires human approval before merge.
func TestPhase6_LevelB_CoreModificationRequiresHumanApprovalGate(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	gitRepo := setupSelfModTestGitRepo(t)
	pipeline := selfmodSvc.NewPipeline(gitRepo, app.SelfModRepo, app.ApprovalSvc, app.AuditRepo, app.DecisionRepo, app.EventBus)

	// Level B target: internal/autonomy/observer.go
	input := selfmodSvc.ProposalInput{
		Title:       "Enhance CodeHealthObserver Diagnostics",
		Description: "Add compiler panic backtrace parser",
		Rationale:   "Improve RCA precision for syntax crashes",
		TargetFiles: []string{"internal/autonomy/observer_enhancement.go"},
		Patches: map[string]string{
			"internal/autonomy/observer_enhancement.go": "package autonomy\nfunc ParsePanicBacktrace() string { return \"ok\" }\n",
		},
	}

	proposal, err := pipeline.Propose(ctx, input)
	if err != nil {
		t.Fatalf("propose failed: %v", err)
	}
	if proposal.ScopeLevel != selfmod.LevelBCore || !proposal.RequiresApproval {
		t.Fatalf("expected LevelBCore and RequiresApproval=true, got %+v", proposal)
	}

	// Execute sandbox verification
	verified, err := pipeline.ExecuteSandboxVerification(ctx, proposal.ID, func(ctx context.Context, workDir string) (bool, string, error) {
		return true, "L2 test suite: 18 passed, 0 failed", nil
	})
	if err != nil {
		t.Fatalf("sandbox execution failed: %v", err)
	}

	if verified.Status != selfmod.StatusPendingApproval {
		t.Fatalf("expected Level B proposal to wait in PENDING_APPROVAL, got %s", verified.Status)
	}
	if verified.ApprovalID == nil {
		t.Fatalf("expected ApprovalID to be created for Level B proposal")
	}

	// 1. Agent self-approval attempt: MUST be rejected
	agentActor := governance.Actor{Type: "AGENT", ID: "coding-agent"}
	errMergeAgent := pipeline.ApplyMerge(ctx, verified.ID, agentActor)
	if errMergeAgent != selfmod.ErrAgentSelfApprovalForbidden {
		t.Fatalf("expected ErrAgentSelfApprovalForbidden when agent attempts merge, got %v", errMergeAgent)
	}

	// 2. Human owner approves the proposal gate
	humanActor := governance.Actor{Type: "HUMAN", ID: "wade-owner"}
	if err := app.ApprovalSvc.Approve(ctx, *verified.ApprovalID, "wade-owner", "Approved code enhancement after diff review", "HUMAN"); err != nil {
		t.Fatalf("human approval failed: %v", err)
	}

	// 3. Apply merge with human authorization
	if err := pipeline.ApplyMerge(ctx, verified.ID, humanActor); err != nil {
		t.Fatalf("apply merge failed with human authorization: %v", err)
	}

	// Verify proposal state updated to MERGED
	updatedProp, err := app.SelfModRepo.Get(ctx, verified.ID)
	if err != nil || updatedProp.Status != selfmod.StatusMerged {
		t.Fatalf("expected proposal status MERGED, got %v (err: %v)", updatedProp.Status, err)
	}

	// Verify file merged into main repo
	mergedPath := filepath.Join(gitRepo, "internal", "autonomy", "observer_enhancement.go")
	if _, err := os.Stat(mergedPath); err != nil {
		t.Fatalf("merged file not found on main branch: %v", err)
	}
}

// 4. Verification Failure triggers automatic clean rollback with 0 residual code and unchanged SHA.
func TestPhase6_VerificationFailure_AutomaticCleanRollback(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	gitRepo := setupSelfModTestGitRepo(t)
	pipeline := selfmodSvc.NewPipeline(gitRepo, app.SelfModRepo, app.ApprovalSvc, app.AuditRepo, app.DecisionRepo, app.EventBus)

	cmdRev := exec.Command("git", "rev-parse", "HEAD")
	cmdRev.Dir = gitRepo
	shaBefore, _ := cmdRev.Output()

	input := selfmodSvc.ProposalInput{
		Title:       "Faulty Patch that breaks tests",
		Description: "Injecting invalid syntax or breaking logic",
		Rationale:   "Testing rollback invariants",
		TargetFiles: []string{"broken.txt"},
		Patches: map[string]string{
			"broken.txt": "syntax error: unexpected token",
		},
	}

	proposal, err := pipeline.Propose(ctx, input)
	if err != nil {
		t.Fatalf("propose failed: %v", err)
	}

	// Test runner that fails
	_, err = pipeline.ExecuteSandboxVerification(ctx, proposal.ID, func(ctx context.Context, workDir string) (bool, string, error) {
		return false, "compile error: broken.txt syntax corrupted", nil
	})

	if err == nil {
		t.Fatalf("expected sandbox execution to fail")
	}
	if !strings.Contains(err.Error(), "clean rollback executed") {
		t.Errorf("expected clean rollback error message, got: %v", err)
	}

	// Check proposal status is ROLLED_BACK
	pAfter, err := app.SelfModRepo.Get(ctx, proposal.ID)
	if err != nil || pAfter.Status != selfmod.StatusRolledBack {
		t.Fatalf("expected proposal status ROLLED_BACK, got %v", pAfter.Status)
	}
	if !strings.Contains(pAfter.RCALog, "broken.txt syntax corrupted") {
		t.Errorf("expected RCA log to capture failure message, got: %s", pAfter.RCALog)
	}

	// Invariant checks on main repository:
	// 1. Commit SHA unchanged
	cmdRevAfter := exec.Command("git", "rev-parse", "HEAD")
	cmdRevAfter.Dir = gitRepo
	shaAfter, _ := cmdRevAfter.Output()
	if string(shaBefore) != string(shaAfter) {
		t.Fatalf("HEAD SHA changed after rollback: %s != %s", string(shaBefore), string(shaAfter))
	}

	// 2. 0 residual broken.txt file
	if _, statErr := os.Stat(filepath.Join(gitRepo, "broken.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("residual broken.txt remained in main repo after rollback!")
	}

	// 3. 0 dirty files in git status
	cmdStatus := exec.Command("git", "status", "--porcelain")
	cmdStatus.Dir = gitRepo
	statusOut, _ := cmdStatus.Output()
	if strings.TrimSpace(string(statusOut)) != "" {
		t.Fatalf("main repository dirty after rollback: %s", string(statusOut))
	}
}

// 5. Autonomous Evolution Cycle: Observation ➔ Proposal ➔ Sandbox L2 ➔ Governed Merge
func TestPhase6_AutonomousEvolutionCycle_EndToEnd(t *testing.T) {
	app, tmpDir := createTestApp(t)
	defer os.RemoveAll(tmpDir)
	ctx := context.Background()

	gitRepo := setupSelfModTestGitRepo(t)
	pipeline := selfmodSvc.NewPipeline(gitRepo, app.SelfModRepo, app.ApprovalSvc, app.AuditRepo, app.DecisionRepo, app.EventBus)
	evolutionEng := autonomy.NewSelfEvolutionEngine(pipeline)

	opp := autonomy.EvolutionOpportunity{
		ID:          "opp-domain-rule-01",
		Domain:      "writing",
		Summary:     "Add Celestial Coordinate Consistency Rule",
		Rationale:   "Enable multi-planetary space opera validation",
		TargetFiles: []string{"rules/celestial_rules.md"},
		ProposedFix: "# Celestial Rules\n- orbit_period_kepler_consistency\n",
	}

	// Autonomous cycle for Level A
	proposal, err := evolutionEng.RunEvolutionCycle(ctx, opp, func(ctx context.Context, workDir string) (bool, string, error) {
		return true, "L2 verification passed: rule syntax valid", nil
	}, nil)

	if err != nil {
		t.Fatalf("autonomous evolution cycle failed: %v", err)
	}
	if proposal.Status != selfmod.StatusMerged {
		t.Fatalf("expected evolution proposal to be MERGED, got %s", proposal.Status)
	}

	// Verify merged in main repo
	mergedRule := filepath.Join(gitRepo, "rules", "celestial_rules.md")
	content, err := os.ReadFile(mergedRule)
	if err != nil || !strings.Contains(string(content), "orbit_period_kepler_consistency") {
		t.Fatalf("evolution fix not found in main repo: %v", err)
	}
}
