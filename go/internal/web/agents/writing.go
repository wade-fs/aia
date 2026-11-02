package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/config"
	domainProject "github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/orchestrator"
	agentService "github.com/wade/aia/internal/service/agent"
	execService "github.com/wade/aia/internal/service/execution"
	projectService "github.com/wade/aia/internal/service/project"
	taskService "github.com/wade/aia/internal/service/task"
	writingService "github.com/wade/aia/internal/service/writing"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/web/host"
	"gopkg.in/yaml.v3"
)

type WritingWebModule struct {
	agentSvc     *agentService.Service
	projSvc      *projectService.Service
	taskSvc      *taskService.Service
	sched        *orchestrator.Scheduler
	execSvc      *execService.Service
	writingObs   *autonomy.ManuscriptConsistencyObserver
	workflowSvc  *writingService.WorkflowService
	workflowRepo *repository.WritingWorkflowRepository
}

func (m *WritingWebModule) SetWorkflowDeps(svc *writingService.WorkflowService, repo *repository.WritingWorkflowRepository) {
	m.workflowSvc = svc
	m.workflowRepo = repo
}

func NewWritingWebModuleWithDeps(
	agentSvc *agentService.Service,
	projSvc *projectService.Service,
	taskSvc *taskService.Service,
	sched *orchestrator.Scheduler,
	execSvc *execService.Service,
	writingObs *autonomy.ManuscriptConsistencyObserver,
) *WritingWebModule {
	return &WritingWebModule{
		agentSvc:   agentSvc,
		projSvc:    projSvc,
		taskSvc:    taskSvc,
		sched:      sched,
		execSvc:    execSvc,
		writingObs: writingObs,
	}
}

func (m *WritingWebModule) AgentID() string {
	return "writing"
}

func (m *WritingWebModule) NavItems() []host.NavItem {
	return []host.NavItem{
		{
			ID:       "writing",
			Label:    "Writing Agent",
			Path:     "#/writing",
			Icon:     "edit-3",
			Order:    30,
			Required: "writing.read",
		},
	}
}

func (m *WritingWebModule) RegisterRoutes(mux *http.ServeMux) error {
	mux.HandleFunc("/info", m.handleInfo)
	mux.HandleFunc("/instructions", m.handleInstructions)
	mux.HandleFunc("/projects", m.handleProjects)
	mux.HandleFunc("/manifest", m.handleManifest)
	mux.HandleFunc("/scan", m.handleScan)
	mux.HandleFunc("/prompt", m.handlePrompt)
	mux.HandleFunc("/tasks", m.handleTasks)
	mux.HandleFunc("/brief", m.handleBrief)
	mux.HandleFunc("/clarifications", m.handleClarifications)
	mux.HandleFunc("/clarifications/answer", m.handleAnswerClarification)
	mux.HandleFunc("/initiate", m.handleInitiate)
	return nil
}

func (m *WritingWebModule) handleTasks(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()
	limit := 10
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	tasks, err := m.taskSvc.List(ctx, repository.TaskFilter{
		AgentID:     "writing-agent",
		OrderByDesc: true,
		Limit:       limit,
	})
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}

	items := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		item := map[string]any{
			"id":           t.ID,
			"title":        t.Title,
			"description":  t.Description,
			"agent_id":     t.AgentID,
			"project_id":   t.ProjectID,
			"status":       t.Status,
			"created_at":   t.CreatedAt,
			"started_at":   t.StartedAt,
			"completed_at": t.CompletedAt,
		}
		if m.execSvc != nil {
			execs, _ := m.execSvc.ListByTask(ctx, t.ID)
			if len(execs) > 0 {
				item["conversation_id"] = execs[0].ConversationID
				item["execution_id"] = execs[0].ID
				item["execution_status"] = execs[0].Status
			}
		}
		items = append(items, item)
	}
	host.WriteJSON(w, http.StatusOK, items, reqID)
}

