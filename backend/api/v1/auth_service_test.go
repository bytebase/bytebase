package v1

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{
			domain: "www.google.com",
			want:   "google.com",
		},
		{
			domain: "code.google.com",
			want:   "google.com",
		},
		{
			domain: "code.google.com.cn",
			want:   "google.com.cn",
		},
		{
			domain: "google.com",
			want:   "google.com",
		},
	}

	for _, test := range tests {
		got := extractDomain(test.domain)
		if got != test.want {
			t.Errorf("extractDomain %s, got %s, want %s", test.domain, got, test.want)
		}
	}
}

// TestSwitchWorkspaceMCPRecognition pins the SwitchWorkspace guard predicate
// across both MCP credential generations. An MCP session must never mint a
// plain user token: that token is not audience-bound to the MCP resource, does
// not die with the OAuth grant, and ignores the workspace MCP kill switch.
//
// The delegated-credential row is the P1a PR 4 half. Tool traffic now rides the
// internal transport, so the bearer this guard sees is the delegated
// credential, not the client's MCP token — and it is signed with a derived key,
// invisible to the raw-secret extraction the guard used to do (asserted below).
// Recognizing only extractable claims would fail OPEN for every MCP session.
func TestSwitchWorkspaceMCPRecognition(t *testing.T) {
	const secret = "test-secret"

	delegated, err := auth.GenerateInternalMCPToken(auth.DelegatedMCPCredential{
		Principal:   "demo@example.com",
		WorkspaceID: "ws-test",
		ClientID:    "client-A",
	}, secret)
	require.NoError(t, err)
	_, extractErr := auth.ExtractClaimsFromExpiredToken(delegated, secret)
	require.Error(t, extractErr, "the delegated credential must not verify under the raw secret")
	require.True(t, auth.IsMCPOriginatedToken(delegated, secret),
		"the delegated credential must be recognized as MCP-originated")

	// Current external MCP token: recognized by token_use, since its audience is
	// a per-deployment resource URI that cannot be matched by value.
	mcpToken, err := auth.GenerateOAuth2AccessToken("demo@example.com", "client-A", "ws-test", "https://bb.example.com/mcp", "", secret, time.Hour)
	require.NoError(t, err)
	require.True(t, auth.IsMCPOriginatedToken(mcpToken, secret))

	// Pre-3.23 external token: recognized by the fixed legacy audience.
	require.True(t, auth.IsMCPOriginatedToken(mustLegacyOAuth2Token(t, secret), secret))

	webToken, err := auth.GenerateAccessToken("demo@example.com", "ws-test", secret, time.Hour)
	require.NoError(t, err)
	require.False(t, auth.IsMCPOriginatedToken(webToken, secret),
		"a web session token must stay eligible to switch workspaces")
	require.False(t, auth.IsMCPOriginatedToken("", secret))
	require.False(t, auth.IsMCPOriginatedToken("not-a-jwt", secret))
}

// mustLegacyOAuth2Token mints a pre-PR-3 fixed-audience OAuth2 token.
func mustLegacyOAuth2Token(t *testing.T, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":          "bytebase",
		"sub":          "demo@example.com",
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

func TestLoginAuthMethodRequiresPasswordReset(t *testing.T) {
	emailCode := "123456"
	tests := []struct {
		name    string
		request *v1pb.LoginRequest
		want    bool
	}{
		{
			name:    "password login enforces password reset",
			request: &v1pb.LoginRequest{Email: "user@example.com", Password: "password"},
			want:    true,
		},
		{
			name:    "idp login skips password reset",
			request: &v1pb.LoginRequest{IdpName: "idps/okta"},
			want:    false,
		},
		{
			name:    "email code login skips password reset",
			request: &v1pb.LoginRequest{Email: "user@example.com", EmailCode: &emailCode},
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := loginAuthMethodFromRequest(test.request).requiresPasswordReset()
			require.Equal(t, test.want, got)
		})
	}
}

func TestMFATempTokenPreservesLoginAuthMethod(t *testing.T) {
	const secret = "test-secret"

	tests := []struct {
		name       string
		method     loginAuthMethod
		wantReset  bool
		wantMethod loginAuthMethod
	}{
		{
			name:       "password mfa completion enforces password reset",
			method:     loginAuthMethodPassword,
			wantReset:  true,
			wantMethod: loginAuthMethodPassword,
		},
		{
			name:       "idp mfa completion skips password reset",
			method:     loginAuthMethodIDP,
			wantReset:  false,
			wantMethod: loginAuthMethodIDP,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := auth.GenerateMFATempTokenWithLoginMethod("user@example.com", string(test.method), secret, time.Minute)
			require.NoError(t, err)

			email, method, err := loginAuthMethodFromMFATempToken(token, secret)
			require.NoError(t, err)
			require.Equal(t, "user@example.com", email)
			require.Equal(t, test.wantMethod, method)
			require.Equal(t, test.wantReset, method.requiresPasswordReset())
		})
	}
}

func TestLegacyMFATempTokenDefaultsToPasswordLoginAuthMethod(t *testing.T) {
	const secret = "test-secret"

	token, err := auth.GenerateMFATempToken("user@example.com", secret, time.Minute)
	require.NoError(t, err)

	email, method, err := loginAuthMethodFromMFATempToken(token, secret)
	require.NoError(t, err)
	require.Equal(t, "user@example.com", email)
	require.Equal(t, loginAuthMethodPassword, method)
	require.True(t, method.requiresPasswordReset())
}
