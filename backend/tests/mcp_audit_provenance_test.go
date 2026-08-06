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

// TestMCPAuditProvenance is the P1a PR 5b operator-story e2e: a workspace
// member's MCP session — real OAuth grant, real /mcp session — makes one
// permitted call and one call the ACL interceptor denies, and an operator
// investigating the agent sees BOTH through the v1 audit-log API, each row
// carrying the delegation provenance (grant scope and resource verbatim,
// client ID, session correlation ID), the denial with its denied status.
// Before this PR the denied call vanished entirely (ACL returned before the
// audit interceptor on the internal chain) and no row carried MCP provenance.
func TestMCPAuditProvenance(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// A plain workspace member drives the MCP session, so an IAM-gated admin
	// action is genuinely denied by the ACL interceptor, not the handler.
	const memberEmail = "agent-driver@example.com"
	const memberPassword = "1024bytebase"
	memberResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "agent driver",
			Email:    memberEmail,
			Password: memberPassword,
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, memberResp.Msg.Workspace, "user:"+memberEmail, "roles/workspaceMember")
	a.NoError(err)

	memberLogin, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    memberEmail,
		Password: memberPassword,
	}))
	a.NoError(err)
	workspace := memberLogin.Msg.GetUser().GetWorkspace()
	a.NotEmpty(workspace)

	// The member consents a real OAuth grant; every audit row the session
	// produces must carry this grant's stored scope and resource verbatim.
	mcpToken, clientID := mintMCPOAuthToken(t, ctl, memberLogin.Msg.Token)

	client := mcp.NewClient(&mcp.Implementation{Name: "bb-e2e", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   ctl.rootURL + "/mcp",
		HTTPClient: &http.Client{Transport: &bearerTransport{token: mcpToken}},
	}, nil)
	a.NoError(err)
	defer session.Close()

	callAPI := func(operationID string, body map[string]any) int {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "call_api",
			Arguments: map[string]any{"operationId": operationID, "body": body},
		})
		a.NoError(err)
		raw, err := json.Marshal(result.StructuredContent)
		a.NoError(err)
		var out struct {
			Status int    `json:"status"`
			Error  string `json:"error"`
		}
		a.NoError(json.Unmarshal(raw, &out))
		return out.Status
	}

	// Permitted audited call: the member updates their own title.
	a.Equal(http.StatusOK, callAPI("UserService/UpdateUser", map[string]any{
		"user":       map[string]any{"name": "users/" + memberEmail, "title": "agent driver (updated)"},
		"updateMask": "title",
	}))

	// Denied audited call: creating a user needs bb.users.create, which a
	// workspace member does not hold — the ACL interceptor refuses it.
	a.Equal(http.StatusForbidden, callAPI("UserService/CreateUser", map[string]any{
		"user": map[string]any{"email": "sneaky@example.com", "title": "sneaky", "password": memberPassword},
	}))

	// The operator's view: both calls surface through the v1 audit-log read
	// API, attributed to the member.
	searchMemberRows := func(method string) []*v1pb.AuditLog {
		resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:  workspace,
			Filter:  `method == "` + method + `"`,
			OrderBy: "create_time desc",
		}))
		a.NoError(err)
		var rows []*v1pb.AuditLog
		for _, l := range resp.Msg.AuditLogs {
			if l.User == "users/"+memberEmail {
				rows = append(rows, l)
			}
		}
		return rows
	}

	permittedRows := searchMemberRows("/bytebase.v1.UserService/UpdateUser")
	a.Len(permittedRows, 1, "the permitted MCP call must produce exactly one audit row")
	permitted := permittedRows[0]
	a.Nil(permitted.Status, "the permitted call's row keeps its success status")
	a.NotNil(permitted.McpDelegation, "the permitted row must be marked MCP-originated")
	a.Equal("mcp:read-only", permitted.McpDelegation.Scope, "the grant scope must be recorded verbatim")
	a.Equal(ctl.rootURL+"/mcp", permitted.McpDelegation.Resource, "the grant resource must be recorded verbatim")
	a.Equal(clientID, permitted.McpDelegation.ClientId)
	a.NotEmpty(permitted.McpDelegation.CorrelationId)

	deniedRows := searchMemberRows("/bytebase.v1.UserService/CreateUser")
	a.Len(deniedRows, 1, "an ACL-denied MCP call must still produce an audit row — it is exactly the event an operator investigating an agent needs")
	denied := deniedRows[0]
	a.NotNil(denied.Status, "the denied row must reflect the denial")
	a.Equal(int32(connect.CodePermissionDenied), denied.Status.Code)
	a.NotNil(denied.McpDelegation)
	a.Equal("mcp:read-only", denied.McpDelegation.Scope)
	a.Equal(ctl.rootURL+"/mcp", denied.McpDelegation.Resource)
	a.Equal(clientID, denied.McpDelegation.ClientId)
	a.Equal(permitted.McpDelegation.CorrelationId, denied.McpDelegation.CorrelationId,
		"one MCP session carries one correlation ID across all of its tool calls")

	// A denial on a project-scoped resource keeps its true project parent —
	// the denied probe must show up for that project's auditors. (Only
	// UNVALIDATED resources — the workspace-mismatch arm — fall back to the
	// caller's workspace; an IAM denial's resources passed workspace-scoped
	// validation.)
	projects, err := ctl.projectServiceClient.ListProjects(ctx, connect.NewRequest(&v1pb.ListProjectsRequest{}))
	a.NoError(err)
	a.NotEmpty(projects.Msg.Projects)
	projectName := projects.Msg.Projects[0].Name
	a.Equal(http.StatusForbidden, callAPI("ProjectService/SetIamPolicy", map[string]any{
		"resource": projectName,
		"policy":   map[string]any{"bindings": []any{}},
	}))
	projectDenied, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: projectName,
		Filter: `method == "/bytebase.v1.ProjectService/SetIamPolicy"`,
	}))
	a.NoError(err)
	a.Len(projectDenied.Msg.AuditLogs, 1, "the project-scoped denial must be audited under the project it targeted")
	projectRow := projectDenied.Msg.AuditLogs[0]
	a.True(strings.HasPrefix(projectRow.Name, projectName+"/auditLogs/"))
	a.Equal("users/"+memberEmail, projectRow.User)
	a.Equal(int32(connect.CodePermissionDenied), projectRow.Status.GetCode())
	a.Equal(permitted.McpDelegation.CorrelationId, projectRow.McpDelegation.GetCorrelationId())

	// Public-chain rows are untouched: the admin's direct v1 CreateUser (the
	// member's own creation, same audited method as the denial) carries no
	// MCP fields.
	adminRows, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.UserService/CreateUser"`,
	}))
	a.NoError(err)
	var adminCreates int
	for _, row := range adminRows.Msg.AuditLogs {
		if row.User == ctl.principalName {
			adminCreates++
			a.Nil(row.McpDelegation, "a public-chain row must never carry MCP provenance")
		}
	}
	a.Positive(adminCreates, "the admin's public-chain CreateUser must have been audited")
}
