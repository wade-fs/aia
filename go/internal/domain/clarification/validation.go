package clarification

import (
	"fmt"
	"strings"
)

const MaxBatchQuestions = 5

// Validate is an alias for ValidateRequest.
func Validate(req *ClarificationRequest) error {
	return ValidateRequest(req)
}

// ValidateRequest checks if an individual clarification request conforms to governance rules (AIA-21 §2.3, §5.5, §6.3).
func ValidateRequest(req *ClarificationRequest) error {
	if strings.TrimSpace(req.ProjectID) == "" {
		return fmt.Errorf("project_id cannot be empty")
	}
	if strings.TrimSpace(req.FieldKey) == "" {
		return fmt.Errorf("field_key cannot be empty")
	}
	if strings.TrimSpace(req.Question) == "" {
		return fmt.Errorf("question cannot be empty")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return fmt.Errorf("question on '%s' must provide a decision-justifying reason (AIA-21 §2.3)", req.FieldKey)
	}
	if strings.TrimSpace(req.Impact) == "" {
		return fmt.Errorf("question on '%s' must state downstream architectural impact (AIA-21 §5.2)", req.FieldKey)
	}
	return nil
}

// ValidateBatch verifies that a collection of questions respects discipline limits (AIA-21 §5.4).
func ValidateBatch(requests []*ClarificationRequest) error {
	if len(requests) > MaxBatchQuestions {
		return fmt.Errorf("question discipline violation: batch contains %d questions (maximum allowed is %d, AIA-21 §5.4)", len(requests), MaxBatchQuestions)
	}

	seenKeys := make(map[string]bool)
	for _, req := range requests {
		if err := ValidateRequest(req); err != nil {
			return err
		}
		if seenKeys[req.FieldKey] {
			return fmt.Errorf("duplicate clarification question for field '%s' in same batch", req.FieldKey)
		}
		seenKeys[req.FieldKey] = true
	}
	return nil
}
