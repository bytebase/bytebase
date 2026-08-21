package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func newSampleProjectInstanceFixture(t *testing.T) (context.Context, *sql.DB, *store.Store) {
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
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return ctx, db, s
}

func TestReserveSampleProjectInstanceSerializesWorkspaceEntitlement(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	lockTx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer lockTx.Rollback()
	var workspace string
	require.NoError(t, lockTx.QueryRowContext(ctx, `
		SELECT resource_id FROM workspace WHERE resource_id = 'workspace-a' FOR UPDATE
	`).Scan(&workspace))

	type result struct {
		created bool
		err     error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			_, created, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
			results <- result{created: created, err: err}
		}()
	}
	require.Eventually(t, func() bool {
		var waiting int
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM pg_stat_activity
			WHERE wait_event_type = 'Lock'
				AND query LIKE '%SELECT deleted%'
		`).Scan(&waiting)
		return err == nil && waiting == 2
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, lockTx.Commit())

	var successes, created, existing int
	for range 2 {
		result := <-results
		if result.err == nil {
			successes++
			if result.created {
				created++
			} else {
				existing++
			}
		}
	}
	require.Equal(t, 2, successes)
	require.Equal(t, 1, created)
	require.Equal(t, 1, existing)
}

func TestReserveSampleProjectInstanceRejectsMissingAndDeletedWorkspace(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	_, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("missing"))
	require.Error(t, err)
	require.NoError(t, db.QueryRowContext(ctx, `
		UPDATE workspace SET deleted = TRUE WHERE resource_id = 'workspace-b' RETURNING resource_id
	`).Scan(new(string)))
	_, _, err = s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-b"))
	require.Error(t, err)
}

func TestReserveSampleProjectInstanceRejectsOtherUniqueCollisions(t *testing.T) {
	ctx, _, s := newSampleProjectInstanceFixture(t)
	created, wasCreated, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	require.True(t, wasCreated)

	conflicting := sampleProjectInstance("workspace-b")
	conflicting.InstanceID = created.InstanceID
	_, wasCreated, err = s.ReserveSampleProjectInstance(ctx, conflicting)
	require.Error(t, err)
	require.False(t, wasCreated)
}

func TestActivateSampleProjectInstanceAppliesImmutableLifecycleState(t *testing.T) {
	ctx, _, s := newSampleProjectInstanceFixture(t)
	reservation, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	expiresAt := time.Date(2026, time.August, 24, 9, 0, 0, 0, time.UTC)
	activateSampleProjectInstance(ctx, t, s, reservation, expiresAt)
	activated, err := s.ActivateSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.ReplicaID, expiresAt.Add(time.Hour))
	require.NoError(t, err)
	require.False(t, activated)
}

func TestActivateSampleProjectInstanceSerializesWithWorkspaceDeletion(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	reservation, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	deleteTx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	deletionCommitted := false
	defer func() {
		if !deletionCommitted {
			_ = deleteTx.Rollback()
		}
	}()
	_, err = deleteTx.ExecContext(ctx, "UPDATE workspace SET deleted = TRUE WHERE resource_id = 'workspace-a'")
	require.NoError(t, err)

	result := make(chan error, 1)
	go func() {
		_, err := s.ActivateSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.ReplicaID, time.Now().Add(time.Hour))
		result <- err
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE wait_event_type = 'Lock'
					AND query LIKE '%SELECT deleted%FROM workspace%FOR UPDATE%'
			)
		`).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, deleteTx.Commit())
	deletionCommitted = true
	require.Equal(t, common.NotFound, common.ErrorCode(<-result))
}

