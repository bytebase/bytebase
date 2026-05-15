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

type OAuth2ClientMessage struct {
	ClientID string
	// Workspace is empty for clients registered via unauthenticated DCR
	// (the workspace is bound to the issued authorization code / refresh
	// token at consent time instead).
	Workspace        string
	ClientSecretHash string
	Config           *storepb.OAuth2ClientConfig
	LastActiveAt     time.Time
}

func (s *Store) CreateOAuth2Client(ctx context.Context, create *OAuth2ClientMessage) (*OAuth2ClientMessage, error) {
	configBytes, err := protojson.Marshal(create.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
	}

	var workspaceArg any
	if create.Workspace != "" {
		workspaceArg = create.Workspace
	}

	q := qb.Q().Space(`
		INSERT INTO oauth2_client (client_id, workspace, client_secret_hash, config, last_active_at)
		VALUES (?, ?, ?, ?, NOW())
		RETURNING last_active_at
	`, create.ClientID, workspaceArg, create.ClientSecretHash, configBytes)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.LastActiveAt); err != nil {
		return nil, errors.Wrap(err, "failed to create OAuth2 client")
	}
	return create, nil
}

// GetOAuth2Client looks up a client by client_id (the table's primary key).
// Workspace is no longer a lookup key — clients are workspace-agnostic since
// DCR runs unauthenticated.
func (s *Store) GetOAuth2Client(ctx context.Context, clientID string) (*OAuth2ClientMessage, error) {
	q := qb.Q().Space(`
		SELECT client_id, workspace, client_secret_hash, config, last_active_at
		FROM oauth2_client
		WHERE client_id = ?
	`, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	client := &OAuth2ClientMessage{}
	var workspace sql.NullString
	var configBytes []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan( // NOSONAR: query is parameterized via qb.Query
		&client.ClientID, &workspace, &client.ClientSecretHash, &configBytes, &client.LastActiveAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to query OAuth2 client")
	}
	client.Workspace = workspace.String
	client.Config = &storepb.OAuth2ClientConfig{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(configBytes, client.Config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
	}
	return client, nil
}

func (s *Store) UpdateOAuth2ClientLastActiveAt(ctx context.Context, clientID string) error {
	q := qb.Q().Space(`
		UPDATE oauth2_client
		SET last_active_at = NOW()
		WHERE client_id = ?
	`, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil { // NOSONAR: query is parameterized via qb.Query
		return errors.Wrap(err, "failed to update OAuth2 client last active at")
	}
	return nil
}

func (s *Store) DeleteOAuth2Client(ctx context.Context, clientID string) error {
	q := qb.Q().Space(`
		DELETE FROM oauth2_client
		WHERE client_id = ?
	`, clientID)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete OAuth2 client")
	}
	return nil
}

func (s *Store) DeleteExpiredOAuth2Clients(ctx context.Context, expireBefore time.Time) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM oauth2_client
		WHERE last_active_at < ?
	`, expireBefore)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete expired OAuth2 clients")
	}
	return result.RowsAffected()
}
