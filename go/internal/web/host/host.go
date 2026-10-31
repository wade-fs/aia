package host

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type Host struct {
	AuthMgr   *AuthManager
	EventBus  *eventbus.Bus
	AuditRepo *repository.AuditRepository
	Modules   *ModuleRegistry
	rootMux   *http.ServeMux
	apiMux    *http.ServeMux
	staticFS  fs.FS
}

func NewHost(authMgr *AuthManager, bus *eventbus.Bus, auditRepo *repository.AuditRepository, staticFS fs.FS) *Host {
	h := &Host{
		AuthMgr:   authMgr,
		EventBus:  bus,
		AuditRepo: auditRepo,
		Modules:   NewModuleRegistry(),
		rootMux:   http.NewServeMux(),
		apiMux:    http.NewServeMux(),
		staticFS:  staticFS,
	}

	h.setupStaticAndRoot()
	return h
}

// APIMux returns the router where console endpoints are registered.
func (h *Host) APIMux() *http.ServeMux {
	return h.apiMux
}

// RegisterAgentModule registers an agent module with an isolated, dedicated mux.
// Host mounts this mux onto /api/v1/agents/{agent_id}/.
func (h *Host) RegisterAgentModule(m AgentWebModule) error {
	agentMux, err := h.Modules.Register(m)
	if err != nil {
		return err
	}

	// Mount isolated mux to /api/v1/agents/{id}/
	prefix := fmt.Sprintf("/api/v1/agents/%s/", m.AgentID())
	h.apiMux.Handle(prefix, http.StripPrefix(strings.TrimSuffix(prefix, "/"), agentMux))
	return nil
}

func (h *Host) setupStaticAndRoot() {
	// 1. Mount API mux with middleware under /api/v1/
	h.rootMux.Handle("/api/v1/", h.AuthMiddleware(h.AuditMiddleware(h.apiMux)))

	// 2. SSE Event Stream
	h.apiMux.HandleFunc("/api/v1/events", h.HandleSSE)

	// 3. Static asset file server (Native HTML/CSS/JS embedded via go:embed)
	if h.staticFS != nil {
		fileServer := http.FileServer(http.FS(h.staticFS))
		h.rootMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// If request is for an API path that wasn't matched, return 404 JSON
			if strings.HasPrefix(r.URL.Path, "/api/") {
				reqID, _ := r.Context().Value(reqIDKey).(string)
				WriteError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint not found", reqID)
				return
			}

			// SPA Fallback: if not an asset request (css/js/images), serve index.html
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path != "" && !strings.Contains(path, ".") {
				r.URL.Path = "/"
			}

			w.Header().Set("Cache-Control", "no-cache")
			fileServer.ServeHTTP(w, r)
		})
	}
}

// Handler returns the fully wrapped HTTP handler for the server.
func (h *Host) Handler() http.Handler {
	return RequestIDMiddleware(CSRFProtection(h.rootMux))
}
