package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	testSecret   = "test-secret"
	testResource = "https://bb.example.com/mcp"
)

// TestGenerateOAuth2AccessTokenClaims pins the P1a PR 3 token shape: the
// audience is the canonical MCP resource URI the grant was consented for (passed
// in by the token endpoint from the stored grant, never recomputed from live
// config), and token_use marks the token as an MCP credential so the general
// API interceptor can recognize it without knowing every deployment's resource
// URI.
func TestGenerateOAuth2AccessTokenClaims(t *testing.T) {
	tokenStr, err := GenerateOAuth2AccessToken("demo@example.com", "client-A", "ws-test", testResource, "", testSecret, time.Hour)
	require.NoError(t, err)

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(_ *jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})
	require.NoError(t, err)

	require.Equal(t, []any{testResource}, claims["aud"],
		"audience must be the stored canonical resource URI, not the fixed bb.oauth2.access string")
	require.Equal(t, TokenUseMCP, claims["token_use"])
	require.Equal(t, "demo@example.com", claims["sub"])
	require.Equal(t, "client-A", claims["client_id"])
	require.Equal(t, "ws-test", claims["workspace_id"])
}

// TestWebTokensCarryNoTokenUse pins that only OAuth2 MCP tokens get the
// token_use claim; a web session token must not be mistakable for one.
func TestWebTokensCarryNoTokenUse(t *testing.T) {
	tokenStr, err := GenerateAccessToken("demo@example.com", "ws-test", testSecret, time.Hour)
	require.NoError(t, err)

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(_ *jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})
	require.NoError(t, err)

	require.Equal(t, []any{AccessTokenAudience}, claims["aud"])
	require.NotContains(t, claims, "token_use")
}

// TestExtractClaimsFromExpiredTokenTokenUse verifies the expired-token
// extraction used by SwitchWorkspace surfaces token_use, because that guard can
// no longer key on the fixed bb.oauth2.access audience once tokens carry a
// per-deployment resource URI instead.
func TestExtractClaimsFromExpiredTokenTokenUse(t *testing.T) {
	tokenStr, err := GenerateOAuth2AccessToken("demo@example.com", "client-A", "ws-test", testResource, "", testSecret, -time.Minute)
	require.NoError(t, err)

	claims, err := ExtractClaimsFromExpiredToken(tokenStr, testSecret)
	require.NoError(t, err)
	require.Equal(t, TokenUseMCP, claims.TokenUse)
	require.Equal(t, []string{testResource}, []string(claims.Audience))

	webToken, err := GenerateAccessToken("demo@example.com", "ws-test", testSecret, time.Hour)
	require.NoError(t, err)
	webClaims, err := ExtractClaimsFromExpiredToken(webToken, testSecret)
	require.NoError(t, err)
	require.Empty(t, webClaims.TokenUse)
}
