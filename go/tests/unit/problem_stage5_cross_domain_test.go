package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	domainAgent "github.com/wade/aia/internal/domain/agent"
	domainPlan "github.com/wade/aia/internal/domain/plan"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	problemService "github.com/wade/aia/internal/service/problem"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// setupCrossDomainTestEnvironment prepares a clean SQLite database with domain agents and project/problem/plan/task services.
func setupCrossDomainTestEnvironment(t *testing.T) (
	*repository.ProjectRepository,
	*problemService.Service,
	*taskService.Service,
) {
	ctx := context.Background()
	db := setupTestDB(t)

	projRepo := repository.NewProjectRepository(db)
	probRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	agRepo := repository.NewAgentRepository(db)

	taskSvc := taskService.NewService(taskRepo, auditRepo)
	probSvc := problemService.NewService(probRepo, planRepo, projRepo, nil)
	probSvc.SetAuditRepo(auditRepo)
	probSvc.SetTaskService(taskSvc)

	// Seed domain agents
	_ = agRepo.Upsert(ctx, &domainAgent.Agent{
		ID:      "writing-agent",
		Name:    "Writing Specialist Agent",
		Domain:  "writing",
		Enabled: true,
	})
	_ = agRepo.Upsert(ctx, &domainAgent.Agent{
		ID:      "coding-agent",
		Name:    "Software Engineering Agent",
		Domain:  "coding",
		Enabled: true,
	})
	_ = agRepo.Upsert(ctx, &domainAgent.Agent{
		ID:      "investment-agent",
		Name:    "Quantitative Investment Agent",
		Domain:  "investment",
		Enabled: true,
	})
	_ = agRepo.Upsert(ctx, &domainAgent.Agent{
		ID:      "personal-agent",
		Name:    "Personal Assistant Agent",
		Domain:  "personal",
		Enabled: true,
	})

	return projRepo, probSvc, taskSvc
}

