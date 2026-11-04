package recall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/autonomy/clock"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
	"github.com/wade/aia/internal/domain/decision"
	"github.com/wade/aia/internal/domain/goal"
	"github.com/wade/aia/internal/domain/knowledge"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/domain/memory"
	domainPlan "github.com/wade/aia/internal/domain/plan"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var (
	ErrHypothesisCannotBeVerified = errors.New("memory governance violation: HYPOTHESIS memory cannot be marked VERIFIED (AIA-ARCH-DATA-01 §3.6B)")
	ErrFactMissingProvenanceRef   = errors.New("memory governance violation: FACT_BASED memory cannot be VERIFIED without EVIDENCE or SOURCE reference")
)

// MemoryConflict describes a detected contradiction preserved for reconciliation rather than silent overwrite (docs/memory.md §4).
type MemoryConflict struct {
	Topic       string   `json:"topic"`
	ClaimIDs    []string `json:"claim_ids,omitempty"`
	MemoryIDs   []string `json:"memory_ids,omitempty"`
	Description string   `json:"description"`
}

// RecalledBundle represents the L1–L5 Associative Recall result for an Autonomy Cycle (Plan §0.5B, Phase 3 §4.1).
type RecalledBundle struct {
	// L1: Immutable Event History
	RecentEvents []*continuity.EventHistory `json:"recent_events,omitempty"`
	// L2: Grounded Evidence & Source Provenance
	GroundedEvidence []*knowledge.Evidence        `json:"grounded_evidence,omitempty"`
	Sources          map[string]*knowledge.Source `json:"sources,omitempty"`
	// L3: Associatively Ranked Memories & Claims
	Memories         []memory.Memory   `json:"memories,omitempty"`
	AcceptedClaims   []*knowledge.Claim `json:"accepted_claims,omitempty"`
	ConflictedClaims []*knowledge.Claim `json:"conflicted_claims,omitempty"`
	Conflicts        []MemoryConflict  `json:"conflicts,omitempty"`
	// L4: Project & Continuity Context State
	LatestSnapshot  *continuity.ContextSnapshot `json:"latest_snapshot,omitempty"`
	ActiveGoals     []*goal.Goal                `json:"active_goals,omitempty"`
	RecentDecisions []*decision.Decision        `json:"recent_decisions,omitempty"`
	PendingIssues   []string                    `json:"pending_issues,omitempty"`
	ActiveTasks       []task.Task                 `json:"active_tasks,omitempty"`
	RecentFailedTasks []task.Task                 `json:"recent_failed_tasks,omitempty"`
	ActiveProblems    []*domainProblem.Problem    `json:"active_problems,omitempty"`
	ActivePlans     []*domainPlan.Plan          `json:"active_plans,omitempty"`
	// Phase 4 L0–L3 Epistemic Learning Partitions (AIA-11 §21.2)
	VerifiedLearnings   []*domainLearning.LearningRecord `json:"verified_learnings,omitempty"`   // L2/L3 EFFECTIVE
	NegativeConstraints []*domainLearning.LearningRecord `json:"negative_constraints,omitempty"` // REGRESSED / INEFFECTIVE anti-patterns
	PendingHypotheses   []*domainLearning.LearningRecord `json:"pending_hypotheses,omitempty"`   // L0/L1/INCONCLUSIVE (zero weight boost)
	// L5: Bounded Active Context Digest
	ActiveDigest string `json:"active_digest"`
}

type RecallInput struct {
	ProjectID    string
	GoalID       *string
	Observations []domainAutonomy.Observation
	MaxMemories  int
}

// Engine implements L1–L5 Associative Retrieval and Memory Integrity Governance (docs/memory.md).
type Engine struct {
	eventRepo      *repository.EventHistoryRepository
	knowledgeRepo  *repository.KnowledgeRepository
	memoryRepo     *repository.MemoryRepository
	continuityRepo *repository.ContinuityRepository
	goalRepo       *repository.GoalRepository
	decisionRepo   *repository.DecisionRepository
	projectRepo    *repository.ProjectRepository
	taskRepo       *repository.TaskRepository
	learningRepo   *repository.LearningRepository
	problemRepo    *repository.ProblemRepository
	planRepo       *repository.PlanRepository
}

