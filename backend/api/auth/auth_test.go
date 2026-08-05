package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// TestCheckTokenAudience pins the general-API audience policy for P1a PR 3:
// the fixed audiences keep working, an MCP resource-bound token (token_use=mcp)
// is admitted but flagged for audit logging, and everything else is refused.
// PR 5 replaces the admission with a rejection once PR 4's private transport
// stops /mcp tool calls from carrying the inbound bearer here.
func TestCheckTokenAudience(t *testing.T) {
	claimsWith := func(aud string, tokenUse string) *claimsMessage {
		return &claimsMessage{
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{aud}},
			TokenUse:         tokenUse,
		}
	}

	t.Run("web session audience is accepted and not flagged", func(t *testing.T) {
		mcpToken, err := checkTokenAudience(claimsWith(AccessTokenAudience, ""))
		require.NoError(t, err)
		require.False(t, mcpToken)
	})

	t.Run("legacy oauth2 audience is accepted and not flagged", func(t *testing.T) {
		mcpToken, err := checkTokenAudience(claimsWith(OAuth2AccessTokenAudience, ""))
		require.NoError(t, err)
		require.False(t, mcpToken)
	})

	t.Run("mcp resource-bound token is admitted but flagged for auditing", func(t *testing.T) {
		mcpToken, err := checkTokenAudience(claimsWith(testResource, TokenUseMCP))
		require.NoError(t, err)
		require.True(t, mcpToken)
	})

	t.Run("unknown audience without token_use is refused", func(t *testing.T) {
		_, err := checkTokenAudience(claimsWith("wrong.audience", ""))
		require.Error(t, err)
		require.Contains(t, err.Error(), "audience mismatch")
	})
}
