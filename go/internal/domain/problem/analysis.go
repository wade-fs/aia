package problem

import (
	"errors"
	"fmt"
	"strings"

	domainClarify "github.com/wade/aia/internal/domain/clarification"
)

var (
	ErrEmptyObjective             = errors.New("problem objective cannot be empty")
	ErrInvalidFactConfidence       = errors.New("known facts must have confidence = 1.0 and a valid source")
	ErrInvalidAssumptionConfidence = errors.New("assumptions/hypotheses must have confidence < 1.0")
	ErrInvalidUnknownConfidence    = errors.New("epistemic unknowns must have confidence = 0.0")
	ErrTooManyQuestions            = errors.New("clarification batch exceeds discipline limit of 5 questions")
)

// GapCategory specifies the type of gap or conflict discovered (AIA-22 §4.5).
type GapCategory string

const (
	GapDataGap                 GapCategory = "DATA_GAP"
	GapGoalAmbiguity           GapCategory = "GOAL_AMBIGUITY"
	GapRuleConflict            GapCategory = "RULE_CONFLICT"
	GapResourceLimit           GapCategory = "RESOURCE_LIMIT"
	GapTechConstraint          GapCategory = "TECH_CONSTRAINT"
	GapUnknownDependency       GapCategory = "UNKNOWN_DEPENDENCY"
	GapPotentialRisk           GapCategory = "POTENTIAL_RISK"
	GapContradictoryAssumption GapCategory = "CONTRADICTORY_ASSUMPTION"
)

// GapHandlingStrategy defines the recommended next action for a gap (AIA-22 §4.5, §7.4).
type GapHandlingStrategy string

const (
	HandlingResearch              GapHandlingStrategy = "RESEARCH"
	HandlingClarify               GapHandlingStrategy = "CLARIFY"
	HandlingProvisionalAssumption GapHandlingStrategy = "PROVISIONAL_ASSUMPTION"
	HandlingDefer                 GapHandlingStrategy = "DEFER"
	HandlingBlock                 GapHandlingStrategy = "BLOCK"
)

// GapConflictItem represents a discovered gap, unknown, or conflict (AIA-22 §4.5).
type GapConflictItem struct {
	ID                    string              `json:"id"`
	Category              GapCategory         `json:"category"`
	Description           string              `json:"description"`
	Source                string              `json:"source"`
	Impact                string              `json:"impact"`
	Severity              string              `json:"severity"` // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	Confidence            float64             `json:"confidence"`
	SuggestedHandling     GapHandlingStrategy `json:"suggested_handling"`
	RequiresClarification bool                `json:"requires_clarification"`
}

// IntentSummary encapsulates the understood user intent (AIA-22 §4.2).
type IntentSummary struct {
	UserGoal        string   `json:"user_goal"`
	RequestedMethod string   `json:"requested_method,omitempty"`
	Constraints     []string `json:"constraints,omitempty"`
	Assumptions     []string `json:"assumptions,omitempty"`
	SuccessCriteria []string `json:"success_criteria,omitempty"`
	AnalysisOnly    bool     `json:"analysis_only"` // If true, cannot expand to execution without authorization
}

// ContextSummary captures the findings from investigating existing resources (AIA-22 §4.3).
type ContextSummary struct {
	FoundResources       []string `json:"found_resources,omitempty"`
	MissingResources     []string `json:"missing_resources,omitempty"`
	TrustedData          []string `json:"trusted_data,omitempty"`
	NeedsVerification    []string `json:"needs_verification,omitempty"`
	ConflictingData      []string `json:"conflicting_data,omitempty"`
}

// AgentProblemAnalysis represents the structured output of an agent's problem analysis (AIA-22 §4).
type AgentProblemAnalysis struct {
	Intent                 IntentSummary                    `json:"intent"`
	Context                ContextSummary                   `json:"context"`
	GapsAndConflicts       []GapConflictItem                `json:"gaps_and_conflicts,omitempty"`
	ModelItems             []ModelItem                      `json:"model_items,omitempty"`
	ClarificationQuestions []*domainClarify.ClarificationRequest `json:"clarification_questions,omitempty"`
}

// ValidateProblemAnalysis enforces epistemic separation, confidence integrity, and discipline limits (AIA-22 §4, §7).
func ValidateProblemAnalysis(a *AgentProblemAnalysis) error {
	if a == nil {
		return errors.New("agent problem analysis is nil")
	}

	// 1. Intent validation
	if strings.TrimSpace(a.Intent.UserGoal) == "" {
		return ErrEmptyObjective
	}

	// 2. Epistemic item validation (AIA-22 §7.1, §7.2)
	for _, item := range a.ModelItems {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("invalid model item %q: %w", item.ID, err)
		}

		switch item.Category {
		case CategoryKnownFact:
			// Verified facts must have 1.0 confidence and verifiable source
			if item.Confidence != 1.0 || strings.TrimSpace(item.Source) == "" {
				return fmt.Errorf("%w: item %q (confidence=%.2f, source=%q)", ErrInvalidFactConfidence, item.ID, item.Confidence, item.Source)
			}
		case CategoryAssumption:
			// Assumptions must have confidence < 1.0
			if item.Confidence >= 1.0 {
				return fmt.Errorf("%w: assumption %q cannot claim 1.0 confidence", ErrInvalidAssumptionConfidence, item.ID)
			}
		case CategoryUnknown:
			// Epistemic unknowns have 0.0 confidence
			if item.Confidence != 0.0 {
				return fmt.Errorf("%w: unknown %q must have confidence 0.0", ErrInvalidUnknownConfidence, item.ID)
			}
		}
	}

	// 3. Question discipline limit (AIA-21 §5.4, AIA-22 §4.5)
	if len(a.ClarificationQuestions) > 5 {
		return fmt.Errorf("%w: got %d questions", ErrTooManyQuestions, len(a.ClarificationQuestions))
	}

	for i, q := range a.ClarificationQuestions {
		if strings.TrimSpace(q.Question) == "" {
			return fmt.Errorf("clarification question [%d] cannot have empty question text", i)
		}
		if strings.TrimSpace(q.Reason) == "" {
			return fmt.Errorf("clarification question [%d] must state resolution reason (決策價值)", i)
		}
		if strings.TrimSpace(q.Impact) == "" {
			return fmt.Errorf("clarification question [%d] must state impact on problem", i)
		}
	}

	return nil
}