func NewEngine(
	eventRepo *repository.EventHistoryRepository,
	knowledgeRepo *repository.KnowledgeRepository,
	memoryRepo *repository.MemoryRepository,
	continuityRepo *repository.ContinuityRepository,
	goalRepo *repository.GoalRepository,
	decisionRepo *repository.DecisionRepository,
	projectRepo *repository.ProjectRepository,
	taskRepo *repository.TaskRepository,
) *Engine {
	return &Engine{
		eventRepo:      eventRepo,
		knowledgeRepo:  knowledgeRepo,
		memoryRepo:     memoryRepo,
		continuityRepo: continuityRepo,
		goalRepo:       goalRepo,
		decisionRepo:   decisionRepo,
		projectRepo:    projectRepo,
		taskRepo:       taskRepo,
	}
}

func (e *Engine) SetLearningRepository(repo *repository.LearningRepository) {
	e.learningRepo = repo
}

func (e *Engine) SetProblemRepository(repo *repository.ProblemRepository) {
	e.problemRepo = repo
}

func (e *Engine) SetPlanRepository(repo *repository.PlanRepository) {
	e.planRepo = repo
}

// Recall reconstructs L1–L5 context associatively using project state, goals, observations, and provenance weights.
func (e *Engine) Recall(ctx context.Context, in RecallInput) (*RecalledBundle, error) {
	now := clock.Now(ctx)
	asOf, hasAsOf := clock.AsOf(ctx)
	bundle := &RecalledBundle{
		Sources: make(map[string]*knowledge.Source),
	}

	maxMem := in.MaxMemories
	if maxMem <= 0 {
		maxMem = 12
	}

	// --- L4: Project State, Snapshot, Goals, Pending Issues, Active Tasks, Recent Decisions ---
	if e.continuityRepo != nil && in.ProjectID != "" {
		if snap, err := e.continuityRepo.GetLatestSnapshotByProject(ctx, in.ProjectID); err == nil && snap != nil {
			bundle.LatestSnapshot = snap
			var issues []string
			if json.Unmarshal([]byte(snap.PendingIssuesJSON), &issues) == nil {
				bundle.PendingIssues = append(bundle.PendingIssues, issues...)
			}
		}
	}

	if e.projectRepo != nil && in.ProjectID != "" {
		if proj, err := e.projectRepo.Get(ctx, in.ProjectID); err == nil && proj != nil && len(bundle.PendingIssues) == 0 {
			var issues []string
			if json.Unmarshal([]byte(proj.PendingIssuesJSON), &issues) == nil {
				bundle.PendingIssues = append(bundle.PendingIssues, issues...)
			}
		}
	}

	if e.goalRepo != nil && in.ProjectID != "" {
		if goals, err := e.goalRepo.ListByProject(ctx, in.ProjectID); err == nil {
			for _, g := range goals {
				if g.Status == goal.StatusActive || g.Status == goal.StatusDraft {
					bundle.ActiveGoals = append(bundle.ActiveGoals, g)
				}
			}
		}
		if in.GoalID != nil && *in.GoalID != "" {
			alreadyLoaded := false
			for _, g := range bundle.ActiveGoals {
				if g.ID == *in.GoalID {
					alreadyLoaded = true
					break
				}
			}
			if !alreadyLoaded {
				if g, err := e.goalRepo.Get(ctx, *in.GoalID); err == nil && g != nil {
					bundle.ActiveGoals = append(bundle.ActiveGoals, g)
				}
			}
		}
	}

	if e.decisionRepo != nil && in.ProjectID != "" {
		if decs, err := e.decisionRepo.ListByProject(ctx, in.ProjectID, 10); err == nil {
			for _, d := range decs {
				if hasAsOf && d.CreatedAt.After(asOf) {
					continue // Zero look-ahead filter
				}
				bundle.RecentDecisions = append(bundle.RecentDecisions, d)
			}
		}
	}

	if e.taskRepo != nil && in.ProjectID != "" {
		if tasks, err := e.taskRepo.List(ctx, repository.TaskFilter{ProjectID: in.ProjectID, Limit: 50}); err == nil {
			for _, tk := range tasks {
				if hasAsOf && tk.CreatedAt.After(asOf) {
					continue
				}
				if !tk.IsTerminal() {
					bundle.ActiveTasks = append(bundle.ActiveTasks, tk)
				} else if tk.Status == task.StatusFailed {
					bundle.RecentFailedTasks = append(bundle.RecentFailedTasks, tk)
				}
			}
		}
	}

	// --- L4: Problems & Versioned Plans (AIA-22 §10.1~10.3) ---
	if e.problemRepo != nil && in.ProjectID != "" {
		if probs, err := e.problemRepo.ListByProject(ctx, in.ProjectID); err == nil {
			for _, p := range probs {
				if hasAsOf && p.CreatedAt.After(asOf) {
					continue
				}
				if p.Status != domainProblem.StatusResolved && p.Status != domainProblem.StatusClosed {
					bundle.ActiveProblems = append(bundle.ActiveProblems, p)
					if e.planRepo != nil && p.CurrentPlanVersion > 0 {
						if pl, err := e.planRepo.GetByProblemAndVersion(ctx, p.ID, p.CurrentPlanVersion); err == nil && pl != nil {
							bundle.ActivePlans = append(bundle.ActivePlans, pl)
						}
					}
				}
			}
		}
	}

	// --- L1: Event History for Project ---
	if e.eventRepo != nil && in.ProjectID != "" {
		if evts, err := e.eventRepo.ListByAggregate(ctx, "PROJECT", in.ProjectID); err == nil {
			var filtered []*continuity.EventHistory
			for _, ev := range evts {
				if hasAsOf && ev.OccurredAt.After(asOf) {
					continue
				}
				filtered = append(filtered, ev)
			}
			if len(filtered) > 10 {
				filtered = filtered[len(filtered)-10:]
			}
			bundle.RecentEvents = filtered
		}
	}

	// --- L2: Grounded Evidence & Source Provenance linked to recent Decisions ---
	if e.decisionRepo != nil && e.knowledgeRepo != nil {
		seenEvidence := map[string]bool{}
		for _, dec := range bundle.RecentDecisions {
			links, err := e.decisionRepo.GetEvidenceLinks(ctx, dec.ID)
			if err != nil {
				continue
			}
			for _, lk := range links {
				if seenEvidence[lk.EvidenceID] {
					continue
				}
				seenEvidence[lk.EvidenceID] = true
				evi, err := e.knowledgeRepo.GetEvidence(ctx, lk.EvidenceID)
				if err != nil || evi == nil {
					continue
				}
				// Temporal Management (docs/memory.md §4 Rule 4): exclude expired evidence
				if evi.ValidUntil != nil && now.After(*evi.ValidUntil) {
					continue
				}
				bundle.GroundedEvidence = append(bundle.GroundedEvidence, evi)
				if _, ok := bundle.Sources[evi.SourceID]; !ok {
					if src, sErr := e.knowledgeRepo.GetSource(ctx, evi.SourceID); sErr == nil && src != nil {
						bundle.Sources[evi.SourceID] = src
					}
				}
			}
		}
	}

	// --- L3: Claims, Conflict Detection & Associative Memory Retrieval ---
	if e.knowledgeRepo != nil {
		if accepted, err := e.knowledgeRepo.ListClaims(ctx, knowledge.AssertionAccepted, 20); err == nil {
			bundle.AcceptedClaims = accepted
		}
		if conflicted, err := e.knowledgeRepo.ListClaims(ctx, knowledge.AssertionConflicted, 20); err == nil && len(conflicted) > 0 {
			bundle.ConflictedClaims = conflicted
			for _, c := range conflicted {
				bundle.Conflicts = append(bundle.Conflicts, MemoryConflict{
					Topic:       c.Statement,
					ClaimIDs:    []string{c.ID},
					Description: fmt.Sprintf("Conflicted claim requires reconciliation: %s", c.Statement),
				})
			}
		}
	}

	if e.memoryRepo != nil {
		// Retrieve candidate memories across project + global scope without single-keyword tunnel vision
		candidates, err := e.memoryRepo.Search(ctx, repository.MemoryQuery{
			ProjectID: in.ProjectID,
			Limit:     100,
		})
		if err == nil && len(candidates) > 0 {
			assocCues := extractAssociationCues(bundle.ActiveGoals, in.Observations, bundle.PendingIssues)
			type rankedMem struct {
				mem   memory.Memory
				score float64
			}
			var ranked []rankedMem

			// Track titles to detect conflicting active memories on identical subject
			byTitle := map[string][]memory.Memory{}

			for _, m := range candidates {
				if hasAsOf && m.CreatedAt.After(asOf) {
					continue // Zero look-ahead filter
				}
				// Temporal validity filter: skip expired memories
				if m.ExpiresAt != nil && now.After(*m.ExpiresAt) {
					continue
				}
				normTitle := strings.ToLower(strings.TrimSpace(m.Title))
				if normTitle != "" {
					byTitle[normTitle] = append(byTitle[normTitle], m)
				}

				score := computeAssociativeScore(m, assocCues, in.ProjectID, now)
				ranked = append(ranked, rankedMem{mem: m, score: score})
			}

			for title, group := range byTitle {
				if len(group) > 1 && hasContradictoryContent(group) {
					ids := make([]string, 0, len(group))
					for _, gm := range group {
						ids = append(ids, gm.ID)
					}
					bundle.Conflicts = append(bundle.Conflicts, MemoryConflict{
						Topic:       title,
						MemoryIDs:   ids,
						Description: fmt.Sprintf("Multiple active memories for %q contain conflicting assertions; preserved for review", title),
					})
				}
			}

			sort.Slice(ranked, func(i, j int) bool {
				return ranked[i].score > ranked[j].score
			})

			if len(ranked) > maxMem {
				ranked = ranked[:maxMem]
			}
			for _, rm := range ranked {
				bundle.Memories = append(bundle.Memories, rm.mem)
				_ = e.memoryRepo.TouchLastUsed(ctx, rm.mem.ID)
			}
		}
	}

	// --- Phase 4 Learning Records Partition (L2/L3 Positive vs REGRESSED/INEFFECTIVE Anti-Patterns vs L0/L1 Pending) ---
	if e.learningRepo != nil {
		if records, err := e.learningRepo.ListLearningRecords(ctx, in.ProjectID, "", 30); err == nil {
			for _, rec := range records {
				switch {
				case rec.EffectivenessVerdict == domainLearning.VerdictEffective &&
					(rec.LearningLevel == domainLearning.LevelL2Verified || rec.LearningLevel == domainLearning.LevelL3Generalized):
					bundle.VerifiedLearnings = append(bundle.VerifiedLearnings, rec)
				case rec.EffectivenessVerdict == domainLearning.VerdictRegressed ||
					rec.EffectivenessVerdict == domainLearning.VerdictIneffective:
					bundle.NegativeConstraints = append(bundle.NegativeConstraints, rec)
				default:
					// L0_RECORDED, L1_APPLIED, INCONCLUSIVE: never elevate decision weight
					bundle.PendingHypotheses = append(bundle.PendingHypotheses, rec)
				}
			}
		}
	}

	// --- L5: Assemble Bounded Active Context Digest ---
	bundle.ActiveDigest = buildActiveDigest(bundle)
	return bundle, nil
}

