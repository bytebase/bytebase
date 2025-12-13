package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
)

type OAuth2RefreshTokenMessage struct {
	TokenHash string
	ClientID  string
	UserEmail string
	ExpiresAt time.Time
}

func (s *Store) CreateOAuth2RefreshToken(ctx context.Context, create *OAuth2RefreshTokenMessage) (*OAuth2RefreshTokenMessage, error) {
	q := qb.Q().Space(`
		INSERT INTO oauth2_refresh_token (token_hash, client_id, user_email, expires_at)
		VALUES (?, ?, ?, ?)
	`, create.TokenHash, create.ClientID, create.UserEmail, create.ExpiresAt)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrap(err, "failed to create OAuth2 refresh token")
	}
	return create, nil
}

func (s *Store) GetOAuth2RefreshToken(ctx context.Context, tokenHash string) (*OAuth2RefreshTokenMessage, error) {
	q := qb.Q().Space(`
		SELECT token_hash, client_id, user_email, expires_at
		FROM oauth2_refresh_token
		WHERE token_hash = ?
	`, tokenHash)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	msg := &OAuth2RefreshTokenMessage{}
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&msg.TokenHash, &msg.ClientID, &msg.UserEmail, &msg.ExpiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get OAuth2 refresh token")
	}
	return msg, nil
}

func (s *Store) DeleteOAuth2RefreshToken(ctx context.Context, tokenHash string) error {
	q := qb.Q().Space(`
		DELETE FROM oauth2_refresh_token
		WHERE token_hash = ?
	`, tokenHash)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete OAuth2 refresh token")
	}
	return nil
}

func (s *Store) DeleteOAuth2RefreshTokensByUserAndClient(ctx context.Context, userEmail string, clientID string) error {
	q := qb.Q().Space(`
		DELETE FROM oauth2_refresh_token
		WHERE user_email = ? AND client_id = ?
	`, userEmail, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete OAuth2 refresh tokens")
	}
	return nil
}

func (s *Store) DeleteExpiredOAuth2RefreshTokens(ctx context.Context) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM oauth2_refresh_token
		WHERE expires_at < NOW()
	`)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete expired OAuth2 refresh tokens")
	}
	return result.RowsAffected()
}
