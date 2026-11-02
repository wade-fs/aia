package brief

import (
	"fmt"
	"strings"
)

// AssessmentResult contains the computed readiness status, defect explanations, and assumptions.
type AssessmentResult struct {
	Status           ReadinessStatus `json:"status"`
	Rationale        string          `json:"rationale"`
	MissingP0Fields  []string        `json:"missing_p0_fields"`
	MissingP1Fields  []string        `json:"missing_p1_fields"`
	PendingAssumes   []string        `json:"pending_assumptions"`
	ConflictedFields []string        `json:"conflicted_fields"`
}

// AssessReadiness evaluates the project brief against its domain catalog (AIA-21 §8.1, §8.2).
// It distinguishes hard blocking unknowns (P0) from soft defaults (READY_WITH_ASSUMPTIONS).
func AssessReadiness(brief *ProjectBrief, catalog *DomainCatalog) AssessmentResult {
	res := AssessmentResult{
		Status:           ReadinessNotReady,
		MissingP0Fields:  make([]string, 0),
		MissingP1Fields:  make([]string, 0),
		PendingAssumes:   make([]string, 0),
		ConflictedFields: make([]string, 0),
	}

	if catalog == nil {
		res.Rationale = "No domain catalog available for validation"
		return res
	}

	for key, def := range catalog.Fields {
		f, exists := brief.Fields[key]

		// Check conflicted state
		if exists && f.Status == FieldStatusConflicted {
			res.ConflictedFields = append(res.ConflictedFields, key)
			continue
		}

		// Check if missing or unknown
		if !exists || f == nil || f.Status == FieldStatusUnknown || f.Status == FieldStatusAsked {
			if def.Importance == ImportanceP0 && def.RequiredForPlanning {
				res.MissingP0Fields = append(res.MissingP0Fields, fmt.Sprintf("%s (%s)", key, def.Description))
			} else if def.Importance == ImportanceP1 && def.RequiredForPlanning {
				res.MissingP1Fields = append(res.MissingP1Fields, fmt.Sprintf("%s (%s)", key, def.Description))
			}
			continue
		}

		// Check planning-critical assumptions (DEFAULTED or explicit confirmation required)
		if def.RequiredForPlanning && (f.Status == FieldStatusDefaulted || f.ConfirmationRequired) {
			res.PendingAssumes = append(res.PendingAssumes, fmt.Sprintf("%s: %v (來源: %s, 理由: %s)", key, f.Value, f.Source, f.Rationale))
		}
	}

	// 1. Conflict takes highest priority -> NEEDS_REASSESSMENT / NOT_READY
	if len(res.ConflictedFields) > 0 {
		res.Status = ReadinessNeedsReassessment
		res.Rationale = fmt.Sprintf("存在相互矛盾的需求設定 (%s)，需優先釐清衝突", strings.Join(res.ConflictedFields, ", "))
		return res
	}

	// 2. Missing critical P0 fields -> NOT_READY
	if len(res.MissingP0Fields) > 0 {
		res.Status = ReadinessNotReady
		res.Rationale = fmt.Sprintf("尚缺少重大必要規格 (P0: %s)，無法安全進入規劃階段", strings.Join(res.MissingP0Fields, "、"))
		return res
	}

	// 3. Missing P1 fields without even an assumption -> NOT_READY
	if len(res.MissingP1Fields) > 0 {
		res.Status = ReadinessNotReady
		res.Rationale = fmt.Sprintf("缺少主要規劃參數 (P1: %s)", strings.Join(res.MissingP1Fields, "、"))
		return res
	}

	// 4. All planning-required fields present, but some are DEFAULTED assumptions -> READY_WITH_ASSUMPTIONS
	if len(res.PendingAssumes) > 0 {
		res.Status = ReadinessReadyWithAssumptions
		res.Rationale = fmt.Sprintf("主要規格已就緒，但包含 %d 項暫定假設待確認: %s", len(res.PendingAssumes), strings.Join(res.PendingAssumes, "; "))
		return res
	}

	// 5. Check if ready for actual execution
	allExecReady := true
	for key, def := range catalog.Fields {
		if def.RequiredForExecution {
			f, exists := brief.Fields[key]
			if !exists || (f.Status != FieldStatusConfirmed && f.Status != FieldStatusAnswered) {
				allExecReady = false
				break
			}
		}
	}

	if allExecReady {
		res.Status = ReadinessReadyForExecution
		res.Rationale = "所有必要參數與執行邊界均已驗證，具備實際執行條件"
	} else {
		res.Status = ReadinessReadyForPlanning
		res.Rationale = "所有規劃需求均已確認，可正式生成全書大綱或架構工單"
	}

	return res
}
