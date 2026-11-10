package store_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	cacheLifetimeWorkspace = "cache-lifetime"
	cacheLifetimeEmail     = "cache-lifetime@example.com"
)

var cacheLifetimeMember = common.FormatUserEmail(cacheLifetimeEmail)

// recoveryChanges write the three rows that `bytebase recovery` changes. seen
// reports whether a read through the given store returns the change; it
// returns an error rather than failing the test because require.Eventually
// calls it on another goroutine.
var recoveryChanges = []struct {
	name  string
	apply func(ctx context.Context, t *testing.T, recovery *store.Store)
	seen  func(ctx context.Context, server *store.Store) (bool, error)
}{
	{
		name: "user password",
		apply: func(ctx context.Context, t *testing.T, recovery *store.Store) {
			user, err := recovery.GetUserByEmail(ctx, cacheLifetimeEmail)
			require.NoError(t, err)
			_, err = recovery.UpdateUser(ctx, user, &store.UpdateUserMessage{PasswordHash: new("new-hash")})
			require.NoError(t, err)
		},
		seen: func(ctx context.Context, server *store.Store) (bool, error) {
			user, err := server.GetUserByEmail(ctx, cacheLifetimeEmail)
			if err != nil {
				return false, err
			}
			return user != nil && user.PasswordHash == "new-hash", nil
		},
	},
	{
		name: "password sign-in setting",
		apply: func(ctx context.Context, t *testing.T, recovery *store.Store) {
			setDisallowPasswordSignin(ctx, t, recovery, false)
		},
		seen: func(ctx context.Context, server *store.Store) (bool, error) {
			profile, err := server.GetWorkspaceProfileSetting(ctx, cacheLifetimeWorkspace)
			if err != nil {
				return false, err
			}
			return !profile.DisallowPasswordSignin, nil
		},
	},
	{
		name: "workspace role",
		apply: func(ctx context.Context, t *testing.T, recovery *store.Store) {
			_, err := recovery.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
				Workspace: cacheLifetimeWorkspace,
				Member:    cacheLifetimeMember,
				Roles:     []string{"roles/workspaceMember"},
			})
			require.NoError(t, err)
		},
		seen: func(ctx context.Context, server *store.Store) (bool, error) {
			policy, err := server.GetWorkspaceIamPolicy(ctx, cacheLifetimeWorkspace)
			if err != nil {
				return false, err
			}
			return hasMember(policy.Policy, cacheLifetimeMember), nil
		},
	},
}

// newCacheLifetimeFixture returns a store shaped like a running server, with
// its caches on and expiring after lifetime, and the fixture's store, shaped
// like `bytebase recovery` (its own process, caches off), on the same
// database. The seeded workspace is locked out the way recovery finds it:
// password sign-in is disallowed and the one user has no workspace role.
func newCacheLifetimeFixture(t *testing.T, lifetime time.Duration) (context.Context, *store.Store, *store.Store) {
	t.Helper()
	fixture := newStorePostgresFixture(t, "")
	ctx, recovery := fixture.ctx, fixture.store
	server, err := store.NewWithOutOfBandCacheLifetimeForTest(ctx, fixture.pgURL, true, lifetime)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, server.Close()) })

	_, err = recovery.CreateWorkspace(ctx, &store.WorkspaceMessage{ResourceID: cacheLifetimeWorkspace}, "admin@example.com")
	require.NoError(t, err)
	setDisallowPasswordSignin(ctx, t, recovery, true)
	_, err = recovery.CreateUser(ctx, &store.UserMessage{
		Email:        cacheLifetimeEmail,
		Name:         cacheLifetimeEmail,
		PasswordHash: "old-hash",
	})
	require.NoError(t, err)
	return ctx, server, recovery
}

func TestServerReadsRecoveryChangesAfterCacheLifetime(t *testing.T) {
	t.Parallel()
	ctx, server, recovery := newCacheLifetimeFixture(t, 500*time.Millisecond)
	warmThenApplyRecoveryChanges(ctx, t, server, recovery)

	for _, change := range recoveryChanges {
		require.Eventually(t, func() bool {
			seen, err := change.seen(ctx, server)
			return err == nil && seen
		}, 15*time.Second, 50*time.Millisecond, "%s: the server still serves the old value after the cache lifetime", change.name)
	}
}

func TestServerServesCachedValueWithinCacheLifetime(t *testing.T) {
	t.Parallel()
	ctx, server, recovery := newCacheLifetimeFixture(t, store.OutOfBandCacheLifetime)
	warmThenApplyRecoveryChanges(ctx, t, server, recovery)

	for _, change := range recoveryChanges {
		seen, err := change.seen(ctx, server)
		require.NoError(t, err)
		require.False(t, seen, "%s: the server read the change before the cache lifetime ended", change.name)
	}
}

func warmThenApplyRecoveryChanges(ctx context.Context, t *testing.T, server, recovery *store.Store) {
	t.Helper()
	for _, change := range recoveryChanges {
		seen, err := change.seen(ctx, server)
		require.NoError(t, err)
		require.False(t, seen, change.name)
	}
	for _, change := range recoveryChanges {
		change.apply(ctx, t, recovery)
	}
}

func hasMember(policy *storepb.IamPolicy, member string) bool {
	for _, binding := range policy.Bindings {
		if slices.Contains(binding.Members, member) {
			return true
		}
	}
	return false
}

func setDisallowPasswordSignin(ctx context.Context, t *testing.T, s *store.Store, disallow bool) {
	t.Helper()
	_, err := s.UpdateSettingAtomic(ctx, cacheLifetimeWorkspace, storepb.SettingName_WORKSPACE_PROFILE,
		func(current proto.Message) (proto.Message, error) {
			profile, ok := current.(*storepb.WorkspaceProfileSetting)
			if !ok {
				return nil, errors.Errorf("unexpected type %T", current)
			}
			profile.DisallowPasswordSignin = disallow
			return profile, nil
		}, nil)
	require.NoError(t, err)
}
