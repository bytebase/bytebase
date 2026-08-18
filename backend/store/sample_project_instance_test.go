package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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

func TestWithLockedSampleProjectInstanceAppliesImmutableLifecycleState(t *testing.T) {
	ctx, _, s := newSampleProjectInstanceFixture(t)
	_, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	expiresAt := time.Date(2026, time.August, 24, 9, 0, 0, 0, time.UTC)
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(ctx context.Context, tx *store.SampleProjectInstanceTx, _ *store.SampleProjectInstanceMessage) error {
		return tx.SetExpiration(ctx, expiresAt)
	}))

	var activated *store.SampleProjectInstanceMessage
	err = s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(ctx context.Context, tx *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
		activated = message
		return tx.SetExpiration(ctx, expiresAt.Add(time.Hour))
	})
	require.Error(t, err)
	require.Equal(t, &expiresAt, activated.ExpiresAt)
}

func TestWithLockedSampleProjectInstanceResetsStaleReservationAndCountsCleanup(t *testing.T) {
	ctx, _, s := newSampleProjectInstanceFixture(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	_, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance("workspace-a"))
	require.NoError(t, err)
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(ctx context.Context, tx *store.SampleProjectInstanceTx, _ *store.SampleProjectInstanceMessage) error {
		return tx.ResetCreatedAt(ctx, now)
	}))
	count, err := s.CountSampleProjectInstancesForCleanup(ctx, now.Add(time.Hour), now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(_ context.Context, _ *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
		require.Equal(t, now, message.CreatedAt)
		return nil
	}))
}

func TestWithLockedSampleProjectInstanceCleanupRecordIteratesAfterCallbackError(t *testing.T) {
	ctx, db, s := newSampleProjectInstanceFixture(t)
	now := time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)
	for _, workspace := range []string{"workspace-a", "workspace-b", "workspace-d"} {
		_, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance(workspace))
		require.NoError(t, err)
	}
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(ctx context.Context, tx *store.SampleProjectInstanceTx, _ *store.SampleProjectInstanceMessage) error {
		return tx.SetExpiration(ctx, now.Add(-time.Second))
	}))
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-b", func(ctx context.Context, tx *store.SampleProjectInstanceTx, _ *store.SampleProjectInstanceMessage) error {
		return tx.SetExpiration(ctx, now.Add(time.Second))
	}))
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-d", func(ctx context.Context, tx *store.SampleProjectInstanceTx, _ *store.SampleProjectInstanceMessage) error {
		return tx.SetExpiration(ctx, now.Add(-time.Second))
	}))
	_, err := db.ExecContext(ctx, `
		INSERT INTO sample_project_instance (workspace, project, instance, db_name, role_name, created_at)
		VALUES ('workspace-c', 'project-workspace-c', 'instance-workspace-c', 'database-workspace-c', 'role-workspace-c', $1)
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

	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-a", func(_ context.Context, _ *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
		require.Nil(t, message.DeletedAt)
		return nil
	}))
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-b", func(_ context.Context, _ *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
		require.Nil(t, message.DeletedAt)
		return nil
	}))
	require.Error(t, s.WithLockedSampleProjectInstance(ctx, "workspace-c", func(context.Context, *store.SampleProjectInstanceTx, *store.SampleProjectInstanceMessage) error {
		return nil
	}))
	require.NoError(t, s.WithLockedSampleProjectInstance(ctx, "workspace-d", func(_ context.Context, _ *store.SampleProjectInstanceTx, message *store.SampleProjectInstanceMessage) error {
		require.Equal(t, &now, message.DeletedAt)
		return nil
	}))

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
		_, _, err := s.ReserveSampleProjectInstance(ctx, sampleProjectInstance(workspace))
		require.NoError(t, err)
		require.NoError(t, s.WithLockedSampleProjectInstance(ctx, workspace, func(ctx context.Context, tx *store.SampleProjectInstanceTx, _ *store.SampleProjectInstanceMessage) error {
			return tx.SetExpiration(ctx, now.Add(-time.Second))
		}))
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
	}
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
