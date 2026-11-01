package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/governance"
	domainPolicy "github.com/wade/aia/internal/domain/policy"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type Decision string

const (
	DecisionAllow Decision = "ALLOW"
	DecisionAsk   Decision = "ASK"
	DecisionDeny  Decision = "DENY"
)

type EvaluationResult struct {
	Decision Decision
	Reason   string
	Risk     domain.RiskLevel
}

// PolicyDecision encapsulates the complete governance outcome for an operation.
type PolicyDecision struct {
	Allowed          bool                        `json:"allowed"`
	RequiresApproval bool                        `json:"requires_approval"`
	RiskTier         governance.RiskTier         `json:"risk_tier"`
	GovernanceAction governance.GovernanceAction `json:"governance_action"`
	PolicyID         string                      `json:"policy_id"`
	RuleName         string                      `json:"rule_name"`
	Reason           string                      `json:"reason"`
}

// OperationContext describes an action requested by an agent or subsystem.
type OperationContext struct {
	ActorType     string  `json:"actor_type"` // e.g. "AGENT", "USER", "SYSTEM"
	ActorID       string  `json:"actor_id"`
	OperationType string  `json:"operation_type"` // "FILE_WRITE", "FILE_DELETE", "FINANCIAL_TRANSACTION", "SYSTEM_COMMAND", "API_CALL"
	TargetPath    string  `json:"target_path,omitempty"`
	Command       string  `json:"command,omitempty"`
	Amount        float64 `json:"amount,omitempty"`
	IsRealMoney   bool    `json:"is_real_money,omitempty"`
	DailySpend    float64 `json:"daily_spend,omitempty"`
	MaxDailySpend float64 `json:"max_daily_spend,omitempty"`
	TaskID        string  `json:"task_id,omitempty"`
	DecisionID    string  `json:"decision_id,omitempty"`
}

type BudgetLimits struct {
	MaxDailyTokens           int64   `json:"max_daily_tokens"`
	MaxDailyCycles           int     `json:"max_daily_cycles"`
	MaxSingleExecutionTokens int64   `json:"max_single_execution_tokens"`
	MaxDailySpend            float64 `json:"max_daily_spend"`
}

type Evaluator struct {
	defaultPolicy domainPolicy.Policy
	auditRepo     *repository.AuditRepository
	allowedPaths  []string
	maxDailySpend float64
	budgetLimits  BudgetLimits
	mu            sync.RWMutex
}

func NewEvaluator() *Evaluator {
	return NewEvaluatorWithOptions(nil, nil, 0)
}

func NewEvaluatorWithOptions(auditRepo *repository.AuditRepository, allowedPaths []string, maxDailySpend float64) *Evaluator {
	if len(allowedPaths) == 0 {
		allowedPaths = []string{
			"/home/wade/src/github/aia",
			"/tmp",
			"/var/tmp",
		}
	}

	return &Evaluator{
		defaultPolicy: domainPolicy.Policy{
			ID:          "policy-governance-core-v1",
			Name:        "AIA Core Governance Safety Policy",
			Description: "Three-layer operational governance enforcing file boundaries, virtual financial assets, and human approval gates.",
			Rules: []domainPolicy.Rule{
				{
					Name:        "Deny System Destructive Tools",
					Pattern:     "rm -rf /",
					Action:      domainPolicy.ActionDeny,
					MaxRisk:     domain.RiskCritical,
					Description: "Strictly forbids deleting system root or critical files",
				},
				{
					Name:        "Deny Real Financial Transactions",
					Pattern:     "financial:real_money",
					Action:      domainPolicy.ActionDeny,
					MaxRisk:     domain.RiskCritical,
					Description: "AIA operates strictly in paper trading mode; real money transactions are forbidden",
				},
				{
					Name:        "Enforce Path Whitelist",
					Pattern:     "file:write_boundary",
					Action:      domainPolicy.ActionDeny,
					MaxRisk:     domain.RiskHigh,
					Description: "File mutations outside whitelisted directories are intercepted and denied",
				},
				{
					Name:        "High Risk Requires Approval",
					Pattern:     "*",
					Action:      domainPolicy.ActionAsk,
					MaxRisk:     domain.RiskHigh,
					Description: "High-risk tasks require explicit human owner confirmation",
				},
			},
		},
		auditRepo:     auditRepo,
		allowedPaths:  allowedPaths,
		maxDailySpend: maxDailySpend,
		budgetLimits: BudgetLimits{
			MaxDailySpend: maxDailySpend,
		},
	}
}

