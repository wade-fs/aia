package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// ContextAssembler compiles complete, authoritative execution context for agy
// combining Agent Profile, Domain Rules, Project Brief, and Project Memory (AIA-21 §3.1, §8.4).
type ContextAssembler struct {
	projectRepo      *repository.ProjectRepository
	memoryRepo       *repository.MemoryRepository
	workflowRepo     *repository.WritingWorkflowRepository
	genericBriefRepo *repository.GenericBriefRepository
	problemRepo      *repository.ProblemRepository
}

// NewContextAssembler creates an assembler instance.
func NewContextAssembler(
	projectRepo *repository.ProjectRepository,
	memoryRepo *repository.MemoryRepository,
	workflowRepo *repository.WritingWorkflowRepository,
) *ContextAssembler {
	return &ContextAssembler{
		projectRepo:  projectRepo,
		memoryRepo:   memoryRepo,
		workflowRepo: workflowRepo,
	}
}

// SetGenericBriefRepository wires the universal project brief repository.
func (ca *ContextAssembler) SetGenericBriefRepository(gbr *repository.GenericBriefRepository) {
	ca.genericBriefRepo = gbr
}

// SetProblemRepository wires the complex problem and planning repository (AIA-22 §10.3).
func (ca *ContextAssembler) SetProblemRepository(pr *repository.ProblemRepository) {
	ca.problemRepo = pr
}

