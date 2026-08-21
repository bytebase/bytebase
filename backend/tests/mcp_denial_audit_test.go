package tests

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	a.Contains(connection.Status.Message, "MCP access is disabled")
	a.NotNil(connection.McpDelegation, "the row wears the MCP badge")
	a.Equal(clientID, connection.McpDelegation.ClientId)
	a.NotEmpty(connection.McpDelegation.CorrelationId)

	// The same workspace now refuses a NEW authorization too, and that refusal
	// is a second row. Without it an operator sees an agent stop connecting and
	// cannot tell a client that gave up from one that was never let in.
	consentRefused := consentUnderDisabledCeiling(t, ctl)
	a.Equal(http.StatusForbidden, consentRefused)

	consentRows := searchMCP(`method == "/bytebase.mcp.Consent/Approve"`)
	a.Len(consentRows, 1, "the refused consent is on the audit page too")
	a.Contains(consentRows[0].Status.GetMessage(), "turned MCP access off")
	a.Empty(consentRows[0].McpDelegation.GetCorrelationId(),
		"a consent never reached the boundary that mints one")

	// The operator's two filters, against the rows they exist to find.
	a.Len(searchMCP(`mcp == "true"`), 2,
		"both denials answer the MCP filter; nothing else in this workspace does")
	byCorrelation := searchMCP(`mcp_correlation_id == "` + connection.McpDelegation.CorrelationId + `"`)
	a.Len(byCorrelation, 1)
	a.Equal(connection.Name, byCorrelation[0].Name)
}

// consentUnderDisabledCeiling registers a fresh MCP client and posts an
// approved consent form, returning the HTTP status the authorize handler
// answered with. It is the first half of mintMCPOAuthToken's flow, stopped
// where a refused consent stops it.
func consentUnderDisabledCeiling(t *testing.T, ctl *controller) int {
	t.Helper()
	httpClient := &http.Client{}

	resp, err := httpClient.Post(ctl.rootURL+"/api/oauth2/register", "application/json",
		strings.NewReader(`{"client_name":"bb-e2e-refused","redirect_uris":["http://localhost/cb"],"grant_types":["authorization_code"],"token_endpoint_auth_method":"none"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	var reg struct {
		ClientID string `json:"client_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reg))
	require.NotEmpty(t, reg.ClientID)

	verifier := "e2eVerifier_e2eVerifier_e2eVerifier_e2eVerifier"
	challenge := sha256.Sum256([]byte(verifier))
	form := url.Values{
		"client_id":             {reg.ClientID},
		"redirect_uri":          {"http://localhost/cb"},
		"state":                 {"e2e-state"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
		"action":                {"allow"},
		"resource":              {ctl.rootURL + "/mcp"},
		"scope":                 {"mcp:read-only"},
	}
	req, err := http.NewRequest(http.MethodPost, ctl.rootURL+"/api/oauth2/authorize", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+ctl.authInterceptor.token)
	consentResp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer consentResp.Body.Close()
	body, err := io.ReadAll(consentResp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(body), "code=", "a refused consent must issue no authorization code")
	return consentResp.StatusCode
}
