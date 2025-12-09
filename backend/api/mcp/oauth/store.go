package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// AuthorizationCode represents a pending OAuth authorization.
type AuthorizationCode struct {
	Code                string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	Scopes              []string
	UserID              int // Bytebase user ID after IdP auth
	UserEmail           string
	ExpiresAt           time.Time
	IDPName             string // Which IdP was used
	IDPContext          string // IdP-specific context (e.g., state)
}

// AuthCodeStore manages authorization codes in memory.
// For production, consider using Redis or database storage.
type AuthCodeStore struct {
	mu    sync.RWMutex
	codes map[string]*AuthorizationCode
}

// NewAuthCodeStore creates a new authorization code store.
func NewAuthCodeStore() *AuthCodeStore {
	store := &AuthCodeStore{
		codes: make(map[string]*AuthorizationCode),
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

// GenerateCode creates a new authorization code.
func (*AuthCodeStore) GenerateCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Store saves an authorization code.
func (s *AuthCodeStore) Store(code *AuthorizationCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code.Code] = code
}

// Consume retrieves and deletes an authorization code (one-time use).
func (s *AuthCodeStore) Consume(code string) (*AuthorizationCode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	authCode, ok := s.codes[code]
	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(authCode.ExpiresAt) {
		delete(s.codes, code)
		return nil, false
	}

	// Delete after retrieval (one-time use)
	delete(s.codes, code)
	return authCode, true
}

// StorePending stores a pending authorization (before IdP callback).
func (s *AuthCodeStore) StorePending(state string, code *AuthorizationCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Use state as temporary key
	s.codes["pending:"+state] = code
}

// ConsumePending retrieves pending authorization by state.
func (s *AuthCodeStore) ConsumePending(state string) (*AuthorizationCode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	authCode, ok := s.codes["pending:"+state]
	if !ok {
		return nil, false
	}

	delete(s.codes, "pending:"+state)
	return authCode, true
}

// cleanup removes expired codes periodically.
func (s *AuthCodeStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for code, authCode := range s.codes {
			if now.After(authCode.ExpiresAt) {
				delete(s.codes, code)
			}
		}
		s.mu.Unlock()
	}
}
