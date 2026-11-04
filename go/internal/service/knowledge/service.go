package knowledge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/knowledge"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// EvidenceTrace packages a single Evidence with its resolved Source provenance.
type EvidenceTrace struct {
	Evidence *knowledge.Evidence `json:"evidence"`
	Source   *knowledge.Source   `json:"source"`
}

// DecisionProvenanceTrace represents the full contextual trace of a Decision (Milestone M2).
type DecisionProvenanceTrace struct {
	Decision        *decision.Decision          `json:"decision"`
	ContextSnapshot *continuity.ContextSnapshot `json:"context_snapshot,omitempty"`
	EvidenceChain   []*EvidenceTrace            `json:"evidence_chain"`
}

// PromotionValidationResult defines the outcome of verifying knowledge promotion (AIA-16).
type PromotionValidationResult struct {
	Allowed          bool                       `json:"allowed"`
	ProvenanceStatus knowledge.ProvenanceStatus `json:"provenance_status"`
	Reason           string                     `json:"reason"`
	EvidenceIDs      []string                   `json:"evidence_ids,omitempty"`
}

// Service provides high-level business operations for Knowledge, Provenance, Claims, and Decision Tracing.
type Service struct {
	knowledgeRepo  *repository.KnowledgeRepository
	decisionRepo   *repository.DecisionRepository
	continuityRepo *repository.ContinuityRepository
}

func NewService(
	knowledgeRepo *repository.KnowledgeRepository,
	decisionRepo *repository.DecisionRepository,
	continuityRepo *repository.ContinuityRepository,
) *Service {
	return &Service{
		knowledgeRepo:  knowledgeRepo,
		decisionRepo:   decisionRepo,
		continuityRepo: continuityRepo,
	}
}

// RegisterSource saves or registers a data origin with 5-dimensional scores.
func (s *Service) RegisterSource(ctx context.Context, src *knowledge.Source) error {
	if src.AuthorityScore < 0 || src.AuthorityScore > 1.0 {
		return fmt.Errorf("authority_score must be between 0.0 and 1.0")
	}
	return s.knowledgeRepo.CreateSource(ctx, src)
}

// RecordEvidence saves an observed evidence grounded in a registered source.
func (s *Service) RecordEvidence(ctx context.Context, evi *knowledge.Evidence) error {
	// Verify that the Source exists
	_, err := s.knowledgeRepo.GetSource(ctx, evi.SourceID)
	if err != nil {
		return fmt.Errorf("cannot record evidence: referenced source %s not found: %w", evi.SourceID, err)
	}
	return s.knowledgeRepo.CreateEvidence(ctx, evi)
}

// CreateClaim creates a claim with its supporting or refuting evidence links.
func (s *Service) CreateClaim(ctx context.Context, c *knowledge.Claim, links []*knowledge.ClaimEvidenceLink) error {
	if err := s.knowledgeRepo.CreateClaim(ctx, c); err != nil {
		return err
	}
	for _, l := range links {
		if l.ClaimID == "" {
			l.ClaimID = c.ID
		}
		if err := s.knowledgeRepo.LinkClaimEvidence(ctx, l); err != nil {
			return err
		}
	}
	return nil
}

// ReviseClaim appends an immutable claim revision record.
func (s *Service) ReviseClaim(ctx context.Context, claimID, newStatement, reason, revisedDecisionID string) error {
	return s.knowledgeRepo.ReviseClaim(ctx, claimID, newStatement, reason, revisedDecisionID)
}

