package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestDeleteUser(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	expectErrorMsg := "workspace must have at least one admin"

	memberResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "member",
			Email:    "member@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	member := memberResp.Msg

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: member.Name,
	}))
	a.NoError(err)

	// Test: cannot delete the last admin.
	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: ctl.principalName,
	}))
	a.Error(err)
	a.ErrorContains(err, expectErrorMsg)

	actuator, err := ctl.actuatorServiceClient.GetActuatorInfo(ctx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{
		Name: memberResp.Msg.Workspace,
	}))
	a.NoError(err)

	serviceAccountResp, err := ctl.serviceAccountServiceClient.CreateServiceAccount(ctx, connect.NewRequest(&v1pb.CreateServiceAccountRequest{
		Parent:           actuator.Msg.Workspace,
		ServiceAccountId: "bot",
		ServiceAccount: &v1pb.ServiceAccount{
			Title: "bot",
		},
	}))
	a.NoError(err)
	serviceAccount := serviceAccountResp.Msg

	_, err = ctl.addMemberToWorkspaceIAM(ctx, actuator.Msg.Workspace, fmt.Sprintf("serviceAccount:%v", serviceAccount.Email), "roles/workspaceAdmin")
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: ctl.principalName,
	}))
	a.Error(err)
	a.ErrorContains(err, expectErrorMsg)

	_, err = ctl.userServiceClient.UndeleteUser(ctx, connect.NewRequest(&v1pb.UndeleteUserRequest{
		Name: member.Name,
	}))
	a.NoError(err)

	// Test: can delete the admin if member count > 1
	newPolicy, err := ctl.addMemberToWorkspaceIAM(ctx, actuator.Msg.Workspace, fmt.Sprintf("user:%v", member.Email), "roles/workspaceAdmin")
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: ctl.principalName,
	}))
	a.NoError(err)

	// Switch context.
	resp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    member.Email,
		Password: "1024bytebase",
	}))
	a.NoError(err)
	ctl.authInterceptor.token = resp.Msg.Token

	// Test: check allUser in the binding
	for _, binding := range newPolicy.Bindings {
		if binding.Role == "roles/workspaceAdmin" {
			binding.Members = []string{common.AllUsers}
			break
		}
	}
	_, err = ctl.workspaceServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Etag:     newPolicy.Etag,
		Policy:   newPolicy,
		Resource: actuator.Msg.Workspace,
	}))
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: member.Name,
	}))
	a.Error(err)
	a.ErrorContains(err, expectErrorMsg)

	_, err = ctl.userServiceClient.UndeleteUser(ctx, connect.NewRequest(&v1pb.UndeleteUserRequest{
		Name: ctl.principalName,
	}))
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: member.Name,
	}))
	a.NoError(err)
}

