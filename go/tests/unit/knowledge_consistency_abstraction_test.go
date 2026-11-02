package unit

import (
	"context"
	"os"
	"testing"

	"github.com/wade/aia/internal/domain/consistency"
	"github.com/wade/aia/internal/domain/knowledge"
	domainProject "github.com/wade/aia/internal/domain/project"
	"github.com/wade/aia/internal/domain/writing"
)

func TestKnowledgeCore_EntityAndEpistemicModels(t *testing.T) {
	// 1. Entity creation across diverse domains (Plan2.md §2)
	// Character entity (historical)
	charEnt := knowledge.NewEntity("ent-tuotuo", "脫脫", knowledge.EntityCharacter, "historical")
	charEnt.Attributes["birth_year"] = 1314
	charEnt.Attributes["death_year"] = 1355
	if charEnt.Name != "脫脫" || charEnt.Type != knowledge.EntityCharacter {
		t.Fatalf("unexpected character entity: %+v", charEnt)
	}

	// Concept entity (scientific)
	conceptEnt := knowledge.NewEntity("ent-entropy", "Entropy", knowledge.EntityConcept, "scientific")
	if conceptEnt.Type != knowledge.EntityConcept {
		t.Fatalf("unexpected concept entity: %+v", conceptEnt)
	}

	// Celestial entity (space research)
	planetEnt := knowledge.NewEntity("ent-mars", "Mars", knowledge.EntityCelestial, "space")
	if planetEnt.Type != knowledge.EntityCelestial {
		t.Fatalf("unexpected celestial entity: %+v", planetEnt)
	}

	// 2. Temporal Reference
	anchorYear := 1368
	tempPt := knowledge.TemporalReference{AnchorYear: &anchorYear}
	if tempPt.Year() != 1368 || tempPt.IsRange() {
		t.Errorf("expected point year 1368, got %d", tempPt.Year())
	}

	start := 1350
	end := 1368
	tempRange := knowledge.TemporalReference{StartYear: &start, EndYear: &end}
	if !tempRange.IsRange() || !tempRange.ContainsYear(1360) || tempRange.ContainsYear(1370) {
		t.Errorf("temporal range checks failed for 1350-1368")
	}

	// 3. Epistemic Status & Domain Classification separation (Plan2.md §3)
	histAsrt := knowledge.Assertion{
		ID:              "asrt-tuotuo-dismissal",
		SubjectEntityID: "ent-tuotuo",
		Predicate:       "dismissed_from_chancellorship",
		ObjectValue:     "Gaoyou battle failure",
		Epistemic:       knowledge.StatusSourced,
		DomainClass:     knowledge.ClassFactHistory,
	}
	if histAsrt.Epistemic != knowledge.StatusSourced || histAsrt.DomainClass != knowledge.ClassFactHistory {
		t.Errorf("unexpected epistemic or domain class: %+v", histAsrt)
	}

	sciHypo := knowledge.Assertion{
		ID:              "asrt-quantum-gravity",
		SubjectEntityID: "ent-entropy",
		Predicate:       "microscopic_degrees_of_freedom",
		Epistemic:       knowledge.StatusHypothesis,
		DomainClass:     knowledge.ClassResearchHypothesis,
	}
	if sciHypo.Epistemic != knowledge.StatusHypothesis || sciHypo.DomainClass != knowledge.ClassResearchHypothesis {
		t.Errorf("unexpected scientific hypothesis assertion: %+v", sciHypo)
	}
}

