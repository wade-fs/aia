package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/knowledge"
	"github.com/wade/aia/internal/domain/project"
	knowledgeService "github.com/wade/aia/internal/service/knowledge"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestKnowledgeService_PromotionRules(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	kRepo := repository.NewKnowledgeRepository(db)
	dRepo := repository.NewDecisionRepository(db)
	cRepo := repository.NewContinuityRepository(db)
	svc := knowledgeService.NewService(kRepo, dRepo, cRepo)

	// Rule 5: Pure model-generated content CANNOT be automatically promoted to VERIFIED
	res, err := svc.ValidateKnowledgePromotion(ctx, knowledge.SourceModel, nil)
	if err != nil {
		t.Fatalf("unexpected error validating model promotion: %v", err)
	}
	if res.Allowed {
		t.Errorf("expected promotion to be disallowed for ungrounded Model source, but got allowed")
	}
	if res.ProvenanceStatus != knowledge.ProvenanceStatusUnverified {
		t.Errorf("expected UNVERIFIED status, got: %v", res.ProvenanceStatus)
	}

	// Setup: Create an Official source with high authority
	now := time.Now().UTC()
	err = svc.RegisterSource(ctx, &knowledge.Source{
		ID:                   "src-twse-official",
		SourceType:           knowledge.SourceOfficial,
		URI:                  "https://www.twse.com.tw/announcements",
		Publisher:            "Taiwan Stock Exchange",
		AuthorityScore:       0.95,
		DirectnessScore:      0.90,
		TimelinessScore:      0.88,
		IndependenceScore:    1.0,
		ReproducibilityScore: 1.0,
		CreatedAt:            now,
	})
	if err != nil {
		t.Fatalf("failed to register source: %v", err)
	}

	// Create valid evidence linked to that source
	validUntil := now.Add(24 * time.Hour)
	err = svc.RecordEvidence(ctx, &knowledge.Evidence{
		ID:             "evi-market-data",
		SourceID:       "src-twse-official",
		ContentSnippet: "Daily turnover exceeded 300B TWD",
		ContentHash:    "sha256-hash-turnover",
		CaptureMethod:  knowledge.CaptureAPIJSON,
		CapturedAt:     now,
		ValidUntil:     &validUntil,
		CreatedAt:      now,
	})
	if err != nil {
		t.Fatalf("failed to record evidence: %v", err)
	}

	// Validate promotion with high-authority evidence -> Allowed and VERIFIED
	res, err = svc.ValidateKnowledgePromotion(ctx, knowledge.SourceModel, []string{"evi-market-data"})
	if err != nil {
		t.Fatalf("unexpected error validating promotion with evidence: %v", err)
	}
	if !res.Allowed {
		t.Errorf("expected promotion to be allowed with official evidence, got disallowed: %s", res.Reason)
	}
	if res.ProvenanceStatus != knowledge.ProvenanceStatusVerified {
		t.Errorf("expected VERIFIED status, got: %v", res.ProvenanceStatus)
	}
}

func TestKnowledgeService_TraceDecisionProvenance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	kRepo := repository.NewKnowledgeRepository(db)
	dRepo := repository.NewDecisionRepository(db)
	cRepo := repository.NewContinuityRepository(db)
	svc := knowledgeService.NewService(kRepo, dRepo, cRepo)

	now := time.Now().UTC()

	// 1. Create Source & Evidence
	_ = svc.RegisterSource(ctx, &knowledge.Source{
		ID:                   "src-sec-filing",
		SourceType:           knowledge.SourcePrimary,
		URI:                  "https://sec.gov/filings/10k",
		Publisher:            "SEC Edgar",
		AuthorityScore:       0.99,
		DirectnessScore:      1.0,
		TimelinessScore:      0.80,
		IndependenceScore:    1.0,
		ReproducibilityScore: 1.0,
		CreatedAt:            now,
	})

	_ = svc.RecordEvidence(ctx, &knowledge.Evidence{
		ID:             "evi-sec-revenue",
		SourceID:       "src-sec-filing",
		ContentSnippet: "FY2025 Revenue was 12.5B USD",
		ContentHash:    "sec-hash-123",
		CaptureMethod:  knowledge.CaptureHTTPGet,
		CapturedAt:     now,
		CreatedAt:      now,
	})

	// 2. Create Project & Context Snapshot
	pRepo := repository.NewProjectRepository(db)
	_ = pRepo.Create(ctx, &project.Project{
		ID:        "proj-pilot",
		Name:      "Pilot",
		Workspace: "/tmp/pilot",
		Status:    project.StatusActive,
		CreatedAt: now,
	})

	snapID := "snap-pre-decision-001"
	err := cRepo.CreateSnapshot(ctx, &continuity.ContextSnapshot{
		ID:                  snapID,
		Scope:               continuity.SnapshotScopeProject,
		ProjectID:           "proj-pilot",
		SnapshotType:        continuity.SnapshotTypePostCycle,
		WorkspaceStateJSON:  `{"branch":"main"}`,
		ActiveTasksJSON:     `[]`,
		RecentDecisionsJSON: `[]`,
		PendingIssuesJSON:   `["Review risk thresholds"]`,
		CreatedAt:           now,
	})
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	// 3. Create Decision with Evidence link and NO_ACTION action type
	decID := "dec-no-action-001"
	dec := &decision.Decision{
		ID:                decID,
		ProjectID:         &[]string{"proj-pilot"}[0],
		ActionType:        decision.ActionNoAction,
		IntentSummary:     "Decided to hold off execution due to market conditions",
		Justification:     "Revenue target met; volatility remains high according to SEC filing",
		ContextSnapshotID: &snapID,
		GovernanceResult:  decision.GovernanceAllowed,
		CreatedAt:         now,
	}
	links := []*decision.DecisionEvidenceLink{
		{
			DecisionID:    decID,
			EvidenceID:    "evi-sec-revenue",
			RelevanceNote: "Supports rationale for safe holding",
		},
	}
	if err := dRepo.Create(ctx, dec, links); err != nil {
		t.Fatalf("failed to create decision: %v", err)
	}

	// 4. Trace decision provenance (Milestone M2 verification)
	trace, err := svc.TraceDecisionProvenance(ctx, decID)
	if err != nil {
		t.Fatalf("failed to trace decision provenance: %v", err)
	}

	if trace.Decision == nil || trace.Decision.ID != decID {
		t.Fatalf("expected decision ID %s, got: %+v", decID, trace.Decision)
	}
	if trace.Decision.ActionType != decision.ActionNoAction {
		t.Errorf("expected ActionType NO_ACTION, got: %s", trace.Decision.ActionType)
	}
	if trace.ContextSnapshot == nil || trace.ContextSnapshot.ID != snapID {
		t.Errorf("expected snapshot ID %s, got: %+v", snapID, trace.ContextSnapshot)
	}
	if len(trace.EvidenceChain) != 1 {
		t.Fatalf("expected 1 evidence trace, got %d", len(trace.EvidenceChain))
	}
	evTrace := trace.EvidenceChain[0]
	if evTrace.Evidence.ID != "evi-sec-revenue" {
		t.Errorf("expected evidence ID evi-sec-revenue, got: %s", evTrace.Evidence.ID)
	}
	if evTrace.Source == nil || evTrace.Source.AuthorityScore != 0.99 {
		t.Errorf("expected source with authority 0.99, got: %+v", evTrace.Source)
	}
}
