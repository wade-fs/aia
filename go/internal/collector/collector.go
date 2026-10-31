package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/investment"
)

// CollectScope specifies the targets and bounds for a collection run.
type CollectScope struct {
	Symbols   []string
	Since     time.Time
	Limit     int
	ExtraArgs map[string]string
}

// CollectBatch holds raw documents and market data collected during a run.
type CollectBatch struct {
	SourceID    domain.ID
	SourceName  string
	Documents   []investment.RawDocument
	MarketData  []investment.MarketData
	CollectedAt time.Time
}

// Provider defines the interface for an external data source collector (AIA-05 §6, §20).
type Provider interface {
	ID() string
	Name() string
	SourceType() string
	BaseURL() string
	Collect(ctx context.Context, scope CollectScope) (*CollectBatch, error)
	HealthCheck(ctx context.Context) error
}

// Manager coordinates registered data source collectors and handles concurrency & health.
type Manager struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]Provider),
	}
}

func (m *Manager) Register(p Provider) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p == nil {
		return fmt.Errorf("provider cannot be nil")
	}
	m.providers[p.ID()] = p
	return nil
}

func (m *Manager) Get(id string) (Provider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.providers[id]
	return p, ok
}

func (m *Manager) Providers() []Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]Provider, 0, len(m.providers))
	for _, p := range m.providers {
		list = append(list, p)
	}
	return list
}
