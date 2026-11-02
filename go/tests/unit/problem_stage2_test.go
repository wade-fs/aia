package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainClarify "github.com/wade/aia/internal/domain/clarification"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	problemService "github.com/wade/aia/internal/service/problem"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/web/console"
)

type mockClarifyDispatcher struct {
	created []*domainClarify.ClarificationRequest
}

func (m *mockClarifyDispatcher) CreateRequests(ctx context.Context, requests []*domainClarify.ClarificationRequest) error {
	m.created = append(m.created, requests...)
	return nil
}

func TestProblemAnalysis_EpistemicValidation_And_DisciplineLimits(t *testing.T) {
	// 1. Empty user goal rejected (AIA-22 §4.2)
	badIntent := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: ""},
	}
	if err := domainProblem.ValidateProblemAnalysis(badIntent); err == nil {
		t.Fatalf("expected error on empty user goal, got nil")
	}

	// 2. Fact without 1.0 confidence rejected (AIA-22 §7.1, §7.2)
	badFact := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: "Audit 540 chapters"},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "fact-1",
				Category:   domainProblem.CategoryKnownFact,
				Statement:  "The book has 10 volumes",
				Source:     "manifest.yaml",
				Confidence: 0.95, // must be 1.0
			},
		},
	}
	if err := domainProblem.ValidateProblemAnalysis(badFact); err == nil {
		t.Fatalf("expected error on fact with confidence < 1.0, got nil")
	}

	// 3. Fact without source rejected
	badFactSource := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: "Audit 540 chapters"},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "fact-2",
				Category:   domainProblem.CategoryKnownFact,
				Statement:  "The book has 10 volumes",
				Source:     "", // empty source rejected
				Confidence: 1.0,
			},
		},
	}
	if err := domainProblem.ValidateProblemAnalysis(badFactSource); err == nil {
		t.Fatalf("expected error on fact with empty source, got nil")
	}

	// 4. Assumption claiming 1.0 confidence rejected (AIA-22 §7.1: "不能將推測寫入 Verified Fact")
	badAssumption := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: "Audit 540 chapters"},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "assump-1",
				Category:   domainProblem.CategoryAssumption,
				Statement:  "Historical accuracy is strict Tang Dynasty",
				Confidence: 1.0, // assumption cannot be 1.0
			},
		},
	}
	if err := domainProblem.ValidateProblemAnalysis(badAssumption); err == nil {
		t.Fatalf("expected error on assumption claiming 1.0 confidence, got nil")
	}

	// 5. Unknown with non-zero confidence rejected
	badUnknown := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: "Audit 540 chapters"},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "unk-1",
				Category:   domainProblem.CategoryUnknown,
				Statement:  "Actual delivery deadline",
				Confidence: 0.5, // unknown must be 0.0
			},
		},
	}
	if err := domainProblem.ValidateProblemAnalysis(badUnknown); err == nil {
		t.Fatalf("expected error on unknown with confidence != 0.0, got nil")
	}

	// 6. Question discipline: more than 5 questions rejected (AIA-21 §5.4, AIA-22 §4.5)
	badQuestions := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: "Audit 540 chapters"},
		ClarificationQuestions: []*domainClarify.ClarificationRequest{
			{FieldKey: "f1", Question: "q1", Reason: "r1", Impact: "i1"},
			{FieldKey: "f2", Question: "q2", Reason: "r2", Impact: "i2"},
			{FieldKey: "f3", Question: "q3", Reason: "r3", Impact: "i3"},
			{FieldKey: "f4", Question: "q4", Reason: "r4", Impact: "i4"},
			{FieldKey: "f5", Question: "q5", Reason: "r5", Impact: "i5"},
			{FieldKey: "f6", Question: "q6", Reason: "r6", Impact: "i6"}, // 6th question violates discipline
		},
	}
	if err := domainProblem.ValidateProblemAnalysis(badQuestions); err == nil {
		t.Fatalf("expected error on > 5 clarification questions, got nil")
	}
}

