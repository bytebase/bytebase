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

func newSampleInstanceFixture(t *testing.T) (context.Context, *sql.DB, *store.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES
			('workspace-a'), ('workspace-b'), ('workspace-c'), ('workspace-d');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-workspace-a', 'workspace-a', 'Project A'),
			('project-workspace-b', 'workspace-b', 'Project B'),
			('project-workspace-c', 'workspace-c', 'Project C'),
			('project-workspace-d', 'workspace-d', 'Project D');
	`)
	require.NoError(t, err)
	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })
	return ctx, db, stores
}

func TestSampleInstanceSetupPersistsOpaquePayload(t *testing.T) {
	ctx, _, stores := newSampleInstanceFixture(t)
	create := &store.SampleInstanceSetupMessage{
		WorkspaceID: "workspace-a",
		ReplicaID:   "replica-a",
		Payload:     []byte(`{"instances":[{"instanceId":"sample-a"}]}`),
	}

	got, created, err := stores.ReserveSampleInstanceSetup(ctx, create)
	require.NoError(t, err)
	require.True(t, created)
	require.JSONEq(t, string(create.Payload), string(got.Payload))

	again, created, err := stores.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: "workspace-a",
		ReplicaID:   "replica-b",
		Payload:     []byte(`{"instanceId":"different"}`),
	})
	require.NoError(t, err)
	require.False(t, created)
	require.JSONEq(t, string(create.Payload), string(again.Payload))

	count, err := stores.CountSampleInstanceSetupsForCleanup(
		ctx,
		time.Now().Add(time.Hour),
		time.Now().Add(time.Hour),
	)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestSampleInstanceSetupPermanentLifecycle(t *testing.T) {
	ctx, _, stores := newSampleInstanceFixture(t)
	setup := &store.SampleInstanceSetupMessage{
		WorkspaceID: "workspace-a",
		ReplicaID:   "replica-a",
		Payload:     []byte(`{"databaseProjectId":"project-workspace-a","instances":[]}`),
	}
	_, created, err := stores.ReserveSampleInstanceSetup(ctx, setup)
	require.NoError(t, err)
	require.True(t, created)

	activatedAt := time.Now().UTC().Truncate(time.Microsecond)
	activated, err := stores.ActivateSampleInstanceSetup(
		ctx,
		"workspace-a",
		"replica-a",
		[]string{"project-workspace-a"},
		activatedAt,
		nil,
	)
	require.NoError(t, err)
	require.True(t, activated)
	active, err := stores.GetSampleInstanceSetup(ctx, "workspace-a")
	require.NoError(t, err)
	require.Equal(t, activatedAt, *active.ActivatedAt)
	require.Nil(t, active.ExpiresAt)

	count, err := stores.CountSampleInstanceSetupsForCleanup(
		ctx,
		activatedAt.Add(365*24*time.Hour),
		activatedAt.Add(365*24*time.Hour),
	)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSampleInstanceSetupCleanupIsWorkspaceScoped(t *testing.T) {
	ctx, _, stores := newSampleInstanceFixture(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	for _, workspace := range []string{"workspace-a", "workspace-b"} {
		_, created, err := stores.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
			WorkspaceID: workspace,
			ReplicaID:   "replica-a",
			Payload:     []byte(fmt.Sprintf(`{"instanceId":"sample-%s"}`, workspace)),
		})
		require.NoError(t, err)
		require.True(t, created)
	}
	_, created, err := stores.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: "workspace-c",
		ReplicaID:   "replica-a",
		Payload:     []byte(`{"instances":[]}`),
	})
	require.NoError(t, err)
	require.True(t, created)

	beforeB, err := stores.GetSampleInstanceSetup(ctx, "workspace-b")
	require.NoError(t, err)
	result, err := stores.WithLockedSampleInstanceSetupForCleanup(
		ctx,
		now,
		now.Add(time.Hour),
		"",
		func(ctx context.Context, tx *store.SampleInstanceSetupTx, setup *store.SampleInstanceSetupMessage) error {
			require.Equal(t, "workspace-a", setup.WorkspaceID)
			return tx.DeleteReservation(ctx)
		},
	)
	require.NoError(t, err)
	require.True(t, result.Found)
	require.Equal(t, "workspace-a", result.WorkspaceID)

	afterB, err := stores.GetSampleInstanceSetup(ctx, "workspace-b")
	require.NoError(t, err)
	require.Equal(t, beforeB, afterB)
	selfHost, err := stores.GetSampleInstanceSetup(ctx, "workspace-c")
	require.NoError(t, err)
	require.NotNil(t, selfHost)
}

func TestSampleInstanceSetupDeletedRowRemainsTombstone(t *testing.T) {
	ctx, _, stores := newSampleInstanceFixture(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	expiresAt := now.Add(time.Hour)
	setup := &store.SampleInstanceSetupMessage{
		WorkspaceID: "workspace-a",
		ReplicaID:   "replica-a",
		Payload:     []byte(`{"projectId":"project-workspace-a","instanceId":"sample-a"}`),
	}
	_, created, err := stores.ReserveSampleInstanceSetup(ctx, setup)
	require.NoError(t, err)
	require.True(t, created)
	activated, err := stores.ActivateSampleInstanceSetup(
		ctx,
		"workspace-a",
		"replica-a",
		[]string{"project-workspace-a"},
		now,
		&expiresAt,
	)
	require.NoError(t, err)
	require.True(t, activated)

	cleanupAt := expiresAt.Add(time.Second)
	result, err := stores.WithLockedSampleInstanceSetupForCleanup(
		ctx,
		cleanupAt,
		cleanupAt.Add(-time.Hour),
		"",
		func(ctx context.Context, tx *store.SampleInstanceSetupTx, _ *store.SampleInstanceSetupMessage) error {
			return tx.MarkDeleted(ctx, cleanupAt)
		},
	)
	require.NoError(t, err)
	require.True(t, result.Found)

	deleted, err := stores.GetSampleInstanceSetup(ctx, "workspace-a")
	require.NoError(t, err)
	require.Equal(t, cleanupAt, *deleted.DeletedAt)
	again, created, err := stores.ReserveSampleInstanceSetup(ctx, setup)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, deleted, again)
}

func TestActivateSampleInstanceSetupSerializesWithProjectArchive(t *testing.T) {
	ctx, db, stores := newSampleInstanceFixture(t)
	_, created, err := stores.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: "workspace-a",
		ReplicaID:   "replica-a",
		Payload:     []byte(`{"projectId":"project-workspace-a","instanceId":"sample-a"}`),
	})
	require.NoError(t, err)
	require.True(t, created)

	archiveTx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	committed := false
	defer func() {
		if !committed {
			_ = archiveTx.Rollback()
		}
	}()
	_, err = archiveTx.ExecContext(ctx, `
		UPDATE project SET deleted = TRUE
		WHERE workspace = 'workspace-a' AND resource_id = 'project-workspace-a'
	`)
	require.NoError(t, err)

	result := make(chan error, 1)
	go func() {
		_, err := stores.ActivateSampleInstanceSetup(
			ctx,
			"workspace-a",
			"replica-a",
			[]string{"project-workspace-a"},
			time.Now(),
			nil,
		)
		result <- err
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE wait_event_type = 'Lock'
					AND query LIKE '%SELECT deleted FROM project%FOR UPDATE%'
			)
		`).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, archiveTx.Commit())
	committed = true
	require.Equal(t, common.NotFound, common.ErrorCode(<-result))

	current, err := stores.GetSampleInstanceSetup(ctx, "workspace-a")
	require.NoError(t, err)
	require.Nil(t, current.ActivatedAt)
}

