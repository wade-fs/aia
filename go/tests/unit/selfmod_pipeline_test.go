package unit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/selfmod"
	selfmodSvc "github.com/wade/aia/internal/service/selfmod"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestScopeClassifier_ClassificationRules(t *testing.T) {
	classifier := selfmodSvc.NewScopeClassifier()

	// 1. Level C: Governance / Security (Strictly Forbidden / Immutable)
	forbiddenPaths := [][]string{
		{"internal/policy/evaluator.go"},
		{"internal/service/governance/service.go"},
		{"internal/domain/governance/models.go"},
		{"internal/service/approval/service.go"},
		{"internal/storage/sqlite/migrations/014_schema.go"},
		{".git/config"},
		{"config/service.env"},
		{"data/sqlite.db"},
		{"data/sqlite.db-wal"},
	}
	for _, p := range forbiddenPaths {
		level, err := classifier.Classify(p)
		if err != selfmod.ErrLevelCModificationForbidden || level != selfmod.LevelCGovernance {
			t.Errorf("expected LevelCGovernance for %v, got level=%s err=%v", p, level, err)
		}
	}

	// 2. Level B: AIA Core Non-Governance (Human Approval Required)
	corePaths := [][]string{
		{"internal/service/task/service.go"},
		{"internal/autonomy/engine.go"},
		{"internal/worker/manager.go"},
		{"internal/domain/task/task.go"},
		{"cmd/aia/main.go"},
		{"go.mod"},
	}
	for _, p := range corePaths {
		level, err := classifier.Classify(p)
		if err != nil || level != selfmod.LevelBCore {
			t.Errorf("expected LevelBCore for %v, got level=%s err=%v", p, level, err)
		}
	}

	// 3. Level A: Domain rules, prompts, profiles, docs (Autonomous Sandbox + L2 verification)
	domainPaths := [][]string{
		{"rules/consistency_rules.yaml"},
		{"prompts/writing_prompt.md"},
		{"profiles/space_opera.json"},
		{"novel/yuan/characters.md"},
		{"docs/roadmap.md"},
	}
	for _, p := range domainPaths {
		level, err := classifier.Classify(p)
		if err != nil || level != selfmod.LevelADomain {
			t.Errorf("expected LevelADomain for %v, got level=%s err=%v", p, level, err)
		}
	}

	// 4. Mixed targets: If any file is Level C -> Level C wins
	mixedWithC := []string{"rules/space.yaml", "internal/policy/evaluator.go"}
	levelC, errC := classifier.Classify(mixedWithC)
	if errC != selfmod.ErrLevelCModificationForbidden || levelC != selfmod.LevelCGovernance {
		t.Errorf("expected Level C to dominate mixed targets, got level=%s err=%v", levelC, errC)
	}

	// If mix of Level A and Level B -> Level B wins
	mixedWithB := []string{"rules/space.yaml", "internal/service/task/service.go"}
	levelB, errB := classifier.Classify(mixedWithB)
	if errB != nil || levelB != selfmod.LevelBCore {
		t.Errorf("expected Level B to dominate Level A in mixed targets, got level=%s err=%v", levelB, errB)
	}
}

func TestSelfModRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	repo := repository.NewSelfModRepository(db)

	prop := &selfmod.SelfModificationProposal{
		ID:          "prop-test-01",
		Title:       "Test Proposal",
		Description: "Testing repository persistence",
		Rationale:   "Unit verification",
		TargetFiles: []string{"rules/test.yaml"},
		Patches: map[string]string{
			"rules/test.yaml": "rules: []",
		},
		ScopeLevel:       selfmod.LevelADomain,
		RequiresApproval: false,
		Status:           selfmod.StatusDraft,
		CreatedAt:        time.Now().UTC(),
	}

	// Create
	if err := repo.Create(ctx, prop); err != nil {
		t.Fatalf("create proposal failed: %v", err)
	}

	// Get
	retrieved, err := repo.Get(ctx, prop.ID)
	if err != nil {
		t.Fatalf("get proposal failed: %v", err)
	}
	if retrieved.Title != prop.Title || retrieved.ScopeLevel != prop.ScopeLevel {
		t.Errorf("retrieved proposal mismatch: %+v", retrieved)
	}

	// Update
	prop.Status = selfmod.StatusVerified
	prop.BranchName = "aia-test-branch"
	prop.PostCommitSHA = "sha256-abcdef123"
	if err := repo.Update(ctx, prop); err != nil {
		t.Fatalf("update proposal failed: %v", err)
	}

	updated, err := repo.Get(ctx, prop.ID)
	if err != nil {
		t.Fatalf("get updated proposal failed: %v", err)
	}
	if updated.Status != selfmod.StatusVerified || updated.PostCommitSHA != "sha256-abcdef123" {
		t.Errorf("updated proposal mismatch: %+v", updated)
	}

	// ListByStatus
	list, err := repo.ListByStatus(ctx, selfmod.StatusVerified)
	if err != nil {
		t.Fatalf("list by status failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != prop.ID {
		t.Fatalf("expected 1 verified proposal, got %d", len(list))
	}
}

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

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# AIA Self-Mod Repository\nInitial baseline commit\n"), 0644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}
	gitRun("add", "README.md")
	gitRun("commit", "-m", "Initial baseline commit")

	return dir
}