func TestProblemService_ApplyAnalysis_And_ModelEvolution(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)
	dispatcher := &mockClarifyDispatcher{}

	svc := problemService.NewService(problemRepo, planRepo, projRepo, dispatcher)

	// Create project and initial problem
	proj := &domainProject.Project{
		ID:         "proj-novel-540",
		Name:       "長安全書合規稽核專案",
		Workspace:  "/workspace/changan",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}

	prob, err := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "請幫我看看這本540章的小說是否符合所有規範",
		Objective: "全書合規性檢查",
		Priority:  domainProblem.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("failed to create initial problem: %v", err)
	}

	// Construct structured agent analysis
	analysis := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{
			UserGoal:        "執行全書 540 章規範合規性稽核，產出完整稽核報告與衝突清單",
			RequestedMethod: "靜態檔案掃描結合史料比對",
			Constraints:     []string{"不可直接修改原文", "只進行稽核報告產出"},
			SuccessCriteria: []string{"所有卷章均有完整稽核日誌", "標示所有違規與衝突點"},
			AnalysisOnly:    true, // AIA-22 §4.2
		},
		Context: domainProblem.ContextSummary{
			FoundResources:    []string{"book.yaml", "manifest/rules.md", "chapters/001-540.md"},
			NeedsVerification: []string{"唐朝官制稱謂是否合規", "人物壽命年代表是否出現矛盾"},
			ConflictingData:   []string{"manifest/rules.md 第12條與第45條關於官名沿革互有出入"},
		},
		GapsAndConflicts: []domainProblem.GapConflictItem{
			{
				ID:                    "gap-rule-conflict-01",
				Category:              domainProblem.GapRuleConflict,
				Description:           "規範文件第12條與第45條官制稱謂定義衝突",
				Source:                "manifest/rules.md#L12,#L45",
				Impact:                "導致第3卷多個章節判定標準不明確",
				Severity:              "HIGH",
				SuggestedHandling:     domainProblem.HandlingClarify,
				RequiresClarification: true,
			},
		},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:          "fact-chapters",
				Category:    domainProblem.CategoryKnownFact,
				Statement:   "全書包含 10 卷共 540 個 Markdown 章節檔案",
				Source:      "workspace_file_scan",
				Confidence:  1.0,
				Status:      domainProblem.ItemStatusConfirmed,
				CreatedBy:   "writing-agent",
				CreatedAt:   time.Now().UTC(),
			},
			{
				ID:          "assump-format",
				Category:    domainProblem.CategoryAssumption,
				Statement:   "每章標題皆採用標準 Markdown 二級標題格式",
				Source:      "heuristic_sampling",
				Confidence:  0.80,
				Status:      domainProblem.ItemStatusActive,
				CreatedBy:   "writing-agent",
				CreatedAt:   time.Now().UTC(),
			},
			{
				ID:          "unk-canon-version",
				Category:    domainProblem.CategoryUnknown,
				Statement:   "規範文件衝突時以哪份歷史典籍為最高權威",
				Source:      "gap_conflict_analysis",
				Confidence:  0.0,
				Status:      domainProblem.ItemStatusActive,
				CreatedBy:   "writing-agent",
				CreatedAt:   time.Now().UTC(),
			},
		},
		ClarificationQuestions: []*domainClarify.ClarificationRequest{
			{
				FieldKey: "official_title_standard",
				Question: "請問官制稱謂衝突時，應以貞觀六部規制為準，還是開元改制為準？",
				Options:  []string{"貞觀六部制", "開元改制", "以故事發生年份為準"},
				Reason:   "規範第12條與第45條存在歷史分期衝突，影響第3卷判定標準",
				Impact:   "決定第3卷 50 處官制提及是否判定為瑕疵",
				Priority: domainClarify.PriorityP1,
				Blocking: true,
			},
		},
	}

	// Apply Analysis
	model, err := svc.ApplyAnalysis(ctx, prob.ID, analysis, "writing-agent", "exec-01")
	if err != nil {
		t.Fatalf("failed to apply analysis: %v", err)
	}

	// 1. Verify Model Version bumped to 1
	if model.Version != 1 {
		t.Fatalf("expected Model Version 1, got %d", model.Version)
	}
	if len(model.KnownFacts) != 1 || model.KnownFacts[0].ID != "fact-chapters" {
		t.Fatalf("known facts not mapped properly: %+v", model.KnownFacts)
	}
	if len(model.Assumptions) != 1 || model.Assumptions[0].ID != "assump-format" {
		t.Fatalf("assumptions not mapped properly: %+v", model.Assumptions)
	}
	if len(model.Unknowns) != 1 || model.Unknowns[0].ID != "unk-canon-version" {
		t.Fatalf("unknowns not mapped properly: %+v", model.Unknowns)
	}

	// 2. Verify Analysis Records (both INTENT_AND_MODEL and GAP_CONFLICT recorded)
	records, err := svc.ListAnalysisRecords(ctx, prob.ID)
	if err != nil || len(records) != 2 {
		t.Fatalf("expected 2 analysis records, got %d (err: %v)", len(records), err)
	}
	if records[0].AnalysisType != "INTENT_AND_MODEL" || records[1].AnalysisType != "GAP_CONFLICT" {
		t.Fatalf("unexpected record types: %+v", records)
	}

	// Verify Analysis-Only scope constraint captured
	foundScopeConstraint := false
	for _, c := range records[0].Conclusions {
		if c == "Scope Constraint: Analysis-only requested by user (execution prohibited without explicit approval)" {
			foundScopeConstraint = true
			break
		}
	}
	if !foundScopeConstraint {
		t.Fatalf("analysis-only scope constraint was not captured in conclusions: %+v", records[0].Conclusions)
	}

	// 3. Verify Clarification Dispatching
	if len(dispatcher.created) != 1 {
		t.Fatalf("expected 1 clarification request dispatched, got %d", len(dispatcher.created))
	}
	dispatched := dispatcher.created[0]
	if dispatched.FieldKey != "official_title_standard" || dispatched.ProjectID != string(proj.ID) {
		t.Fatalf("unexpected dispatched clarification: %+v", dispatched)
	}

	// 4. Verify Problem entity updated
	probReloaded, err := svc.GetProblem(ctx, prob.ID)
	if err != nil {
		t.Fatalf("failed to reload problem: %v", err)
	}
	if probReloaded.CurrentModelVersion != 1 {
		t.Fatalf("expected CurrentModelVersion = 1, got %d", probReloaded.CurrentModelVersion)
	}
	if probReloaded.Status != domainProblem.StatusAnalyzing {
		t.Fatalf("expected Status = ANALYZING, got %s", probReloaded.Status)
	}
}

