package writing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/wade/aia/internal/domain/consistency"
	"github.com/wade/aia/internal/domain/knowledge"
	"gopkg.in/yaml.v3"
)

// ChapterChronology captures temporal and character occurrence in a chapter (Plan.md §5.3).
type ChapterChronology struct {
	VolumeNum          int      `json:"volume_num"`
	ChapterNum         int      `json:"chapter_num"`
	ChapterRef         string   `json:"chapter_ref"`
	Title              string   `json:"title"`
	StatedYear         int      `json:"stated_year"`
	StatedMonth        int      `json:"stated_month,omitempty"`
	MentionedCharacters []string `json:"mentioned_characters"`
	KnowledgeTier      KnowledgeTier `json:"knowledge_tier"`
}

// ConsistencyEngine implements general-purpose cross-document manuscript validation (Plan.md §5.3).
// It replaces demo-style hardcoded checks with dynamic entity and chronology parsing.
type ConsistencyEngine struct{}

func NewConsistencyEngine() *ConsistencyEngine {
	return &ConsistencyEngine{}
}

var (
	reHeaderChar = regexp.MustCompile(`(?m)^##\s+([^\n\r]+)`)
	reBulletChar = regexp.MustCompile(`(?m)^[-*]\s*(?:\*\*)?([^\s:：*]+)(?:\*\*)?[:：]\s*(.*)$`)
	reYearSpan   = regexp.MustCompile(`(?:\(|\b)(13[0-9]{2})\s*[-–—‑至到]\s*(13[0-9]{2})(?:\)|\b)`)
	reSingleYear = regexp.MustCompile(`\b(13[0-9]{2})\b`)
	reChapHeader = regexp.MustCompile(`(?m)^####\s+第(\d+)章\s+([^\n\r]+)`)
	reTimeLoc    = regexp.MustCompile(`(?m)-\s*\*\*時地\*\*：([^\n\r]+)`)
	rePersp      = regexp.MustCompile(`(?m)-\s*\*\*視角\*\*：([^\n\r]+)`)
	reTierFact   = regexp.MustCompile(`【史實骨架】`)
	reTierInfer  = regexp.MustCompile(`【合理推演】`)
	reTierFic    = regexp.MustCompile(`【小說虛構】`)
)

// CanonicalHistoricalFigures provides historical baseline lifespans (Plan.md §5.3).
var CanonicalHistoricalFigures = map[string]HistoricalCharacter{
	"脫脫": {
		ID:            "脫脫",
		Name:          "脫脫",
		DeathYear:     1355,
		BirthYear:     1314,
		KnowledgeTier: TierFactHistory,
	},
	"郭子興": {
		ID:            "郭子興",
		Name:          "郭子興",
		DeathYear:     1355,
		BirthYear:     1297,
		KnowledgeTier: TierFactHistory,
	},
	"朱元璋": {
		ID:            "朱元璋",
		Name:          "朱元璋",
		DeathYear:     1398,
		BirthYear:     1328,
		KnowledgeTier: TierFactHistory,
	},
	"陳友諒": {
		ID:            "陳友諒",
		Name:          "陳友諒",
		DeathYear:     1363,
		BirthYear:     1320,
		KnowledgeTier: TierFactHistory,
	},
	"張士誠": {
		ID:            "張士誠",
		Name:          "張士誠",
		DeathYear:     1367,
		BirthYear:     1321,
		KnowledgeTier: TierFactHistory,
	},
	"徐達": {
		ID:            "徐達",
		Name:          "徐達",
		DeathYear:     1385,
		BirthYear:     1332,
		KnowledgeTier: TierFactHistory,
	},
	"常遇春": {
		ID:            "常遇春",
		Name:          "常遇春",
		DeathYear:     1369,
		BirthYear:     1330,
		KnowledgeTier: TierFactHistory,
	},
	"妥懽帖睦爾": {
		ID:            "妥懽帖睦爾",
		Name:          "妥懽帖睦爾",
		DeathYear:     1370,
		BirthYear:     1320,
		KnowledgeTier: TierFactHistory,
		Aliases:       []string{"元順帝"},
	},
}

