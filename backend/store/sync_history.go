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

type SyncHistory struct {
	UID          int64
	InstanceID   string
	DatabaseName string
	Schema       string
	Metadata     *storepb.DatabaseSchemaMetadata

	CreatedAt time.Time
}

func (s *Store) GetSyncHistoryByUID(ctx context.Context, uid int64) (*SyncHistory, error) {
	q := qb.Q().Space(`
		SELECT
			id,
			created_at,
			instance,
			db_name,
			metadata,
			raw_dump
		FROM sync_history
		WHERE id = ?
	`, uid)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	h := SyncHistory{
		Metadata: &storepb.DatabaseSchemaMetadata{},
	}

	var m []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&h.UID,
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

	return &h, nil
}

// UpsertDBSchema upserts a database schema.
func (s *Store) CreateSyncHistory(ctx context.Context, instanceID, databaseName string, metadata *storepb.DatabaseSchemaMetadata, schema string) (int64, error) {
	metadataBytes, err := protojson.Marshal(metadata)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to marshal")
	}

	q := qb.Q().Space(`
		INSERT INTO sync_history (
			instance,
			db_name,
			metadata,
			raw_dump
		)
		VALUES (?, ?, ?, ?)
		RETURNING id
	`, instanceID, databaseName, metadataBytes, schema)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var id int64
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return 0, errors.Wrapf(err, "failed to insert")
	}

	return id, nil
}
