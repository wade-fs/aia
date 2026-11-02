package project

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

// ProjectStatus represents the lifecycle state of a project.
type ProjectStatus string

const (
	StatusActive   ProjectStatus = "ACTIVE"
	StatusPaused   ProjectStatus = "PAUSED"
	StatusArchived ProjectStatus = "ARCHIVED"
)

// DomainType classifies the project by its managing domain agent.
// This determines which Observer seeds workspace paths and which Pilot
// handles execution proposals.
type DomainType string

const (
	DomainWriting    DomainType = "writing"
	DomainCoding     DomainType = "coding"
	DomainInvestment DomainType = "investment"
	DomainGeneral    DomainType = "general"
)

type Project struct {
	ID                domain.ID     `json:"id"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	Workspace         string        `json:"workspace"`
	// DomainType identifies which domain agent manages this project.
	// Set via `aia project register --type <writing|coding|investment>`.
	DomainType        DomainType    `json:"domain_type,omitempty"`
	PolicyID          domain.ID     `json:"policy_id"`
	Status            ProjectStatus `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	PrimaryGoalID     *string       `json:"primary_goal_id,omitempty"`
	LatestSnapshotID  *string       `json:"latest_snapshot_id,omitempty"`
	PendingIssuesJSON string        `json:"pending_issues_json,omitempty"`
}

// ManifestProfile designates the domain profile governing knowledge schema and rules (Plan2.md §5).
type ManifestProfile struct {
	Name    ProfileType `yaml:"name" json:"name"`
	Version string      `yaml:"version" json:"version"`
}

// ManifestKnowledge declares paths to knowledge and assertion directories (Plan2.md §5).
type ManifestKnowledge struct {
	Entities   string `yaml:"entities,omitempty" json:"entities,omitempty"`
	Evidence   string `yaml:"evidence,omitempty" json:"evidence,omitempty"`
	Timeline   string `yaml:"timeline,omitempty" json:"timeline,omitempty"`
	Unresolved string `yaml:"unresolved,omitempty" json:"unresolved,omitempty"`
}

// ManifestDocuments specifies canonical human-readable files across creative or scientific research.
type ManifestDocuments struct {
	Outline       string `yaml:"outline,omitempty" json:"outline,omitempty"`
	Characters    string `yaml:"characters,omitempty" json:"characters,omitempty"`
	Worldbuilding string `yaml:"worldbuilding,omitempty" json:"worldbuilding,omitempty"`
	Volumes       string `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Chapters      string `yaml:"chapters,omitempty" json:"chapters,omitempty"`
	Methodology   string `yaml:"methodology,omitempty" json:"methodology,omitempty"`
	Results       string `yaml:"results,omitempty" json:"results,omitempty"`
	Discussion    string `yaml:"discussion,omitempty" json:"discussion,omitempty"`
}

// ProjectManifest describes the runtime, knowledge, and governance configuration of a project (Plan2.md §5).
type ProjectManifest struct {
	Project struct {
		ID          string     `yaml:"id" json:"id"`
		Name        string     `yaml:"name" json:"name"`
		Type        string     `yaml:"type" json:"type"`
		DomainAgent string     `yaml:"domain_agent,omitempty" json:"domain_agent,omitempty"`
		Language    string     `yaml:"language,omitempty" json:"language,omitempty"`
	} `yaml:"project" json:"project"`

	Profile ManifestProfile `yaml:"profile,omitempty" json:"profile,omitempty"`

	Storage struct {
		Type string `yaml:"type" json:"type"`
		Root string `yaml:"root" json:"root"`
	} `yaml:"storage,omitempty" json:"storage,omitempty"`

	VersionControl struct {
		Type          string `yaml:"type" json:"type"`
		Enabled       bool   `yaml:"enabled" json:"enabled"`
		DefaultBranch string `yaml:"default_branch" json:"default_branch"`
	} `yaml:"version_control,omitempty" json:"version_control,omitempty"`

	Knowledge ManifestKnowledge `yaml:"knowledge,omitempty" json:"knowledge,omitempty"`
	Documents ManifestDocuments `yaml:"documents,omitempty" json:"documents,omitempty"`
	Rules     []string          `yaml:"rules,omitempty" json:"rules,omitempty"`

	Continuity struct {
		Enabled bool `yaml:"enabled" json:"enabled"`
	} `yaml:"continuity,omitempty" json:"continuity,omitempty"`

	Memory struct {
		Enabled bool `yaml:"enabled" json:"enabled"`
	} `yaml:"memory,omitempty" json:"memory,omitempty"`

	Governance struct {
		DirectMainWrite                     bool `yaml:"direct_main_write" json:"direct_main_write"`
		RequireDiffReview                   bool `yaml:"require_diff_review" json:"require_diff_review"`
		RequireHumanApprovalForMajorChanges bool `yaml:"require_human_approval_for_major_changes" json:"require_human_approval_for_major_changes"`
	} `yaml:"governance,omitempty" json:"governance,omitempty"`
}
