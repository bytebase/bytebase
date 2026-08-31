package store

import (
	"context"
	"database/sql"
	"slices"
	"strings"
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

// ClaimLoginAttempt atomically takes one of maxAttempts slots for the identity
// before the credential is checked (docs/design/login-attempt-lockout.md).
// Returns (true, nil) when a slot was granted, or (false, nil) when the
// identity already holds maxAttempts and the latest was under window ago — the
// caller must refuse the attempt without touching the credential. A refused
// claim leaves the row untouched, so a lock lasts exactly window from the
// maxAttempts-th attempt; a claim more than window after the latest attempt
// restarts the counter at one. The row lock serializes concurrent claims and
// now() is database time, so replicas cannot disagree.
func (s *Store) ClaimLoginAttempt(ctx context.Context, identity string, kind storepb.LoginAttemptKind, maxAttempts int, window time.Duration) (bool, error) {
	if err := validateLoginAttemptKey(identity, kind); err != nil {
		return false, err
	}
	return claimLoginAttemptRow(ctx, s.GetDB(), identity, kind, maxAttempts, window)
}

// LoginAttemptBucket is one (identity, limit) pair of a multi-bucket claim.
type LoginAttemptBucket struct {
	Identity string
	Max      int
}

// ClaimLoginAttemptBuckets takes one slot in every bucket or none at all, so a
// limit composed of several buckets cannot be partially spent. A caller that
// claimed them one at a time would leave the earlier buckets consumed whenever a
// later one refused — turning every refusal into free consumption of the buckets
// ahead of it, which is the denial of service a multi-bucket limit exists to
// bound. Returns (false, nil) when any bucket is out of slots; the transaction
// rolls back, so no bucket is charged for the refusal.
//
// Buckets are claimed in identity order. kind is fixed for the call, so that is
// full primary-key order per backend/store/AGENTS.md, and two concurrent claims
// over overlapping buckets cannot form a wait-for cycle.
func (s *Store) ClaimLoginAttemptBuckets(ctx context.Context, kind storepb.LoginAttemptKind, window time.Duration, buckets []LoginAttemptBucket) (bool, error) {
	if len(buckets) == 0 {
		return true, nil
	}
	for _, bucket := range buckets {
		if err := validateLoginAttemptKey(bucket.Identity, kind); err != nil {
			return false, err
		}
	}
	ordered := slices.Clone(buckets)
	slices.SortFunc(ordered, func(a, b LoginAttemptBucket) int { return strings.Compare(a.Identity, b.Identity) })

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	for _, bucket := range ordered {
		granted, err := claimLoginAttemptRow(ctx, tx, bucket.Identity, kind, bucket.Max, window)
		if err != nil {
			return false, err
		}
		if !granted {
			return false, nil
		}
	}
	if err := tx.Commit(); err != nil {
		return false, errors.Wrap(err, "failed to commit transaction")
	}
	return true, nil
}

// validateLoginAttemptKey backstops against programmer error — request fields
// that become identities are bounded at the proto edge, so an unkeyed or
// structurally oversized row here means a caller composed an identity from an
// unbounded source.
func validateLoginAttemptKey(identity string, kind storepb.LoginAttemptKind) error {
	if identity == "" || len(identity) > maxIdentityBytes || kind == storepb.LoginAttemptKind_LOGIN_ATTEMPT_KIND_UNSPECIFIED {
		return errors.Errorf("login attempt requires a kind and an identity of at most %d bytes", maxIdentityBytes)
	}
	return nil
}

// rowQuerier is the subset of the database handle the claim statement needs, so
// one statement serves both the direct claim and the transactional batch.
type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func claimLoginAttemptRow(ctx context.Context, q rowQuerier, identity string, kind storepb.LoginAttemptKind, maxAttempts int, window time.Duration) (bool, error) {
	stmt := qb.Q().Space(`
		INSERT INTO login_attempt (identity, kind, attempts, last_attempt_at)
		VALUES (?, ?, 1, now())
		ON CONFLICT (identity, kind) DO UPDATE SET
			attempts = CASE
				WHEN login_attempt.last_attempt_at < now() - make_interval(secs => ?) THEN 1
				ELSE login_attempt.attempts + 1
			END,
			last_attempt_at = now()
		WHERE login_attempt.attempts < ?
			OR login_attempt.last_attempt_at < now() - make_interval(secs => ?)
		RETURNING 1
	`, identity, kind.String(), window.Seconds(), maxAttempts, window.Seconds())

	query, args, err := stmt.ToSQL()
	if err != nil {
		return false, err
	}
	var dummy int
	if err := q.QueryRowContext(ctx, query, args...).Scan(&dummy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // locked out — no slot granted
		}
		return false, errors.Wrap(err, "failed to claim login attempt")
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
