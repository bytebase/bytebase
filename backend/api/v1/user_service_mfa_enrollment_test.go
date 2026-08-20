package v1

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

const mfaEnrollmentTestWorkspace = "mfa-enrollment-test"

// newMFAEnrollmentFixture boots a migrated store with one workspace, because
// convertToUser resolves the subject's groups through the IAM manager and that
// needs somewhere real to look.
func newMFAEnrollmentFixture(t *testing.T) (*store.Store, *iam.Manager) {
	t.Helper()
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO workspace (resource_id) VALUES ('%s')`, mfaEnrollmentTestWorkspace))
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	return stores, iamManager
}

// TestConvertToUserWithholdsAnExpiredEnrollment covers the third way an MFA
// enrollment window ends, and the only one nothing writes down.
//
// A commit nils temp_otp_secret_created_time and a regenerate moves it, so
// either is visible to a client comparing what it holds against what it reads.
// Expiry does neither: isMFATempSecretExpired decides it by comparing the
// stored timestamp with the clock, and nothing clears the row. So a setup the
// user abandoned would go on reporting an open window forever, and the console,
// which carries the enrollment secrets across a refresh for exactly as long as
// the window it holds is the window being read, would carry them forever with
// it.
//
// The rule for what counts as expired is not duplicated to get this: the
// converter asks the same function the OTP verification asks.
func TestConvertToUserWithholdsAnExpiredEnrollment(t *testing.T) {
	ctx := context.Background()
	stores, iamManager := newMFAEnrollmentFixture(t)

	enrolling := func(t *testing.T, email string, age time.Duration) (context.Context, *store.UserMessage) {
		t.Helper()
		user, err := stores.CreateUser(ctx, &store.UserMessage{
			Email: email,
			Name:  email,
			Type:  storepb.PrincipalType_END_USER,
		})
		require.NoError(t, err)

		// CreateUser answers with an empty MFAConfig whatever it was handed, so
		// the enrollment has to be written as an update. Getting this wrong
		// makes every assertion below pass for the wrong reason: an empty
		// config reads as expired, so nothing is exposed and the test proves
		// nothing.
		user, err = stores.UpdateUser(ctx, user, &store.UpdateUserMessage{
			MFAConfig: &storepb.MFAConfig{
				TempOtpSecret:            "SEEDTHECONSOLEISSHOWING",
				TempRecoveryCodes:        []string{"code-1", "code-2"},
				TempOtpSecretCreatedTime: timestamppb.New(time.Now().Add(-age)),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, user.MFAConfig.TempOtpSecretCreatedTime,
			"precondition: the enrollment has to be stored for this to be testing anything")

		// The subject has to be the caller: the enrollment is only ever exposed
		// to the person enrolling.
		selfCtx := context.WithValue(ctx, common.WorkspaceIDContextKey, mfaEnrollmentTestWorkspace)
		selfCtx = context.WithValue(selfCtx, common.UserContextKey, user)
		return selfCtx, user
	}

	t.Run("an open window reads back open, without its secrets", func(t *testing.T) {
		selfCtx, user := enrolling(t, "open@example.com", time.Minute)

		got, err := convertToUser(selfCtx, iamManager, user)
		require.NoError(t, err)
		require.NotNil(t, got.TempOtpSecretCreatedTime,
			"the console counts down from this and it is not a secret")
		require.Empty(t, got.TempOtpSecret)
		require.Empty(t, got.TempRecoveryCodes)
	})

	t.Run("and the request that mints it gets the secrets too", func(t *testing.T) {
		selfCtx, user := enrolling(t, "minting@example.com", time.Minute)

		got, err := convertToUserMintingMFAEnrollment(selfCtx, iamManager, user)
		require.NoError(t, err)
		require.Equal(t, "SEEDTHECONSOLEISSHOWING", got.TempOtpSecret)
		require.Len(t, got.TempRecoveryCodes, 2)
	})

	t.Run("an expired window reads back as no window at all", func(t *testing.T) {
		selfCtx, user := enrolling(t, "expired@example.com", 6*time.Minute)

		got, err := convertToUser(selfCtx, iamManager, user)
		require.NoError(t, err)
		require.Nil(t, got.TempOtpSecretCreatedTime,
			"an abandoned setup must stop reporting an open enrollment, or the console holds its seed forever")
		require.Empty(t, got.TempOtpSecret)
		require.Empty(t, got.TempRecoveryCodes)
	})

	t.Run("and not even to the request that would have minted it", func(t *testing.T) {
		// UpdateUser refuses to verify against an expired seed, so a response
		// carrying one would be offering something the server will not accept.
		selfCtx, user := enrolling(t, "expired-mint@example.com", 6*time.Minute)

		got, err := convertToUserMintingMFAEnrollment(selfCtx, iamManager, user)
		require.NoError(t, err)
		require.Empty(t, got.TempOtpSecret)
		require.Empty(t, got.TempRecoveryCodes)
		require.Nil(t, got.TempOtpSecretCreatedTime)
	})

	t.Run("somebody else's enrollment is never exposed, open or not", func(t *testing.T) {
		selfCtx, subject := enrolling(t, "subject@example.com", time.Minute)
		_ = selfCtx
		other, err := stores.CreateUser(ctx, &store.UserMessage{
			Email: "onlooker@example.com",
			Name:  "onlooker",
			Type:  storepb.PrincipalType_END_USER,
		})
		require.NoError(t, err)

		otherCtx := context.WithValue(ctx, common.WorkspaceIDContextKey, mfaEnrollmentTestWorkspace)
		otherCtx = context.WithValue(otherCtx, common.UserContextKey, other)

		got, err := convertToUser(otherCtx, iamManager, subject)
		require.NoError(t, err)
		require.Nil(t, got.TempOtpSecretCreatedTime)
		require.Empty(t, got.TempOtpSecret)
		require.Empty(t, got.TempRecoveryCodes)
	})
}
