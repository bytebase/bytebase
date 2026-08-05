package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCheckReleaseDatabaseTargetsKeepCanonicalNames(t *testing.T) {
	ctx, stores, instanceID, databaseName := setupProjectInstanceReleaseCheckTest(t)
	projectID := "project-a"
	projectTarget := common.FormatProjectDatabase(projectID, instanceID, databaseName)
	workspaceTarget := common.FormatDatabase(instanceID, databaseName)

	response, err := NewReleaseService(stores, sheet.NewManager(), nil, nil).CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent:  common.FormatProject(projectID),
		Release: declarativeReleaseWithDisallowedStatement(),
		Targets: []string{projectTarget},
	}))
	require.NoError(t, err)
	require.Len(t, response.Msg.Results, 1)
	require.Equal(t, projectTarget, response.Msg.Results[0].Target)

	_, err = NewReleaseService(stores, sheet.NewManager(), nil, nil).CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent:  common.FormatProject(projectID),
		Release: declarativeReleaseWithDisallowedStatement(),
		Targets: []string{workspaceTarget},
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCheckReleaseRejectsCrossProjectTargets(t *testing.T) {
	ctx, stores, instanceID, databaseName := setupProjectInstanceReleaseCheckTest(t)
	service := NewReleaseService(stores, sheet.NewManager(), nil, nil)
	request := func(target string) *connect.Request[v1pb.CheckReleaseRequest] {
		return connect.NewRequest(&v1pb.CheckReleaseRequest{
			Parent:  common.FormatProject("project-a"),
			Release: declarativeReleaseWithDisallowedStatement(),
			Targets: []string{target},
		})
	}

	for _, target := range []string{
		common.FormatProjectDatabase("project-b", instanceID, databaseName),
		common.FormatProject("project-b") + "/databaseGroups/all",
	} {
		_, err := service.CheckRelease(ctx, request(target))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), target)
	}
}

func TestCheckReleaseRejectsArchivedProject(t *testing.T) {
	ctx, stores, instanceID, databaseName := setupProjectInstanceReleaseCheckTest(t)
	archived := true
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Delete:     &archived,
	}))

	_, err := NewReleaseService(stores, sheet.NewManager(), nil, nil).CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent:  common.FormatProject("project-a"),
		Release: declarativeReleaseWithDisallowedStatement(),
		Targets: []string{common.FormatProjectDatabase("project-a", instanceID, databaseName)},
	}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func declarativeReleaseWithDisallowedStatement() *v1pb.Release {
	return &v1pb.Release{
		Type: v1pb.Release_DECLARATIVE,
		Files: []*v1pb.Release_File{{
			Path:      "schema.sql",
			Version:   "1",
			Statement: []byte("DROP TABLE obsolete;"),
		}},
	}
}

func setupProjectInstanceReleaseCheckTest(t *testing.T) (context.Context, *store.Store, string, string) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-a', 'default', 'Project A'),
			('project-b', 'default', 'Project B');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	projectID := "project-a"
	instanceID := "project-instance"
	_, err = stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: instanceID,
		Workspace:  "default",
		ProjectID:  &projectID,
		Metadata: &storepb.Instance{
			Engine: storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{
				Id:   "admin",
				Type: storepb.DataSourceType_ADMIN,
			}},
		},
	})
	require.NoError(t, err)

	databaseName := "app"
	_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		ProjectID:    projectID,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	return ctx, stores, instanceID, databaseName
}
