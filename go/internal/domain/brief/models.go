package brief

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// FieldStatus represents the provenance and decision lifecycle state of an individual Brief field (AIA-21 §4.4).
type FieldStatus string

const (
	FieldStatusUnknown    FieldStatus = "UNKNOWN"    // Information not yet acquired
	FieldStatusAsked      FieldStatus = "ASKED"      // Clarification question asked to user
	FieldStatusAnswered   FieldStatus = "ANSWERED"   // User provided an explicit answer
	FieldStatusDefaulted  FieldStatus = "DEFAULTED"  // Agent proposed a provisional default with rationale
	FieldStatusResearched FieldStatus = "RESEARCHED" // Information acquired through evidence-based research
	FieldStatusConfirmed  FieldStatus = "CONFIRMED"  // Adopted as confirmed formal specification
	FieldStatusConflicted FieldStatus = "CONFLICTED" // Contradictory information detected
	FieldStatusSuperseded FieldStatus = "SUPERSEDED" // Replaced by a newer version/decision
)

// ReadinessStatus indicates the actionable maturity stage of the Project Brief (AIA-21 §8.2).
type ReadinessStatus string

const (
	ReadinessNotReady             ReadinessStatus = "NOT_READY"              // Major critical unknowns remain
	ReadinessReadyWithAssumptions ReadinessStatus = "READY_WITH_ASSUMPTIONS" // Proceedable, but explicit assumptions pending confirmation
	ReadinessReadyForResearch     ReadinessStatus = "READY_FOR_RESEARCH"     // Prerequisites ready for evidence/data gathering
	ReadinessReadyForPlanning     ReadinessStatus = "READY_FOR_PLANNING"     // Sufficient specifications for formal planning/outline
	ReadinessReadyForExecution    ReadinessStatus = "READY_FOR_EXECUTION"    // All constraints verified for chapter/code execution
	ReadinessNeedsReassessment    ReadinessStatus = "NEEDS_REASSESSMENT"     // New external findings invalidate previous assumptions
)

// FieldState encapsulates the value, status, origin, and justification of a single Brief field (AIA-21 §4.4).
type FieldState struct {
	Key                    string      `yaml:"key" json:"key"`
	Value                  any         `yaml:"value" json:"value"`
	Unit                   string      `yaml:"unit,omitempty" json:"unit,omitempty"`
	Status                 FieldStatus `yaml:"status" json:"status"`
	Source                 string      `yaml:"source" json:"source"`                                 // e.g. "user", "agent_recommendation", "research"
	Confidence             float64     `yaml:"confidence" json:"confidence"`                         // 0.0 to 1.0
	Rationale              string      `yaml:"rationale" json:"rationale"`                           // Explanation of value and decision
	ConfirmationRequired   bool        `yaml:"confirmation_required" json:"confirmation_required"` // True if user confirmation is mandatory
	RelatedClarificationID string      `yaml:"related_clarification_id,omitempty" json:"related_clarification_id,omitempty"`
	EvidenceRefs           []string    `yaml:"evidence_refs,omitempty" json:"evidence_refs,omitempty"`
	UpdatedAt              time.Time   `yaml:"updated_at" json:"updated_at"`
	UpdatedBy              string      `yaml:"updated_by,omitempty" json:"updated_by,omitempty"`
}

// FieldHistoryEntry records an immutable change to a Brief field over time (AIA-21 §4.5).
type FieldHistoryEntry struct {
	Version                int         `yaml:"version" json:"version"`
	OldValue               any         `yaml:"old_value" json:"old_value"`
	NewValue               any         `yaml:"new_value" json:"new_value"`
	OldStatus              FieldStatus `yaml:"old_status" json:"old_status"`
	NewStatus              FieldStatus `yaml:"new_status" json:"new_status"`
	Reason                 string      `yaml:"reason" json:"reason"`
	Source                 string      `yaml:"source" json:"source"`
	RelatedTaskID          string      `yaml:"related_task_id,omitempty" json:"related_task_id,omitempty"`
	RelatedClarificationID string      `yaml:"related_clarification_id,omitempty" json:"related_clarification_id,omitempty"`
	Timestamp              time.Time   `yaml:"timestamp" json:"timestamp"`
}

// GenericBrief contains common project definition dimensions across all domains (AIA-21 §4.2).
type GenericBrief struct {
	Purpose         string   `yaml:"purpose" json:"purpose"`                   // 專案目的
	ProjectType     string   `yaml:"project_type" json:"project_type"`         // 專案類型
	Audience        string   `yaml:"audience" json:"audience"`                 // 目標對象
	Scope           string   `yaml:"scope" json:"scope"`                       // 工作範圍
	Depth           string   `yaml:"depth" json:"depth"`                       // 深度與詳細程度
	Style           string   `yaml:"style" json:"style"`                       // 風格或表達方式
	Deliverables    []string `yaml:"deliverables" json:"deliverables"`         // 預期交付成果
	Constraints     []string `yaml:"constraints" json:"constraints"`           // 限制條件
	Schedule        string   `yaml:"schedule" json:"schedule"`                 // 時間規劃
	QualityCriteria []string `yaml:"quality_criteria" json:"quality_criteria"` // 品質與驗收標準
	SuccessCriteria []string `yaml:"success_criteria" json:"success_criteria"` // 成功條件
}

