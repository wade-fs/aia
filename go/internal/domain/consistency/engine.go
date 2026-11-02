package consistency

import (
	"context"
	"fmt"
	"sync"
)

// Engine executes registered consistency rules over a KnowledgeSnapshot (Plan2.md §4).
type Engine struct {
	mu    sync.RWMutex
	rules map[string]ConsistencyRule
}

func NewEngine() *Engine {
	e := &Engine{
		rules: make(map[string]ConsistencyRule),
	}
	// Register default core rules
	e.RegisterRule(&MissingReferenceRule{})
	e.RegisterRule(&TemporalConflictRule{})
	return e
}

func (e *Engine) RegisterRule(rule ConsistencyRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.ID()] = rule
}

func (e *Engine) Evaluate(ctx context.Context, snapshot *KnowledgeSnapshot) ([]Finding, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var allFindings []Finding
	for _, rule := range e.rules {
		findings, err := rule.Evaluate(ctx, snapshot)
		if err != nil {
			return nil, fmt.Errorf("rule %s evaluation failed: %w", rule.ID(), err)
		}
		allFindings = append(allFindings, findings...)
	}
	return allFindings, nil
}

func (e *Engine) EvaluateDomain(ctx context.Context, domain string, snapshot *KnowledgeSnapshot) ([]Finding, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var allFindings []Finding
	for _, rule := range e.rules {
		if rule.Domain() == "generic" || rule.Domain() == domain {
			findings, err := rule.Evaluate(ctx, snapshot)
			if err != nil {
				return nil, fmt.Errorf("rule %s evaluation failed for domain %s: %w", rule.ID(), domain, err)
			}
			allFindings = append(allFindings, findings...)
		}
	}
	return allFindings, nil
}
