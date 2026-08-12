package tests

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestMCPCannotChangeOwnPasswordAndLogIn runs the escape end to end. The chain
// is two ordinary call_api invocations:
//
//  1. UserService/UpdateUser{password}. The self-update branch authorizes on
//     caller == subject alone — no permission check, no proof of the old
//     password — so an MCP session could rewrite its own user's password.
//  2. AuthService/Login{web:false}. Login is allow_without_credential and the
//     non-web branch returns the token in the response body.
//
// The result is a plain bb.user.access token: not audience-bound to the MCP
// resource, alive after the OAuth grant is revoked, unaffected by the workspace
// MCP kill switch — plus a hijacked account, since the human's password is now
// the agent's. Both steps are FORBIDDEN on the internal chain.
//
// The RED state is the whole point of the assertions below. With the
// interceptor unregistered, step 1 answers 200, step 2 hands back a token, and
// the old-password login fails because the password really did change.
func TestMCPCannotChangeOwnPasswordAndLogIn(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// A dedicated end user drives the session: the escape is about an MCP
	// session rewriting its OWN credentials, and using the fixture's admin
	// would leave the rest of the suite's login helpers holding a stale
	// password if a regression let the change through.
	const agentEmail = "mcp-agent@example.com"
	const oldPassword = "1024bytebase"
	const hijackPassword = "2048bytebase"
	created, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "mcp agent",
			Email:    agentEmail,
			Password: oldPassword,
		},
	}))
	a.NoError(err)
	workspace := created.Msg.Workspace
	a.NotEmpty(workspace)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, workspace, "user:"+agentEmail, "roles/workspaceMember")
	a.NoError(err)

	login, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    agentEmail,
		Password: oldPassword,
	}))
	a.NoError(err)
	agentToken := login.Msg.Token
	a.NotEmpty(agentToken)

	// One session for both steps, the way an agent would run the chain.
	session := openMCPSession(ctx, t, ctl, agentToken)
	defer session.Close()

	changed := callAPIOnSession(ctx, t, session, "UserService/UpdateUser", map[string]any{
		"user":       map[string]any{"name": "users/" + agentEmail, "password": hijackPassword},
		"updateMask": "password",
	})
	// Step 2 runs whatever step 1 answered: it is the half that actually
	// escapes the token boundary, and it must be refused on its own. Running it
	// unconditionally — and logging both steps before asserting either — is
	// what makes an unguarded build report the whole escape (200, then a token)
	// instead of stopping at the first assertion.
	minted := callAPIOnSession(ctx, t, session, "AuthService/Login", map[string]any{
		"email":    agentEmail,
		"password": hijackPassword,
		"web":      false,
	})
	t.Logf("MCP UpdateUser{password} → status=%d error=%q", changed.Status, changed.Error)
	t.Logf("MCP Login{web:false} → status=%d error=%q token=%q", minted.Status, minted.Error, minted.Response.Token)

	a.Equal(http.StatusForbidden, changed.Status,
		"an MCP session must not be able to rewrite its own user's password")
	a.Contains(changed.Error, "not available to MCP sessions",
		"the refusal must come from the FORBIDDEN gate, not from password-complexity validation")
	a.Equal(http.StatusForbidden, minted.Status, "Login must be refused on the internal chain")
	a.Contains(minted.Error, "not available to MCP sessions")
	a.Empty(minted.Response.Token, "an MCP session must never receive a plain user access token")

	// Login with the credentials the session tried to set must also fail, so
	// the refusal is not the only thing standing between the agent and the
	// account.
	loggedInAsAgent := callAPIOnSession(ctx, t, session, "AuthService/Login", map[string]any{
		"email":    agentEmail,
		"password": oldPassword,
		"web":      false,
	})
	a.Equal(http.StatusForbidden, loggedInAsAgent.Status,
		"Login is refused whatever credentials it names")
	a.Empty(loggedInAsAgent.Response.Token)

	// The account is untouched: the old password still works and the one the
	// session tried to install does not. This is what discriminates a gate that
	// refuses before dispatch from one that refuses after the store write.
	_, err = ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    agentEmail,
		Password: hijackPassword,
	}))
	a.Error(err, "the password the MCP session tried to install must not work")
	a.Equal(connect.CodeUnauthenticated, connect.CodeOf(err))

	stillOld, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    agentEmail,
		Password: oldPassword,
	}))
	a.NoError(err, "the user's real password must still be their real password")
	a.NotEmpty(stillOld.Msg.Token)

	// The operator's view: a denied MCP call is exactly the event worth
	// investigating, so both denials must be on the audit page — including
	// Login, which the handler never reached.
	deniedRows := func(method string) []*v1pb.AuditLog {
		resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:  workspace,
			Filter:  `method == "` + method + `"`,
			OrderBy: "create_time desc",
		}))
		a.NoError(err)
		var rows []*v1pb.AuditLog
		for _, row := range resp.Msg.AuditLogs {
			if row.User == "users/"+agentEmail && row.McpDelegation != nil {
				rows = append(rows, row)
			}
		}
		return rows
	}

	updateRows := deniedRows("/bytebase.v1.UserService/UpdateUser")
	a.Len(updateRows, 1, "the denied password change must produce exactly one audit row")
	a.Equal(int32(connect.CodePermissionDenied), updateRows[0].Status.GetCode(),
		"the row must record the denial, not a success")
	a.NotEmpty(updateRows[0].McpDelegation.GetCorrelationId(),
		"the denial must be correlatable back to the agent session that made it")

	loginRows := deniedRows("/bytebase.v1.AuthService/Login")
	a.Len(loginRows, 2, "both refused Login attempts must be audited")
	for _, row := range loginRows {
		a.Equal(int32(connect.CodePermissionDenied), row.Status.GetCode())
	}
	a.Equal(updateRows[0].McpDelegation.GetCorrelationId(), loginRows[0].McpDelegation.GetCorrelationId(),
		"one session, one correlation ID across the whole chain")
}

// TestWebUserStillChangesPasswordAndLogsIn is the regression half: the gate
// lives on the internal MCP chain only, so the ordinary self-service flow — a
// signed-in user changing their own password in the console, then logging in
// with it — must be untouched. The same two methods, the same principal, the
// public chain.
func TestWebUserStillChangesPasswordAndLogsIn(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	const webEmail = "web-user@example.com"
	const oldPassword = "1024bytebase"
	const newPassword = "2048bytebase"
	created, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "web user",
			Email:    webEmail,
			Password: oldPassword,
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, created.Msg.Workspace, "user:"+webEmail, "roles/workspaceMember")
	a.NoError(err)

	login, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    webEmail,
		Password: oldPassword,
	}))
	a.NoError(err)

	// The user's own session changes their own password, exactly as the
	// console does.
	admin := ctl.authInterceptor.token
	ctl.authInterceptor.token = login.Msg.Token
	_, err = ctl.userServiceClient.UpdateUser(ctx, connect.NewRequest(&v1pb.UpdateUserRequest{
		User:       &v1pb.User{Name: "users/" + webEmail, Password: newPassword},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"password"}},
	}))
	ctl.authInterceptor.token = admin
	a.NoError(err, "a normal session must still be able to change its own password")

	relogin, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    webEmail,
		Password: newPassword,
	}))
	a.NoError(err, "the new password must work on the public login path")
	a.NotEmpty(relogin.Msg.Token)
}
