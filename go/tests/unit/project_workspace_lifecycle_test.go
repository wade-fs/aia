package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	domainProject "github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/domain/writing"
	projectService "github.com/wade/aia/internal/service/project"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/workspace"
)

func TestWorkspaceRootResolutionAndBoundaryEnforcement(t *testing.T) {
	// 1. Empty string resolves to current directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	resolvedEmpty, err := workspace.ResolveWorkspaceRoot("")
	if err != nil {
		t.Fatalf("expected empty path to resolve cleanly, got: %v", err)
	}
	if resolvedEmpty != cwd {
		t.Errorf("expected cwd %s, got %s", cwd, resolvedEmpty)
	}

	// 2. ~ resolution
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}
	resolvedHome, err := workspace.ResolveWorkspaceRoot("~")
	if err != nil {
		t.Fatalf("expected ~ to resolve, got: %v", err)
	}
	if resolvedHome != homeDir {
		t.Errorf("expected home dir %s, got %s", homeDir, resolvedHome)
	}

	// 3. Nonexistent directory returns error
	_, err = workspace.ResolveWorkspaceRoot("/nonexistent/random/directory/aia_should_fail_12345")
	if err == nil {
		t.Errorf("expected error for nonexistent directory, got nil")
	}

	// 4. Boundary enforcement (anti-traversal)
	tmpDir := t.TempDir()
	validTarget := filepath.Join("subdir", "chapter1.md")
	safePath, err := workspace.ValidateBoundaries(tmpDir, validTarget)
	if err != nil {
		t.Fatalf("expected valid boundary check to succeed, got: %v", err)
	}
	expectedSafe := filepath.Join(tmpDir, "subdir", "chapter1.md")
	if safePath != expectedSafe {
		t.Errorf("expected %s, got %s", expectedSafe, safePath)
	}

	// Traversal attempt
	_, err = workspace.ValidateBoundaries(tmpDir, "../outside.txt")
	if err == nil {
		t.Errorf("expected error for parent directory traversal, got nil")
	}

	_, err = workspace.ValidateBoundaries(tmpDir, "../../etc/passwd")
	if err == nil {
		t.Errorf("expected error for deep traversal, got nil")
	}
}

func TestWorkspaceChangeTracking(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "note.md")
	if err := os.WriteFile(fileA, []byte("# Initial Title\nLine 1\n"), 0644); err != nil {
		t.Fatalf("write fileA failed: %v", err)
	}

	// Initial scan
	h1, changes1, err := workspace.TrackWorkspaceChanges(tmpDir, nil)
	if err != nil {
		t.Fatalf("initial change tracking failed: %v", err)
	}
	if len(changes1) != 1 || changes1[0].ChangeType != "ADDED" || changes1[0].RelPath != "note.md" {
		t.Fatalf("expected 1 ADDED change for note.md, got %+v", changes1)
	}
	if h1["note.md"] == "" {
		t.Fatalf("expected sha256 hash for note.md")
	}

	// Add fileB and modify fileA
	fileB := filepath.Join(tmpDir, "draft.txt")
	if err := os.WriteFile(fileB, []byte("draft text"), 0644); err != nil {
		t.Fatalf("write fileB failed: %v", err)
	}
	if err := os.WriteFile(fileA, []byte("# Updated Title\nLine 1\nLine 2\n"), 0644); err != nil {
		t.Fatalf("modify fileA failed: %v", err)
	}

	h2, changes2, err := workspace.TrackWorkspaceChanges(tmpDir, h1)
	if err != nil {
		t.Fatalf("second change tracking failed: %v", err)
	}
	if len(changes2) != 2 {
		t.Fatalf("expected 2 changes, got %d (%+v)", len(changes2), changes2)
	}

	changeMap := make(map[string]string)
	for _, c := range changes2 {
		changeMap[c.RelPath] = c.ChangeType
	}
	if changeMap["draft.txt"] != "ADDED" {
		t.Errorf("expected draft.txt ADDED, got %s", changeMap["draft.txt"])
	}
	if changeMap["note.md"] != "MODIFIED" {
		t.Errorf("expected note.md MODIFIED, got %s", changeMap["note.md"])
	}

	// Delete fileB
	if err := os.Remove(fileB); err != nil {
		t.Fatalf("delete fileB failed: %v", err)
	}
	_, changes3, err := workspace.TrackWorkspaceChanges(tmpDir, h2)
	if err != nil {
		t.Fatalf("third change tracking failed: %v", err)
	}
	if len(changes3) != 1 || changes3[0].ChangeType != "DELETED" || changes3[0].RelPath != "draft.txt" {
		t.Fatalf("expected 1 DELETED change for draft.txt, got %+v", changes3)
	}
}