func (e *Evaluator) SetBudgetLimits(limits BudgetLimits) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.budgetLimits = limits
	if limits.MaxDailySpend > 0 {
		e.maxDailySpend = limits.MaxDailySpend
	}
}

func (e *Evaluator) GetBudgetLimits() BudgetLimits {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.budgetLimits
}

// EvaluateResourceBudget enforces hard daily Token & Autonomy Cycle budget gates prior to execution or cycle dispatch (AIA-08, AIA-17, AIA-19).
func (e *Evaluator) EvaluateResourceBudget(ctx context.Context, actorID string, dailyTokens int64, dailyCycles int) PolicyDecision {
	e.mu.RLock()
	limits := e.budgetLimits
	repo := e.auditRepo
	policyID := string(e.defaultPolicy.ID)
	e.mu.RUnlock()

	dec := PolicyDecision{
		Allowed:          true,
		RequiresApproval: false,
		RiskTier:         governance.RiskTierLow,
		GovernanceAction: governance.GovernanceActionAllowed,
		PolicyID:         policyID,
		RuleName:         "Resource Budget Gate",
		Reason:           "Daily token and cycle usage within configured budget.",
	}

	if limits.MaxDailyTokens > 0 && dailyTokens >= limits.MaxDailyTokens {
		dec.Allowed = false
		dec.RiskTier = governance.RiskTierHigh
		dec.GovernanceAction = governance.GovernanceActionBlocked
		dec.RuleName = "Daily Token Budget Hard Gate"
		dec.Reason = fmt.Sprintf("Daily token consumption (%d) has reached or exceeded hard limit (%d)", dailyTokens, limits.MaxDailyTokens)
		e.recordAudit(ctx, repo, OperationContext{
			ActorType:     "AGENT",
			ActorID:       actorID,
			OperationType: "BUDGET_CHECK_TOKENS",
		}, dec)
		return dec
	}

	if limits.MaxDailyCycles > 0 && dailyCycles >= limits.MaxDailyCycles {
		dec.Allowed = false
		dec.RiskTier = governance.RiskTierMedium
		dec.GovernanceAction = governance.GovernanceActionBlocked
		dec.RuleName = "Daily Autonomy Cycle Hard Gate"
		dec.Reason = fmt.Sprintf("Daily autonomy cycle count (%d) has reached or exceeded hard limit (%d)", dailyCycles, limits.MaxDailyCycles)
		e.recordAudit(ctx, repo, OperationContext{
			ActorType:     "AGENT",
			ActorID:       actorID,
			OperationType: "BUDGET_CHECK_CYCLES",
		}, dec)
		return dec
	}

	return dec
}

func (e *Evaluator) SetAuditRepo(repo *repository.AuditRepository) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.auditRepo = repo
}

func (e *Evaluator) SetAllowedPaths(paths []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowedPaths = paths
}

func (e *Evaluator) AddAllowedPath(path string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowedPaths = append(e.allowedPaths, path)
}

// EvaluateRisk provides backwards compatibility for pure RiskLevel queries.
func (e *Evaluator) EvaluateRisk(ctx context.Context, risk domain.RiskLevel) EvaluationResult {
	if risk == domain.RiskCritical {
		return EvaluationResult{
			Decision: DecisionDeny,
			Reason:   "Critical risk actions are not allowed in autonomous mode",
			Risk:     risk,
		}
	}
	if risk == domain.RiskHigh {
		return EvaluationResult{
			Decision: DecisionAsk,
			Reason:   "High risk task requires explicit owner approval",
			Risk:     risk,
		}
	}
	return EvaluationResult{
		Decision: DecisionAllow,
		Reason:   "Action risk within safe limits",
		Risk:     risk,
	}
}

