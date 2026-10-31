package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/audit"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type contextKey string

const (
	reqIDKey  contextKey = "request_id"
	actorKey  contextKey = "actor_id"
	sessIDKey contextKey = "session_id"
)

// Standard API response structures
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      any         `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	RequestID string      `json:"request_id"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data any, reqID string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success:   status >= 200 && status < 300,
		Data:      data,
		RequestID: reqID,
	})
}

func WriteError(w http.ResponseWriter, status int, code, message string, reqID string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
		RequestID: reqID,
	})
}

// RequestIDMiddleware injects unique request_id and sets up the audit origin.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = "req_" + uuid.New().String()
		}
		w.Header().Set("X-Request-ID", reqID)

		ctx := context.WithValue(r.Context(), reqIDKey, reqID)
		ctx = audit.WithOrigin(ctx, audit.Origin{
			RequestID: reqID,
			Source:    audit.SourceWeb,
			ActorType: "USER",
			ActorID:   "anonymous",
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CSRFProtection verifies origin/referer for state-changing HTTP methods.
func CSRFProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch {
			origin := r.Header.Get("Origin")
			if origin != "" {
				u, err := url.Parse(origin)
				if err != nil || (u.Host != r.Host && !strings.HasPrefix(u.Host, "localhost:") && !strings.HasPrefix(u.Host, "127.0.0.1:")) {
					reqID, _ := r.Context().Value(reqIDKey).(string)
					WriteError(w, http.StatusForbidden, "CSRF_ORIGIN_INVALID", "Invalid or forbidden request Origin", reqID)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware enforces valid server-side session cookies on protected API endpoints.
func (h *Host) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID, _ := r.Context().Value(reqIDKey).(string)
		path := r.URL.Path

		// Allow public API endpoints
		if path == "/api/v1/auth/login" || path == "/api/v1/system/health" || path == "/api/v1/system/activity" || !strings.HasPrefix(path, "/api/v1") {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("aia_session")
		if err != nil || cookie.Value == "" {
			WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required. Please log in with a one-time token.", reqID)
			return
		}

		sess, ok := h.AuthMgr.ValidateSession(cookie.Value)
		if !ok {
			h.AuthMgr.ClearSessionCookie(w)
			WriteError(w, http.StatusUnauthorized, "SESSION_EXPIRED", "Session expired or invalid. Please log in again.", reqID)
			return
		}

		ctx := context.WithValue(r.Context(), actorKey, sess.ActorID)
		ctx = context.WithValue(ctx, sessIDKey, sess.ID)
		ctx = audit.WithOrigin(ctx, audit.Origin{
			RequestID: reqID,
			Source:    audit.SourceWeb,
			ActorType: "USER",
			ActorID:   sess.ActorID,
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuditMiddleware logs modifying operations to the immutable audit repository.
func (h *Host) AuditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Only log API operations
		if !strings.HasPrefix(r.URL.Path, "/api/v1") || r.URL.Path == "/api/v1/auth/login" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now().UTC()
		next.ServeHTTP(w, r)

		if h.AuditRepo != nil {
			reqID, _ := r.Context().Value(reqIDKey).(string)
			actorID, _ := r.Context().Value(actorKey).(string)
			if actorID == "" {
				actorID = "web-user"
			}

			meta := map[string]any{
				"method":   r.Method,
				"path":     r.URL.Path,
				"duration": time.Since(start).String(),
			}

			_ = h.AuditRepo.Append(r.Context(), &repository.AuditLogEntry{
				ID:           uuid.New().String(),
				ActorType:    "USER",
				ActorID:      actorID,
				Action:       fmt.Sprintf("HTTP_%s", r.Method),
				ResourceType: "WEB_API",
				ResourceID:   r.URL.Path,
				Outcome:      "SUCCESS",
				RequestID:    reqID,
				Source:       string(audit.SourceWeb),
				Metadata:     audit.SanitizeMetadata(meta),
				CreatedAt:    start,
			})
		}
	})
}