// ValidateMemoryWrite enforces the Provenance Integrity Matrix (AIA-ARCH-DATA-01 §3.6B, docs/memory.md §4).
func ValidateMemoryWrite(m *memory.Memory) error {
	if m == nil {
		return fmt.Errorf("nil memory")
	}
	if m.ProvenanceType == "" {
		m.ProvenanceType = "INFERENCE"
	}
	if m.ProvenanceRefType == "" {
		m.ProvenanceRefType = "NONE"
	}
	if m.ProvenanceStatus == "" {
		m.ProvenanceStatus = "UNVERIFIED"
	}

	switch m.ProvenanceType {
	case "HYPOTHESIS":
		if m.ProvenanceStatus == "VERIFIED" {
			return ErrHypothesisCannotBeVerified
		}
		m.ProvenanceStatus = "UNVERIFIED"

	case "FACT_BASED":
		hasValidAnchor := (m.ProvenanceRefType == "EVIDENCE" || m.ProvenanceRefType == "SOURCE") &&
			m.ProvenanceRefID != nil && strings.TrimSpace(*m.ProvenanceRefID) != ""
		if m.ProvenanceStatus == "VERIFIED" && !hasValidAnchor {
			return ErrFactMissingProvenanceRef
		}
		if !hasValidAnchor {
			m.ProvenanceStatus = "UNVERIFIED"
		}

	case "AGENT_DECISION":
		if m.ProvenanceRefType == "DECISION" && m.ProvenanceRefID != nil && *m.ProvenanceRefID != "" {
			m.ProvenanceStatus = "VERIFIED"
		} else if m.ProvenanceStatus == "VERIFIED" {
			m.ProvenanceStatus = "UNVERIFIED"
		}
	}

	return nil
}

