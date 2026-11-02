package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainWriting "github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/service/writing"
	"gopkg.in/yaml.v3"
)

func TestWritingClosedLoop_InceptionClarificationAndConsistency(t *testing.T) {
	appInst, _ := createTestApp(t)
	defer appInst.Close()

	ctx := context.Background()
	wsDir := filepath.Join(t.TempDir(), "novel_yuan_flow")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("failed to create temp workspace: %v", err)
	}

	// =========================================================================
	// Node A & B: 使用者提出創作意圖 ➔ 建立 Project
	// =========================================================================
	rawIntent := "想創作一部以元末脫脫丞相主持治河、整飭軍政為起點的歷史群像長篇小說，" +
		"與朱元璋從底層崛起的軌跡互為照應；同時加入外星文明人類觀測實驗室作為高維冷眼記錄的雙世界觀。"

	initReq := writing.InitiateProjectRequest{
		ProjectID: "proj-yuan-test",
		Title:     "天火流光",
		RawIntent: rawIntent,
		Genre:     "dual_world_historical",
		Workspace: wsDir,
	}

	initResp, err := appInst.WritingWorkflowSvc.InitiateProject(ctx, initReq)
	if err != nil {
		t.Fatalf("Node A/B InitiateProject failed: %v", err)
	}

	// =========================================================================
	// Node C, D, E: 載入 Manifest / Brief ➔ agy 分析需求缺口 ➔ 程式驗證 Brief 欄位
	// Node F, G, H: 是否有重要待釐清事項？ ➔ [有] ➔ 建立 Clarification Requests ➔ ASK_HUMAN
	// =========================================================================
	if initResp.Status != domainWriting.BriefStatusUnderClarification {
		t.Fatalf("expected brief status UNDER_CLARIFICATION, got %s", initResp.Status)
	}
	if len(initResp.ClarificationRequests) < 3 {
		t.Fatalf("expected at least 3 clarification requests (scale, era, worldview), got %d", len(initResp.ClarificationRequests))
	}

	// 驗證 brief.md 已在 workspace 生成
	briefFile := filepath.Join(wsDir, "brief.md")
	if _, err := os.Stat(briefFile); err != nil {
		t.Fatalf("brief.md not found in workspace: %v", err)
	}

	// 取得 SQLite 中的待釐清問題清單
	pending, err := appInst.WritingWorkflowRepo.ListPendingClarifications(ctx, "proj-yuan-test")
	if err != nil || len(pending) == 0 {
		t.Fatalf("failed to retrieve pending clarifications: %v", err)
	}

	// 找出各分類的問題 ID
	var scaleReqID, eraReqID, worldviewReqID string
	for _, cr := range pending {
		switch cr.Category {
		case domainWriting.CategoryScale:
			scaleReqID = cr.ID
		case domainWriting.CategoryPlotPolicy:
			eraReqID = cr.ID
		case domainWriting.CategoryWorldview:
			worldviewReqID = cr.ID
		}
	}
	if scaleReqID == "" || eraReqID == "" || worldviewReqID == "" {
		t.Fatalf("missing required categories: scale=%s, era=%s, worldview=%s", scaleReqID, eraReqID, worldviewReqID)
	}

	// =========================================================================
	// Node I, J: 使用者回答 ➔ 更新 Brief / Project Memory ➔ 回到 Node D 再次分析
	// =========================================================================

	// 回覆 1: 篇幅規模
	scaleAns := "採 6 卷共 540 章（標準史詩體量，每卷 90 章）"
	ansResp1, err := appInst.WritingWorkflowSvc.AnswerClarification(ctx, scaleReqID, scaleAns, wsDir)
	if err != nil {
		t.Fatalf("failed to answer scale clarification: %v", err)
	}
	if ansResp1.Brief.Scale.TotalVolumes != 6 || ansResp1.Brief.Scale.TotalChapters != 540 {
		t.Errorf("expected scale updated to 6 vols / 540 chapters, got vols=%d, chaps=%d",
			ansResp1.Brief.Scale.TotalVolumes, ansResp1.Brief.Scale.TotalChapters)
	}
	if ansResp1.AllClarified {
		t.Errorf("expected not all clarified yet after 1 answer")
	}

	// 回覆 2: 世界觀外星觀測原則
	wvAns := "嚴格不干預原則：外星文明僅作為高維冷眼觀測與記錄者，絕不直接插手或改變歷史進程"
	ansResp2, err := appInst.WritingWorkflowSvc.AnswerClarification(ctx, worldviewReqID, wvAns, wsDir)
	if err != nil {
		t.Fatalf("failed to answer worldview clarification: %v", err)
	}
	if ansResp2.Brief.WorldviewSettings["non_interference"] != true {
		t.Errorf("expected worldview non_interference=true")
	}

	// 回覆 3: 歷史年代起訖
	eraAns := "元末至正元年（1341）至至正二十八年（1368）"
	ansResp3, err := appInst.WritingWorkflowSvc.AnswerClarification(ctx, eraReqID, eraAns, wsDir)
	if err != nil {
		t.Fatalf("failed to answer era clarification: %v", err)
	}

	// =========================================================================
	// Node F (第二次判定): 是否有重要待釐清事項？ ➔ [沒有]
	// Node K: 建立創作定位與全書大綱 (book.yaml, outline.md, master_bible.md, characters.md)
	// =========================================================================
	if !ansResp3.AllClarified {
		t.Fatalf("expected all clarified after all 3 answers")
	}
	if ansResp3.Brief.Status != domainWriting.BriefStatusApproved {
		t.Errorf("expected brief status APPROVED, got %s", ansResp3.Brief.Status)
	}
	if !ansResp3.OutlineEstablished {
		t.Errorf("expected OutlineEstablished=true upon resolving all clarifications")
	}

	// 檢查 Node K 生成的核心資產文件
	bookYamlPath := filepath.Join(wsDir, "book.yaml")
	outlinePath := filepath.Join(wsDir, "outline.md")
	biblePath := filepath.Join(wsDir, "master_bible.md")
	charsPath := filepath.Join(wsDir, "characters.md")

	for _, p := range []string{bookYamlPath, outlinePath, biblePath, charsPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected generated file %s not found: %v", p, err)
		}
	}

	// 檢驗 book.yaml 規格與約束
	manifestBytes, err := os.ReadFile(bookYamlPath)
	if err != nil {
		t.Fatalf("failed to read book.yaml: %v", err)
	}
	var manifest domainWriting.BookManifest
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("failed to parse generated book.yaml: %v", err)
	}

	if manifest.Scale.TotalVolumes != 6 || manifest.Scale.TotalChapters != 540 {
		t.Errorf("manifest scale mismatch: %+v", manifest.Scale)
	}
	if manifest.Laboratory == nil || !manifest.Laboratory.NonInterference {
		t.Errorf("manifest laboratory non-interference setting missing: %+v", manifest.Laboratory)
	}
	if !manifest.WritingPolicy.PreserveHistoricalFigures || !manifest.WritingPolicy.RequireConsistencyCheck {
		t.Errorf("manifest writing policy constraints missing: %+v", manifest.WritingPolicy)
	}

	// 檢驗 outline.md 是否包含 6 卷結構
	outlineContent, _ := os.ReadFile(outlinePath)
	if !strings.Contains(string(outlineContent), "第 6 卷") {
		t.Errorf("outline.md does not contain Volume 6")
	}

	// =========================================================================
	// Node L, M, N: 分段研究與事實查證 ➔ 產生初稿 ➔ 一致性檢查與修訂 (ConsistencyEngine)
	// =========================================================================
	// 建立一致性分卷章節 volume1.md (史實一致：至正五年 1345 年，脫脫巡視河工)
	vol1Path := filepath.Join(wsDir, "volume1.md")
	vol1Content := `
#### 第1章 治河之始
- **時地**：至正五年（1345年），黃河沿線
- **視角**：脫脫
- **標記**：【史實骨架】

至正五年春，右丞相脫脫親赴黃河故道，巡視水患與民變流民……
`
	_ = os.WriteFile(vol1Path, []byte(vol1Content), 0644)

	// 使用 ConsistencyEngine 進行初次掃描
	engine := domainWriting.NewConsistencyEngine()
	findings1, err := engine.ScanProject(wsDir, nil)
	if err != nil {
		t.Fatalf("ScanProject failed: %v", err)
	}
	if len(findings1) > 0 {
		t.Fatalf("expected 0 anomalies for consistent chapter 1, got %d: %+v", len(findings1), findings1)
	}

	// 注入史實衝突章節 (故意在 1360 年讓已於 1355 年被賜死的脫脫登場)
	vol1Contradiction := vol1Content + `
#### 第50章 違背史實之登場
- **時地**：至正二十年（1360年），大都中書省
- **視角**：脫脫
- **標記**：【史實骨架】

至正二十年秋，脫脫丞相在大都中書省召集百官議事……
`
	_ = os.WriteFile(vol1Path, []byte(vol1Contradiction), 0644)

	findings2, err := engine.ScanProject(wsDir, nil)
	if err != nil {
		t.Fatalf("ScanProject with contradiction failed: %v", err)
	}

	// 檢驗 Node N 是否成功捕捉到 POST_MORTEM_APPEARANCE (卒後登場)
	if len(findings2) == 0 {
		t.Fatalf("expected consistency engine to detect post-mortem contradiction, but got 0 findings")
	}

	foundPostMortem := false
	for _, f := range findings2 {
		if f.AnomalyType == domainWriting.AnomalyPostMortemAppearance && f.CharacterName == "脫脫" {
			foundPostMortem = true
			if f.DeathYear != 1355 || f.StatedYear != 1360 {
				t.Errorf("anomaly year bounds mismatch: death=%d, stated=%d", f.DeathYear, f.StatedYear)
			}
		}
	}
	if !foundPostMortem {
		t.Errorf("expected to find POST_MORTEM_APPEARANCE for 脫脫 in 1360, got: %+v", findings2)
	}

	// =========================================================================
	// Node O, P: 評估進度與下一步 ➔ 觸發修訂建議 (ErratumProposal)
	// =========================================================================
	proposal := domainWriting.ErratumProposal{
		AnomalyID:       findings2[0].ID,
		ChapterRef:      "volume1.md (第50章)",
		OriginalText:    "至正二十年秋，脫脫丞相在大都中書省召集百官議事",
		ProposedFix:     "至正二十年秋，中書省百官追憶故丞相脫脫當年在位之治績",
		HistoricalBasis: "脫脫已於至正十五年（1355年）被貶吐蕃途中於雲南遭鴆殺，1360年不得在朝議事",
	}

	if proposal.HistoricalBasis == "" || proposal.ProposedFix == "" {
		t.Errorf("erratum proposal was not properly formulated")
	}
}