// ValidateKnowledgePromotion validates whether candidate information can be promoted to VERIFIED Knowledge (AIA-16 Section 8.3 & Rule 5).
// RULE: Model-generated content (SourceModel) can NEVER be promoted to VERIFIED Knowledge without independent
// external evidence from an OFFICIAL or PRIMARY source with high authority (>= 0.7).
func (s *Service) ValidateKnowledgePromotion(
	ctx context.Context,
	sourceType knowledge.SourceType,
	evidenceIDs []string,
) (*PromotionValidationResult, error) {
	now := time.Now().UTC()

	// Case 1: Pure model generated with no supporting external evidence
	if sourceType == knowledge.SourceModel && len(evidenceIDs) == 0 {
		return &PromotionValidationResult{
			Allowed:          false,
			ProvenanceStatus: knowledge.ProvenanceStatusUnverified,
			Reason:           "AIA-16 Rule 5: Model-generated content cannot be automatically promoted to VERIFIED knowledge without independent external evidence.",
		}, nil
	}

	// Case 2: No evidence provided for any non-primary source
	if len(evidenceIDs) == 0 {
		if sourceType == knowledge.SourceOfficial || sourceType == knowledge.SourcePrimary {
			return &PromotionValidationResult{
				Allowed:          true,
				ProvenanceStatus: knowledge.ProvenanceStatusVerified,
				Reason:           "Direct official/primary source accepted as verified.",
			}, nil
		}
		return &PromotionValidationResult{
			Allowed:          false,
			ProvenanceStatus: knowledge.ProvenanceStatusPartial,
			Reason:           "Insufficient external evidence to establish verified status.",
		}, nil
	}

	// Case 3: Validate each supporting evidence
	hasValidOfficialOrPrimary := false
	validEvidenceIDs := make([]string, 0, len(evidenceIDs))

	for _, id := range evidenceIDs {
		evi, err := s.knowledgeRepo.GetEvidence(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to load evidence %s: %w", id, err)
		}

		// Check temporal validity
		if evi.ValidUntil != nil && now.After(*evi.ValidUntil) {
			continue // Expired evidence cannot support promotion
		}

		// Check source reliability
		src, err := s.knowledgeRepo.GetSource(ctx, evi.SourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to load source %s for evidence %s: %w", evi.SourceID, id, err)
		}

		if (src.SourceType == knowledge.SourceOfficial || src.SourceType == knowledge.SourcePrimary) && src.AuthorityScore >= 0.7 {
			hasValidOfficialOrPrimary = true
		}
		validEvidenceIDs = append(validEvidenceIDs, id)
	}

	if !hasValidOfficialOrPrimary {
		return &PromotionValidationResult{
			Allowed:          false,
			ProvenanceStatus: knowledge.ProvenanceStatusPartial,
			Reason:           "Supporting evidence lacks an active, high-authority OFFICIAL or PRIMARY source.",
			EvidenceIDs:      validEvidenceIDs,
		}, nil
	}

	return &PromotionValidationResult{
		Allowed:          true,
		ProvenanceStatus: knowledge.ProvenanceStatusVerified,
		Reason:           "Evidence grounded in verified high-authority external sources meets AIA-16 promotion criteria.",
		EvidenceIDs:      validEvidenceIDs,
	}, nil
}

// TraceDecisionProvenance reconstructs the complete provenance chain for a given decision (Milestone M2).
// Returns the decision, referenced snapshot (if any), and list of supporting/refuting evidences with their sources.
func (s *Service) TraceDecisionProvenance(ctx context.Context, decisionID string) (*DecisionProvenanceTrace, error) {
	d, err := s.decisionRepo.Get(ctx, decisionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get decision %s: %w", decisionID, err)
	}

	trace := &DecisionProvenanceTrace{
		Decision:      d,
		EvidenceChain: make([]*EvidenceTrace, 0),
	}

	// 1. Resolve context snapshot if present
	if d.ContextSnapshotID != nil && *d.ContextSnapshotID != "" {
		snap, err := s.continuityRepo.GetSnapshot(ctx, *d.ContextSnapshotID)
		if err == nil {
			trace.ContextSnapshot = snap
		}
	}

	// 2. Resolve evidence links
	links, err := s.decisionRepo.GetEvidenceLinks(ctx, decisionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence links for decision %s: %w", decisionID, err)
	}

	for _, link := range links {
		evi, err := s.knowledgeRepo.GetEvidence(ctx, link.EvidenceID)
		if err != nil {
			continue
		}
		src, err := s.knowledgeRepo.GetSource(ctx, evi.SourceID)
		if err != nil {
			continue
		}
		trace.EvidenceChain = append(trace.EvidenceChain, &EvidenceTrace{
			Evidence: evi,
			Source:   src,
		})
	}

	return trace, nil
}

