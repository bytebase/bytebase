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

type OAuth2AuthorizationCodeMessage struct {
	Code      string
	ClientID  string
	UserEmail string
	// Workspace is the workspace the user was acting in when they granted
	// consent. It is propagated into the issued access token's workspace_id
	// claim. Empty only for codes created before the workspace column
	// migration (3.18.2); handlers fall back to the client's workspace in
	// that case.
	Workspace string
	Config    *storepb.OAuth2AuthorizationCodeConfig
	ExpiresAt time.Time
}

func (s *Store) CreateOAuth2AuthorizationCode(ctx context.Context, create *OAuth2AuthorizationCodeMessage) (*OAuth2AuthorizationCodeMessage, error) {
	configBytes, err := protojson.Marshal(create.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
	}

	var workspaceArg any
	if create.Workspace != "" {
		workspaceArg = create.Workspace
	}

	q := qb.Q().Space(`
		INSERT INTO oauth2_authorization_code (code, client_id, user_email, workspace, config, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, create.Code, create.ClientID, create.UserEmail, workspaceArg, configBytes, create.ExpiresAt)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil { // NOSONAR: query is parameterized via qb.Query
		return nil, errors.Wrap(err, "failed to create OAuth2 authorization code")
	}
	return create, nil
}

func (s *Store) GetOAuth2AuthorizationCode(ctx context.Context, clientID, code string) (*OAuth2AuthorizationCodeMessage, error) {
	q := qb.Q().Space(`
		SELECT code, client_id, user_email, workspace, config, expires_at
		FROM oauth2_authorization_code
		WHERE code = ? AND client_id = ?
	`, code, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	msg := &OAuth2AuthorizationCodeMessage{}
	var workspace sql.NullString
	var configBytes []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan( // NOSONAR: query is parameterized via qb.Query
		&msg.Code, &msg.ClientID, &msg.UserEmail, &workspace, &configBytes, &msg.ExpiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get OAuth2 authorization code")
	}
	msg.Workspace = workspace.String

	msg.Config = &storepb.OAuth2AuthorizationCodeConfig{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(configBytes, msg.Config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
	}
	return msg, nil
}

func (s *Store) DeleteOAuth2AuthorizationCode(ctx context.Context, clientID, code string) error {
	q := qb.Q().Space(`
		DELETE FROM oauth2_authorization_code
		WHERE code = ? AND client_id = ?
	`, code, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil { // NOSONAR: query is parameterized via qb.Query
		return errors.Wrap(err, "failed to delete OAuth2 authorization code")
	}
	return nil
}

func (s *Store) DeleteExpiredOAuth2AuthorizationCodes(ctx context.Context) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM oauth2_authorization_code
		WHERE expires_at < NOW()
	`)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete expired OAuth2 authorization codes")
	}
	return result.RowsAffected()
}
