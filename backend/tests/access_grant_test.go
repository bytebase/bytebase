package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/durationpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestAccessGrantApproverVisibility verifies that:
// (a) a user with bb.accessGrants.get (projectOwner) can view the grant,
// (b) an approver (projectDeveloper) without bb.accessGrants.get can view the grant, and
// (c) a non-approver (projectViewer) is denied.
func TestAccessGrantApproverVisibility(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create project.
	projectID := generateRandomString("grant-vis")
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:              fmt.Sprintf("projects/%s", projectID),
			Title:             projectID,
			AllowSelfApproval: true,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

	// Create SQLite instance and database.
	instanceDir := t.TempDir()
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			Title:       "Test Instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{
				Type: v1pb.DataSourceType_ADMIN,
				Host: instanceDir,
				Id:   "admin",
			}},
		},
	}))
	a.NoError(err)

	dbName := generateRandomString("db")
	err = ctl.createDatabase(ctx, project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)
	dbTarget := fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName)

	// Set approval rule requiring projectDeveloper for REQUEST_ACCESS in this project.
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								Source: v1pb.WorkspaceApprovalSetting_Rule_REQUEST_ACCESS,
								Condition: &expr.Expr{
									Expression: fmt.Sprintf(`resource.project_id == "%s"`, projectID),
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Developer Approval",
									Flow: &v1pb.ApprovalFlow{
										Roles: []string{"roles/projectDeveloper"},
									},
								},
							},
						},
					},
				},
			},
		},
	}))
	a.NoError(err)

	adminToken := ctl.authInterceptor.token

	// Create access grant (as admin/projectOwner).
	grantResp, err := ctl.accessGrantServiceClient.CreateAccessGrant(ctx, connect.NewRequest(&v1pb.CreateAccessGrantRequest{
		Parent: project.Name,
		AccessGrant: &v1pb.AccessGrant{
			Creator: "users/demo@example.com",
			Targets: []string{dbTarget},
			Query:   "SELECT 1",
			Reason:  "Testing approver visibility",
			Expiration: &v1pb.AccessGrant_Ttl{
				Ttl: &durationpb.Duration{Seconds: 3600},
			},
		},
	}))
	a.NoError(err)
	grantName := grantResp.Msg.Name

	// Find the linked issue and poll until approval finding completes.
	issuesResp, err := ctl.issueServiceClient.SearchIssues(ctx, connect.NewRequest(&v1pb.SearchIssuesRequest{
		Parent: project.Name,
		Filter: `type = "ACCESS_GRANT"`,
	}))
	a.NoError(err)
	a.NotEmpty(issuesResp.Msg.Issues, "issue should have been created by CreateAccessGrant")
	issueName := issuesResp.Msg.Issues[0].Name

	var issue *v1pb.Issue
	for i := 0; i < 10; i++ {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		issueGetResp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
			Name: issueName,
		}))
		a.NoError(err)
		issue = issueGetResp.Msg
		if issue.ApprovalStatus != v1pb.Issue_CHECKING {
			break
		}
	}
	a.NotNil(issue)
	a.NotEqual(v1pb.Issue_CHECKING, issue.ApprovalStatus, "approval finding should have completed")
	a.Equal(v1pb.Issue_PENDING, issue.ApprovalStatus, "issue should be pending approval")

	// (a) projectOwner (admin) can view the grant.
	getResp, err := ctl.accessGrantServiceClient.GetAccessGrant(ctx, connect.NewRequest(&v1pb.GetAccessGrantRequest{
		Name: grantName,
	}))
	a.NoError(err)
	a.Equal(grantName, getResp.Msg.Name)

	// Create developer user (approver role, but no bb.accessGrants.get).
	devEmail := fmt.Sprintf("dev-%s@example.com", projectID)
	devPassword := "1024bytebase"
	devUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    devEmail,
			Password: devPassword,
			Title:    "Developer",
		},
	}))
	a.NoError(err)

	_, err = ctl.addMemberToWorkspaceIAM(ctx, devUser.Msg.Workspace, fmt.Sprintf("user:%s", devEmail), "roles/workspaceMember")
	a.NoError(err)

	// Add developer to project with projectDeveloper role.
	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: project.Name,
	}))
	a.NoError(err)
	policy := policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/projectDeveloper",
		Members: []string{fmt.Sprintf("user:%s", devEmail)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	// Login as developer.
	devLogin, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    devEmail,
		Password: devPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = devLogin.Msg.Token

	// (b) Developer (approver) can view the grant.
	getResp, err = ctl.accessGrantServiceClient.GetAccessGrant(ctx, connect.NewRequest(&v1pb.GetAccessGrantRequest{
		Name: grantName,
	}))
	a.NoError(err, "Approver with projectDeveloper role should be able to view the grant")
	a.Equal(grantName, getResp.Msg.Name)

	// Switch back to admin to create the viewer user.
	ctl.authInterceptor.token = adminToken

	viewerEmail := fmt.Sprintf("viewer-%s@example.com", projectID)
	viewerPassword := "1024bytebase"
	viewerUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    viewerEmail,
			Password: viewerPassword,
			Title:    "Viewer",
		},
	}))
	a.NoError(err)

	_, err = ctl.addMemberToWorkspaceIAM(ctx, viewerUser.Msg.Workspace, fmt.Sprintf("user:%s", viewerEmail), "roles/workspaceMember")
	a.NoError(err)

	// Add viewer to project with projectViewer role.
	policyResp, err = ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: project.Name,
	}))
	a.NoError(err)
	policy = policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/projectViewer",
		Members: []string{fmt.Sprintf("user:%s", viewerEmail)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	// Login as viewer.
	viewerLogin, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    viewerEmail,
		Password: viewerPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = viewerLogin.Msg.Token

	// (c) Viewer (not an approver) is denied.
	_, err = ctl.accessGrantServiceClient.GetAccessGrant(ctx, connect.NewRequest(&v1pb.GetAccessGrantRequest{
		Name: grantName,
	}))
	a.Error(err, "Non-approver should be denied access to the grant")
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))
}
