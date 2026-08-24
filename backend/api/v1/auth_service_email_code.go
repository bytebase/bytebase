package v1

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/mailer"
	"github.com/bytebase/bytebase/backend/store"
)

// Emailed verification codes: sending 6-digit codes for sign-in and password
// reset, and verifying them under the EMAIL_CODE lockout.

const (
	emailCodeLength         = 6
	emailCodeExpiry         = 10 * time.Minute
	emailCodeResendCooldown = 60 * time.Second

	errMsgInvalidEmailCode = "invalid or expired code"
)

// RequestPasswordReset sends a password reset email. Always returns success to avoid leaking email existence.
func (s *AuthService) RequestPasswordReset(ctx context.Context, req *connect.Request[v1pb.RequestPasswordResetRequest]) (*connect.Response[emptypb.Empty], error) {
	email := normalizeEmail(req.Msg.Email)
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}

	workspaceID, err := s.parseAndSetAuditWorkspace(ctx, email, req.Msg.Workspace)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Send synchronously, but swallow errors to avoid email enumeration — a fast silent
	// "success" for unknown emails must be indistinguishable from an SMTP failure for
	// known ones. Errors are logged server-side for operator visibility.
	if err := s.sendEmailVerificationCode(
		ctx,
		workspaceID,
		email,
		storepb.EmailVerificationCodePurpose_PASSWORD_RESET,
		"[Bytebase] Reset your password",
		"Hi,\n\nYour password reset code is: %s\n\nThis code expires in %d minutes. If you didn't request this, you can safely ignore this email.\n\n— Bytebase",
	); err != nil {
		slog.Warn("failed to send password reset email", slog.String("to", email), log.BBError(err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ResetPassword verifies the 6-digit code and updates the user's password.
// Also revokes all refresh tokens to force re-login.
func (s *AuthService) ResetPassword(ctx context.Context, req *connect.Request[v1pb.ResetPasswordRequest]) (*connect.Response[emptypb.Empty], error) {
	email := normalizeEmail(req.Msg.Email)
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}
	if req.Msg.Code == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("code is required"))
	}
	if req.Msg.NewPassword == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("new_password is required"))
	}

	if err := s.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_PASSWORD_RESET, req.Msg.Code); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user"))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user not found"))
	}

	// The code proved control of the email, so the user is known: the password
	// policy and the audit workspace come from the user's own memberships — the
	// singleton on self-hosted — never from anything the caller sent.
	workspaceID, err := s.resolveWorkspaceForLogin(ctx, user, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to resolve workspace"))
	}
	common.SetAuditWorkspaceID(ctx, workspaceID)
	restriction, err := getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, workspaceID)
	if err != nil {
		return nil, err
	}
	if err := validatePasswordWithRestriction(req.Msg.NewPassword, convertToStorePasswordRestriction(restriction.PasswordRestriction)); err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Msg.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to hash password"))
	}
	passwordHashStr := string(passwordHash)
	if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
		Email:        &user.Email,
		PasswordHash: &passwordHashStr,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update password"))
	}

	if err := s.store.DeleteWebRefreshTokensByUser(ctx, user.Email); err != nil {
		slog.Warn("failed to revoke refresh tokens after password reset", log.BBError(err))
	}

	// The user proved control of the email and holds a fresh password: a lock
	// they guessed themselves into must not outlive the reset.
	s.clearLoginAttempt(ctx, email, storepb.LoginAttemptKind_PASSWORD)

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// SendEmailLoginCode sends a 6-digit verification code. Always returns success
// (no email enumeration). Rate limit: 60-sec resend cooldown enforced atomically
// via the store — effective cap ≈ 60 sends/hour/email.
func (s *AuthService) SendEmailLoginCode(ctx context.Context, req *connect.Request[v1pb.SendEmailLoginCodeRequest]) (*connect.Response[emptypb.Empty], error) {
	email := normalizeEmail(req.Msg.Email)
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}
	workspaceID, err := s.parseAndSetAuditWorkspace(ctx, email, req.Msg.Workspace)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if workspaceID != "" {
		if err := validateEmailWithDomains(ctx, s.licenseService, s.store, workspaceID, email, false); err != nil {
			return nil, err
		}
	} else if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}

	// Gate on AllowEmailCodeSignin — no point emailing a code the workspace won't accept.
	// getAccountRestriction handles all cases (including empty workspace for brand-new SaaS
	// signup, where it resolves via EMAIL_CONFIG + SaaS override).
	restriction, err := getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, workspaceID)
	if err != nil {
		return nil, err
	}
	if !restriction.AllowEmailCodeSignin {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("email code login is not enabled for this workspace"))
	}

	// Send synchronously so the caller learns about actionable failures (missing EMAIL
	// setting, SMTP unreachable, etc.). No enumeration risk here: LOGIN always attempts to
	// send regardless of whether the email exists (sign-up is handled on verify).
	if err := s.sendEmailVerificationCode(
		ctx,
		workspaceID,
		email,
		storepb.EmailVerificationCodePurpose_LOGIN,
		"[Bytebase] Your sign-in code",
		"Hi,\n\nYour sign-in code is: %s\n\nThis code expires in %d minutes. If you didn't request this, you can safely ignore this email.\n\n— Bytebase",
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// authenticateEmailCodeLogin handles the email + 6-digit code flow.
// Existing users: verify code → return user (downstream pipeline handles workspace-level gates).
// Unknown emails: verify code → gate checks on pre-invited workspace → create user + provision workspace.
func (s *AuthService) authenticateEmailCodeLogin(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	if request.Password != "" || request.GetIdpName() != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email_code is mutually exclusive with password and idp_name"))
	}
	email := normalizeEmail(request.Email)
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}

	if err := s.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, *request.EmailCode); err != nil {
		return nil, err
	}

	// Existing user → return. Domain and allow_email_code_signin are checked later
	// in validateLoginPermissions against the actually-resolved login workspace
	// (which may not match the send-time workspace for multi-workspace users —
	// resolveWorkspaceForLogin prefers LastLoginWorkspace).
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user"))
	}
	if user != nil {
		return user, nil
	}

	// Gate checks run BEFORE user creation to prevent orphan accounts, against
	// the workspace the email would land in — the workspace whose invitation the
	// email holds, or the self-hosted singleton; provisioning below reuses this
	// resolution. A brand-new SaaS signup has neither and gets the SaaS defaults.
	targetWorkspaceID, targetIsMember, err := s.resolveWorkspaceIDByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to resolve target workspace"))
	}
	if targetWorkspaceID != "" {
		if err := validateEmailWithDomains(ctx, s.licenseService, s.store, targetWorkspaceID, email, false); err != nil {
			return nil, err
		}
	} else if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}

	// We only consult AllowEmailCodeSignin here: DisallowSignup governs password self-service
	// signup (the Signup RPC), not email-code onboarding — the two paths are independent.
	// Admins who want to block new users via email-code set AllowEmailCodeSignin=false.
	restriction, err := getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, targetWorkspaceID)
	if err != nil {
		return nil, err
	}
	if !restriction.AllowEmailCodeSignin {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("email code login is not enabled for this workspace"))
	}
	// Signup is always allowed for SaaS.
	if !s.profile.SaaS && restriction.DisallowSignup {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("sign up is disallowed for this workspace"))
	}

	// Provision workspace BEFORE creating the user so retries are self-healing: if user
	// creation fails, the next attempt's FindWorkspace(email) finds the already-provisioned
	// workspace via its IAM binding and returns it. The reverse order would leave a user
	// without a workspace, and subsequent retries would early-return via GetUserByEmail and
	// never re-run provisioning — permanently stuck. Matches the Signup RPC's ordering.
	if _, err := s.provisionResolvedWorkspace(ctx, email, targetWorkspaceID, targetIsMember); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to provision workspace"))
	}

	// Create principal with random bcrypt password.
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate random password"))
	}
	passwordHash, err := bcrypt.GenerateFromPassword(randomBytes, bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to hash password"))
	}

	// Derive display name from email local-part.
	name := email
	if i := strings.Index(email, "@"); i > 0 {
		name = email[:i]
	}

	newUser, err := s.store.CreateUser(ctx, &store.UserMessage{
		Email:        email,
		Name:         name,
		Type:         storepb.PrincipalType_END_USER,
		PasswordHash: string(passwordHash),
		Profile:      &storepb.UserProfile{},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create user"))
	}

	return newUser, nil
}

