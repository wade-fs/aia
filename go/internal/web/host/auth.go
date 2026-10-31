package host

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// Session represents an authenticated user browser session.
type Session struct {
	ID        string
	ActorID   string
	CreatedAt time.Time
	LastSeen  time.Time
}

// AuthManager manages one-time tokens and active server-side sessions.
type AuthManager struct {
	mu           sync.RWMutex
	tokens       map[string]tokenEntry
	sessions     map[string]*Session
	tokenTTL     time.Duration
	idleTTL      time.Duration
	absoluteTTL  time.Duration
	cookieSecure bool
}

type tokenEntry struct {
	token     string
	createdAt time.Time
	expiresAt time.Time
	used      bool
}

func NewAuthManager(cookieSecure bool) *AuthManager {
	return &AuthManager{
		tokens:       make(map[string]tokenEntry),
		sessions:     make(map[string]*Session),
		tokenTTL:     15 * time.Minute,
		idleTTL:      30 * time.Minute,
		absoluteTTL:  12 * time.Hour,
		cookieSecure: cookieSecure,
	}
}

// GenerateOneTimeToken produces a cryptographically secure, high-entropy one-time login token.
func (m *AuthManager) GenerateOneTimeToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(bytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	m.tokens[tok] = tokenEntry{
		token:     tok,
		createdAt: now,
		expiresAt: now.Add(m.tokenTTL),
		used:      false,
	}

	return tok, nil
}

// RedeemOneTimeToken verifies and immediately burns the token, returning a new Session.
func (m *AuthManager) RedeemOneTimeToken(tokenStr string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.tokens[tokenStr]
	if !ok || entry.used {
		return nil, ErrInvalidToken
	}

	now := time.Now().UTC()
	if now.After(entry.expiresAt) {
		delete(m.tokens, tokenStr)
		return nil, ErrExpiredToken
	}

	// Invalidate immediately (single-use)
	delete(m.tokens, tokenStr)

	// Create server-side session
	sessBytes := make([]byte, 32)
	if _, err := rand.Read(sessBytes); err != nil {
		return nil, err
	}
	sessID := hex.EncodeToString(sessBytes)

	sess := &Session{
		ID:        sessID,
		ActorID:   "admin",
		CreatedAt: now,
		LastSeen:  now,
	}
	m.sessions[sessID] = sess

	return sess, nil
}

// ValidateSession checks idle and absolute expirations.
func (m *AuthManager) ValidateSession(sessID string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[sessID]
	if !ok {
		return nil, false
	}

	now := time.Now().UTC()
	// Check absolute timeout
	if now.Sub(sess.CreatedAt) > m.absoluteTTL {
		delete(m.sessions, sessID)
		return nil, false
	}
	// Check idle timeout
	if now.Sub(sess.LastSeen) > m.idleTTL {
		delete(m.sessions, sessID)
		return nil, false
	}

	sess.LastSeen = now
	return sess, true
}

// InvalidateSession terminates a session (logout).
func (m *AuthManager) InvalidateSession(sessID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessID)
}

// SetSessionCookie sets a secure, HttpOnly, SameSite=Strict cookie on w.
func (m *AuthManager) SetSessionCookie(w http.ResponseWriter, sessID string) {
	cookie := &http.Cookie{
		Name:     "aia_session",
		Value:    sessID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   m.cookieSecure,
		MaxAge:   int(m.absoluteTTL.Seconds()),
	}
	http.SetCookie(w, cookie)
}

// ClearSessionCookie clears the cookie on logout.
func (m *AuthManager) ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "aia_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   m.cookieSecure,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}
