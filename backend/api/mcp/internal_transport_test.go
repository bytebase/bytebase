package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

// runAuthMiddleware pushes one authenticated request through authMiddleware and
// returns a credential minted from the delegated identity the next handler
// observed in context — the same thing apiRequest does on every internal call.
func runAuthMiddleware(t *testing.T, s *Server, bearer string) (credential string, status int) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := s.authMiddleware(func(c *echo.Context) error {
		identity, ok := getDelegatedIdentity(c.Request().Context())
		if ok {
			minted, err := auth.GenerateInternalMCPToken(identity, s.secret)
			require.NoError(t, err)
			credential = minted
		}
		return c.String(http.StatusOK, "success")
	})
	if err := handler(c); err != nil {
		echo.DefaultHTTPErrorHandler(true)(c, err)
	}
	return credential, rec.Code
}

// TestMCPAuthMiddlewareMintsDelegatedCredential pins the PR 4 boundary
// behavior: every admitted /mcp request mints a fresh internal credential
// carrying the principal, workspace, client_id, a correlation ID, and the
// grant's stored scope + resource copied verbatim from the inbound token. The
// inbound bearer is replaced, not wrapped.
func TestMCPAuthMiddlewareMintsDelegatedCredential(t *testing.T) {
	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := NewServer(nil, profile, secret, nil)
	require.NoError(t, err)

	inbound, err := auth.GenerateOAuth2AccessToken("test@example.com", "client-A", "ws-test", "https://bb.example.com/mcp", "mcp:read-only", secret, time.Hour)
	require.NoError(t, err)

	credential, status := runAuthMiddleware(t, s, inbound)
	require.Equal(t, http.StatusOK, status)
	require.NotEmpty(t, credential, "an admitted request must mint a delegated credential")
	require.NotEqual(t, inbound, credential, "the credential replaces the inbound bearer, never wraps or reuses it")

	cred, err := auth.VerifyInternalMCPToken(credential, secret)
	require.NoError(t, err)
	require.Equal(t, "test@example.com", cred.Principal)
	require.Equal(t, "ws-test", cred.WorkspaceID)
	require.Equal(t, "client-A", cred.ClientID)
	require.NotEmpty(t, cred.CorrelationID)
	require.Equal(t, "mcp:read-only", cred.Scope, "the grant's stored scope travels verbatim")
	require.Equal(t, "https://bb.example.com/mcp", cred.Resource, "the resource-bound audience is the grant's stored resource")

	// Minting is per MCP request: a second request gets its own credential and
	// correlation ID.
	second, status := runAuthMiddleware(t, s, inbound)
	require.Equal(t, http.StatusOK, status)
	secondCred, err := auth.VerifyInternalMCPToken(second, secret)
	require.NoError(t, err)
	require.NotEqual(t, cred.CorrelationID, secondCred.CorrelationID)
}

// TestMCPAuthMiddlewareLegacySessionEmptyGrantState pins that legacy sessions
// keep working through tools: a plain bb.user.access web token and a legacy
// bb.oauth2.access token both mint a credential from their principal with
// EMPTY grant state — no scope, no resource. PR 5 carries that state verbatim
// into common.AuthContext; P1b resolves it (common.DelegatedGrant).
func TestMCPAuthMiddlewareLegacySessionEmptyGrantState(t *testing.T) {
	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := NewServer(nil, profile, secret, nil)
	require.NoError(t, err)

	for name, bearer := range map[string]string{
		"plain user session token":  generateUserAudienceToken(t, secret),
		"legacy oauth2 mcp token":   generateValidToken(t, secret),
		"legacy oauth2 with client": generateOAuth2MCPToken(t, secret, "client-A", "ws-test"),
	} {
		t.Run(name, func(t *testing.T) {
			credential, status := runAuthMiddleware(t, s, bearer)
			require.Equal(t, http.StatusOK, status)
			require.NotEmpty(t, credential)

			cred, err := auth.VerifyInternalMCPToken(credential, secret)
			require.NoError(t, err)
			require.Equal(t, "test@example.com", cred.Principal)
			require.Empty(t, cred.Scope, "legacy sessions have no stored grant scope")
			require.Empty(t, cred.Resource, "a legacy fixed audience is not a grant resource")
			require.NotEmpty(t, cred.CorrelationID)
		})
	}
}

