package policy

import (
	"github.com/wade/aia/internal/domain"
)

type ActionType string

const (
	ActionAllow ActionType = "ALLOW"
	ActionAsk   ActionType = "ASK"
	ActionDeny  ActionType = "DENY"
)

type Rule struct {
	Name        string           `json:"name" yaml:"name"`
	Pattern     string           `json:"pattern" yaml:"pattern"`
	Action      ActionType       `json:"action" yaml:"action"`
	MaxRisk     domain.RiskLevel `json:"max_risk" yaml:"max_risk"`
	Description string           `json:"description" yaml:"description"`
}

type Policy struct {
	ID          domain.ID `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description" yaml:"description"`
	Rules       []Rule    `json:"rules" yaml:"rules"`
}