// EvaluateOperation performs multi-dimensional governance policy evaluation
// against the three governance boundaries (Path, Financial, Risk Tier),
// and writes an immutable audit record whenever an intercept or decision occurs.
func (e *Evaluator) EvaluateOperation(ctx context.Context, op OperationContext) PolicyDecision {
	e.mu.RLock()
	allowedPaths := make([]string, len(e.allowedPaths))
	copy(allowedPaths, e.allowedPaths)
	maxSpend := e.maxDailySpend
	repo := e.auditRepo
	policyID := string(e.defaultPolicy.ID)
	e.mu.RUnlock()

	actorType := op.ActorType
	if actorType == "" {
		actorType = "AGENT"
	}
	actorID := op.ActorID
	if actorID == "" {
		actorID = "autonomous-agent"
	}

	var dec PolicyDecision
	dec.PolicyID = policyID

	// 1. Financial Transaction Guard: Real capital operations are unconditionally blocked.
	if op.OperationType == "FINANCIAL_TRANSACTION" || strings.HasPrefix(op.OperationType, "FINANCIAL_") {
		if op.IsRealMoney {
			dec.Allowed = false
			dec.RequiresApproval = false
			dec.RiskTier = governance.RiskTierCritical
			dec.GovernanceAction = governance.GovernanceActionBlocked
			dec.RuleName = "Deny Real Financial Transactions"
			dec.Reason = "Real-money financial trading is strictly blocked. AIA executes only virtual/paper simulation orders."
			e.recordAudit(ctx, repo, op, dec)
			return dec
		}
	}

	// 2. Destructive System Commands Guard
	if op.Command != "" {
		cmdLower := strings.ToLower(strings.TrimSpace(op.Command))
		if strings.Contains(cmdLower, "rm -rf /") ||
			strings.Contains(cmdLower, "mkfs") ||
			strings.Contains(cmdLower, "dd if=/dev/zero") ||
			strings.Contains(cmdLower, ":(){ :|:& };:") {
			dec.Allowed = false
			dec.RequiresApproval = false
			dec.RiskTier = governance.RiskTierCritical
			dec.GovernanceAction = governance.GovernanceActionBlocked
			dec.RuleName = "Deny System Destructive Tools"
			dec.Reason = fmt.Sprintf("Command %q contains destructive system payload and is permanently forbidden.", op.Command)
			e.recordAudit(ctx, repo, op, dec)
			return dec
		}
	}

	// 3. Path Whitelist Guard
	if (op.OperationType == "FILE_WRITE" || op.OperationType == "FILE_DELETE" || op.OperationType == "FILE_MUTATE") && op.TargetPath != "" {
		cleanTarget := filepath.Clean(op.TargetPath)
		pathAllowed := false
		for _, ap := range allowedPaths {
			cleanAllowed := filepath.Clean(ap)
			rel, err := filepath.Rel(cleanAllowed, cleanTarget)
			if err == nil && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
				pathAllowed = true
				break
			}
		}

		if !pathAllowed {
			dec.Allowed = false
			dec.RequiresApproval = false
			dec.RiskTier = governance.RiskTierHigh
			dec.GovernanceAction = governance.GovernanceActionBlocked
			dec.RuleName = "Enforce Path Whitelist"
			dec.Reason = fmt.Sprintf("Path %q is outside all whitelisted directories: %v", op.TargetPath, allowedPaths)
			e.recordAudit(ctx, repo, op, dec)
			return dec
		}
	}

	// 4. Budget Limit Guard
	if maxSpend > 0 && op.DailySpend+op.Amount > maxSpend {
		dec.Allowed = false
		dec.RequiresApproval = true
		dec.RiskTier = governance.RiskTierHigh
		dec.GovernanceAction = governance.GovernanceActionRequiredApproval
		dec.RuleName = "Budget Limit Guard"
		dec.Reason = fmt.Sprintf("Cumulative daily spend (%.2f + %.2f) exceeds configured limit of %.2f", op.DailySpend, op.Amount, maxSpend)
		e.recordAudit(ctx, repo, op, dec)
		return dec
	}

	// 5. Default Risk Tier Assessment
	switch op.OperationType {
	case "FINANCIAL_TRANSACTION":
		// Virtual trade proposal > high value requires approval
		if op.Amount > 100000 {
			dec.Allowed = false
			dec.RequiresApproval = true
			dec.RiskTier = governance.RiskTierHigh
			dec.GovernanceAction = governance.GovernanceActionRequiredApproval
			dec.RuleName = "High Value Proposal Requires Approval"
			dec.Reason = fmt.Sprintf("Virtual transaction amount %.2f requires human review.", op.Amount)
		} else {
			dec.Allowed = true
			dec.RequiresApproval = false
			dec.RiskTier = governance.RiskTierMedium
			dec.GovernanceAction = governance.GovernanceActionAllowed
			dec.RuleName = "Routine Virtual Trade Allowed"
			dec.Reason = "Virtual transaction within autonomous limits."
		}

	case "SYSTEM_COMMAND":
		dec.Allowed = false
		dec.RequiresApproval = true
		dec.RiskTier = governance.RiskTierHigh
		dec.GovernanceAction = governance.GovernanceActionRequiredApproval
		dec.RuleName = "System Commands Require Approval"
		dec.Reason = "Execution of external OS command requires explicit owner approval."

	default:
		dec.Allowed = true
		dec.RequiresApproval = false
		dec.RiskTier = governance.RiskTierLow
		dec.GovernanceAction = governance.GovernanceActionAllowed
		dec.RuleName = "Default Safe Limit"
		dec.Reason = "Operation is within standard safe operational boundaries."
	}

	e.recordAudit(ctx, repo, op, dec)
	return dec
}