// ParseCharacters extracts character profiles, lifespans and knowledge tiers from characters.md content.
func (e *ConsistencyEngine) ParseCharacters(content string) []HistoricalCharacter {
	var characters []HistoricalCharacter

	sections := strings.Split(content, "\n## ")
	for i, sec := range sections {
		if i == 0 && !strings.HasPrefix(content, "## ") {
			// Skip title / preface before first character
			continue
		}
		lines := strings.Split(sec, "\n")
		rawName := strings.TrimSpace(lines[0])
		name := strings.TrimPrefix(rawName, "## ")
		if name == "" || name == "目錄" || strings.HasPrefix(name, "增補") || name == "角色檔案（元朝明初）" {
			continue
		}
		// Clean any parenthetical alias from name header, e.g. "元順帝（妥懽帖睦爾）" -> "元順帝"
		cleanName := name
		if idx := strings.IndexAny(cleanName, "（("); idx != -1 {
			cleanName = strings.TrimSpace(cleanName[:idx])
		}

		char := HistoricalCharacter{
			ID:             cleanName,
			Name:           cleanName,
			HistoricalRole: "歷史人物",
			KnowledgeTier:  TierFactHistory,
		}

		if canon, ok := CanonicalHistoricalFigures[cleanName]; ok {
			char.BirthYear = canon.BirthYear
			char.DeathYear = canon.DeathYear
			char.KnowledgeTier = canon.KnowledgeTier
			char.Aliases = append(char.Aliases, canon.Aliases...)
		}

		secText := sec
		// Determine Knowledge Tier
		if reTierFic.MatchString(secText) || strings.Contains(secText, "虛構") || strings.Contains(secText, "方外") {
			char.KnowledgeTier = TierFiction
		} else if reTierInfer.MatchString(secText) {
			char.KnowledgeTier = TierPlausibleInference
		}

		// Parse birth and death years
		// Check for range e.g. (1328–1398) or (1314‑1340)
		if matches := reYearSpan.FindAllStringSubmatch(secText, -1); len(matches) > 0 {
			first := matches[0]
			bYear, _ := strconv.Atoi(first[1])
			dYear, _ := strconv.Atoi(first[2])
			char.BirthYear = bYear
			// Check if later sections record a later death year (e.g. 1355-1356 in death stage)
			for _, m := range matches {
				y2, _ := strconv.Atoi(m[2])
				if y2 > dYear {
					dYear = y2
				}
			}
			char.DeathYear = dYear
		} else if yMatches := reSingleYear.FindAllStringSubmatch(secText, -1); len(yMatches) > 0 {
			bYear, _ := strconv.Atoi(yMatches[0][1])
			if char.BirthYear == 0 {
				char.BirthYear = bYear
			}
		}

		// Special death year markers: e.g. "流放與死亡 (1355‑1356)" or "1355 年...身亡"
		if strings.Contains(secText, "死亡") || strings.Contains(secText, "病亡") || strings.Contains(secText, "去世") || strings.Contains(secText, "賜死") {
			deathMatches := regexp.MustCompile(`(?:流放與死亡|去世|身亡|賜死)[^\n\r]*\b(13[0-9]{2})\b`).FindStringSubmatch(secText)
			if len(deathMatches) > 1 {
				dYear, _ := strconv.Atoi(deathMatches[1])
				char.DeathYear = dYear
			}
		}

		// Also extract aliases from **別稱**
		if aliasMatch := regexp.MustCompile(`(?m)\*\*別稱\*\*：([^\n\r]+)`).FindStringSubmatch(secText); len(aliasMatch) > 1 {
			rawAliases := aliasMatch[1]
			for _, a := range strings.Split(rawAliases, "、") {
				for _, subA := range strings.Split(a, "｜") {
					cleaned := strings.TrimSpace(subA)
					if cleaned != "" && cleaned != cleanName {
						char.Aliases = append(char.Aliases, cleaned)
					}
				}
			}
		}

		characters = append(characters, char)
	}

	// Also support bullet list entries (e.g. `- 脫脫: 元朝重臣。1360 年依然在位指揮大軍。`)
	if bulletMatches := reBulletChar.FindAllStringSubmatch(content, -1); len(bulletMatches) > 0 {
		for _, bm := range bulletMatches {
			rawName := strings.TrimSpace(bm[1])
			desc := strings.TrimSpace(bm[2])
			cleanName := rawName
			if idx := strings.IndexAny(cleanName, "（("); idx != -1 {
				cleanName = strings.TrimSpace(cleanName[:idx])
			}
			if cleanName == "" || cleanName == "生卒年" || cleanName == "生年" || cleanName == "卒年" ||
				cleanName == "身分" || cleanName == "身份" || cleanName == "標記" || cleanName == "時地" ||
				cleanName == "視角" || cleanName == "別稱" {
				continue
			}

			foundIdx := -1
			for idx, ch := range characters {
				if ch.Name == cleanName {
					foundIdx = idx
					break
				}
			}

			if foundIdx >= 0 {
				characters[foundIdx].KeyEvents = append(characters[foundIdx].KeyEvents, desc)
			} else {
				c := HistoricalCharacter{
					ID:             cleanName,
					Name:           cleanName,
					HistoricalRole: "歷史人物",
					KnowledgeTier:  TierFactHistory,
					KeyEvents:      []string{desc},
				}
				if canon, ok := CanonicalHistoricalFigures[cleanName]; ok {
					c.BirthYear = canon.BirthYear
					c.DeathYear = canon.DeathYear
					c.KnowledgeTier = canon.KnowledgeTier
					c.Aliases = append(c.Aliases, canon.Aliases...)
				}
				if reTierFic.MatchString(desc) || strings.Contains(desc, "虛構") {
					c.KnowledgeTier = TierFiction
				} else if reTierInfer.MatchString(desc) {
					c.KnowledgeTier = TierPlausibleInference
				}
				if matches := reYearSpan.FindAllStringSubmatch(desc, -1); len(matches) > 0 {
					b, _ := strconv.Atoi(matches[0][1])
					d, _ := strconv.Atoi(matches[0][2])
					c.BirthYear = b
					c.DeathYear = d
				}
				characters = append(characters, c)
			}
		}
	}

	return characters
}

