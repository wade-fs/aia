package writing

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

type KnowledgeTier string

const (
	TierFactHistory        KnowledgeTier = "FACT_HISTORY"        // 【史實骨架】：無可爭辯的正史記載
	TierPlausibleInference KnowledgeTier = "PLAUSIBLE_INFERENCE" // 【合理推演】：符合時代民俗地理之推論
	TierFiction            KnowledgeTier = "FICTION"             // 【小說虛構】：藝術創作、基層原創人物、方外穿線
)

type AnomalyType string

const (
	AnomalyPostMortemAppearance AnomalyType = "POST_MORTEM_APPEARANCE" // 卒後登場
	AnomalyAnachronism          AnomalyType = "ANACHRONISM"           // 時代錯置 (如提早使用未出現年號或名詞)
	AnomalyLocationConflict     AnomalyType = "LOCATION_CONFLICT"      // 瞬間移動 (無合理行程時間之地理衝突)
	AnomalyCharacterMismatch    AnomalyType = "CHARACTER_MISMATCH"     // 設定矛盾 (年齡、身分、親緣關係相悖)
	AnomalyFictionDeviation     AnomalyType = "FICTION_DEVIATION"      // 虛構設定偏離
)

// HistoricalCharacter represents a character profile tracked across volumes (e.g. in characters.md).
type HistoricalCharacter struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Aliases       []string      `json:"aliases,omitempty"`
	BirthYear     int           `json:"birth_year,omitempty"`
	DeathYear     int           `json:"death_year,omitempty"`
	HistoricalRole string       `json:"historical_role"`
	KnowledgeTier KnowledgeTier `json:"knowledge_tier"`
	KeyEvents     []string      `json:"key_events,omitempty"`
}

// TimelineEvent represents an anchored historical or narrative milestone (e.g. in outline.md).
type TimelineEvent struct {
	ID                 string        `json:"id"`
	Year               int           `json:"year"`
	Month              int           `json:"month,omitempty"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	Location           string        `json:"location"`
	RelevantCharacters []string      `json:"relevant_characters"`
	KnowledgeTier      KnowledgeTier `json:"knowledge_tier"`
}

// ConsistencyAnomaly represents a chronological or character violation detected in a manuscript.
type ConsistencyAnomaly struct {
	ID                  string        `json:"id"`
	AnomalyType         AnomalyType   `json:"anomaly_type"`
	CharacterName       string        `json:"character_name"`
	ChapterRef          string        `json:"chapter_ref"`
	StatedYear          int           `json:"stated_year"`
	ExpectedYear        int           `json:"expected_year,omitempty"`
	DeathYear           int           `json:"death_year,omitempty"`
	Description         string        `json:"description"`
	KnowledgeTier       KnowledgeTier `json:"knowledge_tier"`
	IsDeliberateFiction bool          `json:"is_deliberate_fiction"` // 若為刻意小說虛構，Deliberation 應判定 NO_ACTION
	Severity            int           `json:"severity"`              // 1~10
}

// ManuscriptProject links to a novel codebase (e.g. novel/yuan/ under the AIA project root).
type ManuscriptProject struct {
	ID             domain.ID `json:"id"`
	Name           string    `json:"name"`
	BasePath       string    `json:"base_path"`
	VolumeCount    int       `json:"volume_count"`
	OutlineFile    string    `json:"outline_file"`
	CharactersFile string    `json:"characters_file"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ErratumProposal represents an actionable revision proposal to fix a manuscript contradiction.
type ErratumProposal struct {
	ID              domain.ID `json:"id"`
	AnomalyID       string    `json:"anomaly_id"`
	ChapterRef      string    `json:"chapter_ref"`
	OriginalText    string    `json:"original_text"`
	ProposedFix     string    `json:"proposed_fix"`
	HistoricalBasis string    `json:"historical_basis"`
	CreatedAt       time.Time `json:"created_at"`
}

// BookManifest represents a book.yaml project entrypoint (thinking.md §一 1).
// It defines the book structure, writing policy classifications, and authority levels.
type BookManifest struct {
	Project struct {
		ID       string `yaml:"id" json:"id"`
		Name     string `yaml:"name" json:"name"`
		Type     string `yaml:"type" json:"type"`
		Language string `yaml:"language" json:"language"`
	} `yaml:"project" json:"project"`

	Structure struct {
		Outline       string `yaml:"outline" json:"outline"`
		Timeline      string `yaml:"timeline,omitempty" json:"timeline,omitempty"`
		TimelineAudit string `yaml:"timeline_audit,omitempty" json:"timeline_audit,omitempty"`
		Volumes       string `yaml:"volumes,omitempty" json:"volumes,omitempty"`
		Chapters      string `yaml:"chapters,omitempty" json:"chapters,omitempty"`
		Characters    string `yaml:"characters,omitempty" json:"characters,omitempty"`
		Research      string `yaml:"research,omitempty" json:"research,omitempty"`
		Consistency   string `yaml:"consistency,omitempty" json:"consistency,omitempty"`
	} `yaml:"structure" json:"structure"`

	WritingPolicy struct {
		HistoricalAccuracy        string   `yaml:"historical_accuracy" json:"historical_accuracy"`
		BattleTacticsFocus        bool     `yaml:"battle_tactics_focus" json:"battle_tactics_focus"`
		Classifications           []string `yaml:"classifications" json:"classifications"`
		PreserveHistoricalFigures bool     `yaml:"preserve_historical_figures" json:"preserve_historical_figures"`
		RequireConsistencyCheck   bool     `yaml:"require_consistency_check" json:"require_consistency_check"`
	} `yaml:"writing_policy" json:"writing_policy"`

	Authority struct {
		ConfirmedSettings   string `yaml:"confirmed_settings" json:"confirmed_settings"`
		HistoricalFacts     string `yaml:"historical_facts" json:"historical_facts"`
		UnresolvedQuestions string `yaml:"unresolved_questions" json:"unresolved_questions"`
	} `yaml:"authority" json:"authority"`
}
