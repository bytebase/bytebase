package mcp

import (
	"io"
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
