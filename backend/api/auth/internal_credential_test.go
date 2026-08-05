package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func testDelegatedCredential() DelegatedMCPCredential {
	return DelegatedMCPCredential{
		Principal:     "demo@example.com",
		WorkspaceID:   "ws-test",
		ClientID:      "client-A",
		CorrelationID: "corr-123",
		Scope:         "mcp:read-only",
		Resource:      testResource,
	}
}

// TestInternalMCPCredentialRoundTrip pins the delegated credential shape: every
// claim the P1b contract keys on (principal, workspace, client_id, correlation
// ID, grant scope + resource) survives a mint/verify round trip verbatim.
func TestInternalMCPCredentialRoundTrip(t *testing.T) {
	tokenStr, err := GenerateInternalMCPToken(testDelegatedCredential(), testSecret)
	require.NoError(t, err)

	got, err := VerifyInternalMCPToken(tokenStr, testSecret)
	require.NoError(t, err)
	require.Equal(t, testDelegatedCredential(), *got)
}

// TestInternalMCPCredentialEmptyGrantState pins that legacy sessions (plain
// bb.user.access at /mcp, pre-scope OAuth2 tokens) mint a credential with empty
// scope and resource — PR 5, not this PR, assigns their LEGACY_FULL semantics.
func TestInternalMCPCredentialEmptyGrantState(t *testing.T) {
	cred := testDelegatedCredential()
	cred.Scope = ""
	cred.Resource = ""
	tokenStr, err := GenerateInternalMCPToken(cred, testSecret)
	require.NoError(t, err)

	got, err := VerifyInternalMCPToken(tokenStr, testSecret)
	require.NoError(t, err)
	require.Empty(t, got.Scope)
	require.Empty(t, got.Resource)
	require.Equal(t, "demo@example.com", got.Principal)
}

// TestInternalMCPCredentialWireShape pins the boundary markers on the wire:
// a dedicated kid (so the public keyfuncs, which only ever return the key for
// kid "v1", refuse to even verify the signature), a dedicated audience, and a
// dedicated token_use. Each is an independent rejection layer on the public
// surfaces.
func TestInternalMCPCredentialWireShape(t *testing.T) {
	tokenStr, err := GenerateInternalMCPToken(testDelegatedCredential(), testSecret)
	require.NoError(t, err)

	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
	require.NoError(t, err)
	require.Equal(t, internalMCPKeyID, token.Header["kid"])
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	require.Equal(t, []any{InternalMCPAudience}, claims["aud"])
	require.Equal(t, TokenUseMCPInternal, claims["token_use"])

	// The signing key must not be the raw secret: even a hypothetical kid-check
	// bypass on a public surface fails signature verification.
	_, err = jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(_ *jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})
	require.Error(t, err, "internal credential must not verify under the raw shared secret")
}

// TestGeneralAPIRejectsInternalMCPCredential is the public-surface boundary
// test for the v1 API: the interceptor's authenticate path must refuse the
// internal credential outright. The kid gate fires before any store access, so
// a nil-store interceptor suffices.
func TestGeneralAPIRejectsInternalMCPCredential(t *testing.T) {
	tokenStr, err := GenerateInternalMCPToken(testDelegatedCredential(), testSecret)
	require.NoError(t, err)

	in := New(nil, testSecret, nil, nil, nil)
	_, _, err = in.authenticate(context.Background(), tokenStr, "/bytebase.v1.SQLService/Query")
	require.Error(t, err, "the general API must never accept the internal MCP credential")
}

// TestCheckTokenAudienceRejectsInternalCredential covers the second boundary
// layer on the general API: even if the internal credential's signature were
// somehow accepted, its audience and token_use admit it nowhere.
func TestCheckTokenAudienceRejectsInternalCredential(t *testing.T) {
	claims := &claimsMessage{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{InternalMCPAudience},
		},
		TokenUse: TokenUseMCPInternal,
	}
	_, err := checkTokenAudience(claims)
	require.Error(t, err)
}

// TestVerifyInternalMCPTokenRejectsPublicTokens is the mirror boundary: the
// internal transport accepts ONLY the internal credential. Web session tokens
// and OAuth2 MCP tokens (both kid "v1") must be refused, as must expired
// internal credentials and garbage.
func TestVerifyInternalMCPTokenRejectsPublicTokens(t *testing.T) {
	webToken, err := GenerateAccessToken("demo@example.com", "ws-test", testSecret, time.Hour)
	require.NoError(t, err)
	_, err = VerifyInternalMCPToken(webToken, testSecret)
	require.Error(t, err, "web session token must be refused on the internal transport")

	mcpToken, err := GenerateOAuth2AccessToken("demo@example.com", "client-A", "ws-test", testResource, "mcp:read-only", testSecret, time.Hour)
	require.NoError(t, err)
	_, err = VerifyInternalMCPToken(mcpToken, testSecret)
	require.Error(t, err, "external MCP OAuth2 token must be refused on the internal transport")

	expired := testDelegatedCredential()
	expiredStr, err := generateInternalMCPTokenWithExpiry(expired, testSecret, time.Now().Add(-time.Minute))
	require.NoError(t, err)
	_, err = VerifyInternalMCPToken(expiredStr, testSecret)
	require.Error(t, err, "expired internal credential must be refused")

	_, err = VerifyInternalMCPToken("not-a-jwt", testSecret)
	require.Error(t, err)
}

// TestOAuth2AccessTokenCarriesScope pins the enabler this PR adds to the public
// OAuth2 token: the grant's stored scope travels on the access token verbatim
// (RFC 9068 shape), so the /mcp boundary can copy it onto the delegated
// credential without a store lookup. Empty scope (legacy grants) omits the
// claim.
func TestOAuth2AccessTokenCarriesScope(t *testing.T) {
	tokenStr, err := GenerateOAuth2AccessToken("demo@example.com", "client-A", "ws-test", testResource, "mcp:read-only", testSecret, time.Hour)
	require.NoError(t, err)

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(_ *jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})
	require.NoError(t, err)
	require.Equal(t, "mcp:read-only", claims["scope"])

	emptyScope, err := GenerateOAuth2AccessToken("demo@example.com", "client-A", "ws-test", testResource, "", testSecret, time.Hour)
	require.NoError(t, err)
	claims = jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(emptyScope, claims, func(_ *jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})
	require.NoError(t, err)
	require.NotContains(t, claims, "scope")
}
