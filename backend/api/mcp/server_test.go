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
	// the nil store, and the minted tokens carry a workspace so no singleton
	// workspace lookup runs either; it's also the canonical URL the
	// WWW-Authenticate resource_metadata pointer should resolve to.
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
			s, err := newServerWithStore(newTestServerStore(), profile, secret, nil)
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

func TestNewServerRequiresStore(t *testing.T) {
	_, err := NewServer(nil, &config.Profile{}, "test-secret", nil)
	require.Error(t, err)
}

func TestMCPAuthFailsExplicitlyWithoutExternalURL(t *testing.T) {
	s := &Server{
		store:   newTestServerStore(),
		profile: &config.Profile{SaaS: true},
	}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := s.authMiddleware(func(*echo.Context) error { return nil })(c)
	require.Error(t, err)
	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusServiceUnavailable, httpErr.Code)
}

func TestExtractTokenIdentityRequiresWorkspace(t *testing.T) {
	identity, errMsg := extractTokenIdentity(jwt.MapClaims{
		"sub": "test@example.com",
		"aud": "https://bb.example.com/mcp",
	})

	require.Nil(t, identity)
	require.Equal(t, "invalid token: missing workspace", errMsg)
}

func TestMCPAuthMiddlewareValidToken(t *testing.T) {
	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateValidToken(t, secret))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Use a lightweight store double; production requires the real dependency.
	s, err := newServerWithStore(newTestServerStore(), profile, secret, nil)
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
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateOAuth2MCPToken(t, secret, "client-A", "ws-test"))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s, err := newServerWithStore(newTestServerStore(), profile, secret, nil)
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

	s, err := newServerWithStore(newTestServerStore(), profile, secret, nil)
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

	s, err := newServerWithStore(newTestServerStore(), profile, secret, nil)
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
// durably accepted audience is the live-config MCP resource URI (external URL
// + /mcp, trusted config only — never request headers). Legacy bb.oauth2.access
// tokens are accepted while they remain unexpired: this release never mints
// that audience, so acceptance drains no later than one access-token lifetime
// after the last legacy-capable replica leaves service (the expired-token 401
// in TestMCPAuthMiddleware is the other half of that invariant). Plain
// bb.user.access session tokens are accepted until the scoped service-account
// flow lands (spec "Deferred product decisions"; proposal §6.5).
func TestMCPAuthMiddlewareAudienceMatrix(t *testing.T) {
	secret := "test-secret-key"

	mintResourceToken := func(t *testing.T, resource string) string {
		t.Helper()
		token, err := auth.GenerateOAuth2AccessToken("test@example.com", "client-A", "ws-test", resource, "", secret, time.Hour)
		require.NoError(t, err)
		return token
	}

	tests := []struct {
		name string
		// externalURL is the trusted live config. The unconfigured ("") case
		// needs a store to resolve, so it lives in TestDecideAudience.
		externalURL    string
		token          func(t *testing.T) string
		expectedStatus int
	}{
		{
			name:           "matching resource audience is accepted",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return mintResourceToken(t, "https://bb.example.com/mcp") },
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
			// Unexpired legacy tokens (minted by a pre-upgrade replica, which
			// during a rolling deploy can outlive any single process's start)
			// are accepted; their own exp claim is what retires them.
			name:           "unexpired legacy oauth2 audience is accepted",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return generateValidToken(t, secret) },
			expectedStatus: http.StatusOK,
		},
		{
			// bb.user.access retirement is gated on the scoped service-account
			// flow landing first (PR 5/6).
			name:           "plain user audience is accepted",
			externalURL:    "https://bb.example.com",
			token:          func(t *testing.T) string { return generateUserAudienceToken(t, secret) },
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: tc.externalURL}
			s, err := newServerWithStore(newTestServerStore(), profile, secret, nil)
			require.NoError(t, err)

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
// without a DB-backed store: a deliberately unconfigured deployment admits only
// the legacy audiences (resource-bound tokens fail closed), while an infra
// failure reading the trusted config surfaces as an error — a server problem,
// never a verdict against the token.
func TestDecideAudience(t *testing.T) {
	const expected = "https://bb.example.com/mcp"
	unconfigured := connect.NewError(connect.CodeFailedPrecondition, errors.New("external URL isn't setup yet"))
	infraDown := connect.NewError(connect.CodeInternal, errors.New("failed to get workspace setting: db unreachable"))

	t.Run("matching expected audience is allowed", func(t *testing.T) {
		allowed, err := decideAudience(expected, expected, nil)
		require.NoError(t, err)
		require.True(t, allowed)
	})

	t.Run("unknown audience is refused", func(t *testing.T) {
		allowed, err := decideAudience("https://old.example.com/mcp", expected, nil)
		require.NoError(t, err)
		require.False(t, allowed)
	})

	t.Run("legacy oauth2 audience is accepted while its token lives", func(t *testing.T) {
		// No clock gate: this release never mints bb.oauth2.access, so the
		// population drains by each token's own exp — no later than one
		// access-token lifetime after the last legacy-capable replica leaves
		// service. A process-start window would race rolling deploys.
		allowed, err := decideAudience(auth.OAuth2AccessTokenAudience, expected, nil)
		require.NoError(t, err)
		require.True(t, allowed)
	})

	t.Run("unconfigured external URL still admits legacy audiences", func(t *testing.T) {
		allowed, err := decideAudience(auth.OAuth2AccessTokenAudience, "", unconfigured)
		require.NoError(t, err)
		require.True(t, allowed)
	})

	t.Run("unconfigured external URL fails closed for resource tokens", func(t *testing.T) {
		// No trusted config means no expected audience to compare against; the
		// request-derived host must never substitute for it.
		allowed, err := decideAudience("https://bb.example.com/mcp", "", unconfigured)
		require.NoError(t, err)
		require.False(t, allowed)
	})

	t.Run("plain user audience is accepted", func(t *testing.T) {
		allowed, err := decideAudience(auth.AccessTokenAudience, expected, nil)
		require.NoError(t, err)
		require.True(t, allowed)
	})

	t.Run("infra failure is reported as an error, not a token verdict", func(t *testing.T) {
		allowed, err := decideAudience(expected, "", infraDown)
		require.Error(t, err)
		require.False(t, allowed)
	})
}

func generateUserAudienceToken(t *testing.T, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":          "bytebase",
		"sub":          "test@example.com",
		"aud":          auth.AccessTokenAudience,
		"workspace_id": "ws-test",
		"exp":          time.Now().Add(time.Hour).Unix(),
		"iat":          time.Now().Unix(),
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
		"iss":          "bytebase",
		"sub":          "test@example.com",
		"aud":          auth.OAuth2AccessTokenAudience,
		"workspace_id": "ws-test",
		"exp":          time.Now().Add(time.Hour).Unix(),
		"iat":          time.Now().Unix(),
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

// generateExpiredToken mints an expired legacy-audience (bb.oauth2.access)
// token. The middleware matrix accepts that audience with no clock gate, so
// the "expired token returns 401" row above is the other half of the drain
// invariant: JWT expiry validation, not a window, is what retires the legacy
// population.
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
		"iss":          "bytebase",
		"sub":          "test@example.com",
		"aud":          "wrong.audience",
		"workspace_id": "ws-test",
		"exp":          time.Now().Add(time.Hour).Unix(),
		"iat":          time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "v1"
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}
