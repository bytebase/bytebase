package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// The four BatchGet RPCs answer one contract: resources in request order, a
// repeated name served once, and a name that resolves to nothing reported in
// unmatched_names instead of silently dropped or 404ing the whole call.

func TestBatchGetProjectsReportsMissingNamesInsteadOfFailing(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	projectA, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("bg-a"),
		Project:   &v1pb.Project{Title: "Batch Get A"},
	}))
	a.NoError(err)
	projectB, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("bg-b"),
		Project:   &v1pb.Project{Title: "Batch Get B"},
	}))
	a.NoError(err)

	const missing = "projects/no-such-project"
	// B before A, so a response in store order would not match.
	response, err := ctl.projectServiceClient.BatchGetProjects(ctx, connect.NewRequest(&v1pb.BatchGetProjectsRequest{
		Names: []string{projectB.Msg.Name, missing, projectA.Msg.Name, projectB.Msg.Name},
	}))
	a.NoError(err, "one missing name must not fail the whole batch")
	a.Equal(
		[]string{projectB.Msg.Name, projectA.Msg.Name},
		projectNames(response.Msg.Projects),
		"projects come back in request order, each requested name served once",
	)
	a.Equal([]string{missing}, response.Msg.UnmatchedNames)
}

func TestBatchGetProjectsAuthorizesEachNamedProject(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	mine, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("bg-mine"),
		Project:   &v1pb.Project{Title: "Batch Get Mine"},
	}))
	a.NoError(err)
	theirs, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("bg-theirs"),
		Project:   &v1pb.Project{Title: "Batch Get Theirs"},
	}))
	a.NoError(err)

	// A member whose only project grant is on `mine`. bb.projects.get is not a
	// Workspace Member permission, so authorizing this request against the
	// workspace — which is all the ACL interceptor can do with a repeated names
	// field — denied the call outright.
	const email = "batch-get-member@example.com"
	const password = "1024bytebase"
	created, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Title: "batch get member", Email: email, Password: password},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, created.Msg.Workspace, "user:"+email, "roles/workspaceMember")
	a.NoError(err)

	policy, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: mine.Msg.Name,
	}))
	a.NoError(err)
	policy.Msg.Bindings = append(policy.Msg.Bindings, &v1pb.Binding{
		Role:    "roles/projectOwner",
		Members: []string{fmt.Sprintf("user:%s", email)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: mine.Msg.Name,
		Policy:   policy.Msg,
	}))
	a.NoError(err)

	login, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    email,
		Password: password,
	}))
	a.NoError(err)
	asMember := v1connect.NewProjectServiceClient(ctl.client, ctl.rootURL,
		connect.WithInterceptors(&authInterceptor{token: login.Msg.Token}))

	response, err := asMember.BatchGetProjects(ctx, connect.NewRequest(&v1pb.BatchGetProjectsRequest{
		Names: []string{mine.Msg.Name, theirs.Msg.Name},
	}))
	a.NoError(err, "a project-scoped role must be able to batch get its own projects")
	a.Equal([]string{mine.Msg.Name}, projectNames(response.Msg.Projects))
	a.Equal([]string{theirs.Msg.Name}, response.Msg.UnmatchedNames,
		"a project the caller may not see is unmatched, indistinguishable from one that does not exist")
}

func TestBatchGetUsersReturnsRequestOrder(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	first, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Title: "first", Email: "batch-get-first@example.com", Password: "1024bytebase"},
	}))
	a.NoError(err)
	second, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Title: "second", Email: "batch-get-second@example.com", Password: "1024bytebase"},
	}))
	a.NoError(err)

	// Newest first, so the store's own created_at ordering would invert this.
	const missing = "users/batch-get-nobody@example.com"
	response, err := ctl.userServiceClient.BatchGetUsers(ctx, connect.NewRequest(&v1pb.BatchGetUsersRequest{
		Names: []string{second.Msg.Name, missing, first.Msg.Name, second.Msg.Name},
	}))
	a.NoError(err)
	names := make([]string, 0, len(response.Msg.Users))
	for _, user := range response.Msg.Users {
		names = append(names, user.Name)
	}
	a.Equal([]string{second.Msg.Name, first.Msg.Name}, names)
	a.Equal([]string{missing}, response.Msg.UnmatchedNames)
}

func TestBatchGetGroupsReportsMissingNames(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	group, err := ctl.groupServiceClient.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
		Group:      &v1pb.Group{Title: "Batch Get Group"},
		GroupEmail: "batch-get-group@example.com",
	}))
	a.NoError(err)

	const missing = "groups/no-such-group@example.com"
	response, err := ctl.groupServiceClient.BatchGetGroups(ctx, connect.NewRequest(&v1pb.BatchGetGroupsRequest{
		Names: []string{missing, group.Msg.Name},
	}))
	a.NoError(err)
	a.Len(response.Msg.Groups, 1)
	a.Equal(group.Msg.Name, response.Msg.Groups[0].Name)
	a.Equal([]string{missing}, response.Msg.UnmatchedNames,
		"a group that does not exist is reported, not swallowed with every other error")
}

func TestBatchGetDatabasesReportsMissingNames(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	missing := []string{
		"instances/no-such-instance/databases/no-such-db",
		"instances/no-such-instance/databases/other-db",
	}
	response, err := ctl.databaseServiceClient.BatchGetDatabases(ctx, connect.NewRequest(&v1pb.BatchGetDatabasesRequest{
		Parent: "-",
		Names:  missing,
	}))
	a.NoError(err)
	a.Empty(response.Msg.Databases)
	a.Equal(missing, response.Msg.UnmatchedNames,
		"the caller must be able to tell which names came back empty")

	// A name the caller malformed is still their error, and still fails the call.
	_, err = ctl.databaseServiceClient.BatchGetDatabases(ctx, connect.NewRequest(&v1pb.BatchGetDatabasesRequest{
		Parent: "-",
		Names:  []string{"not-a-database-name"},
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}

func projectNames(projects []*v1pb.Project) []string {
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names
}
