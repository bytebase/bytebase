package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// EmailVerificationCodeMessage represents a row in the email_verification_code table.
type EmailVerificationCodeMessage struct {
	Email      string
	Purpose    storepb.EmailVerificationCodePurpose
	CodeHash   string
	Attempts   int
	ExpiresAt  time.Time
	LastSentAt time.Time
	// Workspace is the workspace context captured when the code was sent.
	// Empty string means no workspace (SaaS brand-new signup).
	Workspace string
}

// UpsertEmailVerificationCodeIfCooldownExpired inserts or updates the row for (email, purpose),
// but ONLY if no row exists OR the existing row's last_sent_at is older than the cooldown.
// Returns (true, nil) if the upsert happened (caller may proceed to send), or
// (false, nil) if the cooldown is still active (caller must skip the send).
//
// Atomic check-and-set: uses INSERT ... ON CONFLICT DO UPDATE ... WHERE ... RETURNING 1
// to ensure only one concurrent request proceeds when both see an expired cooldown.
// RETURNING is used (not RowsAffected) because Postgres returns unreliable counts
// when the DO UPDATE's WHERE filters out the update.
func (s *Store) UpsertEmailVerificationCodeIfCooldownExpired(ctx context.Context, msg *EmailVerificationCodeMessage, cooldown time.Duration) (bool, error) {
	cooldownSeconds := int64(cooldown.Seconds())
	var workspace sql.NullString
	if msg.Workspace != "" {
		workspace = sql.NullString{String: msg.Workspace, Valid: true}
	}
	q := qb.Q().Space(`
		INSERT INTO email_verification_code (email, purpose, code_hash, attempts, expires_at, last_sent_at, workspace)
		VALUES (?, ?, ?, 0, ?, ?, ?)
		ON CONFLICT (email, purpose) DO UPDATE SET
			code_hash = EXCLUDED.code_hash,
			attempts = 0,
			expires_at = EXCLUDED.expires_at,
			last_sent_at = EXCLUDED.last_sent_at,
			workspace = EXCLUDED.workspace
		WHERE email_verification_code.last_sent_at < EXCLUDED.last_sent_at - (?::bigint * interval '1 second')
		RETURNING 1
	`, msg.Email, msg.Purpose.String(), msg.CodeHash, msg.ExpiresAt, msg.LastSentAt, workspace, cooldownSeconds)

	query, args, err := q.ToSQL()
	if err != nil {
		return false, err
	}
	var dummy int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&dummy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // cooldown blocked the upsert
		}
		return false, errors.Wrap(err, "failed to upsert email verification code")
	}
	return true, nil
}

// GetEmailVerificationCode returns the verification code record for the given email and purpose,
// or (nil, nil) if no row exists.
func (s *Store) GetEmailVerificationCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose) (*EmailVerificationCodeMessage, error) {
	q := qb.Q().Space(`
		SELECT email, purpose, code_hash, attempts, expires_at, last_sent_at, workspace
		FROM email_verification_code
		WHERE email = ? AND purpose = ?
	`, email, purpose.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	msg := &EmailVerificationCodeMessage{}
	var purposeStr string
	var workspace sql.NullString
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&msg.Email, &purposeStr, &msg.CodeHash, &msg.Attempts, &msg.ExpiresAt, &msg.LastSentAt, &workspace,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get email verification code")
	}
	msg.Purpose = storepb.EmailVerificationCodePurpose(storepb.EmailVerificationCodePurpose_value[purposeStr])
	if workspace.Valid {
		msg.Workspace = workspace.String
	}
	return msg, nil
}

// IncrementEmailVerificationCodeAttempts increments the attempt counter for the given email and purpose.
func (s *Store) IncrementEmailVerificationCodeAttempts(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose) error {
	q := qb.Q().Space(`
		UPDATE email_verification_code
		SET attempts = attempts + 1
		WHERE email = ? AND purpose = ?
	`, email, purpose.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to increment email verification code attempts")
	}
	return nil
}

// DeleteEmailVerificationCode deletes the verification code record for the given email and purpose.
func (s *Store) DeleteEmailVerificationCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose) error {
	q := qb.Q().Space(`
		DELETE FROM email_verification_code
		WHERE email = ? AND purpose = ?
	`, email, purpose.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete email verification code")
	}
	return nil
}

// DeleteExpiredEmailVerificationCodes deletes all verification code records that have expired.
// Returns the number of deleted rows.
func (s *Store) DeleteExpiredEmailVerificationCodes(ctx context.Context) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM email_verification_code
		WHERE expires_at < NOW()
	`)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}
	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete expired email verification codes")
	}
	return result.RowsAffected()
}
