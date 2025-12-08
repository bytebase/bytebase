package tests

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

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
			UserType: v1pb.UserType_USER,
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

	serviceAccountResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "bot",
			UserType: v1pb.UserType_SERVICE_ACCOUNT,
			Email:    "bot@service.bytebase.com",
		},
	}))
	a.NoError(err)
	serviceAccount := serviceAccountResp.Msg

	policyResp, err := ctl.workspaceServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{}))
	a.NoError(err)
	policy := policyResp.Msg

	// Test: only count the end user.
	for _, binding := range policy.Bindings {
		if binding.Role == "roles/workspaceAdmin" {
			binding.Members = append(binding.Members, fmt.Sprintf("user:%s", serviceAccount.Email))
			break
		}
	}
	updatedPolicyResp, err := ctl.workspaceServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Etag:   policy.Etag,
		Policy: policy,
	}))
	a.NoError(err)
	updatedPolicy := updatedPolicyResp.Msg

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
	for _, binding := range updatedPolicy.Bindings {
		if binding.Role == "roles/workspaceAdmin" {
			binding.Members = append(binding.Members, fmt.Sprintf("user:%s", member.Email))
			break
		}
	}
	newPolicyResp, err := ctl.workspaceServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Etag:   updatedPolicy.Etag,
		Policy: updatedPolicy,
	}))
	a.NoError(err)
	newPolicy := newPolicyResp.Msg

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
		Etag:   newPolicy.Etag,
		Policy: newPolicy,
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

