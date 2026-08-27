package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// Login-attempt lockout (docs/design/login-attempt-lockout.md): every
// guessable credential claims a per-identity slot before it is checked, and
// success forgets the counter. The store owns the atomic mechanism; this file
// owns the kinds, the shared limit, and the lockout answers.

const (
	errMsgTooManyPassword  = "too many failed login attempts, please try again later"
	errMsgTooManyMFA       = "too many failed MFA attempts, please try again later"
	errMsgTooManyEmailCode = "too many attempts, please try again later"
)

// Every credential kind shares one attempt limit
// (docs/design/login-attempt-lockout.md): loginAttemptMax slots per identity,
// and a claim more than loginAttemptWindow after the latest restarts the
// counter — so a lock lasts exactly loginAttemptWindow from the last slot.
const (
	loginAttemptMax    = 5
	loginAttemptWindow = 10 * time.Minute
)

// loginAttemptLockedMsg answers a refused claim, per credential kind.
var loginAttemptLockedMsg = map[storepb.LoginAttemptKind]string{
	storepb.LoginAttemptKind_PASSWORD:   errMsgTooManyPassword,
	storepb.LoginAttemptKind_EMAIL_CODE: errMsgTooManyEmailCode,
	storepb.LoginAttemptKind_MFA:        errMsgTooManyMFA,
}

// claimLoginAttempt claims one attempt slot for the identity under attack,
// before the credential is checked (docs/design/login-attempt-lockout.md).
// A locked identity gets ResourceExhausted without any credential comparison.
// The identity must be server-resolved and globally unique, never optional
// request context a caller can omit or forge; its size is bounded by the
// proto field limits, with the store refusing oversized or unkeyed rows.
func (s *AuthService) claimLoginAttempt(ctx context.Context, identity string, kind storepb.LoginAttemptKind) error {
	return claimLoginAttempt(ctx, s.store, identity, kind)
}

// claimLoginAttempt is package-level so UserService can claim the same buckets
// when it verifies a CredentialProof: a proof channel without that bound is a
// guessing oracle.
func claimLoginAttempt(ctx context.Context, stores *store.Store, identity string, kind storepb.LoginAttemptKind) error {
	granted, err := stores.ClaimLoginAttempt(ctx, identity, kind, loginAttemptMax, loginAttemptWindow)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if !granted {
		return connect.NewError(connect.CodeResourceExhausted, errors.New(loginAttemptLockedMsg[kind]))
	}
	return nil
}

// clearLoginAttempt forgets the identity's failed attempts after a successful
// verification. Success has already consumed a slot, so a swallowed failure
// here would hold a lock against a proven credential — retry once, then log;
// a still-standing counter expires on its own within the window.
func (s *AuthService) clearLoginAttempt(ctx context.Context, identity string, kind storepb.LoginAttemptKind) {
	clearLoginAttempt(ctx, s.store, identity, kind)
}

func clearLoginAttempt(ctx context.Context, stores *store.Store, identity string, kind storepb.LoginAttemptKind) {
	err := stores.ClearLoginAttempt(ctx, identity, kind)
	if err != nil {
		err = stores.ClearLoginAttempt(ctx, identity, kind)
	}
	if err != nil {
		slog.Error("login attempt clear failed", slog.String("kind", kind.String()), log.BBError(err))
	}
}

// challengeMFAAndClear verifies the OTP or recovery code (one is present —
// callers refuse codeless requests before claiming) and forgets the email's
// MFA failures on success.
func (s *AuthService) challengeMFAAndClear(ctx context.Context, user *store.UserMessage, email string, otpCode, recoveryCode *string) error {
	if otpCode != nil {
		if err := challengeMFACode(user, *otpCode); err != nil {
			return err
		}
	} else if err := s.challengeRecoveryCode(ctx, user, *recoveryCode); err != nil {
		return err
	}
	s.clearLoginAttempt(ctx, email, storepb.LoginAttemptKind_MFA)
	return nil
}

// ldapLoginIdentity keys the PASSWORD lockout bucket for an LDAP bind: the
// identity-provider ID joined with the submitted username, verbatim. The
// provider must be part of the key — keyed by bare username, an attacker who
// controls the same username in a directory of their own could clear this
// directory's counter by logging in there (success deletes the row). The
// username is not normalized: a case-exact directory attribute can name two
// accounts differing only by case, and merging them would let either lock —
// or, on success, clear — the other. One bucket per submitted form is the
// design's accepted trade. The separator is ":" because it is legal in
// neither IDP resource IDs ([a-z0-9-]) nor email addresses, so an LDAP
// bucket can never collide with a plain email account's bucket.
func ldapLoginIdentity(idpID, username string) string {
	return fmt.Sprintf("%s:%s", idpID, username)
}
