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
// scope empty. The state has two indistinguishable origins — scope-omitting
// clients, which make it permanent rather than only migration-era, and,
// transiently, PR-3-era tokens whose grant DID record a scope — and it must
// never collapse into row 3. The AuthContext-level distinction is pinned
// (internal_interceptor_livestate_test.go/"grant-backed token: resource
// present, scope empty"); nothing minted the state from a real scope-less
// token, carried it to an audit row, or checked it survives a refresh that
// names a scope — the one path that could widen a grant which recorded none.
// NEW: TestMCPMigrationGrantStateMatrix, which asserts rows 3 and 4 against
// each other.
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

// postMCP sends one well-formed MCP initialize request to /mcp and returns the
// boundary's verdict, so admission and refusal are both observable positively:
// an admitted request reaches the handler and answers 200, a refused one
// carries the refusing layer's status and message. Reading the status directly
// keeps these rows off the SDK's error shape.
func postMCP(t *testing.T, ctl *controller, bearer string) (int, string) {
	t.Helper()
	const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"bb-e2e","version":"0"}}}`
	req, err := http.NewRequest(http.MethodPost, ctl.rootURL+"/mcp", strings.NewReader(initialize))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
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
// assumed: the state does not fully drain (a client that never sends `scope`
// produces it forever), so it outlives the migration window that also mints
// it.
func TestMCPMigrationGrantStateMatrix(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// The probe is an audited mutation, so each session leaves exactly one
	// provenance-carrying row to sort out below, and it is a WRITE method
	// because the MCP ceiling serves READ and WRITE only. Every audited
	// serving-class method is project-scoped, hence the project.
	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: "mcp-grant-matrix",
		Project:   &v1pb.Project{Title: "MCP grant matrix"},
	}))
	a.NoError(err)
	createSheet := func(sql string) map[string]any {
		return map[string]any{
			"parent": project.Msg.Name,
			"sheet":  map[string]any{"content": base64.StdEncoding.EncodeToString([]byte(sql))},
		}
	}

	// Row 3: the legacy admission — a plain web-session token driving MCP.
	plainSession := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer plainSession.Close()
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, plainSession, "SheetService/CreateSheet",
		createSheet("SELECT 'driven by a pre-grant session';")),
		"a plain web-session token must keep working through tools")

	// Row 4: a real consent that never names a scope.
	scopelessToken, scopelessRefresh, clientID := mintMCPOAuthTokenWithScope(t, ctl, ctl.authInterceptor.token, "")
	claims := jwtClaims(t, scopelessToken)
	_, hasScope := claims["scope"]
	a.False(hasScope, "a scope-less consent must mint a token carrying no scope claim at all")
	a.Equal("mcp", claims["token_use"], "the token is still MCP-minted; only the scope is absent")

	// The state has to survive the grant's own life, not just its first
	// issuance. A refresh re-issues a grant as consented, and a scope-less
	// grant is the one shape the consented-scope check waves through
	// unvalidated (there is no consented value to compare against), so naming
	// a scope on refresh is the way this state could quietly widen. Widening
	// belongs to a fresh consent.
	refreshedToken := refreshMCPGrant(t, ctl, clientID, scopelessRefresh, "mcp:read-write")
	_, widened := jwtClaims(t, refreshedToken)["scope"]
	a.False(widened,
		"a refresh must carry the grant forward unchanged; naming a scope must not widen a grant that recorded none")

	scopelessSession := openMCPSession(ctx, t, ctl, scopelessToken)
	defer scopelessSession.Close()
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, scopelessSession, "SheetService/CreateSheet",
		createSheet("SELECT 'driven by a scope-less grant';")),
		"a grant that recorded no scope must still work end to end")

	// What an operator investigating these sessions sees.
	rows, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  project.Msg.Name,
		Filter:  `method == "/bytebase.v1.SheetService/CreateSheet"`,
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

	// Each row is attributed by the OAuth client it came from — an axis
	// independent of the resource — so the resource assertions below test the
	// discriminator instead of restating how the rows were sorted.
	var preGrant, resourceOnly *v1pb.MCPDelegation
	for _, d := range delegated {
		if d.ClientId == "" {
			preGrant = d
		} else {
			resourceOnly = d
		}
	}
	a.NotNil(preGrant, "a pasted web token was never issued to an OAuth client, so its row carries no client")
	a.NotNil(resourceOnly, "the consented grant's row must name the client it was issued to")
	a.Equal(clientID, resourceOnly.ClientId)

	a.Empty(preGrant.Scope, "a pre-grant session has no consented scope")
	a.Empty(preGrant.Resource, "a pre-grant session was never bound to a resource")
	a.NotEmpty(preGrant.CorrelationId, "every MCP-originated row must be correlatable to its session")

	a.Empty(resourceOnly.Scope, "the grant recorded no scope, so the row records none — never a resolved label")
	a.Equal(ctl.rootURL+"/mcp", resourceOnly.Resource,
		"the grant's stored resource travels verbatim — it is the only thing keeping this state out of the pre-grant one")
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

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	workspaceID := strings.TrimPrefix(workspace.Msg.Name, "workspaces/")

	mcpToken, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)

	status, body := postMCP(t, ctl, mcpToken)
	a.Equal(http.StatusOK, status,
		"control: with a readable policy the ceiling admits this session; %s", body)

	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()

	// The stored ceiling is given a value the profile cannot be parsed with, so
	// the read errors outright. That is a distinct arm from an unrecognized
	// enum NAME, which the store's unmarshaler discards: a name it does not
	// know is caught by the raw-key check in GetMCPCapabilityUncached and also
	// fails closed, and TestMCPCeilingStoredValueFailsClosed
	// (backend/api/mcp) owns that case. This one is the unmarshal error itself.
	restore := func() {
		result, err := db.ExecContext(ctx, `
			UPDATE setting SET value = value - 'mcpCapability'
			WHERE workspace = $1 AND name = 'WORKSPACE_PROFILE';
		`, workspaceID)
		a.NoError(err)
		affected, err := result.RowsAffected()
		a.NoError(err)
		a.Equal(int64(1), affected, "the corrupted policy row must be restored")
	}
	result, err := db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{mcpCapability}', 'true')
		WHERE workspace = $1 AND name = 'WORKSPACE_PROFILE';
	`, workspaceID)
	a.NoError(err)
	// Registered before the assertions below so a failing one cannot leave the
	// server running on an unreadable profile through teardown, burying the
	// real failure under unrelated errors.
	restored := false
	t.Cleanup(func() {
		if !restored {
			restore()
		}
	})
	affected, err := result.RowsAffected()
	a.NoError(err)
	a.Equal(int64(1), affected, "the workspace profile row must exist for this test to mean anything")

	status, body = postMCP(t, ctl, mcpToken)
	a.Equal(http.StatusForbidden, status,
		"a ceiling that cannot be read must fail closed, not fall back to permitting MCP; %s", body)
	a.Contains(body, "MCP access is disabled")

	// Restoring the row proves the refusal was the unreadable policy and
	// nothing else: the very same token opens a session again.
	restore()
	restored = true

	session := openMCPSession(ctx, t, ctl, mcpToken)
	defer session.Close()
	// The probe is a READ method deliberately: the MCP gate serves READ and
	// WRITE and refuses everything else, so an EXCLUDED probe would answer 403
	// for a reason that has nothing to do with the ceiling this test is about.
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, session, "WorkspaceService/ListWorkspaces", nil))
}

