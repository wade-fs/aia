package project

import (
	"fmt"
	"strings"

	"github.com/wade/aia/internal/domain/writing"
	"gopkg.in/yaml.v3"
)

// ParseProjectManifest parses project manifest data, supporting both universal ProjectManifest
// and legacy BookManifest formats seamlessly (Plan2.md §5).
func ParseProjectManifest(data []byte) (*ProjectManifest, error) {
	// First check if this is a legacy BookManifest (has structure or scale sections)
	dataStr := string(data)
	if strings.Contains(dataStr, "structure:") || strings.Contains(dataStr, "total_volumes:") || strings.Contains(dataStr, "chapters_per_volume:") {
		var bm writing.BookManifest
		if err := yaml.Unmarshal(data, &bm); err == nil && (bm.Structure.Outline != "" || bm.Structure.Characters != "" || (bm.Scale != nil && bm.Scale.TotalVolumes > 0)) {
			// Convert legacy BookManifest to universal ProjectManifest
			pm := &ProjectManifest{}
			pm.Project.ID = bm.Project.ID
			pm.Project.Name = bm.Project.Name
			pm.Project.Type = bm.Project.Type
			pm.Project.DomainAgent = bm.Project.DomainAgent
			pm.Project.Language = bm.Project.Language

			pm.Profile = ManifestProfile{
				Name:    ProfileHistoricalFiction,
				Version: "1.0",
			}

			if bm.Storage != nil {
				pm.Storage.Type = bm.Storage.Type
				pm.Storage.Root = bm.Storage.Root
			}

			if bm.VersionControl != nil {
				pm.VersionControl.Type = bm.VersionControl.Type
				pm.VersionControl.Enabled = bm.VersionControl.Enabled
				pm.VersionControl.DefaultBranch = bm.VersionControl.DefaultBranch
			}

			pm.Documents.Outline = bm.Structure.Outline
			pm.Documents.Characters = bm.Structure.Characters
			pm.Documents.Chapters = bm.Structure.Chapters

			pm.Rules = []string{"historical_accuracy", "temporal_consistency", "character_continuity"}

			if bm.Continuity != nil {
				pm.Continuity.Enabled = bm.Continuity.Enabled
			}
			if bm.Memory != nil {
				pm.Memory.Enabled = bm.Memory.Enabled
			}

			if bm.Governance != nil {
				pm.Governance.DirectMainWrite = bm.Governance.DirectMainWrite
				pm.Governance.RequireDiffReview = bm.Governance.RequireDiffReview
				pm.Governance.RequireHumanApprovalForMajorChanges = bm.Governance.RequireHumanApprovalForMajorChanges
			}

			return pm, nil
		}
	}

	// Otherwise, parse standard universal ProjectManifest
	var pm ProjectManifest
	if err := yaml.Unmarshal(data, &pm); err != nil {
		return nil, fmt.Errorf("failed to parse ProjectManifest: %w", err)
	}
	return &pm, nil
}
