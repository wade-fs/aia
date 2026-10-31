package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type Service struct {
	repo *repository.AgentRepository
}

func NewService(repo *repository.AgentRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Upsert(ctx context.Context, a *agent.Agent) error {
	return s.repo.Upsert(ctx, a)
}

func (s *Service) Get(ctx context.Context, id domain.ID) (*agent.Agent, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]agent.Agent, error) {
	return s.repo.List(ctx)
}

func (s *Service) SetEnabled(ctx context.Context, id domain.ID, enabled bool) error {
	return s.repo.SetEnabled(ctx, id, enabled)
}

func (s *Service) LoadAgentsFromDir(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read agents dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasPrefix(name, ".") || (!strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml")) {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var ag agent.Agent
		if err := yaml.Unmarshal(data, &ag); err != nil {
			continue
		}

		if ag.ID == "" {
			ag.ID = entry.Name()
		}
		_ = s.repo.Upsert(ctx, &ag)
	}
	return nil
}
