package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wade/aia/internal/app"
	domainClarify "github.com/wade/aia/internal/domain/clarification"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/domain/memory"
	domainProject "github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/provider/mock"
	taskService "github.com/wade/aia/internal/service/task"
)

func setupTestWeb(t *testing.T) (*app.App, http.Handler, string) {
	t.Helper()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfgContent := "runtime:\n  data_dir: \"" + tmpDir + "\"\n  workspace_dir: \"" + tmpDir + "/workspaces\"\ndatabase:\n  driver: \"sqlite\"\n  path: \"" + tmpDir + "/test.db\"\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	mockProv := mock.NewMockProvider()
	appInst, err := app.NewApp(cfgPath, mockProv)
	if err != nil {
		t.Fatalf("failed to init app: %v", err)
	}

	webHost, token, err := appInst.CreateWebHost()
	if err != nil {
		t.Fatalf("failed to create web host: %v", err)
	}

	return appInst, webHost.Handler(), token
}

func TestWeb_StaticAssetsEmbedded(t *testing.T) {
	appInst, handler, _ := setupTestWeb(t)
	defer appInst.Close()

	// 1. Root serves index.html
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for index.html, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "AIA Console") {
		t.Errorf("expected index.html content to contain 'AIA Console'")
	}

	// 2. CSS is served
	reqCSS := httptest.NewRequest("GET", "/css/style.css", nil)
	recCSS := httptest.NewRecorder()
	handler.ServeHTTP(recCSS, reqCSS)
	if recCSS.Code != http.StatusOK {
		t.Errorf("expected 200 for css, got %d", recCSS.Code)
	}

	// 3. JS is served
	reqJS := httptest.NewRequest("GET", "/js/app.js", nil)
	recJS := httptest.NewRecorder()
	handler.ServeHTTP(recJS, reqJS)
	if recJS.Code != http.StatusOK {
		t.Errorf("expected 200 for js, got %d", recJS.Code)
	}
}

func TestWeb_OneTimeToken_AuthAndSession(t *testing.T) {
	appInst, handler, token := setupTestWeb(t)
	defer appInst.Close()

	// 1. Protected endpoint without token returns 401
	reqUnauth := httptest.NewRequest("GET", "/api/v1/system/status", nil)
	recUnauth := httptest.NewRecorder()
	handler.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", recUnauth.Code)
	}

	// 2. Redeem one-time token
	loginBody, _ := json.Marshal(map[string]string{"token": token})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	recLogin := httptest.NewRecorder()
	handler.ServeHTTP(recLogin, reqLogin)

	if recLogin.Code != http.StatusOK {
		t.Fatalf("login failed: %d body: %s", recLogin.Code, recLogin.Body.String())
	}

	cookies := recLogin.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "aia_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("expected aia_session cookie to be set")
	}
	if !sessionCookie.HttpOnly {
		t.Errorf("expected session cookie to be HttpOnly")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected session cookie to have SameSite=Strict")
	}

	// 3. One-time token cannot be reused (single-use validation)
	recReuse := httptest.NewRecorder()
	reqReuse := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	reqReuse.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recReuse, reqReuse)
	if recReuse.Code != http.StatusUnauthorized {
		t.Errorf("reused token must be rejected with 401, got %d", recReuse.Code)
	}

	// 4. Access protected endpoint with session cookie
	reqAuth := httptest.NewRequest("GET", "/api/v1/system/status", nil)
	reqAuth.AddCookie(sessionCookie)
	recAuth := httptest.NewRecorder()
	handler.ServeHTTP(recAuth, reqAuth)
	if recAuth.Code != http.StatusOK {
		t.Errorf("expected 200 with valid session, got %d: %s", recAuth.Code, recAuth.Body.String())
	}

	// 5. Logout invalidates session
	reqLogout := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	reqLogout.AddCookie(sessionCookie)
	recLogout := httptest.NewRecorder()
	handler.ServeHTTP(recLogout, reqLogout)
	if recLogout.Code != http.StatusOK {
		t.Errorf("logout failed: %d", recLogout.Code)
	}

	// 6. Accessing protected endpoint after logout fails
	reqPostLogout := httptest.NewRequest("GET", "/api/v1/system/status", nil)
	reqPostLogout.AddCookie(sessionCookie)
	recPostLogout := httptest.NewRecorder()
	handler.ServeHTTP(recPostLogout, reqPostLogout)
	if recPostLogout.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 after logout, got %d", recPostLogout.Code)
	}
}

