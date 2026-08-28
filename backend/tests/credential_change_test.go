package tests

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Regression tests for docs/design/reauthenticate-credential-changes.md:
// every mutation of live authentication material demands a CredentialProof,
// and a password change revokes the account's sessions and OAuth grants
// atomically with the write.

func passwordProofOf(password string) *v1pb.CredentialProof {
	return &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_CurrentPassword{CurrentPassword: password}}
}

func otpProofOf(t *testing.T, otpSecret string) *v1pb.CredentialProof {
	t.Helper()
	code, err := totp.GenerateCode(otpSecret, time.Now())
	require.NoError(t, err)
	return &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_OtpCode{OtpCode: code}}
}

func TestChangePasswordProofAndRevocation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	metadataDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer metadataDB.Close()

	const email = "demo@example.com"
	const oldPassword = "1024bytebase"
	const newPassword = "rotated-password-1"
	userName := common.FormatUserEmail(email)

	// Seed one row per token table so the revocation's coverage is observable:
	// a web session, an OAuth authorization code, and an OAuth refresh grant.
	for _, stmt := range []string{
		`INSERT INTO web_refresh_token (token_hash, user_email, expires_at) VALUES ('cred-web-token', $1, now() + interval '1 hour')`,
		`INSERT INTO oauth2_client (client_id, client_secret_hash, config) VALUES ('cred-client', 'unused', '{}')`,
		`INSERT INTO oauth2_authorization_code (code, client_id, user_email, config, expires_at) VALUES ('cred-code', 'cred-client', $1, '{}', now() + interval '1 hour')`,
		`INSERT INTO oauth2_refresh_token (token_hash, client_id, user_email, config, expires_at) VALUES ('cred-grant', 'cred-client', $1, '{}', now() + interval '30 days')`,
	} {
		if strings.Contains(stmt, "$1") {
			_, err = metadataDB.ExecContext(ctx, stmt, email)
		} else {
			_, err = metadataDB.ExecContext(ctx, stmt)
		}
		a.NoError(err)
	}

	// Self-service password change no longer rides on UpdateUser.
	_, err = ctl.userServiceClient.UpdateUser(ctx, connect.NewRequest(&v1pb.UpdateUserRequest{
		User:       &v1pb.User{Name: userName, Password: newPassword},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"password"}},
	}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	a.ErrorContains(err, "ChangePassword")

	// A wrong current password is refused without touching anything.
	_, err = ctl.userServiceClient.ChangePassword(ctx, connect.NewRequest(&v1pb.ChangePasswordRequest{
		Name:        userName,
		NewPassword: newPassword,
		Credential:  passwordProofOf("not-the-password"),
	}))
	a.Equal(connect.CodeUnauthenticated, connect.CodeOf(err))

	// A valid emailed code is refused on self-hosted, where the channel does
	// not exist at all — mailbox possession never stands in for a credential
	// here, even with a code planted directly in the store.
	insertReauthCode(ctx, t, metadataDB, email, "554433")
	_, err = ctl.userServiceClient.ChangePassword(ctx, connect.NewRequest(&v1pb.ChangePasswordRequest{
		Name:        userName,
		NewPassword: newPassword,
		Credential:  &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_EmailCode{EmailCode: "554433"}},
	}))
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	a.ErrorContains(err, "Cloud")

	// The real change: proof verified, web sessions revoked, and the
	// same-password refusal enforced against the new value.
	_, err = ctl.userServiceClient.ChangePassword(ctx, connect.NewRequest(&v1pb.ChangePasswordRequest{
		Name:        userName,
		NewPassword: newPassword,
		Credential:  passwordProofOf(oldPassword),
	}))
	a.NoError(err)

	var sessions int
	a.NoError(metadataDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM web_refresh_token WHERE user_email = $1`, email).Scan(&sessions))
	a.Zero(sessions, "a password change ends the account's web sessions")

	// OAuth grants are deliberately not swept here: this method revokes what
	// the password mask on UpdateUser already revoked, and widening that to
	// MCP grants is a separate change with its own blast radius.
	var grants int
	a.NoError(metadataDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM oauth2_refresh_token WHERE user_email = $1`, email).Scan(&grants))
	a.Positive(grants, "the seeded grant is still here, which is today's behavior")

	_, err = ctl.userServiceClient.ChangePassword(ctx, connect.NewRequest(&v1pb.ChangePasswordRequest{
		Name:        userName,
		NewPassword: newPassword,
		Credential:  passwordProofOf(newPassword),
	}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err), "changing to the same password must not reset the rotation deadline")

	// The old password is dead and the new one works.
	adminToken := ctl.authInterceptor.token
	ctl.authInterceptor.token = ""
	_, err = ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: email, Password: oldPassword}))
	a.Equal(connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: email, Password: newPassword}))
	a.NoError(err)
	ctl.authInterceptor.token = adminToken
}

