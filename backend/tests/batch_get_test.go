package tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// The BatchGet RPCs answer one resource per requested name, in request order,
// and the first name that does not resolve fails the whole call. Each check
// below is a behavior that used to be a silent omission.
func TestBatchGetIsOrderedAndAtomic(t *testing.T) {
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
	users, err := ctl.userServiceClient.BatchGetUsers(ctx, connect.NewRequest(&v1pb.BatchGetUsersRequest{
		Names: []string{second.Msg.Name, first.Msg.Name},
	}))
	a.NoError(err)
	names := make([]string, 0, len(users.Msg.Users))
	for _, user := range users.Msg.Users {
		names = append(names, user.Name)
	}
	a.Equal([]string{second.Msg.Name, first.Msg.Name}, names, "users come back in request order")

	_, err = ctl.userServiceClient.BatchGetUsers(ctx, connect.NewRequest(&v1pb.BatchGetUsersRequest{
		Names: []string{first.Msg.Name, "users/batch-get-nobody@example.com"},
	}))
	a.Error(err, "a user that does not exist fails the whole call")
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// This used to swallow every error, which made a store failure
	// indistinguishable from "no such group".
	_, err = ctl.groupServiceClient.BatchGetGroups(ctx, connect.NewRequest(&v1pb.BatchGetGroupsRequest{
		Names: []string{"groups/no-such-group@example.com"},
	}))
	a.Error(err, "a group that does not exist fails the whole call")
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// This used to answer 200 with the database missing from the list.
	_, err = ctl.databaseServiceClient.BatchGetDatabases(ctx, connect.NewRequest(&v1pb.BatchGetDatabasesRequest{
		Parent: "-",
		Names:  []string{"instances/no-such-instance/databases/no-such-db"},
	}))
	a.Error(err, "a database that does not exist fails the whole call")
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
}
