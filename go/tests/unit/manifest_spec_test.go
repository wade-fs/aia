package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wade/aia/internal/domain/writing"
	"gopkg.in/yaml.v3"
)

func TestBookManifest_RealNovels_FullVerification(t *testing.T) {
	cases := []struct {
		id               string
		dir              string
		expectedName     string
		expectedVolumes  int
		expectedChapters int
		hasDualWorld     bool
		hasWorldview     bool
	}{
		{
			id:               "tang",
			dir:              "/opt/src/github/novel2movie/tang",
			expectedName:     "唐末",
			expectedVolumes:  16,
			expectedChapters: 1440,
			hasDualWorld:     true,
			hasWorldview:     false,
		},
		{
			id:               "song",
			dir:              "/opt/src/github/novel2movie/song",
			expectedName:     "宋末",
			expectedVolumes:  8,
			expectedChapters: 720,
			hasDualWorld:     false,
			hasWorldview:     false,
		},
		{
			id:               "yuan",
			dir:              "/opt/src/github/novel2movie/yuan",
			expectedName:     "元末明初歷史群像小說",
			expectedVolumes:  6,
			expectedChapters: 540,
			hasDualWorld:     true,
			hasWorldview:     false,
		},
		{
			id:               "ming",
			dir:              "/opt/src/github/novel2movie/ming",
			expectedName:     "明末清初歷史小說",
			expectedVolumes:  9,
			expectedChapters: 810,
			hasDualWorld:     false,
			hasWorldview:     false,
		},
		{
			id:               "ching",
			dir:              "/opt/src/github/novel2movie/ching",
			expectedName:     "天下易主",
			expectedVolumes:  16,
			expectedChapters: 1440,
			hasDualWorld:     false,
			hasWorldview:     false,
		},
		{
			id:               "xiu-xian",
			dir:              "/opt/src/github/novel2movie/xiu-xian",
			expectedName:     "求理與歷劫",
			expectedVolumes:  0,
			expectedChapters: 0,
			hasDualWorld:     false,
			hasWorldview:     true,
		},
		{
			id:               "aia-yuan",
			dir:              "/opt/src/github/aia/novel/yuan",
			expectedName:     "元末明初歷史群像小說",
			expectedVolumes:  6,
			expectedChapters: 540,
			hasDualWorld:     true,
			hasWorldview:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			manifestPath := filepath.Join(tc.dir, "book.yaml")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", manifestPath, err)
			}

			var manifest writing.BookManifest
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				t.Fatalf("failed to unmarshal yaml for %s: %v", tc.id, err)
			}

			// 1. Validate domain invariants
			if err := manifest.Validate(); err != nil {
				t.Fatalf("manifest.Validate() failed for %s: %v", tc.id, err)
			}

			// 2. Validate Project Meta
			if manifest.Project.Name != tc.expectedName {
				t.Errorf("[%s] expected name %q, got %q", tc.id, tc.expectedName, manifest.Project.Name)
			}
			if manifest.Project.Language != "zh-TW" {
				t.Errorf("[%s] expected language zh-TW, got %q", tc.id, manifest.Project.Language)
			}

			// 3. Validate scale
			if manifest.GetVolumeCount() != tc.expectedVolumes {
				t.Errorf("[%s] expected %d volumes, got %d", tc.id, tc.expectedVolumes, manifest.GetVolumeCount())
			}
			if manifest.GetChapterCount() != tc.expectedChapters {
				t.Errorf("[%s] expected %d chapters, got %d", tc.id, tc.expectedChapters, manifest.GetChapterCount())
			}

			// 4. Validate dual world
			if manifest.HasDualWorld() != tc.hasDualWorld {
				t.Errorf("[%s] expected HasDualWorld=%v, got %v", tc.id, tc.hasDualWorld, manifest.HasDualWorld())
			}

			// 5. Validate worldview
			if tc.hasWorldview {
				if manifest.Worldview == nil || len(manifest.Worldview.FormalRealms) == 0 {
					t.Fatalf("[%s] expected worldview realms, got nil or empty", tc.id)
				}
			}

			// 6. Verify declared files actually exist on disk
			checkFileExists := func(name, relPath string) {
				if relPath == "" || relPath == "." {
					return
				}
				targetPath := filepath.Join(tc.dir, relPath)
				if _, err := os.Stat(targetPath); err != nil {
					t.Errorf("[%s] declared %s file %q does not exist: %v", tc.id, name, targetPath, err)
				}
			}

			s := manifest.Structure
			checkFileExists("outline", s.Outline)
			checkFileExists("volumes_plan", s.VolumesPlan)
			checkFileExists("timeline", s.Timeline)
			checkFileExists("timeline_audit", s.TimelineAudit)
			checkFileExists("chapter_ledger", s.ChapterLedger)
			checkFileExists("narrative_validation", s.NarrativeValidation)
			checkFileExists("research_qa", s.ResearchQA)
			checkFileExists("master_bible", s.MasterBible)
			checkFileExists("rules", s.Rules)
			checkFileExists("guidelines", s.Guidelines)
			checkFileExists("characters", s.Characters)
			checkFileExists("character_network", s.CharacterNetwork)
			checkFileExists("character_lifecycle", s.CharacterLifecycle)
			checkFileExists("geography", s.Geography)
			checkFileExists("social_systems", s.SocialSystems)
			checkFileExists("military_reference", s.MilitaryReference)
			checkFileExists("bibliography", s.Bibliography)
			checkFileExists("evidence_review", s.EvidenceReview)

			if manifest.Laboratory != nil && manifest.Laboratory.Enabled {
				checkFileExists("lab_spec", manifest.Laboratory.Specification)
				checkFileExists("lab_guidelines", manifest.Laboratory.Guidelines)
				checkFileExists("lab_volume_map", manifest.Laboratory.VolumeMap)
				checkFileExists("lab_integration", manifest.Laboratory.Integration)
			}
		})
	}
}

func TestBookManifest_ValidationFailures(t *testing.T) {
	// Missing Project ID
	m1 := writing.BookManifest{}
	m1.Project.Name = "Test"
	m1.Structure.Outline = "outline.md"
	if err := m1.Validate(); err == nil {
		t.Errorf("expected error for missing project ID")
	}

	// Missing Project Name
	m2 := writing.BookManifest{}
	m2.Project.ID = "test"
	m2.Structure.Outline = "outline.md"
	if err := m2.Validate(); err == nil {
		t.Errorf("expected error for missing project Name")
	}

	// Missing Outline
	m3 := writing.BookManifest{}
	m3.Project.ID = "test"
	m3.Project.Name = "Test"
	if err := m3.Validate(); err == nil {
		t.Errorf("expected error for missing outline")
	}
}