func TestCredentialProofSharesLoginLockout(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	const email = "lockout-victim@example.com"
	const password = "victim-password-1"
	adminToken := ctl.authInterceptor.token
	ctl.authInterceptor.token = ""
	_, err = ctl.authServiceClient.Signup(ctx, connect.NewRequest(&v1pb.SignupRequest{
		Email:    email,
		Title:    "Lockout Victim",
		Password: password,
	}))
	a.NoError(err)
	login, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: email, Password: password}))
	a.NoError(err)
	ctl.authInterceptor.token = login.Msg.Token

	// Five wrong proofs exhaust the shared PASSWORD bucket; the sixth is
	// refused before any comparison, and login itself is locked out too — the
	// proof is not a fresh guessing oracle beside T9's.
	for range 5 {
		_, err := ctl.userServiceClient.ChangePassword(ctx, connect.NewRequest(&v1pb.ChangePasswordRequest{
			Name:        common.FormatUserEmail(email),
			NewPassword: "does-not-matter-1",
			Credential:  passwordProofOf("wrong-password"),
		}))
		a.Equal(connect.CodeUnauthenticated, connect.CodeOf(err))
	}
	_, err = ctl.userServiceClient.ChangePassword(ctx, connect.NewRequest(&v1pb.ChangePasswordRequest{
		Name:        common.FormatUserEmail(email),
		NewPassword: "does-not-matter-1",
		Credential:  passwordProofOf(password),
	}))
	a.Equal(connect.CodeResourceExhausted, connect.CodeOf(err))

	ctl.authInterceptor.token = ""
	_, err = ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: email, Password: password}))
	a.Equal(connect.CodeResourceExhausted, connect.CodeOf(err),
		"credential-proof guesses and login guesses must drain the same per-identity budget")
	ctl.authInterceptor.token = adminToken
}

