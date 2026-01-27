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

type ExportArchiveMessage struct {
	UID       int
	CreatedAt time.Time
	Bytes     []byte
	Payload   *storepb.ExportArchivePayload
}

// FindExportArchiveMessage is the API message for finding export archives.
type FindExportArchiveMessage struct {
	UID *int
}

// GetExportArchive gets a export archive.
func (s *Store) GetExportArchive(ctx context.Context, find *FindExportArchiveMessage) (*ExportArchiveMessage, error) {
	exportArchives, err := s.ListExportArchives(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(exportArchives) == 0 {
		return nil, nil
	}
	if len(exportArchives) > 1 {
		return nil, errors.Errorf("expected 1 export archive, got %d", len(exportArchives))
	}
	return exportArchives[0], nil
}

// ListExportArchives lists export archives.
func (s *Store) ListExportArchives(ctx context.Context, find *FindExportArchiveMessage) ([]*ExportArchiveMessage, error) {
	q := qb.Q().Space(`
		SELECT
			id,
			created_at,
			bytes,
			payload
		FROM export_archive
		WHERE TRUE
	`)

	if v := find.UID; v != nil {
		q.And("id = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exportArchives []*ExportArchiveMessage
	for rows.Next() {
		var exportArchive ExportArchiveMessage
		var bytes, payload []byte
		if err := rows.Scan(
			&exportArchive.UID,
			&exportArchive.CreatedAt,
			&bytes,
			&payload,
		); err != nil {
			return nil, err
		}
		exportArchivePayload := &storepb.ExportArchivePayload{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, exportArchivePayload); err != nil {
			return nil, err
		}
		exportArchive.Payload = exportArchivePayload
		exportArchive.Bytes = bytes
		exportArchives = append(exportArchives, &exportArchive)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return exportArchives, nil
}

// CreateExportArchive creates a export archive.
func (s *Store) CreateExportArchive(ctx context.Context, create *ExportArchiveMessage) (*ExportArchiveMessage, error) {
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO export_archive (
			bytes,
			payload
		)
		VALUES (?, ?)
		RETURNING id
	`, create.Bytes, payload)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.UID); err != nil {
		return nil, err
	}

	return create, nil
}

// DeleteExportArchive deletes a export archive.
func (s *Store) DeleteExportArchive(ctx context.Context, uid int) error {
	q := qb.Q().Space("DELETE FROM export_archive WHERE id = ?", uid)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return nil
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
