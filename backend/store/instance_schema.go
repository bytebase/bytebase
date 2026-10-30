package store

import (
	"context"
	"database/sql"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
)

// GetInstanceSchema returns the schema that no database of the instance owns,
// such as Oracle's PUBLIC, or nil when no sync has stored one.
func (s *Store) GetInstanceSchema(ctx context.Context, instanceID string) (*metadatapb.SchemaMetadata, error) {
	var metadata []byte
	if err := s.GetDB().QueryRowContext(ctx, `
		SELECT metadata FROM instance_schema WHERE instance = $1
	`, instanceID).Scan(&metadata); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get the instance schema of instance %s", instanceID)
	}
	schema := &metadatapb.SchemaMetadata{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, schema); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal the instance schema of instance %s", instanceID)
	}
	return schema, nil
}

// UpsertInstanceSchema stores the schema that no database of the instance owns.
// Every database sync of the instance calls it, so an unchanged schema is not
// written again.
func (s *Store) UpsertInstanceSchema(ctx context.Context, instanceID string, schema *metadatapb.SchemaMetadata) error {
	metadata, err := protojson.Marshal(schema)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal the instance schema of instance %s", instanceID)
	}
	if _, err := s.GetDB().ExecContext(ctx, `
		INSERT INTO instance_schema (instance, metadata)
		VALUES ($1, $2)
		ON CONFLICT (instance) DO UPDATE SET metadata = EXCLUDED.metadata
		WHERE instance_schema.metadata IS DISTINCT FROM EXCLUDED.metadata
	`, instanceID, metadata); err != nil {
		return errors.Wrapf(err, "failed to upsert the instance schema of instance %s", instanceID)
	}
	return nil
}
