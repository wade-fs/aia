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
		Status:      project.StatusActive,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Create(ctx context.Context, name, description, workspacePath string) (*project.Project, error) {
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
		Status:      project.StatusActive,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, id domain.ID) (*project.Project, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]project.Project, error) {
	return s.repo.List(ctx)
}

func (s *Service) Archive(ctx context.Context, id domain.ID) error {
	return s.repo.UpdateStatus(ctx, id, project.StatusArchived)
}