// TestListUsers_PermissionFilter tests the permission filter for ListUsers API.
// This is the primary use case for masking exemption user selection.
func TestListUsers_PermissionFilter(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	projectID := "permission-test-project"
	projectName := fmt.Sprintf("projects/%s", projectID)

	// Create test project
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

	// Create test users
	// alice - will be projectOwner (has bb.sql.select)
	aliceResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "alice",
			UserType: v1pb.UserType_USER,
			Email:    "alice@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	alice := aliceResp.Msg

	// bob - will be in qa-team group with sqlEditorUser role (has bb.sql.select)
	bobResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "bob",
			UserType: v1pb.UserType_USER,
			Email:    "bob@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	bob := bobResp.Msg

	// charlie - will be projectDeveloper (no bb.sql.select)
	charlieResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "charlie",
			UserType: v1pb.UserType_USER,
			Email:    "charlie@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	charlie := charlieResp.Msg

	// eve - not assigned to project at all
	eveResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "eve",
			UserType: v1pb.UserType_USER,
			Email:    "eve@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	eve := eveResp.Msg

	// Create qa-team group with bob as member
	groupResp, err := ctl.groupServiceClient.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
		Group: &v1pb.Group{
			Title: "QA Team",
			Members: []*v1pb.GroupMember{
				{
					Member: fmt.Sprintf("users/%s", bob.Email),
					Role:   v1pb.GroupMember_MEMBER,
				},
			},
		},
		GroupEmail: "qa-team@bytebase.com",
	}))
	a.NoError(err)
	qaTeam := groupResp.Msg

	// Get current project IAM policy
	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: project.Name,
	}))
	a.NoError(err)
	policy := policyResp.Msg

	// Set project IAM policy:
	// - alice: projectOwner
	// - qa-team group: sqlEditorUser
	// - charlie: projectDeveloper
	policy.Bindings = append(policy.Bindings,
		&v1pb.Binding{
			Role:    "roles/projectOwner",
			Members: []string{fmt.Sprintf("user:%s", alice.Email)},
		},
		&v1pb.Binding{
			Role:    "roles/sqlEditorUser",
			Members: []string{fmt.Sprintf("group:%s", "qa-team@bytebase.com")},
		},
		&v1pb.Binding{
			Role:    "roles/projectDeveloper",
			Members: []string{fmt.Sprintf("user:%s", charlie.Email)},
		},
	)

	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	// Helper to check if user is in the list
	containsUser := func(users []*v1pb.User, email string) bool {
		return slices.ContainsFunc(users, func(u *v1pb.User) bool {
			return u.Email == email
		})
	}

	// Test Case 1: Project filter only - groups NOT expanded (backward compatibility)
	t.Run("ProjectFilterOnly_GroupsNotExpanded", func(_ *testing.T) {
		listResp, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter: fmt.Sprintf(`project == "%s"`, projectName),
		}))
		a.NoError(err)
		users := listResp.Msg.Users

		// alice and charlie should be returned (direct members)
		a.True(containsUser(users, alice.Email), "alice should be in list (direct projectOwner)")
		a.True(containsUser(users, charlie.Email), "charlie should be in list (direct projectDeveloper)")
		// bob should NOT be returned (groups not expanded)
		a.False(containsUser(users, bob.Email), "bob should NOT be in list (group not expanded)")
	})

	// Test Case 2: ExpandGroups only - groups expanded, all roles
	t.Run("ExpandGroupsOnly_GroupsExpanded", func(_ *testing.T) {
		listResp, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter: fmt.Sprintf(`project == "%s" && expand_groups == true`, projectName),
		}))
		a.NoError(err)
		users := listResp.Msg.Users

		// All project members should be returned including bob via group
		a.True(containsUser(users, alice.Email), "alice should be in list")
		a.True(containsUser(users, charlie.Email), "charlie should be in list")
		a.True(containsUser(users, bob.Email), "bob should be in list (group expanded)")
	})

	// Test Case 3: Permission filter with expand_groups - masking exemption use case
	t.Run("ExpandGroupsWithPermission_MaskingExemption", func(_ *testing.T) {
		listResp, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter: fmt.Sprintf(`project == "%s" && permission == "bb.sql.select" && expand_groups == true`, projectName),
		}))
		a.NoError(err)
		users := listResp.Msg.Users

		// alice (projectOwner has bb.sql.select) - included
		a.True(containsUser(users, alice.Email), "alice should be in list (projectOwner has bb.sql.select)")
		// bob (via qa-team with sqlEditorUser which has bb.sql.select) - included
		a.True(containsUser(users, bob.Email), "bob should be in list (sqlEditorUser via group has bb.sql.select)")
		// charlie (projectDeveloper lacks bb.sql.select) - excluded
		a.False(containsUser(users, charlie.Email), "charlie should NOT be in list (projectDeveloper lacks bb.sql.select)")
		// eve (not assigned) - excluded
		a.False(containsUser(users, "eve@bytebase.com"), "eve should NOT be in list (not assigned to project)")
	})

	// Test Case 4: Permission filter excludes user not in group
	t.Run("PermissionFilter_ExcludesUserNotInGroup", func(_ *testing.T) {
		listResp, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter: fmt.Sprintf(`project == "%s" && permission == "bb.sql.select" && expand_groups == true`, projectName),
		}))
		a.NoError(err)
		users := listResp.Msg.Users

		// eve is not a member of qa-team and not directly assigned
		a.False(containsUser(users, "eve@bytebase.com"), "eve should NOT be in list")
	})

	// Test Case 5: Invalid permission returns error
	t.Run("InvalidPermission_ReturnsError", func(_ *testing.T) {
		_, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter: fmt.Sprintf(`project == "%s" && permission == "invalid.permission"`, projectName),
		}))
		a.Error(err)
		a.Contains(err.Error(), "unknown permission")
	})

	// Cleanup: Delete all created resources in reverse order of creation
	// 1. Delete group first (before deleting users who are members)
	_, err = ctl.groupServiceClient.DeleteGroup(ctx, connect.NewRequest(&v1pb.DeleteGroupRequest{
		Name: qaTeam.Name,
	}))
	a.NoError(err)

	// 2. Delete users
	for _, userName := range []string{alice.Name, bob.Name, charlie.Name, eve.Name} {
		_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
			Name: userName,
		}))
		a.NoError(err)
	}

	// 3. Delete project
	_, err = ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{
		Name:  project.Name,
		Force: true,
	}))
	a.NoError(err)
}

