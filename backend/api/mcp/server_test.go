package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func TestMCPAuthMiddleware(t *testing.T) {
	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev}

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
			handler := s.authMiddleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			})

			err = handler(c)
			if err != nil {
				// Echo error handler
				e.HTTPErrorHandler(err, c)
			}

			require.Equal(t, tc.expectedStatus, rec.Code)
			require.Contains(t, strings.ToLower(rec.Body.String()), strings.ToLower(tc.expectedBody))
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
	handler := s.authMiddleware(func(c echo.Context) error {
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