func TestProjectService_LifecycleAndValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := repository.NewProjectRepository(db)
	svc := projectService.NewService(repo)
	ctx := context.Background()

	realYuanPath := "/home/wade/src/github/aia/novel/yuan"
	if _, err := os.Stat(realYuanPath); err != nil {
		t.Skipf("skipping test: %s not accessible", realYuanPath)
	}

	// 1. Create a project pointing to real novel/yuan
	p := &domainProject.Project{
		ID:          "proj-yuan-001",
		Name:        "元末歷史演義",
		Description: "六卷本歷史小說專案",
		Workspace:   realYuanPath,
		Status:      domainProject.StatusActive,
		DomainType:  domainProject.DomainWriting,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	// 2. Test Pause and Reopen
	if err := svc.Pause(ctx, p.ID); err != nil {
		t.Fatalf("pause project failed: %v", err)
	}
	pPaused, err := repo.Get(ctx, p.ID)
	if err != nil || pPaused.Status != domainProject.StatusPaused {
		t.Fatalf("expected status PAUSED, got %v (err: %v)", pPaused.Status, err)
	}

	if err := svc.Reopen(ctx, p.ID); err != nil {
		t.Fatalf("reopen project failed: %v", err)
	}
	pReopened, err := repo.Get(ctx, p.ID)
	if err != nil || pReopened.Status != domainProject.StatusActive {
		t.Fatalf("expected status ACTIVE, got %v (err: %v)", pReopened.Status, err)
	}

	// 3. Test DiscoverManifest
	manifest, err := svc.DiscoverManifest(ctx, p.ID)
	if err != nil {
		t.Fatalf("discover manifest failed: %v", err)
	}
	if manifest.Scale.TotalVolumes != 6 {
		t.Errorf("expected 6 volumes, got %d", manifest.Scale.TotalVolumes)
	}
	if manifest.Scale.TotalChapters != 540 {
		t.Errorf("expected 540 chapters, got %d", manifest.Scale.TotalChapters)
	}

	// 4. Test ValidateProject
	report, err := svc.ValidateProject(ctx, p.ID)
	if err != nil {
		t.Fatalf("validate project failed: %v", err)
	}
	if !report.WorkspaceValid || !report.HasManifest || !report.ManifestValid {
		t.Errorf("expected valid report for novel/yuan, got %+v", report)
	}
	if len(report.Errors) > 0 {
		t.Errorf("expected 0 validation errors for novel/yuan, got: %+v", report.Errors)
	}
}

func TestConsistencyEngine_DynamicNovelValidation(t *testing.T) {
	realYuanPath := "/home/wade/src/github/aia/novel/yuan"
	charFile := filepath.Join(realYuanPath, "characters.md")
	content, err := os.ReadFile(charFile)
	if err != nil {
		t.Skipf("characters.md not accessible at %s", charFile)
	}

	engine := writing.NewConsistencyEngine()
	characters := engine.ParseCharacters(string(content))
	if len(characters) < 20 {
		t.Fatalf("expected >20 characters parsed from characters.md, got %d", len(characters))
	}

	charMap := make(map[string]writing.HistoricalCharacter)
	for _, c := range characters {
		charMap[c.Name] = c
	}

	// Verify Tuotuo (脫脫)
	tuotuo, ok := charMap["脫脫"]
	if !ok {
		t.Fatalf("expected character 脫脫 in characters.md")
	}
	if tuotuo.KnowledgeTier != writing.TierFactHistory {
		t.Errorf("expected 脫脫 to have TierFactHistory, got %s", tuotuo.KnowledgeTier)
	}
	if tuotuo.DeathYear != 1355 && tuotuo.DeathYear != 1356 {
		t.Errorf("expected 脫脫 death year to be 1355 or 1356, got %d", tuotuo.DeathYear)
	}

	// Verify intentional fiction character (青囊子)
	qingnang, ok := charMap["青囊子"]
	if !ok {
		t.Fatalf("expected character 青囊子 in characters.md")
	}
	if qingnang.KnowledgeTier != writing.TierFiction {
		t.Errorf("expected 青囊子 to have TierFiction, got %s", qingnang.KnowledgeTier)
	}

	// Full scan of real novel/yuan
	anomalies, err := engine.ScanProject(realYuanPath, nil)
	if err != nil {
		t.Fatalf("scan project failed: %v", err)
	}
	for _, a := range anomalies {
		if !a.IsDeliberateFiction {
			t.Errorf("unexpected historical contradiction found in real novel: %+v", a)
		}
	}
}
