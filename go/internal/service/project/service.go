package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/workspace"
	"gopkg.in/yaml.v3"
)

type Service struct {
	repo *repository.ProjectRepository
}

func NewService(repo *repository.ProjectRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) EnsureDefault(ctx context.Context, workspacePath string) (*project.Project, error) {
	p, err := s.repo.Get(ctx, "default")
	if err == nil && p != nil {
		return p, nil
	}
	if workspacePath == "" {
		workspacePath, _ = os.Getwd()
	}
	p = &project.Project{
		ID:          "default",
		Name:        "Default Project",
		Description: "Default project for personal agent tasks",
		Workspace:   workspacePath,
		DomainType:  project.DomainGeneral,
		Status:      project.StatusActive,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Create(ctx context.Context, name, description, workspacePath string) (*project.Project, error) {
	return s.Register(ctx, name, description, workspacePath, project.DomainGeneral)
}

// Register creates a new project with an explicit domain type.
// This is the primary entry point for `aia project register --type <domain>`.
// workspacePath supports ~/ expansion; if empty it defaults to the current
// working directory so no absolute path needs to be hardcoded by callers.
func (s *Service) Register(ctx context.Context, name, description, workspacePath string, dt project.DomainType) (*project.Project, error) {
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}
	if workspacePath == "" {
		workspacePath, _ = os.Getwd()
	}

	p := &project.Project{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Workspace:   workspacePath,
		DomainType:  dt,
		Status:      project.StatusActive,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// RegisterFromConfig seeds the project registry from config file entries.
// It is idempotent: if a project with the same ID already exists, it is skipped.
func (s *Service) RegisterFromConfig(ctx context.Context, id, name, workspacePath string, dt project.DomainType) (*project.Project, error) {
	existing, err := s.repo.Get(ctx, domain.ID(id))
	if err == nil && existing != nil {
		// Already registered; update workspace in case it moved.
		return existing, nil
	}
	p := &project.Project{
		ID:         domain.ID(id),
		Name:       name,
		Workspace:  workspacePath,
		DomainType: dt,
		Status:     project.StatusActive,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to seed project %s from config: %w", id, err)
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, id domain.ID) (*project.Project, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]project.Project, error) {
	return s.repo.List(ctx)
}

// ListByDomain returns ACTIVE projects for the given domain type.
// Used at boot to seed domain observers without hardcoded paths.
func (s *Service) ListByDomain(ctx context.Context, dt project.DomainType) ([]project.Project, error) {
	return s.repo.ListByDomain(ctx, dt)
}

func (s *Service) Archive(ctx context.Context, id domain.ID) error {
	return s.repo.UpdateStatus(ctx, id, project.StatusArchived)
}

func (s *Service) Delete(ctx context.Context, id domain.ID) error {
	if string(id) == "default" {
		return fmt.Errorf("cannot delete default project")
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) Pause(ctx context.Context, id domain.ID) error {
	return s.repo.UpdateStatus(ctx, id, project.StatusPaused)
}

func (s *Service) Reopen(ctx context.Context, id domain.ID) error {
	return s.repo.UpdateStatus(ctx, id, project.StatusActive)
}

type ProjectValidationReport struct {
	ProjectID      string   `json:"project_id"`
	WorkspacePath  string   `json:"workspace_path"`
	WorkspaceValid bool     `json:"workspace_valid"`
	HasManifest    bool     `json:"has_manifest"`
	ManifestValid  bool     `json:"manifest_valid"`
	MissingFiles   []string `json:"missing_files,omitempty"`
	Errors         []string `json:"errors,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

// DiscoverManifest finds and parses book.yaml in the project's workspace.
func (s *Service) DiscoverManifest(ctx context.Context, id domain.ID) (*writing.BookManifest, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}
	wsRoot, err := workspace.ResolveWorkspaceRoot(p.Workspace)
	if err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(wsRoot, "book.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest book.yaml not found: %w", err)
	}

	var m writing.BookManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse book.yaml: %w", err)
	}
	return &m, nil
}

// ValidateProject checks workspace existence, manifest syntax, and declared files on disk.
func (s *Service) ValidateProject(ctx context.Context, id domain.ID) (*ProjectValidationReport, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	report := &ProjectValidationReport{
		ProjectID:     string(id),
		WorkspacePath: p.Workspace,
	}

	wsRoot, err := workspace.ResolveWorkspaceRoot(p.Workspace)
	if err != nil {
		report.WorkspaceValid = false
		report.Errors = append(report.Errors, fmt.Sprintf("invalid workspace path: %v", err))
		return report, nil
	}
	report.WorkspaceValid = true

	manifestPath := filepath.Join(wsRoot, "book.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		report.HasManifest = false
		report.Warnings = append(report.Warnings, "book.yaml is missing; project can be initialized with InitializeWritingProject")
		return report, nil
	}
	report.HasManifest = true

	var m writing.BookManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		report.ManifestValid = false
		report.Errors = append(report.Errors, fmt.Sprintf("book.yaml syntax error: %v", err))
		return report, nil
	}

	if err := m.Validate(); err != nil {
		report.ManifestValid = false
		report.Errors = append(report.Errors, fmt.Sprintf("book.yaml validation error: %v", err))
	} else {
		report.ManifestValid = true
	}

	checkFile := func(rel string) {
		if rel == "" || rel == "." {
			return
		}
		target := filepath.Join(wsRoot, rel)
		if _, err := os.Stat(target); err != nil {
			report.MissingFiles = append(report.MissingFiles, rel)
		}
	}

	checkFile(m.Structure.Outline)
	checkFile(m.Structure.Characters)
	checkFile(m.Structure.Rules)
	checkFile(m.Structure.Guidelines)
	checkFile(m.Structure.Chapters)
	if m.Laboratory.Enabled {
		checkFile(m.Laboratory.Specification)
		checkFile(m.Laboratory.Guidelines)
	}

	return report, nil
}

// InitializeWritingProject scans the workspace, generates book.yaml if missing, and validates it (Plan.md §4.3).
func (s *Service) InitializeWritingProject(ctx context.Context, id domain.ID) (*writing.BookManifest, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}
	wsRoot, err := workspace.ResolveWorkspaceRoot(p.Workspace)
	if err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(wsRoot, "book.yaml")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var existing writing.BookManifest
		if err := yaml.Unmarshal(data, &existing); err == nil && existing.Validate() == nil {
			return &existing, nil
		}
	}

	// Discover project structure from files
	manifest := writing.BookManifest{}
	manifest.Project.ID = string(p.ID)
	manifest.Project.Name = p.Name
	manifest.Project.Language = "zh-TW"
	manifest.Project.Type = writing.TypeHistoricalFiction
	manifest.Project.DomainAgent = "writing"
	manifest.Project.Status = "in_progress"

	manifest.Storage.Type = "filesystem"
	manifest.Storage.Root = wsRoot

	manifest.VersionControl.Type = "git"
	manifest.VersionControl.Enabled = true
	manifest.VersionControl.DefaultBranch = "main"

	manifest.Governance.DirectMainWrite = false
	manifest.Governance.RequireDiffReview = true
	manifest.Governance.RequireHumanApprovalForMajorChanges = true

	manifest.Continuity.Enabled = true
	manifest.Memory.Enabled = true

	// Auto-detect structure files
	if _, err := os.Stat(filepath.Join(wsRoot, "outline.md")); err == nil {
		manifest.Structure.Outline = "outline.md"
	}
	if _, err := os.Stat(filepath.Join(wsRoot, "characters.md")); err == nil {
		manifest.Structure.Characters = "characters.md"
	}
	if _, err := os.Stat(filepath.Join(wsRoot, "chapter_writer_rules.md")); err == nil {
		manifest.Structure.Rules = "chapter_writer_rules.md"
	}
	if _, err := os.Stat(filepath.Join(wsRoot, "chapters")); err == nil {
		manifest.Structure.Chapters = "chapters/"
	}

	// Detect volumes count
	volCount := 0
	for i := 1; i <= 20; i++ {
		fn := fmt.Sprintf("volume%d.md", i)
		if _, err := os.Stat(filepath.Join(wsRoot, fn)); err == nil {
			volCount++
		}
	}
	if volCount > 0 {
		manifest.Scale.TotalVolumes = volCount
		manifest.Scale.VolumeStructure = "files"
		manifest.Scale.ChaptersPerVolume = 90
		manifest.Scale.TotalChapters = volCount * 90
	}

	// Detect dual-world laboratory
	if _, err := os.Stat(filepath.Join(wsRoot, "laboratory.md")); err == nil {
		manifest.Laboratory.Enabled = true
		manifest.Laboratory.Specification = "laboratory.md"
		manifest.Laboratory.Perspective = "外星文明人類觀測實驗室"
		manifest.Laboratory.NonInterference = true
		manifest.Project.Type = writing.TypeDualWorldHistorical
		if _, gErr := os.Stat(filepath.Join(wsRoot, "laboratory_writing_guidelines.md")); gErr == nil {
			manifest.Laboratory.Guidelines = "laboratory_writing_guidelines.md"
		}
	}

	// Set writing policy defaults
	manifest.WritingPolicy.HistoricalAccuracy = "high"
	manifest.WritingPolicy.RequireConsistencyCheck = true
	manifest.WritingPolicy.AntiAnachronism = true
	manifest.WritingPolicy.AntiTeleportation = true
	manifest.WritingPolicy.Classifications = []string{"historical_fact", "reasonable_inference", "fictional_content"}

	data, err := yaml.Marshal(&manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal generated book.yaml: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write book.yaml: %w", err)
	}

	return &manifest, nil
}
