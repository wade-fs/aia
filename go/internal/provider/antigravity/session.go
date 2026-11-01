package antigravity

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wade/aia/internal/provider"
)

type Session struct {
	req       provider.StartRequest
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	events    chan provider.ProviderEvent
	stderrBuf bytes.Buffer
	seqCounter uint64

	resultMu  sync.Mutex
	result    provider.ProviderResult
	errResult error
	done      chan struct{}

	closed    bool
	mu        sync.Mutex
}

func NewSession(ctx context.Context, executable string, args []string, req provider.StartRequest) (*Session, error) {
	plan := &SandboxPlan{
		Executable:    executable,
		Args:          args,
		Env:           ScrubEnvironment(os.Environ(), nil),
		IsolationTier: IsolationTierEnvOnly,
	}
	return NewSessionWithPlan(ctx, plan, req)
}

func NewSessionWithPlan(ctx context.Context, plan *SandboxPlan, req provider.StartRequest) (*Session, error) {
	if plan == nil {
		return nil, fmt.Errorf("sandbox plan is required")
	}
	cmd := exec.Command(plan.Executable, plan.Args...)
	if req.WorkDir != "" {
		cmd.Dir = req.WorkDir
	}
	cmd.Env = plan.Env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	s := &Session{
		req:    req,
		cmd:    cmd,
		stdin:  stdin,
		events: make(chan provider.ProviderEvent, 100),
		done:   make(chan struct{}),
	}
	cmd.Stderr = &s.stderrBuf

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to start antigravity process (%s): %w", plan.Executable, err)
	}

	go s.readStdout(stdout)

	// Send initial prompt if present
	if req.Prompt != "" {
		if err := s.Send(ctx, req.Prompt); err != nil {
			_ = s.Stop(ctx)
			return nil, fmt.Errorf("failed to send initial prompt: %w", err)
		}
	}

	return s, nil
}

func (s *Session) PID() int {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Pid
	}
	return 0
}

func (s *Session) Send(ctx context.Context, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	msg := map[string]any{
		"event": "user",
		"message": map[string]string{
			"content": content,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal prompt message: %w", err)
	}

	data = append(data, '\n')
	_, err = s.stdin.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}
	return nil
}

func (s *Session) Events() <-chan provider.ProviderEvent {
	return s.events
}

func (s *Session) CloseInput() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stdin != nil {
		return s.stdin.Close()
	}
	return nil
}

func (s *Session) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	if s.cmd != nil && s.cmd.Process != nil {
		// Send interrupt first
		_ = s.cmd.Process.Signal(os.Interrupt)

		// Wait briefly or force kill
		done := make(chan struct{})
		go func() {
			_ = s.cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = s.cmd.Process.Kill()
		}
	}
	return nil
}

func (s *Session) Wait(ctx context.Context) (provider.ProviderResult, error) {
	select {
	case <-ctx.Done():
		_ = s.Stop(ctx)
		return provider.ProviderResult{Status: "CANCELLED"}, ctx.Err()
	case <-s.done:
		s.resultMu.Lock()
		defer s.resultMu.Unlock()
		return s.result, s.errResult
	}
}

func (s *Session) readStdout(r io.Reader) {
	defer close(s.events)
	defer close(s.done)

	scanner := bufio.NewScanner(r)
	// Buffer up to 10MB per line for large tool outputs
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var lastResponse strings.Builder
	var lastResultErr string
	var lastStatus = "UNKNOWN"
	var lastConvID = s.req.ConversationID
	var usage provider.TokenUsage
	var duration time.Duration

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var generic map[string]any
		if err := json.Unmarshal(line, &generic); err != nil {
			continue
		}

		eventType, _ := generic["event"].(string)
		if convID, ok := generic["conversation_id"].(string); ok && convID != "" {
			lastConvID = convID
		}

		seq := atomic.AddUint64(&s.seqCounter, 1)
		ev := provider.ProviderEvent{
			ExecutionID: s.req.ExecutionID,
			Sequence:    seq,
			Type:        eventType,
			Payload:     line,
			Timestamp:   time.Now().UTC(),
		}
		s.events <- ev

		if eventType == "step_update" {
			if su, ok := generic["step_update"].(map[string]any); ok {
				if delta, ok := su["text_delta"].(string); ok {
					lastResponse.WriteString(delta)
				}
			}
		} else if eventType == "result" {
			if res, ok := generic["result"].(map[string]any); ok {
				if st, ok := res["status"].(string); ok {
					lastStatus = st
				}
				if errMsg, ok := res["error"].(string); ok && errMsg != "" {
					lastResultErr = errMsg
				}
				if resp, ok := res["response"].(string); ok && resp != "" {
					lastResponse.Reset()
					lastResponse.WriteString(resp)
				}
				if durSec, ok := res["duration_seconds"].(float64); ok {
					duration = time.Duration(durSec * float64(time.Second))
				}
				if u, ok := res["usage"].(map[string]any); ok {
					if it, ok := u["input_tokens"].(float64); ok {
						usage.InputTokens = int64(it)
					}
					if ot, ok := u["output_tokens"].(float64); ok {
						usage.OutputTokens = int64(ot)
					}
					if tt, ok := u["total_tokens"].(float64); ok {
						usage.TotalTokens = int64(tt)
					}
				}
			}
			// Closing stdin indicates turn completion to agy so process exits
			s.mu.Lock()
			if s.stdin != nil {
				_ = s.stdin.Close()
				s.stdin = nil
			}
			s.mu.Unlock()
			break
		}
	}

	waitErr := s.cmd.Wait()
	errOut := strings.TrimSpace(s.stderrBuf.String())
	if errOut == "" && lastResultErr != "" {
		errOut = lastResultErr
	}
	if waitErr != nil {
		if lastStatus == "UNKNOWN" {
			lastStatus = "ERROR"
		}
		if errOut == "" {
			errOut = waitErr.Error()
		}
	}

	s.resultMu.Lock()
	s.result = provider.ProviderResult{
		ConversationID: lastConvID,
		Status:         lastStatus,
		Response:       lastResponse.String(),
		Error:          errOut,
		Usage:          usage,
		Duration:       duration,
	}
	s.resultMu.Unlock()
}
