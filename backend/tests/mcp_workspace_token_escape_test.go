package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// mcpCallResult is the slice of call_api's structured output these tests care
// about: the status the agent sees, the error text, and any token the response
// body carried.
type mcpCallResult struct {
	Status   int    `json:"status"`
	Error    string `json:"error"`
	Response struct {
		Token string `json:"token"`
	} `json:"response"`
}

// callAPIViaMCP drives one call_api tool call over a live /mcp session bound to
// bearer, exactly as an agent would, and decodes what that agent gets back.
func callAPIViaMCP(ctx context.Context, t *testing.T, ctl *controller, bearer, operationID string, body map[string]any) mcpCallResult {
	t.Helper()
	session := openMCPSession(ctx, t, ctl, bearer)
	defer session.Close()
	return callAPIOnSession(ctx, t, session, operationID, body)
}

// callAPIOnSession is callAPIViaMCP against an already-open session, for the
// tests whose subject is a multi-call chain: one agent session, several tool
// calls, one correlation ID.
func callAPIOnSession(ctx context.Context, t *testing.T, session *mcp.ClientSession, operationID string, body map[string]any) mcpCallResult {
	t.Helper()
	a := require.New(t)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "call_api",
		Arguments: map[string]any{
			"operationId": operationID,
			"body":        body,
		},
	})
	a.NoError(err)

	raw, err := json.Marshal(result.StructuredContent)
	a.NoError(err)
	var out mcpCallResult
	a.NoError(json.Unmarshal(raw, &out))
	// A tool-level failure — a mistyped operationId, a transport error — leaves
	// StructuredContent nil, which decodes to a zero status rather than an
	// error. Without this the test would report a status mismatch that has
	// nothing to do with the guard it exists to check.
	a.NotEqual(0, out.Status, "call_api returned no status: %s", out.Error)
	return out
}

// workspaceHasMember reports whether any binding in the workspace IAM policy
// still names email. Matching on the substring keeps this independent of the
// member-identifier spelling the API happens to use. The probe runs as bearer
// rather than as the subject, so it still answers after the subject has lost
// its membership.
func workspaceHasMember(ctx context.Context, t *testing.T, ctl *controller, bearer, workspace, email string) bool {
	t.Helper()
	previous := ctl.authInterceptor.token
	ctl.authInterceptor.token = bearer
	defer func() { ctl.authInterceptor.token = previous }()

	policy, err := ctl.workspaceServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: workspace,
	}))
	require.NoError(t, err)
	for _, binding := range policy.Msg.Bindings {
		for _, member := range binding.Members {
			if strings.Contains(member, email) {
				return true
			}
		}
	}
	return false
}

// TestMCPCannotLeaveWorkspace pins the sibling of the SwitchWorkspace boundary
// escape that TestMCPCannotMintPlainUserToken covers. LeaveWorkspace ends in
// AuthService.switchWorkspaceInternal, which puts a plain bb.user.access token
// in the response body whenever the caller has no refresh cookie — and an MCP
// session never has one. That token is not audience-bound to the MCP resource,
// survives revocation of the OAuth grant, and ignores the workspace MCP kill
// switch.
//
// The guard has to run before the IAM mutation, not after: leaving the
// workspace and only then refusing the token would strand the caller. That
// ordering is what the membership assertion below pins — with the guard
// removed this test fails twice over, on the status and on the destroyed
// membership.
//
// Since the FORBIDDEN gate landed, the refusal an MCP session actually meets
// comes from the interceptor, ahead of the handler — so this test no longer
// discriminates on WHICH layer refused, only that the session is refused and
// the membership survives. The handler guard keeps its own coverage in
// backend/api/v1: TestLeaveAndDeleteWorkspaceRefuseMCPCaller pins that each
// handler refuses before it touches the store, and
// TestSwitchWorkspaceInternalRefusesMCPCaller pins the shared mint point.
func TestMCPCannotLeaveWorkspace(t *testing.T) {
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
	workspaceName := workspace.Msg.Name

	// LeaveWorkspace refuses the last admin. Seating a second admin is what
	// makes the guard — rather than that precondition — the reason the MCP
	// call fails: with the guard removed the leave goes through.
	const secondAdminEmail = "second-admin@example.com"
	const secondAdminPassword = "1024bytebase"
	_, err = ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    secondAdminEmail,
			Password: secondAdminPassword,
			Title:    "second admin",
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, workspaceName, "user:"+secondAdminEmail, "roles/workspaceAdmin")
	a.NoError(err)

	// The second admin's own session probes the IAM policy: it survives the
	// unguarded outcome, where the caller has just deleted its own membership
	// and can no longer read anything.
	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    secondAdminEmail,
		Password: secondAdminPassword,
	}))
	a.NoError(err)
	secondAdminToken := loginResp.Msg.Token
	demoToken := ctl.authInterceptor.token

	a.True(workspaceHasMember(ctx, t, ctl, secondAdminToken, workspaceName, "demo@example.com"),
		"precondition: the calling user is a member before the MCP attempt")

	out := callAPIViaMCP(ctx, t, ctl, demoToken,
		"WorkspaceService/LeaveWorkspace", map[string]any{"name": workspaceName})
	stillMember := workspaceHasMember(ctx, t, ctl, secondAdminToken, workspaceName, "demo@example.com")
	t.Logf("MCP LeaveWorkspace → status=%d error=%q token=%q stillMember=%v",
		out.Status, out.Error, out.Response.Token, stillMember)

	a.Equal(http.StatusForbidden, out.Status,
		"an MCP session must be refused before it can leave a workspace")
	a.Contains(out.Error, "not available to MCP sessions",
		"the refusal must come from the MCP guard, not from some other precondition")
	// This fixture has a single workspace, so switchWorkspaceInternal's nextWS
	// is always nil and the mint is out of reach even unguarded: the assertion
	// states the invariant but cannot fail here. The load-bearing pins are the
	// status and the membership below.
	a.Empty(out.Response.Token,
		"an MCP session must never receive a plain user access token")
	a.True(stillMember,
		"the guard must run before the IAM mutation — the caller must still be a member")

	// The guard must not cost a normal caller the ability to leave. The second
	// admin holds an ordinary session token, so this is the non-MCP path.
	ctl.authInterceptor.token = secondAdminToken
	_, err = ctl.workspaceServiceClient.LeaveWorkspace(ctx, connect.NewRequest(&v1pb.LeaveWorkspaceRequest{
		Name: workspaceName,
	}))
	ctl.authInterceptor.token = demoToken
	a.NoError(err, "a normal session must still be able to leave the workspace")
	a.False(workspaceHasMember(ctx, t, ctl, demoToken, workspaceName, secondAdminEmail),
		"the normal leave must actually remove the caller's bindings")
}