func TestActivateSampleProjectInstanceSerializesWithProjectArchive(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	reservation, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	archiveTx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	archiveCommitted := false
	defer func() {
		if !archiveCommitted {
			_ = archiveTx.Rollback()
		}
	}()
	_, err = archiveTx.ExecContext(ctx, "UPDATE project SET deleted = TRUE WHERE resource_id = 'project-workspace-a' AND workspace = 'workspace-a'")
	require.NoError(t, err)

	result := make(chan error, 1)
	go func() {
		_, err := s.ActivateSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.ReplicaID, time.Now().Add(time.Hour))
		result <- err
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE wait_event_type = 'Lock'
					AND query LIKE '%SELECT deleted%FROM project%FOR UPDATE%'
			)
		`).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, archiveTx.Commit())
	archiveCommitted = true
	require.Equal(t, common.NotFound, common.ErrorCode(<-result))
}

func TestActivateSampleProjectInstanceRejectsMissingAndDeletedProject(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	expiresAt := time.Date(2026, time.August, 24, 9, 0, 0, 0, time.UTC)

	missing := sampleProjectInstance("workspace-a")
	missing.ProjectID = "missing"
	reservation, _, err := s.ReserveSampleProjectInstance(ctx, missing)
	require.NoError(t, err)
	_, err = s.ActivateSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.ReplicaID, expiresAt)
	require.Equal(t, common.NotFound, common.ErrorCode(err))

	reservation, _, err = s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-b"))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE project SET deleted = TRUE WHERE resource_id = 'project-workspace-b' AND workspace = 'workspace-b'")
	require.NoError(t, err)
	_, err = s.ActivateSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.ReplicaID, expiresAt)
	require.Equal(t, common.NotFound, common.ErrorCode(err))
}

func TestCountSampleProjectInstancesForCleanupIncludesStaleReservation(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	reservation, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE sample_project_instance SET created_at = $1 WHERE workspace = $2 AND instance = $3", now, reservation.WorkspaceID, reservation.InstanceID)
	require.NoError(t, err)
	count, err := s.CountSampleProjectInstancesForCleanup(ctx, now.Add(time.Hour), now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestWithLockedSampleProjectInstanceCleanupRecordIteratesAfterCallbackError(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	for _, workspace := range []string{"workspace-a", "workspace-b", "workspace-d"} {
		reservation, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance(workspace))
		require.NoError(t, err)
		if workspace == "workspace-b" {
			activateSampleProjectInstance(ctx, t, s, reservation, now.Add(time.Second))
		} else {
			activateSampleProjectInstance(ctx, t, s, reservation, now.Add(-time.Second))
		}
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO sample_project_instance (workspace, project, instance, db_name, role_name, replica_id, created_at)
		VALUES ('workspace-c', 'project-workspace-c', 'instance-workspace-c', 'database-workspace-c', 'role-workspace-c', 'replica-a', $1)
	`, now.Add(-time.Hour))
	require.NoError(t, err)

	var processed []string
	var callbackErr error
	afterWorkspace := ""
	for {
		result, err := s.WithLockedSampleProjectInstanceCleanupRecord(ctx, now, now.Add(-time.Hour), afterWorkspace, func(_ context.Context, _ *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
			processed = append(processed, message.WorkspaceID)
			if message.WorkspaceID == "workspace-a" {
				return errors.New("physical cleanup failed")
			}
			return nil
		})
		require.NoError(t, err)
		if !result.Found {
			break
		}
		afterWorkspace = result.WorkspaceID
		callbackErr = errors.Join(callbackErr, result.CallbackErr)
	}
	require.Error(t, callbackErr)
	require.Equal(t, []string{"workspace-a", "workspace-c", "workspace-d"}, processed)

	reservation, err := s.GetSampleProjectInstance(ctx, "workspace-a")
	require.NoError(t, err)
	require.Nil(t, reservation.DeletedAt)
	reservation, err = s.GetSampleProjectInstance(ctx, "workspace-b")
	require.NoError(t, err)
	require.Nil(t, reservation.DeletedAt)
	reservation, err = s.GetSampleProjectInstance(ctx, "workspace-c")
	require.NoError(t, err)
	require.Nil(t, reservation)
	reservation, err = s.GetSampleProjectInstance(ctx, "workspace-d")
	require.NoError(t, err)
	require.Equal(t, &now, reservation.DeletedAt)

	result, err := s.WithLockedSampleProjectInstanceCleanupRecord(ctx, now, now.Add(-time.Hour), "", func(_ context.Context, _ *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
		processed = append(processed, message.WorkspaceID)
		return nil
	})
	require.NoError(t, err)
	require.True(t, result.Found)
	require.Equal(t, "workspace-a", result.WorkspaceID)
	require.NoError(t, result.CallbackErr)
}

