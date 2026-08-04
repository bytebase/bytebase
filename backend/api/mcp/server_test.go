package mcp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func TestMCPAuthMiddleware(t *testing.T) {
	secret := "test-secret-key"
	// ExternalURL short-circuits utils.GetEffectiveExternalURL away from
	// the nil store; it's also the canonical URL the WWW-Authenticate
	// resource_metadata pointer should resolve to.
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "no authorization header returns 401",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "authorization required",
		},
		{
			name:           "malformed authorization header returns 401",
			authHeader:     "NotBearer token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Bearer",
		},
		{
			name:           "invalid token returns 401",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "invalid",
		},
		{
			name:           "expired token returns 401",
			authHeader:     "Bearer " + generateExpiredToken(t, secret),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "expired",
		},
		{
			name:           "wrong audience returns 401",
			authHeader:     "Bearer " + generateTokenWithWrongAudience(t, secret),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "audience",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Create server with auth
			s, err := NewServer(nil, profile, secret)
			require.NoError(t, err)
			handler := s.authMiddleware(func(c *echo.Context) error {
				return c.String(http.StatusOK, "success")
			})

			err = handler(c)
			if err != nil {
				// Echo error handler
				echo.DefaultHTTPErrorHandler(true)(c, err)
			}

			require.Equal(t, tc.expectedStatus, rec.Code)
			require.Contains(t, strings.ToLower(rec.Body.String()), strings.ToLower(tc.expectedBody))

			// Every 401 must carry an RFC 9728 / MCP-authorization-spec
			// WWW-Authenticate header so unauthenticated clients can
			// auto-discover the authorization server.
			wwwAuth := rec.Header().Get("WWW-Authenticate")
			require.NotEmpty(t, wwwAuth, "401 response missing WWW-Authenticate header")
			require.Contains(t, wwwAuth, "Bearer")
			require.Contains(t, wwwAuth, `realm="OAuth"`)
			require.Contains(t, wwwAuth, "resource_metadata=")
			require.Contains(t, wwwAuth, `error="invalid_token"`)
			// The resource_metadata URL must (a) use the configured external
			// URL rather than the inbound request Host (proxied-deployment
			// phishing-pivot fix) and (b) include the /mcp path suffix so
			// RFC 9728 §3.3 strict clients receive metadata whose `resource`
			// field matches the URL they were accessing.
			require.Contains(t, wwwAuth, "https://bb.example.com/.well-known/oauth-protected-resource/mcp")
			// Containment for the MCP scope vocabulary: the challenge must not
			// advertise scopes while P1a persists and echoes a consented mode
			// that nothing enforces yet (the access token is still a generic
			// bearer the whole API accepts). Advertising here is what would make
			// clients request a mode and treat the echoed `scope` as a guarantee.
			// The v1 bootstrap does call for listing the full mode set in this
			// challenge — but only in the change that lands P1b enforcement.
			require.NotContains(t, wwwAuth, "scope=",
				"the challenge may advertise scopes only alongside P1b enforcement")
			require.NotContains(t, wwwAuth, "mcp:read",
				"no scope token may leak into the challenge")
		})
	}
}

