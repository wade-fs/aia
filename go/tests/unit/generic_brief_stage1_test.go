package unit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	domainProject "github.com/wade/aia/internal/domain/project"
	briefService "github.com/wade/aia/internal/service/brief"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestGenericBrief_FieldState_And_ImmutableHistory(t *testing.T) {
	b := domainBrief.NewProjectBrief("brief-test-01", "proj-walk-01", "nonfiction_plan", "徒步環台灣本島")

	// 1. Initial State: Set default for daily_mileage
	b.SetField(
		"daily_mileage_km",
		20,
		domainBrief.FieldStatusDefaulted,
		"agent_recommendation",
		0.65,
		"以一般休閒徒步者為初步規劃對象，先採用較保守的每日步行距離。",
		true,
		"agent",
		"task-01",
		"",
	)

	fs, ok := b.GetField("daily_mileage_km")
	if !ok || fs.Value != 20 || fs.Status != domainBrief.FieldStatusDefaulted {
		t.Fatalf("expected defaulted field value 20, got %+v", fs)
	}
	if !fs.ConfirmationRequired {
		t.Fatalf("expected confirmation required to be true for agent default")
	}

	// 2. User Answer: changes to 25 km
	b.SetField(
		"daily_mileage_km",
		25,
		domainBrief.FieldStatusAnswered,
		"user",
		1.0,
		"使用者表示平時有慢跑與登山習慣，希望每天推進 25 公里以縮短總天數。",
		false,
		"user",
		"task-01",
		"clarify-01",
	)

	fs2, _ := b.GetField("daily_mileage_km")
	if fs2.Value != 25 || fs2.Status != domainBrief.FieldStatusAnswered {
		t.Fatalf("expected answered field value 25, got %+v", fs2)
	}

	// 3. Post-research confirmation: refined to 22 km due to mountain road condition
	b.SetField(
		"daily_mileage_km",
		22,
		domainBrief.FieldStatusConfirmed,
		"research_refined",
		0.95,
		"綜合台東與花蓮路段地形坡度研究後，雙方確認採 22 公里作為基準。",
		false,
		"agent",
		"task-02",
		"",
	)

	// 4. Verify immutable field history preserves all 3 versions (AIA-21 §4.5)
	history := b.History["daily_mileage_km"]
	if len(history) != 3 {
		t.Fatalf("expected 3 history entries, got %d", len(history))
	}

	// Version 1
	if history[0].Version != 1 || history[0].NewValue != 20 || history[0].NewStatus != domainBrief.FieldStatusDefaulted {
		t.Errorf("v1 mismatch: %+v", history[0])
	}
	// Version 2
	if history[1].Version != 2 || history[1].OldValue != 20 || history[1].NewValue != 25 || history[1].NewStatus != domainBrief.FieldStatusAnswered {
		t.Errorf("v2 mismatch: %+v", history[1])
	}
	// Version 3
	if history[2].Version != 3 || history[2].OldValue != 25 || history[2].NewValue != 22 || history[2].NewStatus != domainBrief.FieldStatusConfirmed {
		t.Errorf("v3 mismatch: %+v", history[2])
	}
}

