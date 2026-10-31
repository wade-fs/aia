package policy

import (
	"context"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/policy"
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

type Evaluator struct {
	defaultPolicy policy.Policy
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		defaultPolicy: policy.Policy{
			ID:          "default-policy",
			Name:        "Default Safety Policy",
			Description: "Denies destructive OS actions and asks for high-risk operations",
			Rules: []policy.Rule{
				{
					Name:        "Deny System Destructive Tools",
					Pattern:     "rm -rf /",
					Action:      policy.ActionDeny,
					MaxRisk:     domain.RiskCritical,
					Description: "Strictly forbids deleting system root",
				},
				{
					Name:        "High Risk Requires Approval",
					Pattern:     "*",
					Action:      policy.ActionAsk,
					MaxRisk:     domain.RiskHigh,
					Description: "High-risk tasks require OWNER confirmation",
				},
			},
		},
	}
}

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
