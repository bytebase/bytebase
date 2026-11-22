package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestCreateDatabaseKeepsEarlierDatabasesRegistered pins what tests with
// parallel subtests on one instance depend on: the instance sync marks deleted
// every database the instance's sync list leaves out, so createDatabase writes
// that list from every database it has registered rather than from a read of
// the instance, which the server can serve from a cache filled before another
// subtest's update. The out-of-band write below stands in for such a read.
func TestCreateDatabaseKeepsEarlierDatabasesRegistered(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startProject(ctx, t)

	pgContainer := sharedPgTarget(t)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			SyncDatabases: &v1pb.SyncDatabases{},
			Title:         "sync list instance",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	first, second := uniqueDB("sync_list_first"), uniqueDB("sync_list_second")
	a.NoError(ctl.createDatabase(ctx, ctl.project, instance, nil, first, "postgres"))

	_, err = ctl.instanceServiceClient.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
		Instance: &v1pb.Instance{
			Name:          instance.Name,
			SyncDatabases: &v1pb.SyncDatabases{},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"sync_databases"}},
	}))
	a.NoError(err)

	a.NoError(ctl.createDatabase(ctx, ctl.project, instance, nil, second, "postgres"))

	firstResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, first),
	}))
	a.NoError(err)
	a.Equal(v1pb.State_ACTIVE, firstResp.Msg.State, "registering a database must leave the ones registered before it")
}
