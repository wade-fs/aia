package unit

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	domainPlan "github.com/wade/aia/internal/domain/plan"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	domainProject "github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestProblemModel_EntityAndValidation(t *testing.T) {
	// 1. Validation error on empty ID
	p := &domainProblem.Problem{
		ProjectID: "proj-1",
		Title:     "Audit 540 chapters",
		Objective: "Verify all chapters comply with manifest rules",
	}
	if err := p.Validate(); err == nil {
		t.Fatalf("expected error on empty problem ID, got nil")
	}

	// 2. Validation error on empty ProjectID
	p.ID = "prob-1"
	p.ProjectID = ""
	if err := p.Validate(); err == nil {
		t.Fatalf("expected error on empty project ID, got nil")
	}

	// 3. Validation error on empty Title
	p.ProjectID = "proj-1"
	p.Title = ""
	if err := p.Validate(); err == nil {
		t.Fatalf("expected error on empty title, got nil")
	}

	// 4. Validation error on empty Objective
	p.Title = "Audit 540 chapters"
	p.Objective = ""
	if err := p.Validate(); err == nil {
		t.Fatalf("expected error on empty objective, got nil")
	}

	// 5. Successful validation with defaults applied
	p.Objective = "Verify compliance"
	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if p.Status != domainProblem.StatusOpen {
		t.Fatalf("expected default status OPEN, got %s", p.Status)
	}
	if p.Priority != domainProblem.PriorityMedium {
		t.Fatalf("expected default priority MEDIUM, got %s", p.Priority)
	}

	// 6. ModelItem validation: confidence bounds
	itemBad := &domainProblem.ModelItem{
		ID:         "item-1",
		Category:   domainProblem.CategoryAssumption,
		Statement:  "User intends historical fiction",
		Confidence: 1.5, // invalid > 1.0
	}
	if err := itemBad.Validate(); err == nil {
		t.Fatalf("expected error on confidence > 1.0, got nil")
	}
	itemBad.Confidence = -0.1
	if err := itemBad.Validate(); err == nil {
		t.Fatalf("expected error on confidence < 0.0, got nil")
	}

	itemValid := &domainProblem.ModelItem{
		ID:         "item-1",
		Category:   domainProblem.CategoryKnownFact,
		Statement:  "The project has 540 chapters written in markdown",
		Confidence: 1.0,
	}
	if err := itemValid.Validate(); err != nil {
		t.Fatalf("unexpected error validating valid ModelItem: %v", err)
	}
	if itemValid.Status != domainProblem.ItemStatusActive {
		t.Fatalf("expected default item status ACTIVE, got %s", itemValid.Status)
	}
}

func TestProblemModel_Repository_CRUD_And_Hierarchy(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)

	// Ensure project exists
	proj := &domainProject.Project{
		ID:         "proj-novel-compliance",
		Name:       "小說全書稽核",
		Workspace:  "/workspace/novel",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}

	// 1. Create parent problem (AIA-22 §6.1, §5.1)
	parent := &domainProblem.Problem{
		ID:          "prob-whole-book-audit",
		ProjectID:   proj.ID,
		Title:       "全書規範合規性稽核",
		Description: "稽核 540 章是否全面符合規範文件與角色約束",
		Objective:   "產出完整的稽核報告、衝突矩陣與修訂建議清單",
		Scope:       "全書 540 章，共 10 卷",
		Status:      domainProblem.StatusAnalyzing,
		Priority:    domainProblem.PriorityHigh,
	}
	if err := problemRepo.Create(ctx, parent); err != nil {
		t.Fatalf("failed to create parent problem: %v", err)
	}

	// 2. Create child sub-problems (AIA-22 §5.1 Recursive Decomposition)
	child1 := &domainProblem.Problem{
		ID:          "prob-rule-catalog",
		ProjectID:   proj.ID,
		ParentID:    parent.ID,
		Title:       "規範文件提取與衝突清點",
		Objective:   "提取所有規範規則並消除定義衝突",
		Status:      domainProblem.StatusOpen,
		Priority:    domainProblem.PriorityHigh,
	}
	child2 := &domainProblem.Problem{
		ID:          "prob-chapter-audit",
		ProjectID:   proj.ID,
		ParentID:    parent.ID,
		Title:       "分章結構與史實合規性檢查",
		Objective:   "分批檢查每章格式、字數與史料依據",
		Status:      domainProblem.StatusOpen,
		Priority:    domainProblem.PriorityMedium,
	}
	if err := problemRepo.Create(ctx, child1); err != nil {
		t.Fatalf("failed to create child1: %v", err)
	}
	if err := problemRepo.Create(ctx, child2); err != nil {
		t.Fatalf("failed to create child2: %v", err)
	}

	// 3. List children
	children, err := problemRepo.ListChildren(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to list child problems: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	if children[0].ID != child1.ID || children[1].ID != child2.ID {
		t.Fatalf("unexpected children ordering or IDs: %+v", children)
	}

	// 4. Update mutable fields
	parent.Status = domainProblem.StatusPlanned
	parent.Scope = "全書 540 章，共 10 卷及番外"
	if err := problemRepo.Update(ctx, parent); err != nil {
		t.Fatalf("failed to update problem: %v", err)
	}

	reloaded, err := problemRepo.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to get reloaded problem: %v", err)
	}
	if reloaded.Status != domainProblem.StatusPlanned || reloaded.Scope != parent.Scope {
		t.Fatalf("problem update failed to persist: %+v", reloaded)
	}
}

