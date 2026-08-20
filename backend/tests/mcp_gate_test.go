package tests

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

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
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

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

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	// READ: served under every ceiling that admits a connection at all.
	a.Equal(http.StatusOK, callAPIStatus(ctx, t, session, "WorkspaceService/ListWorkspaces", nil))

	// WRITE: served because the workspace ceiling is unset, which resolves
	// READ_WRITE.
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

	// The EXCLUDED denial is audited, and ListUsers carries no audit
	// annotation: the record comes from the gate, not from the method.
	a.Len(deniedMCPRows(ctx, t, ctl, workspaceName, "/bytebase.v1.UserService/ListUsers"), 1,
		"a policy denial is recorded even where the method asks for no audit row")
}

// TestMCPGateRefusesGrantIssues is the one refusal a per-method class cannot
// express, over the live handler. CreateIssue is WRITE for the database-change
// issue an agent exists to compose; a ROLE_GRANT issue completes on creation
// when the workspace rule produces no approval template and writes the project
// IAM binding the request named, with no human in it.
func TestMCPGateRefusesGrantIssues(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: "mcp-grant-issue",
		Project:   &v1pb.Project{Title: "MCP grant issue"},
	}))
	a.NoError(err)
	projectName := project.Msg.Name

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

	// The bypass: UpdateIssue with allow_missing creates the issue when it does
	// not exist, by calling CreateIssue directly — a Go call, so the ceiling
	// gate never sees it. The gate deliberately does not refuse UpdateIssue,
	// because an AIP upsert sends the complete resource and the request cannot
	// say whether it will create anything; the guard sits at the creation
	// instead, and this is what proves it is there.
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
	a.Contains(upsert.Error, "may not create a ROLE_GRANT issue")

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
}

// TestMCPCannotRunAnIssuelessRollout is the rule rollout_service.proto says the
// gate PR owes, and the reason it is a rule rather than a classification.
//
// Three approval RPCs are FORBIDDEN so an agent cannot move its own change
// through the human gate. That buys nothing if the agent can go round it: both
// approval checks on the execution path are guarded on a LINKED ISSUE existing,
// so a plan created without one reaches task creation with no approval whatever
// require_issue_approval says. CreatePlan, CreateRollout and BatchRunTasks are
// all WRITE and stay WRITE — issueless deployment is a supported contract, the
// shipped GitOps Service Agent role holds no issue permission at all — so the
// refusal is per session, not per method.
//
// The ceiling gate cannot take it: "the plan has no linked issue" is not a
// field of either request. The guard sits where the fact is loaded, and the
// control that matters most is the last one — a human doing the identical thing
// must be unaffected, because that is the risk of putting MCP policy in a
// handler at all.
func TestMCPCannotRunAnIssuelessRollout(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			Title:       "MCP rollout instance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
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
