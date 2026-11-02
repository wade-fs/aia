package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainPlan "github.com/wade/aia/internal/domain/plan"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	domainAgent "github.com/wade/aia/internal/domain/agent"
	domainTask "github.com/wade/aia/internal/domain/task"
	problemService "github.com/wade/aia/internal/service/problem"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/web/console"
)

func TestPlan_DAG_CycleDetection_And_Validation(t *testing.T) {
	// 1. Missing acceptance criteria rejected (AIA-22 §4.7)
	pBadAc := &domainPlan.Plan{
		ID:        "plan-bad-ac",
		ProblemID: "prob-1",
		Objective: "Test Objective",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-1",
				Title:              "Task without AC",
				AcceptanceCriteria: []string{}, // empty rejected
			},
		},
	}
	if err := pBadAc.Validate(); err == nil {
		t.Fatalf("expected error on empty acceptance criteria, got nil")
	}

	// 2. Non-existent dependency rejected
	pDangling := &domainPlan.Plan{
		ID:        "plan-dangling",
		ProblemID: "prob-1",
		Objective: "Test Objective",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-1",
				Title:              "Task 1",
				Dependencies:       []string{"wi-ghost"}, // ghost dependency
				AcceptanceCriteria: []string{"Criterion 1"},
			},
		},
	}
	if err := pDangling.Validate(); err == nil {
		t.Fatalf("expected error on non-existent dependency, got nil")
	}

	// 3. Self dependency rejected
	pSelf := &domainPlan.Plan{
		ID:        "plan-self",
		ProblemID: "prob-1",
		Objective: "Test Objective",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-1",
				Title:              "Task 1",
				Dependencies:       []string{"wi-1"}, // self dependency
				AcceptanceCriteria: []string{"Criterion 1"},
			},
		},
	}
	if err := pSelf.Validate(); err == nil {
		t.Fatalf("expected error on self dependency, got nil")
	}

	// 4. Circular dependency A -> B -> A rejected
	pCycle2 := &domainPlan.Plan{
		ID:        "plan-cycle-2",
		ProblemID: "prob-1",
		Objective: "Test Objective",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-a",
				Title:              "Task A",
				Dependencies:       []string{"wi-b"},
				AcceptanceCriteria: []string{"Criterion A"},
			},
			{
				ID:                 "wi-b",
				Title:              "Task B",
				Dependencies:       []string{"wi-a"},
				AcceptanceCriteria: []string{"Criterion B"},
			},
		},
	}
	if err := pCycle2.Validate(); err == nil {
		t.Fatalf("expected error on 2-node circular dependency, got nil")
	}

	// 5. Circular dependency A -> B -> C -> A rejected
	pCycle3 := &domainPlan.Plan{
		ID:        "plan-cycle-3",
		ProblemID: "prob-1",
		Objective: "Test Objective",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-1",
				Title:              "Task 1",
				Dependencies:       []string{"wi-2"},
				AcceptanceCriteria: []string{"AC 1"},
			},
			{
				ID:                 "wi-2",
				Title:              "Task 2",
				Dependencies:       []string{"wi-3"},
				AcceptanceCriteria: []string{"AC 2"},
			},
			{
				ID:                 "wi-3",
				Title:              "Task 3",
				Dependencies:       []string{"wi-1"}, // cycle back to 1
				AcceptanceCriteria: []string{"AC 3"},
			},
		},
	}
	if err := pCycle3.Validate(); err == nil {
		t.Fatalf("expected error on 3-node circular dependency, got nil")
	}

	// 6. Valid Diamond DAG: wi-1 -> wi-2, wi-1 -> wi-3, (wi-2, wi-3) -> wi-4
	pDiamond := &domainPlan.Plan{
		ID:        "plan-diamond",
		ProblemID: "prob-1",
		Objective: "Valid Diamond Plan",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-1",
				Title:              "Root task",
				AcceptanceCriteria: []string{"Root done"},
			},
			{
				ID:                 "wi-2",
				Title:              "Branch Left",
				Dependencies:       []string{"wi-1"},
				AcceptanceCriteria: []string{"Left done"},
			},
			{
				ID:                 "wi-3",
				Title:              "Branch Right",
				Dependencies:       []string{"wi-1"},
				AcceptanceCriteria: []string{"Right done"},
			},
			{
				ID:                 "wi-4",
				Title:              "Join task",
				Dependencies:       []string{"wi-2", "wi-3"},
				AcceptanceCriteria: []string{"Join done"},
			},
		},
	}
	if err := pDiamond.Validate(); err != nil {
		t.Fatalf("unexpected validation error on valid diamond DAG: %v", err)
	}
}

