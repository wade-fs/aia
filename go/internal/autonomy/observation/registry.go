package observation

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
)

// Registry manages domain and runtime EnvironmentObservers for Step 1 (Observe).
type Registry struct {
	mu        sync.RWMutex
	observers map[string]domainAutonomy.EnvironmentObserver
}

func NewRegistry() *Registry {
	return &Registry{
		observers: make(map[string]domainAutonomy.EnvironmentObserver),
	}
}

func (r *Registry) Register(name string, obs domainAutonomy.EnvironmentObserver) {
	if obs == nil || name == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers[name] = obs
}

func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.observers, name)
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.observers))
	for k := range r.observers {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ObserveAll invokes all registered observers in deterministic name order.
func (r *Registry) ObserveAll(ctx context.Context, projectID string) []domainAutonomy.Observation {
	r.mu.RLock()
	names := make([]string, 0, len(r.observers))
	snapshot := make(map[string]domainAutonomy.EnvironmentObserver, len(r.observers))
	for k, v := range r.observers {
		names = append(names, k)
		snapshot[k] = v
	}
	r.mu.RUnlock()

	sort.Strings(names)
	now := time.Now().UTC()
	var out []domainAutonomy.Observation

	for _, name := range names {
		obs := snapshot[name]
		list, err := obs.Observe(ctx, projectID)
		if err != nil {
			out = append(out, domainAutonomy.Observation{
				Source:     name,
				Category:   domainAutonomy.ObsCategoryAnomaly,
				Summary:    fmt.Sprintf("Observer %q reported error: %v", name, err),
				Confidence: domainAutonomy.ConfidenceConfirmed,
				Severity:   6,
				Timestamp:  now,
			})
			continue
		}
		for _, item := range list {
			if item.Source == "" {
				item.Source = name
			}
			if item.Timestamp.IsZero() {
				item.Timestamp = now
			}
			out = append(out, item)
		}
	}

	return out
}
