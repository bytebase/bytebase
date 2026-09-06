package oauth2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestConsentRefusal pins what a consent refused by the MCP ceiling leaves
// behind: the audit row an operator finds with `mcp == true`, and the page the
// user sees. The verdict itself is auth.ClassifyMCPCeiling's, pinned in
// api/auth; that a refused consent writes no authorization code and lands this
// row on a live server is TestMCPPolicyDenialsReachTheAuditPage in
// backend/tests.
func TestConsentRefusal(t *testing.T) {
	attempt := consentAttempt{
		user:        consentingUser{email: "demo@example.com", workspaceID: "ws-disabled"},
		clientID:    "client-A",
		params:      grantParams{resource: "https://bb.example.com/mcp", scope: "mcp:read-only"},
		redirectURI: "http://localhost/cb",
		state:       "state-1",
	}
	for _, tc := range []struct {
		name       string
		capability storepb.MCPSetting_Capability
		message    string
		page       []string
	}{
		{
			name:       "MCP turned off",
			capability: storepb.MCPSetting_DISABLED,
			message:    "turned MCP access off",
			page:       []string{"<title>MCP access is turned off</title>", "turned MCP access off for this workspace", "in the workspace settings"},
		},
		{
			name:       "a ceiling this build does not serve",
			capability: storepb.MCPSetting_Capability(2),
			message:    "not one this build serves",
			page:       []string{"not one this build serves", "not one this version supports"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verdict := auth.ClassifyMCPCeiling(&storepb.MCPSetting{Capability: tc.capability}, nil)
			require.True(t, verdict.IsPolicy())

			row := consentRefusalRow(attempt, verdict, &storepb.RequestMetadata{CallerIp: "10.0.1.50"})
			require.Equal(t, common.AuditMethodMCPConsentApprove, row.Method)
			require.Equal(t, "workspaces/ws-disabled", row.Parent)
			require.Equal(t, "users/demo@example.com", row.User)
			require.EqualValues(t, 7, row.GetStatus().GetCode(), "PermissionDenied")
			require.Contains(t, row.GetStatus().GetMessage(), tc.message)
			require.Equal(t, "10.0.1.50", row.GetRequestMetadata().GetCallerIp())
			// The MCP marker, so `mcp == true` finds this alongside the
			// connection denial the same user would have hit next.
			require.Equal(t, "client-A", row.GetMcpDelegation().GetClientId())
			require.Equal(t, "mcp:read-only", row.GetMcpDelegation().GetScope())
			require.Equal(t, "https://bb.example.com/mcp", row.GetMcpDelegation().GetResource())
			require.Empty(t, row.GetMcpDelegation().GetCorrelationId(),
				"a consent never reached the /mcp boundary that mints one")

			page := consentRefusedHTML(verdict, attempt.redirectURI, attempt.state)
			for _, want := range tc.page {
				require.Contains(t, page, want)
			}
			require.NotContains(t, page, "code=", "no authorization code may be issued")
			require.Contains(t, page, "error=access_denied", "the way back tells the client the policy refused it")
			require.True(t, strings.Contains(page, "Nothing was connected."))
		})
	}
}
