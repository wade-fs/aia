package consistency

import "time"

type FindingSeverity int

const (
	SeverityInfo     FindingSeverity = 1
	SeverityLow      FindingSeverity = 3
	SeverityMedium   FindingSeverity = 5
	SeverityHigh     FindingSeverity = 8
	SeverityCritical FindingSeverity = 10
)

type FindingStatus string

const (
	StatusOpen              FindingStatus = "OPEN"
	StatusAcceptedDeviation FindingStatus = "ACCEPTED_DEVIATION" // e.g. intentional artistic license
	StatusResolved          FindingStatus = "RESOLVED"
	StatusRejected          FindingStatus = "REJECTED"
)

// Finding represents a detected inconsistency, gap, anomaly, or rule violation (Plan2.md §4).
type Finding struct {
	ID                    string          `json:"id" yaml:"id"`
	RuleID                string          `json:"rule_id" yaml:"rule_id"`
	Domain                string          `json:"domain" yaml:"domain"`
	Category              string          `json:"category" yaml:"category"` // e.g. "POST_MORTEM_APPEARANCE", "ANACHRONISM", "CONTRADICTION"
	Severity              FindingSeverity `json:"severity" yaml:"severity"`
	Confidence            float64         `json:"confidence" yaml:"confidence"`                       // 0.0 ~ 1.0
	EvidenceCompleteness  float64         `json:"evidence_completeness" yaml:"evidence_completeness"` // 0.0 ~ 1.0
	Summary               string          `json:"summary" yaml:"summary"`
	Details               string          `json:"details" yaml:"details"`
	SubjectEntityIDs      []string        `json:"subject_entity_ids" yaml:"subject_entity_ids"`
	FileRef               string          `json:"file_ref,omitempty" yaml:"file_ref,omitempty"`
	StatedYear            int             `json:"stated_year,omitempty" yaml:"stated_year,omitempty"`
	ExpectedYear          int             `json:"expected_year,omitempty" yaml:"expected_year,omitempty"`
	IsDeliberate          bool            `json:"is_deliberate" yaml:"is_deliberate"`
	RequiresHumanDecision bool            `json:"requires_human_decision" yaml:"requires_human_decision"`
	Status                FindingStatus   `json:"status" yaml:"status"`
	DetectedAt            time.Time       `json:"detected_at" yaml:"detected_at"`
}
