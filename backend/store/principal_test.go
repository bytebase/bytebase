package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// The second user's email contains the first's, so a substring rewrite would
// wrongly touch it.
func TestUpdateUserEmailRewritesTaskSkipperAndRevisionDeleters(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES
			('alex', 'alex@example.com', 'x'),
			('xalex', 'xalex@example.com', 'x');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description, config) VALUES
			(101, 'alex@example.com', 'project-a', 'p', '', '{}');
		INSERT INTO task (id, project, plan_id, instance, type, payload) VALUES
			(201, 'project-a', 101, 'instance-a', 'DATABASE_MIGRATE',
			 '{"skipped": true, "skipper": "alex@example.com", "releaseState": {"deletedRevisions": [{"version": "003", "deleter": "alex@example.com"}, {"version": "004", "deleter": "xalex@example.com"}]}}'),
			(202, 'project-a', 101, 'instance-a', 'DATABASE_MIGRATE',
			 '{"skipped": true, "skipper": "xalex@example.com", "releaseState": {"deletedRevisions": [{"version": "003", "deleter": "xalex@example.com"}]}}'),
			(203, 'project-a', 101, 'instance-a', 'DATABASE_MIGRATE', '{"releaseState": {"appliedCount": 2}}');
	`)
	require.NoError(t, err)

	user, err := s.GetUserByEmail(ctx, "alex@example.com")
	require.NoError(t, err)
	_, err = s.UpdateUserEmail(ctx, user, "alex2@example.com")
	require.NoError(t, err)

	read := func(id int, path string) string {
		t.Helper()
		var value string
		require.NoError(t, db.QueryRowContext(ctx,
			`SELECT COALESCE(payload #>> $1::TEXT[], '') FROM task WHERE project = 'project-a' AND id = $2`,
			path, id).Scan(&value))
		return value
	}
	require.Equal(t, "alex2@example.com", read(201, "{skipper}"))
	require.Equal(t, "alex2@example.com", read(201, "{releaseState,deletedRevisions,0,deleter}"))
	require.Equal(t, "003", read(201, "{releaseState,deletedRevisions,0,version}"))
	require.Equal(t, "xalex@example.com", read(201, "{releaseState,deletedRevisions,1,deleter}"))
	require.Equal(t, "xalex@example.com", read(202, "{skipper}"))
	require.Equal(t, "xalex@example.com", read(202, "{releaseState,deletedRevisions,0,deleter}"))
	require.Equal(t, "2", read(203, "{releaseState,appliedCount}"))
}
