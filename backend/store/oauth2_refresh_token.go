package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type OAuth2RefreshTokenMessage struct {
	TokenHash string
	ClientID  string
	UserEmail string
	// Workspace is the workspace the original consent was granted for.
	// Preserved across refresh so re-issued access tokens carry the same
	// workspace_id claim. Empty only for refresh tokens created before the
	// 3.18.2 migration.
	Workspace string
	// Config is the consented grant state (resource, scope) inherited from the
	// authorization code. Carried forward unchanged by every refresh — a refresh
	// never widens a grant. Empty for tokens created before the 3.22.1 migration.
	Config    *storepb.OAuth2RefreshTokenConfig
	ExpiresAt time.Time
}

func (s *Store) CreateOAuth2RefreshToken(ctx context.Context, create *OAuth2RefreshTokenMessage) (*OAuth2RefreshTokenMessage, error) {
	configBytes, err := protojson.Marshal(create.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
	}

	var workspaceArg any
	if create.Workspace != "" {
		workspaceArg = create.Workspace
	}

	q := qb.Q().Space(`
		INSERT INTO oauth2_refresh_token (token_hash, client_id, user_email, workspace, config, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, create.TokenHash, create.ClientID, create.UserEmail, workspaceArg, configBytes, create.ExpiresAt)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrap(err, "failed to create OAuth2 refresh token")
	}
	return create, nil
}

func (s *Store) GetOAuth2RefreshToken(ctx context.Context, clientID, tokenHash string) (*OAuth2RefreshTokenMessage, error) {
	q := qb.Q().Space(`
		SELECT token_hash, client_id, user_email, workspace, config, expires_at
		FROM oauth2_refresh_token
		WHERE token_hash = ? AND client_id = ?
	`, tokenHash, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	msg := &OAuth2RefreshTokenMessage{}
	var workspace sql.NullString
	var configBytes []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&msg.TokenHash, &msg.ClientID, &msg.UserEmail, &workspace, &configBytes, &msg.ExpiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get OAuth2 refresh token")
	}
	msg.Workspace = workspace.String

	msg.Config = &storepb.OAuth2RefreshTokenConfig{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(configBytes, msg.Config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
	}
	return msg, nil
}

// ConsumeOAuth2RefreshToken atomically deletes a refresh token and reports
// whether this call is the one that claimed it. The DELETE ... RETURNING is the
// single-use rotation gate: concurrent refreshes of the same token race here and
// exactly one observes consumed=true, so the caller must issue a new token pair
// only when consumed is true. RETURNING (not RowsAffected) is used deliberately
// — Postgres row counts are unreliable for this check. A false return means the
// token was already consumed or never existed; a non-nil error is a real failure
// and must abort issuance.
func (s *Store) ConsumeOAuth2RefreshToken(ctx context.Context, clientID, tokenHash string) (bool, error) {
	q := qb.Q().Space(`
		DELETE FROM oauth2_refresh_token
		WHERE token_hash = ? AND client_id = ?
		RETURNING token_hash
	`, tokenHash, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return false, err
	}

	var consumedHash string
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&consumedHash); err != nil { // NOSONAR: query is parameterized via qb.Query
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to consume OAuth2 refresh token")
	}
	return true, nil
}

func (s *Store) DeleteOAuth2RefreshToken(ctx context.Context, clientID, tokenHash string) error {
	q := qb.Q().Space(`
		DELETE FROM oauth2_refresh_token
		WHERE token_hash = ? AND client_id = ?
	`, tokenHash, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete OAuth2 refresh token")
	}
	return nil
}

func (s *Store) DeleteOAuth2RefreshTokensByUserAndClient(ctx context.Context, userEmail, clientID string) error {
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
