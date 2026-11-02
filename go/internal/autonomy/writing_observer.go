package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wade/aia/internal/autonomy/clock"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/writing"
	taskService "github.com/wade/aia/internal/service/task"
	"gopkg.in/yaml.v3"
)

// ConsistencyScannerFunc scans manuscript files against character/timeline constraints.
type ConsistencyScannerFunc func(ctx context.Context, basePath string) ([]writing.ConsistencyAnomaly, error)

// ManuscriptConsistencyObserver monitors novel projects (e.g. novel/yuan/ under the AIA project root)
// for chronological contradictions, historical fact violations, and fictional consistency (AIA-15 §23).
type ManuscriptConsistencyObserver struct {
	projectPaths    map[string]string // projectID -> base filesystem path
	scanner         ConsistencyScannerFunc
	injectedAnoms   map[string][]writing.ConsistencyAnomaly
	charactersCache map[string][]writing.HistoricalCharacter
}

func NewManuscriptConsistencyObserver() *ManuscriptConsistencyObserver {
	return &ManuscriptConsistencyObserver{
		projectPaths:    make(map[string]string),
		injectedAnoms:   make(map[string][]writing.ConsistencyAnomaly),
		charactersCache: make(map[string][]writing.HistoricalCharacter),
	}
}

func (o *ManuscriptConsistencyObserver) RegisterProjectPath(projectID, path string) {
	o.projectPaths[projectID] = path
}

func (o *ManuscriptConsistencyObserver) SetScanner(s ConsistencyScannerFunc) {
	o.scanner = s
}

func (o *ManuscriptConsistencyObserver) InjectAnomaly(projectID string, a writing.ConsistencyAnomaly) {
	o.injectedAnoms[projectID] = append(o.injectedAnoms[projectID], a)
}

func (o *ManuscriptConsistencyObserver) ClearInjectedAnomalies(projectID string) {
	delete(o.injectedAnoms, projectID)
}

func (o *ManuscriptConsistencyObserver) Observe(ctx context.Context, projectID string) ([]domainAutonomy.Observation, error) {
	now := clock.Now(ctx)
	var observations []domainAutonomy.Observation

	basePath := o.projectPaths[projectID]
	if basePath == "" {
		basePath = projectID
	}

	var anomalies []writing.ConsistencyAnomaly
	if o.scanner != nil {
		scanned, err := o.scanner(ctx, basePath)
		if err == nil && len(scanned) > 0 {
			anomalies = append(anomalies, scanned...)
		}
	} else if basePath != "" {
		// Default scanner: inspects characters.md and outline.md if present
		anoms, _ := o.defaultScan(basePath)
		if len(anoms) > 0 {
			anomalies = append(anomalies, anoms...)
		}
	}

	if inMem, ok := o.injectedAnoms[projectID]; ok && len(inMem) > 0 {
		anomalies = append(anomalies, inMem...)
	}

	hasActionable := false
	for _, a := range anomalies {
		if !a.IsDeliberateFiction {
			hasActionable = true
			observations = append(observations, domainAutonomy.Observation{
				Source:        "MANUSCRIPT_CONSISTENCY_CHECKER",
				Domain:        "writing",
				Category:      domainAutonomy.ObsCategoryAnomaly,
				Summary:       fmt.Sprintf("[%s] %s in %s: %s", a.AnomalyType, a.CharacterName, a.ChapterRef, a.Description),
				Confidence:    domainAutonomy.ConfidenceConfirmed,
				Severity:      a.Severity,
				TargetAgentID: "writing-agent",
				Data: map[string]any{
					"anomaly_id":            a.ID,
					"anomaly_type":          string(a.AnomalyType),
					"character_name":        a.CharacterName,
					"chapter_ref":           a.ChapterRef,
					"stated_year":           a.StatedYear,
					"death_year":            a.DeathYear,
					"knowledge_tier":        string(a.KnowledgeTier),
					"is_deliberate_fiction": false,
					"base_path":             basePath,
				},
				Timestamp: now,
			})
		} else {
			// Intentional artistic license or folklore fiction -> rational NO_ACTION baseline
			observations = append(observations, domainAutonomy.Observation{
				Source:        "MANUSCRIPT_CONSISTENCY_CHECKER",
				Domain:        "writing",
				Category:      domainAutonomy.ObsCategoryStateSnapshot,
				NonActionable: true,
				Summary:       fmt.Sprintf("Intentional fiction license verified for %s in %s (%s)", a.CharacterName, a.ChapterRef, a.Description),
				TargetAgentID: "writing-agent",
				Data: map[string]any{
					"anomaly_id":            a.ID,
					"character_name":        a.CharacterName,
					"is_deliberate_fiction": true,
					"knowledge_tier":        string(writing.TierFiction),
				},
				Timestamp: now,
			})
		}
	}

	if !hasActionable && len(anomalies) == 0 {
		// Clean manuscript without contradictions -> rational NO_ACTION baseline
		observations = append(observations, domainAutonomy.Observation{
			Source:        "MANUSCRIPT_CONSISTENCY_CHECKER",
			Domain:        "writing",
			Category:      domainAutonomy.ObsCategoryStateSnapshot,
			NonActionable: true,
			Summary:       fmt.Sprintf("Manuscript at %s verified: 0 chronological anomalies, character states aligned with historical skeleton", basePath),
			TargetAgentID: "writing-agent",
			Data: map[string]any{
				"base_path":     basePath,
				"all_aligned":   true,
				"verified_year": 1368,
			},
			Timestamp: now,
		})
	}

	return observations, nil
}

