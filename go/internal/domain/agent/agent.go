package agent

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

type Agent struct {
	ID                domain.ID `json:"id" yaml:"id"`
	Name              string    `json:"name" yaml:"name"`
	Description       string    `json:"description" yaml:"description"`
	Domain            string    `json:"domain" yaml:"domain"`
	Instructions      string    `json:"instructions" yaml:"instructions"`
	Capabilities      []string  `json:"capabilities" yaml:"capabilities"`
	DefaultProfileID  domain.ID `json:"default_profile_id" yaml:"default_profile_id"`
	ProviderName      string    `json:"provider_name" yaml:"provider_name"`
	ProviderAgentName string    `json:"provider_agent_name" yaml:"provider_agent_name"`
	Enabled           bool      `json:"enabled" yaml:"enabled"`
	Version           int       `json:"version" yaml:"version"`
	CreatedAt         time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" yaml:"updated_at"`
}
