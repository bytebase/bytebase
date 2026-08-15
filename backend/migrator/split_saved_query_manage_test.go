package migrator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// TestSplitSavedQueryManageMigration exercises 3.22/0009's jsonb transform:
// bb.savedQueries.manage expands to the four per-verb permissions —
// deliberately without setIamPolicy — deduplicated, while rows without
// manage and rows with a non-array permissions payload stay untouched.
func TestSplitSavedQueryManageMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainer test in short mode")
	}
	ctx := context.Background()
	pg := testcontainer.GetTestPgContainer(ctx, t)
	defer pg.Close(ctx)
	db := pg.GetDB()

	_, err := db.ExecContext(ctx, `
		CREATE TABLE role (
			resource_id text NOT NULL PRIMARY KEY,
			permissions jsonb NOT NULL DEFAULT '{}'
		);
		INSERT INTO role (resource_id, permissions) VALUES
			('manage-holder', '{"permissions": ["bb.savedQueries.manage", "bb.savedQueries.get", "bb.projects.get"]}'),
			('no-manage', '{"permissions": ["bb.savedQueries.search", "bb.projects.get"]}'),
			('non-array', '{"permissions": {"oops": true}}'),
			('empty', '{}');
	`)
	require.NoError(t, err)

	migration, err := migrationFS.ReadFile("migration/3.22/0009##split_saved_query_manage.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)

	permissionsOf := func(roleID string) []string {
		var raw []byte
		require.NoError(t, db.QueryRowContext(ctx, `SELECT permissions->'permissions' FROM role WHERE resource_id = $1`, roleID).Scan(&raw))
		if string(raw) == "null" {
			return nil
		}
		var list []string
		require.NoError(t, json.Unmarshal(raw, &list))
		return list
	}

	// manage expands to the four verbs, deduplicated against the get the
	// role already held; setIamPolicy is deliberately absent.
	require.ElementsMatch(t, []string{
		"bb.projects.get",
		"bb.savedQueries.delete",
		"bb.savedQueries.get",
		"bb.savedQueries.getIamPolicy",
		"bb.savedQueries.update",
	}, permissionsOf("manage-holder"))
	require.NotContains(t, permissionsOf("manage-holder"), "bb.savedQueries.setIamPolicy")

	// Rows without manage keep their permissions verbatim.
	require.Equal(t, []string{"bb.savedQueries.search", "bb.projects.get"}, permissionsOf("no-manage"))

	// Non-array and empty payloads are untouched rather than mangled.
	var nonArray string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT permissions::text FROM role WHERE resource_id = 'non-array'`).Scan(&nonArray))
	require.JSONEq(t, `{"permissions": {"oops": true}}`, nonArray)
	var empty string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT permissions::text FROM role WHERE resource_id = 'empty'`).Scan(&empty))
	require.JSONEq(t, `{}`, empty)
}