func TestProblemModel_Versioning_And_EpistemicIntegrity(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)

	proj := &domainProject.Project{
		ID:         "proj-coding-audit",
		Name:       "Go 專案架構改善",
		Workspace:  "/workspace/go-repo",
		DomainType: domainProject.DomainCoding,
		Status:     domainProject.StatusActive,
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	prob := &domainProblem.Problem{
		ID:        "prob-go-refactor",
		ProjectID: proj.ID,
		Title:     "核心併發架構與技術債重構",
		Objective: "改善模組依賴與記憶體佔用",
		Status:    domainProblem.StatusAnalyzing,
	}
	if err := problemRepo.Create(ctx, prob); err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	// 1. Save ProblemModel Version 1 (AIA-22 §6.2, §7.1)
	m1 := &domainProblem.ProblemModel{
		ID:        "model-go-v1",
		ProblemID: prob.ID,
		Version:   1,
		KnownFacts: []domainProblem.ModelItem{
			{
				ID:          "fact-1",
				Category:    domainProblem.CategoryKnownFact,
				Statement:   "Repository has 45 packages and 120,000 LOC",
				Source:      "git_workspace_scan",
				Confidence:  1.0,
				Status:      domainProblem.ItemStatusConfirmed,
				CreatedBy:   "coding-agent",
				CreatedAt:   time.Now().UTC(),
			},
		},
		Assumptions: []domainProblem.ModelItem{
			{
				ID:          "assump-1",
				Category:    domainProblem.CategoryAssumption,
				Statement:   "Primary memory bottleneck is in eventbus worker pool",
				Source:      "static_call_graph",
				Confidence:  0.70,
				Status:      domainProblem.ItemStatusActive,
				CreatedBy:   "coding-agent",
				CreatedAt:   time.Now().UTC(),
			},
		},
		Unknowns: []domainProblem.ModelItem{
			{
				ID:          "unk-1",
				Category:    domainProblem.CategoryUnknown,
				Statement:   "Maximum concurrency peak in production environment",
				Source:      "architecture_gap_analysis",
				Confidence:  0.0,
				Status:      domainProblem.ItemStatusActive,
				CreatedBy:   "coding-agent",
				CreatedAt:   time.Now().UTC(),
			},
		},
	}
	if err := problemRepo.SaveModel(ctx, m1); err != nil {
		t.Fatalf("failed to save model v1: %v", err)
	}

	// Verify current_model_version bumped to 1 on Problem
	pReload, err := problemRepo.Get(ctx, prob.ID)
	if err != nil || pReload.CurrentModelVersion != 1 {
		t.Fatalf("expected CurrentModelVersion = 1, got %d (err: %v)", pReload.CurrentModelVersion, err)
	}

	// 2. Save ProblemModel Version 2 (new evidence collected, assumption confirmed)
	m2 := &domainProblem.ProblemModel{
		ID:        "model-go-v2",
		ProblemID: prob.ID,
		Version:   2,
		KnownFacts: []domainProblem.ModelItem{
			m1.KnownFacts[0],
			{
				ID:          "fact-2",
				Category:    domainProblem.CategoryKnownFact,
				Statement:   "Benchmark profile confirms eventbus worker pool accounts for 68% allocations",
				Source:      "pprof_bench_run",
				Confidence:  1.0,
				Status:      domainProblem.ItemStatusConfirmed,
				CreatedBy:   "coding-agent",
				CreatedAt:   time.Now().UTC(),
			},
		},
		Assumptions: []domainProblem.ModelItem{
			{
				ID:          "assump-1",
				Category:    domainProblem.CategoryAssumption,
				Statement:   "Primary memory bottleneck is in eventbus worker pool",
				Source:      "pprof_bench_run",
				Confidence:  1.0,
				Status:      domainProblem.ItemStatusConfirmed, // Promoted from hypothesis to confirmed fact
				CreatedBy:   "coding-agent",
				CreatedAt:   time.Now().UTC(),
			},
		},
		Unknowns: []domainProblem.ModelItem{}, // Resolved unknown
	}
	if err := problemRepo.SaveModel(ctx, m2); err != nil {
		t.Fatalf("failed to save model v2: %v", err)
	}

	// Verify current_model_version bumped to 2
	pReload2, err := problemRepo.Get(ctx, prob.ID)
	if err != nil || pReload2.CurrentModelVersion != 2 {
		t.Fatalf("expected CurrentModelVersion = 2, got %d", pReload2.CurrentModelVersion)
	}

	// 3. Immutability check: historical version 1 must remain intact
	m1History, err := problemRepo.GetModel(ctx, prob.ID, 1)
	if err != nil {
		t.Fatalf("failed to get model v1 from history: %v", err)
	}
	if len(m1History.KnownFacts) != 1 || len(m1History.Unknowns) != 1 {
		t.Fatalf("historical model v1 was mutated! got %+v", m1History)
	}
	if m1History.Assumptions[0].Status != domainProblem.ItemStatusActive {
		t.Fatalf("historical model v1 assumption status was mutated: %s", m1History.Assumptions[0].Status)
	}

	// 4. Latest model check
	latestModel, err := problemRepo.GetLatestModel(ctx, prob.ID)
	if err != nil || latestModel.Version != 2 {
		t.Fatalf("expected latest model version 2, got version %d (err: %v)", latestModel.Version, err)
	}
	if len(latestModel.KnownFacts) != 2 {
		t.Fatalf("expected 2 known facts in model v2, got %d", len(latestModel.KnownFacts))
	}
}