// Assemble compiles the structured markdown context prompt for agy.
func (ca *ContextAssembler) Assemble(ctx context.Context, t *task.Task, ag *agent.Agent) (string, error) {
	var sb strings.Builder

	// 1. Agent Profile & Instructions
	sb.WriteString("# AGENT PROFILE & SYSTEM INSTRUCTIONS\n")
	if ag != nil {
		sb.WriteString(fmt.Sprintf("**Agent:** %s (%s)\n", ag.Name, ag.ID))
		if ag.Domain != "" {
			sb.WriteString(fmt.Sprintf("**Domain:** %s\n", ag.Domain))
		}
		if ag.Instructions != "" {
			sb.WriteString("\n## Instructions & Operational Persona:\n")
			sb.WriteString(ag.Instructions)
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("Role: Autonomous AIA Agent\n")
	}

	// 2. Domain Guidelines & Rules
	sb.WriteString("\n---\n# DOMAIN GOVERNANCE & EXECUTION BOUNDARIES\n")
	sb.WriteString("1. Level C Governance Immutability: Do not attempt self-approval, policy evasion, or unauthorized state changes.\n")
	sb.WriteString("2. Strict Workspace Isolation: All operations, artifacts, and edits MUST reside inside the designated project workspace.\n")
	sb.WriteString("3. Verification Gate: Distinguish verified facts (L2) from plausible inferences and artistic licenses.\n")

	// 3. Project Brief & Living Requirements
	projID := string(t.ProjectID)
	if projID != "" {
		sb.WriteString(fmt.Sprintf("\n---\n# PROJECT CONTEXT & BRIEF (Project: %s)\n", projID))

		// Check database brief
		var briefLoaded bool
		if ca.genericBriefRepo != nil {
			if gb, err := ca.genericBriefRepo.GetByProjectID(ctx, projID); err == nil && gb != nil {
				briefLoaded = true
				sb.WriteString(fmt.Sprintf("**Domain:** %s\n", gb.Domain))
				sb.WriteString(fmt.Sprintf("**Brief Version:** %d\n", gb.BriefVersion))
				sb.WriteString(fmt.Sprintf("**Readiness Stage:** %s (%s)\n", gb.Readiness, gb.ReadinessRationale))
				if gb.RawIntent != "" {
					sb.WriteString(fmt.Sprintf("**Raw Intent:** %s\n", gb.RawIntent))
				}
				if len(gb.Fields) > 0 {
					sb.WriteString("\n### Brief Field Specifications & States:\n")
					for k, f := range gb.Fields {
						sb.WriteString(fmt.Sprintf("- **%s**: %v [Status: %s, Source: %s, Conf: %.2f] — %s\n",
							k, f.Value, f.Status, f.Source, f.Confidence, f.Rationale))
					}
				}
			}
		}

		if !briefLoaded && ca.workflowRepo != nil {
			if brief, err := ca.workflowRepo.GetBriefByProjectID(ctx, projID); err == nil && brief != nil {
				briefLoaded = true
				sb.WriteString(fmt.Sprintf("**Title:** %s\n", brief.Title))
				sb.WriteString(fmt.Sprintf("**Status:** %s (Version: %d)\n", brief.Status, brief.Version))
				sb.WriteString(fmt.Sprintf("**Genre:** %s\n", brief.Genre))
				if brief.Era != "" {
					sb.WriteString(fmt.Sprintf("**Era / Historical Setting:** %s\n", brief.Era))
				}
				sb.WriteString(fmt.Sprintf("**Scale:** %d volumes, %d total chapters (%d chapters/vol)\n",
					brief.Scale.TotalVolumes, brief.Scale.TotalChapters, brief.Scale.ChaptersPerVolume))
				if len(brief.ThemeAndPhilosophy) > 0 {
					sb.WriteString("**Worldview & Core Principles:**\n")
					for _, th := range brief.ThemeAndPhilosophy {
						sb.WriteString(fmt.Sprintf("  • %s\n", th))
					}
				}
				if len(brief.CoreProtagonists) > 0 {
					sb.WriteString("**Core Characters / Protagonists:**\n")
					for _, cp := range brief.CoreProtagonists {
						sb.WriteString(fmt.Sprintf("  • %s\n", cp))
					}
				}
				if brief.RawIntent != "" {
					sb.WriteString(fmt.Sprintf("\n**Original Author Intent:**\n%s\n", brief.RawIntent))
				}
			}
		}

		// Also check workspace filesystem brief.md if present
		if ca.projectRepo != nil {
			if p, err := ca.projectRepo.Get(ctx, t.ProjectID); err == nil && p != nil {
				sb.WriteString(fmt.Sprintf("**Workspace Root:** %s\n", p.Workspace))
				if !briefLoaded && p.Workspace != "" {
					briefPath := filepath.Join(p.Workspace, "brief.md")
					if data, rErr := os.ReadFile(briefPath); rErr == nil && len(data) > 0 {
						sb.WriteString("\n**Brief Document (brief.md):**\n")
						sb.WriteString(string(data))
						sb.WriteString("\n")
					}
				}
			}
		}

		// 4. Project Memory & Recent Prior Decisions
		if ca.memoryRepo != nil {
			memories, err := ca.memoryRepo.Search(ctx, repository.MemoryQuery{
				ProjectID: t.ProjectID,
				Limit:     8,
			})
			if err == nil && len(memories) > 0 {
				sb.WriteString("\n---\n# PROJECT MEMORY & PRIOR DECISIONS\n")
				for _, m := range memories {
					sb.WriteString(fmt.Sprintf("- [%s] **%s**: %s\n", m.Type, m.Title, m.Content))
				}
			}
		}

		// 4.1 Active Complex Problems & Planning (AIA-22 §10.3)
		if ca.problemRepo != nil {
			if probs, err := ca.problemRepo.ListByProject(ctx, t.ProjectID); err == nil && len(probs) > 0 {
				sb.WriteString("\n---\n# ACTIVE PROBLEMS & PLANNING CONTEXT\n")
				for _, pr := range probs {
					if pr.Status != "RESOLVED" && pr.Status != "CLOSED" {
						sb.WriteString(fmt.Sprintf("- **Problem [%s]**: %s (Status: %s, Priority: %s)\n", pr.ID, pr.Title, pr.Status, pr.Priority))
						if pr.Objective != "" {
							sb.WriteString(fmt.Sprintf("  • Objective: %s\n", pr.Objective))
						}
					}
				}
			}
		}
	}

	// 5. Task Acceptance & Constraints
	if len(t.Acceptance) > 0 {
		sb.WriteString("\n---\n# ACCEPTANCE CRITERIA\n")
		for _, a := range t.Acceptance {
			sb.WriteString(fmt.Sprintf("- %s\n", a))
		}
	}

	return sb.String(), nil
}