// parseAndSetAuditWorkspace parses the requested workspace and attributes the
// audit event only when the email belongs to an active user in that workspace.
// Membership affects audit attribution only; the parsed workspace ID is still
// returned for email-setting resolution and verification-code storage.
func (s *AuthService) parseAndSetAuditWorkspace(ctx context.Context, email string, workspaceName *string) (string, error) {
	workspaceID, err := parseOptionalWorkspace(workspaceName)
	if err != nil {
		return "", err
	}
	if workspaceID != "" {
		account, err := s.store.GetAccountByEmail(ctx, email)
		if err == nil && account != nil && account.Type == storepb.PrincipalType_END_USER && !account.MemberDeleted {
			workspace, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
				WorkspaceID:    &workspaceID,
				Email:          email,
				IncludeAllUser: !s.profile.SaaS,
			})
			if err == nil && workspace != nil {
				common.SetAuditWorkspaceID(ctx, workspace.ResourceID)
			}
		}
	}
	return workspaceID, nil
}

// resolvePreLoginEmailSetting returns the EMAIL setting to use for unauthenticated flows
// (email-code sign-in, password reset). Resolution order:
//  1. If `workspaceID` is provided, use that workspace's EMAIL setting. The frontend
//     resolves the workspace (from the route query or actuator context) and always passes
//     it when one exists — self-host, multi-workspace SaaS, and pre-invited emails all
//     flow through this path.
//  2. EMAIL_CONFIG env var — deployment-wide fallback for SaaS brand-new signup, where
//     the caller has no workspace context yet (no pre-invite, no workspace in the URL).
func resolvePreLoginEmailSetting(
	ctx context.Context,
	stores *store.Store,
	workspaceID string,
) (*storepb.EmailSetting, error) {
	if workspaceID != "" {
		emailSettingMsg, err := stores.GetSetting(ctx, workspaceID, storepb.SettingName_EMAIL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load email setting")
		}
		if emailSettingMsg == nil {
			return nil, nil
		}
		es, ok := emailSettingMsg.Value.(*storepb.EmailSetting)
		if !ok {
			return nil, nil
		}
		return es, nil
	}

	if raw := os.Getenv("EMAIL_CONFIG"); raw != "" {
		emailSetting := &storepb.EmailSetting{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(raw), emailSetting); err != nil {
			return nil, errors.Wrap(err, "failed to parse EMAIL_CONFIG")
		}
		return emailSetting, nil
	}

	return nil, nil
}

