package autonomy

import (
	"context"
	"fmt"

	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/domain/selfmod"
	selfmodSvc "github.com/wade/aia/internal/service/selfmod"
)

// EvolutionOpportunity represents an internally detected limitation or improvement opportunity (Phase 6).
type EvolutionOpportunity struct {
	ID          string   `json:"id"`
	Domain      string   `json:"domain"`
	Summary     string   `json:"summary"`
	Rationale   string   `json:"rationale"`
	TargetFiles []string `json:"target_files"`
	ProposedFix string   `json:"proposed_fix"`
}

// SelfEvolutionEngine manages autonomous identification and remediation of system capabilities.
type SelfEvolutionEngine struct {
	pipeline *selfmodSvc.Pipeline
}

func NewSelfEvolutionEngine(pipeline *selfmodSvc.Pipeline) *SelfEvolutionEngine {
	return &SelfEvolutionEngine{pipeline: pipeline}
}

// RunEvolutionCycle evaluates an identified evolution opportunity through the safe self-modification pipeline.
func (e *SelfEvolutionEngine) RunEvolutionCycle(
	ctx context.Context,
	opp EvolutionOpportunity,
	testRunner selfmodSvc.SandboxTestRunner,
	humanApprover *governance.Actor,
) (*selfmod.SelfModificationProposal, error) {
	if e.pipeline == nil {
		return nil, fmt.Errorf("self-modification pipeline not configured")
	}

	patches := make(map[string]string)
	if len(opp.TargetFiles) > 0 {
		patches[opp.TargetFiles[0]] = opp.ProposedFix
	}

	// 1. Propose
	proposal, err := e.pipeline.Propose(ctx, selfmodSvc.ProposalInput{
		Title:       opp.Summary,
		Description: fmt.Sprintf("Autonomous self-evolution for domain %s", opp.Domain),
		Rationale:   opp.Rationale,
		TargetFiles: opp.TargetFiles,
		Patches:     patches,
	})
	if err != nil {
		return nil, fmt.Errorf("evolution proposal rejected: %w", err)
	}

	// 2. Execute Sandbox Verification (L2)
	verifiedProposal, err := e.pipeline.ExecuteSandboxVerification(ctx, proposal.ID, testRunner)
	if err != nil {
		return verifiedProposal, fmt.Errorf("evolution sandbox verification failed: %w", err)
	}

	// 3. If Level B (Core), require human approval to apply
	if verifiedProposal.RequiresApproval && verifiedProposal.Status == selfmod.StatusPendingApproval {
		if humanApprover != nil && humanApprover.IsHuman() {
			if err := e.pipeline.ApplyMerge(ctx, verifiedProposal.ID, *humanApprover); err != nil {
				return verifiedProposal, fmt.Errorf("failed to apply approved merge: %w", err)
			}
		}
	}

	return verifiedProposal, nil
}
