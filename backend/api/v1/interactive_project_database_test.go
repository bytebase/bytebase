package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestBOT36SQLPrepareRelatedMessageRequiresCanonicalActiveOwner(t *testing.T) {
	ctx, stores, projectID, instanceID, databaseName := setupBOT36ProjectDatabase(t)
	service := NewSQLService(stores, nil, nil, nil, nil, nil)

	_, _, _, err := service.prepareRelatedMessage(ctx, common.FormatDatabase(instanceID, databaseName))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, _, _, err = service.prepareRelatedMessage(ctx, common.FormatProjectDatabase("other-project", instanceID, databaseName))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, _, _, err = service.prepareRelatedMessage(ctx, common.FormatProjectDatabase(projectID, instanceID, databaseName))
	require.NoError(t, err)
	_, _, _, err = service.prepareRelatedMessage(ctx, common.FormatDatabase("workspace-instance", "shared"))
	require.NoError(t, err)

	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: projectID,
		Workspace:  "default",
		Delete:     new(true),
	}))
	_, _, _, err = service.prepareRelatedMessage(ctx, common.FormatProjectDatabase(projectID, instanceID, databaseName))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestBOT36WorksheetDatabasesUseCanonicalOwningProjectNames(t *testing.T) {
	ctx, stores, projectID, instanceID, databaseName := setupBOT36ProjectDatabase(t)
	service := NewWorksheetService(stores, nil)

	_, err := service.getWorksheetDatabase(ctx, projectID, common.FormatDatabase(instanceID, databaseName))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = service.getWorksheetDatabase(ctx, projectID, common.FormatProjectDatabase("other-project", instanceID, databaseName))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	database, err := service.getWorksheetDatabase(ctx, projectID, common.FormatProjectDatabase(projectID, instanceID, databaseName))
	require.NoError(t, err)
	require.Equal(t, databaseName, database.DatabaseName)

	worksheet, err := service.convertWorksheetToAPI(ctx, &store.WorkSheetMessage{
		ProjectID:    projectID,
		ResourceID:   "worksheet-a",
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, common.FormatProjectDatabase(projectID, instanceID, databaseName), worksheet.Database)

	workspaceDatabase, err := service.getWorksheetDatabase(ctx, projectID, common.FormatDatabase("workspace-instance", "shared"))
	require.NoError(t, err)
	workspaceWorksheet, err := service.convertWorksheetToAPI(ctx, &store.WorkSheetMessage{
		ProjectID:    projectID,
		ResourceID:   "worksheet-b",
		InstanceID:   &workspaceDatabase.InstanceID,
		DatabaseName: &workspaceDatabase.DatabaseName,
	})
	require.NoError(t, err)
	require.Equal(t, common.FormatDatabase("workspace-instance", "shared"), workspaceWorksheet.Database)
}

func TestBOT36AccessGrantTargetsRequireCanonicalOwningProject(t *testing.T) {
	ctx, stores, projectID, instanceID, databaseName := setupBOT36ProjectDatabase(t)
	service := NewAccessGrantService(stores, nil, nil, nil)

	_, err := service.validateAccessGrantTargets(ctx, projectID, []string{common.FormatDatabase(instanceID, databaseName)})
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = service.validateAccessGrantTargets(ctx, projectID, []string{common.FormatProjectDatabase(projectID, instanceID, databaseName), common.FormatProjectDatabase("other-project", instanceID, databaseName)})
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	instance, err := service.validateAccessGrantTargets(ctx, projectID, []string{common.FormatProjectDatabase(projectID, instanceID, databaseName)})
	require.NoError(t, err)
	require.Equal(t, instanceID, instance.ResourceID)
	instance, err = service.validateAccessGrantTargets(ctx, projectID, []string{common.FormatDatabase("workspace-instance", "shared")})
	require.NoError(t, err)
	require.Equal(t, "workspace-instance", instance.ResourceID)
}

func setupBOT36ProjectDatabase(t *testing.T) (context.Context, *store.Store, string, string, string) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{Email: "user@example.com"})
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	require.NoError(t, migrator.MigrateSchema(ctx, container.GetDB()))
	_, err := container.GetDB().ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A'), ('other-project', 'default', 'Other Project');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	projectID := "project-a"
	instanceID := "instance-a"
	_, err = stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: instanceID,
		Workspace:  "default",
		ProjectID:  &projectID,
		Metadata: &storepb.Instance{DataSources: []*storepb.DataSource{{
			Id:   "admin",
			Type: storepb.DataSourceType_ADMIN,
		}}},
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
	_, err = stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "workspace-instance",
		Workspace:  "default",
		Metadata: &storepb.Instance{DataSources: []*storepb.DataSource{{
			Id:   "admin",
			Type: storepb.DataSourceType_ADMIN,
		}}},
	})
	require.NoError(t, err)
	_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "workspace-instance",
		DatabaseName: "shared",
		ProjectID:    projectID,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	return ctx, stores, projectID, instanceID, databaseName
}