// TestStage5_WritingDomain_540ChaptersNovelAudit tests Scenario 1 from AIA-22 §11.1 & §15.5:
// - Whole-book analysis of 540 chapters
// - Discovery of conflicting requirements between volume rules vs draft chapters
// - Multi-volume recursive decomposition
// - Solution exploration and plan generation with acceptance criteria
// - Materialization to writing-agent
// - Dynamic reassessment upon new evidence invalidating initial conflict assumption
func TestStage5_WritingDomain_540ChaptersNovelAudit(t *testing.T) {
	ctx := context.Background()
	projRepo, probSvc, _ := setupCrossDomainTestEnvironment(t)

	// 1. Create Writing Project
	proj := &domainProject.Project{
		ID:         "proj-novel-540",
		Name:       "540 章修仙長篇小說《問鼎仙途》全書合規與伏筆審計",
		Workspace:  "/workspace/novel-540",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("failed to create writing project: %v", err)
	}

	// 2. Create Open-ended Problem (AIA-22 §16.2 Test A)
	prob, err := probSvc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "540 章完整小說全書合規與世界觀設定稽核",
		Objective: "全面稽核 540 章小說之伏筆、境界體系、人物存活設定，解決前後卷設定衝突",
		Scope:     "全書 1~540 章全文共 9 卷與世界觀設定文件",
	})
	if err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	// 3. Apply Domain Analysis (AIA-22 §4.3, §4.4)
	analysis := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{
			UserGoal:    "全面稽核 540 章小說之伏筆與前後卷世界觀設定是否衝突",
			Constraints: []string{"不得擅自修改未經作者確認的主要情節"},
		},
		Context: domainProblem.ContextSummary{
			FoundResources: []string{"db://chapters/novel_540", "docs/bible_v1.md"},
		},
		GapsAndConflicts: []domainProblem.GapConflictItem{
			{
				ID:                    "gap-conflict-1",
				Category:              domainProblem.GapRuleConflict,
				Description:           "世界觀設定文檔與第 420 章描寫存在潛在規範衝突",
				Source:                "bible_v1.md vs chapter_420.txt",
				Severity:              "HIGH",
				SuggestedHandling:     domainProblem.HandlingClarify,
				RequiresClarification: true,
			},
			{
				ID:                "gap-data-2",
				Category:          domainProblem.GapDataGap,
				Description:       "海外仙島篇缺少勢力譜系與境界換算表",
				Severity:          "MEDIUM",
				SuggestedHandling: domainProblem.HandlingResearch,
			},
		},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "fact-vol-count",
				Category:   domainProblem.CategoryKnownFact,
				Statement:  "全書共 540 章已完稿，共計 185 萬字，分為 9 卷",
				Confidence: 1.0,
				Source:     "db://chapters/novel_540/summary",
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "writing-agent",
				CreatedAt:  time.Now().UTC(),
			},
			{
				ID:         "fact-bible-lore",
				Category:   domainProblem.CategoryKnownFact,
				Statement:  "世界觀設定文檔 (bible_v1.md) 記載主角為天生絕脈無法修習陽性功法",
				Confidence: 1.0,
				Source:     "file:///workspace/novel-540/docs/bible_v1.md#L45",
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "writing-agent",
				CreatedAt:  time.Now().UTC(),
			},
			{
				ID:         "assump-ch420-conflict",
				Category:   domainProblem.CategoryAssumption,
				Statement:  "第 420 章戰鬥描寫主角施展九陽神雷，可能與天生絕脈設定產生硬性衝突",
				Confidence: 0.75,
				Source:     "chapter_420_review",
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "writing-agent",
				CreatedAt:  time.Now().UTC(),
			},
			{
				ID:         "unknown-foreign-isle",
				Category:   domainProblem.CategoryUnknown,
				Statement:  "第 300~350 章海外仙島篇之宗門戰力換算是否已有隱性修訂文檔",
				Confidence: 0.0,
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "writing-agent",
				CreatedAt:  time.Now().UTC(),
			},
		},
	}

	model1, err := probSvc.ApplyAnalysis(ctx, prob.ID, analysis, "writing-agent", "exec-writing-1")
	if err != nil {
		t.Fatalf("failed to apply writing analysis: %v", err)
	}
	if model1.Version != 1 {
		t.Fatalf("expected model version 1, got %d", model1.Version)
	}

	// 4. Recursive Decomposition into multi-volume subproblems (AIA-22 §4.5, §15.5)
	createdSubProblems, err := probSvc.DecomposeProblem(ctx, prob.ID, []problemService.CreateProblemInput{
		{
			Title:     "第 1~3 卷（第 1~180 章）凡人篇設定與伏筆稽核",
			Objective: "審查主角起步設定、絕脈伏筆與初入修真界人物關係網",
			Scope:     "第 1~180 章",
		},
		{
			Title:     "第 4~6 卷（第 181~360 章）宗門篇與海外仙島篇衝突校驗",
			Objective: "稽核海外仙島篇境界換算與宗門戰役邏輯",
			Scope:     "第 181~360 章",
		},
		{
			Title:     "第 7~9 卷（第 361~540 章）飛升篇大結局邏輯閉環稽核",
			Objective: "驗證第 420 章陽雷施展與大結局伏筆回收是否自洽",
			Scope:     "第 361~540 章",
		},
	})
	if err != nil {
		t.Fatalf("failed to decompose problem: %v", err)
	}
	if len(createdSubProblems) != 3 {
		t.Fatalf("expected 3 subproblems, got %d", len(createdSubProblems))
	}

	// Verify problem hierarchy tree
	tree, err := probSvc.GetProblemTree(ctx, prob.ID)
	if err != nil || len(tree.Children) != 3 {
		t.Fatalf("expected tree with 3 children, got %d (err: %v)", len(tree.Children), err)
	}

	// 5. Solution Exploration (AIA-22 §4.6)
	expRes, err := probSvc.ExploreSolutions(ctx, prob.ID, &domainPlan.SolutionExplorationResult{
		ProblemID: prob.ID,
		Alternatives: []domainPlan.SolutionAlternative{
			{
				ID:                   "alt-regex-scan",
				Title:                "正則表達式全域關鍵字快篩",
				Description:          "以 regex 快速掃描 540 章之境界與心法關鍵詞",
				TechnicalFeasibility: "HIGH",
				Reversibility:        "HIGH",
				EstimatedCost:        "1 小時",
				Risks:                []string{"無法理解上下文語義與伏筆反轉"},
			},
			{
				ID:                   "alt-semantic-volume-audit",
				Title:                "分卷語義審查 + 境界演進圖比對",
				Description:          "按卷逐步比對伏筆鏈與境界演變，標註矛盾點與潤修建議",
				TechnicalFeasibility: "HIGH",
				Reversibility:        "HIGH",
				EstimatedCost:        "3 天",
				ExpectedBenefits:     []string{"徹底釐清伏筆與反轉", "產出精確修改清單"},
			},
		},
		SelectedAlternativeID: "alt-semantic-volume-audit",
		OverallRationale:      "正則快篩無法辨識語義反轉（如幻境或替身），必須採用分卷語義審查",
	}, "writing-agent")
	if err != nil || expRes == nil {
		t.Fatalf("failed to explore solutions: %v", err)
	}

	// 6. Plan Generation with explicit acceptance criteria (AIA-22 §4.7)
	plan1, err := probSvc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID: prob.ID,
		Objective: "執行三階段全書語義稽核與衝突修訂",
		Strategy:  "先梳理凡人篇基準，再補齊海外篇譜系，最後攻堅第 420 章前後反轉",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-vol1-3",
				Title:              "凡人篇（1~180 章）伏筆鏈梳理",
				Stage:              "Stage1",
				AssignedAgent:      "writing-agent",
				AcceptanceCriteria: []string{"產出天生絕脈與初期伏筆對照表", "所有角色初登場境界標註完畢"},
			},
			{
				ID:                 "wi-vol4-6",
				Title:              "海外仙島篇（181~360 章）勢力譜系補齊",
				Stage:              "Stage2",
				AssignedAgent:      "writing-agent",
				Dependencies:       []string{"wi-vol1-3"},
				AcceptanceCriteria: []string{"補齊仙島十二派勢力譜系文檔", "海外境界與中原境界換算率標註"},
			},
			{
				ID:                 "wi-vol7-9",
				Title:              "飛升篇（361~540 章）第 420 章衝突實質查證",
				Stage:              "Stage3",
				AssignedAgent:      "writing-agent",
				Dependencies:       []string{"wi-vol4-6"},
				AcceptanceCriteria: []string{"調閱 420~425 章正文確認主角心法來源", "若真衝突則提出改寫方案"},
			},
		},
		AcceptanceCriteria: []string{"全書 9 卷完成伏筆與境界一致性校驗"},
	})
	if err != nil {
		t.Fatalf("failed to generate plan 1: %v", err)
	}

	// 7. Human Governance Approval & Materialization (AIA-22 §9.2, §4.7)
	approvedPlan, err := probSvc.ApprovePlan(ctx, plan1.ID, "chief-editor")
	if err != nil || approvedPlan.Status != domainPlan.StatusApproved {
		t.Fatalf("failed to approve plan: %v", err)
	}

	tasks, err := probSvc.MaterializeAllWorkItems(ctx, plan1.ID)
	if err != nil || len(tasks) != 3 {
		t.Fatalf("expected 3 tasks materialized, got %d (err: %v)", len(tasks), err)
	}

	// Verify domain agent assignment
	if string(tasks[0].AgentID) != "writing-agent" {
		t.Fatalf("expected assigned agent 'writing-agent', got %s", tasks[0].AgentID)
	}

	// 8. Unexpected Evidence Simulation & Reassessment (AIA-22 §8.1, §8.2, §16.2 Test D/E)
	// Finding during execution of wi-vol7-9:
	// Chapter 422 reveals Chapter 420 was an illusion created by a Heart-Demon Tribulation!
	// Therefore, the assumption "assump-ch420-conflict" is CONTRADICTED.
	reassessReq := domainProblem.ReassessmentRequest{
		ProblemID:              prob.ID,
		Trigger:                domainProblem.TriggerHypothesisContradicted,
		TriggerEvidence:        "第 422 章正文明確揭曉第 420 章雷法為心魔幻境試煉，主角肉身並未修習陽性功法",
		RootCause:              domainProblem.RootCauseAssumptionInvalid,
		ContradictedHypotheses: []string{"assump-ch420-conflict"},
		NewFindings: []string{
			"證實第 420 章為心魔幻境，原衝突假設不成立；將第 3 階段修改為強化幻境破綻伏筆而非大篇幅重寫",
		},
		SuggestedAction: domainProblem.ActionModifyPlan,
		Rationale:       "心魔幻境成立，衝突實為情節伏筆，調整計畫修改範圍",
		AgentID:         "writing-agent",
	}

	reassessResult, err := probSvc.TriggerReassessment(ctx, reassessReq)
	if err != nil {
		t.Fatalf("failed to trigger reassessment: %v", err)
	}

	if reassessResult.NewModelVer != 2 {
		t.Fatalf("expected new model version 2, got %d", reassessResult.NewModelVer)
	}

	// Verify hypothesis was marked Contradicted
	modelV2, err := probSvc.GetLatestModel(ctx, prob.ID)
	if err != nil {
		t.Fatalf("failed to reload model v2: %v", err)
	}
	foundContradicted := false
	for _, h := range modelV2.Assumptions {
		if h.ID == "assump-ch420-conflict" && h.Status == domainProblem.ItemStatusContradicted {
			foundContradicted = true
			break
		}
	}
	if !foundContradicted {
		t.Fatalf("expected hypothesis assump-ch420-conflict to be marked contradicted in model v2")
	}

	// 9. Generate Plan v2 superseding Plan v1 (AIA-22 §6.5)
	plan2, err := probSvc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID:        prob.ID,
		Objective:        "修訂為幻境伏筆強化與全書收尾校核",
		Strategy:         "無需大改第 420 章，僅需在第 418 章增添心魔幻境暗示伏筆",
		SupersedesPlanID: plan1.ID,
		ChangeReason:     "第 422 章查證確認為幻境試煉，免去大篇幅重寫，轉為精修伏筆暗示",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-vol7-9-revised",
				Title:              "精修第 418~422 章心魔幻境過渡伏筆",
				Stage:              "Stage3-Revised",
				AssignedAgent:      "writing-agent",
				AcceptanceCriteria: []string{"第 418 章增添幻境微小破綻描寫", "第 422 章破境銜接順暢"},
			},
		},
		AcceptanceCriteria: []string{"幻境伏筆修訂完畢且全書設定自洽"},
	})
	if err != nil {
		t.Fatalf("failed to generate plan 2: %v", err)
	}

	// Verify Plan 1 is SUPERSEDED and Plan 2 is version 2
	p1Reload, _ := probSvc.GetPlan(ctx, plan1.ID)
	if p1Reload.Status != domainPlan.StatusSuperseded {
		t.Fatalf("expected plan 1 status SUPERSEDED, got %s", p1Reload.Status)
	}
	if plan2.Version != 2 || plan2.SupersedesPlanID != plan1.ID {
		t.Fatalf("unexpected plan 2 metadata: ver=%d, supersedes=%s", plan2.Version, plan2.SupersedesPlanID)
	}

	// Verify change log
	changes, err := probSvc.ListPlanChanges(ctx, plan2.ID)
	if err != nil || len(changes) != 1 {
		t.Fatalf("expected 1 change record, got %d (err: %v)", len(changes), err)
	}
	if changes[0].FromVersion != 1 || changes[0].ToVersion != 2 {
		t.Fatalf("unexpected change record version mapping: %+v", changes[0])
	}
}

