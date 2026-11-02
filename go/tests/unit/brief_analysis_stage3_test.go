package unit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	domainClarify "github.com/wade/aia/internal/domain/clarification"
	domainProject "github.com/wade/aia/internal/domain/project"
	domainTask "github.com/wade/aia/internal/domain/task"
	briefService "github.com/wade/aia/internal/service/brief"
	clarifyService "github.com/wade/aia/internal/service/clarification"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestBriefAnalysis_SchemaAndValidation_Rejections(t *testing.T) {
	cat, err := domainBrief.GetCatalog("nonfiction_plan")
	if err != nil {
		t.Fatalf("failed to get catalog: %v", err)
	}

	// 1. Nil analysis rejected
	if err := domainBrief.ValidateBriefAnalysis(nil, cat); err == nil {
		t.Errorf("expected error for nil analysis, got nil")
	}

	// 2. Empty summary rejected
	badSummary := &domainBrief.AgentBriefAnalysis{Summary: ""}
	if err := domainBrief.ValidateBriefAnalysis(badSummary, cat); err == nil {
		t.Errorf("expected error for empty summary, got nil")
	}

	// 3. Unauthorized field in missing_fields rejected (AIA-21 §6.3 rule 3 & 4)
	badMissing := &domainBrief.AgentBriefAnalysis{
		Summary: "Valid summary",
		MissingFields: []domainBrief.MissingFieldAnalysis{
			{
				Field:      "unauthorized_alien_field",
				Importance: "high",
				Blocking:   true,
				Reason:     "Not in catalog",
			},
		},
	}
	if err := domainBrief.ValidateBriefAnalysis(badMissing, cat); err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("expected unauthorized missing field error, got: %v", err)
	}

	// 4. Proposed default with empty rationale rejected (AIA-21 §4.4)
	badDefaultRationale := &domainBrief.AgentBriefAnalysis{
		Summary: "Valid summary",
		ProposedDefaults: []domainBrief.ProposedDefault{
			{
				Field:      "daily_mileage_km",
				Value:      25,
				Rationale:  "", // Missing rationale
				Confidence: 0.7,
			},
		},
	}
	if err := domainBrief.ValidateBriefAnalysis(badDefaultRationale, cat); err == nil {
		t.Errorf("expected error for empty rationale, got nil")
	}

	// 5. Proposed default with confidence 1.0 rejected (AIA-21 §6.3 rule 10: agent guesses cannot be marked as 100% confirmed facts)
	badConfidence := &domainBrief.AgentBriefAnalysis{
		Summary: "Valid summary",
		ProposedDefaults: []domainBrief.ProposedDefault{
			{
				Field:      "daily_mileage_km",
				Value:      25,
				Rationale:  "Valid rationale",
				Confidence: 1.0, // Invalid for agent default
			},
		},
	}
	if err := domainBrief.ValidateBriefAnalysis(badConfidence, cat); err == nil || !strings.Contains(err.Error(), "confidence") {
		t.Errorf("expected error for confidence >= 1.0, got: %v", err)
	}

	// 6. Clarification question with missing reason or impact rejected
	badQuestion := &domainBrief.AgentBriefAnalysis{
		Summary: "Valid summary",
		ClarificationQuestions: []*domainClarify.ClarificationRequest{
			{
				ProjectID: "p1",
				FieldKey:  "route_scope",
				Question:  "徒步路線為何？",
				Reason:    "", // Missing reason
				Impact:    "架構影響",
			},
		},
	}
	if err := domainBrief.ValidateBriefAnalysis(badQuestion, cat); err == nil {
		t.Errorf("expected error for question with missing reason, got nil")
	}

	// 7. Discipline limit: > 5 questions rejected (AIA-21 §5.4, §6.3)
	var overQuestions []*domainClarify.ClarificationRequest
	for i := 0; i < 6; i++ {
		overQuestions = append(overQuestions, &domainClarify.ClarificationRequest{
			ProjectID: "p1",
			FieldKey:  string(rune('a' + i)),
			Question:  "問題？",
			Reason:    "原因",
			Impact:    "影響",
		})
	}
	badOver := &domainBrief.AgentBriefAnalysis{
		Summary:                "Valid summary",
		ClarificationQuestions: overQuestions,
	}
	if err := domainBrief.ValidateBriefAnalysis(badOver, cat); err == nil {
		t.Errorf("expected error for exceeding 5 questions, got nil")
	}

	// 8. Agent attempting to mark question as ANSWERED or CONFIRMED rejected (AIA-21 §6.3 rule 10)
	fakeAnswered := &domainBrief.AgentBriefAnalysis{
		Summary: "Valid summary",
		ClarificationQuestions: []*domainClarify.ClarificationRequest{
			{
				ProjectID: "p1",
				FieldKey:  "route_scope",
				Question:  "徒步路線為何？",
				Reason:    "規劃路線需要",
				Impact:    "全書結構",
				Status:    domainClarify.StatusAnswered, // Agent trying to forge answer
			},
		},
	}
	if err := domainBrief.ValidateBriefAnalysis(fakeAnswered, cat); err == nil || !strings.Contains(err.Error(), "non-OPEN status") {
		t.Errorf("expected error for agent forging non-OPEN question status, got: %v", err)
	}
}

