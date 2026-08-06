package store_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

const instanceArchiveTaskRunSeed = `
	UPDATE project SET deleted = FALSE WHERE resource_id = 'project-a';
	INSERT INTO instance (resource_id, workspace, project)
		VALUES ('instance-a', 'default', 'project-a');
	INSERT INTO plan (id, creator, project, name, description)
		VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
	INSERT INTO task (id, project, plan_id, instance, type)
		VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_CREATE');
`

func archiveInstanceA(fixture *projectDeletionLockOrderFixture) error {
	deleted := true
	_, err := fixture.store.UpdateInstance(fixture.ctx, &store.UpdateInstanceMessage{
		ResourceID: new("instance-a"),
		Workspace:  "default",
		Deleted:    &deleted,
	})
	return err
}

func createInstanceATaskRun(fixture *projectDeletionLockOrderFixture) error {
	return fixture.store.CreatePendingTaskRuns(fixture.ctx, "creator@example.com", &store.TaskRunMessage{
		ProjectID: "project-a",
		TaskUID:   101,
	})
}

func forceArchiveWorkspaceInstanceA(fixture *projectDeletionLockOrderFixture) error {
	deleted := true
	destinationProjectID := "default"
	_, err := fixture.store.UpdateInstance(fixture.ctx, &store.UpdateInstanceMessage{
		ResourceID:               new("instance-a"),
		Workspace:                "default",
		Deleted:                  &deleted,
		MoveDatabasesToProjectID: &destinationProjectID,
	})
	return err
}

func createInstanceADatabase(fixture *projectDeletionLockOrderFixture) error {
	_, err := fixture.store.CreateDatabaseDefault(fixture.ctx, &store.DatabaseMessage{
		InstanceID:   "instance-a",
		DatabaseName: "db-a",
		ProjectID:    "project-a",
	})
	return err
}

func TestArchiveInstanceAndCreateTaskRunSerialize(t *testing.T) {
	t.Run("archive first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, instanceArchiveTaskRunSeed)
		const barrierID = 9945
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER UPDATE OF deleted ON instance FOR EACH ROW")

		archiveResult := make(chan error, 1)
		go func() { archiveResult <- archiveInstanceA(fixture) }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		createResult := make(chan error, 1)
		go func() { createResult <- createInstanceATaskRun(fixture) }()
		barrierReleased := false
		defer func() {
			if !barrierReleased {
				barrier.release(t)
				<-archiveResult
				<-createResult
			}
		}()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)
		barrierReleased = true

		requireMaintenanceResult(t, archiveResult)
		createErr := <-createResult
		require.Error(t, createErr)
		require.Equal(t, common.Conflict, common.ErrorCode(createErr))

		var deleted bool
		var activeTaskRunCount int
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT deleted FROM instance WHERE resource_id = 'instance-a'",
		).Scan(&deleted))
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx, `
			SELECT COUNT(*) FROM task_run
			WHERE project = 'project-a' AND status IN ('PENDING', 'AVAILABLE', 'RUNNING')
		`).Scan(&activeTaskRunCount))
		require.True(t, deleted)
		require.Zero(t, activeTaskRunCount)
	})

	t.Run("creation first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, instanceArchiveTaskRunSeed)
		const barrierID = 9946
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER INSERT ON task_run FOR EACH ROW")

		createResult := make(chan error, 1)
		go func() { createResult <- createInstanceATaskRun(fixture) }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		archiveResult := make(chan error, 1)
		go func() { archiveResult <- archiveInstanceA(fixture) }()
		barrierReleased := false
		defer func() {
			if !barrierReleased {
				barrier.release(t)
				<-createResult
				<-archiveResult
			}
		}()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)
		barrierReleased = true

		requireMaintenanceResult(t, createResult)
		archiveErr := <-archiveResult
		require.Error(t, archiveErr)
		require.Equal(t, common.Conflict, common.ErrorCode(archiveErr))

		var deleted bool
		var activeTaskRunCount int
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT deleted FROM instance WHERE resource_id = 'instance-a'",
		).Scan(&deleted))
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx, `
			SELECT COUNT(*) FROM task_run
			WHERE project = 'project-a' AND status IN ('PENDING', 'AVAILABLE', 'RUNNING')
		`).Scan(&activeTaskRunCount))
		require.False(t, deleted)
		require.Equal(t, 1, activeTaskRunCount)
	})
}

func TestForceArchiveInstanceAndCreateFirstDatabaseSerialize(t *testing.T) {
	const seedSQL = `
		UPDATE project SET deleted = FALSE WHERE resource_id = 'project-a';
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('instance-a', 'default', NULL);
	`

	t.Run("archive first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9947
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER UPDATE OF deleted ON instance FOR EACH ROW")

		archiveResult := make(chan error, 1)
		go func() { archiveResult <- forceArchiveWorkspaceInstanceA(fixture) }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		createResult := make(chan error, 1)
		go func() { createResult <- createInstanceADatabase(fixture) }()
		barrierReleased := false
		defer func() {
			if !barrierReleased {
				barrier.release(t)
				<-archiveResult
				<-createResult
			}
		}()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)
		barrierReleased = true

		requireMaintenanceResult(t, archiveResult)
		createErr := <-createResult
		require.Error(t, createErr)
		require.NotContains(t, strings.ToLower(createErr.Error()), "foreign key")

		var deleted bool
		var databaseCount int
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT deleted FROM instance WHERE resource_id = 'instance-a'",
		).Scan(&deleted))
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT COUNT(*) FROM db WHERE instance = 'instance-a' AND name = 'db-a'",
		).Scan(&databaseCount))
		require.True(t, deleted)
		require.Zero(t, databaseCount)
	})

	t.Run("creation first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9948
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER INSERT ON db FOR EACH ROW")

		createResult := make(chan error, 1)
		go func() { createResult <- createInstanceADatabase(fixture) }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		archiveResult := make(chan error, 1)
		go func() { archiveResult <- forceArchiveWorkspaceInstanceA(fixture) }()
		barrierReleased := false
		defer func() {
			if !barrierReleased {
				barrier.release(t)
				<-createResult
				<-archiveResult
			}
		}()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)
		barrierReleased = true

		requireMaintenanceResult(t, createResult)
		archiveErr := <-archiveResult
		require.Error(t, archiveErr)
		require.Equal(t, common.Conflict, common.ErrorCode(archiveErr))
		require.NotContains(t, strings.ToLower(archiveErr.Error()), "foreign key")

		var deleted bool
		var projectID string
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT deleted FROM instance WHERE resource_id = 'instance-a'",
		).Scan(&deleted))
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT project FROM db WHERE instance = 'instance-a' AND name = 'db-a'",
		).Scan(&projectID))
		require.False(t, deleted)
		require.Equal(t, "project-a", projectID)
	})
}