// WriteGovernedMemory validates provenance constraints and persists a memory without overwriting history.
func (e *Engine) WriteGovernedMemory(ctx context.Context, m *memory.Memory) error {
	if e.memoryRepo == nil {
		return nil
	}
	if err := ValidateMemoryWrite(m); err != nil {
		return err
	}
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.Status == "" {
		m.Status = "ACTIVE"
	}
	if m.Version <= 0 {
		m.Version = 1
	}
	return e.memoryRepo.Create(ctx, m)
}

// RecordCycleDecisionMemory records an immutable episodic decision memory anchored to a Decision ID.
func (e *Engine) RecordCycleDecisionMemory(ctx context.Context, projectID string, dec *decision.Decision, cycleNumber int64) (*memory.Memory, error) {
	if e.memoryRepo == nil || dec == nil {
		return nil, nil
	}
	decID := dec.ID
	mem := &memory.Memory{
		ID:                uuid.New().String(),
		Type:              memory.TypeDecision,
		Scope:             memory.ScopeProject,
		ProjectID:         projectID,
		AgentID:           "autonomy-core",
		Title:             fmt.Sprintf("Cycle #%d Decision (%s)", cycleNumber, dec.ActionType),
		Content:           fmt.Sprintf("Action=%s | Governance=%s | Intent=%s | Justification=%s", dec.ActionType, dec.GovernanceResult, dec.IntentSummary, dec.Justification),
		SourceType:        "AUTONOMY_CYCLE",
		SourceID:          decID,
		Confidence:        0.95,
		Status:            "ACTIVE",
		Version:           1,
		ProvenanceType:    "AGENT_DECISION",
		ProvenanceRefType: "DECISION",
		ProvenanceRefID:   &decID,
		ProvenanceStatus:  "VERIFIED",
	}
	if err := e.WriteGovernedMemory(ctx, mem); err != nil {
		return nil, err
	}
	return mem, nil
}