func TestProblemPrompt_And_Draft07Schema(t *testing.T) {
	p := &domainProblem.Problem{
		ID:        "prob-go-audit",
		ProjectID: "proj-go",
		Title:     "請幫我看看這個 Go 專案還有哪些可以改善的地方",
		Objective: "分析架構、依賴與潛在技術債",
		Scope:     "全 Repository 代碼與測試",
		Priority:  domainProblem.PriorityMedium,
	}
	proj := &domainProject.Project{
		ID:         "proj-go",
		Name:       "AIA Go Runtime",
		Workspace:  "/home/wade/src/github/aia",
		DomainType: domainProject.DomainCoding,
	}

	prompt := domainProblem.BuildProblemAnalysisPrompt(
		p,
		proj,
		"請幫我看看這個 Go 專案還有哪些可以改善的地方",
		[]string{"Found 45 packages in go/internal", "Test coverage at 78%"},
		[]string{"Rule 1: Strict level C immutability", "Rule 2: Zero workspace pollution"},
	)

	if len(prompt) == 0 {
		t.Fatalf("generated prompt is empty")
	}
	if !bytes.Contains([]byte(prompt), []byte("事實與推論分離")) {
		t.Errorf("prompt missing epistemic principles")
	}
	if !bytes.Contains([]byte(prompt), []byte("意圖邊界控制")) {
		t.Errorf("prompt missing intent boundary instructions")
	}

	schema := domainProblem.ProblemAnalysisDraft07Schema()
	if schema["$schema"] != "http://json-schema.org/draft-07/schema#" {
		t.Fatalf("schema missing Draft-07 identifier: %+v", schema)
	}
	if schema["title"] != "AgentProblemAnalysis" {
		t.Fatalf("unexpected schema title: %v", schema["title"])
	}
}

