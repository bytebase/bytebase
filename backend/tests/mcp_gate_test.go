package tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestMCPGateServesAndRefusesByClass is the ceiling gate as an agent meets it:
// one live MCP session under the default workspace ceiling, calling one method
// of each class through call_api.
//
// The session is a workspace admin, so RBAC permits everything here. What
// refuses the two denied calls is the ceiling alone, which is the claim —
// effective = ceiling ∩ RBAC, and this test holds the ceiling term with the
// RBAC term saturated.
func TestMCPGateServesAndRefusesByClass(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	workspaceName := workspace.Msg.Name

	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: "mcp-gate",
		Project:   &v1pb.Project{Title: "MCP gate"},
	}))
	a.NoError(err)
	projectName := project.Msg.Name

	// Pinned so this test does not depend on the resolved default: the WRITE
	// arm below is about what the gate serves at READ_WRITE.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	// READ: served under every ceiling that admits a connection at all.
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, session, "WorkspaceService/ListWorkspaces", nil))

	// WRITE: served because the ceiling above is READ_WRITE.
	sheet := callAPIOnSession(ctx, t, session, "SheetService/CreateSheet", map[string]any{
		"parent": projectName,
		"sheet":  map[string]any{"content": base64.StdEncoding.EncodeToString([]byte("SELECT 1;"))},
	})
	a.Equal(http.StatusOK, sheet.Status, "a read-write session must reach a WRITE method: %s", sheet.Error)

	// EXCLUDED: refused whatever the ceiling, with the reason the annotation
	// records — this one is workspace administration, which no mode this phase
	// ships covers.
	excluded := callAPIOnSession(ctx, t, session, "UserService/ListUsers", map[string]any{})
	a.Equal(http.StatusForbidden, excluded.Status)
	a.Contains(excluded.Error, "administers the workspace")
	a.Contains(excluded.Error, "Bytebase console", "the denial must name where the human can do it instead")

	// FORBIDDEN: refused with a different recorded reason, so the two classes
	// do not collapse into one sentence an operator cannot tell apart.
	forbidden := callAPIOnSession(ctx, t, session, "AuthService/Logout", map[string]any{})
	a.Equal(http.StatusForbidden, forbidden.Status)
	a.Contains(forbidden.Error, "ends the human's own login session")

	// A gate refusal is streamed, never stored; the served CreateSheet is stored.
	a.Empty(mcpAuditRows(ctx, t, ctl, workspaceName, "/bytebase.v1.UserService/ListUsers"),
		"a gate refusal is never stored")
	a.Len(mcpAuditRows(ctx, t, ctl, projectName, "/bytebase.v1.SheetService/CreateSheet"), 1,
		"a served, audited MCP call is stored")
}