func TestUpdateUserEmail(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// 1. Create a user
	originalEmail := "original@bytebase.com"
	newEmail := "updated@bytebase.com"
	userResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "temp-user",
			Email:    originalEmail,
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	user := userResp.Msg

	// Add the created user to workspace IAM as member so they can login.
	_, err = ctl.addMemberToWorkspaceIAM(ctx, user.Workspace, fmt.Sprintf("user:%v", user.Email), "roles/workspaceMember")
	a.NoError(err)

	// 1.5 Grant user permission to create issues in the project
	projectID := "test-project"
	// Login as admin (StartServer returns admin token but let's be explicit or reuse ctl state if it hasn't changed)
	// Actually StartServer sets ctl.authInterceptor.token to admin token.
	// So we are currently admin.

	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: "projects/" + projectID,
	}))
	a.NoError(err)
	policy := policyResp.Msg

	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/projectDeveloper",
		Members: []string{fmt.Sprintf("user:%s", originalEmail)},
	})

	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: "projects/" + projectID,
		Policy:   policy,
	}))
	a.NoError(err)

	// 1.6 Create a group and add the user to it
	groupEmail := "test-group@bytebase.com"
	groupResp, err := ctl.groupServiceClient.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
		Group: &v1pb.Group{
			Title: "Test Group",
		},
		GroupEmail: groupEmail,
	}))
	a.NoError(err)
	group := groupResp.Msg

	// Add user to the group
	updatedGroupResp, err := ctl.groupServiceClient.UpdateGroup(ctx, connect.NewRequest(&v1pb.UpdateGroupRequest{
		Group: &v1pb.Group{
			Name:    group.Name,
			Title:   group.Title,
			Email:   group.Email,
			Members: []*v1pb.GroupMember{{Member: common.FormatUserEmail(originalEmail), Role: v1pb.GroupMember_MEMBER}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"members"}},
	}))
	a.NoError(err)
	a.Len(updatedGroupResp.Msg.Members, 1)
	a.Equal(common.FormatUserEmail(originalEmail), updatedGroupResp.Msg.Members[0].Member)

	// 1.7 Create a masking exception policy with the user
	maskingPolicy, err := ctl.orgPolicyServiceClient.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: "projects/" + projectID,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_MASKING_EXEMPTION,
			Policy: &v1pb.Policy_MaskingExemptionPolicy{
				MaskingExemptionPolicy: &v1pb.MaskingExemptionPolicy{
					Exemptions: []*v1pb.MaskingExemptionPolicy_Exemption{
						{
							Members:   []string{fmt.Sprintf("user:%s", originalEmail)},
							Condition: &expr.Expr{}, // Empty condition means access all databases without expiration
						},
					},
				},
			},
		},
	}))
	a.NoError(err)

	// 2. Login as user and create resources
	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    originalEmail,
		Password: "1024bytebase",
	}))
	a.NoError(err)
	userCtx := ctx
	ctl.authInterceptor.token = loginResp.Msg.Token

	// Create Issue
	// Use ROLE_GRANT to avoid plan requirement and also test payload update.
	grantUser := common.FormatUserEmail(originalEmail)
	issueResp, err := ctl.issueServiceClient.CreateIssue(userCtx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/" + projectID,
		Issue: &v1pb.Issue{
			Title: "Test Grant Request",
			Type:  v1pb.Issue_ROLE_GRANT,
			RoleGrant: &v1pb.RoleGrant{
				Role: "roles/projectDeveloper",
				User: grantUser,
			},
			Description: "desc",
		},
	}))
	a.NoError(err)
	issue := issueResp.Msg

	// Create Issue Comment
	commentResp, err := ctl.issueServiceClient.CreateIssueComment(userCtx, connect.NewRequest(&v1pb.CreateIssueCommentRequest{
		IssueComment: &v1pb.IssueComment{
			Comment: "Test Comment",
		},
		Parent: issue.Name,
	}))
	a.NoError(err)
	comment := commentResp.Msg

	// 3. Update Email (as Admin)
	// Login as admin
	adminLoginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    "demo@example.com",
		Password: "1024bytebase",
	}))
	a.NoError(err)
	ctl.authInterceptor.token = adminLoginResp.Msg.Token

	// Function under test
	updateResp, err := ctl.userServiceClient.UpdateEmail(ctx, connect.NewRequest(&v1pb.UpdateEmailRequest{
		Name:  user.Name,
		Email: newEmail,
	}))
	a.NoError(err)
	a.Equal(newEmail, updateResp.Msg.Email)

	// 4. Verify Cascade

	// Verify Issue Creator
	updatedIssueResp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
		Name: issue.Name,
	}))
	a.NoError(err)
	// Creator is a string "users/{email}"
	a.Equal(common.FormatUserEmail(newEmail), updatedIssueResp.Msg.Creator, "Issue creator should be updated")
	// Verify RoleGrant user in payload.
	a.Equal(common.FormatUserEmail(newEmail), updatedIssueResp.Msg.RoleGrant.User, "RoleGrant user should be updated")

	// Verify Issue Comment Creator
	commentsResp, err := ctl.issueServiceClient.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{
		Parent: issue.Name,
	}))
	a.NoError(err)
	foundComment := false
	for _, c := range commentsResp.Msg.IssueComments {
		if c.Name == comment.Name {
			a.Equal(common.FormatUserEmail(newEmail), c.Creator)
			foundComment = true
		}
	}
	a.True(foundComment)

	// Verify Project Policy
	newPolicyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: "projects/" + projectID,
	}))
	a.NoError(err)
	foundMember := false
	for _, b := range newPolicyResp.Msg.Bindings {
		for _, m := range b.Members {
			if m == fmt.Sprintf("user:%s", newEmail) {
				foundMember = true
			}
			a.NotEqual(fmt.Sprintf("user:%s", originalEmail), m, "Old email should not present")
		}
	}
	a.True(foundMember, "New email should be in policy")

	// Verify Group Membership
	updatedGroup, err := ctl.groupServiceClient.GetGroup(ctx, connect.NewRequest(&v1pb.GetGroupRequest{
		Name: group.Name,
	}))
	a.NoError(err)
	a.Len(updatedGroup.Msg.Members, 1)
	a.Equal(common.FormatUserEmail(newEmail), updatedGroup.Msg.Members[0].Member, "Group member should be updated to new email")
	for _, m := range updatedGroup.Msg.Members {
		a.NotEqual(common.FormatUserEmail(originalEmail), m.Member, "Old email should not present in group")
	}

	// Verify Masking Exception Policy
	updatedMaskingPolicy, err := ctl.orgPolicyServiceClient.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{
		Name: maskingPolicy.Msg.Name,
	}))
	a.NoError(err)
	maskingExemptions := updatedMaskingPolicy.Msg.GetMaskingExemptionPolicy().GetExemptions()
	a.Len(maskingExemptions, 1)
	a.Equal(fmt.Sprintf("user:%s", newEmail), maskingExemptions[0].Members[0], "Masking exemption member should be updated to new email")
	for _, exemption := range maskingExemptions {
		a.NotEqual(fmt.Sprintf("user:%s", originalEmail), exemption.Members[0], "Old email should not present in masking exemption")
	}

	// Verify Audit Logs
	// Search for audit logs related to the project (audit logs were created when user created issue/comment)
	auditLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: "projects/" + projectID,
		Filter: fmt.Sprintf(`user == "%s"`, common.FormatUserEmail(newEmail)),
	}))
	a.NoError(err)
	// We should have at least some audit logs from the issue/comment creation
	a.NotEmpty(auditLogs.Msg.AuditLogs, "Should have audit logs with new email")
	// Verify no audit logs have the old email
	oldEmailAuditLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: "projects/" + projectID,
		Filter: fmt.Sprintf(`user == "%s"`, common.FormatUserEmail(originalEmail)),
	}))
	a.NoError(err)
	a.Empty(oldEmailAuditLogs.Msg.AuditLogs, "Should not have audit logs with old email")
}