func TestWithLockedSampleProjectInstanceCleanupRecordLocksOnlySelectedRow(t *testing.T) {
	ctx, _, s := newSampleProjectInstanceFixture(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	for _, workspace := range []string{"workspace-a", "workspace-b"} {
		reservation, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance(workspace))
		require.NoError(t, err)
		activateSampleProjectInstance(ctx, t, s, reservation, now.Add(-time.Second))
	}

	started := make(chan struct{})
	release := make(chan struct{})
	firstResult := make(chan *store.SampleProjectInstanceCleanupResult, 1)
	firstErr := make(chan error, 1)
	go func() {
		result, err := s.WithLockedSampleProjectInstanceCleanupRecord(ctx, now, now.Add(-time.Hour), "", func(context.Context, *store.SampleProjectInstanceTx, *store.SampleProjectInstanceMessage) error {
			select {
			case <-started:
			default:
				close(started)
			}
			<-release
			return nil
		})
		firstResult <- result
		firstErr <- err
	}()
	<-started

	result, err := s.WithLockedSampleProjectInstanceCleanupRecord(ctx, now, now.Add(-time.Hour), "", func(_ context.Context, _ *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
		require.Equal(t, "workspace-b", message.WorkspaceID)
		return nil
	})
	require.NoError(t, err)
	require.True(t, result.Found)
	require.Equal(t, "workspace-b", result.WorkspaceID)

	close(release)
	require.NoError(t, <-firstErr)
	require.Equal(t, "workspace-a", (<-firstResult).WorkspaceID)
}

func sampleProjectInstance(workspace string) *store.SampleProjectInstanceMessage {
	return &store.SampleProjectInstanceMessage{
		WorkspaceID: workspace,
		ProjectID:   "project-" + workspace,
		InstanceID:  "instance-" + workspace,
		DBName:      "database-" + workspace,
		RoleName:    "role-" + workspace,
		ReplicaID:   "replica-a",
	}
}

func activateSampleProjectInstance(ctx context.Context, t *testing.T, s *store.Store, reservation *store.SampleProjectInstanceMessage, expiresAt time.Time) {
	t.Helper()
	activated, err := s.ActivateSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.ReplicaID, expiresAt)
	require.NoError(t, err)
	require.True(t, activated)
}

