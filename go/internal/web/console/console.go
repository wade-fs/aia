package console

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/continuity"
	domainGoal "github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/investment"
	domainMemory "github.com/wade/aia/internal/domain/memory"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/provider"
	agentService "github.com/wade/aia/internal/service/agent"
	approvalService "github.com/wade/aia/internal/service/approval"
	goalService "github.com/wade/aia/internal/service/goal"
	govService "github.com/wade/aia/internal/service/governance"
	invService "github.com/wade/aia/internal/service/investment"
	memoryService "github.com/wade/aia/internal/service/memory"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/web/host"
	"github.com/wade/aia/internal/worker"
)

type ConsoleServices struct {
	TaskSvc        *taskService.Service
	AgentSvc       *agentService.Service
	MemorySvc      *memoryService.Service
	AuditRepo      *repository.AuditRepository
	InvSvc         *invService.Service
	DB             *sqlite.DB
	Provider       provider.Provider
	Host           *host.Host
	WorkerMgr      *worker.Manager
	ApprovalSvc    *approvalService.Service
	GovernanceSvc  *govService.Service
	GoalSvc        *goalService.Service
	AutonomyEngine *autonomy.Engine
	AutonomyRepo   *repository.AutonomyRepository
	ContinuityRepo *repository.ContinuityRepository
}

func RegisterConsoleRoutes(mux *http.ServeMux, s *ConsoleServices) {
	// 1. Auth routes
	mux.HandleFunc("/api/v1/auth/login", s.handleAuthLogin)
	mux.HandleFunc("/api/v1/auth/logout", s.handleAuthLogout)
	mux.HandleFunc("/api/v1/auth/session", s.handleAuthSession)

	// 2. System Status & Activity Monitor
	mux.HandleFunc("/api/v1/system/status", s.handleSystemStatus)
	mux.HandleFunc("/api/v1/system/health", s.handleSystemHealth)
	mux.HandleFunc("/api/v1/system/activity", s.handleActivityMonitor)

	// 2.1 Governance & Emergency Stop
	mux.HandleFunc("/api/v1/system/emergency-stop", s.handleEmergencyStop)
	mux.HandleFunc("/api/v1/system/resume", s.handleResumeDispatch)

	// 3. Tasks
	mux.HandleFunc("/api/v1/tasks", s.handleTasks)
	mux.HandleFunc("/api/v1/tasks/", s.handleTaskSubroutes)

	// 4. Agents
	mux.HandleFunc("/api/v1/agents", s.handleAgents)
	mux.HandleFunc("/api/v1/agents/", s.handleAgentDetail)

	// 5. Memory
	mux.HandleFunc("/api/v1/memory", s.handleMemory)
	mux.HandleFunc("/api/v1/memory/", s.handleMemorySubroutes)

	// 6. Pending Approvals (Unified: Formal Approvals + Tasks + Memory Candidates + Trade Proposals)
	mux.HandleFunc("/api/v1/pending", s.handlePendingList)
	mux.HandleFunc("/api/v1/pending/", s.handlePendingAction)

	// 6.1 Formal Approvals API
	mux.HandleFunc("/api/v1/approvals", s.handleApprovals)
	mux.HandleFunc("/api/v1/approvals/", s.handleApprovalSubroutes)

	// 7. Audit
	mux.HandleFunc("/api/v1/audit", s.handleAudit)

	// 8. Autonomy Cycles & Long-term Goals (Phase 3)
	mux.HandleFunc("/api/v1/cycles", s.handleCycles)
	mux.HandleFunc("/api/v1/cycles/", s.handleCycleSubroutes)
	mux.HandleFunc("/api/v1/goals", s.handleGoals)
	mux.HandleFunc("/api/v1/goals/", s.handleGoalSubroutes)
}

func getReqID(r *http.Request) string {
	return r.Header.Get("X-Request-ID")
}