// TestListGroups_PermissionFilter tests the permission filter for ListGroups API.
func TestListGroups_PermissionFilter(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	projectID := "group-permission-test-project"
	projectName := fmt.Sprintf("projects/%s", projectID)

	// Create test project
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

	// Create test user for group membership
	userResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "groupuser",
			UserType: v1pb.UserType_USER,
			Email:    "groupuser@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	groupUser := userResp.Msg

	// Create groups
	// qa-team - will have sqlEditorUser role (has bb.sql.select)
	qaTeamResp, err := ctl.groupServiceClient.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
		Group: &v1pb.Group{
			Title: "QA Team",
			Members: []*v1pb.GroupMember{
				{Member: fmt.Sprintf("users/%s", groupUser.Email), Role: v1pb.GroupMember_MEMBER},
			},
		},
		GroupEmail: "qa-team-perm@bytebase.com",
	}))
	a.NoError(err)
	qaTeam := qaTeamResp.Msg

	// dev-team - will have projectDeveloper role (no bb.sql.select)
	devTeamResp, err := ctl.groupServiceClient.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
		Group: &v1pb.Group{
			Title: "Dev Team",
			Members: []*v1pb.GroupMember{
				{Member: fmt.Sprintf("users/%s", groupUser.Email), Role: v1pb.GroupMember_MEMBER},
			},
		},
		GroupEmail: "dev-team-perm@bytebase.com",
	}))
	a.NoError(err)
	devTeam := devTeamResp.Msg

	// unassigned-team - not assigned to project
	unassignedTeamResp, err := ctl.groupServiceClient.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
		Group: &v1pb.Group{
			Title: "Unassigned Team",
			Members: []*v1pb.GroupMember{
				{Member: fmt.Sprintf("users/%s", groupUser.Email), Role: v1pb.GroupMember_MEMBER},
			},
		},
		GroupEmail: "unassigned-team-perm@bytebase.com",
	}))
	a.NoError(err)
	unassignedTeam := unassignedTeamResp.Msg

	// Get current project IAM policy
	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: project.Name,
	}))
	a.NoError(err)
	policy := policyResp.Msg

	// Set project IAM policy:
	// - qa-team: sqlEditorUser (has bb.sql.select)
	// - dev-team: projectDeveloper (no bb.sql.select)
	policy.Bindings = append(policy.Bindings,
		&v1pb.Binding{
			Role:    "roles/sqlEditorUser",
			Members: []string{fmt.Sprintf("group:%s", "qa-team-perm@bytebase.com")},
		},
		&v1pb.Binding{
			Role:    "roles/projectDeveloper",
			Members: []string{fmt.Sprintf("group:%s", "dev-team-perm@bytebase.com")},
		},
	)

	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	// Helper to check if group is in the list
	containsGroup := func(groups []*v1pb.Group, email string) bool {
		return slices.ContainsFunc(groups, func(g *v1pb.Group) bool {
			return g.Email == email
		})
	}

	// Test Case 1: Project filter only - all groups assigned to project
	t.Run("ProjectFilterOnly", func(_ *testing.T) {
		listResp, err := ctl.groupServiceClient.ListGroups(ctx, connect.NewRequest(&v1pb.ListGroupsRequest{
			Filter: fmt.Sprintf(`project == "%s"`, projectName),
		}))
		a.NoError(err)
		groups := listResp.Msg.Groups

		// qa-team and dev-team should be returned (assigned to project)
		a.True(containsGroup(groups, "qa-team-perm@bytebase.com"), "qa-team should be in list")
		a.True(containsGroup(groups, "dev-team-perm@bytebase.com"), "dev-team should be in list")
		// unassigned-team should NOT be returned
		a.False(containsGroup(groups, "unassigned-team-perm@bytebase.com"), "unassigned-team should NOT be in list")
	})

	// Test Case 2: Permission filter - only groups with bb.sql.select permission
	t.Run("PermissionFilter_SqlSelect", func(_ *testing.T) {
		listResp, err := ctl.groupServiceClient.ListGroups(ctx, connect.NewRequest(&v1pb.ListGroupsRequest{
			Filter: fmt.Sprintf(`project == "%s" && permission == "bb.sql.select"`, projectName),
		}))
		a.NoError(err)
		groups := listResp.Msg.Groups

		// qa-team (sqlEditorUser has bb.sql.select) - included
		a.True(containsGroup(groups, "qa-team-perm@bytebase.com"), "qa-team should be in list (sqlEditorUser has bb.sql.select)")
		// dev-team (projectDeveloper lacks bb.sql.select) - excluded
		a.False(containsGroup(groups, "dev-team-perm@bytebase.com"), "dev-team should NOT be in list (projectDeveloper lacks bb.sql.select)")
		// unassigned-team - excluded (not assigned to project)
		a.False(containsGroup(groups, "unassigned-team-perm@bytebase.com"), "unassigned-team should NOT be in list")
	})

	// Test Case 3: Invalid permission returns error
	t.Run("InvalidPermission_ReturnsError", func(_ *testing.T) {
		_, err := ctl.groupServiceClient.ListGroups(ctx, connect.NewRequest(&v1pb.ListGroupsRequest{
			Filter: fmt.Sprintf(`project == "%s" && permission == "invalid.permission"`, projectName),
		}))
		a.Error(err)
		a.Contains(err.Error(), "unknown permission")
	})

	// Cleanup
	for _, groupName := range []string{qaTeam.Name, devTeam.Name, unassignedTeam.Name} {
		_, err = ctl.groupServiceClient.DeleteGroup(ctx, connect.NewRequest(&v1pb.DeleteGroupRequest{
			Name: groupName,
		}))
		a.NoError(err)
	}

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: groupUser.Name,
	}))
	a.NoError(err)

	_, err = ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{
		Name:  project.Name,
		Force: true,
	}))
	a.NoError(err)
}