func TestSolutionExploration_TradeOffs_And_Selection(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)

	svc := problemService.NewService(problemRepo, planRepo, projRepo, nil)

	proj := &domainProject.Project{
		ID:         "proj-audit-exp",
		Name:       "專案方案探索",
		Workspace:  "/workspace/test",
		DomainType: domainProject.DomainCoding,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	prob, err := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "改善資料庫寫入效能瓶頸",
		Objective: "降低併發交易鎖衝突",
	})
	if err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	// 1. Validation test: empty alternatives
	badExp := &domainPlan.SolutionExplorationResult{
		ProblemID:    prob.ID,
		Alternatives: []domainPlan.SolutionAlternative{},
	}
	if err := domainPlan.ValidateSolutionExploration(badExp); err == nil {
		t.Fatalf("expected error on empty alternatives, got nil")
	}

	// 2. Multi-alternative trade-off exploration (AIA-22 §4.6)
	exp := &domainPlan.SolutionExplorationResult{
		ProblemID: prob.ID,
		Alternatives: []domainPlan.SolutionAlternative{
			{
				ID:                   "alt-1-in-memory-queue",
				Title:                "引入記憶體隊列緩衝",
				Description:          "以 channel 與 worker pool 批量匯總寫入",
				TechnicalFeasibility: "HIGH",
				Reversibility:        "HIGH",
				EstimatedCost:        "2 天工程時間",
				EstimatedDuration:    "2 天",
				ExpectedBenefits:     []string{"吞吐量提升 300%", "零架構侵入"},
			},
			{
				ID:                   "alt-2-switch-to-postgres",
				Title:                "全面替換為分散式資料庫",
				Description:          "改寫 SQL 存取層遷移至外部獨立資料庫",
				TechnicalFeasibility: "LOW",
				Reversibility:        "IRREVERSIBLE", // Irreversible action!
				EstimatedCost:        "4 週重構時間",
				EstimatedDuration:    "1 個月",
				Risks:                []string{"破壞單機自包含性", "大量既有測試需重寫"},
			},
		},
		SelectedAlternativeID: "alt-1-in-memory-queue",
		OverallRationale:      "方案 1 完全保留單機自包含優勢，且具備高可逆性與低實施成本",
	}

	rec, err := svc.ExploreSolutions(ctx, prob.ID, exp, "coding-agent")
	if err != nil {
		t.Fatalf("failed to explore solutions: %v", err)
	}

	if rec.AnalysisType != "SOLUTION_EXPLORATION" {
		t.Fatalf("unexpected record analysis type: %s", rec.AnalysisType)
	}
	if len(rec.Findings) != 2 {
		t.Fatalf("expected 2 findings for 2 alternatives, got %d", len(rec.Findings))
	}
}

