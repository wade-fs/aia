package tracing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const (
	traceIDKey contextKey = "aia_trace_id"
	spanKey    contextKey = "aia_current_span"
)

// Standard Span Operation Names (AIA-19 §8.2).
const (
	SpanAutonomyCycle       = "autonomy.cycle"
	SpanAutonomyObserve     = "autonomy.observe"
	SpanAutonomyAssess      = "autonomy.assess"
	SpanAutonomyDiscover    = "autonomy.discover"
	SpanAutonomyDeliberate  = "autonomy.deliberate"
	SpanAutonomyDecide      = "autonomy.decide"
	SpanGovernanceEvaluate  = "governance.evaluate"
	SpanRuntimeDispatch     = "runtime.dispatch"
	SpanWorkerExecute       = "worker.execute"
	SpanHarnessInvoke       = "harness.invoke"
	SpanExecutionVerify     = "execution.verify"
	SpanLearningRecord      = "learning.record"
	SpanProblemAnalysis     = "problem.analyze"
	SpanPlanExecution       = "plan.execute"
	SpanBackupOperation     = "system.backup"
)

// Span represents an individual unit of observable execution (AIA-19 §8.2).
type Span struct {
	TraceID      string         `json:"trace_id"`
	SpanID       string         `json:"span_id"`
	ParentSpanID string         `json:"parent_span_id,omitempty"`
	Name         string         `json:"name"`
	StartTime    time.Time      `json:"start_time"`
	EndTime      *time.Time     `json:"end_time,omitempty"`
	DurationMs   int64          `json:"duration_ms"`
	Status       string         `json:"status"` // OK, ERROR
	Attributes   map[string]any `json:"attributes,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	tracer       *Tracer
	mu           sync.Mutex
}

// End finishes the span, calculates duration, and commits to the tracer recorder.
func (s *Span) End() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.EndTime != nil {
		return
	}
	now := time.Now().UTC()
	s.EndTime = &now
	s.DurationMs = s.EndTime.Sub(s.StartTime).Milliseconds()

	if s.tracer != nil {
		s.tracer.Record(s)
	}
}

// SetAttribute adds a key-value attribute tag to the span.
func (s *Span) SetAttribute(key string, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Attributes == nil {
		s.Attributes = make(map[string]any)
	}
	s.Attributes[key] = val
}

// RecordError flags the span with an error status and captures the message.
func (s *Span) RecordError(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = "ERROR"
	s.ErrorMessage = err.Error()
	if s.Attributes == nil {
		s.Attributes = make(map[string]any)
	}
	s.Attributes["error"] = err.Error()
}

// Tracer manages trace contexts and records active/completed spans (AIA-19 §8).
type Tracer struct {
	maxSpans int
	spans    []*Span
	mu       sync.RWMutex
}

var (
	defaultTracerOnce sync.Once
	defaultTracer     *Tracer
)

// DefaultTracer returns the process-wide default tracer.
func DefaultTracer() *Tracer {
	defaultTracerOnce.Do(func() {
		defaultTracer = NewTracer(1000)
	})
	return defaultTracer
}

// NewTracer creates a tracer with a bounded in-memory circular buffer.
func NewTracer(maxSpans int) *Tracer {
	if maxSpans <= 0 {
		maxSpans = 1000
	}
	return &Tracer{
		maxSpans: maxSpans,
		spans:    make([]*Span, 0, maxSpans),
	}
}

// Start creates a child span or starts a new trace if none exists in context.
func (t *Tracer) Start(ctx context.Context, name string) (context.Context, *Span) {
	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = fmt.Sprintf("trace-%s", uuid.New().String())
		ctx = WithTraceID(ctx, traceID)
	}

	spanID := uuid.New().String()
	var parentID string
	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		parentID = parentSpan.SpanID
	}

	span := &Span{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentID,
		Name:         name,
		StartTime:    time.Now().UTC(),
		Status:       "OK",
		Attributes:   make(map[string]any),
		tracer:       t,
	}

	ctx = context.WithValue(ctx, spanKey, span)
	return ctx, span
}

// Record appends a completed span into the trace buffer.
func (t *Tracer) Record(s *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.spans) >= t.maxSpans {
		// Circular drop oldest
		t.spans = t.spans[1:]
	}
	t.spans = append(t.spans, s)
}

// GetRecentSpans returns up to limit recent spans in reverse chronological order.
func (t *Tracer) GetRecentSpans(limit int) []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit <= 0 || limit > len(t.spans) {
		limit = len(t.spans)
	}

	result := make([]*Span, 0, limit)
	for i := len(t.spans) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, t.spans[i])
	}
	return result
}

// GetSpansByTrace returns all recorded spans for a specific trace ID.
func (t *Tracer) GetSpansByTrace(traceID string) []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*Span
	for _, s := range t.spans {
		if s.TraceID == traceID {
			result = append(result, s)
		}
	}
	return result
}

// WithTraceID injects a trace ID into context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext extracts active trace ID or empty string.
func TraceIDFromContext(ctx context.Context) string {
	if val := ctx.Value(traceIDKey); val != nil {
		if s, ok := val.(string); ok {
			return s
		}
	}
	if span := SpanFromContext(ctx); span != nil {
		return span.TraceID
	}
	return ""
}

// SpanFromContext retrieves the current active span from context.
func SpanFromContext(ctx context.Context) *Span {
	if val := ctx.Value(spanKey); val != nil {
		if s, ok := val.(*Span); ok {
			return s
		}
	}
	return nil
}

// StartSpan convenience helper using the default tracer.
func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	return DefaultTracer().Start(ctx, name)
}