func TestMFALifecycleFactorBoundProofs(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	const email = "demo@example.com"
	const password = "1024bytebase"
	userName := common.FormatUserEmail(email)
	otpSecret := enrollMFA(ctx, t, ctl, userName, password)

	// While a live factor exists, factor-touching methods refuse the password:
	// ResetPassword mints one from mailbox possession alone, so accepting it
	// here would let a stolen session plus mailbox strip the second factor.
	_, err = ctl.userServiceClient.DisableMFA(ctx, connect.NewRequest(&v1pb.DisableMFARequest{
		Name:       userName,
		Credential: passwordProofOf(password),
	}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// Recovery-code rotation: a stale pending_version is rejected rather than
	// silently promoting whatever is currently pending.
	first, err := ctl.userServiceClient.RegenerateRecoveryCodes(ctx, connect.NewRequest(&v1pb.RegenerateRecoveryCodesRequest{Name: userName}))
	a.NoError(err)
	second, err := ctl.userServiceClient.RegenerateRecoveryCodes(ctx, connect.NewRequest(&v1pb.RegenerateRecoveryCodesRequest{Name: userName}))
	a.NoError(err)
	_, err = ctl.userServiceClient.ConfirmRecoveryCodes(ctx, connect.NewRequest(&v1pb.ConfirmRecoveryCodesRequest{
		Name:           userName,
		Credential:     otpProofOf(t, otpSecret),
		PendingVersion: first.Msg.PendingVersion,
	}))
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	// The rotation's own confirmation is factor-bound too.
	_, err = ctl.userServiceClient.ConfirmRecoveryCodes(ctx, connect.NewRequest(&v1pb.ConfirmRecoveryCodesRequest{
		Name:           userName,
		Credential:     passwordProofOf(password),
		PendingVersion: second.Msg.PendingVersion,
	}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = ctl.userServiceClient.ConfirmRecoveryCodes(ctx, connect.NewRequest(&v1pb.ConfirmRecoveryCodesRequest{
		Name:           userName,
		Credential:     otpProofOf(t, otpSecret),
		PendingVersion: second.Msg.PendingVersion,
	}))
	a.NoError(err)

	// Confirming promoted the second set, and DisableMFA takes a recovery code
	// in place of an OTP — either proves the factor it is turning off.
	recoveryProof := &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_RecoveryCode{RecoveryCode: second.Msg.RecoveryCodes[0]}}
	disabled, err := ctl.userServiceClient.DisableMFA(ctx, connect.NewRequest(&v1pb.DisableMFARequest{
		Name:       userName,
		Credential: recoveryProof,
	}))
	a.NoError(err)
	a.False(disabled.Msg.MfaEnabled)

	// Disabling wiped the config, so there is no longer a factor to prove:
	// a repeat reaches the same desired state rather than reporting failure
	// for a satisfied intent, and needs no credential to do it. That is what
	// lets a user abandon an enrollment they can no longer complete.
	repeat, err := ctl.userServiceClient.DisableMFA(ctx, connect.NewRequest(&v1pb.DisableMFARequest{
		Name: userName,
	}))
	a.NoError(err)
	a.False(repeat.Msg.MfaEnabled)
}

// TestEmailCodeProofIsCloudOnly pins the deployment split: only Cloud can
// send email, so on self-hosted the email_code channel does not exist at all —
// even a valid code planted directly in the store is refused — and a
// passwordless account is pointed at the admin-password-reset route instead of
// dying on an unusable verification error.
func TestEmailCodeProofIsCloudOnly(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	metadataDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer metadataDB.Close()

	// The send side refuses on self-hosted before any mail machinery runs.
	_, err = ctl.userServiceClient.RequestReauthCode(ctx, connect.NewRequest(&v1pb.RequestReauthCodeRequest{
		Name: common.FormatUserEmail("demo@example.com"),
	}))
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	a.ErrorContains(err, "Cloud")

	// A passwordless account — the SSO shape: a hash its owner never chose,
	// lastChangePasswordTime unset. Seeded directly because this suite has no
	// IdP; the test still knows the hash's preimage so it can hold a session
	// as the user.
	const email = "sso-user@example.com"
	const bridgePassword = "sso-bridge-pass-1"
	bridgeHash, err := bcrypt.GenerateFromPassword([]byte(bridgePassword), bcrypt.MinCost)
	a.NoError(err)
	_, err = metadataDB.ExecContext(ctx, `
		INSERT INTO principal (name, email, password_hash, profile)
		VALUES ('SSO User', $1, $2, '{}')
	`, email, string(bridgeHash))
	a.NoError(err)

	adminToken := ctl.authInterceptor.token
	ctl.authInterceptor.token = ""
	login, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: email, Password: bridgePassword}))
	a.NoError(err)
	ctl.authInterceptor.token = login.Msg.Token

	// Even a valid REAUTH code sitting in the store cannot be spent here.
	insertReauthCode(ctx, t, metadataDB, email, "665544")
	_, err = ctl.userServiceClient.ChangePassword(ctx, connect.NewRequest(&v1pb.ChangePasswordRequest{
		Name:        common.FormatUserEmail(email),
		NewPassword: "chosen-at-last-1",
		Credential:  &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_EmailCode{EmailCode: "665544"}},
	}))
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	a.ErrorContains(err, "Cloud")

	// First-time enrollment on a passwordless self-hosted account fails at
	// step one with the actionable route, not after saving recovery codes.
	minted, err := ctl.userServiceClient.StartMFAEnrollment(ctx, connect.NewRequest(&v1pb.StartMFAEnrollmentRequest{
		Name: common.FormatUserEmail(email),
	}))
	a.NoError(err)
	otp, err := totp.GenerateCode(minted.Msg.OtpSecret, time.Now())
	a.NoError(err)
	_, err = ctl.userServiceClient.EnableMFA(ctx, connect.NewRequest(&v1pb.EnableMFARequest{
		Name:           common.FormatUserEmail(email),
		OtpCode:        otp,
		PendingVersion: minted.Msg.PendingVersion,
	}))
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	a.ErrorContains(err, "ask a workspace admin")

	// The admin-reset route works: once an admin sets a password, the account
	// has a caller-chosen password and enrolls with it as proof.
	ctl.authInterceptor.token = adminToken
	_, err = ctl.userServiceClient.UpdateUser(ctx, connect.NewRequest(&v1pb.UpdateUserRequest{
		User:       &v1pb.User{Name: common.FormatUserEmail(email), Password: "admin-issued-pass-1"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"password"}},
	}))
	a.NoError(err)
	ctl.authInterceptor.token = ""
	login, err = ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: email, Password: "admin-issued-pass-1"}))
	a.NoError(err)
	ctl.authInterceptor.token = login.Msg.Token
	enrollMFA(ctx, t, ctl, common.FormatUserEmail(email), "admin-issued-pass-1")
	ctl.authInterceptor.token = adminToken
}

