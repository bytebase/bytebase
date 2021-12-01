package store

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

var (
	_ api.LabelService = (*LabelService)(nil)
)

// LabelService represents a service for managing labels.
type LabelService struct {
	l  *zap.Logger
	db *DB
}

// NewLabelService returns a new instance of LabelService.
func NewLabelService(logger *zap.Logger, db *DB) *LabelService {
	return &LabelService{l: logger, db: db}
}

// FindLabelKeys retrieves a list of label keys for labels based on find.
func (s *LabelService) FindLabelKeys(ctx context.Context, find *api.LabelKeyFind) ([]api.LabelKey, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
		    key
		FROM label_key`,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	var ret []api.LabelKey
	for rows.Next() {
		var labelKey api.LabelKey
		if err := rows.Scan(
			&labelKey.ID,
			&labelKey.CreatorID,
			&labelKey.CreatedTs,
			&labelKey.UpdaterID,
			&labelKey.UpdatedTs,
			&labelKey.Key,
		); err != nil {
			return nil, FormatError(err)
		}

		ret = append(ret, labelKey)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil
}