func TestPlanGeneration_Supersedes_And_ChangeHistory(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)

	svc := problemService.NewService(problemRepo, planRepo, projRepo, nil)

	proj := &domainProject.Project{
		ID:         "proj-plan-gen",
		Name:       "計畫生成測試",
		Workspace:  "/workspace/plan",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	prob, _ := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "撰寫徒步環島計畫指南",
		Objective: "產出路線、裝備與預算完整指南",
	})

	// 1. Generate Plan Version 1 (AIA-22 §4.7)
	plan1, err := svc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID: prob.ID,
		Objective: "完成初步章節規劃與第一卷大綱",
		Strategy:  "依地理分區由北至南逐步規劃",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-north",
				Title:              "北部路線規劃（台北至新竹）",
				Stage:              "Stage1",
				AssignedAgent:      "writing-agent",
				AcceptanceCriteria: []string{"每日里程落於 20~25 公里", "列出主要省道與替代步道"},
			},
			{
				ID:                 "wi-central",
				Title:              "中部路線規劃（苗栗至雲林）",
				Stage:              "Stage2",
				AssignedAgent:      "writing-agent",
				Dependencies:       []string{"wi-north"},
				AcceptanceCriteria: []string{"補給點間距小於 10 公里", "標記夏季高溫遮蔭段落"},
			},
		},
		AcceptanceCriteria: []string{"兩卷路線規劃通過一致性檢核"},
	})
	if err != nil {
		t.Fatalf("failed to generate plan 1: %v", err)
	}

	if plan1.Version != 1 || plan1.Status != domainPlan.StatusProposed {
		t.Fatalf("unexpected plan 1: version=%d, status=%s", plan1.Version, plan1.Status)
	}

	// Verify problem updated
	probRel1, _ := svc.GetProblem(ctx, prob.ID)
	if probRel1.CurrentPlanVersion != 1 || probRel1.Status != domainProblem.StatusPlanned {
		t.Fatalf("problem not updated with plan 1: ver=%d, status=%s", probRel1.CurrentPlanVersion, probRel1.Status)
	}

	// 2. Generate Plan Version 2 (Superseding Plan 1 based on human feedback) (AIA-22 §6.5)
	plan2, err := svc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID:        prob.ID,
		Objective:        "擴大為全島四區並行大綱規劃",
		Strategy:         "東南西北四區平行切入以利冬季東北季風氣候避讓",
		SupersedesPlanID: plan1.ID,
		ChangeReason:     "使用者希望優先考量冬季東北季風順風徒步（順時針逆風過大，改採逆時針順風推進）",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-counter-clockwise",
				Title:              "逆時針順風路線總體規劃",
				Stage:              "StrategyRevision",
				AssignedAgent:      "writing-agent",
				AcceptanceCriteria: []string{"冬季新竹段與蘇花段順風行走", "每日落腳點避開偏遠無宿區"},
			},
		},
		AcceptanceCriteria: []string{"全島逆時針順風總大綱通過"},
	})
	if err != nil {
		t.Fatalf("failed to generate plan 2: %v", err)
	}

	if plan2.Version != 2 || plan2.SupersedesPlanID != plan1.ID {
		t.Fatalf("unexpected plan 2: ver=%d, supersedes=%s", plan2.Version, plan2.SupersedesPlanID)
	}

	// 3. Verify Plan 1 is marked as SUPERSEDED
	p1Reload, _ := svc.GetPlan(ctx, plan1.ID)
	if p1Reload.Status != domainPlan.StatusSuperseded {
		t.Fatalf("expected plan 1 to be SUPERSEDED, got %s", p1Reload.Status)
	}

	// 4. Verify PlanChangeRecord created and queryable
	changes, err := svc.ListPlanChanges(ctx, plan2.ID)
	if err != nil || len(changes) != 1 {
		t.Fatalf("expected 1 change record for plan 2, got %d (err: %v)", len(changes), err)
	}
	if changes[0].FromVersion != 1 || changes[0].ToVersion != 2 {
		t.Fatalf("unexpected change record version mapping: %+v", changes[0])
	}
}