func computeAssociativeScore(m memory.Memory, cues []string, projectID string, now time.Time) float64 {
	score := m.Confidence

	// Provenance authority weight (L2 -> L3 grounding)
	switch m.ProvenanceStatus {
	case "VERIFIED":
		score += 0.35
	case "PARTIAL":
		score += 0.15
	}

	// Project boundary affinity ("不混淆" — Plan §0.5D)
	if projectID != "" && m.ProjectID == projectID {
		score += 0.25
	}

	// Multi-cue associative matching (goals + observations + pending issues)
	text := strings.ToLower(m.Title + " " + m.Content)
	matched := 0
	for _, cue := range cues {
		if len(cue) >= 2 && strings.Contains(text, cue) {
			matched++
		}
	}
	if matched > 0 {
		score += float64(matched) * 0.30
	}

	// Temporal recency boost
	age := now.Sub(m.CreatedAt)
	if age < 24*time.Hour {
		score += 0.25
	} else if age < 7*24*time.Hour {
		score += 0.15
	}

	return score
}

func extractAssociationCues(goals []*goal.Goal, obs []domainAutonomy.Observation, issues []string) []string {
	seen := map[string]bool{}
	var cues []string
	addTokens := func(s string) {
		for _, tok := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
			return r == ' ' || r == ',' || r == ':' || r == ';' || r == '(' || r == ')' || r == '[' || r == ']' || r == '"' || r == '/'
		}) {
			if len(tok) >= 3 && !seen[tok] {
				seen[tok] = true
				cues = append(cues, tok)
			}
		}
	}

	for _, g := range goals {
		addTokens(g.Name)
		addTokens(g.Description)
	}
	for _, o := range obs {
		addTokens(o.Source)
		addTokens(o.Summary)
	}
	for _, iss := range issues {
		addTokens(iss)
	}
	return cues
}

func hasContradictoryContent(group []memory.Memory) bool {
	if len(group) < 2 {
		return false
	}
	base := strings.TrimSpace(group[0].Content)
	for _, m := range group[1:] {
		if strings.TrimSpace(m.Content) != base {
			return true
		}
	}
	return false
}

func buildActiveDigest(b *RecalledBundle) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Goals=%d, ActiveTasks=%d, RecentDecisions=%d, Memories=%d, Evidence=%d, Conflicts=%d",
		len(b.ActiveGoals), len(b.ActiveTasks), len(b.RecentDecisions), len(b.Memories), len(b.GroundedEvidence), len(b.Conflicts)))
	if len(b.VerifiedLearnings) > 0 || len(b.NegativeConstraints) > 0 {
		sb.WriteString(fmt.Sprintf(", VerifiedLearnings=%d, AntiPatterns=%d", len(b.VerifiedLearnings), len(b.NegativeConstraints)))
	}
	if len(b.PendingIssues) > 0 {
		sb.WriteString(fmt.Sprintf(", PendingIssues=%d", len(b.PendingIssues)))
	}
	if len(b.RecentFailedTasks) > 0 {
		sb.WriteString(fmt.Sprintf(", FailedTasks=%d", len(b.RecentFailedTasks)))
	}
	if len(b.ActiveProblems) > 0 {
		sb.WriteString(fmt.Sprintf(", ActiveProblems=%d", len(b.ActiveProblems)))
	}
	return sb.String()
}
