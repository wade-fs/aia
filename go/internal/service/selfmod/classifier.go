package selfmod

import (
	"strings"

	"github.com/wade/aia/internal/domain/selfmod"
)

// ScopeClassifier enforces governance boundaries and risk tiering across files (Phase 6).
type ScopeClassifier struct{}

func NewScopeClassifier() *ScopeClassifier {
	return &ScopeClassifier{}
}

// Classify determines the risk level of touching the given target files.
func (c *ScopeClassifier) Classify(targetFiles []string) (selfmod.ModificationLevel, error) {
	level := selfmod.LevelADomain

	for _, file := range targetFiles {
		normalized := strings.TrimPrefix(file, "./")
		normalized = strings.TrimPrefix(normalized, "go/")

		// 1. Level C: Governance, Policy, Security, Secrets, Migrations (Strictly Forbidden / Immutable)
		if c.isLevelC(normalized) {
			return selfmod.LevelCGovernance, selfmod.ErrLevelCModificationForbidden
		}

		// 2. Level B: AIA Core non-governance (Human Approval Required)
		if c.isLevelB(normalized) {
			level = selfmod.LevelBCore
		}
	}

	return level, nil
}

func (c *ScopeClassifier) isLevelC(path string) bool {
	levelCPrefixes := []string{
		"internal/policy",
		"internal/domain/governance",
		"internal/service/governance",
		"internal/service/approval",
		"internal/storage/sqlite/migrations",
		".git",
	}
	for _, p := range levelCPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}

	levelCSuffixes := []string{
		"service.env",
		".db",
		".db-wal",
		".db-shm",
	}
	for _, s := range levelCSuffixes {
		if strings.HasSuffix(path, s) {
			return true
		}
	}

	return false
}

func (c *ScopeClassifier) isLevelB(path string) bool {
	levelBPrefixes := []string{
		"internal/service",
		"internal/autonomy",
		"internal/worker",
		"internal/storage",
		"internal/domain",
		"cmd",
		"internal/app",
		"internal/handler",
		"internal/workspace",
	}
	for _, p := range levelBPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}

	if path == "go.mod" || path == "go.sum" {
		return true
	}

	return false
}
