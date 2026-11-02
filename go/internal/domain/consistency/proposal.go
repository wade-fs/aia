package consistency

import "time"

// ChangeProposal represents a concrete patch or remediation proposal for a Finding (Plan2.md §2).
type ChangeProposal struct {
	ID           string    `json:"id" yaml:"id"`
	FindingID    string    `json:"finding_id,omitempty" yaml:"finding_id,omitempty"`
	TargetFile   string    `json:"target_file" yaml:"target_file"`
	ProposedDiff string    `json:"proposed_diff" yaml:"proposed_diff"`
	Rationale    string    `json:"rationale" yaml:"rationale"`
	CreatedBy    string    `json:"created_by" yaml:"created_by"`
	CreatedAt    time.Time `json:"created_at" yaml:"created_at"`
}
