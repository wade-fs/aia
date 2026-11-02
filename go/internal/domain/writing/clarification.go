package writing

import (
	"time"
)

// QuestionCategory categorizes what domain aspect the clarification pertains to.
type QuestionCategory string

const (
	CategoryScale        QuestionCategory = "SCALE"       // 篇幅規模 (總卷數、章節長度)
	CategoryWorldview    QuestionCategory = "WORLDVIEW"   // 世界觀與法則 (外星觀測雙軌、修真體系法則)
	CategoryCharacter    QuestionCategory = "CHARACTER"   // 關鍵人物性格、生卒年或群像定位
	CategoryPlotPolicy   QuestionCategory = "PLOT_POLICY" // 史實骨架約束、非戰場中心原則、藝術虛構邊界
	CategoryGeneral      QuestionCategory = "GENERAL"     // 其他一般設定
)

// ClarificationStatus tracks whether a clarification request is pending or answered.
type ClarificationStatus string

const (
	ClarificationStatusPending  ClarificationStatus = "PENDING"
	ClarificationStatusAnswered ClarificationStatus = "ANSWERED"
	ClarificationStatusSkipped  ClarificationStatus = "SKIPPED"
)

// ClarificationRequest represents a structured inquiry sent to the author via ASK_HUMAN / WAITING_FOR_INPUT.
type ClarificationRequest struct {
	ID         string              `json:"id"`
	ProjectID  string              `json:"project_id"`
	BriefID    string              `json:"brief_id,omitempty"`
	Category   QuestionCategory    `json:"category"`
	Question   string              `json:"question"`          // 具體問題
	Rationale  string              `json:"rationale"`         // 缺口成因與詢問必要性
	Options    []string            `json:"options,omitempty"` // 建議選項 (若有提供選擇)
	UserAnswer string              `json:"user_answer"`       // 使用者回覆內容
	Status     ClarificationStatus `json:"status"`            // PENDING | ANSWERED | SKIPPED
	CreatedAt  time.Time           `json:"created_at"`
	AnsweredAt *time.Time          `json:"answered_at,omitempty"`
}

// IsResolved returns true if the request has been answered or skipped.
func (c *ClarificationRequest) IsResolved() bool {
	return c.Status == ClarificationStatusAnswered || c.Status == ClarificationStatusSkipped
}