func TestPipeline_MultiTierVerification_And_Retries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	repo := repository.NewSelfModRepository(db)
	gitRepo := setupSelfModTestGitRepo(t)
	pipeline := selfmodSvc.NewPipeline(gitRepo, repo, nil, nil, nil, nil)

	// Test L1, L2, L3 verification and retry on failure
	input := selfmodSvc.ProposalInput{
		Title:          "Multi-tier verification proposal",
		Description:    "Verify 3-tier stages and retry logic",
		Rationale:      "AIA-14 §7.1 and §7.4",
		TargetFiles:    []string{"rules/profile.json"},
		Patches: map[string]string{
			"rules/profile.json": "{\"invalid_json\": ", // Intentionally invalid on attempt 1
		},
		ValidationPlan: []string{selfmod.VerifLevelL1Syntax, selfmod.VerifLevelL2Unit, selfmod.VerifLevelL3Integration},
	}

	prop, err := pipeline.Propose(ctx, input)
	if err != nil {
		t.Fatalf("Propose failed: %v", err)
	}
	if len(prop.ValidationPlan) != 3 {
		t.Fatalf("expected 3 stages in validation plan, got %d", len(prop.ValidationPlan))
	}

	attemptsMade := 0
	verifiedProp, err := pipeline.ExecuteVerificationWithOptions(ctx, prop.ID, selfmodSvc.VerificationOptions{
		MaxRetries: 2,
		RetryPatchModifier: func(ctx context.Context, attempt int, lastErr string, currentPatches map[string]string) (map[string]string, error) {
			attemptsMade = attempt
			return map[string]string{
				"rules/profile.json": "{\"valid\": true}",
			}, nil
		},
		MultiLevelRunner: func(ctx context.Context, workDir string, level string) (*selfmod.TestVerificationResult, error) {
			switch level {
			case selfmod.VerifLevelL1Syntax:
				content, err := os.ReadFile(filepath.Join(workDir, "rules/profile.json"))
				if err != nil || strings.Contains(string(content), "invalid") {
					return &selfmod.TestVerificationResult{
						Level:  selfmod.VerifLevelL1Syntax,
						Passed: false,
						Output: "Syntax validation failed: malformed JSON",
					}, nil
				}
				return &selfmod.TestVerificationResult{
					Level:  selfmod.VerifLevelL1Syntax,
					Passed: true,
					Output: "L1 Syntax check passed",
				}, nil
			case selfmod.VerifLevelL2Unit:
				return &selfmod.TestVerificationResult{
					Level:  selfmod.VerifLevelL2Unit,
					Passed: true,
					Output: "L2 Unit tests passed (10/10)",
				}, nil
			case selfmod.VerifLevelL3Integration:
				return &selfmod.TestVerificationResult{
					Level:  selfmod.VerifLevelL3Integration,
					Passed: true,
					Output: "L3 Integration regression tests passed",
				}, nil
			default:
				return nil, fmt.Errorf("unexpected level %s", level)
			}
		},
	})

	if err != nil {
		t.Fatalf("verification failed despite retry: %v", err)
	}
	if attemptsMade != 2 {
		t.Errorf("expected 2 attempts made, got %d", attemptsMade)
	}
	if len(verifiedProp.VerificationResults) != 3 {
		t.Fatalf("expected 3 verification results, got %d", len(verifiedProp.VerificationResults))
	}
	if verifiedProp.Status != selfmod.StatusMerged { // Level A merges directly
		t.Errorf("expected status MERGED, got %s", verifiedProp.Status)
	}
}

