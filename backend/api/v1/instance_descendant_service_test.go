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
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestProjectInstanceDescendantServicesUseCanonicalNames(t *testing.T) {
	ctx, stores, instanceID, databaseName, changelogID, revisionID := setupProjectInstanceDescendantServiceTest(t)
	projectID := "project-a"
	parent := common.FormatProjectDatabase(projectID, instanceID, databaseName)

	roles, err := NewInstanceRoleService(stores).ListInstanceRoles(ctx, connect.NewRequest(&v1pb.ListInstanceRolesRequest{
		Parent: common.FormatProjectInstance(projectID, instanceID),
	}))
	require.NoError(t, err)
	require.Equal(t, common.FormatProjectInstance(projectID, instanceID)+"/roles/role-a", roles.Msg.Roles[0].Name)

	changelogService := NewChangelogService(stores)
	changelogs, err := changelogService.ListChangelogs(ctx, connect.NewRequest(&v1pb.ListChangelogsRequest{Parent: parent}))
	require.NoError(t, err)
	require.Equal(t, common.FormatProjectChangelog(projectID, instanceID, databaseName, changelogID), changelogs.Msg.Changelogs[0].Name)
	changelog, err := changelogService.GetChangelog(ctx, connect.NewRequest(&v1pb.GetChangelogRequest{
		Name: common.FormatProjectChangelog(projectID, instanceID, databaseName, changelogID),
	}))
	require.NoError(t, err)
	require.Equal(t, common.FormatProjectChangelog(projectID, instanceID, databaseName, changelogID), changelog.Msg.Name)

	revisionService := NewRevisionService(stores)
	revisions, err := revisionService.ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{Parent: parent}))
	require.NoError(t, err)
	require.Equal(t, common.FormatProjectRevision(projectID, instanceID, databaseName, revisionID), revisions.Msg.Revisions[0].Name)
	require.Equal(t, common.FormatSheet(projectID, "sheet-a"), revisions.Msg.Revisions[0].Sheet)
	revision, err := revisionService.GetRevision(ctx, connect.NewRequest(&v1pb.GetRevisionRequest{
		Name: common.FormatProjectRevision(projectID, instanceID, databaseName, revisionID),
	}))
	require.NoError(t, err)
	require.Equal(t, common.FormatProjectRevision(projectID, instanceID, databaseName, revisionID), revision.Msg.Name)
}

func TestProjectInstanceDescendantServicesRejectWrongScopeParents(t *testing.T) {
	ctx, stores, instanceID, databaseName, changelogID, revisionID := setupProjectInstanceDescendantServiceTest(t)
	workspaceParent := common.FormatDatabase(instanceID, databaseName)
	wrongProjectID := "project-b"
	wrongProjectParent := common.FormatProjectDatabase(wrongProjectID, instanceID, databaseName)

	roleService := NewInstanceRoleService(stores)
	_, err := roleService.ListInstanceRoles(ctx, connect.NewRequest(&v1pb.ListInstanceRolesRequest{Parent: common.FormatInstance(instanceID)}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = roleService.ListInstanceRoles(ctx, connect.NewRequest(&v1pb.ListInstanceRolesRequest{
		Parent: common.FormatProjectInstance(wrongProjectID, instanceID),
	}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	changelogService := NewChangelogService(stores)
	_, err = changelogService.ListChangelogs(ctx, connect.NewRequest(&v1pb.ListChangelogsRequest{Parent: workspaceParent}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = changelogService.GetChangelog(ctx, connect.NewRequest(&v1pb.GetChangelogRequest{
		Name: common.FormatProjectChangelog(wrongProjectID, instanceID, databaseName, changelogID),
	}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	revisionService := NewRevisionService(stores)
	_, err = revisionService.ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{Parent: wrongProjectParent}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = revisionService.GetRevision(ctx, connect.NewRequest(&v1pb.GetRevisionRequest{
		Name: common.FormatRevision(instanceID, databaseName, revisionID),
	}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestWorkspaceInstanceDescendantServicesKeepWorkspaceNames(t *testing.T) {
	ctx, stores, instanceID, databaseName, changelogID, revisionID := setupWorkspaceInstanceDescendantServiceTest(t)
	parent := common.FormatDatabase(instanceID, databaseName)

	roles, err := NewInstanceRoleService(stores).ListInstanceRoles(ctx, connect.NewRequest(&v1pb.ListInstanceRolesRequest{
		Parent: common.FormatInstance(instanceID),
	}))
	require.NoError(t, err)
	require.Equal(t, common.FormatInstance(instanceID)+"/roles/role-a", roles.Msg.Roles[0].Name)

	changelogs, err := NewChangelogService(stores).ListChangelogs(ctx, connect.NewRequest(&v1pb.ListChangelogsRequest{Parent: parent}))
	require.NoError(t, err)
	require.Equal(t, common.FormatChangelog(instanceID, databaseName, changelogID), changelogs.Msg.Changelogs[0].Name)

	revisions, err := NewRevisionService(stores).ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{Parent: parent}))
	require.NoError(t, err)
	require.Equal(t, common.FormatRevision(instanceID, databaseName, revisionID), revisions.Msg.Revisions[0].Name)
}

func setupProjectInstanceDescendantServiceTest(t *testing.T) (context.Context, *store.Store, string, string, string, string) {
	t.Helper()
	projectID := "project-a"
	return setupInstanceDescendantServiceTest(t, &projectID)
}

func setupWorkspaceInstanceDescendantServiceTest(t *testing.T) (context.Context, *store.Store, string, string, string, string) {
	t.Helper()
	return setupInstanceDescendantServiceTest(t, nil)
}

func setupInstanceDescendantServiceTest(t *testing.T, instanceProjectID *string) (context.Context, *store.Store, string, string, string, string) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A'), ('project-b', 'default', 'Project B');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	instanceID := "project-instance"
	_, err = stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: instanceID,
		Workspace:  "default",
		ProjectID:  instanceProjectID,
		Metadata: &storepb.Instance{
			DataSources: []*storepb.DataSource{{
				Id:   "admin",
				Type: storepb.DataSourceType_ADMIN,
			}},
			Roles: []*storepb.InstanceRole{{
				Name: "role-a",
			}},
		},
	})
	require.NoError(t, err)

	databaseName := "app"
	databaseProjectID := "project-a"
	_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		ProjectID:    databaseProjectID,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)

	changelogID, err := stores.CreateChangelog(ctx, &store.ChangelogMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		Payload:      &storepb.ChangelogPayload{},
		Status:       store.ChangelogStatusDone,
	})
	require.NoError(t, err)
	revision, err := stores.CreateRevision(ctx, &store.RevisionMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		Version:      "1",
		Payload: &storepb.RevisionPayload{
			SheetSha256: "sheet-a",
			Type:        storepb.SchemaChangeType_VERSIONED,
		},
	})
	require.NoError(t, err)
	return ctx, stores, instanceID, databaseName, changelogID, revision.ResourceID
}