func (s *ConsoleServices) getRequestActor(r *http.Request) governance.Actor {
	actor := governance.Actor{Type: "HUMAN", ID: "web_operator"}
	if cookie, err := r.Cookie("aia_session"); err == nil && cookie.Value != "" && s.Host != nil && s.Host.AuthMgr != nil {
		if sess, ok := s.Host.AuthMgr.ValidateSession(cookie.Value); ok && sess != nil && sess.ActorID != "" {
			actor.ID = sess.ActorID
		}
	}
	return actor
}

// -------------------------------------------------------------
// Auth Handlers
// -------------------------------------------------------------

func (s *ConsoleServices) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		host.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "A valid one-time login token is required", reqID)
		return
	}

	sess, err := s.Host.AuthMgr.RedeemOneTimeToken(strings.TrimSpace(req.Token))
	if err != nil {
		host.WriteError(w, http.StatusUnauthorized, "LOGIN_FAILED", err.Error(), reqID)
		return
	}

	s.Host.AuthMgr.SetSessionCookie(w, sess.ID)
	host.WriteJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"session_id":    sess.ID,
		"actor_id":      sess.ActorID,
	}, reqID)
}

func (s *ConsoleServices) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}

	if cookie, err := r.Cookie("aia_session"); err == nil && cookie.Value != "" {
		s.Host.AuthMgr.InvalidateSession(cookie.Value)
	}
	s.Host.AuthMgr.ClearSessionCookie(w)
	host.WriteJSON(w, http.StatusOK, map[string]bool{"logged_out": true}, reqID)
}

func (s *ConsoleServices) handleAuthSession(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	cookie, err := r.Cookie("aia_session")
	if err != nil || cookie.Value == "" {
		host.WriteError(w, http.StatusUnauthorized, "NOT_LOGGED_IN", "No active session", reqID)
		return
	}
	sess, ok := s.Host.AuthMgr.ValidateSession(cookie.Value)
	if !ok {
		host.WriteError(w, http.StatusUnauthorized, "SESSION_EXPIRED", "Session expired", reqID)
		return
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"session_id":    sess.ID,
		"actor_id":      sess.ActorID,
		"nav_items":     s.Host.Modules.AllNavItems(),
	}, reqID)
}

// -------------------------------------------------------------
// System Handlers
// -------------------------------------------------------------

func (s *ConsoleServices) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	host.WriteJSON(w, http.StatusOK, map[string]string{"status": "UP"}, getReqID(r))
}

func (s *ConsoleServices) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()

	provName := "unknown"
	provStatus := "OK"
	if s.Provider != nil {
		provName = s.Provider.Name()
		if err := s.Provider.HealthCheck(ctx); err != nil {
			provStatus = "DEGRADED: " + err.Error()
		}
	}

	dbStatus := "OK"
	if err := s.DB.Ping(ctx); err != nil {
		dbStatus = "ERROR: " + err.Error()
	}

	schemaVer, _ := s.DB.SchemaVersion(ctx)

	var workerStatus any
	var activitySnap any
	if s.WorkerMgr != nil {
		workerStatus = s.WorkerMgr.Status()
		activitySnap, _ = s.WorkerMgr.GetActivityMonitor(ctx)
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"runtime":        "AIA Personal Agent v1.0",
		"database":       dbStatus,
		"schema_version": schemaVer,
		"workers":        workerStatus,
		"activity":       activitySnap,
		"provider": map[string]string{
			"name":   provName,
			"status": provStatus,
		},
	}, reqID)
}

func (s *ConsoleServices) handleActivityMonitor(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()
	if s.WorkerMgr == nil {
		host.WriteError(w, http.StatusInternalServerError, "WORKER_UNAVAILABLE", "worker manager not initialized", reqID)
		return
	}
	snap, err := s.WorkerMgr.GetActivityMonitor(ctx)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "MONITOR_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, snap, reqID)
}

// -------------------------------------------------------------
// Tasks Handlers (READY instead of QUEUED)
// -------------------------------------------------------------

