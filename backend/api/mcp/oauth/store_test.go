package oauth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenerateCode(t *testing.T) {
	a := require.New(t)
	store := NewAuthCodeStore()

	// Generate multiple codes
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := store.GenerateCode()
		a.NoError(err)
		a.NotEmpty(code)

		// Verify uniqueness
		a.False(codes[code], "generated duplicate code: %s", code)
		codes[code] = true

		// Verify length (32 bytes base64url encoded = 43 characters)
		a.Equal(43, len(code))
	}
}

func TestStoreAndConsume(t *testing.T) {
	a := require.New(t)
	store := NewAuthCodeStore()

	authCode := &AuthorizationCode{
		Code:                "test-code-123",
		ClientID:            "client-id",
		RedirectURI:         "http://localhost:8080/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		Scopes:              []string{"read", "write"},
		UserID:              1,
		UserEmail:           "test@example.com",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		IDPName:             "google",
		IDPContext:          "context",
	}

	// Store the code
	store.Store(authCode)

	// Consume the code (should succeed)
	retrieved, ok := store.Consume("test-code-123")
	a.True(ok)
	a.NotNil(retrieved)
	a.Equal(authCode.Code, retrieved.Code)
	a.Equal(authCode.ClientID, retrieved.ClientID)
	a.Equal(authCode.RedirectURI, retrieved.RedirectURI)
	a.Equal(authCode.CodeChallenge, retrieved.CodeChallenge)
	a.Equal(authCode.UserID, retrieved.UserID)

	// Try to consume again (should fail - one-time use)
	retrieved, ok = store.Consume("test-code-123")
	a.False(ok)
	a.Nil(retrieved)
}

func TestConsumeNonexistent(t *testing.T) {
	a := require.New(t)
	store := NewAuthCodeStore()

	// Try to consume a code that doesn't exist
	retrieved, ok := store.Consume("nonexistent-code")
	a.False(ok)
	a.Nil(retrieved)
}

func TestConsumeExpired(t *testing.T) {
	a := require.New(t)
	store := NewAuthCodeStore()

	// Store a code that's already expired
	authCode := &AuthorizationCode{
		Code:                "expired-code",
		ClientID:            "client-id",
		RedirectURI:         "http://localhost:8080/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		Scopes:              []string{"read"},
		UserID:              1,
		UserEmail:           "test@example.com",
		ExpiresAt:           time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		IDPName:             "google",
		IDPContext:          "context",
	}

	store.Store(authCode)

	// Try to consume expired code (should fail)
	retrieved, ok := store.Consume("expired-code")
	a.False(ok)
	a.Nil(retrieved)

	// Verify it was deleted from store
	retrieved, ok = store.Consume("expired-code")
	a.False(ok)
	a.Nil(retrieved)
}

func TestStorePendingAndConsumePending(t *testing.T) {
	a := require.New(t)
	store := NewAuthCodeStore()

	authCode := &AuthorizationCode{
		Code:                "", // Not yet assigned
		ClientID:            "client-id",
		RedirectURI:         "http://localhost:8080/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		Scopes:              []string{"read", "write"},
		UserID:              0, // Not yet authenticated
		UserEmail:           "",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		IDPName:             "google",
		IDPContext:          "idp-state-123",
	}

	state := "oauth-state-456"

	// Store pending authorization
	store.StorePending(state, authCode)

	// Consume pending authorization
	retrieved, ok := store.ConsumePending(state)
	a.True(ok)
	a.NotNil(retrieved)
	a.Equal(authCode.ClientID, retrieved.ClientID)
	a.Equal(authCode.CodeChallenge, retrieved.CodeChallenge)
	a.Equal(authCode.IDPContext, retrieved.IDPContext)

	// Try to consume again (should fail - one-time use)
	retrieved, ok = store.ConsumePending(state)
	a.False(ok)
	a.Nil(retrieved)
}

func TestConsumePendingNonexistent(t *testing.T) {
	a := require.New(t)
	store := NewAuthCodeStore()

	// Try to consume pending that doesn't exist
	retrieved, ok := store.ConsumePending("nonexistent-state")
	a.False(ok)
	a.Nil(retrieved)
}

func TestMultipleCodes(t *testing.T) {
	a := require.New(t)
	store := NewAuthCodeStore()

	// Store multiple codes
	codes := []*AuthorizationCode{
		{
			Code:                "code-1",
			ClientID:            "client-1",
			RedirectURI:         "http://localhost:8080/callback1",
			CodeChallenge:       "challenge1",
			CodeChallengeMethod: "S256",
			Scopes:              []string{"read"},
			UserID:              1,
			UserEmail:           "user1@example.com",
			ExpiresAt:           time.Now().Add(10 * time.Minute),
			IDPName:             "google",
		},
		{
			Code:                "code-2",
			ClientID:            "client-2",
			RedirectURI:         "http://localhost:8080/callback2",
			CodeChallenge:       "challenge2",
			CodeChallengeMethod: "S256",
			Scopes:              []string{"write"},
			UserID:              2,
			UserEmail:           "user2@example.com",
			ExpiresAt:           time.Now().Add(10 * time.Minute),
			IDPName:             "github",
		},
		{
			Code:                "code-3",
			ClientID:            "client-3",
			RedirectURI:         "http://localhost:8080/callback3",
			CodeChallenge:       "challenge3",
			CodeChallengeMethod: "S256",
			Scopes:              []string{"read", "write"},
			UserID:              3,
			UserEmail:           "user3@example.com",
			ExpiresAt:           time.Now().Add(10 * time.Minute),
			IDPName:             "oauth2",
		},
	}

	for _, code := range codes {
		store.Store(code)
	}

	// Consume codes in different order
	retrieved2, ok := store.Consume("code-2")
	a.True(ok)
	a.Equal("client-2", retrieved2.ClientID)

	retrieved1, ok := store.Consume("code-1")
	a.True(ok)
	a.Equal("client-1", retrieved1.ClientID)

	retrieved3, ok := store.Consume("code-3")
	a.True(ok)
	a.Equal("client-3", retrieved3.ClientID)

	// All should be consumed now
	_, ok = store.Consume("code-1")
	a.False(ok)
	_, ok = store.Consume("code-2")
	a.False(ok)
	_, ok = store.Consume("code-3")
	a.False(ok)
}

func TestPendingAndRegularCodesSeparate(t *testing.T) {
	a := require.New(t)
	store := NewAuthCodeStore()

	// Store regular code
	regularCode := &AuthorizationCode{
		Code:                "regular-code",
		ClientID:            "client-id",
		RedirectURI:         "http://localhost:8080/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		Scopes:              []string{"read"},
		UserID:              1,
		UserEmail:           "test@example.com",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		IDPName:             "google",
	}
	store.Store(regularCode)

	// Store pending code with same actual code value
	pendingCode := &AuthorizationCode{
		Code:                "regular-code", // Same code value
		ClientID:            "pending-client",
		RedirectURI:         "http://localhost:8080/callback",
		CodeChallenge:       "pending-challenge",
		CodeChallengeMethod: "S256",
		Scopes:              []string{"write"},
		UserID:              0,
		UserEmail:           "",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		IDPName:             "github",
	}
	store.StorePending("pending-state", pendingCode)

	// Regular consume should get regular code
	retrieved, ok := store.Consume("regular-code")
	a.True(ok)
	a.Equal("client-id", retrieved.ClientID)

	// Pending consume should still work
	retrieved, ok = store.ConsumePending("pending-state")
	a.True(ok)
	a.Equal("pending-client", retrieved.ClientID)
}