// sendEmailVerificationCode generates a code, atomically stores its hash (subject to cooldown),
// and emails the plain code. Returns nil on a successful send as well as on silent-skip paths
// (cooldown active, or PASSWORD_RESET for an unknown email) — both are intentionally
// indistinguishable to the caller, since both correspond to "no new email was delivered".
// Returns an error only on actionable failures (missing EMAIL setting, SMTP failure, DB error).
// Callers decide whether to propagate the error: `SendEmailLoginCode` surfaces it (users need
// to know email delivery failed), `RequestPasswordReset` swallows it to avoid revealing that
// the account exists. `bodyFmt` must contain one %s for the 6-digit code.
func (s *AuthService) sendEmailVerificationCode(ctx context.Context, workspaceID, email string, purpose storepb.EmailVerificationCodePurpose, subject, bodyFmt string) error {
	// For password reset, only send to active end users — no upsert or email for other targets.
	// Login intentionally skips this account check because email-code login also supports signup.
	if purpose == storepb.EmailVerificationCodePurpose_PASSWORD_RESET {
		account, err := s.store.GetAccountByEmail(ctx, email)
		if err != nil {
			return errors.Wrap(err, "failed to look up account for password reset")
		}
		if account == nil || account.Type != storepb.PrincipalType_END_USER || account.MemberDeleted {
			return nil // silent: account doesn't exist
		}
	}

	// Resolve the EMAIL setting FIRST — fail fast if misconfigured so we don't write a
	// verification row we can't actually deliver.
	emailSetting, err := resolvePreLoginEmailSetting(ctx, s.store, workspaceID)
	if err != nil {
		return err
	}
	if emailSetting == nil {
		return errors.Errorf("cannot found email config for workspace %v", workspaceID)
	}

	code, err := generateEmailCode()
	if err != nil {
		return errors.Wrap(err, "failed to generate code")
	}

	now := time.Now()
	sent, err := s.store.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
		Email:      email,
		Purpose:    purpose,
		CodeHash:   s.hashEmailCode(code),
		ExpiresAt:  now.Add(emailCodeExpiry),
		LastSentAt: now,
	}, emailCodeResendCooldown)
	if err != nil {
		return errors.Wrap(err, "failed to upsert verification code")
	}
	if !sent {
		return nil // cooldown active — silent skip
	}

	sender, err := mailer.NewSender(emailSetting)
	if err != nil {
		return errors.Wrap(err, "failed to create mail sender")
	}

	body := fmt.Sprintf(bodyFmt, code, int(emailCodeExpiry.Minutes()))
	if err := sender.Send(ctx, &mailer.SendRequest{
		To:       []string{email},
		Subject:  subject,
		TextBody: body,
	}); err != nil {
		// Delete the row so the cooldown doesn't block an immediate retry.
		// Match on code_hash to avoid wiping a newer code from a concurrent request.
		_ = s.store.DeleteEmailVerificationCodeIfMatch(ctx, email, purpose, s.hashEmailCode(code))
		return errors.Wrap(err, "failed to send email")
	}
	return nil
}

