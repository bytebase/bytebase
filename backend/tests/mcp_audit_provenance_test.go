package tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestMCPAuditProvenance is the operator-story e2e: a workspace member's MCP
// session — real OAuth grant, real /mcp session — makes permitted and refused
// calls. The permitted call is stored and read back through the v1 audit-log
// API; the refused ones are never stored and are read from the stdout stream.
// Both carry the delegation provenance (grant scope and resource verbatim,
// client ID, session correlation ID).
//
// Not parallel: it captures slog.Default.
func TestMCPAuditProvenance(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	lines := captureAuditStream(t)
	ctl, ctx := startWorkspace(ctx, t)
	ctl.profile.RuntimeEnableAuditLogStdout.Store(true)

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

	// CreateSheet below is a WRITE method. Pinned so this test does not depend
	// on the resolved default; it is about what an audit row carries.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))

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

	// The permitted call has to be one the MCP ceiling serves AND the member is
	// allowed, and no audited READ or WRITE method is reachable on a plain
	// workspace membership alone — every one of them is project-scoped. So the
	// member owns a project. (Creating a group was the natural probe and is
	// EXCLUDED from the ceiling as workspace administration; self-updating the
	// member's title reads even better and is FORBIDDEN — see
	// backend/api/v1/mcp_gate.go.)
	const ownedProjectID = "mcp-audit-owned"
	ownedProject, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: ownedProjectID,
		Project:   &v1pb.Project{Title: "MCP audit owned"},
	}))
	a.NoError(err)
	ownedProjectName := ownedProject.Msg.Name
	ownedPolicy, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ownedProjectName,
	}))
	a.NoError(err)
	policy := ownedPolicy.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/projectOwner",
		Members: []string{"user:" + memberEmail},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: ownedProjectName,
		Policy:   policy,
	}))
	a.NoError(err)

	// A second project the member has no role in, so the denial below is the
	// ACL's and lands on a resource the ACL validated.
	const otherProjectID = "mcp-audit-other"
	otherProject, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: otherProjectID,
		Project:   &v1pb.Project{Title: "MCP audit other"},
	}))
	a.NoError(err)
	otherProjectName := otherProject.Msg.Name

	// Permitted audited call: creating a sheet needs bb.sheets.create, which
	// the member holds on the project it owns.
	a.Equal(http.StatusOK, callAPI("SheetService/CreateSheet", map[string]any{
		"parent": ownedProjectName,
		"sheet":  map[string]any{"content": base64.StdEncoding.EncodeToString([]byte("SELECT 1;"))},
	}))

	// Refused audited call: CreateUser is FORBIDDEN to MCP sessions, so the
	// ceiling gate refuses it before ACL, and its line is parented to the
	// caller's workspace.
	a.Equal(http.StatusForbidden, callAPI("UserService/CreateUser", map[string]any{
		"user": map[string]any{"email": "sneaky@example.com", "title": "sneaky", "password": memberPassword},
	}))

	// The operator's view, attributed to the member.
	memberLine := func(method string) map[string]any {
		var matched []map[string]any
		for _, line := range lines() {
			if line["method"] == method && line["user"] == "users/"+memberEmail {
				matched = append(matched, line)
			}
		}
		a.Len(matched, 1, "the refused %s must be streamed once", method)
		return matched[0]
	}
	searchMemberRows := func(parent, method string) []*v1pb.AuditLog {
		resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:  parent,
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

	permittedRows := searchMemberRows(ownedProjectName, "/bytebase.v1.SheetService/CreateSheet")
	a.Len(permittedRows, 1, "the permitted MCP call must produce exactly one audit row")
	permitted := permittedRows[0]
	a.Nil(permitted.Status, "the permitted call's row keeps its success status")
	a.NotNil(permitted.McpDelegation, "the permitted row must be marked MCP-originated")
	a.Equal("mcp:read-only", permitted.McpDelegation.Scope, "the grant scope must be recorded verbatim")
	a.Equal(ctl.rootURL+"/mcp", permitted.McpDelegation.Resource, "the grant resource must be recorded verbatim")
	a.Equal(clientID, permitted.McpDelegation.ClientId)
	a.NotEmpty(permitted.McpDelegation.CorrelationId)

	a.Empty(searchMemberRows(workspace, "/bytebase.v1.UserService/CreateUser"), "a gate refusal is never stored")
	denied := memberLine("/bytebase.v1.UserService/CreateUser")
	a.Equal(workspace, denied["parent"])
	a.Equal(v1pb.AuditLog_WARNING.String(), denied["severity"])
	a.InDelta(float64(connect.CodePermissionDenied), denied["status_code"], 0)
	a.Equal(true, denied["mcp"])
	a.Equal("mcp:read-only", denied["mcp_scope"])
	a.Equal(ctl.rootURL+"/mcp", denied["mcp_resource"])
	a.Equal(clientID, denied["mcp_client_id"])
	a.Equal(permitted.McpDelegation.CorrelationId, denied["mcp_correlation_id"],
		"one MCP session carries one correlation ID across all of its tool calls")

	// A refusal on a project-scoped resource keeps its true project parent,
	// because ACL publishes the resources it validated before the IAM verdict.
	// The probe is a WRITE method deliberately: the ceiling gate runs before
	// ACL, so a method the ceiling refuses falls back to the caller's workspace
	// and would prove nothing about parents.
	a.Equal(http.StatusForbidden, callAPI("PlanService/CreatePlan", map[string]any{
		"parent": otherProjectName,
		"plan":   map[string]any{"title": "not mine to make"},
	}))
	projectDenied, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: otherProjectName,
		Filter: `method == "/bytebase.v1.PlanService/CreatePlan"`,
	}))
	a.NoError(err)
	a.Empty(projectDenied.Msg.AuditLogs, "an ACL refusal is never stored")
	projectLine := memberLine("/bytebase.v1.PlanService/CreatePlan")
	a.Equal(otherProjectName, projectLine["parent"])
	a.Equal(v1pb.AuditLog_WARNING.String(), projectLine["severity"])
	a.InDelta(float64(connect.CodePermissionDenied), projectLine["status_code"], 0)
	a.Equal(permitted.McpDelegation.CorrelationId, projectLine["mcp_correlation_id"])

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
