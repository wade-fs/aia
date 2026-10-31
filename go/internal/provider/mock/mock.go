package mock

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/wade/aia/internal/provider"
)

type MockProvider struct {
	NameFunc        func() string
	VersionFunc     func(ctx context.Context) (string, error)
	HealthCheckFunc func(ctx context.Context) error
	StartFunc       func(ctx context.Context, req provider.StartRequest) (provider.ProviderSession, error)
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		NameFunc: func() string { return "mock" },
		VersionFunc: func(ctx context.Context) (string, error) {
			return "1.0.0-mock", nil
		},
		HealthCheckFunc: func(ctx context.Context) error {
			return nil
		},
	}
}

func (m *MockProvider) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock"
}

func (m *MockProvider) Version(ctx context.Context) (string, error) {
	if m.VersionFunc != nil {
		return m.VersionFunc(ctx)
	}
	return "1.0.0-mock", nil
}

func (m *MockProvider) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}

func (m *MockProvider) Start(ctx context.Context, req provider.StartRequest) (provider.ProviderSession, error) {
	if m.StartFunc != nil {
		return m.StartFunc(ctx, req)
	}
	return NewMockSession(req), nil
}

type MockSession struct {
	req      provider.StartRequest
	events   chan provider.ProviderEvent
	closed   bool
	mu       sync.Mutex
	Response string
}

func NewMockSession(req provider.StartRequest) *MockSession {
	s := &MockSession{
		req:      req,
		events:   make(chan provider.ProviderEvent, 10),
		Response: "Task executed successfully by mock provider.",
	}
	go s.simulate()
	return s
}

func (s *MockSession) simulate() {
	time.Sleep(10 * time.Millisecond)
	payload, _ := json.Marshal(map[string]any{"text_delta": s.Response})
	s.events <- provider.ProviderEvent{
		ExecutionID: s.req.ExecutionID,
		Sequence:    1,
		Type:        "step_update",
		Payload:     payload,
		Timestamp:   time.Now().UTC(),
	}
	time.Sleep(10 * time.Millisecond)
	close(s.events)
}

func (s *MockSession) Send(ctx context.Context, content string) error {
	return nil
}

func (s *MockSession) Events() <-chan provider.ProviderEvent {
	return s.events
}

func (s *MockSession) CloseInput() error {
	return nil
}

func (s *MockSession) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *MockSession) Wait(ctx context.Context) (provider.ProviderResult, error) {
	return provider.ProviderResult{
		ConversationID: s.req.ConversationID,
		Status:         "SUCCESS",
		Response:       s.Response,
		Usage: provider.TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
		Duration: 20 * time.Millisecond,
	}, nil
}
