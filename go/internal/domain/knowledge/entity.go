package knowledge

// EntityType represents the conceptual category of a knowledge entity (Plan2.md §2).
type EntityType string

const (
	EntityCharacter    EntityType = "character"     // Historical or fictional agent
	EntityConcept      EntityType = "concept"       // Philosophical or scientific theory
	EntityObject       EntityType = "object"        // Tool, artifact, weapon, relic
	EntityLocation     EntityType = "location"      // City, mountain, province, star system
	EntityOrganization EntityType = "organization"  // Dynasty, clan, sect, university, state
	EntityEvent        EntityType = "event"         // Battle, coronation, experiment, discovery
	EntityPhenomenon   EntityType = "phenomenon"    // Natural or supernatural occurrence
	EntityCelestial    EntityType = "celestial"     // Planet, star, moon, orbit
	EntityCustom       EntityType = "custom"        // Domain-specific entity
)

// Entity represents any subject of investigation, worldbuilding, or narrative (Plan2.md §2).
type Entity struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Aliases     []string          `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Type        EntityType        `json:"type" yaml:"type"`
	Domain      string            `json:"domain" yaml:"domain"` // e.g. "writing", "historical", "scientific"
	Description string            `json:"description" yaml:"description"`
	Attributes  map[string]any    `json:"attributes,omitempty" yaml:"attributes,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

func NewEntity(id, name string, entityType EntityType, domain string) Entity {
	return Entity{
		ID:         id,
		Name:       name,
		Type:       entityType,
		Domain:     domain,
		Attributes: make(map[string]any),
		Metadata:   make(map[string]string),
	}
}
