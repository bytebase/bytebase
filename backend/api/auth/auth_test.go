package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// TestCheckTokenAudience pins the general-API audience policy for P1a PR 3:
// the fixed audiences keep working, an MCP resource-bound token (token_use=mcp)
// is admitted but flagged for audit logging, and everything else is refused.
// The reject flag is the PR 5 flip that turns audit-only into a hard reject.
func TestCheckTokenAudience(t *testing.T) {
	claimsWith := func(aud string, tokenUse string) *claimsMessage {
		return &claimsMessage{
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{aud}},
			TokenUse:         tokenUse,
		}
	}

	t.Run("web session audience is accepted and not flagged", func(t *testing.T) {
		mcpToken, err := checkTokenAudience(claimsWith(AccessTokenAudience, ""), false)
		require.NoError(t, err)
		require.False(t, mcpToken)
	})

	t.Run("legacy oauth2 audience is accepted and not flagged", func(t *testing.T) {
		mcpToken, err := checkTokenAudience(claimsWith(OAuth2AccessTokenAudience, ""), false)
		require.NoError(t, err)
		require.False(t, mcpToken)
	})

	t.Run("mcp resource-bound token is admitted but flagged during the audit-only window", func(t *testing.T) {
		mcpToken, err := checkTokenAudience(claimsWith(testResource, TokenUseMCP), false)
		require.NoError(t, err)
		require.True(t, mcpToken)
	})

	t.Run("mcp resource-bound token is refused once the reject flip lands", func(t *testing.T) {
		mcpToken, err := checkTokenAudience(claimsWith(testResource, TokenUseMCP), true)
		require.True(t, mcpToken)
		require.Error(t, err)
		require.Contains(t, err.Error(), "/mcp")
	})

	t.Run("unknown audience without token_use is refused", func(t *testing.T) {
		_, err := checkTokenAudience(claimsWith("wrong.audience", ""), false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "audience mismatch")
	})

	t.Run("the reject flip is inactive this release", func(t *testing.T) {
		require.False(t, rejectMCPTokenOnGeneralAPI,
			"P1a PR 3 ships audit-only; the hard reject is PR 5's deliberate flip")
	})
}
