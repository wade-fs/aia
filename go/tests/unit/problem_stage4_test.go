package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	problemService "github.com/wade/aia/internal/service/problem"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/web/console"
)

func TestProblemDecomposition_DepthAndSubProblemLimits(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)

	svc := problemService.NewService(problemRepo, nil, projRepo, nil)
	// Set custom tight bounds: MaxDepth = 3, MaxSubProblems = 4
	svc.SetDecompositionConfig(domainProblem.DecompositionConfig{
		MaxDepth:       3,
		MaxSubProblems: 4,
	})

	proj := &domainProject.Project{
		ID:         "proj-decomp-test",
		Name:       "遞迴拆解專案",
		Workspace:  "/workspace/decomp",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	// Level 1: Root Problem (Depth 1)
	root, err := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "全書規範合規性稽核",
		Objective: "全書 540 章規範檢查",
	})
	if err != nil {
		t.Fatalf("failed to create root problem: %v", err)
	}

	dRoot, err := svc.GetProblemDepth(ctx, root.ID)
	if err != nil || dRoot != 1 {
		t.Fatalf("expected root depth 1, got %d (err: %v)", dRoot, err)
	}

	// Level 2: Decompose Root -> Child 1 (Depth 2)
	childList, err := svc.DecomposeProblem(ctx, root.ID, []problemService.CreateProblemInput{
		{
			Title:     "分卷結構稽核",
			Objective: "檢查各卷章節結構",
		},
	})
	if err != nil || len(childList) != 1 {
		t.Fatalf("failed to decompose into Level 2: %v", err)
	}
	child1 := childList[0]

	dChild1, err := svc.GetProblemDepth(ctx, child1.ID)
	if err != nil || dChild1 != 2 {
		t.Fatalf("expected child1 depth 2, got %d (err: %v)", dChild1, err)
	}

	// Level 3: Decompose Child 1 -> Grandchild 1 (Depth 3)
	gcList, err := svc.DecomposeProblem(ctx, child1.ID, []problemService.CreateProblemInput{
		{
			Title:     "第 1 卷人物時間軸檢查",
			Objective: "檢查登場人物年齡一致性",
		},
	})
	if err != nil || len(gcList) != 1 {
		t.Fatalf("failed to decompose into Level 3: %v", err)
	}
	gc1 := gcList[0]

	dGc1, err := svc.GetProblemDepth(ctx, gc1.ID)
	if err != nil || dGc1 != 3 {
		t.Fatalf("expected grandchild depth 3, got %d", dGc1)
	}

	// Level 4: Attempt Decompose Grandchild 1 -> Depth 4 (Must exceed MaxDepth = 3) (AIA-22 §5.2)
	_, err = svc.DecomposeProblem(ctx, gc1.ID, []problemService.CreateProblemInput{
		{
			Title:     "第 1 卷第 1 章主角年齡檢查",
			Objective: "細化至單章單人",
		},
	})
	if err == nil {
		t.Fatalf("expected ErrMaxDepthExceeded when exceeding MaxDepth 3, got nil")
	}

	// Budget Check: Total subproblems limit is 4. Currently existing: 2 (child1, gc1).
	// Attempt to add 3 more subproblems (total would be 5 > 4). Must fail with ErrMaxSubProblemsExceeded (AIA-22 §5.2)
	_, err = svc.DecomposeProblem(ctx, root.ID, []problemService.CreateProblemInput{
		{Title: "Sub 2", Objective: "Obj 2"},
		{Title: "Sub 3", Objective: "Obj 3"},
		{Title: "Sub 4", Objective: "Obj 4"},
	})
	if err == nil {
		t.Fatalf("expected ErrMaxSubProblemsExceeded when exceeding MaxSubProblems 4, got nil")
	}
}

