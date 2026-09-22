package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestCreateInstanceWithoutProjectKeepsDefaultProject(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startProject(ctx, t)

	pgContainer := sharedPgTarget(t)
	databaseName := uniqueDB("default_project_assignment")
	createPgDatabase(t, pgContainer, databaseName)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			SyncDatabases: &v1pb.SyncDatabases{Databases: []string{databaseName}},
			Title:         "default-project-sync",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	a.True(strings.HasPrefix(databaseResp.Msg.Project, "projects/default-") || databaseResp.Msg.Project == "projects/default")
	a.NotEqual(ctl.project.Name, databaseResp.Msg.Project)
}

func TestCreateInstanceWithEmptySyncDatabasesSkipsInitialDatabaseSync(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startProject(ctx, t)

	pgContainer := sharedPgTarget(t)
	databaseName := uniqueDB("unsynced_database")
	createPgDatabase(t, pgContainer, databaseName)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:         "skip-initial-database-sync",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{pgContainer.adminDataSource()},
			SyncDatabases: &v1pb.SyncDatabases{},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	databaseResp, err := ctl.databaseServiceClient.ListDatabases(ctx, connect.NewRequest(&v1pb.ListDatabasesRequest{
		Parent:   instance.Name,
		PageSize: 1000,
	}))
	a.NoError(err)
	for _, database := range databaseResp.Msg.Databases {
		a.NotEqual(fmt.Sprintf("%s/databases/%s", instance.Name, databaseName), database.Name)
	}
}

func TestListInstanceDatabaseBeforeCreate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startProject(ctx, t)

	pgContainer := sharedPgTarget(t)
	databaseName := uniqueDB("preview_database")
	createPgDatabase(t, pgContainer, databaseName)

	resp, err := ctl.instanceServiceClient.ListInstanceDatabase(ctx, connect.NewRequest(&v1pb.ListInstanceDatabaseRequest{
		Name: fmt.Sprintf("instances/%s", generateRandomString("instance")),
		Instance: &v1pb.Instance{
			Title:       "preview-databases",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	a.Contains(resp.Msg.Databases, databaseName)
}

// createPgDatabase creates a database behind Bytebase's back, for the tests
// about what a sync discovers. On the shared target the instance that should
// discover it must name it in sync_databases.
func createPgDatabase(t *testing.T, pgContainer *Container, databaseName string) {
	t.Helper()

	_, err := pgContainer.GetDB().Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName))
	require.NoError(t, err)
}