func TestReserveSampleProjectInstanceConsumesWorkspaceEntitlement(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	created, wasCreated, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	require.True(t, wasCreated)
	require.Equal(t, "workspace-a", created.WorkspaceID)
	require.False(t, created.CreatedAt.IsZero())
	require.Nil(t, created.ExpiresAt)
	require.Nil(t, created.DeletedAt)

	existing, wasCreated, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	require.False(t, wasCreated)
	require.Equal(t, created, existing)

	var count int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sample_project_instance WHERE workspace = 'workspace-a'
	`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestClaimSampleProjectInstanceRequiresExpiredAttempt(t *testing.T) {
	ctx, _, s := newSampleProjectInstanceFixture(t)
	reservation, created, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	require.True(t, created)

	claimed, ok, err := s.ClaimSampleProjectInstance(
		ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.CreatedAt,
		"replica-b", time.Hour, time.Minute,
	)
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, claimed)

	current, err := s.GetSampleProjectInstance(ctx, reservation.WorkspaceID)
	require.NoError(t, err)
	require.Equal(t, "replica-a", current.ReplicaID)
}

func TestClaimSampleProjectInstanceAllowsOneStaleOwnerTakeover(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	reservation, created, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	require.True(t, created)
	_, err = db.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET created_at = now() - INTERVAL '4 minutes'
		WHERE workspace = $1 AND instance = $2
	`, reservation.WorkspaceID, reservation.InstanceID)
	require.NoError(t, err)
	reservation, err = s.GetSampleProjectInstance(ctx, reservation.WorkspaceID)
	require.NoError(t, err)

	type result struct {
		owner   string
		claimed bool
		err     error
	}
	results := make(chan result, 2)
	for _, owner := range []string{"replica-b", "replica-c"} {
		go func() {
			_, claimed, err := s.ClaimSampleProjectInstance(
				ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.CreatedAt,
				owner, 3*time.Minute, time.Minute,
			)
			results <- result{owner: owner, claimed: claimed, err: err}
		}()
	}

	var winner string
	for range 2 {
		result := <-results
		require.NoError(t, result.err)
		if result.claimed {
			require.Empty(t, winner)
			winner = result.owner
		}
	}
	require.NotEmpty(t, winner)
	current, err := s.GetSampleProjectInstance(ctx, reservation.WorkspaceID)
	require.NoError(t, err)
	require.Equal(t, winner, current.ReplicaID)
	require.True(t, current.CreatedAt.After(reservation.CreatedAt))
}

func TestClaimSampleProjectInstanceRequiresStaleOtherOwnerButAllowsSelfRetry(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	reservation, created, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	require.True(t, created)
	_, err = db.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET created_at = now() - INTERVAL '4 minutes'
		WHERE workspace = $1 AND instance = $2
	`, reservation.WorkspaceID, reservation.InstanceID)
	require.NoError(t, err)
	require.NoError(t, s.UpsertReplicaHeartbeat(ctx, reservation.ReplicaID))
	reservation, err = s.GetSampleProjectInstance(ctx, reservation.WorkspaceID)
	require.NoError(t, err)

	_, claimed, err := s.ClaimSampleProjectInstance(
		ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.CreatedAt,
		"replica-b", 3*time.Minute, time.Minute,
	)
	require.NoError(t, err)
	require.False(t, claimed)

	claimedReservation, claimed, err := s.ClaimSampleProjectInstance(
		ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.CreatedAt,
		reservation.ReplicaID, 3*time.Minute, time.Minute,
	)
	require.NoError(t, err)
	require.True(t, claimed)
	require.Equal(t, reservation.ReplicaID, claimedReservation.ReplicaID)
}

func TestOwnedSampleProjectInstanceTransitionsFenceFormerOwner(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	reservation, created, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	require.True(t, created)
	_, err = db.ExecContext(ctx, `
		UPDATE sample_project_instance
		SET created_at = now() - INTERVAL '4 minutes'
		WHERE workspace = $1 AND instance = $2
	`, reservation.WorkspaceID, reservation.InstanceID)
	require.NoError(t, err)
	reservation, err = s.GetSampleProjectInstance(ctx, reservation.WorkspaceID)
	require.NoError(t, err)
	_, claimed, err := s.ClaimSampleProjectInstance(
		ctx, reservation.WorkspaceID, reservation.InstanceID, reservation.CreatedAt,
		"replica-b", 3*time.Minute, time.Minute,
	)
	require.NoError(t, err)
	require.True(t, claimed)

	activated, err := s.ActivateSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, "replica-a", time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.False(t, activated)
	deleted, err := s.DeletePendingSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, "replica-a")
	require.NoError(t, err)
	require.False(t, deleted)

	activated, err = s.ActivateSampleProjectInstance(ctx, reservation.WorkspaceID, reservation.InstanceID, "replica-b", time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.True(t, activated)
}