func TestGenericBrief_ReadinessAssessment(t *testing.T) {
	catalog, err := domainBrief.GetCatalog("nonfiction_plan")
	if err != nil {
		t.Fatalf("failed to get catalog: %v", err)
	}

	b := domainBrief.NewProjectBrief("brief-readiness-test", "proj-walk-02", "nonfiction_plan", "徒步環台灣本島")

	// Case 1: Empty brief -> NOT_READY due to missing P0 fields
	res1 := domainBrief.AssessReadiness(b, catalog)
	if res1.Status != domainBrief.ReadinessNotReady {
		t.Fatalf("expected NOT_READY for empty brief, got %s", res1.Status)
	}
	if len(res1.MissingP0Fields) == 0 {
		t.Fatalf("expected missing P0 fields to be populated")
	}

	// Case 2: Populate required planning fields with defaults/assumptions -> READY_WITH_ASSUMPTIONS
	for key, def := range catalog.Fields {
		val := def.DefaultValue
		if val == nil {
			val = "測試預設值"
		}
		b.SetField(key, val, domainBrief.FieldStatusDefaulted, "agent_recommendation", 0.7, "暫定方案", true, "agent", "", "")
	}

	res2 := domainBrief.AssessReadiness(b, catalog)
	if res2.Status != domainBrief.ReadinessReadyWithAssumptions {
		t.Fatalf("expected READY_WITH_ASSUMPTIONS when fields are defaulted, got %s (%s)", res2.Status, res2.Rationale)
	}
	if len(res2.PendingAssumes) == 0 {
		t.Fatalf("expected pending assumptions to be listed in assessment")
	}

	// Case 3: User confirms all planning fields -> READY_FOR_PLANNING
	for key, def := range catalog.Fields {
		if def.RequiredForPlanning {
			b.SetField(key, "確認值", domainBrief.FieldStatusConfirmed, "user", 1.0, "使用者確認", false, "user", "", "")
		}
	}
	res3 := domainBrief.AssessReadiness(b, catalog)
	if res3.Status != domainBrief.ReadinessReadyForPlanning && res3.Status != domainBrief.ReadinessReadyForExecution {
		t.Fatalf("expected READY_FOR_PLANNING or READY_FOR_EXECUTION after confirmation, got %s (%s)", res3.Status, res3.Rationale)
	}

	// Case 4: Conflict detected -> NEEDS_REASSESSMENT (AIA-21 §8.2)
	b.SetField("route_scope", "順時針與逆時針衝突", domainBrief.FieldStatusConflicted, "system", 0.0, "路線方向衝突", false, "system", "", "")
	res4 := domainBrief.AssessReadiness(b, catalog)
	if res4.Status != domainBrief.ReadinessNeedsReassessment {
		t.Fatalf("expected NEEDS_REASSESSMENT on conflicted field, got %s", res4.Status)
	}
}

