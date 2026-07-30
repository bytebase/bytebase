package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

// newSettingAtomicFixture boots a real PostgreSQL, migrates the schema, seeds
// the workspace and an empty WORKSPACE_PROFILE row, and returns a store with
// the setting cache ENABLED — the production non-HA shape, so the tests also
// pin that the cache reflects committed state.
func newSettingAtomicFixture(t *testing.T) (context.Context, *sql.DB, *store.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default');`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, true /* enableCache */)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_WORKSPACE_PROFILE,
		Workspace: "default",
		Value:     &storepb.WorkspaceProfileSetting{},
	})
	require.NoError(t, err)
	return ctx, db, s
}

func mustProfile(t *testing.T, setting *store.SettingMessage) *storepb.WorkspaceProfileSetting {
	t.Helper()
	profile, ok := setting.Value.(*storepb.WorkspaceProfileSetting)
	require.True(t, ok, "unexpected setting value type %T", setting.Value)
	return profile
}

// TestUpdateSettingAtomicInterleaving pins the lost-update protection: while
// one update holds the row lock inside its apply callback, a concurrent update
// to a DIFFERENT field must wait behind it and then merge onto the committed
// result — both mutations survive in the final row and in the setting cache.
// Against a read-then-UpsertSetting implementation this fails: the second
// update merges onto a stale read and silently reverts the first's field.
func TestUpdateSettingAtomicInterleaving(t *testing.T) {
	ctx, db, s := newSettingAtomicFixture(t)

	entered := make(chan struct{})
	release := make(chan struct{})
	firstResult := make(chan error, 1)
	go func() {
		_, err := s.UpdateSettingAtomic(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE,
			func(current proto.Message) (proto.Message, error) {
				profile, ok := current.(*storepb.WorkspaceProfileSetting)
				if !ok {
					return nil, errors.Errorf("unexpected type %T", current)
				}
				profile.EnableMetricCollection = true
				close(entered)
				<-release
				return profile, nil
			}, nil)
		firstResult <- err
	}()
	// The first update is now inside apply and holds the row lock.
	select {
	case <-entered:
	case <-time.After(15 * time.Second):
		t.Fatal("first update never reached its apply callback")
	}

	secondResult := make(chan error, 1)
	go func() {
		_, err := s.UpdateSettingAtomic(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE,
			func(current proto.Message) (proto.Message, error) {
				profile, ok := current.(*storepb.WorkspaceProfileSetting)
				if !ok {
					return nil, errors.Errorf("unexpected type %T", current)
				}
				profile.McpCapability = storepb.WorkspaceProfileSetting_DISABLED
				return profile, nil
			}, nil)
		secondResult <- err
	}()

	// Deterministic barrier: the second update must be observed waiting behind
	// the first's row lock before the first is released.
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity
				WHERE cardinality(pg_blocking_pids(pid)) > 0
					AND query LIKE '%setting%FOR UPDATE%'
			)
		`).Scan(&waiting)
		return err == nil && waiting
	}, 10*time.Second, 10*time.Millisecond,
		"the second update should wait behind the first's row lock")
	select {
	case err := <-secondResult:
		t.Fatalf("second update completed while the first held the row lock: %v", err)
	default:
	}

	close(release)
	for _, result := range []chan error{firstResult, secondResult} {
		select {
		case err := <-result:
			require.NoError(t, err)
		case <-time.After(15 * time.Second):
			t.Fatal("update did not complete; possible deadlock")
		}
	}

	// Both mutations survive in the database ...
	fresh, err := s.GetSettingUncached(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE)
	require.NoError(t, err)
	freshProfile := mustProfile(t, fresh)
	require.True(t, freshProfile.EnableMetricCollection, "first update's field must survive")
	require.Equal(t, storepb.WorkspaceProfileSetting_DISABLED, freshProfile.McpCapability,
		"second update's field must survive")
	// ... and in the setting cache.
	cached, err := s.GetSetting(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE)
	require.NoError(t, err)
	cachedProfile := mustProfile(t, cached)
	require.True(t, cachedProfile.EnableMetricCollection)
	require.Equal(t, storepb.WorkspaceProfileSetting_DISABLED, cachedProfile.McpCapability)
}

// TestUpdateSettingAtomicApplyAbort pins that an error from apply aborts the
// transaction with no write, leaves the cache untouched, and surfaces
// unwrapped so callers keep typed errors.
func TestUpdateSettingAtomicApplyAbort(t *testing.T) {
	ctx, _, s := newSettingAtomicFixture(t)

	sentinel := errors.New("validation failed")
	_, err := s.UpdateSettingAtomic(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE,
		func(current proto.Message) (proto.Message, error) {
			profile, ok := current.(*storepb.WorkspaceProfileSetting)
			if !ok {
				return nil, errors.Errorf("unexpected type %T", current)
			}
			// Mutate before failing: the mutation must not leak into the row
			// or the served cache.
			profile.EnableMetricCollection = true
			return nil, sentinel
		}, nil)
	require.ErrorIs(t, err, sentinel)

	fresh, err := s.GetSettingUncached(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE)
	require.NoError(t, err)
	require.False(t, mustProfile(t, fresh).EnableMetricCollection)
	cached, err := s.GetSetting(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE)
	require.NoError(t, err)
	require.False(t, mustProfile(t, cached).EnableMetricCollection)
}

// TestUpdateSettingAtomicMissingRow pins that a missing setting row is an
// error, not a silent create — the primitive updates existing state only.
func TestUpdateSettingAtomicMissingRow(t *testing.T) {
	ctx, _, s := newSettingAtomicFixture(t)

	_, err := s.UpdateSettingAtomic(ctx, "default", storepb.SettingName_AI,
		func(current proto.Message) (proto.Message, error) {
			return current, nil
		}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestUpdateSettingAtomicPublishOrder pins that the setting cache is published
// in commit order: while one update is paused between its commit and its cache
// publish (test hook), a later update must not be able to commit and publish —
// otherwise the paused update's older value would overwrite the newer one in
// the cache, serving reverted state until the next refresh. Red against an
// implementation whose commit releases the row lock before an unordered cache
// write.
func TestUpdateSettingAtomicPublishOrder(t *testing.T) {
	ctx, _, s := newSettingAtomicFixture(t)

	pauseFirst := make(chan struct{})
	resumeFirst := make(chan struct{})
	// Only the first update pauses; later publishes pass through WITHOUT
	// blocking (sync.Once.Do would serialize concurrent callers and mask the
	// very ordering bug this test pins).
	var first atomic.Bool
	s.SetSettingPublishHookForTest(func() {
		if first.CompareAndSwap(false, true) {
			close(pauseFirst)
			<-resumeFirst
		}
	})

	// postCommit callbacks run inside the same ordered window, so their
	// observed order must match commit order (derived runtime state relies
	// on this).
	var postCommitMu sync.Mutex
	var postCommitOrder []string
	recordPostCommit := func(id string) func() {
		return func() {
			postCommitMu.Lock()
			defer postCommitMu.Unlock()
			postCommitOrder = append(postCommitOrder, id)
		}
	}

	firstResult := make(chan error, 1)
	go func() {
		_, err := s.UpdateSettingAtomic(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE,
			func(current proto.Message) (proto.Message, error) {
				profile, ok := current.(*storepb.WorkspaceProfileSetting)
				if !ok {
					return nil, errors.Errorf("unexpected type %T", current)
				}
				profile.EnableMetricCollection = true
				return profile, nil
			}, recordPostCommit("first"))
		firstResult <- err
	}()
	// The first update has committed and is paused before publishing.
	select {
	case <-pauseFirst:
	case <-time.After(15 * time.Second):
		t.Fatal("first update never reached the publish window")
	}

	secondResult := make(chan error, 1)
	go func() {
		_, err := s.UpdateSettingAtomic(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE,
			func(current proto.Message) (proto.Message, error) {
				profile, ok := current.(*storepb.WorkspaceProfileSetting)
				if !ok {
					return nil, errors.Errorf("unexpected type %T", current)
				}
				profile.McpCapability = storepb.WorkspaceProfileSetting_DISABLED
				return profile, nil
			}, recordPostCommit("second"))
		secondResult <- err
	}()

	// The second update must not complete while the first sits between commit
	// and publish — completing here is exactly the out-of-order publish.
	select {
	case err := <-secondResult:
		t.Fatalf("second update published while the first held the publish window: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	close(resumeFirst)
	for _, result := range []chan error{firstResult, secondResult} {
		select {
		case err := <-result:
			require.NoError(t, err)
		case <-time.After(15 * time.Second):
			t.Fatal("update did not complete; possible deadlock")
		}
	}

	// The served cache must hold the final committed state: the second update
	// ran after the first's commit, so its merge carries BOTH fields.
	cached, err := s.GetSetting(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE)
	require.NoError(t, err)
	cachedProfile := mustProfile(t, cached)
	require.True(t, cachedProfile.EnableMetricCollection, "cache must not serve pre-first-update state")
	require.Equal(t, storepb.WorkspaceProfileSetting_DISABLED, cachedProfile.McpCapability,
		"cache must not be overwritten by the earlier committer's stale publish")
	postCommitMu.Lock()
	defer postCommitMu.Unlock()
	require.Equal(t, []string{"first", "second"}, postCommitOrder,
		"postCommit callbacks must run in commit order")
}

// TestSettingCacheNoFillFromUncachedReads pins that uncached reads do not
// populate the setting cache: GetSettingUncached (and ListSettings under it)
// runs outside the publish-ordering mutex, so a fill from that path could
// cache a pre-commit snapshot after a newer value was already published.
// Uncached readers must leave publication to the ordered paths.
func TestSettingCacheNoFillFromUncachedReads(t *testing.T) {
	ctx, db, s := newSettingAtomicFixture(t)
	// Drop the seed's cache entry so the next cached read must fill.
	s.DeleteCache()

	// An uncached read must not leave a cache entry behind ...
	_, err := s.GetSettingUncached(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE)
	require.NoError(t, err)

	// ... so after an out-of-band change, a cached read serves the new value.
	_, err = db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{enableMetricCollection}', 'true')
		WHERE workspace = 'default' AND name = 'WORKSPACE_PROFILE';
	`)
	require.NoError(t, err)

	cached, err := s.GetSetting(ctx, "default", storepb.SettingName_WORKSPACE_PROFILE)
	require.NoError(t, err)
	require.True(t, mustProfile(t, cached).EnableMetricCollection,
		"GetSetting must not serve a snapshot cached by an uncached read")
}