func TestBriefAnalysis_PromptGeneration(t *testing.T) {
	b := domainBrief.NewProjectBrief("brief-walk", "proj-walk-01", "nonfiction_plan", "徒步環台灣本島一圈計畫")
	cat, err := domainBrief.GetCatalog("nonfiction_plan")
	if err != nil {
		t.Fatalf("failed to get catalog: %v", err)
	}

	prompt := domainBrief.BuildBriefAnalysisPrompt(b, cat, "You are a professional travel writer and plan architect.", []string{
		"Decision: Target completion in 45-50 days",
	})

	// Check essential components in the generated prompt
	if !strings.Contains(prompt, "proj-walk-01") {
		t.Errorf("expected prompt to contain project id")
	}
	if !strings.Contains(prompt, "nonfiction_plan") {
		t.Errorf("expected prompt to contain domain name")
	}
	if !strings.Contains(prompt, "route_scope") || !strings.Contains(prompt, "daily_mileage_km") {
		t.Errorf("expected prompt to list catalog fields")
	}
	if !strings.Contains(prompt, "BriefAnalysisJSONSchema") && !strings.Contains(prompt, "$schema") {
		t.Errorf("expected prompt to include JSON schema")
	}
	if !strings.Contains(prompt, "at most 5 high-impact questions") {
		t.Errorf("expected prompt to enforce question discipline")
	}
}