// ParseChapterChronology parses chapter sections from volume markdown files.
func (e *ConsistencyEngine) ParseChapterChronology(content string, volumeNum int) []ChapterChronology {
	var chapters []ChapterChronology

	matches := reChapHeader.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return chapters
	}

	for i, m := range matches {
		chapNum, _ := strconv.Atoi(content[m[2]:m[3]])
		title := strings.TrimSpace(content[m[4]:m[5]])

		var body string
		if i+1 < len(matches) {
			body = content[m[0]:matches[i+1][0]]
		} else {
			body = content[m[0]:]
		}

		chap := ChapterChronology{
			VolumeNum:  volumeNum,
			ChapterNum: chapNum,
			ChapterRef: fmt.Sprintf("volume%d/ch%02d", volumeNum, chapNum),
			Title:      title,
		}

		// Extract Year from 時地
		if tlMatch := reTimeLoc.FindStringSubmatch(body); len(tlMatch) > 1 {
			timeStr := tlMatch[1]
			if yMatch := reSingleYear.FindStringSubmatch(timeStr); len(yMatch) > 1 {
				year, _ := strconv.Atoi(yMatch[1])
				chap.StatedYear = year
			}
		}

		// Extract Knowledge Tier
		if reTierFic.MatchString(body) {
			chap.KnowledgeTier = TierFiction
		} else if reTierInfer.MatchString(body) {
			chap.KnowledgeTier = TierPlausibleInference
		} else {
			chap.KnowledgeTier = TierFactHistory
		}

		// Extract perspective character if present
		if pMatch := rePersp.FindStringSubmatch(body); len(pMatch) > 1 {
			persp := pMatch[1]
			for _, part := range strings.Split(persp, "、") {
				name := strings.TrimSpace(strings.Split(part, "（")[0])
				if name != "" {
					chap.MentionedCharacters = append(chap.MentionedCharacters, name)
				}
			}
		}

		chapters = append(chapters, chap)
	}

	return chapters
}

