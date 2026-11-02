package writing

import (
	"fmt"
	"strings"
)

// BriefValidationResult encapsulates the programmatic verification findings of a CreativeBrief.
type BriefValidationResult struct {
	IsValid               bool     `json:"is_valid"`
	RequiresClarification bool     `json:"requires_clarification"` // Node F: 是否有重要待釐清事項？
	MissingFields         []string `json:"missing_fields"`         // 缺漏欄位 (如 era, scale)
	StructuralDefects     []string `json:"structural_defects"`     // 結構缺陷或數值矛盾
}

// ValidateBrief performs pure programmatic, deterministic verification on a CreativeBrief.
// It verifies field completeness, scale arithmetic consistency, and genre-specific constraints
// without consuming LLM tokens.
func ValidateBrief(brief *CreativeBrief) BriefValidationResult {
	res := BriefValidationResult{
		IsValid:               true,
		RequiresClarification: false,
		MissingFields:         make([]string, 0),
		StructuralDefects:     make([]string, 0),
	}

	if strings.TrimSpace(brief.Title) == "" {
		res.MissingFields = append(res.MissingFields, "title (書名/專案名稱)")
	}

	if strings.TrimSpace(brief.RawIntent) == "" {
		res.MissingFields = append(res.MissingFields, "raw_intent (原始創作意圖)")
	}

	if strings.TrimSpace(brief.Genre) == "" {
		res.MissingFields = append(res.MissingFields, "genre (專案作品類型)")
	}

	// 規模驗證 (Scale arithmetic)
	scale := brief.Scale
	if scale.TotalVolumes <= 0 {
		res.MissingFields = append(res.MissingFields, "scale.total_volumes (總卷數)")
	}
	if scale.ChaptersPerVolume <= 0 {
		res.MissingFields = append(res.MissingFields, "scale.chapters_per_volume (每卷章數)")
	}
	if scale.TotalChapters <= 0 && scale.TotalVolumes > 0 && scale.ChaptersPerVolume > 0 {
		// 自動補齊
		brief.Scale.TotalChapters = scale.TotalVolumes * scale.ChaptersPerVolume
	} else if scale.TotalVolumes > 0 && scale.ChaptersPerVolume > 0 && scale.TotalChapters > 0 {
		expectedChapters := scale.TotalVolumes * scale.ChaptersPerVolume
		if scale.TotalChapters != expectedChapters {
			res.StructuralDefects = append(res.StructuralDefects,
				fmt.Sprintf("scale arithmetic conflict: total_chapters (%d) != total_volumes (%d) * chapters_per_volume (%d)",
					scale.TotalChapters, scale.TotalVolumes, scale.ChaptersPerVolume))
		}
	}

	// 類型特化驗證 (Genre specific rules)
	genreLower := strings.ToLower(brief.Genre)

	if strings.Contains(genreLower, "historical") {
		if strings.TrimSpace(brief.Era) == "" {
			res.MissingFields = append(res.MissingFields, "era (歷史年代區間，如唐末、宋初、元末)")
		}
	}

	if strings.Contains(genreLower, "dual_world") || strings.Contains(genreLower, "laboratory") {
		if brief.WorldviewSettings == nil || len(brief.WorldviewSettings) == 0 {
			res.MissingFields = append(res.MissingFields, "worldview_settings.perspective (外星/觀測實驗室雙世界設定)")
		}
	}

	if strings.Contains(genreLower, "xianxia") {
		if len(brief.ThemeAndPhilosophy) == 0 {
			res.MissingFields = append(res.MissingFields, "theme_and_philosophy (修真/玄幻核心哲學原則，如以失換得)")
		}
	}

	if len(res.MissingFields) > 0 || len(res.StructuralDefects) > 0 {
		res.IsValid = false
		res.RequiresClarification = true
	}

	return res
}