func TestBriefAnalysis_ApplyAnalysis_And_ReadinessTransition(t *testing.T) {
	// Canonical Acceptance Scenario (AIA-21 §17):
	// Project: "我想寫一本關於計劃徒步環台灣本島一圈的計畫的書，請幫我規劃這本書。"
	// Brief is initialized with unknown fields.
	// Agent analyzes brief and produces:
	// - 1 blocking P0 clarification question for route_scope
	// - 1 proposed default for daily_mileage_km = 20 km (confidence = 0.65, with rationale)
	// ApplyAnalysis updates brief fields, registers clarification request, and pauses running tasks.

	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	wsDir := t.TempDir()
	projRepo := repository.NewProjectRepository(db)
	briefRepo := repository.NewGenericBriefRepository(db)
	clarifyRepo := repository.NewClarificationRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	memRepo := repository.NewMemoryRepository(db)
	agentRepo := repository.NewAgentRepository(db)

	briefSvc := briefService.NewService(briefRepo, projRepo)
	clarifySvc := clarifyService.NewService(clarifyRepo, briefSvc, projRepo, taskRepo, memRepo)

	projID := "proj-taiwan-stage3"
	p := &domainProject.Project{
		ID:          domain.ID(projID),
		Name:        "徒步環島規劃書",
		Workspace:   wsDir,
		DomainType:  "general",
		Status:      domainProject.StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := projRepo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Seed Agent
	ag := &agent.Agent{
		ID:           "agy-planner",
		Name:         "Plan Specialist",
		Domain:       "writing",
		Enabled:      true,
		Instructions: "Analyze nonfiction travel planning briefs",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := agentRepo.Upsert(ctx, ag); err != nil {
		t.Fatalf("failed to upsert agent: %v", err)
	}

	// Create running task
	taskID := "task-analyze-01"
	tk := &domainTask.Task{
		ID:        domain.ID(taskID),
		ProjectID: domain.ID(projID),
		Title:     "分析徒步環島需求缺口",
		AgentID:   "agy-planner",
		Status:    domainTask.StatusRunning,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := taskRepo.Create(ctx, tk, nil); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Initialize Brief
	_, err := briefSvc.InitializeBrief(ctx, projID, "nonfiction_plan", "我想寫一本關於計劃徒步環台灣本島一圈的計畫的書", wsDir)
	if err != nil {
		t.Fatalf("failed to init brief: %v", err)
	}

	// Formulate AgentBriefAnalysis payload
	analysis := &domainBrief.AgentBriefAnalysis{
		Summary: "規劃一本結合徒步環台灣本島實際規劃與個人出版的非虛構計畫書",
		Understanding: []string{
			"需要兼顧實際步行的地理補給與書籍出版架構",
			"需要釐清路線方向以決定分卷章節順序",
		},
		MissingFields: []domainBrief.MissingFieldAnalysis{
			{
				Field:      "route_scope",
				Importance: "high",
				Blocking:   true,
				Reason:     "決定全書各章節路線順序與地形安全防護篇幅",
			},
		},
		ClarificationQuestions: []*domainClarify.ClarificationRequest{
			{
				FieldKey: "route_scope",
				Question: "請問徒步路線是規劃順時針還是逆時針？主走省道公路還是特定自然步道？",
				Reason:   "不同路線的地形起伏、季風風向（如東北季風）及補給點差異極大，直接決定分卷與每日規劃順序。",
				Impact:   "阻擋全書分卷目錄與補給章節篇幅規劃。",
				Options:  []string{"順時針（先西部後東部）", "逆時針（先東部後西部）", "特殊人文步道縱走"},
				Priority: domainClarify.PriorityP0,
				Blocking: true,
			},
		},
		ProposedDefaults: []domainBrief.ProposedDefault{
			{
				Field:      "daily_mileage_km",
				Value:      20,
				Rationale:  "以一般休閒徒步者體力為基準，先採用保守的每日 20 公里預估行程。",
				Confidence: 0.65,
			},
		},
		ResearchCandidates: []string{
			"台灣環島 1 號線與台 9 / 台 11 補給點分佈",
			"四季東北季風對東部徒步體力的影響數據",
		},
		Recommendations: []string{
			"優先確認路線走向，之後可同步進行裝備與住宿資料庫調研",
		},
		Readiness: domainBrief.AnalysisReadiness{
			Status: "NOT_READY",
			Reason: "尚有路線走向關鍵未知事項待使用者確認",
		},
	}

	// 1. Apply Analysis
	updatedBrief, err := briefSvc.ApplyAnalysis(ctx, projID, analysis, clarifySvc, taskID, wsDir)
	if err != nil {
		t.Fatalf("ApplyAnalysis failed: %v", err)
	}

	// 2. Verify Brief fields
	mileageField, ok := updatedBrief.GetField("daily_mileage_km")
	if !ok || mileageField.Status != domainBrief.FieldStatusDefaulted || mileageField.Value != 20 {
		t.Fatalf("expected daily_mileage_km to be DEFAULTED with value 20, got: %+v", mileageField)
	}
	if !mileageField.ConfirmationRequired {
		t.Fatalf("expected proposed default to require confirmation")
	}

	routeField, ok := updatedBrief.GetField("route_scope")
	if !ok || routeField.Status != domainBrief.FieldStatusAsked {
		t.Fatalf("expected route_scope to be ASKED, got: %+v", routeField)
	}

	// 3. Verify Clarification Request was created and persisted
	openQuestions, err := clarifyRepo.ListOpenByProject(ctx, projID)
	if err != nil || len(openQuestions) != 1 {
		t.Fatalf("expected 1 open clarification question, got %d (err: %v)", len(openQuestions), err)
	}
	if openQuestions[0].FieldKey != "route_scope" || !openQuestions[0].Blocking {
		t.Fatalf("unexpected open question: %+v", openQuestions[0])
	}

	// 4. Verify Task transitioned to WAITING_FOR_INPUT (AIA-21 §7.1, §7.2)
	curTask, err := taskRepo.Get(ctx, domain.ID(taskID))
	if err != nil || curTask.Status != domainTask.StatusWaitingInput {
		t.Fatalf("expected task status WAITING_FOR_INPUT due to blocking clarification, got %s", curTask.Status)
	}
}
