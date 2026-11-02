package project

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

// ProjectStatus represents the lifecycle state of a project.
type ProjectStatus string

const (
	StatusActive   ProjectStatus = "ACTIVE"
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

// ProjectManifest describes the runtime and governance configuration of a project (thinking.md §五).
type ProjectManifest struct {
	Project struct {
		ID          string     `yaml:"id" json:"id"`
		Name        string     `yaml:"name" json:"name"`
		Type        DomainType `yaml:"type" json:"type"`
		DomainAgent string     `yaml:"domain_agent" json:"domain_agent"`
	} `yaml:"project" json:"project"`

	Storage struct {
		Type string `yaml:"type" json:"type"`
		Root string `yaml:"root" json:"root"`
	} `yaml:"storage" json:"storage"`

	VersionControl struct {
		Type          string `yaml:"type" json:"type"`
		Enabled       bool   `yaml:"enabled" json:"enabled"`
		DefaultBranch string `yaml:"default_branch" json:"default_branch"`
	} `yaml:"version_control" json:"version_control"`

	Continuity struct {
		Enabled bool `yaml:"enabled" json:"enabled"`
	} `yaml:"continuity" json:"continuity"`

	Memory struct {
		Enabled bool `yaml:"enabled" json:"enabled"`
	} `yaml:"memory" json:"memory"`

	Governance struct {
		DirectMainWrite                     bool `yaml:"direct_main_write" json:"direct_main_write"`
		RequireDiffReview                   bool `yaml:"require_diff_review" json:"require_diff_review"`
		RequireHumanApprovalForMajorChanges bool `yaml:"require_human_approval_for_major_changes" json:"require_human_approval_for_major_changes"`
	} `yaml:"governance" json:"governance"`
}
