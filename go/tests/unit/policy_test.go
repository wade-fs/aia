package unit

import (
	"context"
	"testing"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/policy"
)

func TestPolicyEvaluation(t *testing.T) {
	eval := policy.NewEvaluator()
	ctx := context.Background()

	// Low risk -> ALLOW
	res := eval.EvaluateRisk(ctx, domain.RiskLow)
	if res.Decision != policy.DecisionAllow {
		t.Errorf("expected ALLOW for low risk, got %s", res.Decision)
	}

	// High risk -> ASK
	res = eval.EvaluateRisk(ctx, domain.RiskHigh)
	if res.Decision != policy.DecisionAsk {
		t.Errorf("expected ASK for high risk, got %s", res.Decision)
	}

	// Critical risk -> DENY
	res = eval.EvaluateRisk(ctx, domain.RiskCritical)
	if res.Decision != policy.DecisionDeny {
		t.Errorf("expected DENY for critical risk, got %s", res.Decision)
	}
}