func TestPipeline_HumanReviewOutcomes_RequestChanges_And_Defer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	repo := repository.NewSelfModRepository(db)
	gitRepo := setupSelfModTestGitRepo(t)
	pipeline := selfmodSvc.NewPipeline(gitRepo, repo, nil, nil, nil, nil)

	prop := &selfmod.SelfModificationProposal{
		ID:               "prop-review-001",
		Title:            "Test Review States",
		Description:      "AIA-14 §4.8 review state machine",
		Rationale:        "Review states test",
		TargetFiles:      []string{"rules/demo.yaml"},
		Patches:          map[string]string{"rules/demo.yaml": "demo: true"},
		ScopeLevel:       selfmod.LevelBCore,
		RequiresApproval: true,
		Status:           selfmod.StatusPendingApproval,
		CreatedAt:        time.Now().UTC(),
	}
	if err := repo.Create(ctx, prop); err != nil {
		t.Fatalf("create proposal failed: %v", err)
	}

	human := governance.Actor{Type: "USER", ID: "reviewer-wade"}
	agent := governance.Actor{Type: "AGENT", ID: "agent-personal"}

	// 1. Agent cannot request changes or defer
	if err := pipeline.RequestChanges(ctx, prop.ID, agent, "Needs tests"); err != selfmod.ErrAgentSelfApprovalForbidden {
		t.Errorf("expected agent rejection, got %v", err)
	}

	// 2. Human requests changes
	if err := pipeline.RequestChanges(ctx, prop.ID, human, "Please refactor helper functions"); err != nil {
		t.Fatalf("RequestChanges failed: %v", err)
	}
	updated, _ := repo.Get(ctx, prop.ID)
	if updated.Status != selfmod.StatusChangesRequested {
		t.Errorf("expected StatusChangesRequested, got %s", updated.Status)
	}
	if !strings.Contains(updated.RCALog, "Please refactor helper functions") {
		t.Errorf("expected feedback in RCALog, got %s", updated.RCALog)
	}

	// 3. Human defers proposal
	updated.Status = selfmod.StatusPendingApproval
	_ = repo.Update(ctx, updated)

	if err := pipeline.DeferProposal(ctx, prop.ID, human, "Postponed until next release window"); err != nil {
		t.Fatalf("DeferProposal failed: %v", err)
	}
	deferred, _ := repo.Get(ctx, prop.ID)
	if deferred.Status != selfmod.StatusDeferred {
		t.Errorf("expected StatusDeferred, got %s", deferred.Status)
	}
	if !strings.Contains(deferred.RCALog, "Postponed until next release window") {
		t.Errorf("expected reason in RCALog, got %s", deferred.RCALog)
	}
}

func TestPipeline_PostDeploymentMonitoring_And_Rollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	repo := repository.NewSelfModRepository(db)
	gitRepo := setupSelfModTestGitRepo(t)
	pipeline := selfmodSvc.NewPipeline(gitRepo, repo, nil, nil, nil, nil)

	// Propose, verify, and merge a proposal
	input := selfmodSvc.ProposalInput{
		Title:       "Add Feature Profile",
		Description: "Monitoring test",
		Rationale:   "AIA-14 §4.10",
		TargetFiles: []string{"rules/feature.yaml"},
		Patches:     map[string]string{"rules/feature.yaml": "feature_enabled: true\n"},
	}
	prop, err := pipeline.Propose(ctx, input)
	if err != nil {
		t.Fatalf("Propose failed: %v", err)
	}

	verified, err := pipeline.ExecuteSandboxVerification(ctx, prop.ID, func(ctx context.Context, workDir string) (bool, string, error) {
		return true, "all tests passed", nil
	})
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}
	if verified.Status != selfmod.StatusMerged {
		t.Fatalf("expected StatusMerged, got %s", verified.Status)
	}

	// 1. Healthy metrics
	healthyMetrics := selfmod.PostDeployMetrics{
		ErrorRate:         0.01,
		CrashRate:         0.0,
		AutonomyMetric:    0.95,
		ObservedAt:        time.Now().UTC(),
	}
	reportH, err := pipeline.MonitorPostDeployment(ctx, verified.ID, healthyMetrics)
	if err != nil || !reportH.Healthy || reportH.TriggersRollback {
		t.Fatalf("expected healthy report, got %+v (err: %v)", reportH, err)
	}

	// 2. Unhealthy metrics (CrashRate > 0)
	badMetrics := selfmod.PostDeployMetrics{
		ErrorRate:         0.08,
		CrashRate:         0.02,
		AutonomyMetric:    0.50,
		ObservedAt:        time.Now().UTC(),
	}
	reportBad, err := pipeline.MonitorPostDeployment(ctx, verified.ID, badMetrics)
	if err != nil {
		t.Fatalf("MonitorPostDeployment failed: %v", err)
	}
	if reportBad.Healthy || !reportBad.TriggersRollback {
		t.Fatalf("expected unhealthy report triggering rollback, got %+v", reportBad)
	}
	if len(reportBad.Issues) < 3 {
		t.Errorf("expected 3 issues detected, got %d: %v", len(reportBad.Issues), reportBad.Issues)
	}

	// 3. Rollback merged proposal (AIA-14 §8.1)
	if err := pipeline.RollbackProposal(ctx, verified.ID, "Post-deployment crash rate spike"); err != nil {
		t.Fatalf("RollbackProposal failed: %v", err)
	}

	rolledBack, _ := repo.Get(ctx, verified.ID)
	if rolledBack.Status != selfmod.StatusRolledBack {
		t.Errorf("expected StatusRolledBack, got %s", rolledBack.Status)
	}
}