func TestMCPAuthMiddlewareValidToken(t *testing.T) {
	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateValidToken(t, secret))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create server with auth - note: we pass nil store since we're testing middleware only
	// A full integration test would require a real store
	s, err := NewServer(nil, profile, secret)
	require.NoError(t, err)
	handler := s.authMiddleware(func(c *echo.Context) error {
		// Verify access token is set in request context
		ctx := c.Request().Context()
		token := getAccessToken(ctx)
		require.NotEmpty(t, token)
		return c.String(http.StatusOK, "success")
	})

	err = handler(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestMCPAuthMiddlewareOAuthContext(t *testing.T) {
	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateOAuth2MCPToken(t, secret, "client-A", "ws-test"))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s, err := NewServer(nil, profile, secret)
	require.NoError(t, err)
	handler := s.authMiddleware(func(c *echo.Context) error {
		ctx := c.Request().Context()
		require.Equal(t, "test@example.com", getUserEmail(ctx))
		require.Equal(t, "client-A", getOAuth2ClientID(ctx))
		require.Equal(t, "ws-test", getWorkspaceID(ctx))
		return c.String(http.StatusOK, "success")
	})

	err = handler(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
}

// TestMCPProxiedPublicHostNotRejected is the BYT-9693 regression. Behind a
// same-host reverse proxy (proxy_pass http://127.0.0.1:8080), the connection
// Bytebase accepts has a loopback LocalAddr while the proxy preserves the public
// Host header. The embedded MCP SDK's DNS-rebinding protection then saw
// "loopback connection + non-loopback Host" and returned 403 "Forbidden:
// invalid Host header" for legitimate, already-authenticated traffic. The
// handler must not reject such a request: the bearer token (validated by
// authMiddleware before the request reaches the handler) is the security
// boundary, not network position, so the SDK check is a false positive here.
//
// httptest.NewServer listens on 127.0.0.1, so the accepted connection's
// LocalAddr is loopback — exactly the condition the SDK keys on.
func TestMCPProxiedPublicHostNotRejected(t *testing.T) {
	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}

	s, err := NewServer(nil, profile, secret)
	require.NoError(t, err)

	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	defer ts.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+generateValidToken(t, secret))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	// The proxy preserves the public Host while the connection itself arrives
	// over loopback.
	req.Host = "bb.example.com"

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.NotEqual(t, http.StatusForbidden, resp.StatusCode,
		"authenticated request with a proxied public Host must not be rejected by DNS-rebinding protection; body=%s", string(respBody))
	require.NotContains(t, string(respBody), "invalid Host header")
}

// TestMCPUnauthenticatedRejectedEndToEnd locks in the security boundary that
// makes disabling the SDK's rebinding check safe (BYT-9693): a request without a
// valid bearer token is rejected with 401 by authMiddleware before it ever
// reaches the MCP handler, regardless of Host. Network position is not the gate;
// the token is. A DNS-rebinding attacker — who cannot obtain the token — is
// stopped here, so the disabled rebinding check protects nothing it must.
func TestMCPUnauthenticatedRejectedEndToEnd(t *testing.T) {
	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}

	s, err := NewServer(nil, profile, secret)
	require.NoError(t, err)

	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Host = "bb.example.com"

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestMCPAuthMiddlewareAudienceMatrix is the P1a PR 3 audience boundary. The
// only durably accepted audience is the live-config MCP resource URI
// (external URL + /mcp, trusted config only — never request headers). The two
// fixed legacy audiences are accepted solely inside the migration window: one
// OAuth2 access-token lifetime from process start, by which time every token
// minted by a pre-upgrade release has expired on its own.
func TestMCPAuthMiddlewareAudienceMatrix(t *testing.T) {
	secret := "test-secret-key"

	mintResourceToken := func(t *testing.T, resource string) string {
		t.Helper()
		token, err := auth.GenerateOAuth2AccessToken("test@example.com", "client-A", "ws-test", resource, secret, time.Hour)
		require.NoError(t, err)
		return token
	}

	tests := []struct {
		name string
		// externalURL is the trusted live config ("" = unconfigured).
		externalURL    string
		token          func(t *testing.T) string
		windowExpired  bool
		expectedStatus int
	}{
		{
			name:           "matching resource audience is accepted",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return mintResourceToken(t, "https://bb.example.com/mcp") },
			expectedStatus: http.StatusOK,
		},
		{
			name:           "resource audience still accepted after the migration window",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return mintResourceToken(t, "https://bb.example.com/mcp") },
			windowExpired:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:        "rotated external URL kills outstanding resource tokens",
			externalURL: "https://new.example.com",
			// The token was minted from a grant stored under the old URL; the
			// middleware compares against live config, so rotation 401s it at
			// use time and the client re-authorizes.
			token:          func(t *testing.T) string { return mintResourceToken(t, "https://old.example.com/mcp") },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "bare-origin audience is not the canonical resource",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return mintResourceToken(t, "https://bb.example.com") },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "legacy oauth2 audience accepted during the window",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return generateValidToken(t, secret) },
			expectedStatus: http.StatusOK,
		},
		{
			name:           "legacy oauth2 audience rejected after the window",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return generateValidToken(t, secret) },
			windowExpired:  true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "legacy user audience accepted during the window",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return generateUserAudienceToken(t, secret) },
			expectedStatus: http.StatusOK,
		},
		{
			name:           "legacy user audience rejected after the window",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return generateUserAudienceToken(t, secret) },
			windowExpired:  true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "unconfigured external URL fails closed for resource tokens",
			externalURL: "",
			// No trusted config means no expected audience to compare against;
			// the request-derived host must never substitute for it.
			token:          func(t *testing.T) string { return mintResourceToken(t, "https://bb.example.com/mcp") },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "unconfigured external URL still admits legacy tokens during the window",
			externalURL:    "",
			token:          func(t *testing.T) string { return generateValidToken(t, secret) },
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: tc.externalURL}
			s, err := NewServer(nil, profile, secret)
			require.NoError(t, err)
			if tc.windowExpired {
				s.legacyAudienceWindowEnd = time.Now().Add(-time.Second)
			}

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tc.token(t))
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := s.authMiddleware(func(c *echo.Context) error {
				return c.String(http.StatusOK, "success")
			})
			err = handler(c)
			if err != nil {
				echo.DefaultHTTPErrorHandler(true)(c, err)
			}

			require.Equal(t, tc.expectedStatus, rec.Code, rec.Body.String())
			if tc.expectedStatus == http.StatusUnauthorized {
				require.Contains(t, strings.ToLower(rec.Body.String()), "audience")
				// Containment: even audience-mismatch challenges must not
				// advertise the scope vocabulary before P1b enforcement.
				wwwAuth := rec.Header().Get("WWW-Authenticate")
				require.NotEmpty(t, wwwAuth)
				require.NotContains(t, wwwAuth, "scope=")
				require.NotContains(t, wwwAuth, "mcp:read")
			}
		})
	}
}