func TestProblemHierarchy_TreeTraversal(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)

	svc := problemService.NewService(problemRepo, nil, projRepo, nil)

	proj := &domainProject.Project{
		ID:         "proj-tree-test",
		Name:       "樹狀遍歷專案",
		Workspace:  "/workspace/tree",
		DomainType: domainProject.DomainCoding,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	// 1. Root: Go Repository Refactoring
	root, _ := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "Go 專案架構重構與技術債清理",
		Objective: "清理全專案依賴與提升穩定性",
	})

	// 2. Level 2 Children
	subs, err := svc.DecomposeProblem(ctx, root.ID, []problemService.CreateProblemInput{
		{Title: "儲存庫 Repository 層重構", Objective: "統一 ORM 與介面封裝"},
		{Title: "Autonomy Core 效能調優", Objective: "降低觀察循環 CPU 佔用"},
	})
	if err != nil {
		t.Fatalf("failed to decompose Level 2: %v", err)
	}

	// 3. Level 3 Grandchildren under subs[0]
	_, err = svc.DecomposeProblem(ctx, subs[0].ID, []problemService.CreateProblemInput{
		{Title: "SQLite Migration 版本清理", Objective: "消除歷史重複索引"},
	})
	if err != nil {
		t.Fatalf("failed to decompose Level 3: %v", err)
	}

	// 4. Query Problem Tree (AIA-22 §5.1)
	tree, err := svc.GetProblemTree(ctx, root.ID)
	if err != nil {
		t.Fatalf("failed to get problem tree: %v", err)
	}

	if tree.Problem.ID != root.ID || tree.Depth != 1 {
		t.Fatalf("unexpected root tree node: depth=%d, id=%s", tree.Depth, tree.Problem.ID)
	}
	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 children under root, got %d", len(tree.Children))
	}
	if tree.Children[0].Problem.ID != subs[0].ID || tree.Children[0].Depth != 2 {
		t.Fatalf("unexpected child 0: depth=%d", tree.Children[0].Depth)
	}
	if len(tree.Children[0].Children) != 1 || tree.Children[0].Children[0].Depth != 3 {
		t.Fatalf("unexpected grandchild under child 0: count=%d", len(tree.Children[0].Children))
	}
}

func TestReassessment_Trigger_RootCause_And_HypothesisContradiction(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)

	svc := problemService.NewService(problemRepo, nil, projRepo, nil)

	proj := &domainProject.Project{
		ID:         "proj-reassess-test",
		Name:       "重評驗證專案",
		Workspace:  "/workspace/reassess",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	prob, _ := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "大唐長安歷史考證稽核",
		Objective: "核對官名與年號",
	})

	// Setup initial Model Version 1 with an assumption that turns out to be contradicted
	m1 := &domainProblem.ProblemModel{
		ID:        "model-reassess-v1",
		ProblemID: prob.ID,
		Version:   1,
		Assumptions: []domainProblem.ModelItem{
			{
				ID:         "assump-tang-era",
				Category:   domainProblem.CategoryAssumption,
				Statement:  "主角任職尚書省之時間點設定於貞觀十四年",
				Source:     "initial_draft_notes",
				Confidence: 0.75,
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "writing-agent",
				CreatedAt:  time.Now().UTC(),
			},
			{
				ID:         "assump-location",
				Category:   domainProblem.CategoryAssumption,
				Statement:  "西市胡商聚集於開元天寶年間",
				Source:     "history_handbook",
				Confidence: 0.85,
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "writing-agent",
				CreatedAt:  time.Now().UTC(),
			},
		},
	}
	if err := problemRepo.SaveModel(ctx, m1); err != nil {
		t.Fatalf("failed to save initial model: %v", err)
	}

	// 1. Trigger Reassessment (AIA-22 §8.1, §8.2, §8.3)
	// New historical evidence contradicts "assump-tang-era"
	reassessReq := domainProblem.ReassessmentRequest{
		ProblemID:              prob.ID,
		Trigger:                domainProblem.TriggerHypothesisContradicted,
		TriggerEvidence:        "《舊唐書·職官志》考證：該職官設立於開元二十八年，貞觀年間尚未置此官職",
		RootCause:              domainProblem.RootCauseAssumptionInvalid, // Explicit root-cause attribution
		ContradictedHypotheses: []string{"assump-tang-era"},
		NewFindings: []string{
			"考證確認該官職於開元二十八年方始設立，貞觀年間不存在",
		},
		SuggestedAction: domainProblem.ActionModifyPlan,
		Rationale:       "原設定之貞觀年代假設不成立，必須修訂計畫重新編排第 2 卷登場職官",
		AgentID:         "research-agent",
	}

	result, err := svc.TriggerReassessment(ctx, reassessReq)
	if err != nil {
		t.Fatalf("failed to trigger reassessment: %v", err)
	}

	if result.PreviousModelVer != 1 || result.NewModelVer != 2 {
		t.Fatalf("expected version advance 1 -> 2, got v%d -> v%d", result.PreviousModelVer, result.NewModelVer)
	}
	if result.ContradictedCount != 1 {
		t.Fatalf("expected 1 contradicted hypothesis, got %d", result.ContradictedCount)
	}
	if !result.RequiresNewPlan {
		t.Fatalf("expected RequiresNewPlan = true on MODIFY_PLAN action")
	}

	// 2. Verify Model v2: "assump-tang-era" must now be CONTRADICTED
	m2, err := problemRepo.GetModel(ctx, prob.ID, 2)
	if err != nil {
		t.Fatalf("failed to get model v2: %v", err)
	}

	var foundContradicted bool
	for _, a := range m2.Assumptions {
		if a.ID == "assump-tang-era" && a.Status == domainProblem.ItemStatusContradicted {
			foundContradicted = true
		}
	}
	if !foundContradicted {
		t.Fatalf("assumption assump-tang-era was not marked as CONTRADICTED in model v2: %+v", m2.Assumptions)
	}

	// Verify new finding integrated into KnownFacts of Model v2
	if len(m2.KnownFacts) != 1 || m2.KnownFacts[0].Confidence != 1.0 {
		t.Fatalf("new confirmed finding not added to KnownFacts in model v2: %+v", m2.KnownFacts)
	}

	// 3. Immutability check: Model v1 must remain ACTIVE (unmutated historical state)
	m1History, _ := problemRepo.GetModel(ctx, prob.ID, 1)
	if m1History.Assumptions[0].Status != domainProblem.ItemStatusActive {
		t.Fatalf("historical model v1 assumption was mutated! got status %s", m1History.Assumptions[0].Status)
	}

	// 4. Verify Problem status updated to REASSESSING
	probReload, _ := svc.GetProblem(ctx, prob.ID)
	if probReload.Status != domainProblem.StatusReassessing {
		t.Fatalf("expected problem status REASSESSING, got %s", probReload.Status)
	}

	// 5. Verify immutable AnalysisRecord with REASSESSMENT type recorded
	records, err := svc.ListAnalysisRecords(ctx, prob.ID)
	if err != nil || len(records) != 1 {
		t.Fatalf("expected 1 analysis record, got %d (err: %v)", len(records), err)
	}
	if records[0].AnalysisType != "REASSESSMENT" {
		t.Fatalf("expected record type REASSESSMENT, got %s", records[0].AnalysisType)
	}
}

