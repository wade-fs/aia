package writing

import (
	"context"
	"fmt"
	"strings"

	domainWriting "github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/provider"
)

// GapAnalyzer coordinates Node D (agy 分析需求缺口) and Node E (程式驗證 Brief 欄位)
// to detect gaps and create structured ClarificationRequests (Node G).
type GapAnalyzer struct {
	provider provider.Provider
}

func NewGapAnalyzer(p provider.Provider) *GapAnalyzer {
	return &GapAnalyzer{provider: p}
}

// AnalysisOutcome represents the evaluated gaps and generated clarification requests.
type AnalysisOutcome struct {
	ValidationResult       domainWriting.BriefValidationResult
	ClarificationRequests []*domainWriting.ClarificationRequest
	RequiresHumanClarify   bool
}

// AnalyzeIntentAndBrief evaluates a CreativeBrief programmatically and heuristically.
func (g *GapAnalyzer) AnalyzeIntentAndBrief(ctx context.Context, brief *domainWriting.CreativeBrief) (*AnalysisOutcome, error) {
	// Node E: 程式驗證 Brief 欄位 (Programmatic verification)
	vResult := domainWriting.ValidateBrief(brief)

	var reqs []*domainWriting.ClarificationRequest

	// 1. Scale Clarification (篇幅規模缺漏)
	if brief.Scale.TotalVolumes <= 0 || brief.Scale.ChaptersPerVolume <= 0 {
		reqs = append(reqs, &domainWriting.ClarificationRequest{
			ID:        fmt.Sprintf("clarify-scale-%d", len(reqs)+1),
			ProjectID: brief.ProjectID,
			BriefID:   brief.ID,
			Category:  domainWriting.CategoryScale,
			Question:  "請問全書預計的篇幅規模與卷期劃分是？",
			Rationale: "確立全書分卷與總章數總帳（chapter_ledger.md），是劃分各卷年代跨度與人物弧光的前提。",
			Options: []string{
				"(Recommended) 6 卷共 540 章（標準史詩體量，每卷 90 章）",
				"8 卷共 720 章（大篇幅歷史群像，每卷 90 章）",
				"16 卷共 1440 章（超長篇巨著，每卷 90 章）",
				"3 卷共 270 章（精煉三部曲，每卷 90 章）",
			},
			Status: domainWriting.ClarificationStatusPending,
		})
	}

	// 2. Era / Timeline Clarification (歷史時代缺漏)
	genreLower := strings.ToLower(brief.Genre)
	if strings.Contains(genreLower, "historical") && strings.TrimSpace(brief.Era) == "" {
		reqs = append(reqs, &domainWriting.ClarificationRequest{
			ID:        fmt.Sprintf("clarify-era-%d", len(reqs)+1),
			ProjectID: brief.ProjectID,
			BriefID:   brief.ID,
			Category:  domainWriting.CategoryPlotPolicy,
			Question:  "請問小說展開的具體歷史年代區間是？",
			Rationale: "歷史小說需嚴格鎖定年代範圍，以載入歷史人物生卒約束（如生卒年限）並杜絕時代錯置（Anachronism）。",
			Options: []string{
				"(Recommended) 元末至明初（至正元年 1341 年至洪武元年 1368 年）",
				"唐末五代（黃巢起義 875 年至宋初 979 年）",
				"南宋末年（端平入洛 1234 年至崖山海戰 1279 年）",
			},
			Status: domainWriting.ClarificationStatusPending,
		})
	}

	// 3. Worldview / Laboratory Perspective (雙世界觀/外星觀測視角設定缺漏)
	if (strings.Contains(genreLower, "dual_world") || strings.Contains(genreLower, "laboratory")) &&
		(brief.WorldviewSettings == nil || len(brief.WorldviewSettings) == 0) {
		reqs = append(reqs, &domainWriting.ClarificationRequest{
			ID:        fmt.Sprintf("clarify-worldview-%d", len(reqs)+1),
			ProjectID: brief.ProjectID,
			BriefID:   brief.ID,
			Category:  domainWriting.CategoryWorldview,
			Question:  "請問外星文明觀測實驗室與人類歷史正文的互動原則為何？",
			Rationale: "雙世界架構必須確立觀測者是否干預歷史進程，以維持歷史骨架的真實性與嚴肅性。",
			Options: []string{
				"(Recommended) 嚴格不干預原則：外星文明僅作為高維冷眼觀測與記錄者，絕不直接插手或改變歷史進程",
				"間接擾動：僅記錄歷史人物靈魂波長或認知異化，正文維持完全正史演進",
			},
			Status: domainWriting.ClarificationStatusPending,
		})
	}

	// 4. Xianxia Cultivation Principles (仙俠/玄幻核心哲學原則缺漏)
	if strings.Contains(genreLower, "xianxia") && len(brief.ThemeAndPhilosophy) == 0 {
		reqs = append(reqs, &domainWriting.ClarificationRequest{
			ID:        fmt.Sprintf("clarify-xianxia-%d", len(reqs)+1),
			ProjectID: brief.ProjectID,
			BriefID:   brief.ID,
			Category:  domainWriting.CategoryWorldview,
			Question:  "請問本作修真體系的核心哲學原則為何？",
			Rationale: "修真作品需要明確的力量代價法則，避免數值膨脹與無意義戰力階梯。",
			Options: []string{
				"(Recommended) 以失換得（強大是缺了些什麼的結果，存在本質轉換，非數值化堆疊）",
				"求理與應對（求理問為什麼會這樣，應對問怎麼活下去）",
			},
			Status: domainWriting.ClarificationStatusPending,
		})
	}

	// 5. Protagonists / Core Focus (主角群像與核心衝突缺漏)
	if len(brief.CoreProtagonists) == 0 {
		for _, fig := range []string{"脫脫", "朱元璋", "黃巢", "李克用", "朱溫", "岳飛", "文天祥", "于謙", "賈魯"} {
			if strings.Contains(brief.RawIntent, fig) {
				brief.CoreProtagonists = append(brief.CoreProtagonists, fig)
			}
		}
	}
	if len(brief.CoreProtagonists) == 0 && len(reqs) < 3 {
		reqs = append(reqs, &domainWriting.ClarificationRequest{
			ID:        fmt.Sprintf("clarify-character-%d", len(reqs)+1),
			ProjectID: brief.ProjectID,
			BriefID:   brief.ID,
			Category:  domainWriting.CategoryCharacter,
			Question:  "請問主角群與視角人物的核心衝突定位為何？",
			Rationale: "確立多重視角人物（如朝廷權臣、地方義軍、隱居方士、外星實驗員）能建構多維度的歷史群像故事。",
			Options: []string{
				"(Recommended) 雙雄或多線並進：朝廷改革者（如脫脫）與底層崛起者（如朱元璋）相互映襯",
				"方外遊歷穿線：以虛構方外道人/醫者行走天下，冷眼見證王朝崩解與新生",
			},
			Status: domainWriting.ClarificationStatusPending,
		})
	}

	requiresHuman := len(reqs) > 0 || vResult.RequiresClarification

	return &AnalysisOutcome{
		ValidationResult:       vResult,
		ClarificationRequests: reqs,
		RequiresHumanClarify:   requiresHuman,
	}, nil
}
