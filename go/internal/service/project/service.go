package project

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/storage/sqlite/repository"
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
