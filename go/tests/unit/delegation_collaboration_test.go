package unit

import (
	"context"
	"testing"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	domainExp "github.com/wade/aia/internal/domain/experiment"
	"github.com/wade/aia/internal/orchestrator"
	"github.com/wade/aia/internal/service/delegation"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

func TestDelegation_AcceptanceGateEvaluation_And_ExtendedFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	expRepo := repository.NewExperimentRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	taskSvc := taskService.NewService(taskRepo, nil)
	delSvc := delegation.NewService(expRepo, taskSvc, auditRepo)

	// 1. Test Extended Fields & Collaboration Mode
	dl := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	parentID := "task-parent-123"
	req, err := delSvc.RequestDelegation(ctx, delegation.CreateDelegationInput{
		RequesterAgentID:  "personal-agent",
		TargetAgentID:     "coding-agent",
		ParentTaskID:      &parentID,
		Deadline:          &dl,
		CollaborationMode: domainExp.CollabReviewLoop,
		PermissionScope:   "READ_WRITE_CODE",
		TaskSpec: map[string]any{
			"summary": "Fix critical data race",
		},
		AcceptanceCriteria: map[string]any{
			"required_keys": []string{"artifact_type", "benchmark_score"},
			"min_score":     85.0,
		},
	})
	if err != nil {
		t.Fatalf("RequestDelegation failed: %v", err)
	}

	if req.ParentTaskID == nil || *req.ParentTaskID != parentID {
		t.Errorf("expected parent task ID %s, got %v", parentID, req.ParentTaskID)
	}
	if req.CollaborationMode != domainExp.CollabReviewLoop {
		t.Errorf("expected collaboration mode %s, got %s", domainExp.CollabReviewLoop, req.CollaborationMode)
	}
	if req.PermissionScope != "READ_WRITE_CODE" {
		t.Errorf("expected permission scope READ_WRITE_CODE, got %s", req.PermissionScope)
	}

	// Read back from repo and verify persistence & unpacking
	saved, err := delSvc.Get(ctx, req.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if saved.ParentTaskID == nil || *saved.ParentTaskID != parentID {
		t.Errorf("expected persisted parent task ID %s, got %v", parentID, saved.ParentTaskID)
	}
	if saved.CollaborationMode != domainExp.CollabReviewLoop {
		t.Errorf("expected persisted collaboration mode %s, got %s", domainExp.CollabReviewLoop, saved.CollaborationMode)
	}

	// 2. Deliverable failing acceptance criteria (min_score: 85, deliverable has 60)
	if err := delSvc.AcceptDelegation(ctx, req.ID); err != nil {
		t.Fatalf("AcceptDelegation failed: %v", err)
	}
	if err := delSvc.DeliverWork(ctx, req.ID, map[string]any{
		"artifact_type":   "PATCH",
		"benchmark_score": 60.0,
		"score":           60.0,
	}); err != nil {
		t.Fatalf("DeliverWork failed: %v", err)
	}

	// Even if caller passed=true, AcceptanceGate must enforce failure!
	if err := delSvc.VerifyDelivery(ctx, req.ID, true, "Caller attempted to pass"); err != nil {
		t.Fatalf("VerifyDelivery failed: %v", err)
	}
	saved, _ = delSvc.Get(ctx, req.ID)
	if saved.Status != "FAILED" {
		t.Errorf("expected status FAILED due to acceptance gate score breach, got %s", saved.Status)
	}

	// 3. Deliverable passing acceptance criteria
	req2, err := delSvc.RequestDelegation(ctx, delegation.CreateDelegationInput{
		RequesterAgentID: "personal-agent",
		TargetAgentID:    "coding-agent",
		AcceptanceCriteria: map[string]any{
			"required_keys": []string{"artifact_type"},
			"min_score":     85.0,
		},
	})
	if err != nil {
		t.Fatalf("RequestDelegation 2 failed: %v", err)
	}
	_ = delSvc.AcceptDelegation(ctx, req2.ID)
	_ = delSvc.DeliverWork(ctx, req2.ID, map[string]any{
		"artifact_type": "PATCH",
		"score":         92.5,
	})
	if err := delSvc.VerifyDelivery(ctx, req2.ID, true, "Criteria fully satisfied"); err != nil {
		t.Fatalf("VerifyDelivery 2 failed: %v", err)
	}
	saved2, _ := delSvc.Get(ctx, req2.ID)
	if saved2.Status != "VERIFIED" {
		t.Errorf("expected status VERIFIED, got %s", saved2.Status)
	}
}

func TestDelegation_CapabilityDiscovery(t *testing.T) {
	agents := []agent.Agent{
		{
			ID:           domain.ID("coding-agent"),
			Name:         "Software Engineer",
			Domain:       "coding",
			Capabilities: []string{"go", "unit_test", "refactor", "git"},
			Enabled:      true,
		},
		{
			ID:           domain.ID("writing-agent"),
			Name:         "Content Writer",
			Domain:       "writing",
			Capabilities: []string{"novel", "proofread", "character_arc"},
			Enabled:      true,
		},
		{
			ID:           domain.ID("investment-agent"),
			Name:         "Quantitative Analyst",
			Domain:       "investment",
			Capabilities: []string{"financial_statement", "stock_screen", "brier_score"},
			Enabled:      true,
		},
	}

	// Test capability-based dynamic matching
	match1 := orchestrator.MatchAgentByCapabilities("Please examine financial_statement for TSMC", agents)
	if match1 != "investment-agent" {
		t.Errorf("expected investment-agent, got %s", match1)
	}

	match2 := orchestrator.MatchAgentByCapabilities("We need a refactor of the repository modules", agents)
	if match2 != "coding-agent" {
		t.Errorf("expected coding-agent, got %s", match2)
	}

	match3 := orchestrator.MatchAgentByCapabilities("Check chapter outline and character_arc", agents)
	if match3 != "writing-agent" {
		t.Errorf("expected writing-agent, got %s", match3)
	}
}