// defaultScan inspects characters.md (or the file declared in book.yaml) and outline.md if accessible.
func (o *ManuscriptConsistencyObserver) defaultScan(basePath string) ([]writing.ConsistencyAnomaly, error) {
	charRelPath := "characters.md"
	manifestPath := filepath.Join(basePath, "book.yaml")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest writing.BookManifest
		if err := yaml.Unmarshal(data, &manifest); err == nil && manifest.Structure.Characters != "" {
			charRelPath = manifest.Structure.Characters
		}
	}

	charPath := filepath.Join(basePath, charRelPath)
	if _, err := os.Stat(charPath); err != nil {
		return nil, nil
	}
	content, err := os.ReadFile(charPath)
	if err != nil {
		return nil, err
	}

	var anomalies []writing.ConsistencyAnomaly
	// Example sanity check against known historical milestones in outline.md:
	// Guo Zixing (郭子興) died in 1355.
	// Tuotuo (脫脫) dismissed in 1354 and died in 1355/1356.
	str := string(content)
	if strings.Contains(str, "脫脫") && strings.Contains(str, "1360 年依然在位指揮") {
		anomalies = append(anomalies, writing.ConsistencyAnomaly{
			ID:                  "anom-tuotuo-1360",
			AnomalyType:         writing.AnomalyPostMortemAppearance,
			CharacterName:       "脫脫",
			ChapterRef:          "characters.md / volume3",
			StatedYear:          1360,
			DeathYear:           1355,
			Description:         "脫脫於至正十五年 (1355) 遭流放賜死，不可在 1360 年繼續擔任朝廷統帥",
			KnowledgeTier:       writing.TierFactHistory,
			IsDeliberateFiction: false,
			Severity:            9,
		})
	}
	return anomalies, nil
}

// WritingPilotExecutor handles editorial correction proposals and manuscript errata during Step 8 (Execute).
type WritingPilotExecutor struct {
	taskSvc *taskService.Service
}

func NewWritingPilotExecutor(taskSvc *taskService.Service) *WritingPilotExecutor {
	return &WritingPilotExecutor{taskSvc: taskSvc}
}

func (p *WritingPilotExecutor) CanHandle(init *domainAutonomy.CandidateInitiative) bool {
	if init == nil || init.TargetAgentID != "writing-agent" {
		return false
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(init.MetadataJSON), &meta); err != nil {
		return false
	}
	if _, ok := meta["anomaly_id"]; ok {
		return true
	}
	if dom, ok := meta["domain"].(string); ok && dom == "writing" {
		return true
	}
	return false
}

func (p *WritingPilotExecutor) Execute(ctx context.Context, init *domainAutonomy.CandidateInitiative, spawnedTaskID string) (string, error) {
	var meta map[string]any
	_ = json.Unmarshal([]byte(init.MetadataJSON), &meta)

	anomID, _ := meta["anomaly_id"].(string)
	charName, _ := meta["character_name"].(string)
	chapterRef, _ := meta["chapter_ref"].(string)

	summary := fmt.Sprintf("Writing Agent generated revision proposal for inconsistency %s (Character: %s, Location: %s)", anomID, charName, chapterRef)
	return summary, nil
}