func (m *WritingWebModule) handleInfo(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ag, err := m.agentSvc.Get(r.Context(), "writing-agent")
	if err != nil {
		ag, err = m.agentSvc.Get(r.Context(), "writing")
	}
	if err != nil {
		host.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Writing agent info unavailable", reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, ag, reqID)
}

func (m *WritingWebModule) handleInstructions(w http.ResponseWriter, r *http.Request) {
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
	ag, err := m.agentSvc.Get(ctx, "writing-agent")
	if err != nil {
		host.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Writing agent not found", reqID)
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

func (m *WritingWebModule) handleProjects(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()

	if r.Method == http.MethodGet {
		projects, err := m.projSvc.ListByDomain(ctx, domainProject.DomainWriting)
		if err != nil {
			host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list writing projects", reqID)
			return
		}

		type projectResponse struct {
			domainProject.Project
			HasManifest bool `json:"has_manifest"`
		}
		var list []projectResponse
		for _, p := range projects {
			hasManifest := false
			if _, err := os.Stat(filepath.Join(p.Workspace, "book.yaml")); err == nil {
				hasManifest = true
			}
			list = append(list, projectResponse{
				Project:     p,
				HasManifest: hasManifest,
			})
		}

		host.WriteJSON(w, http.StatusOK, list, reqID)
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
			host.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid novel path: "+err.Error(), reqID)
			return
		}

		p, err := m.projSvc.Register(ctx, req.Name, "Novel Project", absPath, domainProject.DomainWriting)
		if err != nil {
			host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to register project: "+err.Error(), reqID)
			return
		}
		if m.writingObs != nil {
			m.writingObs.RegisterProjectPath(string(p.ID), p.Workspace)
		}
		host.WriteJSON(w, http.StatusCreated, p, reqID)
		return
	}

	if r.Method == http.MethodDelete {
		projectID := r.URL.Query().Get("id")
		if projectID == "" {
			projectID = r.URL.Query().Get("project_id")
		}
		if projectID == "" {
			var body struct {
				ID        string `json:"id"`
				ProjectID string `json:"project_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				projectID = body.ID
				if projectID == "" {
					projectID = body.ProjectID
				}
			}
		}
		if projectID == "" {
			host.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "project id is required", reqID)
			return
		}

		if err := m.projSvc.Delete(ctx, projectID); err != nil {
			host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete writing project: "+err.Error(), reqID)
			return
		}
		if m.writingObs != nil {
			m.writingObs.UnregisterProjectPath(projectID)
		}
		host.WriteJSON(w, http.StatusOK, map[string]any{
			"message": "Writing project unregistered successfully",
			"id":      projectID,
		}, reqID)
		return
	}

	host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not supported", reqID)
}

func (m *WritingWebModule) handleManifest(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()

	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		projects, err := m.projSvc.ListByDomain(ctx, domainProject.DomainWriting)
		if err == nil && len(projects) > 0 {
			projectID = string(projects[0].ID)
		} else {
			projectID = "yuan"
		}
	}

	p, err := m.projSvc.Get(ctx, projectID)
	if err != nil {
		host.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Project not found: "+projectID, reqID)
		return
	}

	manifestFile := filepath.Join(p.Workspace, "book.yaml")
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		host.WriteError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("book.yaml not found in %s", p.Workspace), reqID)
		return
	}

	var manifest writing.BookManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to parse book.yaml: "+err.Error(), reqID)
		return
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"project_id":    projectID,
		"manifest_file": manifestFile,
		"manifest":      manifest,
	}, reqID)
}

func (m *WritingWebModule) handleScan(w http.ResponseWriter, r *http.Request) {
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
		projects, err := m.projSvc.ListByDomain(ctx, domainProject.DomainWriting)
		if err == nil && len(projects) > 0 {
			targetID = string(projects[0].ID)
		} else {
			targetID = "yuan"
		}
	}

	if m.writingObs == nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "WritingObserver not wired", reqID)
		return
	}

	observations, err := m.writingObs.Observe(ctx, targetID)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Consistency scan failed: "+err.Error(), reqID)
		return
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"project_id":   targetID,
		"observations": observations,
	}, reqID)
}

func (m *WritingWebModule) handlePrompt(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST allowed", reqID)
		return
	}

	var req struct {
		Prompt    string `json:"prompt"`
		ProjectID string `json:"project_id"`
		Async     bool   `json:"async"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Prompt == "" {
		host.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "prompt is required", reqID)
		return
	}

	ctx := r.Context()
	targetProj := req.ProjectID
	if targetProj == "" {
		projects, err := m.projSvc.ListByDomain(ctx, domainProject.DomainWriting)
		if err == nil && len(projects) > 0 {
			targetProj = string(projects[0].ID)
		} else {
			targetProj = "yuan"
		}
	}

	title := req.Prompt
	if len(title) > 50 {
		title = title[:50] + "..."
	}

	t, err := m.taskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title:       fmt.Sprintf("Writing: %s", title),
		Description: req.Prompt,
		AgentID:     "writing-agent",
		ProjectID:   targetProj,
	})
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create task: "+err.Error(), reqID)
		return
	}

	if req.Async {
		go func() {
			_ = m.sched.RunTask(context.Background(), t.ID)
		}()
		host.WriteJSON(w, http.StatusOK, map[string]any{
			"task_id": t.ID,
			"status":  "RUNNING",
		}, reqID)
		return
	}

	if err := m.sched.RunTask(ctx, t.ID); err != nil {
		host.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Task run error: "+err.Error(), reqID)
		return
	}

	execs, _ := m.execSvc.ListByTask(ctx, t.ID)
	var latestExec any
	var convID string
	if len(execs) > 0 {
		latestExec = execs[0]
		convID = execs[0].ConversationID
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"task_id":         t.ID,
		"status":          "COMPLETED",
		"conversation_id": convID,
		"execution":       latestExec,
	}, reqID)
}

