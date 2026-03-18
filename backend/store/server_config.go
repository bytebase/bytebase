package store

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/pkg/errors"
)

// GetServerConfig returns the global server configuration.
// Stored in the server_config table (single row, not workspace-scoped).
func (s *Store) GetServerConfig(ctx context.Context) (*storepb.ServerConfigPayload, error) {
	var payloadBytes []byte
	if err := s.GetDB().QueryRowContext(ctx,
		`SELECT payload FROM server_config LIMIT 1`,
	).Scan(&payloadBytes); err != nil {
		return nil, errors.Wrap(err, "failed to get server config")
	}
	config := &storepb.ServerConfigPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal server config")
	}
	return config, nil
}

// GetAuthSecret returns the global auth secret used for JWT signing.
func (s *Store) GetAuthSecret(ctx context.Context) (string, error) {
	config, err := s.GetServerConfig(ctx)
	if err != nil {
		return "", err
	}
	return config.AuthSecret, nil
}