// TestMCPGateRefusesGrantIssues covers, over one live MCP session, the grant
// issues a session may not create and the access request it may file instead,
// which needs a project that allows just-in-time access.
func TestMCPGateRefusesGrantIssues(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: "mcp-grant-issue",
		Project:   &v1pb.Project{Title: "MCP grant issue"},
	}))
	a.NoError(err)
	projectName := project.Msg.Name

	// CreateIssue and CreateAccessGrant are WRITE. Pinned so this test does not
	// depend on the resolved default.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	grant := callAPIOnSession(ctx, t, session, "IssueService/CreateIssue", map[string]any{
		"parent": projectName,
		"issue": map[string]any{
			"type":  "ROLE_GRANT",
			"title": "grant me project owner",
			"roleGrant": map[string]any{
				"role": "roles/projectOwner",
				"user": "users/demo@example.com",
			},
		},
	})
	a.Equal(http.StatusForbidden, grant.Status)
	a.Contains(grant.Error, "ROLE_GRANT", "the denial must name the issue type it refused")
	a.Contains(grant.Error, "no human step")

	upsert := callAPIOnSession(ctx, t, session, "IssueService/UpdateIssue", map[string]any{
		"issue": map[string]any{
			"name":  projectName + "/issues/999999",
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
	a.Equal(http.StatusForbidden, upsert.Status,
		"allow_missing on a missing issue is a creation, and the guard must reach it: %s", upsert.Error)
	a.Contains(upsert.Error, "may not create this issue")
	a.Contains(upsert.Error, "ROLE_GRANT", "the denial must name the issue type it refused")

	// The same upsert for the type an agent is allowed to create reaches the
	// handler, which then asks for the plan it has no way to do without. The
	// refusal is about the issue type, not about allow_missing.
	changeUpsert := callAPIOnSession(ctx, t, session, "IssueService/UpdateIssue", map[string]any{
		"issue": map[string]any{
			"name":  projectName + "/issues/999998",
			"title": "an ordinary change",
			"type":  "DATABASE_CHANGE",
		},
		"updateMask":   "title",
		"allowMissing": true,
	})
	a.NotEqual(http.StatusForbidden, changeUpsert.Status,
		"a database-change upsert must reach the handler: %s", changeUpsert.Error)

	// The same method with the type it is classified for reaches its handler.
	// A missing plan is the handler's own complaint, which is the point: the
	// gate did not answer this one.
	change := callAPIOnSession(ctx, t, session, "IssueService/CreateIssue", map[string]any{
		"parent": projectName,
		"issue":  map[string]any{"type": "DATABASE_CHANGE", "title": "an ordinary change"},
	})
	a.Equal(http.StatusBadRequest, change.Status,
		"a database-change issue must reach the handler, which then asks for its plan: %s", change.Error)
	a.NotContains(change.Error, "not available to MCP sessions")

	// An unset type reaches the handler too, and is the handler's own
	// complaint: an invalid argument, not a policy verdict, so its row stays
	// INFO.
	untyped := callAPIOnSession(ctx, t, session, "IssueService/CreateIssue", map[string]any{
		"parent": projectName,
		"issue":  map[string]any{"title": "no type at all"},
	})
	a.Equal(http.StatusBadRequest, untyped.Status,
		"an unset type is the handler's invalid argument: %s", untyped.Error)
	a.NotContains(untyped.Error, "MCP session")

	accessGrantIssue := callAPIOnSession(ctx, t, session, "IssueService/CreateIssue", map[string]any{
		"parent": projectName,
		"issue":  map[string]any{"type": "ACCESS_GRANT", "title": "let me read the table"},
	})
	a.Equal(http.StatusForbidden, accessGrantIssue.Status)
	a.Contains(accessGrantIssue.Error, "AccessGrantService/CreateAccessGrant",
		"the refusal must name the method that files the request")
	a.NotContains(accessGrantIssue.Error, "Bytebase console")

	pgContainer := sharedPgTarget(t)
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			SyncDatabases: &v1pb.SyncDatabases{},
			Title:         "MCP access grant instance",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	dbName := generateRandomString("db")
	a.NoError(ctl.createDatabase(ctx, project.Msg, instance.Msg, nil, dbName, ""))
	target := fmt.Sprintf("%s/databases/%s", instance.Msg.Name, dbName)
	requestAccess := map[string]any{
		"parent": projectName,
		"accessGrant": map[string]any{
			"targets": []string{target},
			"query":   "SELECT 1",
			"reason":  "the agent needs one read",
			"ttl":     "3600s",
		},
	}
	consoleCreate := func() (*connect.Response[v1pb.AccessGrant], error) {
		return ctl.accessGrantServiceClient.CreateAccessGrant(ctx, connect.NewRequest(&v1pb.CreateAccessGrantRequest{
			Parent: projectName,
			AccessGrant: &v1pb.AccessGrant{
				Targets:    []string{target},
				Query:      "SELECT 1",
				Reason:     "a person asks the same",
				Expiration: &v1pb.AccessGrant_Ttl{Ttl: durationpb.New(time.Hour)},
			},
		}))
	}
	rowCounts := func() (int, int) {
		grants, err := ctl.accessGrantServiceClient.ListAccessGrants(ctx, connect.NewRequest(&v1pb.ListAccessGrantsRequest{Parent: projectName}))
		a.NoError(err)
		issues, err := ctl.issueServiceClient.ListIssues(ctx, connect.NewRequest(&v1pb.ListIssuesRequest{Parent: projectName}))
		a.NoError(err)
		return len(grants.Msg.AccessGrants), len(issues.Msg.Issues)
	}

	// The project does not allow just-in-time access yet.
	grantsAtStart, issuesAtStart := rowCounts()
	jitOff := callAPIOnSession(ctx, t, session, "AccessGrantService/CreateAccessGrant", requestAccess)
	a.Equal(http.StatusBadRequest, jitOff.Status, jitOff.Error)
	var jitOffBody struct {
		Code string `json:"code"`
	}
	a.NoError(json.Unmarshal([]byte(jitOff.RawResponse), &jitOffBody))
	a.Equal("failed_precondition", jitOffBody.Code)
	a.Contains(jitOff.Error, `just-in-time access is off for project "mcp-grant-issue"`)
	_, err = consoleCreate()
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err), "a console request is refused the same way: %v", err)
	grantsRefused, issuesRefused := rowCounts()
	a.Equal(grantsAtStart, grantsRefused, "a request refused for just-in-time access leaves no access grant")
	a.Equal(issuesAtStart, issuesRefused, "a request refused for just-in-time access leaves no issue")
	a.NoError(ctl.allowJustInTimeAccess(ctx, project.Msg))

	// The seeded fallback rule sends the request to a project owner.
	filed := callAPIOnSession(ctx, t, session, "AccessGrantService/CreateAccessGrant", requestAccess)
	a.Equal(http.StatusOK, filed.Status, filed.Error)
	grantsFiled, issuesFiled := rowCounts()
	a.Equal(grantsAtStart+1, grantsFiled, "the row count sees a filed access grant")
	a.Equal(issuesAtStart+1, issuesFiled, "the row count sees a filed issue")
	var pending v1pb.AccessGrant
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(filed.RawResponse), &pending))
	a.Equal(v1pb.AccessGrant_PENDING, pending.Status)
	issue, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{Name: pending.Issue}))
	a.NoError(err)
	a.Equal([]string{"roles/projectOwner"}, issue.Msg.GetApprovalTemplate().GetFlow().GetRoles())
	a.Equal(v1pb.ApprovalStatus_PENDING, issue.Msg.ApprovalStatus)

	activate := callAPIOnSession(ctx, t, session, "AccessGrantService/ActivateAccessGrant", map[string]any{
		"name": pending.Name,
	})
	a.Equal(http.StatusForbidden, activate.Status)
	a.Contains(activate.Error, "approv", "refused as an approval decision, not as workspace administration")
	still, err := ctl.accessGrantServiceClient.GetAccessGrant(ctx, connect.NewRequest(&v1pb.GetAccessGrantRequest{Name: pending.Name}))
	a.NoError(err)
	a.Equal(v1pb.AccessGrant_PENDING, still.Msg.Status)

	consoleRequest := func() *v1pb.AccessGrant {
		granted, err := consoleCreate()
		a.NoError(err)
		return granted.Msg
	}
	consolePending := consoleRequest()
	a.Equal(v1pb.AccessGrant_PENDING, consolePending.Status, "under the seeded rule, a console request waits as before")
	consoleIssue, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{Name: consolePending.Issue}))
	a.NoError(err)
	a.Equal([]string{"roles/projectOwner"}, consoleIssue.Msg.GetApprovalTemplate().GetFlow().GetRoles())

	for _, tc := range []struct {
		name  string
		rules []*v1pb.WorkspaceApprovalSetting_Rule
	}{
		{"a rule that names no approver", []*v1pb.WorkspaceApprovalSetting_Rule{{
			Source:    v1pb.WorkspaceApprovalSetting_Rule_REQUEST_ACCESS,
			Condition: &expr.Expr{Expression: "true"},
			Template:  &v1pb.ApprovalTemplate{Flow: &v1pb.ApprovalFlow{}},
		}}},
		{"no rule at all", nil},
	} {
		_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
			AllowMissing: true,
			Setting: &v1pb.Setting{
				Name: "settings/WORKSPACE_APPROVAL",
				Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{Rules: tc.rules},
				}},
			},
		}))
		a.NoError(err, tc.name)
		grantsBefore, issuesBefore := rowCounts()
		refused := callAPIOnSession(ctx, t, session, "AccessGrantService/CreateAccessGrant", requestAccess)
		a.Equal(http.StatusForbidden, refused.Status, tc.name)
		a.Contains(refused.Error, "no approval rule requires a person to approve it", tc.name)
		a.Contains(refused.Error, "an approval rule that names an approver", tc.name)
		grantsAfter, issuesAfter := rowCounts()
		a.Equal(grantsBefore, grantsAfter, "%s: a refused request leaves no access grant", tc.name)
		a.Equal(issuesBefore, issuesAfter, "%s: a refused request leaves no issue", tc.name)
		a.Equal(v1pb.AccessGrant_ACTIVE, consoleRequest().Status, "%s: a console request is granted at once, as before", tc.name)
	}
}