func (s *ConsoleServices) handleTasks(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		statusFilter := task.TaskStatus(r.URL.Query().Get("status"))
		limit := 50
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
			limit = l
		}
		tasks, err := s.TaskSvc.List(ctx, repository.TaskFilter{Status: statusFilter, Limit: limit})
		if err != nil {
			host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, tasks, reqID)

	case http.MethodPost:
		var req struct {
			Title       string           `json:"title"`
			Description string           `json:"description"`
			AgentID     domain.ID        `json:"agent_id"`
			Priority    int              `json:"priority"`
			Risk        domain.RiskLevel `json:"risk"`
			Acceptance  []string         `json:"acceptance"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Title) == "" {
			host.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "Task title is required", reqID)
			return
		}

		tk, err := s.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
			Title:       req.Title,
			Description: req.Description,
			AgentID:     req.AgentID,
			Priority:    req.Priority,
			Risk:        req.Risk,
			Acceptance:  req.Acceptance,
		})
		if err != nil {
			host.WriteError(w, http.StatusInternalServerError, "TASK_CREATE_FAILED", err.Error(), reqID)
			return
		}

		// Returns READY (never QUEUED)
		host.WriteJSON(w, http.StatusCreated, tk, reqID)

	default:
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
	}
}

func (s *ConsoleServices) handleTaskSubroutes(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()

	// Parse /api/v1/tasks/{id} or /api/v1/tasks/{id}/{action}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		host.WriteError(w, http.StatusBadRequest, "INVALID_PATH", "Task ID required", reqID)
		return
	}
	taskID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
			return
		}
		tk, err := s.TaskSvc.Get(ctx, taskID)
		if err != nil {
			host.WriteError(w, http.StatusNotFound, "TASK_NOT_FOUND", "Task not found", reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, tk, reqID)
		return
	}

	action := parts[1]
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}

	switch action {
	case "cancel":
		if err := s.TaskSvc.Cancel(ctx, taskID); err != nil {
			host.WriteError(w, http.StatusBadRequest, "CANCEL_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "CANCELLED"}, reqID)

	case "resume":
		if err := s.TaskSvc.Resume(ctx, taskID); err != nil {
			host.WriteError(w, http.StatusBadRequest, "RESUME_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "READY"}, reqID)

	case "approve":
		if err := s.TaskSvc.UpdateStatus(ctx, taskID, task.StatusReady); err != nil {
			host.WriteError(w, http.StatusBadRequest, "APPROVE_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "READY"}, reqID)

	case "reject":
		if err := s.TaskSvc.UpdateStatus(ctx, taskID, task.StatusCancelled); err != nil {
			host.WriteError(w, http.StatusBadRequest, "REJECT_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "CANCELLED"}, reqID)

	default:
		host.WriteError(w, http.StatusNotFound, "ACTION_NOT_FOUND", "Unknown task action", reqID)
	}
}

// -------------------------------------------------------------
// Agents Handlers
// -------------------------------------------------------------

func (s *ConsoleServices) handleAgents(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	if r.Method != http.MethodGet {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}
	agents, err := s.AgentSvc.List(r.Context())
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, agents, reqID)
}

func (s *ConsoleServices) handleAgentDetail(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	if r.Method != http.MethodGet {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}
	agentID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/agents/"), "/")
	// If sub-path belongs to agent module, let host sub-mux handle it
	if strings.Contains(agentID, "/") {
		return
	}
	ag, err := s.AgentSvc.Get(r.Context(), agentID)
	if err != nil {
		host.WriteError(w, http.StatusNotFound, "AGENT_NOT_FOUND", "Agent not found", reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, ag, reqID)
}

// -------------------------------------------------------------
// Memory Handlers
// -------------------------------------------------------------

func (s *ConsoleServices) handleMemory(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	if r.Method != http.MethodGet {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	memories, err := s.MemorySvc.Search(r.Context(), repository.MemoryQuery{Limit: limit})
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, memories, reqID)
}

func (s *ConsoleServices) handleMemorySubroutes(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()

	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/memory/"), "/"), "/")
	if len(parts) < 2 {
		host.WriteError(w, http.StatusBadRequest, "INVALID_PATH", "Action required", reqID)
		return
	}
	memID, action := parts[0], parts[1]
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}

	switch action {
	case "approve":
		if err := s.MemorySvc.ApproveCandidate(ctx, memID); err != nil {
			host.WriteError(w, http.StatusBadRequest, "APPROVE_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "ACTIVE"}, reqID)

	case "reject":
		if err := s.MemorySvc.RejectCandidate(ctx, memID); err != nil {
			host.WriteError(w, http.StatusBadRequest, "REJECT_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "REJECTED"}, reqID)

	default:
		host.WriteError(w, http.StatusNotFound, "ACTION_NOT_FOUND", "Unknown action", reqID)
	}
}

// -------------------------------------------------------------
// Pending Approvals (Unified: Tasks + Memory Candidates + Trade Proposals)
// -------------------------------------------------------------

func (s *ConsoleServices) handlePendingList(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()
	if r.Method != http.MethodGet {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}

	// 1. Tasks in WAITING_APPROVAL
	tasks, err := s.TaskSvc.List(ctx, repository.TaskFilter{Status: task.StatusWaitingApproval, Limit: 100})
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}

	// 2. Memory Candidates
	candidates, err := s.MemorySvc.ListCandidates(ctx, domainMemory.ReviewPending)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}

	// 3. Trade Proposals in PENDING
	var tradeProposals []investment.TradeProposal
	if s.InvSvc != nil {
		tradeProposals, _ = s.InvSvc.ListProposals(ctx, investment.ProposalPending, 100)
	}

	// 4. Formal Approvals in PENDING (AIA-17 / Phase 2)
	var formalApprovals []*governance.Approval
	if s.ApprovalSvc != nil {
		formalApprovals, _ = s.ApprovalSvc.ListPending(ctx)
	}

	totalPending := len(tasks) + len(candidates) + len(tradeProposals) + len(formalApprovals)

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"total_count":      totalPending,
		"tasks":            tasks,
		"memories":         candidates,
		"trade_proposals":  tradeProposals,
		"formal_approvals": formalApprovals,
	}, reqID)
}

func (s *ConsoleServices) handlePendingAction(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}

	// Format: /api/v1/pending/{type}/{id}/{approve|reject}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/pending/"), "/"), "/")
	if len(parts) < 3 {
		host.WriteError(w, http.StatusBadRequest, "INVALID_PATH", "Format: /api/v1/pending/{type}/{id}/{action}", reqID)
		return
	}
	itemType, itemID, action := parts[0], parts[1], parts[2]

	switch itemType {
	case "approvals":
		if s.ApprovalSvc == nil {
			host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Approval service not enabled", reqID)
			return
		}
		var req struct {
			Comment string `json:"comment"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		actor := s.getRequestActor(r)

		if action == "approve" {
			if err := s.ApprovalSvc.Approve(ctx, itemID, actor.ID, req.Comment, actor.Type); err != nil {
				host.WriteError(w, http.StatusBadRequest, "APPROVE_FAILED", err.Error(), reqID)
				return
			}
		} else if action == "reject" {
			if err := s.ApprovalSvc.Reject(ctx, itemID, actor.ID, req.Comment, actor.Type); err != nil {
				host.WriteError(w, http.StatusBadRequest, "REJECT_FAILED", err.Error(), reqID)
				return
			}
		} else {
			host.WriteError(w, http.StatusBadRequest, "INVALID_ACTION", "Action must be approve or reject", reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "UPDATED"}, reqID)

	case "tasks":
		if action == "approve" {
			if err := s.TaskSvc.UpdateStatus(ctx, itemID, task.StatusReady); err != nil {
				host.WriteError(w, http.StatusBadRequest, "APPROVE_FAILED", err.Error(), reqID)
				return
			}
		} else if action == "reject" {
			if err := s.TaskSvc.UpdateStatus(ctx, itemID, task.StatusCancelled); err != nil {
				host.WriteError(w, http.StatusBadRequest, "REJECT_FAILED", err.Error(), reqID)
				return
			}
		} else {
			host.WriteError(w, http.StatusBadRequest, "INVALID_ACTION", "Action must be approve or reject", reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "UPDATED"}, reqID)

	case "memory":
		if action == "approve" {
			if err := s.MemorySvc.ApproveCandidate(ctx, itemID); err != nil {
				host.WriteError(w, http.StatusBadRequest, "APPROVE_FAILED", err.Error(), reqID)
				return
			}
		} else if action == "reject" {
			if err := s.MemorySvc.RejectCandidate(ctx, itemID); err != nil {
				host.WriteError(w, http.StatusBadRequest, "REJECT_FAILED", err.Error(), reqID)
				return
			}
		} else {
			host.WriteError(w, http.StatusBadRequest, "INVALID_ACTION", "Action must be approve or reject", reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "UPDATED"}, reqID)

	case "proposals":
		if s.InvSvc == nil {
			host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Investment service not enabled", reqID)
			return
		}
		if action == "approve" {
			if err := s.InvSvc.ApproveProposal(ctx, itemID); err != nil {
				host.WriteError(w, http.StatusBadRequest, "APPROVE_FAILED", err.Error(), reqID)
				return
			}
		} else if action == "reject" {
			if err := s.InvSvc.RejectProposal(ctx, itemID, "Rejected via pending console"); err != nil {
				host.WriteError(w, http.StatusBadRequest, "REJECT_FAILED", err.Error(), reqID)
				return
			}
		} else {
			host.WriteError(w, http.StatusBadRequest, "INVALID_ACTION", "Action must be approve or reject", reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "UPDATED"}, reqID)

	default:
		host.WriteError(w, http.StatusBadRequest, "INVALID_TYPE", "Pending type must be tasks, memory, or proposals", reqID)
	}
}

