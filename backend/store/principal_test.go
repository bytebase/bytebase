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

// Task 101 holds alex's email only as a revision deleter and task 102 only as
// the skipper, so a rename that rewrote one field per statement would lock 102
// before 101, against a skip that locks them in id order.
const renameSkipContentionSeed = `
	INSERT INTO principal (name, email, password_hash) VALUES ('alex', 'alex@example.com', 'x');
	INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
	INSERT INTO plan (id, creator, project, name, description, config) VALUES
		(101, 'alex@example.com', 'default', 'p', '', '{}');
	INSERT INTO task (id, project, plan_id, instance, type, payload) VALUES
		(101, 'default', 101, 'instance-a', 'DATABASE_MIGRATE',
		 '{"releaseState": {"deletedRevisions": [{"version": "003", "deleter": "alex@example.com"}]}}'),
		(102, 'default', 101, 'instance-a', 'DATABASE_MIGRATE',
		 '{"skipped": true, "skipper": "alex@example.com"}');
`

func TestUpdateUserEmailAndBatchSkipTasksBothCommit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		barrierID int
		// The writer parked mid-transaction, holding its task locks.
		parkRename bool
	}{
		{name: "skip lands while the rename holds task locks", barrierID: 9981, parkRename: true},
		{name: "rename lands while the skip holds task locks", barrierID: 9982},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newStorePostgresFixture(t, renameSkipContentionSeed)
			barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, tc.barrierID)
			installMaintenanceLockBarrier(t, fixture.db, tc.barrierID, "AFTER UPDATE ON task FOR EACH STATEMENT")

			rename := func() error {
				user, err := fixture.store.GetUserByEmail(fixture.ctx, "alex@example.com")
				if err != nil {
					return err
				}
				_, err = fixture.store.UpdateUserEmail(fixture.ctx, user, "alex2@example.com")
				return err
			}
			skip := func() error {
				return fixture.store.BatchSkipTasks(fixture.ctx, "default", []int64{101, 102}, "fixed in git", "sam@example.com")
			}
			first, second := skip, rename
			if tc.parkRename {
				first, second = rename, skip
			}

			firstDone := make(chan error, 1)
			go func() { firstDone <- first() }()
			waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, tc.barrierID)
			parkedPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, tc.barrierID)

			secondDone := make(chan error, 1)
			go func() { secondDone <- second() }()
			waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, parkedPID)
			barrier.release(t)

			requireMaintenanceResult(t, firstDone)
			requireMaintenanceResult(t, secondDone)
			var deleter string
			require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
				`SELECT payload #>> '{releaseState,deletedRevisions,0,deleter}' FROM task WHERE project = 'default' AND id = 101`).Scan(&deleter))
			require.Equal(t, "alex2@example.com", deleter)
		})
	}
}