// ScanProject inspects characters and chapters against historical and fictional constraints.
func (e *ConsistencyEngine) ScanProject(wsRoot string, manifest *BookManifest) ([]ConsistencyAnomaly, error) {
	if manifest == nil {
		manifestPath := filepath.Join(wsRoot, "book.yaml")
		data, err := os.ReadFile(manifestPath)
		if err == nil {
			var m BookManifest
			if yErr := yaml.Unmarshal(data, &m); yErr == nil {
				manifest = &m
			}
		}
	}

	charRelPath := "characters.md"
	if manifest != nil && manifest.Structure.Characters != "" {
		charRelPath = manifest.Structure.Characters
	}
	charPath := filepath.Join(wsRoot, charRelPath)
	charData, err := os.ReadFile(charPath)
	if err != nil {
		return nil, fmt.Errorf("characters file not accessible: %w", err)
	}

	characters := e.ParseCharacters(string(charData))
	charMap := make(map[string]HistoricalCharacter)
	for _, c := range characters {
		charMap[c.Name] = c
		for _, alias := range c.Aliases {
			charMap[alias] = c
		}
	}

	var allChapters []ChapterChronology
	// Scan volume files (volume1.md .. volume6.md)
	for v := 1; v <= 20; v++ {
		vFile := filepath.Join(wsRoot, fmt.Sprintf("volume%d.md", v))
		vData, err := os.ReadFile(vFile)
		if err != nil {
			continue
		}
		chaps := e.ParseChapterChronology(string(vData), v)
		allChapters = append(allChapters, chaps...)
	}

	var anomalies []ConsistencyAnomaly

	// Cross-check characters against chapter chronology
	for _, chap := range allChapters {
		if chap.StatedYear <= 0 {
			continue
		}

		for _, charName := range chap.MentionedCharacters {
			c, ok := charMap[charName]
			if !ok {
				continue
			}

			// 1. Post-Mortem Appearance Contradiction
			if c.KnowledgeTier == TierFactHistory && c.DeathYear > 0 && chap.StatedYear > c.DeathYear {
				anomalies = append(anomalies, ConsistencyAnomaly{
					ID:                  fmt.Sprintf("anom-%s-%d-%s", c.Name, chap.StatedYear, chap.ChapterRef),
					AnomalyType:         AnomalyPostMortemAppearance,
					CharacterName:       c.Name,
					ChapterRef:          fmt.Sprintf("%s (%s)", chap.ChapterRef, chap.Title),
					StatedYear:          chap.StatedYear,
					DeathYear:           c.DeathYear,
					Description:         fmt.Sprintf("史實人物 %s 卒於 %d 年，但於 %d 年章節登場統軍或行動", c.Name, c.DeathYear, chap.StatedYear),
					KnowledgeTier:       TierFactHistory,
					IsDeliberateFiction: false,
					Severity:            9,
				})
			}

			// 2. Pre-Birth Appearance Anachronism
			if c.KnowledgeTier == TierFactHistory && c.BirthYear > 0 && chap.StatedYear < c.BirthYear {
				anomalies = append(anomalies, ConsistencyAnomaly{
					ID:                  fmt.Sprintf("anom-anach-%s-%d", c.Name, chap.StatedYear),
					AnomalyType:         AnomalyAnachronism,
					CharacterName:       c.Name,
					ChapterRef:          fmt.Sprintf("%s (%s)", chap.ChapterRef, chap.Title),
					StatedYear:          chap.StatedYear,
					BirthYear:           c.BirthYear,
					Description:         fmt.Sprintf("史實人物 %s 生於 %d 年，但提早於 %d 年章節登場", c.Name, c.BirthYear, chap.StatedYear),
					KnowledgeTier:       TierFactHistory,
					IsDeliberateFiction: false,
					Severity:            8,
				})
			}

			// 3. Intentional Fiction License Tracking
			if c.KnowledgeTier == TierFiction {
				anomalies = append(anomalies, ConsistencyAnomaly{
					ID:                  fmt.Sprintf("anom-fic-%s-%s", c.Name, chap.ChapterRef),
					AnomalyType:         AnomalyFictionDeviation,
					CharacterName:       c.Name,
					ChapterRef:          fmt.Sprintf("%s (%s)", chap.ChapterRef, chap.Title),
					StatedYear:          chap.StatedYear,
					Description:         fmt.Sprintf("藝術虛構人物 %s 登場於 %s（合理小說虛構）", c.Name, chap.ChapterRef),
					KnowledgeTier:       TierFiction,
					IsDeliberateFiction: true,
					Severity:            2,
				})
			}
		}
	}

	// Also inspect character profile entries for internal chronological contradictions
	for _, c := range characters {
		if c.KnowledgeTier == TierFactHistory && c.DeathYear > 0 {
			for _, ev := range c.KeyEvents {
				for _, ym := range reSingleYear.FindAllStringSubmatch(ev, -1) {
					if len(ym) > 1 {
						y, _ := strconv.Atoi(ym[1])
						if y > c.DeathYear {
							anomalies = append(anomalies, ConsistencyAnomaly{
								ID:                  fmt.Sprintf("anom-%s-%d-char", c.Name, y),
								AnomalyType:         AnomalyPostMortemAppearance,
								CharacterName:       c.Name,
								ChapterRef:          charRelPath,
								StatedYear:          y,
								DeathYear:           c.DeathYear,
								Description:         fmt.Sprintf("史實人物 %s 卒於 %d 年，但人物設定描述在 %d 年活動: %s", c.Name, c.DeathYear, y, ev),
								KnowledgeTier:       TierFactHistory,
								IsDeliberateFiction: false,
								Severity:            9,
							})
						}
					}
				}
			}
		}
	}

	return anomalies, nil
}

