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

type ExportArchiveMessage struct {
	ResourceID string
	Workspace  string
	CreatedAt  time.Time
	Bytes      []byte
	Payload    *storepb.ExportArchivePayload
}

// GetExportArchive gets a export archive.
func (s *Store) GetExportArchive(ctx context.Context, resourceID string) (*ExportArchiveMessage, error) {
	q := qb.Q().Space(`
		SELECT
			resource_id,
			workspace,
			created_at,
			bytes,
			payload
		FROM export_archive
		WHERE resource_id = ?
	`, resourceID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var exportArchive ExportArchiveMessage
	var bytesVal, payload []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&exportArchive.ResourceID,
		&exportArchive.Workspace,
		&exportArchive.CreatedAt,
		&bytesVal,
		&payload,
	); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	exportArchivePayload := &storepb.ExportArchivePayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, exportArchivePayload); err != nil {
		return nil, err
	}
	exportArchive.Payload = exportArchivePayload
	exportArchive.Bytes = bytesVal

	return &exportArchive, nil
}

// CreateExportArchive creates a export archive.
func (s *Store) CreateExportArchive(ctx context.Context, create *ExportArchiveMessage) (*ExportArchiveMessage, error) {
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO export_archive (
			workspace,
			bytes,
			payload
		)
		VALUES (?, ?, ?)
		RETURNING resource_id
	`, create.Workspace, create.Bytes, payload)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.ResourceID); err != nil {
		return nil, err
	}

	return create, nil
}

// DeleteExpiredExportArchives deletes export archives older than the specified retention period.
// Returns the number of archives deleted.
func (s *Store) DeleteExpiredExportArchives(ctx context.Context, retentionPeriod time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-retentionPeriod)

	q := qb.Q().Space("DELETE FROM export_archive WHERE created_at < ?", cutoffTime)
	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
