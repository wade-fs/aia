package host

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type NavItem struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Path     string `json:"path"`
	Icon     string `json:"icon,omitempty"`
	Order    int    `json:"order"`
	Required string `json:"required_permission,omitempty"`
}

type AgentWebModule interface {
	AgentID() string
	RegisterRoutes(mux *http.ServeMux) error
	NavItems() []NavItem
}

type ModuleRegistry struct {
	mu      sync.RWMutex
	modules map[string]AgentWebModule
	muxes   map[string]*http.ServeMux
}

func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]AgentWebModule),
		muxes:   make(map[string]*http.ServeMux),
	}
}

func (r *ModuleRegistry) Register(m AgentWebModule) (*http.ServeMux, error) {
	if m == nil {
		return nil, fmt.Errorf("module cannot be nil")
	}
	id := strings.TrimSpace(m.AgentID())
	if id == "" {
		return nil, fmt.Errorf("agent id cannot be empty")
	}

	// Reserved system names
	reserved := map[string]bool{
		"system": true, "tasks": true, "agents": true, "memory": true,
		"pending": true, "approvals": true, "audit": true, "auth": true, "events": true,
	}
	if reserved[id] {
		return nil, fmt.Errorf("agent id '%s' conflicts with system reserved namespace", id)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.modules[id]; exists {
		return nil, fmt.Errorf("agent web module '%s' is already registered", id)
	}

	// Create dedicated, isolated ServeMux for this agent
	agentMux := http.NewServeMux()
	if err := m.RegisterRoutes(agentMux); err != nil {
		return nil, fmt.Errorf("failed to register routes for agent %s: %w", id, err)
	}

	r.modules[id] = m
	r.muxes[id] = agentMux
	return agentMux, nil
}

func (r *ModuleRegistry) Get(id string) (AgentWebModule, *http.ServeMux, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[id]
	mux := r.muxes[id]
	return m, mux, ok
}

func (r *ModuleRegistry) AllNavItems() []NavItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var items []NavItem
	for _, m := range r.modules {
		items = append(items, m.NavItems()...)
	}
	return items
}
