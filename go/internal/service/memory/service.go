package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/memory"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type Service struct {
	repo *repository.MemoryRepository
}

func NewService(repo *repository.MemoryRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, m *memory.Memory) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.Confidence <= 0 {
		m.Confidence = 1.0
	}
	if m.Status == "" {
		m.Status = "ACTIVE"
	}
	if m.Version <= 0 {
		m.Version = 1
	}
	return s.repo.Create(ctx, m)
}

func (s *Service) Search(ctx context.Context, q repository.MemoryQuery) ([]memory.Memory, error) {
	return s.repo.Search(ctx, q)
}

func (s *Service) Delete(ctx context.Context, id domain.ID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) CreateCandidate(ctx context.Context, c *memory.MemoryCandidate) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.Confidence <= 0 {
		c.Confidence = 0.8
	}
	c.ReviewStatus = memory.ReviewPending
	return s.repo.CreateCandidate(ctx, c)
}

func (s *Service) ListCandidates(ctx context.Context, status memory.ReviewStatus) ([]memory.MemoryCandidate, error) {
	return s.repo.ListCandidates(ctx, status)
}

func (s *Service) ApproveCandidate(ctx context.Context, candidateID domain.ID) error {
	c, err := s.repo.GetCandidate(ctx, candidateID)
	if err != nil {
		return err
	}
	if c.ReviewStatus != memory.ReviewPending {
		return fmt.Errorf("candidate is already reviewed: %s", c.ReviewStatus)
	}

	m := &memory.Memory{
		ID:         uuid.New().String(),
		Type:       c.Type,
		Scope:      c.Scope,
		ProjectID:  c.ProjectID,
		AgentID:    c.AgentID,
		Title:      c.Title,
		Content:    c.Content,
		SourceType: c.SourceType,
		SourceID:   c.SourceID,
		Confidence: c.Confidence,
		Status:     "ACTIVE",
		Version:    1,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, m); err != nil {
		return fmt.Errorf("failed to create memory from candidate: %w", err)
	}

	return s.repo.UpdateCandidateStatus(ctx, candidateID, memory.ReviewApproved)
}

func (s *Service) RejectCandidate(ctx context.Context, candidateID domain.ID) error {
	return s.repo.UpdateCandidateStatus(ctx, candidateID, memory.ReviewRejected)
}
