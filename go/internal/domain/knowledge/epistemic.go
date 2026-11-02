package knowledge

// EpistemicStatus captures the truth-functional and evidential standing of a claim or assertion (Plan2.md §3).
type EpistemicStatus string

const (
	StatusObserved   EpistemicStatus = "OBSERVED"   // Directly observed/measured
	StatusSourced    EpistemicStatus = "SOURCED"    // Supported by cited source/literature
	StatusInferred   EpistemicStatus = "INFERRED"   // Derived via deductive or inductive reasoning
	StatusHypothesis EpistemicStatus = "HYPOTHESIS" // Unverified research hypothesis
	StatusStipulated EpistemicStatus = "STIPULATED" // Defined axiomatically by creator/canon
	StatusFictional  EpistemicStatus = "FICTIONAL"  // Deliberate narrative invention
	StatusDisputed   EpistemicStatus = "DISPUTED"   // Multiple conflicting sources or arguments
	StatusRefuted    EpistemicStatus = "REFUTED"    // Falsified by evidence or logical contradiction
	StatusUnknown    EpistemicStatus = "UNKNOWN"    // Unspecified status
)

// DomainClassification provides the domain-specific epistemological categorization (Plan2.md §3).
type DomainClassification string

const (
	// Historical Domain Classifications
	ClassFactHistory        DomainClassification = "FACT_HISTORY"
	ClassPlausibleInference DomainClassification = "PLAUSIBLE_INFERENCE"
	ClassFiction            DomainClassification = "FICTION"

	// Worldbuilding & Creative Domain Classifications
	ClassWorldCanon       DomainClassification = "WORLD_CANON"
	ClassCharacterSetting DomainClassification = "CHARACTER_SETTING"
	ClassWorldHypothesis  DomainClassification = "WORLD_HYPOTHESIS"

	// Scientific Domain Classifications
	ClassExperimentalResult DomainClassification = "EXPERIMENTAL_RESULT"
	ClassTheoreticalModel   DomainClassification = "THEORETICAL_MODEL"
	ClassResearchHypothesis DomainClassification = "RESEARCH_HYPOTHESIS"
)