func TestPlanMaterialization_WorkItemToTask(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	taskSvc := taskService.NewService(taskRepo, nil)
	agRepo := repository.NewAgentRepository(db)
	_ = agRepo.Upsert(ctx, &domainAgent.Agent{
		ID:      "writing-agent",
		Name:    "Writing Agent",
		Domain:  "writing",
		Enabled: true,
	})

	svc := problemService.NewService(problemRepo, planRepo, projRepo, nil)
	svc.SetTaskService(taskSvc)

	proj := &domainProject.Project{
		ID:         "proj-mat-test",
		Name:       "工單具體化專案",
		Workspace:  "/workspace/mat",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	prob, _ := svc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "執行分批小說章節合規性校對",
		Objective: "產出具體驗證工單",
	})

	// Generate Plan with 3 dependent work items
	plan, err := svc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID: prob.ID,
		Objective: "分批校對前三卷",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-vol1",
				Title:              "校對第 1 卷（第 1~50 章）",
				Stage:              "Batch1",
				AssignedAgent:      "writing-agent",
				AcceptanceCriteria: []string{"人物稱謂正確率 100%", "標點格式無異常"},
			},
			{
				ID:                 "wi-vol2",
				Title:              "校對第 2 卷（第 51~100 章）",
				Stage:              "Batch2",
				AssignedAgent:      "writing-agent",
				Dependencies:       []string{"wi-vol1"}, // depends on vol1
				AcceptanceCriteria: []string{"承接第 1 卷伏筆無矛盾"},
			},
		},
		AcceptanceCriteria: []string{"兩卷校對完成"},
	})
	if err != nil {
		t.Fatalf("failed to generate plan: %v", err)
	}

	// 1. Approve Plan
	approvedPlan, err := svc.ApprovePlan(ctx, plan.ID, "chief-editor")
	if err != nil || approvedPlan.Status != domainPlan.StatusApproved {
		t.Fatalf("failed to approve plan: %v", err)
	}

	// 2. Materialize All Work Items into Tasks (AIA-22 §4.7)
	tasks, err := svc.MaterializeAllWorkItems(ctx, plan.ID)
	if err != nil {
		t.Fatalf("failed to materialize work items: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 materialized tasks, got %d", len(tasks))
	}

	t1 := tasks[0]
	t2 := tasks[1]

	// Verify Task 1 has no dependencies and is READY
	if t1.Title != "校對第 1 卷（第 1~50 章）" || t1.Status != domainTask.StatusReady {
		t.Fatalf("unexpected Task 1 state: status=%s, title=%s", t1.Status, t1.Title)
	}

	// Verify Task 2 depends on Task 1 and is WAITING_DEPENDENCY
	if t2.Title != "校對第 2 卷（第 51~100 章）" || t2.Status != domainTask.StatusWaitingDependency {
		t.Fatalf("unexpected Task 2 state: status=%s, title=%s", t2.Status, t2.Title)
	}
	t2Deps, err := taskSvc.GetDependencies(ctx, t2.ID)
	if err != nil || len(t2Deps) != 1 || string(t2Deps[0]) != t1.ID {
		t.Fatalf("task 2 dependency chain broken: expected dependency on %s, got %+v (err: %v)", t1.ID, t2Deps, err)
	}

	// Verify plan status is now ACTIVE and problem status is EXECUTING
	planReloaded, _ := svc.GetPlan(ctx, plan.ID)
	if planReloaded.Status != domainPlan.StatusActive {
		t.Fatalf("expected plan status ACTIVE, got %s", planReloaded.Status)
	}
	probReloaded, _ := svc.GetProblem(ctx, prob.ID)
	if probReloaded.Status != domainProblem.StatusExecuting {
		t.Fatalf("expected problem status EXECUTING, got %s", probReloaded.Status)
	}
}

