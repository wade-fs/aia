package goal

import "time"

type GoalLevel string

const (
	LevelStrategic   GoalLevel = "STRATEGIC"
	LevelTactical    GoalLevel = "TACTICAL"
	LevelOperational GoalLevel = "OPERATIONAL"
)

type GoalStatus string

const (
	StatusDraft      GoalStatus = "DRAFT"
	StatusActive     GoalStatus = "ACTIVE"
	StatusPaused     GoalStatus = "PAUSED"
	StatusAchieved   GoalStatus = "ACHIEVED"
	StatusSuperseded GoalStatus = "SUPERSEDED"
	StatusAbandoned  GoalStatus = "ABANDONED"
)

// Goal represents a long-term desired state or objective.
type Goal struct {
	ID                string     `json:"id"`
	ParentGoalID      *string    `json:"parent_goal_id,omitempty"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	Level             GoalLevel  `json:"level"`
	Status            GoalStatus `json:"status"`
	TargetMetricJSON  string     `json:"target_metric_json,omitempty"`
	CurrentMetricJSON string     `json:"current_metric_json,omitempty"`
	Priority          int        `json:"priority"` // 1 (lowest) ~ 10 (highest)
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// GoalRevision tracks state transitions and revision rationales.
type GoalRevision struct {
	ID                string    `json:"id"`
	GoalID            string    `json:"goal_id"`
	RevisionNumber    int       `json:"revision_number"`
	PreviousStateJSON string    `json:"previous_state_json"`
	NewStateJSON      string    `json:"new_state_json"`
	Reason            string    `json:"reason"`
	DecisionID        string    `json:"decision_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type GoalProjectLinkType string

const (
	LinkContributesTo GoalProjectLinkType = "CONTRIBUTES_TO"
	LinkValidates     GoalProjectLinkType = "VALIDATES"
	LinkDerivedFrom   GoalProjectLinkType = "DERIVED_FROM"
)

// GoalProjectLink represents the M:N relationship between Goals and Projects.
type GoalProjectLink struct {
	GoalID    string              `json:"goal_id"`
	ProjectID string              `json:"project_id"`
	LinkType  GoalProjectLinkType `json:"link_type"`
	CreatedAt time.Time           `json:"created_at"`
}