func TestWeb_Tasks_StatusIsReadyNotQueued(t *testing.T) {
	appInst, handler, token := setupTestWeb(t)
	defer appInst.Close()

	// Login
	loginBody, _ := json.Marshal(map[string]string{"token": token})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	recLogin := httptest.NewRecorder()
	handler.ServeHTTP(recLogin, reqLogin)
	cookie := recLogin.Result().Cookies()[0]

	// Create Task via Web API
	taskReq := map[string]any{
		"title":       "Write tests for personal agent",
		"description": "Ensure READY status machine is strictly respected",
		"agent_id":    "coding-agent",
		"priority":    70,
		"risk":        "LOW",
	}
	body, _ := json.Marshal(taskReq)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool `json:"success"`
		Data    struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)

	// MUST be READY, NEVER QUEUED
	if res.Data.Status != "READY" {
		t.Errorf("expected initial task status to be READY (never QUEUED), got %s", res.Data.Status)
	}
}

func TestWeb_PendingReview_Integration(t *testing.T) {
	appInst, handler, token := setupTestWeb(t)
	defer appInst.Close()
	ctx := t.Context()

	// Login
	loginBody, _ := json.Marshal(map[string]string{"token": token})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	recLogin := httptest.NewRecorder()
	handler.ServeHTTP(recLogin, reqLogin)
	cookie := recLogin.Result().Cookies()[0]

	// 1. Seed a task in WAITING_APPROVAL
	tk, err := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
		Title: "High-risk DB migration",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	_ = appInst.TaskSvc.UpdateStatus(ctx, tk.ID, task.StatusWaitingApproval)

	// 2. Seed a memory candidate
	_ = appInst.MemorySvc.CreateCandidate(ctx, &memory.MemoryCandidate{
		Title:   "User prefers dark mode",
		Content: "Preference for dark theme interface",
		Type:    "PREFERENCE",
	})

	// 3. Seed an investment decision & proposal
	_, _ = appInst.InvSvc.ProcessAgyDecision(ctx, investment.AgyDecisionInput{
		Symbol:       "2317",
		Action:       "BUY",
		TargetAmount: 150000,
		Confidence:   0.80,
	}, 150.0)

	// 4. Query /api/v1/pending
	reqPending := httptest.NewRequest("GET", "/api/v1/pending", nil)
	reqPending.AddCookie(cookie)
	recPending := httptest.NewRecorder()
	handler.ServeHTTP(recPending, reqPending)

	if recPending.Code != http.StatusOK {
		t.Fatalf("expected 200 from pending endpoint, got %d: %s", recPending.Code, recPending.Body.String())
	}

	var pRes struct {
		Data struct {
			TotalCount     int                          `json:"total_count"`
			Tasks          []task.Task                  `json:"tasks"`
			Memories       []memory.MemoryCandidate     `json:"memories"`
			TradeProposals []investment.TradeProposal   `json:"trade_proposals"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recPending.Body.Bytes(), &pRes)

	if pRes.Data.TotalCount != 3 {
		t.Errorf("expected total 3 pending items, got %d", pRes.Data.TotalCount)
	}
	if len(pRes.Data.Tasks) != 1 || pRes.Data.Tasks[0].ID != tk.ID {
		t.Errorf("expected 1 pending task with ID %s", tk.ID)
	}
	if len(pRes.Data.Memories) != 1 {
		t.Errorf("expected 1 pending memory candidate")
	}
	if len(pRes.Data.TradeProposals) != 1 {
		t.Errorf("expected 1 pending trade proposal")
	}

	// 5. Approve Task via pending action endpoint
	reqApproveTask := httptest.NewRequest("POST", "/api/v1/pending/tasks/"+tk.ID+"/approve", nil)
	reqApproveTask.AddCookie(cookie)
	recApproveTask := httptest.NewRecorder()
	handler.ServeHTTP(recApproveTask, reqApproveTask)
	if recApproveTask.Code != http.StatusOK {
		t.Errorf("approve task failed: %d", recApproveTask.Code)
	}

	gotTask, _ := appInst.TaskSvc.Get(ctx, tk.ID)
	if gotTask.Status != task.StatusReady {
		t.Errorf("expected task status READY after approval, got %s", gotTask.Status)
	}
}

func TestWeb_AgentModule_IsolatedRoutingAndLimits(t *testing.T) {
	appInst, handler, token := setupTestWeb(t)
	defer appInst.Close()

	// Login
	loginBody, _ := json.Marshal(map[string]string{"token": token})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	recLogin := httptest.NewRecorder()
	handler.ServeHTTP(recLogin, reqLogin)
	cookie := recLogin.Result().Cookies()[0]

	// 1. Check Investment Profile mounted under /api/v1/agents/investment/profile
	reqProf := httptest.NewRequest("GET", "/api/v1/agents/investment/profile", nil)
	reqProf.AddCookie(cookie)
	recProf := httptest.NewRecorder()
	handler.ServeHTTP(recProf, reqProf)

	if recProf.Code != http.StatusOK {
		t.Fatalf("expected 200 from investment profile, got %d: %s", recProf.Code, recProf.Body.String())
	}

	// 2. Evaluate investment proposal with Go validation & calculations
	evalReq := map[string]any{
		"symbol":        "2317",
		"action":        "BUY",
		"price":         150.0,
		"target_amount": 160000,
		"confidence":    0.85,
		"reasoning":     []string{"Solid growth"},
	}
	evalBody, _ := json.Marshal(evalReq)
	reqEval := httptest.NewRequest("POST", "/api/v1/agents/investment/evaluate", bytes.NewReader(evalBody))
	reqEval.AddCookie(cookie)
	reqEval.Header.Set("Content-Type", "application/json")
	recEval := httptest.NewRecorder()
	handler.ServeHTTP(recEval, reqEval)

	if recEval.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from evaluate, got %d: %s", recEval.Code, recEval.Body.String())
	}

	// 3. Decision that violates confidence (< 0.50) is rejected by Go
	lowConfReq := map[string]any{
		"symbol":     "2317",
		"action":     "BUY",
		"price":      150.0,
		"confidence": 0.30,
	}
	lowConfBody, _ := json.Marshal(lowConfReq)
	reqLow := httptest.NewRequest("POST", "/api/v1/agents/investment/evaluate", bytes.NewReader(lowConfBody))
	reqLow.AddCookie(cookie)
	reqLow.Header.Set("Content-Type", "application/json")
	recLow := httptest.NewRecorder()
	handler.ServeHTTP(recLow, reqLow)

	if recLow.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 Unprocessable Entity for rejected decision, got %d", recLow.Code)
	}
}

func TestWeb_Audit_QueryWithMetadataAndSource(t *testing.T) {
	appInst, handler, token := setupTestWeb(t)
	defer appInst.Close()

	// Login
	loginBody, _ := json.Marshal(map[string]string{"token": token})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	recLogin := httptest.NewRecorder()
	handler.ServeHTTP(recLogin, reqLogin)
	cookie := recLogin.Result().Cookies()[0]

	// Perform a mutating action to produce an audit record
	taskReq := map[string]any{"title": "Audit test task"}
	body, _ := json.Marshal(taskReq)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
	req.AddCookie(cookie)
	req.Header.Set("X-Request-ID", "req_audit_test_123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Query audit logs with request_id filter
	reqAudit := httptest.NewRequest("GET", "/api/v1/audit?request_id=req_audit_test_123", nil)
	reqAudit.AddCookie(cookie)
	recAudit := httptest.NewRecorder()
	handler.ServeHTTP(recAudit, reqAudit)

	if recAudit.Code != http.StatusOK {
		t.Fatalf("expected 200 from audit query, got %d", recAudit.Code)
	}

	var res struct {
		Data []struct {
			RequestID string `json:"request_id"`
			Source    string `json:"source"`
			Action    string `json:"action"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recAudit.Body.Bytes(), &res)

	if len(res.Data) == 0 {
		t.Fatalf("expected audit log record with request_id req_audit_test_123")
	}
	if res.Data[0].RequestID != "req_audit_test_123" {
		t.Errorf("expected request_id req_audit_test_123, got %s", res.Data[0].RequestID)
	}
	if res.Data[0].Source != "WEB" {
		t.Errorf("expected source WEB, got %s", res.Data[0].Source)
	}
}

func TestWeb_CodingAndWritingAgentModules(t *testing.T) {
	appInst, handler, token := setupTestWeb(t)
	defer appInst.Close()

	// Login
	loginBody, _ := json.Marshal(map[string]string{"token": token})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	recLogin := httptest.NewRecorder()
	handler.ServeHTTP(recLogin, reqLogin)
	cookie := recLogin.Result().Cookies()[0]

	// 1. Coding Agent Info
	reqCodingInfo := httptest.NewRequest("GET", "/api/v1/agents/coding/info", nil)
	reqCodingInfo.AddCookie(cookie)
	recCodingInfo := httptest.NewRecorder()
	handler.ServeHTTP(recCodingInfo, reqCodingInfo)
	if recCodingInfo.Code != http.StatusOK {
		t.Fatalf("expected 200 from coding/info, got %d: %s", recCodingInfo.Code, recCodingInfo.Body.String())
	}

	// 2. Coding Register Project
	codingRegBody, _ := json.Marshal(map[string]string{
		"name": "Test Coding Repo",
		"path": ".",
	})
	reqCodingReg := httptest.NewRequest("POST", "/api/v1/agents/coding/projects", bytes.NewReader(codingRegBody))
	reqCodingReg.AddCookie(cookie)
	reqCodingReg.Header.Set("Content-Type", "application/json")
	recCodingReg := httptest.NewRecorder()
	handler.ServeHTTP(recCodingReg, reqCodingReg)
	if recCodingReg.Code != http.StatusCreated {
		t.Fatalf("expected 201 from coding/projects, got %d: %s", recCodingReg.Code, recCodingReg.Body.String())
	}
	var codingCreated struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recCodingReg.Body.Bytes(), &codingCreated)

	// 3. Coding Scan
	codingScanBody, _ := json.Marshal(map[string]string{})
	reqCodingScan := httptest.NewRequest("POST", "/api/v1/agents/coding/scan", bytes.NewReader(codingScanBody))
	reqCodingScan.AddCookie(cookie)
	reqCodingScan.Header.Set("Content-Type", "application/json")
	recCodingScan := httptest.NewRecorder()
	handler.ServeHTTP(recCodingScan, reqCodingScan)
	if recCodingScan.Code != http.StatusOK {
		t.Fatalf("expected 200 from coding/scan, got %d: %s", recCodingScan.Code, recCodingScan.Body.String())
	}

	// 4. Coding Delete Project
	reqCodingDel := httptest.NewRequest("DELETE", "/api/v1/agents/coding/projects?id="+codingCreated.Data.ID, nil)
	reqCodingDel.AddCookie(cookie)
	recCodingDel := httptest.NewRecorder()
	handler.ServeHTTP(recCodingDel, reqCodingDel)
	if recCodingDel.Code != http.StatusOK {
		t.Fatalf("expected 200 from DELETE coding/projects, got %d: %s", recCodingDel.Code, recCodingDel.Body.String())
	}

	// 5. Writing Agent Info
	reqWritingInfo := httptest.NewRequest("GET", "/api/v1/agents/writing/info", nil)
	reqWritingInfo.AddCookie(cookie)
	recWritingInfo := httptest.NewRecorder()
	handler.ServeHTTP(recWritingInfo, reqWritingInfo)
	if recWritingInfo.Code != http.StatusOK {
		t.Fatalf("expected 200 from writing/info, got %d: %s", recWritingInfo.Code, recWritingInfo.Body.String())
	}

	// 6. Writing Register Novel Project
	writingRegBody, _ := json.Marshal(map[string]string{
		"name": "Test Novel",
		"path": ".",
	})
	reqWritingReg := httptest.NewRequest("POST", "/api/v1/agents/writing/projects", bytes.NewReader(writingRegBody))
	reqWritingReg.AddCookie(cookie)
	reqWritingReg.Header.Set("Content-Type", "application/json")
	recWritingReg := httptest.NewRecorder()
	handler.ServeHTTP(recWritingReg, reqWritingReg)
	if recWritingReg.Code != http.StatusCreated {
		t.Fatalf("expected 201 from writing/projects, got %d: %s", recWritingReg.Code, recWritingReg.Body.String())
	}
	var writingCreated struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recWritingReg.Body.Bytes(), &writingCreated)

	// 6.1 Writing Workflow Initiate & Brief Query
	initBody, _ := json.Marshal(map[string]string{
		"project_id": writingCreated.Data.ID,
		"title":      "Test Novel",
		"raw_intent": "歷史小說創作意圖測試",
		"genre":      "historical_fiction",
		"workspace":  t.TempDir(),
	})
	reqInit := httptest.NewRequest("POST", "/api/v1/agents/writing/initiate", bytes.NewReader(initBody))
	reqInit.AddCookie(cookie)
	reqInit.Header.Set("Content-Type", "application/json")
	recInit := httptest.NewRecorder()
	handler.ServeHTTP(recInit, reqInit)
	if recInit.Code != http.StatusOK {
		t.Fatalf("expected 200 from writing/initiate, got %d: %s", recInit.Code, recInit.Body.String())
	}

	// 6.2 Writing Brief Query
	reqBrief := httptest.NewRequest("GET", "/api/v1/agents/writing/brief?project_id="+writingCreated.Data.ID, nil)
	reqBrief.AddCookie(cookie)
	recBrief := httptest.NewRecorder()
	handler.ServeHTTP(recBrief, reqBrief)
	if recBrief.Code != http.StatusOK {
		t.Fatalf("expected 200 from writing/brief, got %d: %s", recBrief.Code, recBrief.Body.String())
	}

	// 6.3 Writing Clarifications Query
	reqClarify := httptest.NewRequest("GET", "/api/v1/agents/writing/clarifications?project_id="+writingCreated.Data.ID, nil)
	reqClarify.AddCookie(cookie)
	recClarify := httptest.NewRecorder()
	handler.ServeHTTP(recClarify, reqClarify)
	if recClarify.Code != http.StatusOK {
		t.Fatalf("expected 200 from writing/clarifications, got %d: %s", recClarify.Code, recClarify.Body.String())
	}

	// 7. Writing Delete Project
	reqWritingDel := httptest.NewRequest("DELETE", "/api/v1/agents/writing/projects?id="+writingCreated.Data.ID, nil)
	reqWritingDel.AddCookie(cookie)
	recWritingDel := httptest.NewRecorder()
	handler.ServeHTTP(recWritingDel, reqWritingDel)
	if recWritingDel.Code != http.StatusOK {
		t.Fatalf("expected 200 from DELETE writing/projects, got %d: %s", recWritingDel.Code, recWritingDel.Body.String())
	}

	// 7b. Writing Project Isolation Check (No hardcoded "yuan" fallback - AIA-21 §8.3)
	reqNoProj := httptest.NewRequest("GET", "/api/v1/agents/writing/manifest", nil)
	reqNoProj.AddCookie(cookie)
	recNoProj := httptest.NewRecorder()
	handler.ServeHTTP(recNoProj, reqNoProj)
	if recNoProj.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when project is omitted and no writing projects exist (no hardcoded 'yuan' fallback), got %d: %s", recNoProj.Code, recNoProj.Body.String())
	}
	if !strings.Contains(recNoProj.Body.String(), "no writing project found") {
		t.Fatalf("expected error message indicating no writing project found, got: %s", recNoProj.Body.String())
	}

	// 8. Investment Agent Info and Instructions Update
	reqInvInfo := httptest.NewRequest("GET", "/api/v1/agents/investment/info", nil)
	reqInvInfo.AddCookie(cookie)
	recInvInfo := httptest.NewRecorder()
	handler.ServeHTTP(recInvInfo, reqInvInfo)
	if recInvInfo.Code != http.StatusOK {
		t.Fatalf("expected 200 from investment/info, got %d: %s", recInvInfo.Code, recInvInfo.Body.String())
	}

	invInstBody, _ := json.Marshal(map[string]string{
		"instructions": "Be ultra conservative with capital.",
	})
	reqInvInst := httptest.NewRequest("PUT", "/api/v1/agents/investment/instructions", bytes.NewReader(invInstBody))
	reqInvInst.AddCookie(cookie)
	reqInvInst.Header.Set("Content-Type", "application/json")
	recInvInst := httptest.NewRecorder()
	handler.ServeHTTP(recInvInst, reqInvInst)
	if recInvInst.Code != http.StatusOK {
		t.Fatalf("expected 200 from investment/instructions, got %d: %s", recInvInst.Code, recInvInst.Body.String())
	}

	// 9. Prompt Submission and Progress Query
	promptBody, _ := json.Marshal(map[string]any{
		"prompt": "Test code analysis prompt",
		"async":  false,
	})
	reqPrompt := httptest.NewRequest("POST", "/api/v1/agents/coding/prompt", bytes.NewReader(promptBody))
	reqPrompt.AddCookie(cookie)
	reqPrompt.Header.Set("Content-Type", "application/json")
	recPrompt := httptest.NewRecorder()
	handler.ServeHTTP(recPrompt, reqPrompt)
	if recPrompt.Code != http.StatusOK {
		t.Fatalf("expected 200 from coding/prompt, got %d: %s", recPrompt.Code, recPrompt.Body.String())
	}

	var promptResp struct {
		Data struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recPrompt.Body.Bytes(), &promptResp)
	if promptResp.Data.TaskID == "" {
		t.Fatalf("expected valid task_id from prompt submission")
	}

	// 10. Query Task Progress
	reqProgress := httptest.NewRequest("GET", "/api/v1/tasks/"+promptResp.Data.TaskID+"/progress", nil)
	reqProgress.AddCookie(cookie)
	recProgress := httptest.NewRecorder()
	handler.ServeHTTP(recProgress, reqProgress)
	if recProgress.Code != http.StatusOK {
		t.Fatalf("expected 200 from tasks/progress, got %d: %s", recProgress.Code, recProgress.Body.String())
	}

	var progressResp struct {
		Data struct {
			TaskID         string `json:"task_id"`
			Status         string `json:"status"`
			ConversationID string `json:"conversation_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recProgress.Body.Bytes(), &progressResp)
	if progressResp.Data.TaskID != promptResp.Data.TaskID {
		t.Fatalf("expected progress for task %s, got %s", promptResp.Data.TaskID, progressResp.Data.TaskID)
	}

	// 11. Query Agent Recent Tasks
	reqAgentTasks := httptest.NewRequest("GET", "/api/v1/agents/coding/tasks?limit=5", nil)
	reqAgentTasks.AddCookie(cookie)
	recAgentTasks := httptest.NewRecorder()
	handler.ServeHTTP(recAgentTasks, reqAgentTasks)
	if recAgentTasks.Code != http.StatusOK {
		t.Fatalf("expected 200 from coding/tasks, got %d: %s", recAgentTasks.Code, recAgentTasks.Body.String())
	}
	var agentTasksResp struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(recAgentTasks.Body.Bytes(), &agentTasksResp)
	if len(agentTasksResp.Data) == 0 {
		t.Fatalf("expected at least 1 recent coding task")
	}
}

func TestWeb_ClarificationsAndProjectBriefs(t *testing.T) {
	appInst, handler, token := setupTestWeb(t)
	defer appInst.Close()
	ctx := context.Background()

	// 1. Authenticate with one-time token
	loginBody, _ := json.Marshal(map[string]string{"token": token})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	recLogin := httptest.NewRecorder()
	handler.ServeHTTP(recLogin, reqLogin)
	if recLogin.Code != http.StatusOK {
		t.Fatalf("login failed: %d body: %s", recLogin.Code, recLogin.Body.String())
	}
	cookie := recLogin.Result().Cookies()[0]

	// 2. Setup project and initialize brief
	projID := "proj-clarify-web"
	wsDir := t.TempDir()
	if _, err := appInst.ProjectSvc.RegisterFromConfig(ctx, projID, "徒步環島規劃", wsDir, domainProject.DomainGeneral); err != nil {
		t.Fatalf("failed to register project: %v", err)
	}
	if _, err := appInst.GenericBriefSvc.InitializeBrief(ctx, projID, "nonfiction_plan", "徒步環台灣本島", wsDir); err != nil {
		t.Fatalf("failed to initialize brief: %v", err)
	}

	// 3. GET /api/v1/projects/{id}/brief
	reqBrief := httptest.NewRequest("GET", "/api/v1/projects/"+projID+"/brief", nil)
	reqBrief.AddCookie(cookie)
	recBrief := httptest.NewRecorder()
	handler.ServeHTTP(recBrief, reqBrief)
	if recBrief.Code != http.StatusOK {
		t.Fatalf("expected 200 from GET /api/v1/projects/{id}/brief, got %d: %s", recBrief.Code, recBrief.Body.String())
	}

	var briefResp struct {
		Data struct {
			ProjectID string `json:"project_id"`
			Domain    string `json:"domain"`
			Readiness string `json:"readiness"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recBrief.Body.Bytes(), &briefResp); err != nil {
		t.Fatalf("failed to parse brief json: %v", err)
	}
	if briefResp.Data.ProjectID != projID || briefResp.Data.Domain != "nonfiction_plan" {
		t.Fatalf("unexpected brief response: %+v", briefResp)
	}

	// 4. Create Clarification Requests
	reqs := []*domainClarify.ClarificationRequest{
		{
			ID:        "cr-web-01",
			ProjectID: projID,
			FieldKey:  "route_scope",
			Question:  "徒步路線是順時針還是逆時針？",
			Reason:    "規劃路線方向與補給順序",
			Impact:    "全書各章節結構",
			Priority:  domainClarify.PriorityP0,
			Blocking:  true,
		},
		{
			ID:        "cr-web-02",
			ProjectID: projID,
			FieldKey:  "accommodation_preference",
			Question:  "沿途住宿方式？",
			Reason:    "預算與裝備重量估算",
			Impact:    "裝備章節篇幅",
			Priority:  domainClarify.PriorityP1,
			Blocking:  false,
		},
	}
	if err := appInst.ClarificationSvc.CreateRequests(ctx, reqs); err != nil {
		t.Fatalf("failed to create clarification requests: %v", err)
	}

	// 5. GET /api/v1/clarifications?project_id=...
	reqList := httptest.NewRequest("GET", "/api/v1/clarifications?project_id="+projID, nil)
	reqList.AddCookie(cookie)
	recList := httptest.NewRecorder()
	handler.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 from GET /api/v1/clarifications, got %d: %s", recList.Code, recList.Body.String())
	}

	var listResp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recList.Body.Bytes(), &listResp); err != nil || len(listResp.Data) != 2 {
		t.Fatalf("expected 2 clarifications in response, got %d (err: %v)", len(listResp.Data), err)
	}

	// 6. GET /api/v1/clarifications/{id}
	reqGet := httptest.NewRequest("GET", "/api/v1/clarifications/cr-web-01", nil)
	reqGet.AddCookie(cookie)
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 from GET /api/v1/clarifications/cr-web-01, got %d: %s", recGet.Code, recGet.Body.String())
	}

	// 7. POST /api/v1/clarifications/{id}/answer
	answerPayload, _ := json.Marshal(map[string]string{"answer": "順時針走海線"})
	reqAns := httptest.NewRequest("POST", "/api/v1/clarifications/cr-web-01/answer", bytes.NewReader(answerPayload))
	reqAns.Header.Set("Content-Type", "application/json")
	reqAns.AddCookie(cookie)
	recAns := httptest.NewRecorder()
	handler.ServeHTTP(recAns, reqAns)
	if recAns.Code != http.StatusOK {
		t.Fatalf("expected 200 from answer, got %d: %s", recAns.Code, recAns.Body.String())
	}

	var ansResp struct {
		Data struct {
			Status string `json:"status"`
			Answer string `json:"answer"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recAns.Body.Bytes(), &ansResp)
	if ansResp.Data.Status != "ANSWERED" || ansResp.Data.Answer != "順時針走海線" {
		t.Fatalf("unexpected answer response: %+v", ansResp)
	}

	// 8. GET /api/v1/projects/{id}/brief/history?field=route_scope
	reqHist := httptest.NewRequest("GET", "/api/v1/projects/"+projID+"/brief/history?field=route_scope", nil)
	reqHist.AddCookie(cookie)
	recHist := httptest.NewRecorder()
	handler.ServeHTTP(recHist, reqHist)
	if recHist.Code != http.StatusOK {
		t.Fatalf("expected 200 from history, got %d: %s", recHist.Code, recHist.Body.String())
	}

	var histResp struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(recHist.Body.Bytes(), &histResp)
	if len(histResp.Data) == 0 {
		t.Fatalf("expected field history entries for route_scope")
	}

	// 9. POST /api/v1/clarifications/{id}/defer
	reqDefer := httptest.NewRequest("POST", "/api/v1/clarifications/cr-web-02/defer", nil)
	reqDefer.AddCookie(cookie)
	recDefer := httptest.NewRecorder()
	handler.ServeHTTP(recDefer, reqDefer)
	if recDefer.Code != http.StatusOK {
		t.Fatalf("expected 200 from defer, got %d: %s", recDefer.Code, recDefer.Body.String())
	}

	// 10. GET /api/v1/projects/{id}/brief/prompt (AIA-21 Stage 3)
	reqPrompt := httptest.NewRequest("GET", "/api/v1/projects/"+projID+"/brief/prompt", nil)
	reqPrompt.AddCookie(cookie)
	recPrompt2 := httptest.NewRecorder()
	handler.ServeHTTP(recPrompt2, reqPrompt)
	if recPrompt2.Code != http.StatusOK {
		t.Fatalf("expected 200 from brief prompt, got %d: %s", recPrompt2.Code, recPrompt2.Body.String())
	}
	var promptResp2 struct {
		Data struct {
			Prompt string `json:"prompt"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recPrompt2.Body.Bytes(), &promptResp2)
	if !strings.Contains(promptResp2.Data.Prompt, projID) || !strings.Contains(promptResp2.Data.Prompt, "AGENT BRIEF ANALYSIS") {
		t.Fatalf("unexpected prompt content: %s", promptResp2.Data.Prompt)
	}

	// 11. POST /api/v1/projects/{id}/brief/analysis (AIA-21 Stage 3)
	analysisPayload := map[string]any{
		"summary": "非虛構徒步環島規劃書分析報告",
		"understanding": []string{
			"規劃實際徒步行程與圖書結構",
		},
		"missing_fields": []map[string]any{},
		"clarification_questions": []map[string]any{},
		"proposed_defaults": []map[string]any{
			{
				"field":      "daily_mileage_km",
				"value":      22,
				"rationale":  "依據一般徒步者體力預設 22 公里",
				"confidence": 0.65,
			},
		},
		"research_candidates": []string{
			"環島路線補給站研究",
		},
		"readiness": map[string]any{
			"status": "READY_WITH_ASSUMPTIONS",
			"reason": "主要路線已確定，部分每日里程採保守預設",
		},
	}
	analysisBytes, _ := json.Marshal(analysisPayload)
	reqApply := httptest.NewRequest("POST", "/api/v1/projects/"+projID+"/brief/analysis", bytes.NewReader(analysisBytes))
	reqApply.Header.Set("Content-Type", "application/json")
	reqApply.AddCookie(cookie)
	recApply := httptest.NewRecorder()
	handler.ServeHTTP(recApply, reqApply)
	if recApply.Code != http.StatusOK {
		t.Fatalf("expected 200 from brief analysis apply, got %d: %s", recApply.Code, recApply.Body.String())
	}

	var applyResp struct {
		Data struct {
			ProjectID string `json:"project_id"`
			Readiness string `json:"readiness"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recApply.Body.Bytes(), &applyResp)
	if applyResp.Data.ProjectID != projID {
		t.Fatalf("expected project %s in apply response, got: %+v", projID, applyResp)
	}
}
