package store

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type OAuth2ClientMessage struct {
	ClientID         string
	ClientSecretHash string
	Config           *storepb.OAuth2ClientConfig
	LastActiveAt     time.Time
}

type FindOAuth2ClientMessage struct {
	ClientID *string
}

func (s *Store) CreateOAuth2Client(ctx context.Context, create *OAuth2ClientMessage) (*OAuth2ClientMessage, error) {
	configBytes, err := protojson.Marshal(create.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
	}

	q := qb.Q().Space(`
		INSERT INTO oauth2_client (client_id, client_secret_hash, config, last_active_at)
		VALUES (?, ?, ?, NOW())
		RETURNING last_active_at
	`, create.ClientID, create.ClientSecretHash, configBytes)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.LastActiveAt); err != nil {
		return nil, errors.Wrap(err, "failed to create OAuth2 client")
	}
	return create, nil
}

func (s *Store) GetOAuth2Client(ctx context.Context, clientID string) (*OAuth2ClientMessage, error) {
	clients, err := s.ListOAuth2Clients(ctx, &FindOAuth2ClientMessage{ClientID: &clientID})
	if err != nil {
		return nil, err
	}
	if len(clients) == 0 {
		return nil, nil
	}
	return clients[0], nil
}

func (s *Store) ListOAuth2Clients(ctx context.Context, find *FindOAuth2ClientMessage) ([]*OAuth2ClientMessage, error) {
	q := qb.Q().Space(`
		SELECT client_id, client_secret_hash, config, last_active_at
		FROM oauth2_client
		WHERE TRUE
	`)

	if v := find.ClientID; v != nil {
		q.And("client_id = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query OAuth2 clients")
	}
	defer rows.Close()

	var clients []*OAuth2ClientMessage
	for rows.Next() {
		client := &OAuth2ClientMessage{}
		var configBytes []byte
		if err := rows.Scan(&client.ClientID, &client.ClientSecretHash, &configBytes, &client.LastActiveAt); err != nil {
			return nil, errors.Wrap(err, "failed to scan OAuth2 client")
		}
		client.Config = &storepb.OAuth2ClientConfig{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(configBytes, client.Config); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal config")
		}
		clients = append(clients, client)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate OAuth2 clients")
	}
	return clients, nil
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

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
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
