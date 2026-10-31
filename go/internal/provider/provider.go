package provider

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/profile"
)

type StartRequest struct {
	ExecutionID    domain.ID
	TaskID         domain.ID
	Agent          agent.Agent
	Prompt         string
	WorkDir        string
	Profile        profile.ExecutionProfile
	ConversationID string
	Timeout        time.Duration
}

type ProviderEvent struct {
	ExecutionID domain.ID       `json:"execution_id"`
	Sequence    uint64          `json:"sequence"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Timestamp   time.Time       `json:"timestamp"`
}

type ProviderResult struct {
	ConversationID string        `json:"conversation_id"`
	Status         string        `json:"status"`
	Response       string        `json:"response"`
	Error          string        `json:"error"`
	Usage          TokenUsage    `json:"usage"`
	Duration       time.Duration `json:"duration"`
}

type TokenUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

type Provider interface {
	Name() string
	Version(ctx context.Context) (string, error)
	HealthCheck(ctx context.Context) error
	Start(ctx context.Context, req StartRequest) (ProviderSession, error)
}

type ProviderSession interface {
	Send(ctx context.Context, content string) error
	Events() <-chan ProviderEvent
	CloseInput() error
	Stop(ctx context.Context) error
	Wait(ctx context.Context) (ProviderResult, error)
}