// PropagationReport summarizes the impact of invalidating or deprecating an evidence item (AIA-18 §8.4).
type PropagationReport struct {
	EvidenceID        string                   `json:"evidence_id"`
	Reason            string                   `json:"reason"`
	PreviousStatus    knowledge.EvidenceStatus `json:"previous_status"`
	NewStatus         knowledge.EvidenceStatus `json:"new_status"`
	AffectedClaimIDs  []string                 `json:"affected_claim_ids"`
	RevisedClaims     []*knowledge.Claim       `json:"revised_claims"`
	AffectedDecisions []string                 `json:"affected_decisions"`
	NotifiedAt        time.Time                `json:"notified_at"`
}

// PropagateEvidenceInvalidation handles the invalidation, deprecation, or refutation of evidence.
// It marks the evidence as DEPRECATED (or CONFLICTED), traces all dependent claims and decisions,
// demotes claims relying on this evidence, and logs historical revisions (AIA-16 §6.5 & AIA-18 §8.4).
func (s *Service) PropagateEvidenceInvalidation(ctx context.Context, evidenceID string, reason string) (*PropagationReport, error) {
	evi, err := s.knowledgeRepo.GetEvidence(ctx, evidenceID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve evidence %s: %w", evidenceID, err)
	}

	prevStatus := evi.Status
	newStatus := knowledge.EvidenceDeprecated
	if strings.Contains(strings.ToUpper(reason), "CONFLICT") {
		newStatus = knowledge.EvidenceConflicted
	}

	if err := s.knowledgeRepo.UpdateEvidenceStatus(ctx, evidenceID, newStatus); err != nil {
		return nil, fmt.Errorf("failed to update evidence status: %w", err)
	}

	report := &PropagationReport{
		EvidenceID:        evidenceID,
		Reason:            reason,
		PreviousStatus:    prevStatus,
		NewStatus:         newStatus,
		AffectedClaimIDs:  make([]string, 0),
		RevisedClaims:     make([]*knowledge.Claim, 0),
		AffectedDecisions: make([]string, 0),
		NotifiedAt:        time.Now().UTC(),
	}

	// 1. Trace and update affected claims
	links, err := s.knowledgeRepo.GetClaimLinksForEvidence(ctx, evidenceID)
	if err == nil {
		for _, l := range links {
			report.AffectedClaimIDs = append(report.AffectedClaimIDs, l.ClaimID)
			claim, err := s.knowledgeRepo.GetClaim(ctx, l.ClaimID)
			if err != nil {
				continue
			}

			// If the invalidated evidence was supporting the claim, degrade its status
			if l.SupportType == knowledge.SupportTypeSupports || l.SupportType == knowledge.SupportTypeDerivedFrom {
				newClaimStatus := knowledge.AssertionUncertain
				if newStatus == knowledge.EvidenceConflicted {
					newClaimStatus = knowledge.AssertionConflicted
				}
				// Demote knowledge status
				_ = s.knowledgeRepo.UpdateClaimStatus(ctx, claim.ID, newClaimStatus, false)
				revReason := fmt.Sprintf("Underlying evidence %s invalidated: %s", evidenceID, reason)
				_ = s.knowledgeRepo.ReviseClaim(ctx, claim.ID, claim.Statement, revReason, "")
				claim.AssertionStatus = newClaimStatus
				claim.IsKnowledge = false
				report.RevisedClaims = append(report.RevisedClaims, claim)
			}
		}
	}

	// 2. Trace affected decisions relying on this evidence
	if s.decisionRepo != nil {
		decisions, err := s.decisionRepo.GetDecisionsForEvidence(ctx, evidenceID)
		if err == nil {
			report.AffectedDecisions = decisions
		}
	}

	return report, nil
}
