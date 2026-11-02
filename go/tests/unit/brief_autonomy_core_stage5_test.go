package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/autonomy/deliberation"
	"github.com/wade/aia/internal/autonomy/observation"
	"github.com/wade/aia/internal/domain"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	"github.com/wade/aia/internal/domain/decision"
	domainProject "github.com/wade/aia/internal/domain/project"
	briefService "github.com/wade/aia/internal/service/brief"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestBriefCompletenessObserver_Observations(t *testing.T) {
	// AIA-21 §10.1: BriefCompletenessObserver emits observations across 5 dimensions:
	// Completeness Gap, Decision Conflict, New Info, Stale Assumption, Stage Readiness.
	// Governance integrity: Observer strictly only emits observations, never modifies brief.

	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	briefRepo := repository.NewGenericBriefRepository(db)
	briefSvc := briefService.NewService(briefRepo, projRepo)

	projID := "proj-obs-taiwan"
	p := &domainProject.Project{
		ID:         domain.ID(projID),
		Name:       "徒步環島自主觀察測試",
		DomainType: "general",
		Status:     domainProject.StatusActive,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	_ = projRepo.Create(ctx, p)

	obs := autonomy.NewBriefCompletenessObserver(briefRepo, projRepo)

	// 1. Initial State: Uninitialized or empty brief -> no crash
	emptyObs, err := obs.Observe(ctx, projID)
	if err != nil {
		t.Fatalf("Observe empty failed: %v", err)
	}
	if len(emptyObs) != 0 {
		t.Fatalf("expected 0 observations for uninitialized brief, got %d", len(emptyObs))
	}

	// 2. Initialize Brief for nonfiction_plan: missing P0 route_scope
	brief, err := briefSvc.InitializeBrief(ctx, projID, "nonfiction_plan", "徒步環台灣本島", "")
	if err != nil {
		t.Fatalf("InitializeBrief failed: %v", err)
	}

	// Observation check: Completeness Gap (P0 missing)
	gapObs, err := obs.Observe(ctx, projID)
	if err != nil {
		t.Fatalf("Observe gap failed: %v", err)
	}
	if len(gapObs) == 0 {
		t.Fatalf("expected completeness gap observation for missing P0 fields")
	}

	foundGap := false
	for _, o := range gapObs {
		if o.Category == domainAutonomy.ObsCategoryConstraintBreach && o.Severity >= 8 {
			if o.Data["defect_type"] == "COMPLETENESS_GAP" && o.Data["requires_human_resolution"] == true {
				foundGap = true
				break
			}
		}
	}
	if !foundGap {
		t.Fatalf("expected COMPLETENESS_GAP observation with requires_human_resolution, got %+v", gapObs)
	}

	// Verify Observer did NOT alter Brief (Governance rule AIA-21 §10.1)
	curBrief, _ := briefSvc.GetBrief(ctx, projID)
	if curBrief.BriefVersion != brief.BriefVersion {
		t.Fatalf("VIOLATION: observer modified brief version! %d != %d", curBrief.BriefVersion, brief.BriefVersion)
	}

	// 3. Conflict State: Add a conflicted field
	_, _ = briefSvc.UpdateField(
		ctx, projID, "route_scope", "順時針海線",
		domainBrief.FieldStatusConflicted, "research_conflict", 0.5,
		"研究發現台 9 線部分路段坍方，與原先海線設定衝突", false, "agent", "", "", "",
	)

	conflictObs, err := obs.Observe(ctx, projID)
	if err != nil {
		t.Fatalf("Observe conflict failed: %v", err)
	}
	foundConflict := false
	for _, o := range conflictObs {
		if o.Category == domainAutonomy.ObsCategoryConflict && o.Severity == 9 {
			if o.Data["defect_type"] == "DECISION_CONFLICT" && o.Data["is_reassess"] == true {
				foundConflict = true
				break
			}
		}
	}
	if !foundConflict {
		t.Fatalf("expected DECISION_CONFLICT observation with is_reassess, got %+v", conflictObs)
	}

	// 4. Stale Assumption: Update field with a past timestamp > 7 days ago
	curBrief, _ = briefSvc.GetBrief(ctx, projID)
	pastTime := time.Now().UTC().Add(-10 * 24 * time.Hour)
	curBrief.Fields["daily_mileage_km"] = &domainBrief.FieldState{
		Key:        "daily_mileage_km",
		Value:      20,
		Status:     domainBrief.FieldStatusDefaulted,
		Confidence: 0.65,
		Rationale:  "預設每日 20 公里",
		UpdatedAt:  pastTime,
	}
	_ = briefRepo.Save(ctx, curBrief)

	staleObs, err := obs.Observe(ctx, projID)
	if err != nil {
		t.Fatalf("Observe stale failed: %v", err)
	}
	foundStale := false
	for _, o := range staleObs {
		if o.Category == domainAutonomy.ObsCategoryDrift && o.Data["defect_type"] == "STALE_ASSUMPTION" {
			foundStale = true
			break
		}
	}
	if !foundStale {
		t.Fatalf("expected STALE_ASSUMPTION observation, got %+v", staleObs)
	}

	// 5. Stage Readiness: When brief achieves READY_FOR_PLANNING
	curBrief, _ = briefSvc.GetBrief(ctx, projID)
	curBrief.Fields["route_scope"].Status = domainBrief.FieldStatusAnswered
	curBrief.Readiness = domainBrief.ReadinessReadyForPlanning
	curBrief.ReadinessRationale = "所有必要規格已具備，可進入大綱規劃"
	_ = briefRepo.Save(ctx, curBrief)

	readyObs, err := obs.Observe(ctx, projID)
	if err != nil {
		t.Fatalf("Observe ready failed: %v", err)
	}
	foundReady := false
	for _, o := range readyObs {
		if o.Category == domainAutonomy.ObsCategoryOpportunity && o.Data["defect_type"] == "STAGE_READINESS" {
			if o.Data["is_proceed"] == true {
				foundReady = true
				break
			}
		}
	}
	if !foundReady {
		t.Fatalf("expected STAGE_READINESS observation with is_proceed, got %+v", readyObs)
	}
}

func TestBriefDeliberator_Integration_Decisions(t *testing.T) {
	// AIA-21 §10.2: Deliberator evaluates observations and chooses among:
	// ASK_HUMAN, RESEARCH, PROPOSE_DEFAULT, DEFER, PROCEED, REASSESS

	delib := deliberation.NewDeliberator()

	// 1. ASK_HUMAN: Evaluated when completeness gap requires human resolution
	candAsk := &domainAutonomy.CandidateInitiative{
		ID:                  "cand-ask-01",
		DiscoveryType:       domainAutonomy.DiscoveryIssue,
		ConfidenceLevel:     domainAutonomy.ConfidenceUnverified,
		PriorityScore:       8,
		DetectedOpportunity: "Completeness Gap: missing P0 route_scope",
		EstimatedCost:       1.0,
		MetadataJSON:        `{"requires_human_resolution": true}`,
	}
	resAsk := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{candAsk},
	})
	if resAsk.RecommendedAction != decision.ActionAskHuman {
		t.Fatalf("expected RecommendedAction ASK_HUMAN, got %s", resAsk.RecommendedAction)
	}

	// 2. REASSESS: Evaluated when conflict requires reassessment
	candReassess := &domainAutonomy.CandidateInitiative{
		ID:                  "cand-reassess-01",
		DiscoveryType:       domainAutonomy.DiscoveryIssue,
		ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
		PriorityScore:       9,
		DetectedOpportunity: "Decision Conflict in route_scope",
		EstimatedCost:       1.0,
		MetadataJSON:        `{"is_reassess": true}`,
	}
	resReassess := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{candReassess},
	})
	if resReassess.RecommendedAction != decision.ActionReassess {
		t.Fatalf("expected RecommendedAction REASSESS, got %s", resReassess.RecommendedAction)
	}

	// 3. PROCEED: Evaluated when stage readiness is achieved
	candProceed := &domainAutonomy.CandidateInitiative{
		ID:                  "cand-proceed-01",
		DiscoveryType:       domainAutonomy.DiscoveryOpportunity,
		ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
		PriorityScore:       8,
		DetectedOpportunity: "Stage Readiness achieved: READY_FOR_PLANNING",
		EstimatedCost:       1.0,
		MetadataJSON:        `{"is_proceed": true}`,
	}
	resProceed := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{candProceed},
	})
	if resProceed.RecommendedAction != decision.ActionProceed {
		t.Fatalf("expected RecommendedAction PROCEED, got %s", resProceed.RecommendedAction)
	}

	// 4. RESEARCH: Evaluated when research is identified
	candResearch := &domainAutonomy.CandidateInitiative{
		ID:                  "cand-research-01",
		DiscoveryType:       domainAutonomy.DiscoveryOpportunity,
		ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
		PriorityScore:       7,
		DetectedOpportunity: "Gather road elevation data for eastern routes",
		EstimatedCost:       1.0,
		MetadataJSON:        `{"is_research": true}`,
	}
	resResearch := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{candResearch},
	})
	if resResearch.RecommendedAction != decision.ActionResearch {
		t.Fatalf("expected RecommendedAction RESEARCH, got %s", resResearch.RecommendedAction)
	}

	// 5. PROPOSE_DEFAULT: Evaluated when default hypothesis can be proposed
	candDefault := &domainAutonomy.CandidateInitiative{
		ID:                  "cand-default-01",
		DiscoveryType:       domainAutonomy.DiscoveryImprovement,
		ConfidenceLevel:     domainAutonomy.ConfidenceConfirmed,
		PriorityScore:       7,
		DetectedOpportunity: "Propose default daily mileage 20 km",
		EstimatedCost:       1.0,
		MetadataJSON:        `{"is_propose_default": true}`,
	}
	resDefault := delib.Deliberate(deliberation.DeliberateInput{
		Candidates: []*domainAutonomy.CandidateInitiative{candDefault},
	})
	if resDefault.RecommendedAction != decision.ActionProposeDefault {
		t.Fatalf("expected RecommendedAction PROPOSE_DEFAULT, got %s", resDefault.RecommendedAction)
	}
}

func TestAutonomyObservationRegistry_IncludesBriefObserver(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	projRepo := repository.NewProjectRepository(db)
	briefRepo := repository.NewGenericBriefRepository(db)
	briefSvc := briefService.NewService(briefRepo, projRepo)

	projID := "proj-auto-engine"
	p := &domainProject.Project{
		ID:         domain.ID(projID),
		Name:       "Autonomy Engine 整合測試",
		DomainType: "general",
		Status:     domainProject.StatusActive,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	_ = projRepo.Create(ctx, p)
	_, _ = briefSvc.InitializeBrief(ctx, projID, "nonfiction_plan", "徒步環台灣本島", "")

	obsReg := observation.NewRegistry()
	briefObs := autonomy.NewBriefCompletenessObserver(briefRepo, projRepo)
	obsReg.Register("brief_completeness", briefObs)

	obsList := obsReg.ObserveAll(ctx, projID)
	if len(obsList) == 0 {
		t.Fatalf("expected observations from BriefCompletenessObserver via Registry")
	}

	found := false
	for _, o := range obsList {
		if o.Source == "BriefCompletenessObserver" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected BriefCompletenessObserver in ObserveAll results, got: %+v", obsList)
	}
}
