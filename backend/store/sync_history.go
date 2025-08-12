package store

import (
	"context"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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
	query := `
		SELECT
			id,
			created_at,
			instance,
			db_name,
			metadata,
			raw_dump
		FROM sync_history
		WHERE id = $1
	`
	h := SyncHistory{
		Metadata: &storepb.DatabaseSchemaMetadata{},
	}

	var m []byte
	if err := s.GetDB().QueryRowContext(ctx, query, uid).Scan(
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

	query := `
		INSERT INTO sync_history (
			instance,
			db_name,
			metadata,
			raw_dump
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var id int64
	if err := tx.QueryRowContext(ctx, query,
		instanceID,
		databaseName,
		metadataBytes,
		schema,
	).Scan(
		&id,
	); err != nil {
		return 0, errors.Wrapf(err, "failed to insert")
	}
	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit")
	}

	return id, nil
}
