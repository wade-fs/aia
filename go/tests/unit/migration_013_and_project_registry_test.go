package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/config"
	domainProject "github.com/wade/aia/internal/domain/project"
	projectService "github.com/wade/aia/internal/service/project"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestMigration013_ProjectDomainType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	ver, err := db.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("failed to read schema version: %v", err)
	}
	if ver < 13 {
		t.Fatalf("expected schema version >= 13, got %d", ver)
	}

	repo := repository.NewProjectRepository(db)

	// 1. Create a writing project
	p1 := &domainProject.Project{
		ID:         "proj-writing-01",
		Name:       "長安秋暮",
		Workspace:  "/tmp/test-changan",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := repo.Create(ctx, p1); err != nil {
		t.Fatalf("failed to create writing project: %v", err)
	}

	// 2. Create a coding project
	p2 := &domainProject.Project{
		ID:         "proj-coding-01",
		Name:       "AIA Core",
		Workspace:  "/tmp/test-aia",
		DomainType: domainProject.DomainCoding,
		Status:     domainProject.StatusActive,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := repo.Create(ctx, p2); err != nil {
		t.Fatalf("failed to create coding project: %v", err)
	}

	// 3. Verify Get returns the correct DomainType
	gotP1, err := repo.Get(ctx, p1.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}
	if gotP1.DomainType != domainProject.DomainWriting {
		t.Fatalf("expected DomainWriting, got %s", gotP1.DomainType)
	}

	// 4. Verify ListByDomain filters correctly
	writingProjects, err := repo.ListByDomain(ctx, domainProject.DomainWriting)
	if err != nil {
		t.Fatalf("failed to list writing projects: %v", err)
	}
	if len(writingProjects) != 1 || writingProjects[0].ID != p1.ID {
		t.Fatalf("expected 1 writing project with ID %s, got %v", p1.ID, writingProjects)
	}

	codingProjects, err := repo.ListByDomain(ctx, domainProject.DomainCoding)
	if err != nil {
		t.Fatalf("failed to list coding projects: %v", err)
	}
	if len(codingProjects) != 1 || codingProjects[0].ID != p2.ID {
		t.Fatalf("expected 1 coding project with ID %s, got %v", p2.ID, codingProjects)
	}
}

func TestProjectService_RegisterAndConfigSeeding(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	repo := repository.NewProjectRepository(db)
	svc := projectService.NewService(repo)

	// 1. Register a writing project via Service
	p, err := svc.Register(ctx, "丹心千秋", "宋朝歷史小說", "/tmp/novel-danxin", domainProject.DomainWriting)
	if err != nil {
		t.Fatalf("failed to register project: %v", err)
	}
	if p.DomainType != domainProject.DomainWriting {
		t.Fatalf("expected domain writing, got %s", p.DomainType)
	}

	// 2. RegisterFromConfig idempotency
	seeded, err := svc.RegisterFromConfig(ctx, "seeded-yuan", "元朝", "/tmp/novel-yuan", domainProject.DomainWriting)
	if err != nil {
		t.Fatalf("failed to seed project from config: %v", err)
	}
	if seeded.ID != "seeded-yuan" {
		t.Fatalf("expected ID seeded-yuan, got %s", seeded.ID)
	}

	// Duplicate call returns existing without error
	seeded2, err := svc.RegisterFromConfig(ctx, "seeded-yuan", "元朝", "/tmp/novel-yuan", domainProject.DomainWriting)
	if err != nil {
		t.Fatalf("expected idempotent seed, got error: %v", err)
	}
	if seeded2.ID != seeded.ID {
		t.Fatalf("expected same project, got %v", seeded2)
	}

	// 3. ListByDomain through service
	list, err := svc.ListByDomain(ctx, domainProject.DomainWriting)
	if err != nil {
		t.Fatalf("failed to list by domain: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 writing projects, got %d", len(list))
	}
}

func TestConfig_DynamicPathResolution(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "aia.yaml")

	yamlContent := `
system:
  name: AIA-Test
  version: "1.0"
runtime:
  data_dir: "` + tmpDir + `/data"
  workspace_dir: "` + tmpDir + `/workspaces"
  max_concurrent_executions: 2
database:
  driver: sqlite
  path: "` + tmpDir + `/test.db"
provider:
  default: antigravity
  antigravity:
    executable: agy
domains:
  writing:
    projects:
      - id: yuan
        name: "元朝系列小說"
        path: "novel/yuan"
  coding:
    projects:
      - id: aia
        name: "AIA Core"
        path: "."
policy:
  allowed_paths:
    - "."
    - "novel"
    - "/tmp"
`
	if err := os.WriteFile(cfgFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify writing project relative path was resolved to absolute path based on cfgFile dir
	if len(cfg.Domains.Writing.Projects) != 1 {
		t.Fatalf("expected 1 writing project, got %d", len(cfg.Domains.Writing.Projects))
	}
	expectedWritingPath := filepath.Join(tmpDir, "novel", "yuan")
	if cfg.Domains.Writing.Projects[0].Path != expectedWritingPath {
		t.Fatalf("expected writing path %s, got %s", expectedWritingPath, cfg.Domains.Writing.Projects[0].Path)
	}

	// Verify coding project "." was resolved to tmpDir
	if len(cfg.Domains.Coding.Projects) != 1 {
		t.Fatalf("expected 1 coding project, got %d", len(cfg.Domains.Coding.Projects))
	}
	if cfg.Domains.Coding.Projects[0].Path != tmpDir {
		t.Fatalf("expected coding path %s, got %s", tmpDir, cfg.Domains.Coding.Projects[0].Path)
	}

	// Verify policy allowed paths were resolved
	if len(cfg.Policy.AllowedPaths) != 3 {
		t.Fatalf("expected 3 allowed paths, got %d", len(cfg.Policy.AllowedPaths))
	}
	if cfg.Policy.AllowedPaths[0] != tmpDir {
		t.Fatalf("expected allowed path 0 to be %s, got %s", tmpDir, cfg.Policy.AllowedPaths[0])
	}
	expectedNovelAllowed := filepath.Join(tmpDir, "novel")
	if cfg.Policy.AllowedPaths[1] != expectedNovelAllowed {
		t.Fatalf("expected allowed path 1 to be %s, got %s", expectedNovelAllowed, cfg.Policy.AllowedPaths[1])
	}
}

func TestWritingObserver_BookManifestIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	bookDir := filepath.Join(tmpDir, "novel", "changan")
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create bookDir: %v", err)
	}

	bookYAML := `
project:
  id: changan
  name: 長安秋暮
  type: historical_fiction
structure:
  outline: custom_outline.md
  characters: custom_characters.md
`
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), []byte(bookYAML), 0644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	// Write custom_characters.md containing contradiction
	customChars := `
# 主要人物
- 脫脫: 元朝重臣。1360 年依然在位指揮大軍。
`
	if err := os.WriteFile(filepath.Join(bookDir, "custom_characters.md"), []byte(customChars), 0644); err != nil {
		t.Fatalf("failed to write custom_characters.md: %v", err)
	}

	obs := autonomy.NewManuscriptConsistencyObserver()
	obs.RegisterProjectPath("changan", bookDir)

	ctx := context.Background()
	observations, err := obs.Observe(ctx, "changan")
	if err != nil {
		t.Fatalf("observe failed: %v", err)
	}

	if len(observations) == 0 {
		t.Fatalf("expected anomaly observation from custom_characters.md via book.yaml, got 0")
	}

	foundTuotuo := false
	for _, o := range observations {
		if o.Domain == "writing" && o.Category == "ANOMALY" {
			foundTuotuo = true
			break
		}
	}
	if !foundTuotuo {
		t.Fatalf("expected ANOMALY observation for 脫脫, got %v", observations)
	}
}

