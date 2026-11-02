package unit

import (
	"context"
	"testing"
	"time"

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