func TestProblemWebConsole_Endpoints(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)

	proj := &domainProject.Project{
		ID:         "proj-web-test",
		Name:       "Web API Test Project",
		Workspace:  "/workspace/test",
		DomainType: domainProject.DomainGeneral,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	probSvc := problemService.NewService(problemRepo, planRepo, projRepo, nil)

	mux := http.NewServeMux()
	console.RegisterConsoleRoutes(mux, &console.ConsoleServices{
		DB:         db,
		ProblemSvc: probSvc,
	})

	// 1. POST /api/v1/problems
	createBody := map[string]string{
		"project_id": "proj-web-test",
		"title":      "虛擬資產組合回測分析",
		"objective":  "評估策略夏普值與最大回撤",
		"priority":   "HIGH",
	}
	payloadBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/problems", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d (body: %s)", w.Code, w.Body.String())
	}
	var createdResp struct {
		Success bool                  `json:"success"`
		Data    domainProblem.Problem `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createdResp)
	createdProblem := createdResp.Data
	if createdProblem.ID == "" || createdProblem.Title != "虛擬資產組合回測分析" {
		t.Fatalf("unexpected created problem: %+v (raw: %s)", createdProblem, w.Body.String())
	}

	// 2. GET /api/v1/problems?project_id=proj-web-test
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/problems?project_id=proj-web-test", nil)
	wList := httptest.NewRecorder()
	mux.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", wList.Code)
	}
	var listResp struct {
		Success bool                     `json:"success"`
		Data    []*domainProblem.Problem `json:"data"`
	}
	_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
	if len(listResp.Data) != 1 || listResp.Data[0].ID != createdProblem.ID {
		t.Fatalf("expected 1 problem in list, got %+v", listResp.Data)
	}

	// 3. GET /api/v1/problems/{id}
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/problems/"+createdProblem.ID, nil)
	wGet := httptest.NewRecorder()
	mux.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", wGet.Code)
	}

	// 4. GET /api/v1/problems/{id}/prompt
	reqPrompt := httptest.NewRequest(http.MethodGet, "/api/v1/problems/"+createdProblem.ID+"/prompt", nil)
	wPrompt := httptest.NewRecorder()
	mux.ServeHTTP(wPrompt, reqPrompt)

	if wPrompt.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK for prompt, got %d", wPrompt.Code)
	}
	var promptResp struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"`
	}
	_ = json.Unmarshal(wPrompt.Body.Bytes(), &promptResp)
	if len(promptResp.Data["prompt"]) == 0 {
		t.Fatalf("expected non-empty prompt in web response")
	}

	// 5. POST /api/v1/problems/{id}/analysis
	analysisPayload := domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{
			UserGoal: "評估夏普值與最大回撤",
		},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "fact-1",
				Category:   domainProblem.CategoryKnownFact,
				Statement:  "Initial capital is 100,000 USD",
				Source:     "user_account_balance",
				Confidence: 1.0,
			},
		},
	}
	analysisBytes, _ := json.Marshal(analysisPayload)
	reqApply := httptest.NewRequest(http.MethodPost, "/api/v1/problems/"+createdProblem.ID+"/analysis", bytes.NewReader(analysisBytes))
	reqApply.Header.Set("Content-Type", "application/json")
	wApply := httptest.NewRecorder()
	mux.ServeHTTP(wApply, reqApply)

	if wApply.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on apply analysis, got %d (body: %s)", wApply.Code, wApply.Body.String())
	}
	var appliedResp struct {
		Success bool                       `json:"success"`
		Data    domainProblem.ProblemModel `json:"data"`
	}
	_ = json.Unmarshal(wApply.Body.Bytes(), &appliedResp)
	appliedModel := appliedResp.Data
	if appliedModel.Version != 1 || len(appliedModel.KnownFacts) != 1 {
		t.Fatalf("unexpected applied model: %+v", appliedModel)
	}
}