// -------------------------------------------------------------
// Audit Handlers
// -------------------------------------------------------------

func (s *ConsoleServices) handleAudit(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	if r.Method != http.MethodGet {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}

	filter := repository.AuditFilter{
		Limit:        50,
		Source:       r.URL.Query().Get("source"),
		ResourceType: r.URL.Query().Get("resource_type"),
		ResourceID:   r.URL.Query().Get("resource_id"),
		RequestID:    r.URL.Query().Get("request_id"),
	}
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		filter.Limit = l
	}

	logs, err := s.AuditRepo.Query(r.Context(), filter)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, logs, reqID)
}

// -------------------------------------------------------------
// Emergency Stop & Governance Handlers (AIA-17 / Phase 2)
// -------------------------------------------------------------

func (s *ConsoleServices) handleEmergencyStop(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method must be POST", reqID)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Reason) == "" {
		req.Reason = "Emergency Stop triggered from Web Console"
	}

	actor := s.getRequestActor(r)

	if s.WorkerMgr != nil {
		if err := s.WorkerMgr.TriggerEmergencyStop(ctx, actor, req.Reason); err != nil {
			host.WriteError(w, http.StatusInternalServerError, "EMERGENCY_STOP_FAILED", err.Error(), reqID)
			return
		}
	} else if s.GovernanceSvc != nil {
		if err := s.GovernanceSvc.TriggerEmergencyStop(ctx, actor, req.Reason); err != nil {
			host.WriteError(w, http.StatusInternalServerError, "EMERGENCY_STOP_FAILED", err.Error(), reqID)
			return
		}
	} else {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Governance runtime not available", reqID)
		return
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"status":    "EMERGENCY_STOP",
		"reason":    req.Reason,
		"actor":     actor,
		"timestamp": time.Now().UTC(),
	}, reqID)
}