func TestReassessment_WebConsole_Endpoints(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)

	svc := problemService.NewService(problemRepo, nil, projRepo, nil)

	proj := &domainProject.Project{
		ID:         "proj-web-reassess",
		Name:       "Web Reassessment Test",
		Workspace:  "/workspace/web-reassess",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	prob, _ := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "Web API 遞迴與重評測試",
		Objective: "測試端點",
	})

	mux := http.NewServeMux()
	console.RegisterConsoleRoutes(mux, &console.ConsoleServices{
		DB:         db,
		ProblemSvc: svc,
	})

	// 1. POST /api/v1/problems/{id}/decompose
	decompPayload := []problemService.CreateProblemInput{
		{
			Title:     "子問題 1",
			Objective: "目標 1",
		},
		{
			Title:     "子問題 2",
			Objective: "目標 2",
		},
	}
	dBytes, _ := json.Marshal(decompPayload)
	reqDecomp := httptest.NewRequest(http.MethodPost, "/api/v1/problems/"+prob.ID+"/decompose", bytes.NewReader(dBytes))
	reqDecomp.Header.Set("Content-Type", "application/json")
	wDecomp := httptest.NewRecorder()
	mux.ServeHTTP(wDecomp, reqDecomp)

	if wDecomp.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for decompose, got %d (body: %s)", wDecomp.Code, wDecomp.Body.String())
	}

	// 2. GET /api/v1/problems/{id}/tree
	reqTree := httptest.NewRequest(http.MethodGet, "/api/v1/problems/"+prob.ID+"/tree", nil)
	wTree := httptest.NewRecorder()
	mux.ServeHTTP(wTree, reqTree)

	if wTree.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for tree, got %d", wTree.Code)
	}
	var treeResp struct {
		Success bool                          `json:"success"`
		Data    domainProblem.ProblemTreeNode `json:"data"`
	}
	_ = json.Unmarshal(wTree.Body.Bytes(), &treeResp)
	if len(treeResp.Data.Children) != 2 {
		t.Fatalf("expected 2 children in tree, got %d", len(treeResp.Data.Children))
	}

	// 3. POST /api/v1/problems/{id}/reassess
	reassessPayload := domainProblem.ReassessmentRequest{
		Trigger:         domainProblem.TriggerTaskFailure,
		TriggerEvidence: "工作單元連續 3 次執行失敗",
		RootCause:       domainProblem.RootCauseExecutionFailure,
		Rationale:       "工具環境相依性缺失，需暫停並修改計畫",
		SuggestedAction: domainProblem.ActionModifyPlan,
	}
	rBytes, _ := json.Marshal(reassessPayload)
	reqReassess := httptest.NewRequest(http.MethodPost, "/api/v1/problems/"+prob.ID+"/reassess", bytes.NewReader(rBytes))
	reqReassess.Header.Set("Content-Type", "application/json")
	wReassess := httptest.NewRecorder()
	mux.ServeHTTP(wReassess, reqReassess)

	if wReassess.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for reassess, got %d (body: %s)", wReassess.Code, wReassess.Body.String())
	}
}
