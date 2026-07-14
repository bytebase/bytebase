package tests

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestCreateInstanceAssignsSyncedDatabasesToProject(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "project-aware-sync")
	a.NoError(err)
	databaseName := "project_assignment"
	createSQLiteDatabase(t, instanceDir, databaseName)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId:             generateRandomString("instance"),
		InitialDatabaseProject: ctl.project.Name,
		Instance: &v1pb.Instance{
			Title:       "project-aware-sync",
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	a.Equal(ctl.project.Name, databaseResp.Msg.Project)
}

func TestCreateInstanceWithoutProjectKeepsDefaultProject(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "default-project-sync")
	a.NoError(err)
	databaseName := "default_project_assignment"
	createSQLiteDatabase(t, instanceDir, databaseName)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "default-project-sync",
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
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

func TestListInstanceDatabaseBeforeCreate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "preview-databases")
	a.NoError(err)
	databaseName := "preview_database"
	createSQLiteDatabase(t, instanceDir, databaseName)

	resp, err := ctl.instanceServiceClient.ListInstanceDatabase(ctx, connect.NewRequest(&v1pb.ListInstanceDatabaseRequest{
		Name: fmt.Sprintf("instances/%s", generateRandomString("instance")),
		Instance: &v1pb.Instance{
			Title:       "preview-databases",
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	a.Contains(resp.Msg.Databases, databaseName)
}

func createSQLiteDatabase(t *testing.T, dir, databaseName string) {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(dir, fmt.Sprintf("%s.db", databaseName)))
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE t(id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
}