// TestApiRequestMintsCredentialPerCall is the session-lifetime pin. The MCP SDK
// hands every tool handler the context of the request that CREATED the session
// (go-sdk v1.6.1 streamable.go: server.Connect(req.Context(), ...)), so anything
// the boundary stashes in that context is frozen at session start. A credential
// minted there would carry a fixed expiry and brick every tool call once its
// short TTL elapsed — while the session stays open, since /mcp never re-runs
// the middleware for it. Minting inside apiRequest instead keeps the TTL
// genuinely request-scale: each internal call gets a freshly minted credential.
func TestApiRequestMintsCredentialPerCall(t *testing.T) {
	const secret = "test-secret-key"
	var credentials []string
	s := newTestServerWithMock(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		credentials = append(credentials, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	s.secret = secret

	// The context a tool handler sees: the session's, established once.
	ctx := withDelegatedIdentity(context.Background(), auth.DelegatedMCPCredential{
		Principal:     "test@example.com",
		WorkspaceID:   "ws-test",
		CorrelationID: "corr-1",
	})

	_, err := s.apiRequest(ctx, "/api/test", nil)
	require.NoError(t, err)
	// JWT expiry has one-second resolution, so separate the calls enough that a
	// call-time mint is observably later than a session-time one.
	time.Sleep(1100 * time.Millisecond)
	_, err = s.apiRequest(ctx, "/api/test", nil)
	require.NoError(t, err)

	require.Len(t, credentials, 2)
	first, _, err := jwt.NewParser().ParseUnverified(credentials[0], jwt.MapClaims{})
	require.NoError(t, err)
	second, _, err := jwt.NewParser().ParseUnverified(credentials[1], jwt.MapClaims{})
	require.NoError(t, err)
	firstExp, err := first.Claims.GetExpirationTime()
	require.NoError(t, err)
	secondExp, err := second.Claims.GetExpirationTime()
	require.NoError(t, err)
	require.True(t, secondExp.After(firstExp.Time),
		"each internal call must mint a fresh credential; a session-frozen one expires mid-session")

	// Both must still be valid credentials carrying the session's identity.
	for _, credential := range credentials {
		cred, err := auth.VerifyInternalMCPToken(credential, secret)
		require.NoError(t, err)
		require.Equal(t, "test@example.com", cred.Principal)
		require.Equal(t, "corr-1", cred.CorrelationID, "the boundary's correlation ID travels on every call")
	}
}

// TestInternalRequestCarriesCallerIP pins that audit keeps seeing who made an
// MCP-originated call. The audit interceptor derives caller IP from
// X-Real-IP -> X-Forwarded-For -> peer address; on an in-memory request all
// three are empty unless the boundary carries the inbound caller's IP forward,
// so audit rows for MCP actions would record no origin at all. (The retired
// loopback transport did no better: it recorded the 127.0.0.1 socket, never the
// real client.)
func TestInternalRequestCarriesCallerIP(t *testing.T) {
	const secret = "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		want       string
	}{
		{
			name:       "X-Real-IP wins",
			headers:    map[string]string{"X-Real-IP": "203.0.113.7", "X-Forwarded-For": "198.51.100.9"},
			remoteAddr: "10.0.0.1:4444",
			want:       "203.0.113.7",
		},
		{
			name:       "X-Forwarded-For is next",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.9"},
			remoteAddr: "10.0.0.1:4444",
			want:       "198.51.100.9",
		},
		{
			name:       "peer address is the fallback, without its port",
			remoteAddr: "10.0.0.1:4444",
			want:       "10.0.0.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedIP string
			stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedIP = r.Header.Get("X-Real-IP")
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{}`)
			})
			s, err := NewServer(nil, profile, secret, stub)
			require.NoError(t, err)

			inbound, err := auth.GenerateOAuth2AccessToken("test@example.com", "client-A", "ws-test", "https://bb.example.com/mcp", "mcp:read-only", secret, time.Hour)
			require.NoError(t, err)

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
			req.Header.Set("Authorization", "Bearer "+inbound)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tc.remoteAddr
			c := e.NewContext(req, httptest.NewRecorder())

			handler := s.authMiddleware(func(c *echo.Context) error {
				_, err := s.apiRequest(c.Request().Context(), "/api/test", nil)
				return err
			})
			require.NoError(t, handler(c))
			require.Equal(t, tc.want, capturedIP)
		})
	}
}

// TestMCPAuthMiddlewareRejectsInternalCredential is the /mcp half of the hard
// boundary: the public /mcp listener must never accept the internal credential
// it mints for the private transport. A leaked credential is useless here.
func TestMCPAuthMiddlewareRejectsInternalCredential(t *testing.T) {
	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := NewServer(nil, profile, secret, nil)
	require.NoError(t, err)

	internal, err := auth.GenerateInternalMCPToken(auth.DelegatedMCPCredential{
		Principal:   "test@example.com",
		WorkspaceID: "ws-test",
	}, secret)
	require.NoError(t, err)

	credential, status := runAuthMiddleware(t, s, internal)
	require.Equal(t, http.StatusUnauthorized, status)
	require.Empty(t, credential, "the handler must not run for an internal credential")
}