func TestAnalysisRecord_Immutability_And_History(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)

	proj := &domainProject.Project{
		ID:         "proj-inv-audit",
		Name:       "投資策略分析",
		Workspace:  "/workspace/inv",
		DomainType: domainProject.DomainInvestment,
		Status:     domainProject.StatusActive,
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	prob := &domainProblem.Problem{
		ID:        "prob-inv-strategy",
		ProjectID: proj.ID,
		Title:     "虛擬資產長期增長策略分析",
		Objective: "分析市場波動風險與資產配置方案",
	}
	if err := problemRepo.Create(ctx, prob); err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	// 1. Record Analysis 1: Intent Understanding (AIA-22 §4.2)
	rec1 := &domainProblem.AnalysisRecord{
		ID:           "rec-intent-01",
		ProblemID:    prob.ID,
		ModelVersion: 1,
		AnalysisType: "INTENT",
		InputRefs:    []string{"user_prompt:10萬元虛擬資產持續增長", "aia.yaml:risk_level"},
		Findings:     []string{"User specified objective without constraining asset universe", "Max drawdown tolerance not specified"},
		Conclusions:  []string{"Adopt balanced multi-asset allocation as baseline"},
		OpenQuestions: []string{"Whether shorting is permissible"},
		AgentID:      "investment-agent",
	}
	if err := problemRepo.CreateAnalysisRecord(ctx, rec1); err != nil {
		t.Fatalf("failed to record analysis 1: %v", err)
	}

	// 2. Record Analysis 2: Context Investigation (AIA-22 §4.3)
	rec2 := &domainProblem.AnalysisRecord{
		ID:           "rec-context-02",
		ProblemID:    prob.ID,
		ModelVersion: 1,
		AnalysisType: "CONTEXT",
		InputRefs:    []string{"db:investment_instruments", "db:investment_predictions"},
		Findings:     []string{"Available instruments: 12 stocks, 3 bonds", "Historical prediction calibration MAE = 0.045"},
		Conclusions:  []string{"Sufficient instruments available for diversification"},
		AgentID:      "investment-agent",
	}
	if err := problemRepo.CreateAnalysisRecord(ctx, rec2); err != nil {
		t.Fatalf("failed to record analysis 2: %v", err)
	}

	// 3. Query all analysis records
	records, err := problemRepo.ListAnalysisRecords(ctx, prob.ID)
	if err != nil {
		t.Fatalf("failed to list analysis records: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 analysis records, got %d", len(records))
	}
	if records[0].AnalysisType != "INTENT" || records[1].AnalysisType != "CONTEXT" {
		t.Fatalf("unexpected analysis records order: %+v", records)
	}
	if len(records[0].OpenQuestions) != 1 || records[0].OpenQuestions[0] != "Whether shorting is permissible" {
		t.Fatalf("open questions not preserved: %+v", records[0].OpenQuestions)
	}
}