func (e *Evaluator) recordAudit(ctx context.Context, repo *repository.AuditRepository, op OperationContext, dec PolicyDecision) {
	if repo == nil {
		return
	}

	actorType := op.ActorType
	if actorType == "" {
		actorType = "AGENT"
	}
	actorID := op.ActorID
	if actorID == "" {
		actorID = "autonomous-agent"
	}

	details := map[string]any{
		"rule_name":         dec.RuleName,
		"reason":            dec.Reason,
		"allowed":           dec.Allowed,
		"requires_approval": dec.RequiresApproval,
		"operation_type":    op.OperationType,
		"target_path":       op.TargetPath,
		"command":           op.Command,
		"amount":            op.Amount,
		"is_real_money":     op.IsRealMoney,
	}
	detailsJSON, _ := json.Marshal(details)

	policyID := dec.PolicyID
	var decisionID *string
	if op.DecisionID != "" {
		decisionID = &op.DecisionID
	}
	riskTierStr := string(dec.RiskTier)
	govActionStr := string(dec.GovernanceAction)

	resourceType := op.OperationType
	if resourceType == "" {
		resourceType = "OPERATION"
	}
	resourceID := op.TargetPath
	if resourceID == "" {
		resourceID = op.TaskID
	}
	if resourceID == "" {
		resourceID = "system"
	}

	_ = repo.Append(ctx, &repository.AuditLogEntry{
		ID:               uuid.New().String(),
		ActorType:        actorType,
		ActorID:          actorID,
		Action:           "POLICY_EVALUATION",
		ResourceType:     resourceType,
		ResourceID:       resourceID,
		Outcome:          govActionStr,
		Details:          detailsJSON,
		PolicyID:         &policyID,
		DecisionID:       decisionID,
		RiskTier:         &riskTierStr,
		GovernanceAction: &govActionStr,
		CreatedAt:        time.Now().UTC(),
	})
}
