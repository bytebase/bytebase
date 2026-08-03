package store_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestCreateSyncHistoryAndDeleteProjectSerialize(t *testing.T) {
	const seedSQL = `
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('instance-a', 'default', 'project-a');
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db-a', 'project-a');
		INSERT INTO sync_history (instance, db_name) VALUES ('instance-a', 'db-a');
	`
	createHistory := func(fixture *projectDeletionLockOrderFixture) error {
		_, err := fixture.store.CreateSyncHistory(fixture.ctx, "instance-a", "db-a", &storepb.DatabaseSchemaMetadata{}, "")
		return err
	}

	t.Run("purge first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9931
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER DELETE ON sync_history FOR EACH ROW")

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		writerResult := make(chan error, 1)
		go func() { writerResult <- createHistory(fixture) }()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, purgeResult)
		err := <-writerResult
		require.Error(t, err)
		require.NotContains(t, strings.ToLower(err.Error()), "foreign key")
	})

	t.Run("writer first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9932
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		_, err := fixture.db.ExecContext(fixture.ctx, `
			CREATE FUNCTION block_sync_history_insert() RETURNS trigger AS $$
			BEGIN
				PERFORM pg_advisory_xact_lock(9932);
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
			CREATE TRIGGER block_sync_history_insert
			BEFORE INSERT ON sync_history FOR EACH ROW
			EXECUTE FUNCTION block_sync_history_insert();
		`)
		require.NoError(t, err)

		writerResult := make(chan error, 1)
		go func() { writerResult <- createHistory(fixture) }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, writerResult)
		requireMaintenanceResult(t, purgeResult)
	})
}

func TestCreateSyncHistoryAndDeleteInstanceSerialize(t *testing.T) {
	const seedSQL = `
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('instance-a', 'default', 'project-a');
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db-a', 'project-a');
		INSERT INTO sync_history (instance, db_name) VALUES ('instance-a', 'db-a');
	`
	createHistory := func(fixture *projectDeletionLockOrderFixture) error {
		_, err := fixture.store.CreateSyncHistory(fixture.ctx, "instance-a", "db-a", &storepb.DatabaseSchemaMetadata{}, "")
		return err
	}
	archiveAndPurge := func(fixture *projectDeletionLockOrderFixture) error {
		if _, err := fixture.db.ExecContext(fixture.ctx, `
			UPDATE instance SET deleted = TRUE WHERE resource_id = 'instance-a'
		`); err != nil {
			return err
		}
		return fixture.store.DeleteInstance(fixture.ctx, "default", "instance-a")
	}

	t.Run("purge first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9933
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER DELETE ON sync_history FOR EACH ROW")

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- archiveAndPurge(fixture) }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		writerResult := make(chan error, 1)
		go func() { writerResult <- createHistory(fixture) }()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, purgeResult)
		err := <-writerResult
		require.Error(t, err)
		require.NotContains(t, strings.ToLower(err.Error()), "foreign key")
	})

	t.Run("writer first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9934
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		_, err := fixture.db.ExecContext(fixture.ctx, `
			CREATE FUNCTION block_sync_history_insert() RETURNS trigger AS $$
			BEGIN
				PERFORM pg_advisory_xact_lock(9934);
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
			CREATE TRIGGER block_sync_history_insert
			BEFORE INSERT ON sync_history FOR EACH ROW
			EXECUTE FUNCTION block_sync_history_insert();
		`)
		require.NoError(t, err)

		writerResult := make(chan error, 1)
		go func() { writerResult <- createHistory(fixture) }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- archiveAndPurge(fixture) }()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, writerResult)
		requireMaintenanceResult(t, purgeResult)
	})
}