// TestDecideAudience covers the resolver branches the HTTP matrix cannot reach
// without a failing store: a deliberately unconfigured deployment falls back to
// the legacy window, while an infra failure reading the trusted config surfaces
// as an error — a server problem, never a verdict against the token.
func TestDecideAudience(t *testing.T) {
	const expected = "https://bb.example.com/mcp"
	unconfigured := connect.NewError(connect.CodeFailedPrecondition, errors.New("external URL isn't setup yet"))
	infraDown := connect.NewError(connect.CodeInternal, errors.New("failed to get workspace setting: db unreachable"))

	t.Run("matching expected audience is allowed", func(t *testing.T) {
		allowed, err := decideAudience(expected, expected, nil, false)
		require.NoError(t, err)
		require.True(t, allowed)
	})

	t.Run("mismatch with a closed window is refused", func(t *testing.T) {
		allowed, err := decideAudience("https://old.example.com/mcp", expected, nil, false)
		require.NoError(t, err)
		require.False(t, allowed)
	})

	t.Run("unconfigured external URL falls back to the legacy window", func(t *testing.T) {
		allowed, err := decideAudience(auth.OAuth2AccessTokenAudience, "", unconfigured, true)
		require.NoError(t, err)
		require.True(t, allowed)
	})

	t.Run("unconfigured external URL after the window refuses everything", func(t *testing.T) {
		allowed, err := decideAudience(auth.OAuth2AccessTokenAudience, "", unconfigured, false)
		require.NoError(t, err)
		require.False(t, allowed)
	})

	t.Run("infra failure is reported as an error, not a token verdict", func(t *testing.T) {
		allowed, err := decideAudience(expected, "", infraDown, true)
		require.Error(t, err)
		require.False(t, allowed)
	})
}

func generateUserAudienceToken(t *testing.T, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": "bytebase",
		"sub": "test@example.com",
		"aud": auth.AccessTokenAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "v1"
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

func generateValidToken(t *testing.T, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": "bytebase",
		"sub": "test@example.com",
		"aud": auth.OAuth2AccessTokenAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "v1"
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

func generateOAuth2MCPToken(t *testing.T, secret, clientID, workspaceID string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":          "bytebase",
		"sub":          "test@example.com",
		"aud":          auth.OAuth2AccessTokenAudience,
		"exp":          time.Now().Add(time.Hour).Unix(),
		"iat":          time.Now().Unix(),
		"client_id":    clientID,
		"workspace_id": workspaceID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "v1"
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

func generateExpiredToken(t *testing.T, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": "bytebase",
		"sub": "test@example.com",
		"aud": auth.OAuth2AccessTokenAudience,
		"exp": time.Now().Add(-time.Hour).Unix(), // expired
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "v1"
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

func generateTokenWithWrongAudience(t *testing.T, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": "bytebase",
		"sub": "test@example.com",
		"aud": "wrong.audience",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "v1"
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}
