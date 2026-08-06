package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// TestCheckTokenAudience pins the general-API audience policy for P1a PR 5:
// the fixed audiences keep working, an MCP token (token_use=mcp) is refused
// outright — since PR 4's private transport, /mcp tool traffic never presents
// it here, so any appearance is a leaked or misused token, not tool traffic —
// and everything else is refused as an audience mismatch.
func TestCheckTokenAudience(t *testing.T) {
	claimsWith := func(aud string, tokenUse string) *claimsMessage {
		return &claimsMessage{
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{aud}},
			TokenUse:         tokenUse,
		}
	}

	t.Run("web session audience is accepted", func(t *testing.T) {
		err := checkTokenAudience(claimsWith(AccessTokenAudience, ""))
		require.NoError(t, err)
	})

	t.Run("legacy oauth2 audience is refused: it is MCP-minted too", func(t *testing.T) {
		// bb.oauth2.access was only ever minted by the MCP authorization
		// server (pre-PR-3, before tokens carried token_use), so it is an MCP
		// token by provenance and gets the same refusal. Unlike at /mcp, no
		// legitimate traffic drains through here: the pre-PR-4 loopback
		// transport dialed its own replica, so even mid-rolling-upgrade no old
		// replica ever lands a legacy bearer on this chain.
		err := checkTokenAudience(claimsWith(OAuth2AccessTokenAudience, ""))
		require.Error(t, err)
		require.Contains(t, err.Error(), "only accepted at /mcp")
	})

	t.Run("mcp resource-bound token is refused", func(t *testing.T) {
		err := checkTokenAudience(claimsWith(testResource, TokenUseMCP))
		require.Error(t, err)
		require.Contains(t, err.Error(), "only accepted at /mcp")
	})

	t.Run("token_use=mcp is refused even with the accepted fixed audience", func(t *testing.T) {
		// Nothing mints this combination; if one ever appears, the MCP marker
		// must win over the audience allowlist — the rejection keys on what the
		// token IS, not on which audience it also happens to carry.
		err := checkTokenAudience(claimsWith(AccessTokenAudience, TokenUseMCP))
		require.Error(t, err)
		require.Contains(t, err.Error(), "only accepted at /mcp")
	})

	t.Run("unknown audience without token_use is refused", func(t *testing.T) {
		err := checkTokenAudience(claimsWith("wrong.audience", ""))
		require.Error(t, err)
		require.Contains(t, err.Error(), "audience mismatch")
	})
}