func TestGetCurrentUser_ServiceAccount(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a dummy user to get the workspace reference.
	userResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "dummy",
			Email:    "dummy@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	workspace := userResp.Msg.Workspace

	// Get actuator info using the workspace reference.
	actuator, err := ctl.actuatorServiceClient.GetActuatorInfo(ctx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{
		Name: workspace,
	}))
	a.NoError(err)

	// Create a service account.
	saResp, err := ctl.serviceAccountServiceClient.CreateServiceAccount(ctx, connect.NewRequest(&v1pb.CreateServiceAccountRequest{
		Parent:           actuator.Msg.Workspace,
		ServiceAccountId: "sa-test",
		ServiceAccount: &v1pb.ServiceAccount{
			Title: "SA Test",
		},
	}))
	a.NoError(err)
	sa := saResp.Msg

	// Grant the service account workspace admin role.
	_, err = ctl.addMemberToWorkspaceIAM(ctx, actuator.Msg.Workspace, fmt.Sprintf("serviceAccount:%v", sa.Email), "roles/workspaceAdmin")
	a.NoError(err)

	// Login as the service account using its service_key as password.
	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    sa.Email,
		Password: sa.ServiceKey,
	}))
	a.NoError(err)

	// Swap the controller token to the service account's token.
	originalToken := ctl.authInterceptor.token
	ctl.authInterceptor.token = loginResp.Msg.Token
	defer func() {
		ctl.authInterceptor.token = originalToken
	}()

	meResp, err := ctl.userServiceClient.GetCurrentUser(ctx, connect.NewRequest(&emptypb.Empty{}))

	// Assertions per the task specification.
	a.NoError(err)
	a.NotNil(meResp.Msg)
	a.Equal(sa.Email, meResp.Msg.Email)
	a.NotNil(meResp.Msg.Profile)
}