func TestConsistencyEngine_UniversalRuleEvaluation(t *testing.T) {
	ctx := context.Background()
	engine := consistency.NewEngine()
	snapshot := consistency.NewKnowledgeSnapshot()

	// 1. Setup entities
	eTuotuo := knowledge.NewEntity("ent-tuotuo", "脫脫", knowledge.EntityCharacter, "historical")
	eTuotuo.Attributes["birth_year"] = 1314
	eTuotuo.Attributes["death_year"] = 1355
	snapshot.Entities["ent-tuotuo"] = eTuotuo

	eQingnang := knowledge.NewEntity("ent-qingnang", "青囊子", knowledge.EntityCharacter, "writing")
	eQingnang.Attributes["death_year"] = 1340
	eQingnang.Attributes["is_fiction"] = true
	snapshot.Entities["ent-qingnang"] = eQingnang

	// 2. Add an assertion that contradicts historical fact (Tuotuo active in 1360)
	y1360 := 1360
	snapshot.Assertions = append(snapshot.Assertions, knowledge.Assertion{
		ID:              "asrt-tuotuo-1360",
		SubjectEntityID: "ent-tuotuo",
		Predicate:       "commands_imperial_army",
		ObjectValue:     "General commander",
		Temporal:        &knowledge.TemporalReference{AnchorYear: &y1360},
		SourceFile:      "volume3/ch12.md",
		DomainClass:     knowledge.ClassFactHistory,
	})

	// 3. Add an assertion that is deliberate fiction (Qingnang active in 1355)
	y1355 := 1355
	snapshot.Assertions = append(snapshot.Assertions, knowledge.Assertion{
		ID:                  "asrt-qingnang-1355",
		SubjectEntityID:     "ent-qingnang",
		Predicate:           "heals_rebel_soldier",
		Temporal:            &knowledge.TemporalReference{AnchorYear: &y1355},
		SourceFile:          "volume1/ch05.md",
		IsDeliberateFiction: true,
		Epistemic:           knowledge.StatusFictional,
	})

	// 4. Add a relation referencing a nonexistent entity
	snapshot.Relations = append(snapshot.Relations, knowledge.Relation{
		ID:             "rel-ghost-ally",
		SourceEntityID: "ent-tuotuo",
		TargetEntityID: "ent-nonexistent-general",
		Predicate:      "allied_with",
	})

	// Evaluate
	findings, err := engine.Evaluate(ctx, snapshot)
	if err != nil {
		t.Fatalf("engine evaluation failed: %v", err)
	}

	var foundMissingRef, foundHistPostMortem, foundFictionDeviation bool
	for _, f := range findings {
		if f.Category == "MISSING_REFERENCE" {
			foundMissingRef = true
		}
		if f.Category == "POST_MORTEM_APPEARANCE" {
			if !f.IsDeliberate && f.Severity == consistency.SeverityHigh && f.Status == consistency.StatusOpen {
				foundHistPostMortem = true
			}
			if f.IsDeliberate && f.Severity == consistency.SeverityLow && f.Status == consistency.StatusAcceptedDeviation {
				foundFictionDeviation = true
			}
		}
	}

	if !foundMissingRef {
		t.Errorf("expected MISSING_REFERENCE finding")
	}
	if !foundHistPostMortem {
		t.Errorf("expected actionable POST_MORTEM_APPEARANCE finding for Tuotuo")
	}
	if !foundFictionDeviation {
		t.Errorf("expected accepted deviation for deliberate fiction Qingnangzi")
	}
}

func TestProjectManifest_PolymorphicProfilesAndLegacyCompatibility(t *testing.T) {
	// 1. Legacy BookManifest YAML payload
	legacyBookYAML := `
project:
  id: yuan-novel
  name: 元末歷史演義
  type: historical_fiction
  domain_agent: writing
structure:
  outline: outline.md
  characters: characters.md
  chapters: chapters/
scale:
  total_volumes: 6
  total_chapters: 540
`
	pmLegacy, err := domainProject.ParseProjectManifest([]byte(legacyBookYAML))
	if err != nil {
		t.Fatalf("failed to parse legacy book.yaml: %v", err)
	}
	if pmLegacy.Profile.Name != domainProject.ProfileHistoricalFiction {
		t.Errorf("expected ProfileHistoricalFiction, got %s", pmLegacy.Profile.Name)
	}
	if pmLegacy.Documents.Outline != "outline.md" || pmLegacy.Documents.Characters != "characters.md" {
		t.Errorf("expected mapped documents outline and characters, got %+v", pmLegacy.Documents)
	}

	// 2. Scientific Research ProjectManifest YAML
	sciResearchYAML := `
project:
  id: gravitational-waves-01
  name: 重力波訊號分析研究
  type: scientific_research
  domain_agent: research
profile:
  name: scientific_research
  version: "1.0"
knowledge:
  entities: entities/
  evidence: evidence/
  hypotheses: hypotheses/
documents:
  methodology: methodology.md
  results: results.md
rules:
  - evidence_traceability
  - methodology_consistency
`
	pmSci, err := domainProject.ParseProjectManifest([]byte(sciResearchYAML))
	if err != nil {
		t.Fatalf("failed to parse scientific research manifest: %v", err)
	}
	if pmSci.Profile.Name != domainProject.ProfileScientificResearch {
		t.Errorf("expected ProfileScientificResearch, got %s", pmSci.Profile.Name)
	}
	if pmSci.Documents.Methodology != "methodology.md" || pmSci.Documents.Results != "results.md" {
		t.Errorf("expected methodology and results documents, got %+v", pmSci.Documents)
	}
	if len(pmSci.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(pmSci.Rules))
	}
}

func TestWritingConsistencyEngine_UniversalSnapshotBridge(t *testing.T) {
	realYuanPath := "/home/wade/src/github/aia/novel/yuan"
	if _, err := os.Stat(realYuanPath); err != nil {
		t.Skipf("real novel/yuan not accessible at %s", realYuanPath)
	}

	ctx := context.Background()
	engine := writing.NewConsistencyEngine()

	snapshot, findings, err := engine.ScanProjectSnapshot(ctx, realYuanPath, nil)
	if err != nil {
		t.Fatalf("scan project snapshot failed: %v", err)
	}
	if len(snapshot.Entities) < 20 {
		t.Errorf("expected >20 entities in snapshot, got %d", len(snapshot.Entities))
	}

	// Verify real novel has 0 unaccepted actionable findings
	for _, f := range findings {
		if !f.IsDeliberate && f.Status == consistency.StatusOpen {
			t.Errorf("unexpected actionable inconsistency in novel/yuan: %+v", f)
		}
	}
}