// TestStage5_CodingDomain_UnfamiliarGoRepoTechDebt tests Scenario 2 from AIA-22 §11.2 & §15.5:
// - Technical debt and architecture investigation of an unfamiliar Go codebase
// - Uncovering hidden dependencies and missing benchmarks
// - Hypothesis testing (HTTP body leak vs DB lock contention)
// - Plan generation with measurable engineering acceptance criteria
// - Materialization to coding-agent
// - Reassessment upon benchmark evidence refuting initial bottleneck assumption
func TestStage5_CodingDomain_UnfamiliarGoRepoTechDebt(t *testing.T) {
	ctx := context.Background()
	projRepo, probSvc, _ := setupCrossDomainTestEnvironment(t)

	// 1. Create Coding Project
	proj := &domainProject.Project{
		ID:         "proj-go-debt",
		Name:       "陌生 Go 微服務架構瓶頸與 Tech-Debt 分析",
		Workspace:  "/workspace/go-service",
		DomainType: domainProject.DomainCoding,
		Status:     domainProject.StatusActive,
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("failed to create coding project: %v", err)
	}

	// 2. Create Open-ended Problem
	prob, err := probSvc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "分析陌生 Go 微服務 Repository 架構瓶頸與 Tech-Debt",
		Objective: "自主分析系統高負載下延遲驟增之架構瓶頸，產出可驗證重構計畫",
		Scope:     "48 個 Go Package 與核心資料存取層",
	})
	if err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	// 3. Domain Analysis with Facts, Assumptions, and Unknowns (AIA-22 §4.3)
	analysis := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{
			UserGoal:        "排查高負載延遲驟增之系統瓶頸",
			SuccessCriteria: []string{"高負載下 p99 延遲降至合理範圍"},
		},
		Context: domainProblem.ContextSummary{
			FoundResources:   []string{"file:///workspace/go-service/go.mod"},
			MissingResources: []string{"tests/benchmark"},
		},
		GapsAndConflicts: []domainProblem.GapConflictItem{
			{
				ID:                "gap-bench-missing",
				Category:          domainProblem.GapDataGap,
				Description:       "缺少性能基準測試（Benchmark），無法客觀衡量優化成效",
				Source:            "tests/ directory lacks bench_test.go",
				Severity:          "HIGH",
				SuggestedHandling: domainProblem.HandlingResearch,
			},
		},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "fact-pkg-count",
				Category:   domainProblem.CategoryKnownFact,
				Statement:  "專案包含 48 個 Go Package，核心依賴 SQLite 與 net/http",
				Confidence: 1.0,
				Source:     "file:///workspace/go-service/go.mod#L1-L25",
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "coding-agent",
				CreatedAt:  time.Now().UTC(),
			},
			{
				ID:         "assump-http-leak",
				Category:   domainProblem.CategoryAssumption,
				Statement:  "高負載延遲主要源於 HTTP Client 未重用連線與 Response Body 洩漏",
				Confidence: 0.65,
				Source:     "code_inspection_http",
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "coding-agent",
				CreatedAt:  time.Now().UTC(),
			},
			{
				ID:         "unknown-pprof-profile",
				Category:   domainProblem.CategoryUnknown,
				Statement:  "500 QPS 負載下的真實 CPU、Heap 與 Goroutine 堆疊分佈 (pprof)",
				Confidence: 0.0,
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "coding-agent",
				CreatedAt:  time.Now().UTC(),
			},
		},
	}

	_, err = probSvc.ApplyAnalysis(ctx, prob.ID, analysis, "coding-agent", "exec-coding-1")
	if err != nil {
		t.Fatalf("failed to apply coding analysis: %v", err)
	}

	// 4. Generate Plan with quantifiable engineering acceptance criteria (AIA-22 §4.7)
	plan1, err := probSvc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID: prob.ID,
		Objective: "建立性能基準測試並驗證 HTTP 連線洩漏假說",
		Strategy:  "先建立可重現之基準負載測試，再採樣 pprof 火焰圖驗證瓶頸位置",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-bench-suite",
				Title:              "構建 500 QPS 基準壓測腳本與 pprof 採樣套件",
				Stage:              "BenchmarkBaseline",
				AssignedAgent:      "coding-agent",
				AcceptanceCriteria: []string{"可重複重現 500 QPS 負載", "自動採樣 cpu.pprof 與 heap.pprof"},
			},
			{
				ID:                 "wi-profile-analysis",
				Title:              "分析 pprof 採樣火焰圖並確認 HTTP Client 洩漏程度",
				Stage:              "Profiling",
				AssignedAgent:      "coding-agent",
				Dependencies:       []string{"wi-bench-suite"},
				AcceptanceCriteria: []string{"產出瓶頸熱點函數佔比報告", "堆疊記憶體每秒分配量量化"},
			},
		},
		AcceptanceCriteria: []string{"確認真實系統瓶頸位置並產出修復驗證指標"},
	})
	if err != nil {
		t.Fatalf("failed to generate plan 1: %v", err)
	}

	// Approve & Materialize
	_, err = probSvc.ApprovePlan(ctx, plan1.ID, "tech-lead")
	if err != nil {
		t.Fatalf("failed to approve plan: %v", err)
	}
	tasks, err := probSvc.MaterializeAllWorkItems(ctx, plan1.ID)
	if err != nil || len(tasks) != 2 {
		t.Fatalf("failed to materialize coding tasks: %v", err)
	}
	if string(tasks[0].AgentID) != "coding-agent" {
		t.Fatalf("expected coding-agent, got %s", tasks[0].AgentID)
	}

	// 5. Benchmark Execution Feedback Refuting Assumption (AIA-22 §8.1, §8.3 Root Cause Attribution)
	// Evidence shows HTTP Client is fine; the real bottleneck is SQLite lock contention in WAL mode!
	reassessReq := domainProblem.ReassessmentRequest{
		ProblemID:              prob.ID,
		Trigger:                domainProblem.TriggerTaskFailure,
		TriggerEvidence:        "pprof 火焰圖顯示 78% 的等待時間在 database/sql.(*DB).connLock 與 sqlite3_step 互斥鎖，HTTP 僅佔 3%",
		RootCause:              domainProblem.RootCauseModelError,
		ContradictedHypotheses: []string{"assump-http-leak"},
		NewFindings: []string{
			"初始假說歸因錯誤；瓶頸並非 HTTP 連線未釋放，而是 SQLite 寫入鎖競爭",
		},
		SuggestedAction: domainProblem.ActionRebuildPlan,
		Rationale:       "初始問題模型歸因錯誤（MODEL_ERROR），真正瓶頸為 SQLite 寫入鎖競爭，必須重構資料庫存取層",
		AgentID:         "coding-agent",
	}

	reassessRes, err := probSvc.TriggerReassessment(ctx, reassessReq)
	if err != nil || reassessRes.NewModelVer != 2 {
		t.Fatalf("failed to trigger coding reassessment: %v", err)
	}

	// 6. Generate Revised Plan v2 with targeted SQLite architecture fixes
	plan2, err := probSvc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID:        prob.ID,
		Objective:        "解決 SQLite 寫入鎖競爭：引入記憶體寫入隊列與批次提交",
		Strategy:         "保留單機 SQLite 優勢，透過 Go Channel 緩衝寫入請求並以單一 worker 批量事務提交",
		SupersedesPlanID: plan1.ID,
		ChangeReason:     "pprof 實測證實瓶頸為 SQLite 寫入鎖競爭而非 HTTP 連線",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-sqlite-batch-writer",
				Title:              "實作非同步批次寫入隊列 (Batch Commit Buffer)",
				Stage:              "StorageRefactor",
				AssignedAgent:      "coding-agent",
				AcceptanceCriteria: []string{"在 1000 QPS 下 p99 延遲降至 50ms 以下", "無 database is locked (5) 錯誤"},
			},
		},
		AcceptanceCriteria: []string{"壓測 1000 QPS 持續 5 分鐘零錯誤且延遲達標"},
	})
	if err != nil || plan2.Version != 2 {
		t.Fatalf("failed to generate plan 2: %v", err)
	}
}