func (m *WritingWebModule) handleBrief(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		host.WriteError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id query parameter is required", reqID)
		return
	}
	if m.workflowRepo == nil {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Workflow repository not wired", reqID)
		return
	}
	brief, err := m.workflowRepo.GetBriefByProjectID(ctx, projectID)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	if brief == nil {
		host.WriteError(w, http.StatusNotFound, "NOT_FOUND", "creative brief not found", reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, brief, reqID)
}

func (m *WritingWebModule) handleClarifications(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		host.WriteError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id query parameter is required", reqID)
		return
	}
	if m.workflowRepo == nil {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Workflow repository not wired", reqID)
		return
	}
	reqs, err := m.workflowRepo.ListClarificationsByProjectID(ctx, projectID)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, reqs, reqID)
}

func (m *WritingWebModule) handleAnswerClarification(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST required", reqID)
		return
	}
	var req struct {
		ClarificationID string `json:"clarification_id"`
		Answer          string `json:"answer"`
		Workspace       string `json:"workspace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		host.WriteError(w, http.StatusBadRequest, "INVALID_BODY", err.Error(), reqID)
		return
	}
	if req.ClarificationID == "" || req.Answer == "" {
		host.WriteError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "clarification_id and answer are required", reqID)
		return
	}
	if m.workflowSvc == nil {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Workflow service not wired", reqID)
		return
	}
	resp, err := m.workflowSvc.AnswerClarification(ctx, req.ClarificationID, req.Answer, req.Workspace)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "WORKFLOW_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, resp, reqID)
}

func (m *WritingWebModule) handleInitiate(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST required", reqID)
		return
	}
	var req writingService.InitiateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		host.WriteError(w, http.StatusBadRequest, "INVALID_BODY", err.Error(), reqID)
		return
	}
	if req.RawIntent == "" {
		host.WriteError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "raw_intent is required", reqID)
		return
	}
	if m.workflowSvc == nil {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Workflow service not wired", reqID)
		return
	}
	resp, err := m.workflowSvc.InitiateProject(ctx, req)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "WORKFLOW_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, resp, reqID)
}
