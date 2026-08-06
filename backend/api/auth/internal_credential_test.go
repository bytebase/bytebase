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
// bb.user.access at /mcp, pre-scope OAuth2 tokens) mint a credential with
// empty scope and resource, carried verbatim into common.AuthContext — P1b,
// not this layer, resolves what empty grant state may do
// (common.DelegatedGrant).
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

// TestInternalMCPCredentialWireShape pins what separates the internal
// credential from every public one: a signing key derived from the secret, and
// a dedicated audience. Each layer independently refuses in both directions, so
// this test is the only place the derivation itself is pinned — the
// public-surface rejection tests stay green on the audience check alone.
//
// There is deliberately no token_use: unlike the external MCP token, whose
// per-deployment resource audience cannot be matched against a constant, this
// audience is a constant and already says "internal". The kid is shared with
// public tokens, so it discriminates nothing; that is what the derived key and
// audience are for.
func TestInternalMCPCredentialWireShape(t *testing.T) {
	tokenStr, err := GenerateInternalMCPToken(testDelegatedCredential(), testSecret)
	require.NoError(t, err)

	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
	require.NoError(t, err)
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	require.Equal(t, []any{internalMCPAudience}, claims["aud"])
	require.NotContains(t, claims, "token_use",
		"the audience identifies the credential; a token_use would be redundant")

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
	_, _, err = in.authenticate(context.Background(), tokenStr)
	require.Error(t, err, "the general API must never accept the internal MCP credential")
}

// TestCheckTokenAudienceRejectsInternalCredential covers the second boundary
// layer on the general API: even if the internal credential's signature were
// somehow accepted, its audience admits it nowhere. Carrying no token_use is
// part of that — the claim is absent, so the audience mismatch branch refuses
// it rather than the MCP rejection (which keys on token_use == TokenUseMCP).
func TestCheckTokenAudienceRejectsInternalCredential(t *testing.T) {
	claims := &claimsMessage{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{internalMCPAudience},
		},
	}
	require.Error(t, checkTokenAudience(claims))
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
