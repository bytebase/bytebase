package v1

import (
	"testing"
	"time"

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

// TestIsMCPBoundToken pins the SwitchWorkspace guard predicate through the real
// mint -> extract pipeline. For every token minted after P1a PR 3 the token_use
// clause is the only one that fires (the audience is a per-deployment resource
// URI, not the fixed legacy string), so dropping it would let a workspace-bound
// MCP token mint plain user tokens for other workspaces.
func TestIsMCPBoundToken(t *testing.T) {
	const secret = "test-secret"

	t.Run("current MCP token is caught via token_use", func(t *testing.T) {
		tokenStr, err := auth.GenerateOAuth2AccessToken("demo@example.com", "client-A", "ws-test", "https://bb.example.com/mcp", secret, time.Hour)
		require.NoError(t, err)
		claims, err := auth.ExtractClaimsFromExpiredToken(tokenStr, secret)
		require.NoError(t, err)
		require.True(t, isMCPBoundToken(claims))
	})

	t.Run("legacy oauth2 token is caught via the fixed audience", func(t *testing.T) {
		require.True(t, isMCPBoundToken(&auth.ExpiredTokenClaims{
			Audience: []string{auth.OAuth2AccessTokenAudience},
		}))
	})

	t.Run("web session token is not caught", func(t *testing.T) {
		tokenStr, err := auth.GenerateAccessToken("demo@example.com", "ws-test", secret, time.Hour)
		require.NoError(t, err)
		claims, err := auth.ExtractClaimsFromExpiredToken(tokenStr, secret)
		require.NoError(t, err)
		require.False(t, isMCPBoundToken(claims))
	})
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