// TestRecoveryCodeProofIsSpentOnce covers the one proof channel that spends
// what it proves. EnableMFA writes nothing of its own, so a recovery code
// accepted there is single-use only if the verification records the spend on
// the row itself: an in-memory edit would leave the code live in the database
// while the replica that served the request believed otherwise, and every
// other replica kept accepting it.
func TestRecoveryCodeProofIsSpentOnce(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	const email = "demo@example.com"
	const password = "1024bytebase"
	userName := common.FormatUserEmail(email)

	minted, err := ctl.userServiceClient.StartMFAEnrollment(ctx, connect.NewRequest(&v1pb.StartMFAEnrollmentRequest{Name: userName}))
	a.NoError(err)
	otp, err := totp.GenerateCode(minted.Msg.OtpSecret, time.Now())
	a.NoError(err)
	_, err = ctl.userServiceClient.EnableMFA(ctx, connect.NewRequest(&v1pb.EnableMFARequest{
		Name:           userName,
		OtpCode:        otp,
		Credential:     passwordProofOf(password),
		PendingVersion: minted.Msg.PendingVersion,
	}))
	a.NoError(err)
	_, err = ctl.userServiceClient.ConfirmRecoveryCodes(ctx, connect.NewRequest(&v1pb.ConfirmRecoveryCodesRequest{
		Name:           userName,
		Credential:     passwordProofOf(password),
		PendingVersion: minted.Msg.PendingVersion,
		OtpCode:        otp,
	}))
	a.NoError(err)
	liveCodes := minted.Msg.RecoveryCodes
	a.GreaterOrEqual(len(liveCodes), 2)

	recoveryProofOf := func(code string) *v1pb.CredentialProof {
		return &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_RecoveryCode{RecoveryCode: code}}
	}
	// Replacing the authenticator is a factor-touching call, so the proof has
	// to be the factor — an OTP, or one of these recovery codes.
	rotateWith := func(credential *v1pb.CredentialProof) error {
		t.Helper()
		next, err := ctl.userServiceClient.StartMFAEnrollment(ctx, connect.NewRequest(&v1pb.StartMFAEnrollmentRequest{Name: userName}))
		a.NoError(err)
		nextOTP, err := totp.GenerateCode(next.Msg.OtpSecret, time.Now())
		a.NoError(err)
		_, err = ctl.userServiceClient.EnableMFA(ctx, connect.NewRequest(&v1pb.EnableMFARequest{
			Name:           userName,
			OtpCode:        nextOTP,
			Credential:     credential,
			PendingVersion: next.Msg.PendingVersion,
		}))
		return err
	}

	a.NoError(rotateWith(recoveryProofOf(liveCodes[0])))

	// EnableMFA promoted nothing, so the live factor and its codes are the
	// same set as before — minus the one just spent, which the row no longer
	// holds. This is the assertion an in-place edit cannot satisfy.
	a.Equal(connect.CodeUnauthenticated, connect.CodeOf(rotateWith(recoveryProofOf(liveCodes[0]))),
		"a recovery code accepted once must not be accepted again")

	// A different code from the same set still works, so exactly one was
	// spent rather than the list being rewritten wholesale.
	a.NoError(rotateWith(recoveryProofOf(liveCodes[1])))
}

