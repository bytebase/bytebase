package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

// runAuthMiddleware pushes one authenticated request through authMiddleware and
// returns the delegated credential the next handler observed in context.
func runAuthMiddleware(t *testing.T, s *Server, bearer string) (credential string, status int) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := s.authMiddleware(func(c *echo.Context) error {
		credential = getDelegatedCredential(c.Request().Context())
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
// EMPTY grant state — no scope, no resource. PR 5, not this PR, assigns their
// synthetic LEGACY_FULL semantics.
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
