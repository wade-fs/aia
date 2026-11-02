package unit

import (
	"context"
	"testing"
	"time"

	domainWriting "github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/service/writing"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestBriefValidator_DeterministicRules(t *testing.T) {
	// Case 1: Empty brief
	emptyBrief := &domainWriting.CreativeBrief{}
	res := domainWriting.ValidateBrief(emptyBrief)
	if res.IsValid {
		t.Errorf("expected empty brief to be invalid")
	}
	if len(res.MissingFields) < 3 {
		t.Errorf("expected at least 3 missing fields, got %d", len(res.MissingFields))
	}

	// Case 2: Historical fiction without Era
	histBrief := &domainWriting.CreativeBrief{
		Title:     "天命有歸",
		RawIntent: "寫一本元末明初的小說",
		Genre:     "historical_fiction",
		Scale: domainWriting.BookScale{
			TotalVolumes:      6,
			TotalChapters:     540,
			ChaptersPerVolume: 90,
		},
	}
	resHist := domainWriting.ValidateBrief(histBrief)
	if resHist.IsValid {
		t.Errorf("expected historical brief without era to be invalid")
	}
	hasEraMissing := false
	for _, f := range resHist.MissingFields {
		if f == "era (歷史年代區間，如唐末、宋初、元末)" {
			hasEraMissing = true
		}
	}
	if !hasEraMissing {
		t.Errorf("expected missing era field flag, got %v", resHist.MissingFields)
	}

	// Case 3: Scale Arithmetic conflict
	conflictBrief := &domainWriting.CreativeBrief{
		Title:     "矛盾測試",
		RawIntent: "測試規模算術矛盾",
		Genre:     "general",
		Scale: domainWriting.BookScale{
			TotalVolumes:      6,
			TotalChapters:     500, // Should be 6 * 90 = 540
			ChaptersPerVolume: 90,
		},
	}
	resConflict := domainWriting.ValidateBrief(conflictBrief)
	if resConflict.IsValid {
		t.Errorf("expected scale mismatch to be invalid")
	}
	if len(resConflict.StructuralDefects) == 0 {
		t.Errorf("expected structural defect for scale mismatch")
	}

	// Case 4: Completely valid brief
	validBrief := &domainWriting.CreativeBrief{
		Title:     "長安秋暮",
		RawIntent: "唐末五代歷史群像",
		Genre:     "historical_fiction",
		Era:       "唐末五代 (875-979)",
		Scale: domainWriting.BookScale{
			TotalVolumes:      16,
			TotalChapters:     1440,
			ChaptersPerVolume: 90,
		},
		ThemeAndPhilosophy: []string{"非戰場中心原則"},
		CoreProtagonists:   []string{"黃巢", "李克用", "朱溫"},
	}
	resValid := domainWriting.ValidateBrief(validBrief)
	if !resValid.IsValid || resValid.RequiresClarification {
		t.Errorf("expected valid brief to pass, got defects=%v missing=%v", resValid.StructuralDefects, resValid.MissingFields)
	}
}

func TestWritingWorkflowRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := repository.NewWritingWorkflowRepository(db)

	// First insert a parent project to satisfy foreign key
	_, err := db.RawDB().ExecContext(ctx, `
		INSERT INTO projects (id, name, description, workspace, domain_type, status, created_at, updated_at)
		VALUES ('proj-yuan', '元末風雲', 'desc', '/tmp/yuan', 'writing', 'ACTIVE', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("failed to insert test project: %v", err)
	}

	// 1. Brief CRUD
	brief := &domainWriting.CreativeBrief{
		ID:        "brief-001",
		ProjectID: "proj-yuan",
		Title:     "元末風雲",
		RawIntent: "描寫脫脫與朱元璋",
		Genre:     "historical_fiction",
		Era:       "元末 (1340-1368)",
		Scale: domainWriting.BookScale{
			TotalVolumes:      6,
			TotalChapters:     540,
			ChaptersPerVolume: 90,
		},
		Status:           domainWriting.BriefStatusDraft,
		PendingQuestions: 2,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	if err := repo.CreateBrief(ctx, brief); err != nil {
		t.Fatalf("CreateBrief failed: %v", err)
	}

	fetched, err := repo.GetBriefByProjectID(ctx, "proj-yuan")
	if err != nil || fetched == nil {
		t.Fatalf("GetBriefByProjectID failed: %v", err)
	}
	if fetched.Title != "元末風雲" || fetched.Scale.TotalChapters != 540 {
		t.Errorf("unexpected brief values: %+v", fetched)
	}

	fetched.Status = domainWriting.BriefStatusApproved
	fetched.PendingQuestions = 0
	if err := repo.UpdateBrief(ctx, fetched); err != nil {
		t.Fatalf("UpdateBrief failed: %v", err)
	}

	updated, _ := repo.GetBriefByID(ctx, "brief-001")
	if updated.Status != domainWriting.BriefStatusApproved || updated.PendingQuestions != 0 {
		t.Errorf("brief status update failed: %+v", updated)
	}

	// 2. Clarification CRUD
	c1 := &domainWriting.ClarificationRequest{
		ID:        "clarify-1",
		ProjectID: "proj-yuan",
		BriefID:   "brief-001",
		Category:  domainWriting.CategoryScale,
		Question:  "全書預計規模？",
		Rationale: "需要確定分卷總帳",
		Options:   []string{"6 卷 540 章", "8 卷 720 章"},
		Status:    domainWriting.ClarificationStatusPending,
		CreatedAt: time.Now().UTC(),
	}
	c2 := &domainWriting.ClarificationRequest{
		ID:        "clarify-2",
		ProjectID: "proj-yuan",
		BriefID:   "brief-001",
		Category:  domainWriting.CategoryWorldview,
		Question:  "外星實驗室觀測互動原則？",
		Rationale: "維持正史嚴肅性",
		Options:   []string{"不干預原則"},
		Status:    domainWriting.ClarificationStatusPending,
		CreatedAt: time.Now().UTC(),
	}

	if err := repo.CreateClarification(ctx, c1); err != nil {
		t.Fatalf("CreateClarification c1 failed: %v", err)
	}
	if err := repo.CreateClarification(ctx, c2); err != nil {
		t.Fatalf("CreateClarification c2 failed: %v", err)
	}

	all, err := repo.ListClarificationsByProjectID(ctx, "proj-yuan")
	if err != nil || len(all) != 2 {
		t.Fatalf("expected 2 clarifications, got %d, err=%v", len(all), err)
	}

	pending, err := repo.ListPendingClarifications(ctx, "proj-yuan")
	if err != nil || len(pending) != 2 {
		t.Fatalf("expected 2 pending clarifications, got %d", len(pending))
	}

	// Answer c1
	c1.UserAnswer = "6 卷共 540 章"
	c1.Status = domainWriting.ClarificationStatusAnswered
	now := time.Now().UTC()
	c1.AnsweredAt = &now
	if err := repo.UpdateClarification(ctx, c1); err != nil {
		t.Fatalf("UpdateClarification failed: %v", err)
	}

	pendingAfter, _ := repo.ListPendingClarifications(ctx, "proj-yuan")
	if len(pendingAfter) != 1 || pendingAfter[0].ID != "clarify-2" {
		t.Errorf("expected 1 remaining pending clarification (clarify-2), got %d", len(pendingAfter))
	}
}

func TestGapAnalyzer_DetectGaps(t *testing.T) {
	analyzer := writing.NewGapAnalyzer(nil)
	ctx := context.Background()

	// Initial rough brief from user
	brief := &domainWriting.CreativeBrief{
		ID:        "brief-test",
		ProjectID: "proj-test",
		Title:     "草莽龍蛇",
		RawIntent: "寫一本元末至正年間的小說，包含外星文明觀測實驗室雙世界觀",
		Genre:     "dual_world_historical",
		Status:    domainWriting.BriefStatusDraft,
	}

	outcome, err := analyzer.AnalyzeIntentAndBrief(ctx, brief)
	if err != nil {
		t.Fatalf("AnalyzeIntentAndBrief failed: %v", err)
	}

	if !outcome.RequiresHumanClarify {
		t.Errorf("expected brief with missing scale/era/worldview to require human clarification")
	}

	if len(outcome.ClarificationRequests) < 3 {
		t.Errorf("expected at least 3 clarification requests, got %d", len(outcome.ClarificationRequests))
	}

	categories := make(map[domainWriting.QuestionCategory]bool)
	for _, cr := range outcome.ClarificationRequests {
		categories[cr.Category] = true
	}

	if !categories[domainWriting.CategoryScale] {
		t.Errorf("expected CategoryScale clarification")
	}
	if !categories[domainWriting.CategoryPlotPolicy] {
		t.Errorf("expected CategoryPlotPolicy (era) clarification")
	}
	if !categories[domainWriting.CategoryWorldview] {
		t.Errorf("expected CategoryWorldview clarification")
	}
}
