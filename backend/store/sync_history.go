package store

import (
	"context"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type SyncHistory struct {
	UID         int64
	DatabaseUID int
	Schema      string

	CreatorUID int
}

// UpsertDBSchema upserts a database schema.
func (s *Store) CreateSyncHistory(ctx context.Context, databaseID int, metadata *storepb.DatabaseSchemaMetadata, schema string, updaterID int) (int64, error) {
	metadataBytes, err := protojson.Marshal(metadata)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to marshal")
	}

	query := `
		INSERT INTO sync_history (
			creator_id,
			database_id,
			metadata,
			raw_dump
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var id int64
	if err := tx.QueryRowContext(ctx, query,
		updaterID,
		databaseID,
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
