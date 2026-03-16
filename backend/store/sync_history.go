package store

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type SyncHistory struct {
	ResourceID   string
	InstanceID   string
	DatabaseName string
	Schema       string
	Metadata     *storepb.DatabaseSchemaMetadata

	CreatedAt time.Time
}

func (s *Store) GetSyncHistory(ctx context.Context, resourceID string) (*SyncHistory, error) {
	q := qb.Q().Space(`
		SELECT
			resource_id,
			created_at,
			instance,
			db_name,
			metadata,
			raw_dump
		FROM sync_history
		WHERE resource_id = ?
	`, resourceID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	h := SyncHistory{
		Metadata: &storepb.DatabaseSchemaMetadata{},
	}

	var m []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&h.ResourceID,
		&h.CreatedAt,
		&h.InstanceID,
		&h.DatabaseName,
		&m,
		&h.Schema,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}

	if err := common.ProtojsonUnmarshaler.Unmarshal(m, h.Metadata); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal")
	}

	// Sanitize invalid UTF-8 from historical rows (e.g. TiDB/OceanBase schema syncs).
	h.Schema = strings.ToValidUTF8(h.Schema, "")

	return &h, nil
}

func (s *Store) CreateSyncHistory(ctx context.Context, instanceID, databaseName string, metadata *storepb.DatabaseSchemaMetadata, schema string) (string, error) {
	// Sanitize schema to prevent storing invalid UTF-8 bytes from external databases.
	schema = strings.ToValidUTF8(schema, "")

	metadataBytes, err := protojson.Marshal(metadata)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal")
	}

	q := qb.Q().Space(`
		INSERT INTO sync_history (
			instance,
			db_name,
			metadata,
			raw_dump
		)
		VALUES (?, ?, ?, ?)
		RETURNING resource_id
	`, instanceID, databaseName, metadataBytes, schema)

	query, args, err := q.ToSQL()
	if err != nil {
		return "", errors.Wrapf(err, "failed to build sql")
	}

	var resourceID string
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&resourceID); err != nil {
		return "", errors.Wrapf(err, "failed to insert")
	}

	return resourceID, nil
}
