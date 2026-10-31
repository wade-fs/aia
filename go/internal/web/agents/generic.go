package agents

import (
	"net/http"

	agentService "github.com/wade/aia/internal/service/agent"
	"github.com/wade/aia/internal/web/host"
)

type GenericAgentWebModule struct {
	agentID  string
	navLabel string
	path     string
	icon     string
	order    int
	svc      *agentService.Service
}

func NewCodingWebModule(svc *agentService.Service) *GenericAgentWebModule {
	return &GenericAgentWebModule{
		agentID:  "coding",
		navLabel: "Coding Agent",
		path:     "#/coding",
		icon:     "code",
		order:    20,
		svc:      svc,
	}
}

func NewWritingWebModule(svc *agentService.Service) *GenericAgentWebModule {
	return &GenericAgentWebModule{
		agentID:  "writing",
		navLabel: "Writing Agent",
		path:     "#/writing",
		icon:     "edit-3",
		order:    30,
		svc:      svc,
	}
}

func (m *GenericAgentWebModule) AgentID() string {
	return m.agentID
}

func (m *GenericAgentWebModule) NavItems() []host.NavItem {
	return []host.NavItem{
		{
			ID:       m.agentID,
			Label:    m.navLabel,
			Path:     m.path,
			Icon:     m.icon,
			Order:    m.order,
			Required: m.agentID + ".read",
		},
	}
}

func (m *GenericAgentWebModule) RegisterRoutes(mux *http.ServeMux) error {
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		ag, err := m.svc.Get(r.Context(), m.agentID+"-agent")
		if err != nil {
			// Try without suffix
			ag, err = m.svc.Get(r.Context(), m.agentID)
		}
		if err != nil {
			host.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Agent info unavailable", reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, ag, reqID)
	})
	return nil
}