func TestPlan_Versioning_Supersedes_And_ChangeLog(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	projRepo := repository.NewProjectRepository(db)
	problemRepo := repository.NewProblemRepository(db)
	planRepo := repository.NewPlanRepository(db)

	proj := &domainProject.Project{
		ID:         "proj-plan-test",
		Name:       "測試計畫專案案案",
		Workspace:  "/workspace/plan-test",
		DomainType: domainProject.DomainWriting,
		Status:     domainProject.StatusActive,
	}
	if err := projRepo.Create(ctx, proj); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	prob := &domainProblem.Problem{
		ID:        "prob-plan-root",
		ProjectID: proj.ID,
		Title:     "全書章節結構合規計畫",
		Objective: "完成章節稽核與重構",
	}
	if err := problemRepo.Create(ctx, prob); err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	// 1. Create Plan Version 1 (AIA-22 §6.4)
	plan1 := &domainPlan.Plan{
		ID:        "plan-v1",
		ProblemID: prob.ID,
		Version:   1,
		Objective: "分批執行 540 章靜態語法與格式驗證",
		Strategy:  "循序單執行緒驗證",
		Status:    domainPlan.StatusDraft,
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-1",
				Title:              "驗證第 1~100 章",
				Stage:              "Batch1",
				AssignedAgent:      "writing-agent",
				AcceptanceCriteria: []string{"Zero format error", "Word count >= 3000"},
				Status:             domainPlan.WorkItemReady,
			},
			{
				ID:                 "wi-2",
				Title:              "驗證第 101~200 章",
				Stage:              "Batch2",
				AssignedAgent:      "writing-agent",
				Dependencies:       []string{"wi-1"},
				AcceptanceCriteria: []string{"Zero format error", "Word count >= 3000"},
				Status:             domainPlan.WorkItemPending,
			},
		},
		AcceptanceCriteria: []string{"All 200 chapters validated"},
	}
	if err := planRepo.Create(ctx, plan1); err != nil {
		t.Fatalf("failed to create plan 1: %v", err)
	}

	// Check problem current_plan_version
	pReload, _ := problemRepo.Get(ctx, prob.ID)
	if pReload.CurrentPlanVersion != 1 {
		t.Fatalf("expected CurrentPlanVersion = 1, got %d", pReload.CurrentPlanVersion)
	}

	// 2. Create Plan Version 2 (reassessed: switch to 5-worker parallel batching)
	plan2 := &domainPlan.Plan{
		ID:               "plan-v2",
		ProblemID:        prob.ID,
		Version:          2,
		Objective:        "平行化執行 540 章靜態語法與格式驗證",
		Strategy:         "5 組並行工作單元提升吞吐量",
		Status:           domainPlan.StatusProposed,
		SupersedesPlanID: plan1.ID, // Supersedes Plan 1 (AIA-22 §6.4)
		WorkItems: []domainPlan.WorkItemDef{
			{
				ID:                 "wi-p1",
				Title:              "並行驗證分卷 1~5",
				Stage:              "ParallelBatch",
				AssignedAgent:      "writing-agent",
				AcceptanceCriteria: []string{"All chapters pass format verification"},
				Status:             domainPlan.WorkItemReady,
			},
			{
				ID:                 "wi-p2",
				Title:              "並行驗證分卷 6~10",
				Stage:              "ParallelBatch",
				AssignedAgent:      "writing-agent",
				AcceptanceCriteria: []string{"All chapters pass format verification"},
				Status:             domainPlan.WorkItemReady,
			},
		},
		AcceptanceCriteria: []string{"All 10 volumes validated in under 3 minutes"},
	}
	if err := planRepo.Create(ctx, plan2); err != nil {
		t.Fatalf("failed to create plan 2: %v", err)
	}

	// Check problem current_plan_version updated to 2
	pReload2, _ := problemRepo.Get(ctx, prob.ID)
	if pReload2.CurrentPlanVersion != 2 {
		t.Fatalf("expected CurrentPlanVersion = 2, got %d", pReload2.CurrentPlanVersion)
	}

	// 3. Record PlanChangeRecord (AIA-22 §6.5)
	changeRec := &domainPlan.PlanChangeRecord{
		ID:                    "change-01",
		PlanID:                plan2.ID,
		FromVersion:           1,
		ToVersion:             2,
		Reason:                "循序執行 540 章耗時過久（預估逾 40 分鐘），改以分卷並行處理可縮短至 3 分鐘",
		EvidenceRefs:          []string{"benchmark:sequential_throughput_too_slow"},
		AffectedWorkItems:     []string{"wi-1", "wi-2"},
		CostRiskImpact:        "增加瞬時 SQLite 連線負載，但整體時間節省 90%",
		RequiresHumanApproval: false,
	}
	if err := planRepo.RecordChange(ctx, changeRec); err != nil {
		t.Fatalf("failed to record plan change: %v", err)
	}

	// Verify change record query
	changes, err := planRepo.ListChanges(ctx, plan2.ID)
	if err != nil || len(changes) != 1 {
		t.Fatalf("failed to list changes for plan 2: %v, count=%d", err, len(changes))
	}
	if changes[0].FromVersion != 1 || changes[0].ToVersion != 2 {
		t.Fatalf("unexpected change record version mapping: %+v", changes[0])
	}

	// 4. Update status to APPROVED
	if err := planRepo.UpdateStatus(ctx, plan2.ID, domainPlan.StatusApproved); err != nil {
		t.Fatalf("failed to update plan status: %v", err)
	}
	pApproved, err := planRepo.Get(ctx, plan2.ID)
	if err != nil || pApproved.Status != domainPlan.StatusApproved {
		t.Fatalf("expected plan status APPROVED, got %s (err: %v)", pApproved.Status, err)
	}
	if pApproved.SupersedesPlanID != plan1.ID {
		t.Fatalf("expected supersedes plan ID = %s, got %s", plan1.ID, pApproved.SupersedesPlanID)
	}
}