// BuildKnowledgeSnapshot converts parsed characters and chapter chronologies into universal Knowledge Core models (Plan2.md §6).
func (e *ConsistencyEngine) BuildKnowledgeSnapshot(characters []HistoricalCharacter, chapters []ChapterChronology) *consistency.KnowledgeSnapshot {
	snapshot := consistency.NewKnowledgeSnapshot()

	for _, c := range characters {
		ent := knowledge.NewEntity(c.ID, c.Name, knowledge.EntityCharacter, "writing")
		ent.Aliases = c.Aliases
		ent.Description = c.HistoricalRole
		ent.Attributes["death_year"] = c.DeathYear
		ent.Attributes["birth_year"] = c.BirthYear
		ent.Attributes["is_fiction"] = (c.KnowledgeTier == TierFiction)

		snapshot.Entities[c.Name] = ent
		for _, a := range c.Aliases {
			snapshot.Entities[a] = ent
		}

		for _, ev := range c.KeyEvents {
			for _, ym := range reSingleYear.FindAllStringSubmatch(ev, -1) {
				if len(ym) > 1 {
					y, _ := strconv.Atoi(ym[1])
					snapshot.Assertions = append(snapshot.Assertions, knowledge.Assertion{
						ID:              fmt.Sprintf("asrt-%s-%d", c.Name, y),
						SubjectEntityID: c.Name,
						Predicate:       "active_in_key_event",
						ObjectValue:     ev,
						Epistemic:       knowledge.StatusSourced,
						DomainClass:     knowledge.ClassFactHistory,
						Temporal:        &knowledge.TemporalReference{AnchorYear: &y},
						SourceFile:      "characters.md",
					})
				}
			}
		}
	}

	for _, chap := range chapters {
		y := chap.StatedYear
		for _, charName := range chap.MentionedCharacters {
			snapshot.Assertions = append(snapshot.Assertions, knowledge.Assertion{
				ID:                  fmt.Sprintf("asrt-chap-%s-%s", chap.ChapterRef, charName),
				SubjectEntityID:     charName,
				Predicate:           "appears_in_chapter",
				ObjectValue:         chap.Title,
				Temporal:            &knowledge.TemporalReference{AnchorYear: &y},
				SourceFile:          fmt.Sprintf("%s (%s)", chap.ChapterRef, chap.Title),
				IsDeliberateFiction: (chap.KnowledgeTier == TierFiction),
			})
		}
	}

	return snapshot
}

// ScanProjectSnapshot scans a project and returns both universal KnowledgeSnapshot and Findings (Plan2.md §4, §6).
func (e *ConsistencyEngine) ScanProjectSnapshot(ctx context.Context, wsRoot string, manifest *BookManifest) (*consistency.KnowledgeSnapshot, []consistency.Finding, error) {
	if manifest == nil {
		manifestPath := filepath.Join(wsRoot, "book.yaml")
		data, err := os.ReadFile(manifestPath)
		if err == nil {
			var m BookManifest
			if yErr := yaml.Unmarshal(data, &m); yErr == nil {
				manifest = &m
			}
		}
	}

	charRelPath := "characters.md"
	if manifest != nil && manifest.Structure.Characters != "" {
		charRelPath = manifest.Structure.Characters
	}
	charPath := filepath.Join(wsRoot, charRelPath)
	charData, err := os.ReadFile(charPath)
	if err != nil {
		return nil, nil, fmt.Errorf("characters file not accessible: %w", err)
	}

	characters := e.ParseCharacters(string(charData))

	var allChapters []ChapterChronology
	for v := 1; v <= 20; v++ {
		vFile := filepath.Join(wsRoot, fmt.Sprintf("volume%d.md", v))
		vData, err := os.ReadFile(vFile)
		if err != nil {
			continue
		}
		chaps := e.ParseChapterChronology(string(vData), v)
		allChapters = append(allChapters, chaps...)
	}

	snapshot := e.BuildKnowledgeSnapshot(characters, allChapters)
	universalEngine := consistency.NewEngine()
	findings, err := universalEngine.Evaluate(ctx, snapshot)
	if err != nil {
		return snapshot, nil, err
	}

	return snapshot, findings, nil
}
