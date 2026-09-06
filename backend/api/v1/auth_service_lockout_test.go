package v1

import (
	"context"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/runner/cleaner"
	"github.com/bytebase/bytebase/backend/store"
)

// The lockout rules of docs/design/login-attempt-lockout.md — every guessable
// credential claims a login_attempt slot for the identity under attack before
// the credential is checked, success deletes the row, and locked identities
// are refused with ResourceExhausted before any bcrypt, TOTP, or hash
// comparison — are pinned on a live server in backend/tests
// (TestLoginFailureLockout), and the claim itself in backend/store
// (TestLoginAttemptClaim). What stays here needs state no API can set up.

// TestEmailCodeLockoutClaims pins the claims around a pending code, which no
// API can plant without SMTP: the row outlives wrong guesses, a resend does
// not reset the counter, a matched code clears it and is single-use, and
// login and reset codes draw from one bucket per email. That the lockout
// precedes the code row at all is TestLoginFailureLockout in backend/tests.
func TestEmailCodeLockoutClaims(t *testing.T) {
	ctx := context.Background()
	stores := newAuthTestStore(t)
	service := &AuthService{store: stores, secret: "test-secret"}

	upsertCode := func(email string, purpose storepb.EmailVerificationCodePurpose, code string) {
		t.Helper()
		sent, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
			Email:      email,
			Purpose:    purpose,
			CodeHash:   hashEmailCode(service.secret, code),
			ExpiresAt:  time.Now().Add(emailCodeExpiry),
			LastSentAt: time.Now(),
		}, 0)
		require.NoError(t, err)
		require.True(t, sent)
	}

	t.Run("resending a code does not reset the counter", func(t *testing.T) {
		const email = "resend@example.com"
		upsertCode(email, storepb.EmailVerificationCodePurpose_LOGIN, "111111")
		for range 4 {
			err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		upsertCode(email, storepb.EmailVerificationCodePurpose_LOGIN, "222222")
		err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

		err = service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "222222")
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err),
			"the bound must survive code rotation: five wrong guesses lock the email even against a fresh valid code")
	})

	t.Run("wrong guesses do not delete the code row", func(t *testing.T) {
		const email = "cooldown@example.com"
		upsertCode(email, storepb.EmailVerificationCodePurpose_LOGIN, "111111")
		for range loginAttemptMax {
			err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		row, err := stores.GetEmailVerificationCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN)
		require.NoError(t, err)
		require.NotNil(t, row, "the code row must live until it expires or is consumed")

		sent, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
			Email:      email,
			Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
			CodeHash:   hashEmailCode(service.secret, "333333"),
			ExpiresAt:  time.Now().Add(emailCodeExpiry),
			LastSentAt: time.Now(),
		}, emailCodeResendCooldown)
		require.NoError(t, err)
		require.False(t, sent, "the resend cooldown must always have a row to evaluate — the exhaustion bypass is closed")
	})

	t.Run("a correct code within the limit clears the counter", func(t *testing.T) {
		const email = "matching@example.com"
		upsertCode(email, storepb.EmailVerificationCodePurpose_LOGIN, "111111")
		for range 4 {
			err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		require.NoError(t, service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "111111"))

		row, err := stores.GetEmailVerificationCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN)
		require.NoError(t, err)
		require.Nil(t, row, "a matched code is single-use")

		for range loginAttemptMax {
			err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		err = service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	})

	t.Run("login and reset codes share one bucket per email", func(t *testing.T) {
		const email = "shared@example.com"
		for range 3 {
			err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		for range 2 {
			err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_PASSWORD_RESET, "000000")
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err),
			"the attempt limit is per identity, across codes and purposes")
	})
}

func TestResetPasswordClearsPasswordLockout(t *testing.T) {
	ctx := context.Background()
	stores := newAuthTestStore(t)
	t.Setenv("EMAIL_CONFIG", "")
	service := &AuthService{store: stores, secret: "test-secret", profile: &config.Profile{}}

	const email = "reset@example.com"
	const oldPassword = "old-password-1024"
	const newPassword = "new-password-1024"
	createLockoutTestUser(ctx, t, stores, email, oldPassword)

	for range loginAttemptMax {
		_, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: email, Password: "wrong-password"})
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	}
	_, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: email, Password: oldPassword})
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	const resetCode = "123456"
	sent, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
		Email:      email,
		Purpose:    storepb.EmailVerificationCodePurpose_PASSWORD_RESET,
		CodeHash:   hashEmailCode(service.secret, resetCode),
		ExpiresAt:  time.Now().Add(emailCodeExpiry),
		LastSentAt: time.Now(),
	}, 0)
	require.NoError(t, err)
	require.True(t, sent)

	_, err = service.ResetPassword(ctx, connect.NewRequest(&v1pb.ResetPasswordRequest{
		Email:       email,
		Code:        resetCode,
		NewPassword: newPassword,
	}))
	require.NoError(t, err)

	user, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: email, Password: newPassword})
	require.NoError(t, err, "a successful password reset must also clear the PASSWORD lock")
	require.Equal(t, email, user.Email)
}

// TestLDAPLoginIdentity pins the provider-scoped lockout key. "/" is legal in
// an email local part, so a "/" separator would let the literal email account
// "corpldap/alice@corp.com" share — lock out, and on success clear — the
// bucket of LDAP user "alice@corp.com" on IDP "corpldap". ":" is legal in
// neither alphabet, so an LDAP identity can never be a valid email.
func TestLDAPLoginIdentity(t *testing.T) {
	t.Parallel()
	// The submitted username is kept verbatim: a case-exact directory
	// attribute can name two accounts differing only by case, and merging
	// them would let either lock — or clear — the other.
	require.Equal(t, "corp-ldap:Alice@Corp.com", ldapLoginIdentity("corp-ldap", "Alice@Corp.com"))
	require.NotEqual(t, ldapLoginIdentity("corp-ldap", "alice"), ldapLoginIdentity("corp-ldap", "Alice"))
	require.True(t, common.IsValidEmail("corpldap/alice@corp.com"))
	require.False(t, common.IsValidEmail(ldapLoginIdentity("corpldap", "alice@corp.com")))
}

// TestLoginAttemptRetentionOutlivesLockouts ties the cleaner's purge horizon
// to the lockout window: a retention shorter than the window would delete
// still-running locks, silently capping every lockout at the retention.
func TestLoginAttemptRetentionOutlivesLockouts(t *testing.T) {
	t.Parallel()
	require.Greater(t, cleaner.LoginAttemptRetentionPeriod, loginAttemptWindow,
		"the hourly purge must never delete a still-running lock")
}

func newAuthTestStore(t *testing.T) *store.Store {
	t.Helper()
	_, stores, _ := testcontainer.NewMetadataDB(t)
	return stores
}

func createLockoutTestUser(ctx context.Context, t *testing.T, stores *store.Store, email, password string) *store.UserMessage {
	t.Helper()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	user, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        email,
		Name:         email,
		Type:         storepb.PrincipalType_END_USER,
		PasswordHash: string(passwordHash),
		Profile:      &storepb.UserProfile{},
	})
	require.NoError(t, err)
	return user
}