func (s *ConsoleServices) handleResumeDispatch(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method must be POST", reqID)
		return
	}

	actor := s.getRequestActor(r)

	var req struct {
		Force bool `json:"force"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if s.WorkerMgr != nil {
		if err := s.WorkerMgr.ResumeDispatch(ctx, actor, req.Force); err != nil {
			host.WriteError(w, http.StatusBadRequest, "RESUME_FAILED", err.Error(), reqID)
			return
		}
	} else if s.GovernanceSvc != nil {
		if err := s.GovernanceSvc.ResumeDispatch(ctx, actor, req.Force); err != nil {
			host.WriteError(w, http.StatusBadRequest, "RESUME_FAILED", err.Error(), reqID)
			return
		}
	} else {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Governance runtime not available", reqID)
		return
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"status":    "RUNNING",
		"actor":     actor,
		"timestamp": time.Now().UTC(),
	}, reqID)
}

// -------------------------------------------------------------
// Formal Approvals Subroutes (AIA-17 §7)
// -------------------------------------------------------------

func (s *ConsoleServices) handleApprovals(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()

	if s.ApprovalSvc == nil {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Approval service not available", reqID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		limit := 100
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
			limit = l
		}

		var approvals []*governance.Approval
		var err error
		if status == "" || strings.EqualFold(status, "PENDING") {
			approvals, err = s.ApprovalSvc.ListPending(ctx)
		} else {
			approvals, err = s.ApprovalSvc.List(ctx, repository.ApprovalFilter{
				Status: governance.ApprovalStatus(strings.ToUpper(status)),
				Limit:  limit,
			})
		}

		if err != nil {
			host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, approvals, reqID)

	default:
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
	}
}

func (s *ConsoleServices) handleApprovalSubroutes(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()

	if s.ApprovalSvc == nil {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Approval service not available", reqID)
		return
	}

	// Format: /api/v1/approvals/{id} or /api/v1/approvals/{id}/{approve|reject}
	subpath := strings.TrimPrefix(r.URL.Path, "/api/v1/approvals/")
	parts := strings.Split(strings.Trim(subpath, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		host.WriteError(w, http.StatusBadRequest, "INVALID_ID", "Approval ID required", reqID)
		return
	}
	apprID := parts[0]

	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			appr, err := s.ApprovalSvc.Get(ctx, apprID)
			if err != nil {
				host.WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), reqID)
				return
			}
			host.WriteJSON(w, http.StatusOK, appr, reqID)
			return
		}
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}

	action := parts[1]
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method must be POST", reqID)
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	actor := s.getRequestActor(r)

	switch strings.ToLower(action) {
	case "approve":
		if err := s.ApprovalSvc.Approve(ctx, apprID, actor.ID, req.Comment, actor.Type); err != nil {
			host.WriteError(w, http.StatusBadRequest, "APPROVE_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "APPROVED"}, reqID)

	case "reject":
		if err := s.ApprovalSvc.Reject(ctx, apprID, actor.ID, req.Comment, actor.Type); err != nil {
			host.WriteError(w, http.StatusBadRequest, "REJECT_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "REJECTED"}, reqID)

	default:
		host.WriteError(w, http.StatusBadRequest, "INVALID_ACTION", "Action must be approve or reject", reqID)
	}
}

// -------------------------------------------------------------
// Autonomy Cycles & Long-Term Goals API (Phase 3)
// -------------------------------------------------------------

func (s *ConsoleServices) handleCycles(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		if s.ContinuityRepo == nil {
			host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Continuity repository not available", reqID)
			return
		}
		limit := 25
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
			limit = l
		}
		cycles, err := s.ContinuityRepo.ListCycles(ctx, limit)
		if err != nil {
			host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, cycles, reqID)

	case http.MethodPost:
		if s.AutonomyEngine == nil {
			host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Autonomy engine not available", reqID)
			return
		}
		var req struct {
			ProjectID string `json:"project_id"`
			Summary   string `json:"summary"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.ProjectID == "" {
			req.ProjectID = "default"
		}
		if req.Summary == "" {
			req.Summary = "Manual autonomy cycle via Web Console"
		}
		outcome, err := s.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   req.ProjectID,
			TriggerType: continuity.TriggerManual,
			TriggerRef:  "web-console",
			Summary:     req.Summary,
		})
		if err != nil {
			host.WriteError(w, http.StatusBadRequest, "CYCLE_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, outcome, reqID)

	default:
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
	}
}

