package project

// ProfileType represents the domain profile governing knowledge schema and consistency rules (Plan2.md §5).
type ProfileType string

const (
	ProfileHistoricalFiction  ProfileType = "historical_fiction"
	ProfileScienceFiction     ProfileType = "science_fiction"
	ProfileScientificResearch ProfileType = "scientific_research"
	ProfileXianxia            ProfileType = "xianxia"
	ProfileGeneral            ProfileType = "general"
)

// DomainProfile specifies the expected document structure, knowledge schema, and active rules for a project.
type DomainProfile struct {
	Name              ProfileType       `json:"name" yaml:"name"`
	Version           string            `json:"version" yaml:"version"`
	DefaultRules      []string          `json:"default_rules" yaml:"default_rules"`
	RequiredDocuments []string          `json:"required_documents" yaml:"required_documents"`
	KnowledgeSchema   map[string]string `json:"knowledge_schema,omitempty" yaml:"knowledge_schema,omitempty"`
}

// GetStandardProfile returns the built-in specification for a given profile type.
func GetStandardProfile(name ProfileType) DomainProfile {
	switch name {
	case ProfileHistoricalFiction:
		return DomainProfile{
			Name:              ProfileHistoricalFiction,
			Version:           "1.0",
			DefaultRules:      []string{"historical_accuracy", "temporal_consistency", "character_continuity"},
			RequiredDocuments: []string{"outline.md", "characters.md"},
			KnowledgeSchema: map[string]string{
				"entities": "characters, locations, regimes",
				"temporal": "dynasty, year, month",
			},
		}
	case ProfileScienceFiction:
		return DomainProfile{
			Name:              ProfileScienceFiction,
			Version:           "1.0",
			DefaultRules:      []string{"world_canon_consistency", "technology_constraints", "temporal_consistency"},
			RequiredDocuments: []string{"outline.md", "worldbuilding.md", "characters.md"},
			KnowledgeSchema: map[string]string{
				"entities": "star_systems, civilizations, technologies",
				"rules":    "physical_laws, canon_axioms",
			},
		}
	case ProfileScientificResearch:
		return DomainProfile{
			Name:              ProfileScientificResearch,
			Version:           "1.0",
			DefaultRules:      []string{"evidence_traceability", "methodology_consistency", "data_conclusion_consistency"},
			RequiredDocuments: []string{"research_plan.md", "methodology.md", "results.md"},
			KnowledgeSchema: map[string]string{
				"entities": "hypotheses, experiments, datasets",
				"evidence": "measurements, citations",
			},
		}
	case ProfileXianxia:
		return DomainProfile{
			Name:              ProfileXianxia,
			Version:           "1.0",
			DefaultRules:      []string{"realm_tier_consistency", "cultivation_rules", "character_continuity"},
			RequiredDocuments: []string{"outline.md", "realms.md", "characters.md"},
			KnowledgeSchema: map[string]string{
				"entities": "cultivators, sects, realms, techniques",
				"rules":    "breakthrough_requirements",
			},
		}
	default:
		return DomainProfile{
			Name:              ProfileGeneral,
			Version:           "1.0",
			DefaultRules:      []string{"missing_reference", "temporal_consistency"},
			RequiredDocuments: []string{"README.md"},
		}
	}
}
