package tests

// The P1a migration matrix: every legacy or migration-era token shape the
// series shipped, and what the server must do with it. The rows below are the
// whole legacy story; the ones already pinned elsewhere are mapped rather than
// restated, so this comment is the index.
//
// Row 1 — scope-less legacy bb.oauth2.access at /mcp: accepted while its OWN
// exp lives, refused once expired (the drain bound is the token lifetime, not
// a process-local window). MAPPED, both halves:
//   - accepted: backend/api/mcp/server_test.go
//     TestMCPAuthMiddlewareAudienceMatrix/"unexpired legacy oauth2 audience is
//     accepted" and TestDecideAudience/"legacy oauth2 audience is accepted
//     while its token lives"; against a real store,
//     backend/api/mcp/server_killswitch_test.go TestMCPKillSwitchEndToEnd
//     (its tokenForWorkspace helper mints the legacy audience).
//   - expired: backend/api/mcp/server_test.go TestMCPAuthMiddleware/"expired
//     token returns 401", whose generateExpiredToken deliberately carries the
//     legacy audience for exactly this reason.
//
// Row 2 — a legacy refresh grant with no stored resource is refused with the
// re-consent signal (invalid_grant naming the reauthorize tool). MAPPED:
// backend/api/oauth2/grant_test.go TestResourceScopeGrantLifecycle/"legacy
// unbound grants are refused at the token endpoint with re-auth guidance" —
// both grant types, plus the negative control that a resource-bound grant
// with an empty scope is NOT legacy.
//
// Row 3 — a plain bb.user.access web token at /mcp works end to end and its
// delegation state is BOTH-EMPTY. The links were pinned separately (session
// works: TestMCPToolCallParity; credential: backend/api/mcp/
// internal_transport_test.go TestMCPAuthMiddlewareLegacySessionEmptyGrantState;
// AuthContext: backend/api/auth/internal_interceptor_livestate_test.go;
// audit row: backend/api/v1/audit_mcp_provenance_test.go) but nothing composed
// them on a live server. NEW: TestMCPMigrationGrantStateMatrix.
//
// Row 4 — a grant whose client omitted `scope` at consent: resource bound,
// scope empty. A PERMANENT population, not a migration artifact, and it must
// never collapse into row 3. The AuthContext-level distinction is pinned
// (internal_interceptor_livestate_test.go/"grant-backed token: resource
// present, scope empty"); nothing minted the state from a real scope-less
// token or carried it to an audit row. NEW:
// TestMCPMigrationGrantStateMatrix, which asserts rows 3 and 4 against each
// other.
//
// Row 5 — bb.oauth2.access on the public v1 API is refused. MAPPED:
// backend/api/auth/auth_test.go TestCheckTokenAudience/"legacy oauth2
// audience is refused: it is MCP-minted too" (the legacy audience by name),
// backend/api/oauth2/grant_test.go TestResourceScopeGrantLifecycle/"an MCP
// token is refused on the general API but keeps serving /mcp", and the
// end-to-end flow in TestMCPTokenIsRejectedOnGeneralAPI.
//
// Row 6 — a ceiling lookup FAILURE fails closed (lookup error → DISABLED →
// connection refused). NEW: TestMCPMigrationCeilingLookupFailureFailsClosed.
// The related trusted-config distinction — a deliberately unconfigured
// external URL fails resource tokens closed, while an infra failure reading
// it is an error rather than a verdict against the token — is MAPPED to
// backend/api/mcp/server_test.go TestDecideAudience.
//
// Row 7 — a tightened ceiling bites the NEXT request of a live MCP session,
// with no re-auth. The request-level half is pinned
// (backend/api/mcp/server_killswitch_test.go
// TestMCPKillSwitchBypassesSettingCache, which also pins that the read
// bypasses the setting cache), but that test has no session and the
// session-level test runs with a nil store, so the ceiling is never consulted
// there. NEW: TestMCPMigrationTightenedCeilingBitesLiveSession.

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// postMCP sends one authenticated request to /mcp and returns the boundary's
// verdict. The body is irrelevant to these rows: authMiddleware admits or
// refuses before the MCP handler ever sees it, so this reads the admission
// decision directly instead of through the SDK's error shape.
func postMCP(t *testing.T, ctl *controller, bearer string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ctl.rootURL+"/mcp", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

// openMCPSession initializes a real MCP session against /mcp, the way a client
// holding this bearer would.
func openMCPSession(ctx context.Context, t *testing.T, ctl *controller, bearer string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "bb-e2e", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   ctl.rootURL + "/mcp",
		HTTPClient: &http.Client{Transport: &bearerTransport{token: bearer}},
	}, nil)
	require.NoError(t, err)
	return session
}