// TestListUsers_Pagination tests pagination for ListUsers API with new filters.
func TestListUsers_Pagination(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	projectID := "pagination-test-project"
	projectName := fmt.Sprintf("projects/%s", projectID)

	// Create test project
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

	// Create multiple test users
	var users []*v1pb.User
	for i := 0; i < 5; i++ {
		userResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
			User: &v1pb.User{
				Title:    fmt.Sprintf("paginationuser%d", i),
				UserType: v1pb.UserType_USER,
				Email:    fmt.Sprintf("paginationuser%d@bytebase.com", i),
				Password: "1024bytebase",
			},
		}))
		a.NoError(err)
		users = append(users, userResp.Msg)
	}

	// Get current project IAM policy and add all users as projectOwner
	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: project.Name,
	}))
	a.NoError(err)
	policy := policyResp.Msg

	var members []string
	for _, u := range users {
		members = append(members, fmt.Sprintf("user:%s", u.Email))
	}
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/projectOwner",
		Members: members,
	})

	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	// Test pagination with project filter
	t.Run("Pagination_ProjectFilter", func(_ *testing.T) {
		// First page
		listResp, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter:   fmt.Sprintf(`project == "%s"`, projectName),
			PageSize: 2,
		}))
		a.NoError(err)
		a.Len(listResp.Msg.Users, 2)
		a.NotEmpty(listResp.Msg.NextPageToken)

		// Second page
		listResp2, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter:    fmt.Sprintf(`project == "%s"`, projectName),
			PageSize:  2,
			PageToken: listResp.Msg.NextPageToken,
		}))
		a.NoError(err)
		a.Len(listResp2.Msg.Users, 2)

		// Verify no duplicate users between pages
		for _, u1 := range listResp.Msg.Users {
			for _, u2 := range listResp2.Msg.Users {
				a.NotEqual(u1.Email, u2.Email, "Users should not duplicate across pages")
			}
		}
	})

	// Test pagination with permission filter
	t.Run("Pagination_PermissionFilter", func(_ *testing.T) {
		// First page with permission filter
		listResp, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter:   fmt.Sprintf(`project == "%s" && permission == "bb.sql.select"`, projectName),
			PageSize: 2,
		}))
		a.NoError(err)
		a.Len(listResp.Msg.Users, 2)
		a.NotEmpty(listResp.Msg.NextPageToken)

		// Second page
		listResp2, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter:    fmt.Sprintf(`project == "%s" && permission == "bb.sql.select"`, projectName),
			PageSize:  2,
			PageToken: listResp.Msg.NextPageToken,
		}))
		a.NoError(err)
		a.Len(listResp2.Msg.Users, 2)
	})

	// Test pagination with expand_groups filter
	t.Run("Pagination_ExpandGroupsFilter", func(_ *testing.T) {
		listResp, err := ctl.userServiceClient.ListUsers(ctx, connect.NewRequest(&v1pb.ListUsersRequest{
			Filter:   fmt.Sprintf(`project == "%s" && expand_groups == true`, projectName),
			PageSize: 2,
		}))
		a.NoError(err)
		a.Len(listResp.Msg.Users, 2)
		a.NotEmpty(listResp.Msg.NextPageToken)
	})

	// Cleanup
	for _, u := range users {
		_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
			Name: u.Name,
		}))
		a.NoError(err)
	}

	_, err = ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{
		Name:  project.Name,
		Force: true,
	}))
	a.NoError(err)
}

