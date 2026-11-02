package knowledge

// RelationType characterizes the bond or dependency between two entities (Plan2.md §2).
type RelationType string

const (
	RelationHierarchical RelationType = "hierarchical" // Part-of, member-of, subordinate
	RelationTemporal     RelationType = "temporal"     // Precedes, succeeds, contemporary
	RelationSpatial      RelationType = "spatial"      // Located-in, adjacent-to, orbits
	RelationCausal       RelationType = "causal"       // Causes, enables, triggers, falsifies
	RelationSocial       RelationType = "social"       // Kinship, alliance, rivalry, teacher-student
	RelationCustom       RelationType = "custom"
)

// Relation connects two knowledge entities with a defined semantic predicate (Plan2.md §2).
type Relation struct {
	ID             string       `json:"id" yaml:"id"`
	SourceEntityID string       `json:"source_entity_id" yaml:"source_entity_id"`
	TargetEntityID string       `json:"target_entity_id" yaml:"target_entity_id"`
	Type           RelationType `json:"type" yaml:"type"`
	Predicate      string       `json:"predicate" yaml:"predicate"` // e.g. "ruled_by", "teacher_of"
	Bidirectional  bool         `json:"bidirectional" yaml:"bidirectional"`
	Description    string       `json:"description,omitempty" yaml:"description,omitempty"`
}
