package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCreatePlanCheckRunDoesNotDeadlockWithProjectDeletion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('default', 'default', 'Default');
		INSERT INTO project (resource_id, workspace, name, deleted) VALUES ('project-a', 'default', 'Project A', TRUE);
		INSERT INTO plan (id, creator, project, name, description) VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
		INSERT INTO plan_check_run (id, project, plan_id, status) VALUES (101, 'project-a', 101, 'FAILED');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	lockConn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer lockConn.Close()
	const advisoryLockID = 9902
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID)
	require.NoError(t, err)
	lockReleased := false
	defer func() {
		if !lockReleased {
			_, _ = lockConn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID)
		}
	}()
	_, err = db.ExecContext(ctx, `
		CREATE FUNCTION block_plan_check_run_delete() RETURNS trigger AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(9902);
			RETURN OLD;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER block_plan_check_run_delete
		AFTER DELETE ON plan_check_run
		FOR EACH ROW EXECUTE FUNCTION block_plan_check_run_delete();
	`)
	require.NoError(t, err)

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- s.DeleteProject(ctx, "default", "project-a")
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_locks
				WHERE locktype = 'advisory'
					AND objid = $1
					AND NOT granted
			)
		`, advisoryLockID).Scan(&waiting)
		return err == nil && waiting
	}, 10*time.Second, 10*time.Millisecond, "project deletion should reach the blocked Plan Check Run delete")

	createResult := make(chan error, 1)
	go func() {
		_, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
			ProjectID: "project-a",
			PlanUID:   101,
			Result:    &storepb.PlanCheckRunResult{},
		})
		createResult <- err
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			WITH delete_backend AS (
				SELECT pid
				FROM pg_locks
				WHERE locktype = 'advisory'
					AND objid = $1
					AND NOT granted
				LIMIT 1
			)
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity AS activity, delete_backend
				WHERE delete_backend.pid = ANY(pg_blocking_pids(activity.pid))
			)
		`, advisoryLockID).Scan(&waiting)
		return err == nil && waiting
	}, 10*time.Second, 10*time.Millisecond, "Plan Check Run creation should wait behind its deletion")

	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
	require.NoError(t, err)
	lockReleased = true
	deleteErr := <-deleteResult
	createErr := <-createResult
	require.NoError(t, deleteErr)
	require.NoError(t, createErr)
}