func TestGenericBriefRepository_Save_And_Load(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// 1. Insert parent project
	_, err := db.RawDB().ExecContext(ctx, `
		INSERT INTO projects (id, name, description, workspace, domain_type, status, created_at, updated_at)
		VALUES ('proj-walk-db', '徒步環島', '旅行規劃', '/tmp/walk', 'general', 'ACTIVE', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("failed to insert project: %v", err)
	}

	repo := repository.NewGenericBriefRepository(db)

	b := domainBrief.NewProjectBrief("brief-walk-db-01", "proj-walk-db", "nonfiction_plan", "我想徒步環島")
	b.Generic.Purpose = "完成台灣本島徒步並出版實用指南"
	b.Generic.Audience = "長途徒步愛好者與一般讀者"
	b.SetField("daily_mileage_km", 20, domainBrief.FieldStatusDefaulted, "agent", 0.65, "保守規劃", true, "agent", "", "")
	b.SetField("daily_mileage_km", 25, domainBrief.FieldStatusAnswered, "user", 1.0, "使用者調整", false, "user", "task-1", "cr-1")

	// Save
	if err := repo.Save(ctx, b); err != nil {
		t.Fatalf("failed to save brief: %v", err)
	}

	// Load
	loaded, err := repo.GetByProjectID(ctx, "proj-walk-db")
	if err != nil || loaded == nil {
		t.Fatalf("failed to load brief: %v", err)
	}

	if loaded.ProjectID != "proj-walk-db" || loaded.Domain != "nonfiction_plan" {
		t.Errorf("loaded brief metadata mismatch: %+v", loaded)
	}
	if loaded.Generic.Purpose != "完成台灣本島徒步並出版實用指南" {
		t.Errorf("loaded generic brief purpose mismatch: %s", loaded.Generic.Purpose)
	}

	fs, ok := loaded.GetField("daily_mileage_km")
	if !ok || fs.Status != domainBrief.FieldStatusAnswered {
		t.Errorf("expected field daily_mileage_km to be ANSWERED, got: %+v", fs)
	}

	// Verify history entries from DB
	dbHist, err := repo.ListFieldHistory(ctx, "proj-walk-db", "daily_mileage_km")
	if err != nil || len(dbHist) != 2 {
		t.Fatalf("expected 2 history entries in DB, got %d (err: %v)", len(dbHist), err)
	}
	if dbHist[0].NewStatus != domainBrief.FieldStatusDefaulted || dbHist[1].NewStatus != domainBrief.FieldStatusAnswered {
		t.Errorf("history status sequence mismatch: %+v", dbHist)
	}
}

func TestNonfictionPlan_WalkingTaiwan_AcceptanceScenario(t *testing.T) {
	// Canonical End-to-End Acceptance Scenario (AIA-21 §17):
	// Author Prompt: "我想寫一本關於計劃徒步環台灣本島一圈的計畫的書，請幫我規劃這本書。"
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	wsDir := t.TempDir()
	projRepo := repository.NewProjectRepository(db)
	briefRepo := repository.NewGenericBriefRepository(db)
	svc := briefService.NewService(briefRepo, projRepo)

	projID := "walk-taiwan"
	p := &domainProject.Project{
		ID:          domain.ID(projID),
		Name:        "徒步環台灣本島計畫書",
		Description: "我想寫一本關於計劃徒步環台灣本島一圈的計畫的書，請幫我規劃這本書。",
		Workspace:   wsDir,
		DomainType:  "general",
		Status:      domainProject.StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := projRepo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// 1. Initialize Brief for nonfiction_plan
	rawIntent := "我想寫一本關於計劃徒步環台灣本島一圈的計畫的書，請幫我規劃這本書。"
	brief, err := svc.InitializeBrief(ctx, projID, "nonfiction_plan", rawIntent, wsDir)
	if err != nil {
		t.Fatalf("initialize brief failed: %v", err)
	}

	// 2. Verify brief.yaml was written to workspace root
	briefYAMLPath := filepath.Join(wsDir, "brief.yaml")
	data, err := os.ReadFile(briefYAMLPath)
	if err != nil {
		t.Fatalf("expected brief.yaml in workspace, err: %v", err)
	}
	if !strings.Contains(string(data), "nonfiction_plan") || !strings.Contains(string(data), "daily_mileage_km") {
		t.Fatalf("brief.yaml content incomplete: %s", string(data))
	}

	// 3. Behavioral check: System identifies unknown critical P0 questions (does not prematurely generate chapters)
	if brief.Readiness != domainBrief.ReadinessNotReady {
		t.Fatalf("expected initial readiness NOT_READY, got %s (%s)", brief.Readiness, brief.ReadinessRationale)
	}

	routeField, hasRoute := brief.GetField("route_scope")
	if !hasRoute || routeField.Status != domainBrief.FieldStatusUnknown {
		t.Errorf("expected route_scope to be UNKNOWN, got %+v", routeField)
	}

	// 4. Agent proposes provisional assumption for daily mileage
	_, err = svc.UpdateField(
		ctx,
		projID,
		"daily_mileage_km",
		20,
		domainBrief.FieldStatusDefaulted,
		"agent_recommendation",
		0.65,
		"以一般休閒徒步者為初步規劃對象，先採用較保守的每日 20 公里",
		true,
		"agent",
		"",
		"",
		wsDir,
	)
	if err != nil {
		t.Fatalf("update field failed: %v", err)
	}

	// Verify brief.yaml was updated on disk
	dataUpdated, _ := os.ReadFile(briefYAMLPath)
	if !strings.Contains(string(dataUpdated), "agent_recommendation") || !strings.Contains(string(dataUpdated), "20") {
		t.Fatalf("brief.yaml was not updated on disk: %s", string(dataUpdated))
	}

	// 5. User clarifies route direction -> 順時針
	updatedBrief2, err := svc.UpdateField(
		ctx,
		projID,
		"route_scope",
		"順時針",
		domainBrief.FieldStatusAnswered,
		"user",
		1.0,
		"使用者明確指定採順時針環島，先西後東",
		false,
		"user",
		"",
		"",
		wsDir,
	)
	if err != nil {
		t.Fatalf("update route_scope failed: %v", err)
	}

	fsRoute, _ := updatedBrief2.GetField("route_scope")
	if fsRoute.Value != "順時針" || fsRoute.Status != domainBrief.FieldStatusAnswered {
		t.Errorf("expected route_scope to be ANSWERED with 順時針, got: %+v", fsRoute)
	}
}
