package writing

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BriefStatus tracks the lifecycle status of a creative brief.
type BriefStatus string

const (
	BriefStatusDraft              BriefStatus = "DRAFT"               // Initial draft created from raw intent
	BriefStatusUnderClarification BriefStatus = "UNDER_CLARIFICATION" // Waiting for human clarification of key questions
	BriefStatusApproved           BriefStatus = "APPROVED"            // Fully clarified and validated, ready for outline & production
)

// CreativeBrief represents the high-level authorial specification and creative brief for a writing project.
type CreativeBrief struct {
	ID                 string         `json:"id"`
	ProjectID          string         `json:"project_id"`
	Title              string         `json:"title"`
	RawIntent          string         `json:"raw_intent"`           // Node A: User's original creative intent
	Genre              string         `json:"genre"`                // Genre / Profile
	Era                string         `json:"era,omitempty"`        // Historical or narrative era (e.g. 元末 1340-1368)
	ThemeAndPhilosophy []string       `json:"theme_and_philosophy"` // Core themes and philosophy
	Scale              BookScale      `json:"scale"`                // TotalVolumes, TotalChapters, etc.
	CoreProtagonists   []string       `json:"core_protagonists"`    // Main characters and roles
	WorldviewSettings  map[string]any `json:"worldview_settings"`   // Dual world / lab / xianxia / sci-fi settings
	WritingRules       []string       `json:"writing_rules"`        // 3-tier rules, perspective constraints, etc.
	Version            int            `json:"version"`              // Brief version incremented on each revision
	Status             BriefStatus    `json:"status"`
	PendingQuestions   int            `json:"pending_questions"` // Number of unanswered clarification requests
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// ToMarkdown generates a structured, human-readable brief.md document for the project workspace.
func (b *CreativeBrief) ToMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 《%s》創作企劃綱要 (Creative Brief)\n\n", b.Title))
	sb.WriteString(fmt.Sprintf("- **狀態**：%s\n", b.Status))
	sb.WriteString(fmt.Sprintf("- **類型**：%s\n", b.Genre))
	if b.Era != "" {
		sb.WriteString(fmt.Sprintf("- **時代背景**：%s\n", b.Era))
	}
	sb.WriteString(fmt.Sprintf("- **規模規劃**：%d 卷共 %d 章（每卷約 %d 章）\n",
		b.Scale.TotalVolumes, b.Scale.TotalChapters, b.Scale.ChaptersPerVolume))
	sb.WriteString(fmt.Sprintf("- **最後更新**：%s\n\n", b.UpdatedAt.Format(time.RFC3339)))

	sb.WriteString("## 1. 原始創作意圖 (User Intent)\n\n")
	sb.WriteString(strings.TrimSpace(b.RawIntent))
	sb.WriteString("\n\n")

	sb.WriteString("## 2. 核心主旨與哲學原則 (Theme & Philosophy)\n\n")
	if len(b.ThemeAndPhilosophy) == 0 {
		sb.WriteString("*(尚未設定)*\n\n")
	} else {
		for _, theme := range b.ThemeAndPhilosophy {
			sb.WriteString(fmt.Sprintf("- %s\n", theme))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 3. 主要人物與群像架構 (Core Protagonists)\n\n")
	if len(b.CoreProtagonists) == 0 {
		sb.WriteString("*(尚未設定)*\n\n")
	} else {
		for _, p := range b.CoreProtagonists {
			sb.WriteString(fmt.Sprintf("- %s\n", p))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 4. 寫作規範與約束 (Writing Policy)\n\n")
	if len(b.WritingRules) == 0 {
		sb.WriteString("*(尚未設定)*\n\n")
	} else {
		for _, rule := range b.WritingRules {
			sb.WriteString(fmt.Sprintf("- %s\n", rule))
		}
		sb.WriteString("\n")
	}

	if len(b.WorldviewSettings) > 0 {
		sb.WriteString("## 5. 世界觀特殊設定 (Worldview & Perspective)\n\n")
		wBytes, _ := json.MarshalIndent(b.WorldviewSettings, "", "  ")
		sb.WriteString("```json\n")
		sb.WriteString(string(wBytes))
		sb.WriteString("\n```\n\n")
	}

	return sb.String()
}
