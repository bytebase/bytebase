package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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

func TestBOT36SavedQueryDatabasesUseCanonicalOwningProjectNames(t *testing.T) {
	ctx, stores, projectID, instanceID, databaseName := setupBOT36ProjectDatabase(t)
	service := NewSavedQueryService(stores, nil)

	_, err := service.validateSavedQueryDatabase(ctx, projectID, common.FormatDatabase(instanceID, databaseName))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = service.validateSavedQueryDatabase(ctx, projectID, common.FormatProjectDatabase("other-project", instanceID, databaseName))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	canonical, err := service.validateSavedQueryDatabase(ctx, projectID, common.FormatProjectDatabase(projectID, instanceID, databaseName))
	require.NoError(t, err)
	require.Equal(t, common.FormatProjectDatabase(projectID, instanceID, databaseName), canonical)

	// The stored canonical name round-trips through the API shape without
	// re-resolution — a dangling reference degrades instead of erroring.
	savedQuery := convertToAPISavedQuery(&store.SavedQueryMessage{
		ProjectID:  projectID,
		ResourceID: "saved-query-a",
		Database:   canonical,
	})
	require.Equal(t, canonical, savedQuery.Database)

	workspaceCanonical, err := service.validateSavedQueryDatabase(ctx, projectID, common.FormatDatabase("workspace-instance", "shared"))
	require.NoError(t, err)
	require.Equal(t, common.FormatDatabase("workspace-instance", "shared"), workspaceCanonical)
}

func TestUpdateSavedQueryKeepsDanglingDatabaseReference(t *testing.T) {
	ctx, stores, projectID, _, _ := setupBOT36ProjectDatabase(t)
	service := NewSavedQueryService(stores, nil)

	// The database reference is a soft link: an autosave that re-sends the
	// stored (now dangling) value must not fail validation — that would
	// brick content saves after the database is deleted or transferred.
	dangling := common.FormatDatabase("deleted-instance", "gone")
	created, err := stores.CreateSavedQuery(ctx, &store.SavedQueryMessage{
		ProjectID: projectID,
		Creator:   "user@example.com",
		Title:     "dangling",
		Statement: "SELECT 1;",
		Database:  dangling,
	})
	require.NoError(t, err)

	updated, err := service.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
		SavedQuery: &v1pb.SavedQuery{
			Name:     common.FormatSavedQuery(projectID, created.ResourceID),
			Database: dangling,
			Content:  []byte("SELECT 2;"),
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"database", "content"}},
	}))
	require.NoError(t, err)
	require.Equal(t, dangling, updated.Msg.Database)
	require.Equal(t, []byte("SELECT 2;"), updated.Msg.Content)

	// An explicit change to a nonexistent database still fails hard.
	_, err = service.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
		SavedQuery: &v1pb.SavedQuery{
			Name:     common.FormatSavedQuery(projectID, created.ResourceID),
			Database: common.FormatDatabase("another-missing", "db"),
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"database"}},
	}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
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
