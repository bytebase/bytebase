package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// maxIdentityBytes caps the identity column. The longest legal identity today
// is an LDAP key: a 63-byte provider ID, ":", and a 254-character
// proto-bounded username, which is up to 254*4 = 1016 bytes of UTF-8 —
// comfortably under this structural bound.
const maxIdentityBytes = 2048

// ClaimSlot atomically takes one of slots claims for (identity, kind) inside
// window, before the thing being limited happens. Returns (false, nil) when the
// slots are spent, leaving the row untouched. The row lock serializes concurrent
// claims and now() is database time, so replicas cannot disagree.
//
// The kind decides what a grant does to the window, as proto/store/store/auth.proto
// records. A credential kind is a lockout: every failure restarts the window, so
// a lock outlives the last attempt. EMAIL_CODE_SEND counts outbound mail, so its
// window is anchored to the first claim — restarting it would mean a steady
// stream never resets and "N per window" would become "N ever".
func (s *Store) ClaimSlot(ctx context.Context, identity string, kind storepb.LoginAttemptKind, slots int, window time.Duration) (bool, error) {
	// Backstop against programmer error — request fields that become identities
	// are bounded at the proto edge, so an unkeyed or structurally oversized row
	// here means a caller composed an identity from an unbounded source.
	if identity == "" || len(identity) > maxIdentityBytes || kind == storepb.LoginAttemptKind_LOGIN_ATTEMPT_KIND_UNSPECIFIED {
		return false, errors.Errorf("a claim requires a kind and an identity of at most %d bytes", maxIdentityBytes)
	}
	restartWindowOnClaim := kind != storepb.LoginAttemptKind_EMAIL_CODE_SEND
	q := qb.Q().Space(`
		INSERT INTO login_attempt (identity, kind, attempts, last_attempt_at)
		VALUES (?, ?, 1, now())
		ON CONFLICT (identity, kind) DO UPDATE SET
			attempts = CASE
				WHEN login_attempt.last_attempt_at < now() - make_interval(secs => ?) THEN 1
				ELSE login_attempt.attempts + 1
			END,
			last_attempt_at = CASE
				WHEN ? OR login_attempt.last_attempt_at < now() - make_interval(secs => ?) THEN now()
				ELSE login_attempt.last_attempt_at
			END
		WHERE login_attempt.attempts < ?
			OR login_attempt.last_attempt_at < now() - make_interval(secs => ?)
		RETURNING 1
	`, identity, kind.String(), window.Seconds(), restartWindowOnClaim, window.Seconds(), slots, window.Seconds())

	query, args, err := q.ToSQL()
	if err != nil {
		return false, err
	}
	var dummy int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&dummy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // no slot granted
		}
		return false, errors.Wrap(err, "failed to claim slot")
	}
	return true, nil
}

// ClearLoginAttempt forgets the identity's failed attempts after a successful
// verification by deleting the row. Clearing an unknown identity is a no-op.
func (s *Store) ClearLoginAttempt(ctx context.Context, identity string, kind storepb.LoginAttemptKind) error {
	q := qb.Q().Space(`
		DELETE FROM login_attempt WHERE identity = ? AND kind = ?
	`, identity, kind.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to clear login attempts")
	}
	return nil
}

// DeleteStaleLoginAttempts deletes rows whose latest attempt is older than
// retention. Returns the number of deleted rows.
func (s *Store) DeleteStaleLoginAttempts(ctx context.Context, retention time.Duration) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM login_attempt WHERE last_attempt_at < now() - make_interval(secs => ?)
	`, retention.Seconds())

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}
	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete stale login attempts")
	}
	return result.RowsAffected()
}