// TestMCPCannotRunAnIssuelessRollout pins rejectMCPOriginatedIssuelessRollout
// over a live session, and that a console user doing the same is unaffected.
func TestMCPCannotRunAnIssuelessRollout(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	// CreateRollout is WRITE and the refusal under test is the
	// issueless-rollout guard. Pinned so this test does not depend on the
	// resolved default.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))

	pgContainer := sharedPgTarget(t)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			SyncDatabases: &v1pb.SyncDatabases{},
			Title:         "MCP rollout instance",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	dbName := generateRandomString("db")
	a.NoError(ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, ""))
	target := fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName)

	// The project requires issue approval, which is the whole point: the
	// customer turned the gate on.
	project := ctl.project
	project.RequireIssueApproval = true
	updated, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project:    project,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"require_issue_approval"}},
	}))
	a.NoError(err)
	a.True(updated.Msg.RequireIssueApproval)

	// A plan with no issue, created the way an agent would.
	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("CREATE TABLE mcp_rollout_guard(id INT);")},
	}))
	a.NoError(err)
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{Specs: []*v1pb.Plan_Spec{{
			Id: uuid.NewString(),
			Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
					Targets: []string{target},
					Sheet:   sheetResp.Msg.Name,
				},
			},
		}}},
	}))
	a.NoError(err)

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	rollout := callAPIOnSession(ctx, t, session, "RolloutService/CreateRollout", map[string]any{
		"parent": planResp.Msg.Name,
	})
	a.Equal(http.StatusForbidden, rollout.Status,
		"an MCP session must not start an issueless change on a project that requires approval: %s", rollout.Error)
	a.Contains(rollout.Error, "plan with no issue")
	a.Contains(rollout.Error, "Bytebase console", "the denial must name where the human can do it instead")

	// The control that makes the handler guard safe: the same call, same plan,
	// same setting, on the public chain. A human — and the GitOps agent, which
	// holds no issue permission at all — must be unaffected.
	consoleRollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: planResp.Msg.Name,
	}))
	a.NoError(err, "the guard binds MCP sessions and nothing else")
	a.NotEmpty(consoleRollout.Msg.Name)
}

// TestMCPDefaultRowIsReadOnly pins the safe default persisted by workspace
// creation: a workspace whose admin never changed MCP does not serve WRITE.
//
// Every other gate test now sets a ceiling explicitly, which is right — each is
// about what a NAMED ceiling serves, and depending on the default made them say
// less than their names claimed. This keeps the persisted default on the live
// path, so changing workspace creation would be caught.
func TestMCPDefaultRowIsReadOnly(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: "mcp-unset",
		Project:   &v1pb.Project{Title: "MCP default ceiling"},
	}))
	a.NoError(err)

	// No setMCPCapability anywhere in this test: workspace creation owns the
	// explicit READ_ONLY fixture.
	setting, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, setting.GetCapability())

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	sheet := callAPIOnSession(ctx, t, session, "SheetService/CreateSheet", map[string]any{
		"parent": project.Msg.Name,
		"sheet":  map[string]any{"content": base64.StdEncoding.EncodeToString([]byte("SELECT 1;"))},
	})
	a.Equal(http.StatusForbidden, sheet.Status,
		"the default READ_ONLY policy must refuse a WRITE method: %s", sheet.Error)
}
