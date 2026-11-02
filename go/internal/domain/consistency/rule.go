package consistency

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wade/aia/internal/domain/knowledge"
)

// KnowledgeSnapshot provides an in-memory view of entities, relations, and assertions for evaluation.
type KnowledgeSnapshot struct {
	Entities   map[string]knowledge.Entity
	Relations  []knowledge.Relation
	Assertions []knowledge.Assertion
	Metadata   map[string]any
}

func NewKnowledgeSnapshot() *KnowledgeSnapshot {
	return &KnowledgeSnapshot{
		Entities:   make(map[string]knowledge.Entity),
		Metadata:   make(map[string]any),
	}
}

// ConsistencyRule defines a pluggable, domain-aware consistency validator (Plan2.md §4).
type ConsistencyRule interface {
	ID() string
	Name() string
	Domain() string
	Evaluate(ctx context.Context, snapshot *KnowledgeSnapshot) ([]Finding, error)
}

// MissingReferenceRule flags references to undeclared entities.
type MissingReferenceRule struct{}

func (r *MissingReferenceRule) ID() string     { return "rule-generic-missing-ref" }
func (r *MissingReferenceRule) Name() string   { return "Missing Reference Validator" }
func (r *MissingReferenceRule) Domain() string { return "generic" }

func (r *MissingReferenceRule) Evaluate(ctx context.Context, snapshot *KnowledgeSnapshot) ([]Finding, error) {
	var findings []Finding
	now := time.Now()

	for _, rel := range snapshot.Relations {
		if _, ok := snapshot.Entities[rel.TargetEntityID]; !ok {
			findings = append(findings, Finding{
				ID:                   fmt.Sprintf("find-miss-ref-%s-%s", rel.SourceEntityID, rel.TargetEntityID),
				RuleID:               r.ID(),
				Domain:               "generic",
				Category:             "MISSING_REFERENCE",
				Severity:             SeverityMedium,
				Confidence:           1.0,
				EvidenceCompleteness: 0.8,
				Summary:              fmt.Sprintf("Entity %s references missing entity %s via relation %s", rel.SourceEntityID, rel.TargetEntityID, rel.Predicate),
				SubjectEntityIDs:     []string{rel.SourceEntityID, rel.TargetEntityID},
				Status:               StatusOpen,
				DetectedAt:           now,
			})
		}
	}

	return findings, nil
}

// TemporalConflictRule detects timeline contradictions against entity lifecycle bounds (Plan2.md §4).
type TemporalConflictRule struct{}

func (r *TemporalConflictRule) ID() string     { return "rule-generic-temporal-conflict" }
func (r *TemporalConflictRule) Name() string   { return "Temporal Conflict Validator" }
func (r *TemporalConflictRule) Domain() string { return "generic" }

func (r *TemporalConflictRule) Evaluate(ctx context.Context, snapshot *KnowledgeSnapshot) ([]Finding, error) {
	var findings []Finding
	now := time.Now()

	for _, a := range snapshot.Assertions {
		if a.Temporal == nil || a.Temporal.Year() == 0 {
			continue
		}

		entity, exists := snapshot.Entities[a.SubjectEntityID]
		if !exists {
			continue
		}

		statedYear := a.Temporal.Year()

		// Check death year in entity attributes
		var deathYear int
		if dyVal, ok := entity.Attributes["death_year"]; ok {
			switch v := dyVal.(type) {
			case int:
				deathYear = v
			case string:
				deathYear, _ = strconv.Atoi(v)
			}
		}

		var birthYear int
		if byVal, ok := entity.Attributes["birth_year"]; ok {
			switch v := byVal.(type) {
			case int:
				birthYear = v
			case string:
				birthYear, _ = strconv.Atoi(v)
			}
		}

		// 1. Post-Mortem Appearance
		if deathYear > 0 && statedYear > deathYear {
			isFiction := a.IsDeliberateFiction || a.Epistemic == knowledge.StatusFictional || entity.Attributes["is_fiction"] == true
			status := StatusOpen
			sev := SeverityHigh
			if isFiction {
				status = StatusAcceptedDeviation
				sev = SeverityLow
			}

			findings = append(findings, Finding{
				ID:                    fmt.Sprintf("find-post-mortem-%s-%d", entity.Name, statedYear),
				RuleID:                r.ID(),
				Domain:                entity.Domain,
				Category:              "POST_MORTEM_APPEARANCE",
				Severity:              sev,
				Confidence:            0.95,
				EvidenceCompleteness:  1.0,
				Summary:               fmt.Sprintf("實體 %s 於 %d 年後仍有活動描述 (主張年份: %d)", entity.Name, deathYear, statedYear),
				Details:               fmt.Sprintf("述詞: %s, 數值: %s, 來源檔案: %s", a.Predicate, a.ObjectValue, a.SourceFile),
				SubjectEntityIDs:      []string{entity.ID},
				FileRef:               a.SourceFile,
				StatedYear:            statedYear,
				ExpectedYear:          deathYear,
				IsDeliberate:          isFiction,
				RequiresHumanDecision: !isFiction,
				Status:                status,
				DetectedAt:            now,
			})
		}

		// 2. Pre-Birth Anachronism
		if birthYear > 0 && statedYear < birthYear {
			findings = append(findings, Finding{
				ID:                   fmt.Sprintf("find-anachronism-%s-%d", entity.Name, statedYear),
				RuleID:               r.ID(),
				Domain:               entity.Domain,
				Category:             "ANACHRONISM",
				Severity:             SeverityHigh,
				Confidence:           0.90,
				EvidenceCompleteness: 0.9,
				Summary:              fmt.Sprintf("實體 %s 生於 %d 年，但於 %d 年提早出現", entity.Name, birthYear, statedYear),
				SubjectEntityIDs:     []string{entity.ID},
				FileRef:              a.SourceFile,
				StatedYear:           statedYear,
				ExpectedYear:         birthYear,
				IsDeliberate:         false,
				Status:               StatusOpen,
				DetectedAt:           now,
			})
		}
	}

	return findings, nil
}
