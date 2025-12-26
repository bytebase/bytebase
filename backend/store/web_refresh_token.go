package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
)

type WebRefreshTokenMessage struct {
	TokenHash string
	UserEmail string
	ExpiresAt time.Time
	RotatedAt *time.Time
}

func (s *Store) CreateWebRefreshToken(ctx context.Context, create *WebRefreshTokenMessage) error {
	q := qb.Q().Space(`
		INSERT INTO web_refresh_token (token_hash, user_email, expires_at)
		VALUES (?, ?, ?)
	`, create.TokenHash, create.UserEmail, create.ExpiresAt)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to create web refresh token")
	}
	return nil
}

func (s *Store) GetWebRefreshToken(ctx context.Context, tokenHash string) (*WebRefreshTokenMessage, error) {
	q := qb.Q().Space(`
		SELECT token_hash, user_email, expires_at, rotated_at
		FROM web_refresh_token
		WHERE token_hash = ?
	`, tokenHash)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	msg := &WebRefreshTokenMessage{}
	var rotatedAt sql.NullTime
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&msg.TokenHash, &msg.UserEmail, &msg.ExpiresAt, &rotatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get web refresh token")
	}
	if rotatedAt.Valid {
		msg.RotatedAt = &rotatedAt.Time
	}
	return msg, nil
}

// MarkWebRefreshTokenRotated atomically marks a token as rotated.
// Returns true if the token was rotated (rotated_at was null), false if already rotated.
// This prevents race conditions when multiple requests try to rotate the same token.
func (s *Store) MarkWebRefreshTokenRotated(ctx context.Context, tokenHash string) (bool, error) {
	q := qb.Q().Space(`
		UPDATE web_refresh_token
		SET rotated_at = NOW()
		WHERE token_hash = ? AND rotated_at IS NULL
	`, tokenHash)

	query, args, err := q.ToSQL()
	if err != nil {
		return false, err
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return false, errors.Wrap(err, "failed to mark web refresh token as rotated")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to get rows affected")
	}

	return rowsAffected > 0, nil
}

func (s *Store) DeleteWebRefreshToken(ctx context.Context, tokenHash string) error {
	q := qb.Q().Space(`
		DELETE FROM web_refresh_token
		WHERE token_hash = ?
	`, tokenHash)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete web refresh token")
	}
	return nil
}

func (s *Store) DeleteWebRefreshTokensByUser(ctx context.Context, userEmail string) error {
	q := qb.Q().Space(`
		DELETE FROM web_refresh_token
		WHERE user_email = ?
	`, userEmail)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete web refresh tokens for user")
	}
	return nil
}

func (s *Store) DeleteExpiredWebRefreshTokens(ctx context.Context) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM web_refresh_token
		WHERE expires_at < NOW()
	`)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete expired web refresh tokens")
	}
	return result.RowsAffected()
}

func (s *Store) DeleteRotatedWebRefreshTokens(ctx context.Context, gracePeriod time.Duration) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM web_refresh_token
		WHERE rotated_at IS NOT NULL AND rotated_at < ?
	`, time.Now().Add(-gracePeriod))

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete rotated web refresh tokens")
	}
	return result.RowsAffected()
}