// TestMCPCannotDeleteWorkspace is the DeleteWorkspace half of the same escape.
// DeleteWorkspace ends in the same switchWorkspaceInternal token mint, and the
// refusal sits ahead of every check in the handler — including the SaaS-only
// precondition. That placement is what the assertion pins: this build is
// self-hosted, so with both the gate and the handler guard removed the MCP call
// reaches the SaaS check and comes back FailedPrecondition rather than
// PermissionDenied.
func TestMCPCannotDeleteWorkspace(t *testing.T) {
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
	workspaceName := workspace.Msg.Name

	out := callAPIViaMCP(ctx, t, ctl, ctl.authInterceptor.token,
		"WorkspaceService/DeleteWorkspace", map[string]any{"name": workspaceName})
	t.Logf("MCP DeleteWorkspace → status=%d error=%q token=%q",
		out.Status, out.Error, out.Response.Token)

	a.Equal(http.StatusForbidden, out.Status,
		"an MCP session must be refused before any deletion work happens")
	a.Contains(out.Error, "not available to MCP sessions",
		"the refusal must come from the MCP guard, ahead of the SaaS precondition")
	// As in the leave test, the mint is unreachable on a self-hosted fixture,
	// so this states the invariant rather than discriminating on it.
	a.Empty(out.Response.Token,
		"an MCP session must never receive a plain user access token")

	// A normal caller is not intercepted by the guard: it reaches the handler
	// and is turned away by the deployment-mode check instead.
	_, err = ctl.workspaceServiceClient.DeleteWorkspace(ctx, connect.NewRequest(&v1pb.DeleteWorkspaceRequest{
		Name: workspaceName,
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err),
		"a normal session must reach the handler, not the MCP guard")
}

// TestMCPOAuthSessionCannotLeaveOrDeleteWorkspace covers the other admission
// path. The two tests above ride the legacy admission — a plain web-session
// token pasted into an MCP client — while this one runs the flow a real MCP
// client performs: RFC 7591 dynamic client registration, PKCE consent, code
// exchange. Both admissions reach these handlers, so both have to be refused.
//
// The grant here consents to mcp:read-only and still reaches two mutating
// handlers, because nothing enforces the consented scope yet (see
// common.DelegatedGrant — P1b's capability gate is the intended consumer). A
// read-only grant is therefore no substitute for this guard.
func TestMCPOAuthSessionCannotLeaveOrDeleteWorkspace(t *testing.T) {
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

	oauthToken, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)

	for _, operation := range []string{
		"WorkspaceService/LeaveWorkspace",
		"WorkspaceService/DeleteWorkspace",
	} {
		out := callAPIViaMCP(ctx, t, ctl, oauthToken, operation,
			map[string]any{"name": workspace.Msg.Name})
		t.Logf("MCP OAuth %s → status=%d error=%q", operation, out.Status, out.Error)

		a.Equal(http.StatusForbidden, out.Status,
			"a real MCP OAuth session must be refused by %s", operation)
		a.Contains(out.Error, "not available to MCP sessions",
			"the refusal must come from the MCP guard")
		a.Empty(out.Response.Token,
			"an MCP session must never receive a plain user access token")
	}
}
