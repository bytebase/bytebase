package store_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"github.com/stretchr/testify/require"
)

func TestClaimAvailableTaskRunsSkipsDeletedProjects(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
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
	t.Parallel()
	fixture := newTaskRunClaimFixture(t, `
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
	t.Parallel()
	fixture := newTaskRunClaimFixture(t, `
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

func newTaskRunClaimFixture(t *testing.T, seedSQL string) *storePostgresFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db, s, _ := testcontainer.NewMetadataDB(t)

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('default', 'default', 'Default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`+seedSQL)
	require.NoError(t, err)

	return &storePostgresFixture{ctx: ctx, db: db, store: s}
}