func TestProblemAndPlan_CrashRecovery_Simulation(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "recovery_test.db")

	// Phase 1: Initialize DB, run migrations, write full problem and plan lifecycle
	{
		db, err := sqlite.NewDB(dbPath)
		if err != nil {
			t.Fatalf("failed to create sqlite DB at %s: %v", dbPath, err)
		}

		projRepo := repository.NewProjectRepository(db)
		problemRepo := repository.NewProblemRepository(db)
		planRepo := repository.NewPlanRepository(db)

		proj := &domainProject.Project{
			ID:         "proj-persist-check",
			Name:       "持久化重啟測試專案",
			Workspace:  tempDir,
			DomainType: domainProject.DomainWriting,
			Status:     domainProject.StatusActive,
		}
		if err := projRepo.Create(ctx, proj); err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		prob := &domainProblem.Problem{
			ID:        "prob-crash-recovery",
			ProjectID: proj.ID,
			Title:     "持久性與中斷恢復驗證問題",
			Objective: "驗證系統重啟後完整問題模型與計畫版本還原能力",
			Status:    domainProblem.StatusAnalyzing,
		}
		if err := problemRepo.Create(ctx, prob); err != nil {
			t.Fatalf("failed to create problem: %v", err)
		}

		// Save Model v1
		m1 := &domainProblem.ProblemModel{
			ID:        "mod-v1",
			ProblemID: prob.ID,
			Version:   1,
			KnownFacts: []domainProblem.ModelItem{
				{
					ID:         "kf-1",
					Category:   domainProblem.CategoryKnownFact,
					Statement:  "Initial state verified",
					Confidence: 1.0,
				},
			},
		}
		if err := problemRepo.SaveModel(ctx, m1); err != nil {
			t.Fatalf("failed to save model v1: %v", err)
		}

		// Save Model v2
		m2 := &domainProblem.ProblemModel{
			ID:        "mod-v2",
			ProblemID: prob.ID,
			Version:   2,
			KnownFacts: []domainProblem.ModelItem{
				{
					ID:         "kf-1",
					Category:   domainProblem.CategoryKnownFact,
					Statement:  "Initial state verified",
					Confidence: 1.0,
				},
				{
					ID:         "kf-2",
					Category:   domainProblem.CategoryKnownFact,
					Statement:  "Subsequent evidence collected",
					Confidence: 1.0,
				},
			},
		}
		if err := problemRepo.SaveModel(ctx, m2); err != nil {
			t.Fatalf("failed to save model v2: %v", err)
		}

		// Save Plan v1
		plan1 := &domainPlan.Plan{
			ID:        "pl-v1",
			ProblemID: prob.ID,
			Version:   1,
			Objective: "Plan version 1 objective",
			Status:    domainPlan.StatusActive,
			WorkItems: []domainPlan.WorkItemDef{
				{
					ID:                 "wi-recovery-1",
					Title:              "Recovery verification task",
					AcceptanceCriteria: []string{"Persisted properly across process boundary"},
				},
			},
		}
		if err := planRepo.Create(ctx, plan1); err != nil {
			t.Fatalf("failed to create plan v1: %v", err)
		}

		// Save Analysis Record
		rec := &domainProblem.AnalysisRecord{
			ID:           "ar-1",
			ProblemID:    prob.ID,
			ModelVersion: 2,
			AnalysisType: "REASSESSMENT",
			Findings:     []string{"Finding preserved across crash"},
		}
		if err := problemRepo.CreateAnalysisRecord(ctx, rec); err != nil {
			t.Fatalf("failed to create analysis record: %v", err)
		}

		// Close DB simulating crash/restart
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close DB: %v", err)
		}
	}

	// Phase 2: Reopen DB, verify all entities recover perfectly without data loss (AIA-22 §12.2)
	{
		dbReopened, err := sqlite.NewDB(dbPath)
		if err != nil {
			t.Fatalf("failed to reopen DB: %v", err)
		}
		defer dbReopened.Close()

		problemRepo := repository.NewProblemRepository(dbReopened)
		planRepo := repository.NewPlanRepository(dbReopened)

		prob, err := problemRepo.Get(ctx, "prob-crash-recovery")
		if err != nil {
			t.Fatalf("failed to recover problem after restart: %v", err)
		}
		if prob.CurrentModelVersion != 2 {
			t.Fatalf("expected CurrentModelVersion = 2 after restart, got %d", prob.CurrentModelVersion)
		}
		if prob.CurrentPlanVersion != 1 {
			t.Fatalf("expected CurrentPlanVersion = 1 after restart, got %d", prob.CurrentPlanVersion)
		}

		latestModel, err := problemRepo.GetLatestModel(ctx, prob.ID)
		if err != nil || latestModel.Version != 2 {
			t.Fatalf("failed to recover latest model v2: %v", err)
		}
		if len(latestModel.KnownFacts) != 2 {
			t.Fatalf("expected 2 known facts in reloaded model v2, got %d", len(latestModel.KnownFacts))
		}

		plan, err := planRepo.GetLatestByProblem(ctx, prob.ID)
		if err != nil || plan.Version != 1 {
			t.Fatalf("failed to recover latest plan v1: %v", err)
		}
		if plan.Objective != "Plan version 1 objective" {
			t.Fatalf("recovered plan objective mismatch: %s", plan.Objective)
		}

		records, err := problemRepo.ListAnalysisRecords(ctx, prob.ID)
		if err != nil || len(records) != 1 {
			t.Fatalf("failed to recover analysis records: %v, count=%d", err, len(records))
		}
		if records[0].Findings[0] != "Finding preserved across crash" {
			t.Fatalf("recovered finding mismatch: %s", records[0].Findings[0])
		}
	}
}
