package project

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

type ProjectStatus string

const (
	StatusActive   ProjectStatus = "ACTIVE"
	StatusArchived ProjectStatus = "ARCHIVED"
)

type Project struct {
	ID                domain.ID     `json:"id"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	Workspace         string        `json:"workspace"`
	PolicyID          domain.ID     `json:"policy_id"`
	Status            ProjectStatus `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	PrimaryGoalID     *string       `json:"primary_goal_id,omitempty"`
	LatestSnapshotID  *string       `json:"latest_snapshot_id,omitempty"`
	PendingIssuesJSON string        `json:"pending_issues_json,omitempty"`
}