// callAPIStatus runs the call_api tool and returns the status the internal
// chain answered with. call_api reports internal-API failures in its
// structured output and never sets IsError, so the status is the only honest
// signal.
func callAPIStatus(ctx context.Context, t *testing.T, session *mcp.ClientSession, operationID string, body map[string]any) int {
	t.Helper()
	arguments := map[string]any{"operationId": operationID}
	if body != nil {
		arguments["body"] = body
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "call_api", Arguments: arguments})
	require.NoError(t, err)
	raw, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var out struct {
		Status int    `json:"status"`
		Error  string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.NotEqual(t, 0, out.Status, "call_api returned no status: %s", out.Error)
	return out.Status
}

// jwtClaims decodes a JWT payload without verifying it — enough to assert
// which claims a minted token carries.
func jwtClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "expected a three-part JWT")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	claims := map[string]any{}
	require.NoError(t, json.Unmarshal(payload, &claims))
	return claims
}

// TestMCPMigrationGrantStateMatrix is matrix rows 3 and 4, asserted against
// each other on one live server: the two empty grant states must stay
// distinguishable all the way to what an operator reads.
//
// A plain bb.user.access web token pasted into an MCP client is a genuinely
// pre-grant session — no scope, no resource. A grant whose client omitted
// `scope` at consent looks the same on the scope axis but IS resource-bound,
// and that resource is the only thing separating it from the pre-grant case.
// Collapsing the two would widen a consented session to full legacy
// semantics, which is why the discriminator is asserted here rather than
// assumed: the state is permanent (a client that never sends `scope` produces
// it forever), not a migration artifact that drains.
func TestMCPMigrationGrantStateMatrix(t *testing.T) {
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

	// Row 3: the legacy admission — a plain web-session token driving MCP.
	plainSession := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer plainSession.Close()
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, plainSession, "UserService/UpdateUser", map[string]any{
		"user":       map[string]any{"name": ctl.principalName, "title": "driven by a pre-grant session"},
		"updateMask": "title",
	}), "a plain web-session token must keep working through tools")

	// Row 4: a real consent that never names a scope.
	scopelessToken, clientID := mintMCPOAuthTokenWithScope(t, ctl, ctl.authInterceptor.token, "")
	claims := jwtClaims(t, scopelessToken)
	_, hasScope := claims["scope"]
	a.False(hasScope, "a scope-less consent must mint a token carrying no scope claim at all")
	a.Equal("mcp", claims["token_use"], "the token is still MCP-minted; only the scope is absent")

	scopelessSession := openMCPSession(ctx, t, ctl, scopelessToken)
	defer scopelessSession.Close()
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, scopelessSession, "UserService/UpdateUser", map[string]any{
		"user":       map[string]any{"name": ctl.principalName, "title": "driven by a scope-less grant"},
		"updateMask": "title",
	}), "a grant that recorded no scope must still work end to end")

	// What an operator investigating these sessions sees.
	rows, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  workspace.Msg.Name,
		Filter:  `method == "/bytebase.v1.UserService/UpdateUser"`,
		OrderBy: "create_time desc",
	}))
	a.NoError(err)
	var delegated []*v1pb.MCPDelegation
	for _, row := range rows.Msg.AuditLogs {
		if row.McpDelegation != nil {
			delegated = append(delegated, row.McpDelegation)
		}
	}
	a.Len(delegated, 2, "each MCP session's call must produce exactly one provenance-carrying row")

	var preGrant, resourceOnly *v1pb.MCPDelegation
	for _, d := range delegated {
		if d.Resource == "" {
			preGrant = d
		} else {
			resourceOnly = d
		}
	}
	a.NotNil(preGrant, "the plain-session row must record an empty resource")
	a.NotNil(resourceOnly, "the scope-less grant's row must record its bound resource")

	a.Empty(preGrant.Scope, "a pre-grant session has no consented scope")
	a.Empty(preGrant.ClientId, "a pasted web token was never issued to an OAuth client")
	a.NotEmpty(preGrant.CorrelationId, "every MCP-originated row must be correlatable to its session")

	a.Empty(resourceOnly.Scope, "the grant recorded no scope, so the row records none — never a resolved label")
	a.Equal(ctl.rootURL+"/mcp", resourceOnly.Resource, "the grant's stored resource travels verbatim")
	a.Equal(clientID, resourceOnly.ClientId)
	a.NotEmpty(resourceOnly.CorrelationId)

	a.NotEqual(preGrant.CorrelationId, resourceOnly.CorrelationId,
		"two sessions are two delegations")
}

