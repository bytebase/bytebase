package tests

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestMCPPolicyDenialsReachTheAuditPage drives both echo-route denials against
// a live server and reads them back through the audit API an operator uses.
//
// It is here rather than in a package test because the halves are wired
// separately and only compose on a running server: the /mcp middleware and the
// OAuth2 consent handler each write their own row, the interceptor that writes
// every other row never sees either, and the filters that find them are the
// store's. A unit test of any one piece passes while the operator's view stays
// empty.
func TestMCPPolicyDenialsReachTheAuditPage(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)

	searchMCP := func(filter string) []*v1pb.AuditLog {
		resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:  workspace.Msg.Name,
			Filter:  filter,
			OrderBy: "create_time desc",
		}))
		a.NoError(err)
		return resp.Msg.AuditLogs
	}

	// A session that connects first, so the denial below is the ceiling and
	// nothing else about this token.
	mcpToken, clientID := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	status, body := postMCP(t, ctl, mcpToken)
	a.Equal(http.StatusOK, status, "control: the ceiling admits this session; %s", body)
	a.Empty(searchMCP(`method == "/bytebase.mcp.Session/Authorize"`),
		"an admitted connection writes no row")

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_DISABLED))

	status, body = postMCP(t, ctl, mcpToken)
	a.Equal(http.StatusForbidden, status, "the ceiling refuses the next request; %s", body)

	connectionRows := searchMCP(`method == "/bytebase.mcp.Session/Authorize"`)
	a.Len(connectionRows, 1, "the refused connection is on the audit page")
	connection := connectionRows[0]
	a.Equal(ctl.principalName, connection.User)
	a.NotNil(connection.Status)
	a.Contains(connection.Status.Message, "turned MCP access off")
	a.NotNil(connection.McpDelegation, "the row wears the MCP badge")
	a.Equal(clientID, connection.McpDelegation.ClientId)
	a.Empty(connection.McpDelegation.CorrelationId,
		"the ceiling refuses before the SDK resolves a session, so there is no session ID to record")

	// The same workspace now refuses a NEW authorization too, and that refusal
	// is a second row. Without it an operator sees an agent stop connecting and
	// cannot tell a client that gave up from one that was never let in.
	consentRefused := consentUnderDisabledCeiling(t, ctl, "mcp:read-only")
	a.Equal(http.StatusForbidden, consentRefused)

	consentRows := searchMCP(`method == "/bytebase.mcp.Consent/Approve"`)
	a.Len(consentRows, 1, "the refused consent is on the audit page too")
	a.Contains(consentRows[0].Status.GetMessage(), "turned MCP access off")
	a.Empty(consentRows[0].McpDelegation.GetCorrelationId(),
		"a consent never reached the boundary that mints one")

	// The operator's two filters, through the read API they are used from.
	a.Len(searchMCP("mcp == true"), 2,
		"both denials answer the MCP filter; nothing else in this workspace does")

	// The correlation key parses and runs here; what it selects is pinned
	// against real session IDs in TestSearchAuditLogsMCPFilters. Neither denial
	// belongs to a session, so neither answers it — which is the point: an
	// operator pivoting on a session must not be handed rows that were never
	// part of one.
	a.Empty(searchMCP(`mcp_correlation_id == "any-session"`))

	// The refusal cannot be walked around by asking for a different scope, or
	// for none: the ceiling decides whether any grant is issued at all, before
	// the requested mode is considered. And every attempt is recorded, not
	// just the first.
	for _, scope := range []string{"mcp:read-write", ""} {
		a.Equal(http.StatusForbidden, consentUnderDisabledCeiling(t, ctl, scope), "scope %q", scope)
	}
	a.Len(searchMCP(`method == "/bytebase.mcp.Consent/Approve"`), 3)
}

// consentUnderDisabledCeiling registers a fresh MCP client and posts an
// approved consent form, returning the HTTP status the authorize handler
// answered with. It is the first half of mintMCPOAuthToken's flow, stopped
// where a refused consent stops it. An empty scope omits the parameter.
func consentUnderDisabledCeiling(t *testing.T, ctl *controller, scope string) int {
	t.Helper()
	extra := url.Values{"resource": {ctl.rootURL + "/mcp"}}
	if scope != "" {
		extra.Set("scope", scope)
	}
	status, body := postOAuth2Consent(t, ctl, ctl.authInterceptor.token, oauth2ConsentForm(registerOAuth2Client(t, ctl), extra))
	require.NotContains(t, body, "code=", "a refused consent must issue no authorization code")
	return status
}
