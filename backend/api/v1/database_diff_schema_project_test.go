package v1

import (
	"context"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testpg"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// The ACL interceptor authorizes request.name only, so the handler confines the
// changelog target to the source's project. One container, since the fixture is
// the expensive part.
func TestDiffSchemaTargetProject(t *testing.T) {
	ctx, stores, instanceID, changelogByDatabase := setupDiffSchemaProjectTest(t)
	service := NewDatabaseService(stores, nil, nil, nil, nil)

	// Databases without a changelog still get a well-formed name, so a
	// rejection is about the project and not about a malformed one.
	changelogOn := func(databaseName string) string {
		changelogID, ok := changelogByDatabase[databaseName]
		if !ok {
			changelogID = "1"
		}
		return common.FormatChangelog(instanceID, databaseName, changelogID)
	}
	diffAgainstChangelog := func(sourceName, changelogName string) (string, error) {
		response, err := service.DiffSchema(ctx, connect.NewRequest(&v1pb.DiffSchemaRequest{
			Name:   sourceName,
			Target: &v1pb.DiffSchemaRequest_Changelog{Changelog: changelogName},
		}))
		if err != nil {
			return "", err
		}
		return response.Msg.Diff, nil
	}

	t.Run("same project diffs", func(t *testing.T) {
		// The diff must actually run: a gate that rejected everything would
		// still satisfy the negative cases below.
		diff, err := diffAgainstChangelog(common.FormatDatabase(instanceID, "app-a"), changelogOn("app-a"))
		require.NoError(t, err)
		require.Empty(t, diff)
	})

	t.Run("changelog source resolves to its database", func(t *testing.T) {
		// request.name may itself be a changelog.
		diff, err := diffAgainstChangelog(changelogOn("app-a"), changelogOn("app-a"))
		require.NoError(t, err)
		require.Empty(t, diff)
	})

	t.Run("foreign project is rejected", func(t *testing.T) {
		// app-b's changelog is real, so without the gate this succeeds and
		// returns another project's schema.
		_, err := diffAgainstChangelog(common.FormatDatabase(instanceID, "app-a"), changelogOn("app-b"))
		require.Error(t, err)
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		require.NotContains(t, err.Error(), "project-b")

		// Indistinguishable from a database that does not exist, or the error
		// itself answers the question.
		_, missing := diffAgainstChangelog(common.FormatDatabase(instanceID, "app-a"), changelogOn("no-such-db"))
		require.Equal(t, connect.CodeOf(err), connect.CodeOf(missing))
	})

	t.Run("foreign changelog id under an owned database is rejected", func(t *testing.T) {
		// Changelog ids are globally unique, so the path's database name alone
		// does not establish which changelog it addresses. Same error as an id
		// that does not exist, which is what the pre-existing not-found path
		// already returns.
		_, err := diffAgainstChangelog(
			common.FormatDatabase(instanceID, "app-a"),
			common.FormatChangelog(instanceID, "app-a", changelogByDatabase["app-b"]),
		)
		require.Error(t, err)

		_, missing := diffAgainstChangelog(
			common.FormatDatabase(instanceID, "app-a"),
			common.FormatChangelog(instanceID, "app-a", "999999"),
		)
		require.Equal(t, connect.CodeOf(missing), connect.CodeOf(err))
	})

	t.Run("schema target skips the gate", func(t *testing.T) {
		// One-resource requests must not reach the gate at all.
		response, err := service.DiffSchema(ctx, connect.NewRequest(&v1pb.DiffSchemaRequest{
			Name:   common.FormatDatabase(instanceID, "app-a"),
			Target: &v1pb.DiffSchemaRequest_Schema{Schema: ""},
		}))
		require.NoError(t, err)
		require.Empty(t, response.Msg.Diff)
	})
}

func setupDiffSchemaProjectTest(t *testing.T) (context.Context, *store.Store, string, map[string]string) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, stores, _ := testpg.New(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-a', 'default', 'Project A'),
			('project-b', 'default', 'Project B');
	`)
	require.NoError(t, err)

	// A workspace-level instance whose databases sit in different projects: the
	// shape that lets one name reach across a project boundary.
	instanceID := "shared-instance"
	_, err = stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: instanceID,
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine: storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{
				Id:   "admin",
				Type: storepb.DataSourceType_ADMIN,
			}},
		},
	})
	require.NoError(t, err)

	for databaseName, projectID := range map[string]string{"app-a": "project-a", "app-b": "project-b"} {
		_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
			InstanceID:   instanceID,
			DatabaseName: databaseName,
			ProjectID:    projectID,
			Metadata:     &storepb.DatabaseMetadata{},
		})
		require.NoError(t, err)
		require.NoError(t, stores.UpsertDBSchema(ctx, instanceID, databaseName,
			&storepb.DatabaseSchemaMetadata{Name: databaseName}, &storepb.DatabaseConfig{}, nil))
	}

	// The foreign changelog must exist, or the cross-project case would fail
	// for the wrong reason and still pass.
	changelogByDatabase := map[string]string{}
	for _, databaseName := range []string{"app-a", "app-b"} {
		changelogID, err := stores.CreateChangelog(ctx, &store.ChangelogMessage{
			InstanceID:   instanceID,
			DatabaseName: databaseName,
			Payload:      &storepb.ChangelogPayload{},
			Status:       store.ChangelogStatusDone,
		})
		require.NoError(t, err)
		changelogByDatabase[databaseName] = changelogID
	}

	return ctx, stores, instanceID, changelogByDatabase
}