func TestPlanWebConsole_Endpoints(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	taskSvc := taskService.NewService(taskRepo, nil)
	agRepo := repository.NewAgentRepository(db)
	_ = agRepo.Upsert(ctx, &domainAgent.Agent{
		ID:      "personal-agent",
		Name:    "Personal Agent",
		Domain:  "orchestrator",
		Enabled: true,
	})

	proj := &domainProject.Project{
		ID:         "proj-web-plan",
		Name:       "Web Plan Test Project",
		Workspace:  "/workspace/test",
		DomainType: domainProject.DomainGeneral,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	probSvc := problemService.NewService(problemRepo, planRepo, projRepo, nil)
	probSvc.SetTaskService(taskSvc)

	prob, _ := probSvc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "API 端點計畫測試問題",
		Objective: "測試計畫 API",
	})

	mux := http.NewServeMux()
	console.RegisterConsoleRoutes(mux, &console.ConsoleServices{
		DB:         db,
		ProblemSvc: probSvc,
		TaskSvc:    taskSvc,
	})

	// 1. POST /api/v1/problems/{id}/solutions
	solPayload := domainPlan.SolutionExplorationResult{
		Alternatives: []domainPlan.SolutionAlternative{
			{
				ID:                   "alt-1",
				Title:                "推薦方案",
				TechnicalFeasibility: "HIGH",
				Reversibility:        "HIGH",
			},
		},
		SelectedAlternativeID: "alt-1",
		OverallRationale:      "最佳效益比",
	}
	solBytes, _ := json.Marshal(solPayload)
	reqSol := httptest.NewRequest(http.MethodPost, "/api/v1/problems/"+prob.ID+"/solutions", bytes.NewReader(solBytes))
	reqSol.Header.Set("Content-Type", "application/json")
	wSol := httptest.NewRecorder()
	mux.ServeHTTP(wSol, reqSol)

	if wSol.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for solutions, got %d (body: %s)", wSol.Code, wSol.Body.String())
	}

	// 2. POST /api/v1/problems/{id}/plans (Generate Plan)
	planPayload := problemService.GeneratePlanInput{
		Objective: "第一階段執行計畫",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-web-1",
				Title:              "初始化工單 1",
				AcceptanceCriteria: []string{"通過基本驗收"},
			},
		},
	}
	planBytes, _ := json.Marshal(planPayload)
	reqPlan := httptest.NewRequest(http.MethodPost, "/api/v1/problems/"+prob.ID+"/plans", bytes.NewReader(planBytes))
	reqPlan.Header.Set("Content-Type", "application/json")
	wPlan := httptest.NewRecorder()
	mux.ServeHTTP(wPlan, reqPlan)

	if wPlan.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for plan, got %d (body: %s)", wPlan.Code, wPlan.Body.String())
	}
	var createdPlanResp struct {
		Success bool            `json:"success"`
		Data    domainPlan.Plan `json:"data"`
	}
	_ = json.Unmarshal(wPlan.Body.Bytes(), &createdPlanResp)
	createdPlan := createdPlanResp.Data
	if createdPlan.ID == "" || createdPlan.Version != 1 {
		t.Fatalf("unexpected created plan: %+v", createdPlan)
	}

	// 3. GET /api/v1/problems/{id}/plan (Get Latest Plan)
	reqLatest := httptest.NewRequest(http.MethodGet, "/api/v1/problems/"+prob.ID+"/plan", nil)
	wLatest := httptest.NewRecorder()
	mux.ServeHTTP(wLatest, reqLatest)

	if wLatest.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for latest plan, got %d", wLatest.Code)
	}

	// 4. POST /api/v1/plans/{id}/approve
	reqApprove := httptest.NewRequest(http.MethodPost, "/api/v1/plans/"+createdPlan.ID+"/approve", nil)
	wApprove := httptest.NewRecorder()
	mux.ServeHTTP(wApprove, reqApprove)

	if wApprove.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for approve plan, got %d", wApprove.Code)
	}

	// 5. POST /api/v1/plans/{id}/materialize
	reqMat := httptest.NewRequest(http.MethodPost, "/api/v1/plans/"+createdPlan.ID+"/materialize", nil)
	wMat := httptest.NewRecorder()
	mux.ServeHTTP(wMat, reqMat)

	if wMat.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for materialize, got %d (body: %s)", wMat.Code, wMat.Body.String())
	}
	var matResp struct {
		Success bool               `json:"success"`
		Data    []*domainTask.Task `json:"data"`
	}
	_ = json.Unmarshal(wMat.Body.Bytes(), &matResp)
	if len(matResp.Data) != 1 || matResp.Data[0].Title != "初始化工單 1" {
		t.Fatalf("unexpected materialized task: %+v", matResp.Data)
	}
}
