package brief

import (
	"context"
	"fmt"

	"github.com/wade/aia/internal/domain"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	domainClarify "github.com/wade/aia/internal/domain/clarification"
)

// ClarificationCreator defines the interface for creating clarification requests.
type ClarificationCreator interface {
	CreateRequests(ctx context.Context, requests []*domainClarify.ClarificationRequest) error
}

// ApplyAnalysis validates and applies an AgentBriefAnalysis to the Project Brief (AIA-21 §6.3, §7.1).
func (s *Service) ApplyAnalysis(
	ctx context.Context,
	projectID string,
	analysis *domainBrief.AgentBriefAnalysis,
	clarifyCreator ClarificationCreator,
	taskID string,
	wsRoot string,
) (*domainBrief.ProjectBrief, error) {
	b, err := s.briefRepo.GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to load project brief: %w", err)
	}
	if b == nil {
		return nil, fmt.Errorf("project brief for %s not found", projectID)
	}

	catalog, err := domainBrief.GetCatalog(b.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain catalog for %s: %w", b.Domain, err)
	}

	// 1. Populate projectID for questions if omitted, then validate against catalog & discipline limits (AIA-21 §6.3)
	for _, q := range analysis.ClarificationQuestions {
		if q.ProjectID == "" {
			q.ProjectID = projectID
		}
	}

	if err := domainBrief.ValidateBriefAnalysis(analysis, catalog); err != nil {
		return nil, fmt.Errorf("brief analysis validation failed: %w", err)
	}

	if wsRoot == "" && s.projectRepo != nil {
		if p, _ := s.projectRepo.Get(ctx, domain.ID(projectID)); p != nil {
			wsRoot = p.Workspace
		}
	}

	// 2. Apply proposed defaults as provisional hypotheses
	for _, pd := range analysis.ProposedDefaults {
		rationale := fmt.Sprintf("Agent 提議預設值: %s", pd.Rationale)
		b.SetField(
			pd.Field,
			pd.Value,
			domainBrief.FieldStatusDefaulted,
			"agent_recommendation",
			pd.Confidence,
			rationale,
			true, // Confirmation required
			"agent",
			taskID,
			"",
		)
	}

	// 3. Mark fields with questions as ASKED
	for _, q := range analysis.ClarificationQuestions {
		q.ProjectID = projectID
		if q.TaskID == "" {
			q.TaskID = taskID
		}
		if f, exists := b.Fields[q.FieldKey]; exists && f.Status == domainBrief.FieldStatusUnknown {
			f.Status = domainBrief.FieldStatusAsked
			f.Rationale = fmt.Sprintf("已向使用者提出釐清問題: %s", q.Question)
		}
	}

	// 4. Re-evaluate readiness
	res := domainBrief.AssessReadiness(b, catalog)
	b.Readiness = res.Status
	b.ReadinessRationale = res.Rationale

	// 5. Persist brief to DB & sync to workspace
	if err := s.briefRepo.Save(ctx, b); err != nil {
		return nil, fmt.Errorf("failed to save brief after analysis: %w", err)
	}
	if wsRoot != "" {
		_ = s.SyncToWorkspace(b, wsRoot)
	}

	// 6. Submit clarification questions if present
	if clarifyCreator != nil && len(analysis.ClarificationQuestions) > 0 {
		if err := clarifyCreator.CreateRequests(ctx, analysis.ClarificationQuestions); err != nil {
			return nil, fmt.Errorf("failed to register clarification questions: %w", err)
		}
	}

	return b, nil
}