// TestListGroups_Pagination tests pagination for ListGroups API with permission filter.
func TestListGroups_Pagination(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	projectID := "group-pagination-test-project"
	projectName := fmt.Sprintf("projects/%s", projectID)

	// Create test project
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

	// Create a user for group membership
	userResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "grouppaguser",
			UserType: v1pb.UserType_USER,
			Email:    "grouppaguser@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	groupUser := userResp.Msg

	// Create multiple test groups
	var groups []*v1pb.Group
	for i := 0; i < 5; i++ {
		groupResp, err := ctl.groupServiceClient.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
			Group: &v1pb.Group{
				Title: fmt.Sprintf("Pagination Group %d", i),
				Members: []*v1pb.GroupMember{
					{Member: fmt.Sprintf("users/%s", groupUser.Email), Role: v1pb.GroupMember_MEMBER},
				},
			},
			GroupEmail: fmt.Sprintf("pagination-group-%d@bytebase.com", i),
		}))
		a.NoError(err)
		groups = append(groups, groupResp.Msg)
	}

	// Get current project IAM policy and add all groups as sqlEditorUser
	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: project.Name,
	}))
	a.NoError(err)
	policy := policyResp.Msg

	var members []string
	for _, g := range groups {
		members = append(members, fmt.Sprintf("group:%s", g.Email))
	}
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/sqlEditorUser",
		Members: members,
	})

	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	// Test pagination with project filter
	t.Run("Pagination_ProjectFilter", func(_ *testing.T) {
		// First page
		listResp, err := ctl.groupServiceClient.ListGroups(ctx, connect.NewRequest(&v1pb.ListGroupsRequest{
			Filter:   fmt.Sprintf(`project == "%s"`, projectName),
			PageSize: 2,
		}))
		a.NoError(err)
		a.Len(listResp.Msg.Groups, 2)
		a.NotEmpty(listResp.Msg.NextPageToken)

		// Second page
		listResp2, err := ctl.groupServiceClient.ListGroups(ctx, connect.NewRequest(&v1pb.ListGroupsRequest{
			Filter:    fmt.Sprintf(`project == "%s"`, projectName),
			PageSize:  2,
			PageToken: listResp.Msg.NextPageToken,
		}))
		a.NoError(err)
		a.Len(listResp2.Msg.Groups, 2)

		// Verify no duplicate groups between pages
		for _, g1 := range listResp.Msg.Groups {
			for _, g2 := range listResp2.Msg.Groups {
				a.NotEqual(g1.Email, g2.Email, "Groups should not duplicate across pages")
			}
		}
	})

	// Test pagination with permission filter
	t.Run("Pagination_PermissionFilter", func(_ *testing.T) {
		// First page with permission filter
		listResp, err := ctl.groupServiceClient.ListGroups(ctx, connect.NewRequest(&v1pb.ListGroupsRequest{
			Filter:   fmt.Sprintf(`project == "%s" && permission == "bb.sql.select"`, projectName),
			PageSize: 2,
		}))
		a.NoError(err)
		a.Len(listResp.Msg.Groups, 2)
		a.NotEmpty(listResp.Msg.NextPageToken)

		// Second page
		listResp2, err := ctl.groupServiceClient.ListGroups(ctx, connect.NewRequest(&v1pb.ListGroupsRequest{
			Filter:    fmt.Sprintf(`project == "%s" && permission == "bb.sql.select"`, projectName),
			PageSize:  2,
			PageToken: listResp.Msg.NextPageToken,
		}))
		a.NoError(err)
		a.Len(listResp2.Msg.Groups, 2)
	})

	// Cleanup
	for _, g := range groups {
		_, err = ctl.groupServiceClient.DeleteGroup(ctx, connect.NewRequest(&v1pb.DeleteGroupRequest{
			Name: g.Name,
		}))
		a.NoError(err)
	}

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: groupUser.Name,
	}))
	a.NoError(err)

	_, err = ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{
		Name:  project.Name,
		Force: true,
	}))
	a.NoError(err)
}
