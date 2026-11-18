package store_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

func TestInstanceQuotaConcurrentWrites(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"create", "activate", "restore"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			ctx, db, s := newInstanceProjectFixture(t)
			const workers = 4
			one := 1
			limits := &store.InstanceLimits{Count: &one}
			if operation == "activate" {
				limits = &store.InstanceLimits{Activated: &one}
			}
			for i := range workers {
				if operation == "create" {
					break
				}
				id := fmt.Sprintf("instance-%d", i)
				_, err := s.CreateInstance(ctx, &store.InstanceMessage{ResourceID: id, Workspace: "default", Metadata: testInstanceMetadata()})
				require.NoError(t, err)
				if operation == "restore" {
					deleted := true
					_, err := s.UpdateInstance(ctx, &store.UpdateInstanceMessage{ResourceID: &id, Workspace: "default", Deleted: &deleted})
					require.NoError(t, err)
				}
			}
			blocker, err := db.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer blocker.Rollback()
			require.NoError(t, store.AcquireAdvisoryXactLockWithStringKey(ctx, blocker, store.AdvisoryLockKeyInstanceQuota, "default"))
			outcomes := make(chan error, workers)
			for i := range workers {
				go func() {
					id := fmt.Sprintf("instance-%d", i)
					metadata := testInstanceMetadata()
					var err error
					switch operation {
					case "create":
						_, err = s.CreateInstance(ctx, &store.InstanceMessage{ResourceID: id, Workspace: "default", Metadata: metadata}, limits)
					case "activate":
						metadata.Activation = true
						_, err = s.UpdateInstance(ctx, &store.UpdateInstanceMessage{ResourceID: &id, Workspace: "default", Metadata: metadata, Limits: limits})
					case "restore":
						deleted := false
						_, err = s.UpdateInstance(ctx, &store.UpdateInstanceMessage{ResourceID: &id, Workspace: "default", Deleted: &deleted, Limits: limits})
					default:
						err = errors.Errorf("unsupported operation %q", operation)
					}
					outcomes <- err
				}()
			}
			require.Eventually(t, func() bool {
				var waiting int
				err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_locks WHERE locktype = 'advisory' AND classid = 1009 AND objid = hashtext('default')::oid AND NOT granted AND database = (SELECT oid FROM pg_database WHERE datname = current_database())`).Scan(&waiting)
				return err == nil && waiting == workers
			}, 10*time.Second, 10*time.Millisecond)
			require.NoError(t, blocker.Commit())
			successes := 0
			for range workers {
				err := <-outcomes
				if err == nil {
					successes++
					continue
				}
				var quota *store.InstanceLimitError
				require.ErrorAs(t, err, &quota)
				require.Equal(t, operation == "activate", quota.Activated)
			}
			require.Equal(t, 1, successes)
			if operation == "activate" {
				count, err := s.GetActivatedInstanceCount(ctx, "default")
				require.NoError(t, err)
				require.Equal(t, 1, count)
			} else {
				count, err := s.CountActiveInstances(ctx, "default")
				require.NoError(t, err)
				require.Equal(t, 1, count)
			}
		})
	}
}

func TestInstanceQuotaWorkspaceIsolation(t *testing.T) {
	t.Parallel()
	ctx, db, s := newInstanceProjectFixture(t)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace(resource_id) VALUES ('other')`)
	require.NoError(t, err)
	one := 1
	limits := &store.InstanceLimits{Count: &one, Activated: &one}
	for _, workspace := range []string{"default", "other"} {
		metadata := testInstanceMetadata()
		metadata.Activation = true
		_, err := s.CreateInstance(ctx, &store.InstanceMessage{ResourceID: workspace + "-instance", Workspace: workspace, Metadata: metadata}, limits)
		require.NoError(t, err)
	}
	_, err = s.CreateInstance(ctx, &store.InstanceMessage{ResourceID: "excess", Workspace: "default", Metadata: testInstanceMetadata()}, limits)
	var quota *store.InstanceLimitError
	require.ErrorAs(t, err, &quota)
	for _, workspace := range []string{"default", "other"} {
		count, err := s.CountActiveInstances(ctx, workspace)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	}
}
