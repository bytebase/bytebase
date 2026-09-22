package tests

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// auditSinkProbe is one call in TestAuditSinksOnBothChains and where it must
// land.
type auditSinkProbe struct {
	name   string
	method string
	member bool // the caller is the workspace member, not the admin
	mcp    bool
	// parent is where the row or line is filed.
	parent string
	// match picks the probe's own row or line apart from other calls to the
	// same method by the same caller.
	match     func(request string) bool
	call      func(t *testing.T)
	checkRow  func(t *testing.T, row *v1pb.AuditLog)
	checkLine func(t *testing.T, line map[string]any)
	// wantRow is whether the call is stored; wantWarning is whether a
	// permission check refused it. A line follows from the two: stdout carries
	// every stored row and every refusal.
	wantRow     bool
	wantWarning bool
}

// TestAuditSinksOnBothChains is the store and stream rule on a booted server,
// through both interceptor chains, with stdout audit on and off:
//
//	store  = audited method and the call reached its handler
//	stream = stdout on and (stored or refused by a permission check)
//
// Not parallel: it captures slog.Default.
func TestAuditSinksOnBothChains(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	lines := captureAuditStream(t)
	ctl, ctx := startWorkspace(ctx, t)

	const memberEmail = "audit-sinks-member@example.com"
	const memberPassword = "1024bytebase"
	created, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Title: "audit sinks member", Email: memberEmail, Password: memberPassword},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, created.Msg.Workspace, "user:"+memberEmail, "roles/workspaceMember")
	a.NoError(err)
	login, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email: memberEmail, Password: memberPassword,
	}))
	a.NoError(err)
	workspace := login.Msg.GetUser().GetWorkspace()
	a.NotEmpty(workspace)
	adminUser, memberUser := ctl.principalName, "users/"+memberEmail
	// The member holds no role on this project.
	project := ctl.project.Name

	memberAuth := connect.WithInterceptors(&authInterceptor{token: login.Msg.Token})
	memberProjects := v1connect.NewProjectServiceClient(ctl.client, ctl.rootURL, memberAuth)
	memberSettings := v1connect.NewSettingServiceClient(ctl.client, ctl.rootURL, memberAuth)
	memberIssues := v1connect.NewIssueServiceClient(ctl.client, ctl.rootURL, memberAuth)
	memberGroups := v1connect.NewDatabaseGroupServiceClient(ctl.client, ctl.rootURL, memberAuth)
	memberSQL := v1connect.NewSQLServiceClient(ctl.client, ctl.rootURL, memberAuth)

	// grantMemberProjectRole binds the member to one project role and returns
	// the undo, so a probe that needs the member past the ACL leaves the rest
	// looking at a member who holds no role.
	grantMemberProjectRole := func(t *testing.T, role string) func() {
		t.Helper()
		set := func(bindings []*v1pb.Binding) {
			policy, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{Resource: project}))
			require.NoError(t, err)
			var kept []*v1pb.Binding
			for _, binding := range policy.Msg.Bindings {
				var members []string
				for _, member := range binding.Members {
					if !strings.Contains(member, memberEmail) {
						members = append(members, member)
					}
				}
				if len(members) > 0 {
					binding.Members = members
					kept = append(kept, binding)
				}
			}
			kept = append(kept, bindings...)
			policy.Msg.Bindings = kept
			_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
				Resource: project, Policy: policy.Msg,
			}))
			require.NoError(t, err)
		}
		set([]*v1pb.Binding{{Role: role, Members: []string{"user:" + memberEmail}}})
		return func() { set(nil) }
	}

	// The clamp fixture leaves the ceiling at READ_ONLY; every other MCP probe
	// runs under READ_WRITE.
	clamp := setupMCPClampFixture(ctx, t, ctl)
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	adminSession := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer adminSession.Close()
	memberToken, _ := mintMCPOAuthToken(t, ctl, login.Msg.Token)
	memberSession := openMCPSession(ctx, t, ctl, memberToken)
	defer memberSession.Close()

	// The issueless-rollout guard needs a plan with no issue. A refused
	// CreateRollout creates nothing, so one plan serves both stdout runs.
	guardSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project,
		Sheet:  &v1pb.Sheet{Content: []byte("CREATE TABLE audit_sinks_guard(id INT);")},
	}))
	a.NoError(err)
	guardPlan, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project,
		Plan: &v1pb.Plan{Specs: []*v1pb.Plan_Spec{{
			Id: uuid.NewString(),
			Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
					Targets: []string{clamp.database},
					Sheet:   guardSheet.Msg.Name,
				},
			},
		}}},
	}))
	a.NoError(err)
	setRequireIssueApproval := func(t *testing.T, on bool) {
		t.Helper()
		ctl.project.RequireIssueApproval = on
		updated, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
			Project:    ctl.project,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"require_issue_approval"}},
		}))
		require.NoError(t, err)
		require.Equal(t, on, updated.Msg.RequireIssueApproval)
	}

	requireCode := func(t *testing.T, want connect.Code, err error) {
		t.Helper()
		if want == 0 {
			require.NoError(t, err)
			return
		}
		require.Equal(t, want, connect.CodeOf(err), "%v", err)
	}
	databaseGroup := func(id string) *v1pb.CreateDatabaseGroupRequest {
		return &v1pb.CreateDatabaseGroupRequest{
			Parent:          project,
			DatabaseGroupId: id,
			DatabaseGroup:   &v1pb.DatabaseGroup{Title: id, DatabaseExpr: &expr.Expr{Expression: "true"}},
			ValidateOnly:    true,
		}
	}
	mcpCall := func(session *mcp.ClientSession, operation string, body map[string]any, wantStatus int) func(t *testing.T) {
		return func(t *testing.T) {
			out := callAPIOnSession(ctx, t, session, operation, body)
			require.Equal(t, wantStatus, out.Status, out.Error)
		}
	}

	probes := []auditSinkProbe{
		{
			name: "public/audited ok", method: v1connect.ProjectServiceUpdateProjectProcedure, parent: project,
			wantRow: true,
			call: func(t *testing.T) {
				_, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
					Project:    &v1pb.Project{Name: project, Title: "audit sinks"},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
				}))
				requireCode(t, 0, err)
			},
		},
		{
			name: "public/audited refused in handler", method: v1connect.SettingServiceUpdateSettingProcedure, member: true, parent: workspace,
			wantRow: true, wantWarning: true,
			call: func(t *testing.T) {
				_, err := memberSettings.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
					Setting: &v1pb.Setting{Name: "settings/AI"},
				}))
				requireCode(t, connect.CodePermissionDenied, err)
			},
		},
		{
			name: "public/audited refused by ACL", method: v1connect.ProjectServiceUpdateProjectProcedure, member: true, parent: project,
			wantWarning: true,
			call: func(t *testing.T) {
				_, err := memberProjects.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
					Project:    &v1pb.Project{Name: project, Title: "not mine"},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
				}))
				requireCode(t, connect.CodePermissionDenied, err)
			},
		},
		{
			name: "public/unaudited refused in handler", method: v1connect.IssueServiceSearchIssuesProcedure, member: true, parent: workspace,
			wantWarning: true,
			call: func(t *testing.T) {
				_, err := memberIssues.SearchIssues(ctx, connect.NewRequest(&v1pb.SearchIssuesRequest{Parent: project}))
				requireCode(t, connect.CodePermissionDenied, err)
			},
		},
		{
			name: "public/unaudited refused by ACL", method: v1connect.ProjectServiceGetProjectProcedure, member: true, parent: project,
			wantWarning: true,
			call: func(t *testing.T) {
				_, err := memberProjects.GetProject(ctx, connect.NewRequest(&v1pb.GetProjectRequest{Name: project}))
				requireCode(t, connect.CodePermissionDenied, err)
			},
		},
		{
			name: "public/unaudited ok", method: v1connect.ProjectServiceGetProjectProcedure, parent: project,
			call: func(t *testing.T) {
				_, err := ctl.projectServiceClient.GetProject(ctx, connect.NewRequest(&v1pb.GetProjectRequest{Name: project}))
				requireCode(t, 0, err)
			},
		},
		{
			name: "public/audited validate-only ok", method: v1connect.DatabaseGroupServiceCreateDatabaseGroupProcedure, parent: project,
			call: func(t *testing.T) {
				_, err := ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(databaseGroup("sinks-admin")))
				requireCode(t, 0, err)
			},
		},
		{
			name: "public/audited validate-only refused by ACL", method: v1connect.DatabaseGroupServiceCreateDatabaseGroupProcedure, member: true, parent: project,
			wantWarning: true,
			call: func(t *testing.T) {
				_, err := memberGroups.CreateDatabaseGroup(ctx, connect.NewRequest(databaseGroup("sinks-member")))
				requireCode(t, connect.CodePermissionDenied, err)
			},
		},
		{
			name: "public/audit log search", method: v1connect.AuditLogServiceSearchAuditLogsProcedure, parent: workspace,
			match:   func(request string) bool { return strings.Contains(request, `"pageSize":7`) },
			wantRow: true,
			call: func(t *testing.T) {
				resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
					Parent: workspace, PageSize: 7,
				}))
				requireCode(t, 0, err)
				require.NotEmpty(t, resp.Msg.AuditLogs)
			},
			checkRow: func(t *testing.T, row *v1pb.AuditLog) {
				require.NotContains(t, row.Response, "auditLogs", "a search's row must not copy the rows it read")
				require.NotEmpty(t, row.Response, "the response was written, minus the rows it read")
			},
		},
		{
			name: "mcp/audited ok", method: v1connect.SheetServiceCreateSheetProcedure, mcp: true, parent: project,
			wantRow: true,
			call: mcpCall(adminSession, "SheetService/CreateSheet", map[string]any{
				"parent": project,
				"sheet":  map[string]any{"content": base64.StdEncoding.EncodeToString([]byte("SELECT 1;"))},
			}, http.StatusOK),
		},
		{
			name: "mcp/audited refused by the clamp in the handler", method: v1connect.SQLServiceQueryProcedure, mcp: true, parent: project,
			wantRow: true, wantWarning: true,
			call: func(t *testing.T) {
				require.NoError(t, ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY))
				defer func() { require.NoError(t, ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE)) }()
				write := queryDatabaseOnSession(ctx, t, clamp.session, clamp.name, "INSERT INTO employee VALUES (2, 'agent')")
				require.True(t, write.isError, write.text)
				require.Contains(t, write.text, "READ_ONLY")
			},
		},
		{
			name: "mcp/audited refused by the grant-issue guard in the handler", method: v1connect.IssueServiceUpdateIssueProcedure, mcp: true, parent: project,
			wantRow: true, wantWarning: true,
			call: func(t *testing.T) {
				out := callAPIOnSession(ctx, t, adminSession, "IssueService/UpdateIssue", map[string]any{
					"issue": map[string]any{
						"name":  project + "/issues/999999",
						"title": "grant me project owner",
						"type":  "ROLE_GRANT",
						"roleGrant": map[string]any{
							"role": "roles/projectOwner",
							"user": "users/demo@example.com",
						},
					},
					"updateMask":   "title",
					"allowMissing": true,
				})
				require.Equal(t, http.StatusForbidden, out.Status, out.Error)
				require.Contains(t, out.Error, "ROLE_GRANT")
			},
		},
		{
			name: "mcp/audited refused by the issueless-rollout guard in the handler", method: v1connect.RolloutServiceCreateRolloutProcedure, mcp: true, parent: project,
			wantRow: true, wantWarning: true,
			call: func(t *testing.T) {
				setRequireIssueApproval(t, true)
				defer setRequireIssueApproval(t, false)
				out := callAPIOnSession(ctx, t, adminSession, "RolloutService/CreateRollout", map[string]any{
					"parent": guardPlan.Msg.Name,
				})
				require.Equal(t, http.StatusForbidden, out.Status, out.Error)
				require.Contains(t, out.Error, "plan with no issue")
			},
		},
		{
			name: "public/audited refused by the SQL Editor access check", method: v1connect.SQLServiceQueryProcedure,
			member: true, parent: project,
			wantRow: true, wantWarning: true,
			call: func(t *testing.T) {
				// projectDeveloper carries bb.databases.get, so the ACL admits
				// the call, and not bb.sql.select, so the handler's own check
				// refuses it.
				restore := grantMemberProjectRole(t, "roles/projectDeveloper")
				defer restore()
				resp, err := memberSQL.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
					Name: clamp.database, Statement: "SELECT * FROM employee;", Limit: 1,
				}))
				// Query reports this refusal inside its response: the RPC
				// answers OK and the row's status stays empty, so severity is
				// the only field that can carry the denial to the audit log.
				require.NoError(t, err)
				require.Len(t, resp.Msg.Results, 1)
				require.Equal(t, []string{"bb.sql.select"},
					resp.Msg.Results[0].GetPermissionDenied().GetRequiredPermissions())
			},
			checkRow: func(t *testing.T, row *v1pb.AuditLog) {
				require.Nil(t, row.Status, "the refusal never reaches the RPC status")
			},
		},
		{
			name: "mcp/audited refused by the gate", method: v1connect.UserServiceCreateUserProcedure, mcp: true, parent: workspace,
			wantWarning: true,
			call: mcpCall(adminSession, "UserService/CreateUser", map[string]any{
				"user": map[string]any{"email": "sneaky@example.com", "title": "sneaky", "password": memberPassword},
			}, http.StatusForbidden),
		},
		{
			name: "mcp/audited refused by ACL", method: v1connect.PlanServiceCreatePlanProcedure, member: true, mcp: true, parent: project,
			wantWarning: true,
			call: mcpCall(memberSession, "PlanService/CreatePlan", map[string]any{
				"parent": project,
				"plan":   map[string]any{"title": "not mine to make"},
			}, http.StatusForbidden),
		},
		{
			name: "mcp/unaudited refused by the gate", method: v1connect.UserServiceListUsersProcedure, mcp: true, parent: workspace,
			wantWarning: true,
			call:        mcpCall(adminSession, "UserService/ListUsers", map[string]any{}, http.StatusForbidden),
		},
		{
			name: "mcp/unaudited refused in handler", method: v1connect.IssueServiceSearchIssuesProcedure, member: true, mcp: true, parent: workspace,
			wantWarning: true,
			call:        mcpCall(memberSession, "IssueService/SearchIssues", map[string]any{"parent": project}, http.StatusForbidden),
		},
		{
			name: "mcp/unaudited ok", method: v1connect.WorkspaceServiceListWorkspacesProcedure, mcp: true, parent: workspace,
			call: mcpCall(adminSession, "WorkspaceService/ListWorkspaces", map[string]any{}, http.StatusOK),
		},
		{
			name: "mcp/audited validate-only ok", method: v1connect.DatabaseGroupServiceCreateDatabaseGroupProcedure, mcp: true, parent: project,
			call: mcpCall(adminSession, "DatabaseGroupService/CreateDatabaseGroup", map[string]any{
				"parent":          project,
				"databaseGroupId": "sinks-mcp",
				"databaseGroup":   map[string]any{"title": "sinks-mcp", "databaseExpr": map[string]any{"expression": "true"}},
				"validateOnly":    true,
			}, http.StatusOK),
		},
		{
			name: "mcp/audited validate-only refused by the gate", method: v1connect.InstanceServiceUpdateDataSourceProcedure, mcp: true, parent: workspace,
			wantWarning: true,
			checkLine: func(t *testing.T, line map[string]any) {
				require.Contains(t, lineText(line, "request"), `"validateOnly":true`,
					"the skip keys on the outcome, not on the flag")
			},
			call: mcpCall(adminSession, "InstanceService/UpdateDataSource", map[string]any{
				"name":         "instances/sinks",
				"dataSource":   map[string]any{"id": "admin", "host": "attacker.example.com"},
				"updateMask":   "host",
				"validateOnly": true,
			}, http.StatusForbidden),
		},
	}

	for _, stdout := range []bool{true, false} {
		ctl.profile.RuntimeEnableAuditLogStdout.Store(stdout)
		for _, p := range probes {
			t.Run(fmt.Sprintf("stdout=%t/%s", stdout, p.name), func(t *testing.T) {
				user := adminUser
				if p.member {
					user = memberUser
				}
				before := len(auditSinkRows(ctx, t, ctl, p, user))
				linesBefore := len(lines())

				p.call(t)

				rows := auditSinkRows(ctx, t, ctl, p, user)
				var got []map[string]any
				for _, line := range lines()[linesBefore:] {
					if line["method"] == p.method && line["user"] == user && lineFlag(line, "mcp") == p.mcp &&
						(p.match == nil || p.match(lineText(line, "request"))) {
						got = append(got, line)
					}
				}

				severity := v1pb.AuditLog_INFO
				if p.wantWarning {
					severity = v1pb.AuditLog_WARNING
				}
				if p.wantRow {
					require.Len(t, rows, before+1, "the call must be stored once")
					// Newest first.
					require.Equal(t, severity, rows[0].Severity)
					require.True(t, strings.HasPrefix(rows[0].Name, p.parent+"/auditLogs/"), rows[0].Name)
					if p.checkRow != nil {
						p.checkRow(t, rows[0])
					}
				} else {
					require.Len(t, rows, before, "the call must not be stored")
				}
				// Stdout carries every stored row and every refusal.
				wantLine := stdout && (p.wantRow || p.wantWarning)
				if !wantLine {
					require.Empty(t, got, "the call must not be streamed")
					return
				}
				require.Len(t, got, 1, "the call must be streamed once")
				require.Equal(t, p.parent, got[0]["parent"])
				require.Equal(t, severity.String(), got[0]["severity"])
				if p.wantRow {
					require.Equal(t, rows[0].Request, lineText(got[0], "request"), "one built row feeds both sinks")
				}
				if p.checkLine != nil {
					p.checkLine(t, got[0])
				}
			})
		}
	}
}

func lineText(line map[string]any, key string) string {
	v, ok := line[key].(string)
	if !ok {
		return ""
	}
	return v
}

func auditSinkRows(ctx context.Context, t *testing.T, ctl *controller, p auditSinkProbe, user string) []*v1pb.AuditLog {
	t.Helper()
	resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		// Unscoped: a row filed under the wrong parent must fail the probe's
		// parent assertion, not vanish from the search.
		Parent:   "projects/-",
		Filter:   `method == "` + p.method + `"`,
		OrderBy:  "create_time desc",
		PageSize: 1000,
	}))
	require.NoError(t, err)
	var rows []*v1pb.AuditLog
	for _, row := range resp.Msg.AuditLogs {
		if row.Actor == user && (row.McpDelegation != nil) == p.mcp && (p.match == nil || p.match(row.Request)) {
			rows = append(rows, row)
		}
	}
	return rows
}