// TestStage5_InvestmentDomain_PortfolioContinuousManagement tests Scenario 3 from AIA-22 §11.3 & §15.5:
// - Quantitative investment portfolio autonomous management
// - Risk parity solution exploration
// - Human governance for irreversible/financial decisions (AIA-22 §9.2)
// - Materialization to investment-agent
// - Market drift volatility shock triggering dynamic reassessment without overwriting history
func TestStage5_InvestmentDomain_PortfolioContinuousManagement(t *testing.T) {
	ctx := context.Background()
	projRepo, probSvc, _ := setupCrossDomainTestEnvironment(t)

	// 1. Create Investment Project
	proj := &domainProject.Project{
		ID:         "proj-portfolio-100k",
		Name:       "100k 虛擬多資產組合動態風險平價管理",
		Workspace:  "/workspace/investment",
		DomainType: domainProject.DomainInvestment,
		Status:     domainProject.StatusActive,
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("failed to create investment project: %v", err)
	}

	// 2. Create Open-ended Problem
	prob, err := probSvc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "建立並持續動態平衡 100k 虛擬多資產投資組合",
		Objective: "最大回撤嚴格限制在 10% 內，追求年化夏普值 > 1.5",
		Scope:     "美股大盤、公債與黃金抗通膨資產配置",
	})
	if err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	// 3. Domain Analysis (AIA-22 §4.3)
	analysis := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{
			UserGoal:    "最大回撤嚴格限制在 10% 內，追求年化夏普值 > 1.5",
			Constraints: []string{"單一標的曝險不得超過 50%", "禁止加槓桿"},
		},
		Context: domainProblem.ContextSummary{
			FoundResources: []string{"ledger://brokerage/cash_usd", "dataset://macro/correlations_20y.parquet"},
		},
		GapsAndConflicts: []domainProblem.GapConflictItem{
			{
				ID:                "gap-macro-uncertainty",
				Category:          domainProblem.GapPotentialRisk,
				Description:       "下季 FOMC 升息路徑存在不確定性",
				Severity:          "HIGH",
				SuggestedHandling: domainProblem.HandlingProvisionalAssumption,
			},
		},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "fact-cash-balance",
				Category:   domainProblem.CategoryKnownFact,
				Statement:  "帳戶當前可用現金為 100,000.00 USD，無未平倉合約",
				Confidence: 1.0,
				Source:     "ledger://brokerage/cash_usd#balance",
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "investment-agent",
				CreatedAt:  time.Now().UTC(),
			},
			{
				ID:         "assump-stock-bond-hedging",
				Category:   domainProblem.CategoryAssumption,
				Statement:  "TLT 與 SPY 在未來 6 個月內維持負相關性以實現風險對沖",
				Confidence: 0.80,
				Source:     "historical_correlation_study",
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "investment-agent",
				CreatedAt:  time.Now().UTC(),
			},
			{
				ID:         "unknown-fomc-path",
				Category:   domainProblem.CategoryUnknown,
				Statement:  "下季 FOMC 升息路徑與縮表速度是否會引發股債雙殺流動性衝擊",
				Confidence: 0.0,
				Status:     domainProblem.ItemStatusActive,
				CreatedBy:  "investment-agent",
				CreatedAt:  time.Now().UTC(),
			},
		},
	}

	_, err = probSvc.ApplyAnalysis(ctx, prob.ID, analysis, "investment-agent", "exec-invest-1")
	if err != nil {
		t.Fatalf("failed to apply investment analysis: %v", err)
	}

	// 4. Solution Exploration: Static 60/40 vs Dynamic Risk Parity (AIA-22 §4.6)
	_, err = probSvc.ExploreSolutions(ctx, prob.ID, &domainPlan.SolutionExplorationResult{
		ProblemID: prob.ID,
		Alternatives: []domainPlan.SolutionAlternative{
			{
				ID:                   "alt-static-60-40",
				Title:                "靜態 60/40 股債配置",
				Description:          "60% SPY + 40% TLT 每季再平衡",
				TechnicalFeasibility: "HIGH",
				Reversibility:        "HIGH",
				EstimatedCost:        "低手續費",
				Risks:                []string{"遭遇高通膨引發股債雙殺時回撤容易突破 15%"},
			},
			{
				ID:                   "alt-dynamic-risk-parity",
				Title:                "動態風險平價 (Risk Parity) + 波動度守衛",
				Description:          "按各資產逆波動度配比，並加入 15% 黃金大宗商品避險",
				TechnicalFeasibility: "HIGH",
				Reversibility:        "HIGH",
				EstimatedCost:        "中等再平衡成本",
				ExpectedBenefits:     []string{"各資產風險貢獻均等", "最大回撤控制在 8% 內"},
			},
		},
		SelectedAlternativeID: "alt-dynamic-risk-parity",
		OverallRationale:      "動態風險平價具備更強抗滯脹能力，符合最大回撤 < 10% 限制",
	}, "investment-agent")
	if err != nil {
		t.Fatalf("failed to explore solutions: %v", err)
	}

	// 5. Generate Plan 1 with Human Approval Flag (AIA-22 §9.2)
	plan1, err := probSvc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID:             prob.ID,
		Objective:             "構建初期風險平價底倉並佈設波動度守衛",
		Strategy:              "45% 股票、35% 長債、20% 黃金商品",
		RequiresHumanApproval: true,
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-allocate-portfolio",
				Title:              "執行初始 100k 資產配置下單",
				Stage:              "Allocation",
				AssignedAgent:      "investment-agent",
				AcceptanceCriteria: []string{"現金使用率 99% 以上", "標的配置權重偏差 < 0.5%"},
			},
			{
				ID:                 "wi-volatility-guard",
				Title:              "啟動即時波動度守衛監控",
				Stage:              "Monitoring",
				AssignedAgent:      "investment-agent",
				Dependencies:       []string{"wi-allocate-portfolio"},
				AcceptanceCriteria: []string{"每 5 分鐘計算投資組合歷史年化波動度", "波動度 > 15% 時發出警報"},
			},
		},
		AcceptanceCriteria: []string{"100k 資金完全配置且守衛常駐執行"},
	})
	if err != nil {
		t.Fatalf("failed to generate plan 1: %v", err)
	}

	// Governance Check: cannot materialize without approval (AIA-22 §9.2, §16.3)
	if _, err := probSvc.MaterializeAllWorkItems(ctx, plan1.ID); err == nil {
		t.Fatalf("expected error materializing unapproved plan, got nil")
	}

	// Human approves plan
	approvedPlan, err := probSvc.ApprovePlan(ctx, plan1.ID, "portfolio-manager")
	if err != nil || approvedPlan.Status != domainPlan.StatusApproved {
		t.Fatalf("failed to approve plan: %v", err)
	}

	tasks, err := probSvc.MaterializeAllWorkItems(ctx, plan1.ID)
	if err != nil || len(tasks) != 2 {
		t.Fatalf("failed to materialize approved investment tasks: %v", err)
	}
	if string(tasks[0].AgentID) != "investment-agent" {
		t.Fatalf("expected investment-agent, got %s", tasks[0].AgentID)
	}

	// 6. Market Drift Reassessment Trigger (AIA-22 §8.1)
	// Sudden macro regime change: persistent inflation spike makes stock and bond correlation jump from -0.35 to +0.65!
	reassessReq := domainProblem.ReassessmentRequest{
		ProblemID:              prob.ID,
		Trigger:                domainProblem.TriggerHypothesisContradicted,
		TriggerEvidence:        "過去 30 天滾動相關性顯示 SPY 與 TLT 相關係數躍升至 +0.68，股債負相關性假設失效",
		RootCause:              domainProblem.RootCauseAssumptionInvalid,
		ContradictedHypotheses: []string{"assump-stock-bond-hedging"},
		NewFindings: []string{
			"宏觀滯脹環境導致股債同跌，需降低長債部位並提升現金與超短期國庫券 (BIL) 部位",
		},
		SuggestedAction: domainProblem.ActionModifyPlan,
		Rationale:       "股債相關性轉正打破原始對沖假設，執行防禦性資產切換避免回撤超標",
		AgentID:         "investment-agent",
	}

	reassessRes, err := probSvc.TriggerReassessment(ctx, reassessReq)
	if err != nil || reassessRes.NewModelVer != 2 {
		t.Fatalf("failed to trigger market drift reassessment: %v", err)
	}

	// 7. Revised Plan v2 with defensive positioning
	plan2, err := probSvc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID:        prob.ID,
		Objective:        "切換至高通膨防禦姿態：減持長債，增持現金等價物",
		Strategy:         "長債部位由 35% 縮減至 10%，剩餘 25% 轉移至 3 個月國庫券 (BIL)",
		SupersedesPlanID: plan1.ID,
		ChangeReason:     "股債相關性轉正打破原始假設，執行防禦性資產切換避免回撤超標",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-rebalance-defensive",
				Title:              "執行防禦性換倉：賣出 25% TLT 並買入 BIL",
				Stage:              "Rebalance",
				AssignedAgent:      "investment-agent",
				AcceptanceCriteria: []string{"TLT 持倉降至 10%", "BIL 增持至 25%", "滑點成本 < 0.1%"},
			},
		},
		AcceptanceCriteria: []string{"投資組合年化波動度降至 10% 以下"},
	})
	if err != nil || plan2.Version != 2 {
		t.Fatalf("failed to generate plan 2: %v", err)
	}

	// Verify plan 1 preserved as SUPERSEDED and plan changes recorded
	p1Reload, _ := probSvc.GetPlan(ctx, plan1.ID)
	if p1Reload.Status != domainPlan.StatusSuperseded {
		t.Fatalf("expected plan 1 to be SUPERSEDED, got %s", p1Reload.Status)
	}
	changes, err := probSvc.ListPlanChanges(ctx, plan2.ID)
	if err != nil || len(changes) != 1 {
		t.Fatalf("expected 1 change record, got %d", len(changes))
	}
}