// TestMCPMigrationTightenedCeilingBitesLiveSession is matrix row 7: tightening
// the workspace ceiling controls entry, so it takes effect on the NEXT request
// of an already-open session — no re-auth, no token re-issue, nothing revoked.
//
// The ceiling is read live on every /mcp request, which is what makes this
// admission control rather than a property of session setup. DISABLED is the
// tightening used here because it is the ceiling that refuses the connection
// itself. Tightening to READ_ONLY bites the same way and at the same moment,
// but a step further in: the session still opens, and what it may then do
// narrows per method and per statement — TestMCPReadOnlyCeilingRefusesAWrite
// drives that half.
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
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, session, "WorkspaceService/ListWorkspaces", nil),
		"the session is live under the default ceiling")

	// An admin tightens the ceiling while the session stays open.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.WorkspaceProfileSetting_DISABLED))

	status, body := postMCP(t, ctl, mcpToken)
	a.Equal(http.StatusForbidden, status,
		"the same bearer must be refused on its next request after the ceiling tightens; %s", body)
	a.Contains(body, "MCP access is disabled")

	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "call_api",
		Arguments: map[string]any{"operationId": "WorkspaceService/ListWorkspaces"},
	})
	// The cause matters, not just the failure: a bare "some error" would go
	// green if the session broke for an unrelated reason while the ceiling
	// exemption regressed.
	a.ErrorContains(err, http.StatusText(http.StatusForbidden),
		"an established session must be stopped BY THE CEILING, not merely fail somehow")

	// Widening again admits a new session, so the refusal was the ceiling
	// rather than damage to the session or the grant.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.WorkspaceProfileSetting_READ_WRITE))
	restored := openMCPSession(ctx, t, ctl, mcpToken)
	defer restored.Close()
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, restored, "WorkspaceService/ListWorkspaces", nil))
}
