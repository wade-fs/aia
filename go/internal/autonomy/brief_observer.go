package autonomy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wade/aia/internal/autonomy/clock"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// BriefCompletenessObserver observes requirement completeness, decision conflicts,
// stale assumptions, and stage readiness across project briefs (AIA-21 §10.1).
// STRICT GOVERNANCE RULE (AIA-21 §10.1): The observer ONLY emits Observations.
// It never directly modifies Briefs, never creates ClarificationRequests,
// never prompts the user, and never approves decisions.
type BriefCompletenessObserver struct {
	briefRepo *repository.GenericBriefRepository
	projRepo  *repository.ProjectRepository
}

// NewBriefCompletenessObserver creates a new BriefCompletenessObserver.
func NewBriefCompletenessObserver(briefRepo *repository.GenericBriefRepository, projRepo *repository.ProjectRepository) *BriefCompletenessObserver {
	return &BriefCompletenessObserver{
		briefRepo: briefRepo,
		projRepo:  projRepo,
	}
}

// Observe inspects the Project Brief and returns structural observations (AIA-21 §10.1).
func (o *BriefCompletenessObserver) Observe(ctx context.Context, projectID string) ([]domainAutonomy.Observation, error) {
	if o.briefRepo == nil {
		return nil, nil
	}

	var observations []domainAutonomy.Observation
	now := clock.Now(ctx)

	// If projectID is not provided, list all projects
	var projectIDs []string
	if projectID != "" {
		projectIDs = []string{projectID}
	} else if o.projRepo != nil {
		projects, err := o.projRepo.List(ctx)
		if err == nil {
			for _, p := range projects {
				projectIDs = append(projectIDs, string(p.ID))
			}
		}
	}

	for _, pid := range projectIDs {
		brief, err := o.briefRepo.GetByProjectID(ctx, pid)
		if err != nil || brief == nil {
			continue
		}

		catalog, err := domainBrief.GetCatalog(brief.Domain)
		if err != nil || catalog == nil {
			continue
		}

		res := domainBrief.AssessReadiness(brief, catalog)

		// 1. Decision Conflict: Conflicted fields detected (AIA-21 §10.1 item 2)
		if len(res.ConflictedFields) > 0 {
			observations = append(observations, domainAutonomy.Observation{
				Source:        "BriefCompletenessObserver",
				Domain:        brief.Domain,
				Category:      domainAutonomy.ObsCategoryConflict,
				Summary:       fmt.Sprintf("Decision Conflict in project %s: fields [%s] are contradicted by newer findings", pid, strings.Join(res.ConflictedFields, ", ")),
				NonActionable: false,
				Confidence:    domainAutonomy.ConfidenceConfirmed,
				Severity:      9,
				Timestamp:     now,
				Data: map[string]any{
					"defect_type":               "DECISION_CONFLICT",
					"project_id":                pid,
					"conflicted_fields":         res.ConflictedFields,
					"requires_human_resolution": true,
					"is_reassess":               true,
				},
			})
		}

		// 2. Completeness Gap: Missing P0 planning-critical fields (AIA-21 §10.1 item 1)
		if len(res.MissingP0Fields) > 0 {
			observations = append(observations, domainAutonomy.Observation{
				Source:        "BriefCompletenessObserver",
				Domain:        brief.Domain,
				Category:      domainAutonomy.ObsCategoryConstraintBreach,
				Summary:       fmt.Sprintf("Completeness Gap in project %s: %d critical P0 planning fields missing", pid, len(res.MissingP0Fields)),
				NonActionable: false,
				Confidence:    domainAutonomy.ConfidenceUnverified,
				Severity:      8,
				Timestamp:     now,
				Data: map[string]any{
					"defect_type":               "COMPLETENESS_GAP",
					"project_id":                pid,
					"missing_p0_fields":         res.MissingP0Fields,
					"requires_human_resolution": true,
				},
			})
		}

		// 3. Stale Assumptions: Assumptions unverified for > 7 days (AIA-21 §10.1 item 4)
		staleCutoff := now.Add(-7 * 24 * time.Hour)
		var staleFields []string
		for k, f := range brief.Fields {
			if f.Status == domainBrief.FieldStatusDefaulted && f.UpdatedAt.Before(staleCutoff) {
				staleFields = append(staleFields, k)
			}
		}
		if len(staleFields) > 0 {
			observations = append(observations, domainAutonomy.Observation{
				Source:        "BriefCompletenessObserver",
				Domain:        brief.Domain,
				Category:      domainAutonomy.ObsCategoryDrift,
				Summary:       fmt.Sprintf("Stale Assumptions in project %s: [%s] defaulted over 7 days ago without human confirmation", pid, strings.Join(staleFields, ", ")),
				NonActionable: false,
				Confidence:    domainAutonomy.ConfidencePotential,
				Severity:      4,
				Timestamp:     now,
				Data: map[string]any{
					"defect_type":  "STALE_ASSUMPTION",
					"project_id":   pid,
					"stale_fields": staleFields,
				},
			})
		}

		// 4. Stage Readiness: Project reached planning or execution readiness (AIA-21 §10.1 item 5)
		if brief.Readiness == domainBrief.ReadinessReadyForPlanning || brief.Readiness == domainBrief.ReadinessReadyForExecution {
			observations = append(observations, domainAutonomy.Observation{
				Source:        "BriefCompletenessObserver",
				Domain:        brief.Domain,
				Category:      domainAutonomy.ObsCategoryOpportunity,
				Summary:       fmt.Sprintf("Stage Readiness achieved in project %s: %s (%s)", pid, brief.Readiness, brief.ReadinessRationale),
				NonActionable: false,
				Confidence:    domainAutonomy.ConfidenceConfirmed,
				Severity:      3,
				Timestamp:     now,
				Data: map[string]any{
					"defect_type": "STAGE_READINESS",
					"project_id":  pid,
					"readiness":   string(brief.Readiness),
					"is_proceed":  true,
				},
			})
		} else if len(res.MissingP0Fields) == 0 && len(res.ConflictedFields) == 0 {
			// Baseline calm snapshot if no critical gaps or conflicts
			observations = append(observations, domainAutonomy.Observation{
				Source:        "BriefCompletenessObserver",
				Domain:        brief.Domain,
				Category:      domainAutonomy.ObsCategoryStateSnapshot,
				Summary:       fmt.Sprintf("Brief for project %s is stable (%s)", pid, brief.Readiness),
				NonActionable: true,
				Confidence:    domainAutonomy.ConfidenceConfirmed,
				Severity:      1,
				Timestamp:     now,
				Data: map[string]any{
					"project_id": pid,
					"readiness":  string(brief.Readiness),
				},
			})
		}
	}

	return observations, nil
}