// TestStage5_NegativeAcceptanceCriteria validates the explicit negative criteria in AIA-22 §16.3:
// - Reject Fact confidence != 1.0
// - Reject Fact without evidence source
// - Reject Assumption confidence == 1.0
// - Reject Unknown confidence != 0.0
// - Reject Clarification questions > 5
// - Reject Recursive decomposition exceeding MaxSubProblems
// - Reject Materializing unapproved or superseded plans
// - Reject Plans with circular dependencies
// - Reject Plans with missing acceptance criteria
func TestStage5_NegativeAcceptanceCriteria(t *testing.T) {
	ctx := context.Background()
	projRepo, probSvc, _ := setupCrossDomainTestEnvironment(t)

	proj := &domainProject.Project{
		ID:         "proj-negative-test",
		Name:       "負向驗收條件專案",
		Workspace:  "/workspace/negative",
		DomainType: domainProject.DomainCoding,
		Status:     domainProject.StatusActive,
	}
	_ = projRepo.Create(ctx, proj)

	prob, _ := probSvc.CreateProblem(ctx, problemService.CreateProblemInput{
		ProjectID: string(proj.ID),
		Title:     "負向邊界條件測試問題",
		Objective: "驗證系統對不合規分析與計畫的防禦機制",
	})

	// 1. NA-1: Reject Fact confidence != 1.0 (AIA-22 §4.3)
	badFactConf := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: "目標"},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "fact-bad-conf",
				Category:   domainProblem.CategoryKnownFact,
				Statement:  "代碼有 1000 行",
				Confidence: 0.9,
				Source:     "file://loc",
				Status:     domainProblem.ItemStatusActive,
				CreatedAt:  time.Now().UTC(),
			},
		},
	}
	if err := domainProblem.ValidateProblemAnalysis(badFactConf); err == nil {
		t.Fatalf("expected error on fact confidence != 1.0, got nil")
	}

	// 2. NA-2: Reject Fact without Source (AIA-22 §4.3)
	badFactSrc := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: "目標"},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "fact-no-source",
				Category:   domainProblem.CategoryKnownFact,
				Statement:  "代碼有 1000 行",
				Confidence: 1.0,
				Source:     "",
				Status:     domainProblem.ItemStatusActive,
				CreatedAt:  time.Now().UTC(),
			},
		},
	}
	if err := domainProblem.ValidateProblemAnalysis(badFactSrc); err == nil {
		t.Fatalf("expected error on fact without source, got nil")
	}

	// 3. NA-3: Reject Assumption confidence == 1.0 (AIA-22 §4.3)
	badAssump1 := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: "目標"},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "assump-too-confident",
				Category:   domainProblem.CategoryAssumption,
				Statement:  "可能存在連線洩漏",
				Confidence: 1.0, // Invalid for assumption
				Status:     domainProblem.ItemStatusActive,
				CreatedAt:  time.Now().UTC(),
			},
		},
	}
	if err := domainProblem.ValidateProblemAnalysis(badAssump1); err == nil {
		t.Fatalf("expected error on assumption confidence == 1.0, got nil")
	}

	// 4. NA-4: Reject Unknown confidence != 0.0 (AIA-22 §4.3)
	badUnknown := &domainProblem.AgentProblemAnalysis{
		Intent: domainProblem.IntentSummary{UserGoal: "目標"},
		ModelItems: []domainProblem.ModelItem{
			{
				ID:         "unknown-bad-conf",
				Category:   domainProblem.CategoryUnknown,
				Statement:  "生產環境未知狀況",
				Confidence: 0.5, // Must be 0.0
				Status:     domainProblem.ItemStatusActive,
				CreatedAt:  time.Now().UTC(),
			},
		},
	}
	if err := domainProblem.ValidateProblemAnalysis(badUnknown); err == nil {
		t.Fatalf("expected error on unknown confidence != 0.0, got nil")
	}

	// 5. NA-6 & NA-7: Decomposition Limits (AIA-22 §5.2)
	// Exceed MaxSubProblems
	var tooManySubProblems []problemService.CreateProblemInput
	for i := 1; i <= 25; i++ {
		tooManySubProblems = append(tooManySubProblems, problemService.CreateProblemInput{
			Title:     fmt.Sprintf("子問題 %d", i),
			Objective: "驗證超過限制",
		})
	}
	_, err := probSvc.DecomposeProblem(ctx, prob.ID, tooManySubProblems)
	if err == nil {
		t.Fatalf("expected error when decomposing > 20 subproblems, got nil")
	}

	// 6. NA-8: Materialize without Approval (AIA-22 §9.2, §16.3)
	planProposed, err := probSvc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID: prob.ID,
		Objective: "測試未批准計畫",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-unapproved",
				Title:              "未批准工作項目",
				AcceptanceCriteria: []string{"通過驗收"},
			},
		},
		AcceptanceCriteria: []string{"計畫完成"},
	})
	if err != nil {
		t.Fatalf("failed to generate plan: %v", err)
	}

	if _, err := probSvc.MaterializeWorkItemToTask(ctx, planProposed.ID, "wi-unapproved"); err == nil {
		t.Fatalf("expected error materializing from unapproved PROPOSED plan, got nil")
	}

	// 7. NA-9: Materialize from SUPERSEDED plan (AIA-22 §6.5, §16.3)
	_, _ = probSvc.ApprovePlan(ctx, planProposed.ID, "manager")
	planSuperseded, err := probSvc.GeneratePlan(ctx, problemService.GeneratePlanInput{
		ProblemID:        prob.ID,
		Objective:        "版本 2 計畫",
		SupersedesPlanID: planProposed.ID,
		ChangeReason:     "改用版本 2",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-v2",
				Title:              "版本 2 工作項目",
				AcceptanceCriteria: []string{"v2 驗收條件"},
			},
		},
		AcceptanceCriteria: []string{"v2 計畫完成"},
	})
	if err != nil || planSuperseded == nil {
		t.Fatalf("failed to generate plan 2: %v", err)
	}

	// Try materializing from the now SUPERSEDED planProposed
	if _, err := probSvc.MaterializeWorkItemToTask(ctx, planProposed.ID, "wi-unapproved"); err == nil {
		t.Fatalf("expected error materializing from SUPERSEDED plan, got nil")
	}

	// 8. NA-10: Reject Plan without Work Items or Missing Acceptance Criteria (AIA-22 §6.4, §16.3)
	emptyWIsPlan := &domainPlan.Plan{
		ID:                 "plan-empty-wis",
		ProblemID:          prob.ID,
		Version:            1,
		Objective:          "無工作項目計畫",
		WorkItems:          []domainPlan.WorkItemDef{},
		AcceptanceCriteria: []string{"目標達成"},
	}
	if err := emptyWIsPlan.Validate(); err == nil {
		t.Fatalf("expected error on plan with no work items, got nil")
	}

	noACPlan := &domainPlan.Plan{
		ID:        "plan-no-ac",
		ProblemID: prob.ID,
		Version:   1,
		Objective: "無驗收條件計畫",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-no-ac",
				Title:              "無驗收條件項目",
				AcceptanceCriteria: []string{}, // Empty acceptance criteria
			},
		},
		AcceptanceCriteria: []string{"目標達成"},
	}
	if err := noACPlan.Validate(); err == nil {
		t.Fatalf("expected error on work item with missing acceptance criteria, got nil")
	}

	// 9. NA-11: Reject Circular Dependency in Plan DAG (AIA-22 §6.4)
	circularPlan := &domainPlan.Plan{
		ID:        "plan-circular",
		ProblemID: prob.ID,
		Version:   1,
		Objective: "循環依賴計畫",
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-a",
				Title:              "項目 A",
				Dependencies:       []string{"wi-b"},
				AcceptanceCriteria: []string{"A 驗收"},
			},
			{
				ID:                 "wi-b",
				Title:              "項目 B",
				Dependencies:       []string{"wi-a"}, // B depends on A, A depends on B
				AcceptanceCriteria: []string{"B 驗收"},
			},
		},
		AcceptanceCriteria: []string{"循環計畫"},
	}
	if err := circularPlan.Validate(); err == nil {
		t.Fatalf("expected error on circular dependency in plan, got nil")
	}
}
