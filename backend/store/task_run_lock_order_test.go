package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestClaimAvailableTaskRunsSkipsDeletedProjects(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description)
			VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
		INSERT INTO task (id, project, plan_id, instance, type)
			VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
		INSERT INTO task_run (id, project, task_id, attempt, status)
			VALUES (101, 'project-a', 101, 0, 'AVAILABLE');
	`)

	claimed, err := fixture.store.ClaimAvailableTaskRuns(fixture.ctx, "replica-a")
	require.Empty(t, claimed)
	require.NoError(t, err)

	var status string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT status FROM task_run WHERE project = 'project-a' AND id = 101",
	).Scan(&status))
	require.Equal(t, "AVAILABLE", status)

	var replicaID sql.NullString
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT replica_id FROM task_run WHERE project = 'project-a' AND id = 101",
	).Scan(&replicaID))
	require.False(t, replicaID.Valid)
}

func TestClaimAvailableTaskRunsSkipsArchivedInstances(t *testing.T) {
	fixture := newTaskRunArchivalLockOrderFixture(t, `
		INSERT INTO instance (resource_id, workspace, deleted) VALUES ('instance-a', 'default', TRUE);
		INSERT INTO plan (id, creator, project, name, description)
			VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
		INSERT INTO task (id, project, plan_id, instance, type)
			VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_MIGRATE');
		INSERT INTO task_run (id, project, task_id, attempt, status)
			VALUES (101, 'project-a', 101, 0, 'AVAILABLE');
	`)

	claimed, err := fixture.store.ClaimAvailableTaskRuns(fixture.ctx, "replica-a")
	require.Empty(t, claimed)
	require.NoError(t, err)

	var status string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT status FROM task_run WHERE project = 'project-a' AND id = 101",
	).Scan(&status))
	require.Equal(t, "AVAILABLE", status)

	var replicaID sql.NullString
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT replica_id FROM task_run WHERE project = 'project-a' AND id = 101",
	).Scan(&replicaID))
	require.False(t, replicaID.Valid)
}

func TestClaimAvailableTaskRunsClaimsLiveInstanceSpecialTaskTypes(t *testing.T) {
	fixture := newTaskRunArchivalLockOrderFixture(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description)
			VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
		INSERT INTO task (id, project, plan_id, instance, type)
			VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_MIGRATE');
		INSERT INTO task (id, project, plan_id, instance, type)
			VALUES (102, 'project-a', 101, 'instance-a', 'DATABASE_CREATE');
		INSERT INTO task_run (id, project, task_id, attempt, status)
			VALUES (101, 'project-a', 101, 0, 'AVAILABLE');
		INSERT INTO task_run (id, project, task_id, attempt, status)
			VALUES (102, 'project-a', 102, 0, 'AVAILABLE');
	`)

	claimed, err := fixture.store.ClaimAvailableTaskRuns(fixture.ctx, "replica-a")
	require.NoError(t, err)
	require.Len(t, claimed, 2, "live-instance task runs of any task type should be claimable")

	var status string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT status FROM task_run WHERE project = 'project-a' AND id = 101",
	).Scan(&status))
	require.Equal(t, "RUNNING", status)
}

func newTaskRunArchivalLockOrderFixture(t *testing.T, seedSQL string) *projectDeletionLockOrderFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('default', 'default', 'Default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`+seedSQL)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	return &projectDeletionLockOrderFixture{ctx: ctx, db: db, store: s}
}

func TestCreatePendingTaskRunsDoesNotDeadlockWithProjectDeletion(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description) VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
		INSERT INTO task (id, project, plan_id, instance, type) VALUES (102, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
		INSERT INTO task (id, project, plan_id, instance, type) VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
	`)
	ctx, db, s := fixture.ctx, fixture.db, fixture.store

	const advisoryLockID = 9901
	lockConn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer lockConn.Close()
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID)
	require.NoError(t, err)
	lockReleased := false
	defer func() {
		if !lockReleased {
			_, _ = lockConn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID)
		}
	}()

	_, err = db.ExecContext(ctx, `
		CREATE FUNCTION block_task_delete_after_higher_lock() RETURNS trigger AS $$
		BEGIN
			PERFORM id FROM task
			WHERE project = 'project-a' AND id = 102
			FOR UPDATE;
			PERFORM pg_advisory_xact_lock(9901);
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER block_task_delete_after_higher_lock
		BEFORE DELETE ON task
		FOR EACH STATEMENT EXECUTE FUNCTION block_task_delete_after_higher_lock();
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
	}, 10*time.Second, 10*time.Millisecond, "project deletion should lock task 102 before deleting the task batch")

	operationResult := make(chan error, 1)
	go func() {
		operationResult <- s.CreatePendingTaskRuns(ctx, "",
			&store.TaskRunMessage{ProjectID: "project-a", TaskUID: 101},
			&store.TaskRunMessage{ProjectID: "project-a", TaskUID: 102},
		)
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
	}, 10*time.Second, 10*time.Millisecond, "task run creation should wait for the project purge")

	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
	require.NoError(t, err)
	lockReleased = true
	require.NoError(t, <-deleteResult)
	require.NoError(t, <-operationResult)
}

func TestCreatePendingTaskRunsRejectsDeletedProjectDuringProjectDeletion(t *testing.T) {
	operationErr, deleteErr := runWithCreationBeforeProjectDeletion(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description) VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
		INSERT INTO task (id, project, plan_id, instance, type) VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
	`, "task", func(ctx context.Context, s *store.Store) error {
		return s.CreatePendingTaskRuns(ctx, "", &store.TaskRunMessage{
			ProjectID: "project-a",
			TaskUID:   101,
		})
	})
	require.NoError(t, deleteErr)
	require.Equal(t, common.NotFound, common.ErrorCode(operationErr))
}
