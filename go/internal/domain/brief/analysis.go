package brief

import (
	"fmt"
	"strings"

	"github.com/wade/aia/internal/domain/clarification"
)

// MissingFieldAnalysis highlights missing critical or supplementary parameters in a Brief (AIA-21 §6.2).
type MissingFieldAnalysis struct {
	Field      string `json:"field"`
	Importance string `json:"importance"` // "high", "medium", "low"
	Blocking   bool   `json:"blocking"`
	Reason     string `json:"reason"`
}

// ProposedDefault represents an agent-recommended provisional default value with explicit rationale (AIA-21 §4.4, §6.2).
type ProposedDefault struct {
	Field      string  `json:"field"`
	Value      any     `json:"value"`
	Rationale  string  `json:"rationale"`
	Confidence float64 `json:"confidence"`
}

// AnalysisReadiness captures the agent's maturity diagnostic.
type AnalysisReadiness struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// AgentBriefAnalysis encapsulates the structured analysis emitted by the agent (AIA-21 §6.2).
type AgentBriefAnalysis struct {
	Summary                string                            `json:"summary"`
	Understanding          []string                          `json:"understanding"`
	MissingFields          []MissingFieldAnalysis            `json:"missing_fields"`
	ClarificationQuestions []*clarification.ClarificationRequest `json:"clarification_questions"`
	ProposedDefaults       []ProposedDefault                 `json:"proposed_defaults"`
	ResearchCandidates     []string                          `json:"research_candidates"`
	Recommendations        []string                          `json:"recommendations"`
	Readiness              AnalysisReadiness                 `json:"readiness"`
}

// ValidateBriefAnalysis validates that the agent's analysis conforms strictly to AIA-21 §6.3 governance rules.
func ValidateBriefAnalysis(analysis *AgentBriefAnalysis, catalog *DomainCatalog) error {
	if analysis == nil {
		return fmt.Errorf("brief analysis payload cannot be nil")
	}
	if strings.TrimSpace(analysis.Summary) == "" {
		return fmt.Errorf("analysis summary cannot be empty")
	}

	if catalog == nil {
		return fmt.Errorf("domain catalog is required for validation")
	}

	// 1. Validate MissingFields against Domain Catalog
	for _, mf := range analysis.MissingFields {
		fKey := strings.TrimSpace(mf.Field)
		if fKey == "" {
			return fmt.Errorf("missing field name cannot be empty")
		}
		if _, exists := catalog.Fields[fKey]; !exists {
			return fmt.Errorf("unauthorized missing field '%s': not found in domain catalog '%s' (AIA-21 §6.3)", fKey, catalog.Domain)
		}
		if strings.TrimSpace(mf.Reason) == "" {
			return fmt.Errorf("missing field '%s' must provide a justification reason", fKey)
		}
	}

	// 2. Validate ProposedDefaults
	for _, pd := range analysis.ProposedDefaults {
		fKey := strings.TrimSpace(pd.Field)
		if fKey == "" {
			return fmt.Errorf("proposed default field cannot be empty")
		}
		if _, exists := catalog.Fields[fKey]; !exists {
			return fmt.Errorf("unauthorized proposed default field '%s': not found in domain catalog '%s' (AIA-21 §6.3)", fKey, catalog.Domain)
		}
		if pd.Value == nil {
			return fmt.Errorf("proposed default value for '%s' cannot be nil", fKey)
		}
		if strings.TrimSpace(pd.Rationale) == "" {
			return fmt.Errorf("proposed default '%s' must provide a decision justification rationale (AIA-21 §4.4)", fKey)
		}
		if pd.Confidence < 0.0 || pd.Confidence > 1.0 {
			return fmt.Errorf("proposed default confidence for '%s' must be between 0.0 and 1.0, got %f", fKey, pd.Confidence)
		}
		if pd.Confidence >= 1.0 {
			// Agent defaults are provisional hypotheses, never 100% human-confirmed
			return fmt.Errorf("proposed default confidence for '%s' cannot be 1.0 (agent guesses cannot be marked as confirmed facts, AIA-21 §6.3 rule 10)", fKey)
		}
	}

	// 3. Validate ClarificationQuestions
	if len(analysis.ClarificationQuestions) > 0 {
		// Use standard batch validation (discipline <= 5, reason/impact, no duplicates)
		if err := clarification.ValidateBatch(analysis.ClarificationQuestions); err != nil {
			return fmt.Errorf("clarification questions validation failed: %w", err)
		}

		for _, cq := range analysis.ClarificationQuestions {
			fKey := strings.TrimSpace(cq.FieldKey)
			if _, exists := catalog.Fields[fKey]; !exists {
				return fmt.Errorf("unauthorized question field '%s': not found in domain catalog '%s'", fKey, catalog.Domain)
			}
			// Enforce security rule: Agent cannot mark questions as answered or confirmed
			if cq.Status != "" && cq.Status != clarification.StatusOpen {
				return fmt.Errorf("agent generated question for '%s' cannot have non-OPEN status '%s'", fKey, cq.Status)
			}
		}
	}

	return nil
}
