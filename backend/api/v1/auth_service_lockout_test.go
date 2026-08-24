package v1

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/runner/cleaner"
	"github.com/bytebase/bytebase/backend/store"
)

// The lockout tests below pin docs/design/login-attempt-lockout.md: every
// guessable credential claims a login_attempt slot for the identity under
// attack before the credential is checked, success deletes the row, and locked
// identities are refused with ResourceExhausted before any bcrypt, TOTP, or
// hash comparison — independent of audit logs, so identical on Cloud and
// self-hosted.

func TestPasswordLockoutClaims(t *testing.T) {
	ctx := context.Background()
	stores := newAuthTestStore(t)
	service := &AuthService{store: stores}

	const password = "1024bytebase"
	createLockoutTestUser(ctx, t, stores, "known@example.com", password)

	t.Run("unknown and known emails lock at the same attempt with the same error", func(t *testing.T) {
		for _, email := range []string{"known@example.com", "unknown@example.com"} {
			for range loginAttemptMax {
				_, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: email, Password: "wrong-password"})
				require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
				require.ErrorContains(t, err, errMsgInvalidCredentials)
			}
			_, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: email, Password: "wrong-password"})
			require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
			require.ErrorContains(t, err, errMsgTooManyPassword)
		}
	})

	t.Run("a locked identity is refused before bcrypt", func(t *testing.T) {
		_, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: "known@example.com", Password: password})
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err),
			"the correct password must not slip through a lock: the slot is claimed before the credential is checked")
	})

	t.Run("success clears the counter", func(t *testing.T) {
		const email = "clearing@example.com"
		createLockoutTestUser(ctx, t, stores, email, password)
		for range 3 {
			_, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: email, Password: "wrong-password"})
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		user, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: email, Password: password})
		require.NoError(t, err)
		require.Equal(t, email, user.Email)

		// The forgotten failures must not count against the fresh window.
		for range loginAttemptMax {
			_, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: email, Password: "wrong-password"})
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		_, err = service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: email, Password: "wrong-password"})
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	})

	t.Run("garbage identities never write a row", func(t *testing.T) {
		// Over-length emails are refused at the proto edge (see
		// TestAuthEmailFieldsAreLengthBounded); invalid syntax is refused here.
		for range loginAttemptMax + 1 {
			_, err := service.getAndVerifyUser(ctx, &v1pb.LoginRequest{Email: "not-an-email", Password: "wrong-password"})
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err),
				"an invalid-syntax email is rejected before the claim, with the same error as a wrong password")
		}
		var count int
		require.NoError(t, stores.GetDB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM login_attempt WHERE identity = 'not-an-email'`).Scan(&count))
		require.Zero(t, count)
	})
}

func TestEmailCodeLockoutClaims(t *testing.T) {
	ctx := context.Background()
	stores := newAuthTestStore(t)
	service := &AuthService{store: stores, secret: "test-secret"}

	upsertCode := func(email string, purpose storepb.EmailVerificationCodePurpose, code string) {
		t.Helper()
		sent, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
			Email:      email,
			Purpose:    purpose,
			CodeHash:   service.hashEmailCode(code),
			ExpiresAt:  time.Now().Add(emailCodeExpiry),
			LastSentAt: time.Now(),
		}, 0)
		require.NoError(t, err)
		require.True(t, sent)
	}

	t.Run("the lockout precedes the code row", func(t *testing.T) {
		const email = "no-code@example.com"
		for range loginAttemptMax {
			err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			require.ErrorContains(t, err, "invalid or expired code")
		}
		err := service.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, "000000")
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err),
			"guessing must be bounded per identity even when no code is pending")
		require.ErrorContains(t, err, errMsgTooManyEmailCode)
	})

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

	t.Run("wrong guesses no longer delete the code row", func(t *testing.T) {
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
			CodeHash:   service.hashEmailCode("333333"),
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
		CodeHash:   service.hashEmailCode(resetCode),
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

func TestMFALockoutClaims(t *testing.T) {
	ctx := context.Background()
	stores := newAuthTestStore(t)
	const secret = "test-secret"
	service := &AuthService{store: stores, secret: secret}

	attempt := func(email string, code *string) error {
		t.Helper()
		token, err := auth.GenerateMFATempTokenWithLoginMethod(email, string(loginAuthMethodPassword), secret, mfaTempTokenDuration)
		require.NoError(t, err)
		_, _, err = service.completeMFALogin(ctx, &v1pb.LoginRequest{MfaTempToken: &token, OtpCode: code})
		return err
	}

	t.Run("guesses are bounded per identity across temp tokens", func(t *testing.T) {
		const email = "mfa-locked@example.com"
		_, otpSecret := setupMFAUser(ctx, t, stores, email)
		for i := range loginAttemptMax {
			// A fresh token per attempt must not buy fresh guesses.
			err := attempt(email, ptr("not-a-code"))
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "attempt %d", i+1)
			require.ErrorContains(t, err, errMsgInvalidMFACode)
		}
		err := attempt(email, ptr("not-a-code"))
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		require.ErrorContains(t, err, errMsgTooManyMFA)

		validOTP, err := totp.GenerateCode(otpSecret, time.Now())
		require.NoError(t, err)
		err = attempt(email, &validOTP)
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err),
			"a locked identity is refused before the TOTP comparison")
	})

	t.Run("success clears the counter", func(t *testing.T) {
		const email = "mfa-clearing@example.com"
		_, otpSecret := setupMFAUser(ctx, t, stores, email)
		for range loginAttemptMax - 1 {
			err := attempt(email, ptr("not-a-code"))
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		validOTP, err := totp.GenerateCode(otpSecret, time.Now())
		require.NoError(t, err)
		require.NoError(t, attempt(email, &validOTP))

		for range loginAttemptMax {
			err := attempt(email, ptr("not-a-code"))
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		}
		err = attempt(email, ptr("not-a-code"))
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	})

	t.Run("codeless requests claim nothing", func(t *testing.T) {
		const email = "mfa-codeless@example.com"
		setupMFAUser(ctx, t, stores, email)
		for range loginAttemptMax + 1 {
			err := attempt(email, nil)
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			require.ErrorContains(t, err, "OTP or recovery code is required")
		}
		// The budget is untouched: a real wrong guess still gets the
		// invalid-code answer, not a lockout.
		err := attempt(email, ptr("not-a-code"))
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		require.ErrorContains(t, err, errMsgInvalidMFACode)
	})
}

func TestSwitchWorkspaceMFAClaims(t *testing.T) {
	ctx := context.Background()
	stores := newAuthTestStore(t)
	const secret = "test-secret"
	const workspaceID = "switch-mfa-test"
	const email = "switcher@example.com"

	_, err := stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: workspaceID,
		Payload:    &storepb.WorkspacePayload{Title: "Switch MFA test"},
	}, email)
	require.NoError(t, err)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	require.NoError(t, licenseService.StoreLicense(ctx, workspaceID, authTestEnterpriseLicense))
	service := NewAuthService(stores, secret, licenseService, &config.Profile{}, nil)

	user, _ := setupMFAUser(ctx, t, stores, email)

	userCtx := context.WithValue(ctx, common.UserContextKey, user)
	token, err := auth.GenerateMFATempTokenWithLoginMethod(email, string(loginAuthMethodPassword), secret, mfaTempTokenDuration)
	require.NoError(t, err)

	// An argument error must not consume attempt budget — the loop below
	// still needs the full allowance before the lock engages.
	_, err = service.SwitchWorkspace(userCtx, connect.NewRequest(&v1pb.SwitchWorkspaceRequest{
		Workspace:    common.FormatWorkspace(workspaceID),
		MfaTempToken: &token,
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	for range loginAttemptMax {
		_, err := service.SwitchWorkspace(userCtx, connect.NewRequest(&v1pb.SwitchWorkspaceRequest{
			Workspace:    common.FormatWorkspace(workspaceID),
			MfaTempToken: &token,
			OtpCode:      ptr("not-a-code"),
		}))
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		require.ErrorContains(t, err, errMsgInvalidMFACode)
	}
	_, err = service.SwitchWorkspace(userCtx, connect.NewRequest(&v1pb.SwitchWorkspaceRequest{
		Workspace:    common.FormatWorkspace(workspaceID),
		MfaTempToken: &token,
		OtpCode:      ptr("not-a-code"),
	}))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err),
		"MFA completion on workspace switch draws from the same per-identity bucket")
	require.ErrorContains(t, err, errMsgTooManyMFA)
}

// TestLDAPLoginIdentity pins the provider-scoped lockout key. "/" is legal in
// an email local part, so a "/" separator would let the literal email account
// "corpldap/alice@corp.com" share — lock out, and on success clear — the
// bucket of LDAP user "alice@corp.com" on IDP "corpldap". ":" is legal in
// neither alphabet, so an LDAP identity can never be a valid email.
func TestLDAPLoginIdentity(t *testing.T) {
	require.Equal(t, "corp-ldap:alice@corp.com", ldapLoginIdentity("corp-ldap", "  Alice@Corp.com "))
	require.True(t, common.IsValidEmail("corpldap/alice@corp.com"))
	require.False(t, common.IsValidEmail(ldapLoginIdentity("corpldap", "alice@corp.com")))
}

func TestLDAPLockoutClaims(t *testing.T) {
	ctx := context.Background()
	stores := newAuthTestStore(t)
	service := &AuthService{store: stores, secret: "test-secret", profile: &config.Profile{ExternalURL: "http://localhost"}}

	const idpID = "ldap-lockout-test"
	_, err := stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: "ldap-lockout-ws",
		Payload:    &storepb.WorkspacePayload{Title: "LDAP lockout test"},
	}, "admin@example.com")
	require.NoError(t, err)
	_, err = stores.CreateIdentityProvider(ctx, &store.IdentityProviderMessage{
		ResourceID: idpID,
		Workspace:  "ldap-lockout-ws",
		Title:      "Unreachable directory",
		Type:       storepb.IdentityProviderType_LDAP,
		Config: &storepb.IdentityProviderConfig{Config: &storepb.IdentityProviderConfig_LdapConfig{LdapConfig: &storepb.LDAPIdentityProviderConfig{
			Host:         "127.0.0.1",
			Port:         1,
			BindDn:       "cn=admin,dc=example,dc=com",
			BindPassword: "unused",
			BaseDn:       "dc=example,dc=com",
			UserFilter:   "(uid=%s)",
			FieldMapping: &storepb.FieldMapping{Identifier: "uid"},
		}}},
	})
	require.NoError(t, err)

	request := &v1pb.LoginRequest{IdpName: "idps/" + idpID, Email: "  Alice  ", Password: "wrong"}
	_, err = service.getOrCreateUserWithIDP(ctx, request)
	require.Error(t, err, "the directory is unreachable")

	// The slot is claimed before the bind, under the provider-scoped key.
	var identity string
	require.NoError(t, stores.GetDB().QueryRowContext(ctx,
		`SELECT identity FROM login_attempt WHERE kind = 'PASSWORD'`).Scan(&identity))
	require.Equal(t, idpID+":alice", identity)

	for range loginAttemptMax - 1 {
		_, err := service.getOrCreateUserWithIDP(ctx, request)
		require.Error(t, err)
		require.NotEqual(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	}
	_, err = service.getOrCreateUserWithIDP(ctx, request)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	require.ErrorContains(t, err, errMsgTooManyPassword)
}

// TestLoginAttemptRetentionOutlivesLockouts ties the cleaner's purge horizon
// to the lockout window: a retention shorter than the window would delete
// still-running locks, silently capping every lockout at the retention.
func TestLoginAttemptRetentionOutlivesLockouts(t *testing.T) {
	require.Greater(t, cleaner.LoginAttemptRetentionPeriod, loginAttemptWindow,
		"the hourly purge must never delete a still-running lock")
}

func newAuthTestStore(t *testing.T) *store.Store {
	t.Helper()
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	require.NoError(t, migrator.MigrateSchema(ctx, container.GetDB()))

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })
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

func setupMFAUser(ctx context.Context, t *testing.T, stores *store.Store, email string) (*store.UserMessage, string) {
	t.Helper()
	user := createLockoutTestUser(ctx, t, stores, email, "1024bytebase")
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "bytebase-test", AccountName: email})
	require.NoError(t, err)
	user, err = stores.UpdateUser(ctx, user, &store.UpdateUserMessage{
		MFAConfig: &storepb.MFAConfig{OtpSecret: key.Secret()},
	})
	require.NoError(t, err)
	return user, key.Secret()
}