// ProjectBrief is the living, evolving requirements model for a project (AIA-21 §4.1).
type ProjectBrief struct {
	ID                 string                         `yaml:"id" json:"id"`
	ProjectID          string                         `yaml:"project_id" json:"project_id"`
	Domain             string                         `yaml:"domain" json:"domain"` // e.g. nonfiction_plan, historical_fiction, coding, investment
	BriefVersion       int                            `yaml:"brief_version" json:"brief_version"`
	Generic            GenericBrief                   `yaml:"generic" json:"generic"`
	Fields             map[string]*FieldState         `yaml:"fields" json:"fields"`
	History            map[string][]FieldHistoryEntry `yaml:"history,omitempty" json:"history,omitempty"`
	Readiness          ReadinessStatus                `yaml:"readiness" json:"readiness"`
	ReadinessRationale string                         `yaml:"readiness_rationale" json:"readiness_rationale"`
	RawIntent          string                         `yaml:"raw_intent" json:"raw_intent"`
	CreatedAt          time.Time                      `yaml:"created_at" json:"created_at"`
	UpdatedAt          time.Time                      `yaml:"updated_at" json:"updated_at"`
}

// NewProjectBrief initializes an empty Project Brief for a project and domain.
func NewProjectBrief(id, projectID, domain string, rawIntent string) *ProjectBrief {
	now := time.Now().UTC()
	return &ProjectBrief{
		ID:                 id,
		ProjectID:          projectID,
		Domain:             domain,
		BriefVersion:       1,
		Fields:             make(map[string]*FieldState),
		History:            make(map[string][]FieldHistoryEntry),
		Readiness:          ReadinessNotReady,
		ReadinessRationale: "Initial creation; requirements analysis pending",
		RawIntent:          rawIntent,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// SetField updates a field and appends an immutable entry to its change history (AIA-21 §4.5).
func (b *ProjectBrief) SetField(
	key string,
	val any,
	status FieldStatus,
	source string,
	confidence float64,
	rationale string,
	confirmationRequired bool,
	updatedBy string,
	relatedTaskID string,
	relatedClarificationID string,
) {
	if b.Fields == nil {
		b.Fields = make(map[string]*FieldState)
	}
	if b.History == nil {
		b.History = make(map[string][]FieldHistoryEntry)
	}

	now := time.Now().UTC()
	oldField, exists := b.Fields[key]

	var oldVal any
	var oldStatus FieldStatus = FieldStatusUnknown
	if exists {
		oldVal = oldField.Value
		oldStatus = oldField.Status
	}

	// Record historical change
	historyEntry := FieldHistoryEntry{
		Version:                len(b.History[key]) + 1,
		OldValue:               oldVal,
		NewValue:               val,
		OldStatus:              oldStatus,
		NewStatus:              status,
		Reason:                 rationale,
		Source:                 source,
		RelatedTaskID:          relatedTaskID,
		RelatedClarificationID: relatedClarificationID,
		Timestamp:              now,
	}
	b.History[key] = append(b.History[key], historyEntry)

	// Update current field state
	b.Fields[key] = &FieldState{
		Key:                    key,
		Value:                  val,
		Status:                 status,
		Source:                 source,
		Confidence:             confidence,
		Rationale:              rationale,
		ConfirmationRequired:   confirmationRequired,
		RelatedClarificationID: relatedClarificationID,
		UpdatedAt:              now,
		UpdatedBy:              updatedBy,
	}

	b.BriefVersion++
	b.UpdatedAt = now
}

// GetField retrieves a field state safely.
func (b *ProjectBrief) GetField(key string) (*FieldState, bool) {
	if b.Fields == nil {
		return nil, false
	}
	fs, ok := b.Fields[key]
	return fs, ok
}

// ToYAML serializes the Project Brief to YAML for workspace persistence (brief.yaml).
func (b *ProjectBrief) ToYAML() (string, error) {
	data, err := yaml.Marshal(b)
	if err != nil {
		return "", fmt.Errorf("failed to marshal project brief to yaml: %w", err)
	}
	return string(data), nil
}

// FromYAML parses a Project Brief from YAML.
func FromYAML(data []byte) (*ProjectBrief, error) {
	var b ProjectBrief
	if err := yaml.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project brief from yaml: %w", err)
	}
	if b.Fields == nil {
		b.Fields = make(map[string]*FieldState)
	}
	if b.History == nil {
		b.History = make(map[string][]FieldHistoryEntry)
	}
	return &b, nil
}
