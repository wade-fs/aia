package agents

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/config"
	domainProject "github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/orchestrator"
	agentService "github.com/wade/aia/internal/service/agent"
	execService "github.com/wade/aia/internal/service/execution"
	projectService "github.com/wade/aia/internal/service/project"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/web/host"
)

type CodingWebModule struct {
	agentSvc   *agentService.Service
	projSvc    *projectService.Service
	taskSvc    *taskService.Service
	sched      *orchestrator.Scheduler
	execSvc    *execService.Service
	codingObs  *autonomy.CodeHealthObserver
}

func NewCodingWebModuleWithDeps(
	agentSvc *agentService.Service,
	projSvc *projectService.Service,
	taskSvc *taskService.Service,
	sched *orchestrator.Scheduler,
	execSvc *execService.Service,
	codingObs *autonomy.CodeHealthObserver,
) *CodingWebModule {
	return &CodingWebModule{
		agentSvc:  agentSvc,
		projSvc:   projSvc,
		taskSvc:   taskSvc,
		sched:     sched,
		execSvc:   execSvc,
		codingObs: codingObs,
	}
}

func (m *CodingWebModule) AgentID() string {
	return "coding"
}

func (m *CodingWebModule) NavItems() []host.NavItem {
	return []host.NavItem{
		{
			ID:       "coding",
			Label:    "Coding Agent",
			Path:     "#/coding",
			Icon:     "code",
			Order:    20,
			Required: "coding.read",
		},
	}
}

func (m *CodingWebModule) RegisterRoutes(mux *http.ServeMux) error {
	mux.HandleFunc("/info", m.handleInfo)
	mux.HandleFunc("/instructions", m.handleInstructions)
	mux.HandleFunc("/projects", m.handleProjects)
	mux.HandleFunc("/scan", m.handleScan)
	mux.HandleFunc("/prompt", m.handlePrompt)
	return nil
}

func (m *CodingWebModule) handleInfo(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ag, err := m.agentSvc.Get(r.Context(), "coding-agent")
	if err != nil {
		ag, err = m.agentSvc.Get(r.Context(), "coding")
	}
	if err != nil {
		host.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Coding agent info unavailable", reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, ag, reqID)
}

func (m *CodingWebModule) handleInstructions(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only PUT or POST allowed", reqID)
		return
	}

	var req struct {
		Instructions string `json:"instructions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		host.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON payload", reqID)
		return
	}

	ctx := r.Context()
	ag, err := m.agentSvc.Get(ctx, "coding-agent")
	if err != nil {
		host.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Coding agent not found", reqID)
		return
	}

	ag.Instructions = req.Instructions
	ag.Version++
	if err := m.agentSvc.Upsert(ctx, ag); err != nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update instructions: "+err.Error(), reqID)
		return
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"message":      "Instructions updated successfully",
		"version":      ag.Version,
		"instructions": ag.Instructions,
	}, reqID)
}

func (m *CodingWebModule) handleProjects(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()

	if r.Method == http.MethodGet {
		projects, err := m.projSvc.ListByDomain(ctx, domainProject.DomainCoding)
		if err != nil {
			host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list coding projects", reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, projects, reqID)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name      string `json:"name"`
			Path      string `json:"path"`
			Workspace string `json:"workspace"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			host.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", reqID)
			return
		}
		if req.Name == "" {
			host.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "name is required", reqID)
			return
		}
		targetPath := req.Path
		if targetPath == "" {
			targetPath = req.Workspace
		}
		if targetPath == "" {
			var err error
			targetPath, err = os.Getwd()
			if err != nil {
				host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
				return
			}
		}
		absPath, err := filepath.Abs(config.ExpandPath(targetPath))
		if err != nil {
			host.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid workspace path: "+err.Error(), reqID)
			return
		}

		p, err := m.projSvc.Register(ctx, req.Name, "Coding Repository", absPath, domainProject.DomainCoding)
		if err != nil {
			host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to register project: "+err.Error(), reqID)
			return
		}
		if m.codingObs != nil {
			m.codingObs.RegisterProjectDir(string(p.ID), p.Workspace)
		}
		host.WriteJSON(w, http.StatusCreated, p, reqID)
		return
	}

	host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not supported", reqID)
}

func (m *CodingWebModule) handleScan(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST allowed", reqID)
		return
	}

	var req struct {
		ProjectID string `json:"project_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ctx := r.Context()
	targetID := req.ProjectID
	if targetID == "" {
		projects, err := m.projSvc.ListByDomain(ctx, domainProject.DomainCoding)
		if err == nil && len(projects) > 0 {
			targetID = string(projects[0].ID)
		} else {
			targetID = "default"
		}
	}

	if m.codingObs == nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "CodingObserver not wired", reqID)
		return
	}

	observations, err := m.codingObs.Observe(ctx, targetID)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Observation scan failed: "+err.Error(), reqID)
		return
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"project_id":   targetID,
		"observations": observations,
	}, reqID)
}

func (m *CodingWebModule) handlePrompt(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST allowed", reqID)
		return
	}

	var req struct {
		Prompt    string `json:"prompt"`
		ProjectID string `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Prompt == "" {
		host.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "prompt is required", reqID)
		return
	}

	ctx := r.Context()
	targetProj := req.ProjectID
	if targetProj == "" {
		projects, err := m.projSvc.ListByDomain(ctx, domainProject.DomainCoding)
		if err == nil && len(projects) > 0 {
			targetProj = string(projects[0].ID)
		} else {
			targetProj = "default"
		}
	}

	title := req.Prompt
	if len(title) > 50 {
		title = title[:50] + "..."
	}

	t, err := m.taskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title:       fmt.Sprintf("Coding: %s", title),
		Description: req.Prompt,
		AgentID:     "coding-agent",
		ProjectID:   targetProj,
	})
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create task: "+err.Error(), reqID)
		return
	}

	// Run via Scheduler
	if err := m.sched.RunTask(ctx, t.ID); err != nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Task run error: "+err.Error(), reqID)
		return
	}

	execs, _ := m.execSvc.ListByTask(ctx, t.ID)
	var latestExec any
	if len(execs) > 0 {
		latestExec = execs[0]
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"task_id":   t.ID,
		"status":    "COMPLETED",
		"execution": latestExec,
	}, reqID)
}
