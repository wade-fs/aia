package clock

import (
	"context"
	"time"
)

type clockKey struct{}

// State holds the simulated Point-in-Time (AsOf) clock and zero-look-ahead replay flag (AIA-13 §21.2).
type State struct {
	AsOf       time.Time
	ReplayMode bool
}

// WithAsOf binds a deterministic Point-in-Time cutoff timestamp to ctx.
// When replayMode is true, downstream observers, repositories, and services MUST:
//  1. Filter all temporal queries strictly to <= asOf (preventing look-ahead bias).
//  2. Block live external HTTP fetches so historical replay never leaks present-day prices/news.
func WithAsOf(ctx context.Context, asOf time.Time, replayMode bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if asOf.IsZero() {
		return ctx
	}
	return context.WithValue(ctx, clockKey{}, State{
		AsOf:       asOf.UTC(),
		ReplayMode: replayMode,
	})
}

// Now returns the active Point-in-Time timestamp if bound in ctx, or time.Now().UTC() otherwise.
func Now(ctx context.Context) time.Time {
	if ctx != nil {
		if st, ok := ctx.Value(clockKey{}).(State); ok && !st.AsOf.IsZero() {
			return st.AsOf
		}
	}
	return time.Now().UTC()
}

// AsOf returns (*asOf, true) when a Point-in-Time cutoff is active on ctx.
func AsOf(ctx context.Context) (time.Time, bool) {
	if ctx != nil {
		if st, ok := ctx.Value(clockKey{}).(State); ok && !st.AsOf.IsZero() {
			return st.AsOf, true
		}
	}
	return time.Time{}, false
}

// IsReplay returns true when ctx is running inside a zero-look-ahead historical replay.
func IsReplay(ctx context.Context) bool {
	if ctx != nil {
		if st, ok := ctx.Value(clockKey{}).(State); ok {
			return st.ReplayMode
		}
	}
	return false
}
