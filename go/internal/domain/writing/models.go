package writing

import (
	"fmt"
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

// BookManifest represents a book.yaml project entrypoint (thinking.md §五, AIA-20).
// It defines the comprehensive book structure, scales, dual-world/laboratory perspective,
// writing policies, authority/adjudication rules, and project governance.
type BookManifest struct {
	Project        ProjectMeta        `yaml:"project" json:"project"`
	Scale          *BookScale         `yaml:"scale,omitempty" json:"scale,omitempty"`
	Structure      BookStructure      `yaml:"structure" json:"structure"`
	Laboratory     *LaboratoryConfig  `yaml:"laboratory,omitempty" json:"laboratory,omitempty"`
	Worldview      *WorldviewConfig   `yaml:"worldview,omitempty" json:"worldview,omitempty"`
	WritingPolicy  BookWritingPolicy  `yaml:"writing_policy" json:"writing_policy"`
	Authority      BookAuthority      `yaml:"authority" json:"authority"`
	Storage        *ProjectStorage    `yaml:"storage,omitempty" json:"storage,omitempty"`
	VersionControl *ProjectVC         `yaml:"version_control,omitempty" json:"version_control,omitempty"`
	Governance     *ProjectGovernance `yaml:"governance,omitempty" json:"governance,omitempty"`
	Continuity     *ProjectContinuity `yaml:"continuity,omitempty" json:"continuity,omitempty"`
	Memory         *ProjectMemory     `yaml:"memory,omitempty" json:"memory,omitempty"`
}

// ProjectMeta describes core book/manuscript metadata.
type ProjectMeta struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Subtitle    string   `yaml:"subtitle,omitempty" json:"subtitle,omitempty"`
	Series      string   `yaml:"series,omitempty" json:"series,omitempty"`
	Type        string   `yaml:"type" json:"type"` // e.g. historical_fiction, dual_world_historical, xianxia_worldbuilding
	DomainAgent string   `yaml:"domain_agent,omitempty" json:"domain_agent,omitempty"`
	Language    string   `yaml:"language" json:"language"`
	Era         string   `yaml:"era,omitempty" json:"era,omitempty"`
	Status      string   `yaml:"status,omitempty" json:"status,omitempty"` // planning, in_progress, completed
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// BookScale defines volume, chapter, and scale metrics for large multi-volume works.
type BookScale struct {
	TotalVolumes      int    `yaml:"total_volumes,omitempty" json:"total_volumes,omitempty"`
	TotalChapters     int    `yaml:"total_chapters,omitempty" json:"total_chapters,omitempty"`
	ChaptersPerVolume int    `yaml:"chapters_per_volume,omitempty" json:"chapters_per_volume,omitempty"`
	VolumeStructure   string `yaml:"volume_structure,omitempty" json:"volume_structure,omitempty"` // "files" or "directories"
}

// BookStructure maps all narrative, chronological, character, geographic, and verification files.
type BookStructure struct {
	Outline            string   `yaml:"outline" json:"outline"`
	VolumesPlan        string   `yaml:"volumes_plan,omitempty" json:"volumes_plan,omitempty"`
	Volumes            string   `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Chapters           string   `yaml:"chapters,omitempty" json:"chapters,omitempty"`
	Timeline           string   `yaml:"timeline,omitempty" json:"timeline,omitempty"`
	TimelineAudit      string   `yaml:"timeline_audit,omitempty" json:"timeline_audit,omitempty"`
	ChapterLedger      string   `yaml:"chapter_ledger,omitempty" json:"chapter_ledger,omitempty"`
	NarrativeValidation string  `yaml:"narrative_validation,omitempty" json:"narrative_validation,omitempty"`
	ResearchQA         string   `yaml:"research_qa,omitempty" json:"research_qa,omitempty"`
	MasterBible        string   `yaml:"master_bible,omitempty" json:"master_bible,omitempty"`
	Rules              string   `yaml:"rules,omitempty" json:"rules,omitempty"`
	Guidelines         string   `yaml:"guidelines,omitempty" json:"guidelines,omitempty"`
	Characters         string   `yaml:"characters,omitempty" json:"characters,omitempty"`
	CharacterNetwork   string   `yaml:"character_network,omitempty" json:"character_network,omitempty"`
	CharacterLifecycle string   `yaml:"character_lifecycle,omitempty" json:"character_lifecycle,omitempty"`
	Geography          string   `yaml:"geography,omitempty" json:"geography,omitempty"`
	SocialSystems      string   `yaml:"social_systems,omitempty" json:"social_systems,omitempty"`
	MilitaryReference  string   `yaml:"military_reference,omitempty" json:"military_reference,omitempty"`
	Bibliography       string   `yaml:"bibliography,omitempty" json:"bibliography,omitempty"`
	EvidenceReview     string   `yaml:"evidence_review,omitempty" json:"evidence_review,omitempty"`
	Scripts            []string `yaml:"scripts,omitempty" json:"scripts,omitempty"`

	// Backward compatibility fields
	Research    string `yaml:"research,omitempty" json:"research,omitempty"`
	Consistency string `yaml:"consistency,omitempty" json:"consistency,omitempty"`
}

// LaboratoryConfig defines dual-world observational perspective settings (as seen in Tang and Yuan series).
type LaboratoryConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`
	Perspective     string `yaml:"perspective,omitempty" json:"perspective,omitempty"`
	Specification   string `yaml:"specification,omitempty" json:"specification,omitempty"`
	Guidelines      string `yaml:"guidelines,omitempty" json:"guidelines,omitempty"`
	VolumeMap       string `yaml:"volume_map,omitempty" json:"volume_map,omitempty"`
	Integration     string `yaml:"integration,omitempty" json:"integration,omitempty"`
	NonInterference bool   `yaml:"non_interference,omitempty" json:"non_interference,omitempty"`
}

// WorldviewConfig defines cultivation, realm, and principle systems (as seen in Xianxia series).
type WorldviewConfig struct {
	SystemType            string   `yaml:"system_type,omitempty" json:"system_type,omitempty"`
	EntryStages           []string `yaml:"entry_stages,omitempty" json:"entry_stages,omitempty"`
	FormalRealms          []string `yaml:"formal_realms,omitempty" json:"formal_realms,omitempty"`
	CultivationPrinciples []string `yaml:"cultivation_principles,omitempty" json:"cultivation_principles,omitempty"`
}

// BookWritingPolicy enforces consistency, anti-anachronism, and historical truth boundaries.
type BookWritingPolicy struct {
	HistoricalAccuracy        string   `yaml:"historical_accuracy" json:"historical_accuracy"`
	BattleTacticsFocus        bool     `yaml:"battle_tactics_focus" json:"battle_tactics_focus"`
	NarrativeDimensions       []string `yaml:"narrative_dimensions,omitempty" json:"narrative_dimensions,omitempty"`
	Classifications           []string `yaml:"classifications" json:"classifications"`
	PreserveHistoricalFigures bool     `yaml:"preserve_historical_figures" json:"preserve_historical_figures"`
	RequireConsistencyCheck   bool     `yaml:"require_consistency_check" json:"require_consistency_check"`
	AntiAnachronism           bool     `yaml:"anti_anachronism,omitempty" json:"anti_anachronism,omitempty"`
	AntiTeleportation         bool     `yaml:"anti_teleportation,omitempty" json:"anti_teleportation,omitempty"`
	InformationBoundary       bool     `yaml:"information_boundary,omitempty" json:"information_boundary,omitempty"`
}

// BookAuthority defines setting approvals, fact adjudication, and evidence mandates.
type BookAuthority struct {
	ConfirmedSettings   string   `yaml:"confirmed_settings" json:"confirmed_settings"`
	HistoricalFacts     string   `yaml:"historical_facts" json:"historical_facts"`
	UnresolvedQuestions string   `yaml:"unresolved_questions" json:"unresolved_questions"`
	Adjudications       []string `yaml:"adjudications,omitempty" json:"adjudications,omitempty"`
}

// ProjectStorage defines the physical or cloud backing store (thinking.md §五).
type ProjectStorage struct {
	Type string `yaml:"type" json:"type"`
	Root string `yaml:"root,omitempty" json:"root,omitempty"`
}

// ProjectVC defines version control parameters.
type ProjectVC struct {
	Type          string `yaml:"type" json:"type"`
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	DefaultBranch string `yaml:"default_branch,omitempty" json:"default_branch,omitempty"`
}

// ProjectGovernance sets human approval and branch write governance.
type ProjectGovernance struct {
	DirectMainWrite                     bool `yaml:"direct_main_write" json:"direct_main_write"`
	RequireDiffReview                   bool `yaml:"require_diff_review" json:"require_diff_review"`
	RequireHumanApprovalForMajorChanges bool `yaml:"require_human_approval_for_major_changes" json:"require_human_approval_for_major_changes"`
}

// ProjectContinuity manages cross-session state persistence.
type ProjectContinuity struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// ProjectMemory manages long-term project knowledge and decisions.
type ProjectMemory struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// Validate checks essential manifest invariants.
func (m *BookManifest) Validate() error {
	if m.Project.ID == "" {
		return fmt.Errorf("project id is required")
	}
	if m.Project.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if m.Structure.Outline == "" {
		return fmt.Errorf("structure.outline path is required")
	}
	return nil
}

// GetVolumeCount returns declared total volumes or 0.
func (m *BookManifest) GetVolumeCount() int {
	if m.Scale != nil && m.Scale.TotalVolumes > 0 {
		return m.Scale.TotalVolumes
	}
	return 0
}

// GetChapterCount returns declared total chapters or 0.
func (m *BookManifest) GetChapterCount() int {
	if m.Scale != nil && m.Scale.TotalChapters > 0 {
		return m.Scale.TotalChapters
	}
	return 0
}

// HasDualWorld checks if dual-world laboratory perspective is activated.
func (m *BookManifest) HasDualWorld() bool {
	return m.Laboratory != nil && m.Laboratory.Enabled
}
