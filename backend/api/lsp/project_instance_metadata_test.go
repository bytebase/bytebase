package lsp

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestMetadataProjectInstanceScope(t *testing.T) {
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
		Metadata: &storepb.Instance{DataSources: []*storepb.DataSource{{
			Id:   "admin",
			Type: storepb.DataSourceType_ADMIN,
		}}},
	})
	require.NoError(t, err)

	h := &Handler{store: stores}
	canonicalName := common.FormatProjectInstance(projectID, instanceID)
	h.setMetadata(SetMetadataCommandArguments{InstanceID: canonicalName})
	require.Equal(t, canonicalName, h.metadata.InstanceID)
	require.Equal(t, instanceID, h.getInstanceID())

	instance, err := h.getMetadataInstance(ctx, canonicalName)
	require.NoError(t, err)
	require.Equal(t, instanceID, instance.ResourceID)

	_, err = h.getMetadataInstance(ctx, common.FormatInstance(instanceID))
	require.Error(t, err)
	_, err = h.getMetadataInstance(ctx, common.FormatProjectInstance("project-b", instanceID))
	require.Error(t, err)

	_, err = db.ExecContext(ctx, "UPDATE project SET deleted = TRUE WHERE resource_id = $1", projectID)
	require.NoError(t, err)
	_, err = h.getMetadataInstance(ctx, canonicalName)
	require.Error(t, err)
}