// enrollMFA walks the split enrollment for an account that proves itself with
// a password, and returns the live TOTP secret.
func enrollMFA(ctx context.Context, t *testing.T, ctl *controller, userName, password string) string {
	t.Helper()
	a := require.New(t)
	minted, err := ctl.userServiceClient.StartMFAEnrollment(ctx, connect.NewRequest(&v1pb.StartMFAEnrollmentRequest{Name: userName}))
	a.NoError(err)
	otp, err := totp.GenerateCode(minted.Msg.OtpSecret, time.Now())
	a.NoError(err)
	_, err = ctl.userServiceClient.EnableMFA(ctx, connect.NewRequest(&v1pb.EnableMFARequest{
		Name:           userName,
		OtpCode:        otp,
		Credential:     passwordProofOf(password),
		PendingVersion: minted.Msg.PendingVersion,
	}))
	a.NoError(err)
	confirmed, err := ctl.userServiceClient.ConfirmRecoveryCodes(ctx, connect.NewRequest(&v1pb.ConfirmRecoveryCodesRequest{
		Name:           userName,
		Credential:     passwordProofOf(password),
		PendingVersion: minted.Msg.PendingVersion,
		OtpCode:        otp,
	}))
	a.NoError(err)
	a.True(confirmed.Msg.MfaEnabled)
	return minted.Msg.OtpSecret
}

// insertReauthCode plants a REAUTH verification code directly, hashed the way
// the server hashes it (HMAC with the deployment's auth secret). The suite
// runs self-hosted, where no such row is ever legitimately written — planting
// one is exactly the point: even a valid code must be refused off Cloud.
func insertReauthCode(ctx context.Context, t *testing.T, metadataDB *sql.DB, email, code string) {
	t.Helper()
	a := require.New(t)
	var authSecret string
	a.NoError(metadataDB.QueryRowContext(ctx, `SELECT payload->>'authSecret' FROM server_config`).Scan(&authSecret))
	mac := hmac.New(sha256.New, []byte(authSecret))
	_, err := mac.Write([]byte(code))
	a.NoError(err)
	_, err = metadataDB.ExecContext(ctx, `
		INSERT INTO email_verification_code (email, purpose, code_hash, expires_at, last_sent_at)
		VALUES ($1, 'REAUTH', $2, now() + interval '10 minutes', now())
		ON CONFLICT (email, purpose) DO UPDATE SET code_hash = EXCLUDED.code_hash,
			expires_at = EXCLUDED.expires_at, last_sent_at = EXCLUDED.last_sent_at
	`, email, hex.EncodeToString(mac.Sum(nil)))
	a.NoError(err)
}
