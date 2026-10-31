package profile

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

type ExecutionProfile struct {
	ID                 domain.ID         `json:"id" yaml:"id"`
	Name               string            `json:"name" yaml:"name"`
	Description        string            `json:"description" yaml:"description"`
	Model              string            `json:"model" yaml:"model"`
	Effort             string            `json:"effort" yaml:"effort"`
	Mode               string            `json:"mode" yaml:"mode"`
	Sandbox            bool              `json:"sandbox" yaml:"sandbox"`
	Timeout            time.Duration     `json:"timeout" yaml:"timeout"`
	MaxTokens          int               `json:"max_tokens" yaml:"max_tokens"`
	ExtraFlags         map[string]string `json:"extra_flags" yaml:"extra_flags"`
	AllowedDirectories []string          `json:"allowed_directories" yaml:"allowed_directories"`
	Version            int               `json:"version" yaml:"version"`
}