func (s *ConsoleServices) handleCycleSubroutes(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()
	if r.Method != http.MethodGet {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}
	if s.ContinuityRepo == nil {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Continuity repository not available", reqID)
		return
	}

	cycleID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/cycles/"), "/")
	if cycleID == "" {
		host.WriteError(w, http.StatusBadRequest, "INVALID_ID", "Cycle ID required", reqID)
		return
	}
	cyc, err := s.ContinuityRepo.GetCycle(ctx, cycleID)
	if err != nil {
		host.WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), reqID)
		return
	}

	resp := map[string]any{"cycle": cyc}
	if s.AutonomyRepo != nil {
		if inits, iErr := s.AutonomyRepo.ListInitiativesByCycle(ctx, cycleID); iErr == nil {
			resp["initiatives"] = inits
		}
		if delib, dErr := s.AutonomyRepo.GetDeliberationByCycle(ctx, cycleID); dErr == nil && delib != nil {
			resp["deliberation"] = delib
		}
	}
	host.WriteJSON(w, http.StatusOK, resp, reqID)
}

func (s *ConsoleServices) handleGoals(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()
	if s.GoalSvc == nil {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Goal service not available", reqID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := domainGoal.GoalStatus(strings.ToUpper(r.URL.Query().Get("status")))
		projectID := r.URL.Query().Get("project_id")
		var goals []*domainGoal.Goal
		var err error
		if projectID != "" {
			goals, err = s.GoalSvc.ListByProject(ctx, projectID)
		} else {
			goals, err = s.GoalSvc.ListAll(ctx, status, 50)
		}
		if err != nil {
			host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, goals, reqID)

	case http.MethodPost:
		var req struct {
			ProjectID         string `json:"project_id"`
			ParentGoalID      string `json:"parent_goal_id"`
			Name              string `json:"name"`
			Description       string `json:"description"`
			Level             string `json:"level"`
			TargetMetricJSON  string `json:"target_metric_json"`
			CurrentMetricJSON string `json:"current_metric_json"`
			Priority          int    `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
			host.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "Goal name is required", reqID)
			return
		}
		var parentPtr *string
		if req.ParentGoalID != "" {
			parentPtr = &req.ParentGoalID
		}
		created, err := s.GoalSvc.Create(ctx, goalService.CreateGoalInput{
			ParentGoalID:      parentPtr,
			ProjectID:         req.ProjectID,
			Name:              req.Name,
			Description:       req.Description,
			Level:             domainGoal.GoalLevel(strings.ToUpper(req.Level)),
			TargetMetricJSON:  req.TargetMetricJSON,
			CurrentMetricJSON: req.CurrentMetricJSON,
			Priority:          req.Priority,
		})
		if err != nil {
			host.WriteError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusCreated, created, reqID)

	default:
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
	}
}

func (s *ConsoleServices) handleGoalSubroutes(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	ctx := r.Context()
	if s.GoalSvc == nil {
		host.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Goal service not available", reqID)
		return
	}

	goalID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/goals/"), "/")
	if goalID == "" {
		host.WriteError(w, http.StatusBadRequest, "INVALID_ID", "Goal ID required", reqID)
		return
	}
	if r.Method != http.MethodGet {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", reqID)
		return
	}

	g, err := s.GoalSvc.Get(ctx, goalID)
	if err != nil {
		host.WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), reqID)
		return
	}
	links, _ := s.GoalSvc.ListDecomposedTasks(ctx, goalID)
	revs, _ := s.GoalSvc.ListRevisions(ctx, goalID)
	host.WriteJSON(w, http.StatusOK, map[string]any{
		"goal":             g,
		"gap_assessment":   goalService.EvaluateGoalGap(g),
		"decomposed_tasks": links,
		"revisions":        revs,
	}, reqID)
}
