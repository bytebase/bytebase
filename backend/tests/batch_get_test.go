package tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// The four BatchGet RPCs answer one contract: one resource per requested name,
// in request order, and the first name that does not resolve fails the whole
// call. No partial responses (AIP-231).

func TestBatchGetProjectsIsOrderedAndAtomic(t *testing.T) {
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

	// B before A, so a response in creation order would not match.
	response, err := ctl.projectServiceClient.BatchGetProjects(ctx, connect.NewRequest(&v1pb.BatchGetProjectsRequest{
		Names: []string{projectB.Msg.Name, projectA.Msg.Name},
	}))
	a.NoError(err)
	a.Equal([]string{projectB.Msg.Name, projectA.Msg.Name}, projectNames(response.Msg.Projects),
		"one project per requested name, in request order")

	_, err = ctl.projectServiceClient.BatchGetProjects(ctx, connect.NewRequest(&v1pb.BatchGetProjectsRequest{
		Names: []string{projectA.Msg.Name, "projects/no-such-project"},
	}))
	a.Error(err, "one name that does not resolve fails the whole call")
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
}

func TestBatchGetUsersIsOrderedAndAtomic(t *testing.T) {
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
	response, err := ctl.userServiceClient.BatchGetUsers(ctx, connect.NewRequest(&v1pb.BatchGetUsersRequest{
		Names: []string{second.Msg.Name, first.Msg.Name},
	}))
	a.NoError(err)
	names := make([]string, 0, len(response.Msg.Users))
	for _, user := range response.Msg.Users {
		names = append(names, user.Name)
	}
	a.Equal([]string{second.Msg.Name, first.Msg.Name}, names)

	_, err = ctl.userServiceClient.BatchGetUsers(ctx, connect.NewRequest(&v1pb.BatchGetUsersRequest{
		Names: []string{first.Msg.Name, "users/batch-get-nobody@example.com"},
	}))
	a.Error(err, "a user that does not exist fails the whole call")
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
}

func TestBatchGetGroupsIsAtomic(t *testing.T) {
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

	response, err := ctl.groupServiceClient.BatchGetGroups(ctx, connect.NewRequest(&v1pb.BatchGetGroupsRequest{
		Names: []string{group.Msg.Name},
	}))
	a.NoError(err)
	a.Len(response.Msg.Groups, 1)
	a.Equal(group.Msg.Name, response.Msg.Groups[0].Name)

	// This used to swallow every error and answer 200 with the group missing,
	// which made a store failure indistinguishable from "no such group".
	_, err = ctl.groupServiceClient.BatchGetGroups(ctx, connect.NewRequest(&v1pb.BatchGetGroupsRequest{
		Names: []string{group.Msg.Name, "groups/no-such-group@example.com"},
	}))
	a.Error(err, "a group that does not exist fails the whole call")
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
}

func TestBatchGetDatabasesIsAtomic(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// This used to answer 200 with the database silently missing from the list.
	_, err = ctl.databaseServiceClient.BatchGetDatabases(ctx, connect.NewRequest(&v1pb.BatchGetDatabasesRequest{
		Parent: "-",
		Names:  []string{"instances/no-such-instance/databases/no-such-db"},
	}))
	a.Error(err, "a database that does not exist fails the whole call")
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

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