// TestMCPMigrationCeilingLookupFailureFailsClosed is matrix row 6: a ceiling
// the server cannot read must refuse the connection, never fall back to
// permitting MCP.
//
// The failure is induced by storing a ceiling the policy row cannot be parsed
// with — a wrong-typed value, which is what the setting read returns an error
// for. That is the reproducible shape of "the policy lookup blew up" on a live
// server; taking the database away instead would take the rest of the server
// with it. Nothing else pinned this branch: the DISABLED verdict
// was pinned as a pure function and as a stored value, but never as the
// answer to a lookup that failed, so the branch could have been flipped to
// READ_WRITE with the whole suite staying green.
func TestMCPMigrationCeilingLookupFailureFailsClosed(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	mcpToken, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)

	status, body := postMCP(t, ctl, mcpToken)
	a.NotEqual(http.StatusForbidden, status,
		"control: with a readable policy the ceiling admits this session; %s", body)

	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()

	// A wrong-typed ceiling is what makes the read fail. (An unrecognized enum
	// NAME would not: the store's unmarshaler discards unknown values, so the
	// field would simply read as unset — a different behavior, reported
	// separately, and deliberately not asserted here.)
	result, err := db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{mcpCapability}', 'true')
		WHERE name = 'WORKSPACE_PROFILE';
	`)
	a.NoError(err)
	affected, err := result.RowsAffected()
	a.NoError(err)
	a.Equal(int64(1), affected, "the workspace profile row must exist for this test to mean anything")

	status, body = postMCP(t, ctl, mcpToken)
	a.Equal(http.StatusForbidden, status,
		"a ceiling that cannot be read must fail closed, not fall back to permitting MCP; %s", body)
	a.Contains(body, "MCP access is disabled")

	// Restoring the row proves the refusal was the unreadable policy and
	// nothing else: the very same token opens a session again.
	_, err = db.ExecContext(ctx, `
		UPDATE setting SET value = value - 'mcpCapability'
		WHERE name = 'WORKSPACE_PROFILE';
	`)
	a.NoError(err)

	session := openMCPSession(ctx, t, ctl, mcpToken)
	defer session.Close()
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, session, "ProjectService/ListProjects", nil))
}

// TestMCPMigrationTightenedCeilingBitesLiveSession is matrix row 7: tightening
// the workspace ceiling controls entry, so it takes effect on the NEXT request
// of an already-open session — no re-auth, no token re-issue, nothing revoked.
//
// The ceiling is read live on every /mcp request, which is what makes this
// admission control rather than a property of session setup. READ_ONLY is the
// tightening used here because this phase cannot yet clamp per tool, so a
// ceiling the server cannot apply must refuse the connection instead of
// silently granting read-write.
func TestMCPMigrationTightenedCeilingBitesLiveSession(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	mcpToken, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	session := openMCPSession(ctx, t, ctl, mcpToken)
	defer session.Close()
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, session, "ProjectService/ListProjects", nil),
		"the session is live under the default ceiling")

	// An admin tightens the ceiling while the session stays open.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.WorkspaceProfileSetting_READ_ONLY))

	status, body := postMCP(t, ctl, mcpToken)
	a.Equal(http.StatusForbidden, status,
		"the same bearer must be refused on its next request after the ceiling tightens; %s", body)
	a.Contains(body, "MCP access is disabled")

	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "call_api",
		Arguments: map[string]any{"operationId": "ProjectService/ListProjects"},
	})
	a.Error(err,
		"an established session must not keep serving tool calls under a ceiling that now refuses it")

	// Widening again admits a new session, so the refusal was the ceiling
	// rather than damage to the session or the grant.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.WorkspaceProfileSetting_READ_WRITE))
	restored := openMCPSession(ctx, t, ctl, mcpToken)
	defer restored.Close()
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, restored, "ProjectService/ListProjects", nil))
}
