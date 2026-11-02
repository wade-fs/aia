package knowledge

// Assertion represents a concrete claim or narrative statement about an Entity (Plan2.md §2).
type Assertion struct {
	ID                  string               `json:"id" yaml:"id"`
	SubjectEntityID     string               `json:"subject_entity_id" yaml:"subject_entity_id"`
	Predicate           string               `json:"predicate" yaml:"predicate"`     // e.g. "active_in", "deceased_in", "discovered"
	ObjectValue         string               `json:"object_value" yaml:"object_value"` // e.g. "1360", "General", "Nanjing"
	Epistemic           EpistemicStatus      `json:"epistemic" yaml:"epistemic"`
	DomainClass         DomainClassification `json:"domain_class" yaml:"domain_class"`
	Temporal            *TemporalReference   `json:"temporal,omitempty" yaml:"temporal,omitempty"`
	Spatial             *SpatialReference    `json:"spatial,omitempty" yaml:"spatial,omitempty"`
	EvidenceIDs         []string             `json:"evidence_ids,omitempty" yaml:"evidence_ids,omitempty"`
	IsDeliberateFiction bool                 `json:"is_deliberate_fiction,omitempty" yaml:"is_deliberate_fiction,omitempty"`
	SourceFile          string               `json:"source_file,omitempty" yaml:"source_file,omitempty"`
	SourceLine          int                  `json:"source_line,omitempty" yaml:"source_line,omitempty"`
}
