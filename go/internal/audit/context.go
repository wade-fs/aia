// Package audit provides request-scoped audit metadata propagation.
//
// Entry points (CLI, Web, Runtime, Recovery, Agent) attach an Origin to the
// context; the audit repository reads it so that every audit record carries
// request_id, source and actor without every service having to thread the
// values explicitly (AIA-06 §8).
package audit

import (
	"context"
	"encoding/json"
	"strings"
)

// Source identifies which entry point produced an audited operation.
type Source string

const (
	SourceCLI      Source = "CLI"
	SourceWeb      Source = "WEB"
	SourceAPI      Source = "API"
	SourceRuntime  Source = "RUNTIME"
	SourceAgent    Source = "AGENT"
	SourceRecovery Source = "RECOVERY"
)

// Origin is the request-scoped audit metadata.
type Origin struct {
	RequestID string
	Source    Source
	ActorType string
	ActorID   string
}

type originKey struct{}

// WithOrigin attaches audit origin metadata to ctx.
func WithOrigin(ctx context.Context, o Origin) context.Context {
	return context.WithValue(ctx, originKey{}, o)
}

// FromContext returns the audit origin stored in ctx, if any.
func FromContext(ctx context.Context) (Origin, bool) {
	if ctx == nil {
		return Origin{}, false
	}
	o, ok := ctx.Value(originKey{}).(Origin)
	return o, ok
}

// sensitiveKeyFragments lists metadata key fragments that must never be
// persisted to the audit log (tokens, cookies, credentials).
var sensitiveKeyFragments = []string{
	"token", "cookie", "password", "passwd", "secret", "api_key", "apikey",
	"authorization", "credential", "session", "csrf", "private_key",
}

// IsSensitiveKey reports whether a metadata key looks like it carries a secret.
func IsSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, f := range sensitiveKeyFragments {
		if strings.Contains(k, f) {
			return true
		}
	}
	return false
}

// SanitizeMetadata removes sensitive keys (recursively) from a metadata map and
// returns its JSON encoding. Nil or empty input yields "{}".
func SanitizeMetadata(meta map[string]any) json.RawMessage {
	if len(meta) == 0 {
		return json.RawMessage("{}")
	}
	clean := sanitizeMap(meta)
	b, err := json.Marshal(clean)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}

// SanitizeRaw parses raw JSON metadata and strips sensitive keys.
func SanitizeRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return json.RawMessage("{}")
	}
	return SanitizeMetadata(m)
}

func sanitizeMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if IsSensitiveKey(k) {
			out[k] = "[REDACTED]"
			continue
		}
		switch t := v.(type) {
		case map[string]any:
			out[k] = sanitizeMap(t)
		default:
			out[k] = v
		}
	}
	return out
}