func TestProjectArchiveWaitsForSampleInstanceSetupActivation(t *testing.T) {
	ctx, db, stores := newSampleInstanceFixture(t)
	_, created, err := stores.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: "workspace-a",
		ReplicaID:   "replica-a",
		Payload:     []byte(`{"projectId":"project-workspace-a","instanceId":"sample-a"}`),
	})
	require.NoError(t, err)
	require.True(t, created)

	workspaceTx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	committed := false
	defer func() {
		if !committed {
			_ = workspaceTx.Rollback()
		}
	}()
	var lockedWorkspace string
	err = workspaceTx.QueryRowContext(ctx, `
		SELECT resource_id FROM workspace
		WHERE resource_id = 'workspace-a'
		FOR UPDATE
	`).Scan(&lockedWorkspace)
	require.NoError(t, err)

	activationResult := make(chan struct {
		activated bool
		err       error
	}, 1)
	go func() {
		activated, err := stores.ActivateSampleInstanceSetup(
			ctx,
			"workspace-a",
			"replica-a",
			[]string{"project-workspace-a"},
			time.Now(),
			nil,
		)
		activationResult <- struct {
			activated bool
			err       error
		}{activated: activated, err: err}
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE wait_event_type = 'Lock'
					AND query LIKE '%SELECT deleted FROM workspace%FOR UPDATE%'
			)
		`).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)

	archiveResult := make(chan error, 1)
	go func() {
		_, err := db.ExecContext(ctx, `
			UPDATE project SET deleted = TRUE
			WHERE workspace = 'workspace-a' AND resource_id = 'project-workspace-a'
		`)
		archiveResult <- err
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE wait_event_type = 'Lock'
					AND query LIKE '%UPDATE project SET deleted = TRUE%'
			)
		`).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)

	require.NoError(t, workspaceTx.Commit())
	committed = true
	activation := <-activationResult
	require.NoError(t, activation.err)
	require.True(t, activation.activated)
	require.NoError(t, <-archiveResult)

	current, err := stores.GetSampleInstanceSetup(ctx, "workspace-a")
	require.NoError(t, err)
	require.NotNil(t, current.ActivatedAt)
	var projectDeleted bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT deleted FROM project
		WHERE workspace = 'workspace-a' AND resource_id = 'project-workspace-a'
	`).Scan(&projectDeleted))
	require.True(t, projectDeleted)
}
