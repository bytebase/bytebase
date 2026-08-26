package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func newMFAPendingTestStore(t *testing.T) *store.Store {
	t.Helper()
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}

// TestUpdateUserMFAConfigIfPending pins the compare-and-swap the confirming
// methods promote through. The version has to be part of the write: a caller
// that checked a user it read moments ago would otherwise still commit, and
// the two states worth protecting are a newer enrollment from another tab and
// an administrator's disable — reviving a factor that was just cleared.
func TestUpdateUserMFAConfigIfPending(t *testing.T) {
	ctx := context.Background()
	s := newMFAPendingTestStore(t)

	user, err := s.CreateUser(ctx, &store.UserMessage{
		Email:        "pending@example.com",
		Name:         "Pending",
		PasswordHash: "unused",
	})
	require.NoError(t, err)

	version := timestamppb.New(time.Now().UTC().Truncate(time.Microsecond))
	_, err = s.UpdateUser(ctx, user, &store.UpdateUserMessage{
		MFAConfig: &storepb.MFAConfig{
			TempOtpSecret:            "pending-secret",
			TempRecoveryCodes:        []string{"code-a"},
			TempOtpSecretCreatedTime: version,
		},
	})
	require.NoError(t, err)

	promoted := &storepb.MFAConfig{OtpSecret: "pending-secret", RecoveryCodes: []string{"code-a"}}

	stale := version.AsTime().Add(-time.Second)
	refused, err := s.UpdateUserMFAConfigIfPending(ctx, user.ID, stale, promoted)
	require.NoError(t, err)
	require.Nil(t, refused, "a version that is not the pending one must not commit")
	unchanged, err := s.GetUserByEmail(ctx, user.Email)
	require.NoError(t, err)
	require.Empty(t, unchanged.MFAConfig.GetOtpSecret(), "the refused write must leave the account alone")

	updated, err := s.UpdateUserMFAConfigIfPending(ctx, user.ID, version.AsTime(), promoted)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "pending-secret", updated.MFAConfig.GetOtpSecret())

	// Promotion clears the pending slot, so replaying the same confirmation —
	// a double-submit, or a tab that never learned it already went through —
	// finds nothing to promote rather than rewriting the factor.
	replayed, err := s.UpdateUserMFAConfigIfPending(ctx, user.ID, version.AsTime(), promoted)
	require.NoError(t, err)
	require.Nil(t, replayed, "a consumed pending version must not promote twice")
}