// verifyEmailCode checks a submitted code against the stored row for the email.
// An EMAIL_CODE slot is claimed before the row is even loaded, so guessing is
// bounded per identity across codes and purposes, and a lock is not an oracle
// for whether a code is pending. Wrong guesses leave the row in place — the
// resend cooldown always has a row to evaluate.
func (s *AuthService) verifyEmailCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose, submittedCode string) error {
	// An invalid-syntax email can hold neither an account nor a code; reject it
	// before the claim so garbage never writes a row.
	if !common.IsValidEmail(email) {
		return connect.NewError(connect.CodeUnauthenticated, errors.New(errMsgInvalidEmailCode))
	}
	if err := s.claimLoginAttempt(ctx, email, storepb.LoginAttemptKind_EMAIL_CODE); err != nil {
		return err
	}
	row, err := s.store.GetEmailVerificationCode(ctx, email, purpose)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get email verification code"))
	}
	if row == nil {
		return connect.NewError(connect.CodeUnauthenticated, errors.New(errMsgInvalidEmailCode))
	}
	if time.Now().After(row.ExpiresAt) {
		_ = s.store.DeleteEmailVerificationCodeIfMatch(ctx, email, purpose, row.CodeHash)
		return connect.NewError(connect.CodeUnauthenticated, errors.New(errMsgInvalidEmailCode))
	}
	if subtle.ConstantTimeCompare([]byte(s.hashEmailCode(submittedCode)), []byte(row.CodeHash)) != 1 {
		return connect.NewError(connect.CodeUnauthenticated, errors.New(errMsgInvalidEmailCode))
	}
	_ = s.store.DeleteEmailVerificationCodeIfMatch(ctx, email, purpose, row.CodeHash)
	s.clearLoginAttempt(ctx, email, storepb.LoginAttemptKind_EMAIL_CODE)
	return nil
}

// generateEmailCode returns a cryptographically-random 6-digit numeric code.
func generateEmailCode() (string, error) {
	const digits = "0123456789"
	b := make([]byte, emailCodeLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b), nil
}

// hashEmailCode returns HMAC-SHA256(code) hex-encoded, keyed with the server's auth secret.
// HMAC with a server-side secret (vs. bare SHA-256) prevents offline brute force of the
// 10^6-size code space if the DB is ever compromised — the attacker would also need the
// auth secret to verify candidate codes.
func (s *AuthService) hashEmailCode(code string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}
