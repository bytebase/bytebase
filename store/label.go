package store

import (
	"context"
	"strings"

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

// FindLabelKeyList retrieves a list of label keys for labels based on find.
func (s *LabelService) FindLabelKeyList(ctx context.Context, find *api.LabelKeyFind) ([]*api.LabelKey, error) {
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
	var ret []*api.LabelKey
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

		ret = append(ret, &labelKey)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil
}

func (s *LabelService) FindDatabaseLabels(ctx context.Context, find *api.DatabaseLabelFind) ([]*api.DatabaseLabel, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	databaseLabelList, err := s.findDatabaseLabels(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return databaseLabelList, nil
}

func (s *LabelService) findDatabaseLabels(ctx context.Context, tx *Tx, find *api.DatabaseLabelFind) ([]*api.DatabaseLabel, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, "row_status = ?"), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, "database_id = ?"), append(args, *v)
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			row_status,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			database_id,
		    key,
			value
		FROM db_label
		WHERE `+strings.Join(where, " AND ")+` ORDER BY key`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	var ret []*api.DatabaseLabel
	for rows.Next() {
		var databaseLabel api.DatabaseLabel
		if err := rows.Scan(
			&databaseLabel.ID,
			&databaseLabel.RowStatus,
			&databaseLabel.CreatorID,
			&databaseLabel.CreatedTs,
			&databaseLabel.UpdaterID,
			&databaseLabel.UpdatedTs,
			&databaseLabel.DatabaseID,
			&databaseLabel.Key,
			&databaseLabel.Value,
		); err != nil {
			return nil, FormatError(err)
		}

		ret = append(ret, &databaseLabel)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil
}

func (s *LabelService) upsertDatabaseLabel(ctx context.Context, tx *Tx, upsert *api.DatabaseLabelUpsert) (*api.DatabaseLabel, error) {
	// Upsert row into db_label
	row, err := tx.QueryContext(ctx, `
		INSERT INTO db_label (
			row_status,
			creator_id,
			updater_id,
			database_id,
			key,
			value
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(database_id, key) DO UPDATE SET
				row_status = excluded.row_status,
				creator_id = excluded.creator_id,
				updater_id = excluded.updater_id,
				value = excluded.value
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, key, value
		`,
		upsert.RowStatus,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.DatabaseID,
		upsert.Key,
		upsert.Value,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var databaseLabel api.DatabaseLabel
	if err := row.Scan(
		&databaseLabel.ID,
		&databaseLabel.RowStatus,
		&databaseLabel.CreatorID,
		&databaseLabel.CreatedTs,
		&databaseLabel.UpdaterID,
		&databaseLabel.UpdatedTs,
		&databaseLabel.DatabaseID,
		&databaseLabel.Key,
		&databaseLabel.Value,
	); err != nil {
		return nil, FormatError(err)
	}

	return &databaseLabel, nil
}

func (s *LabelService) SetDatabaseLabels(ctx context.Context, labels []*api.DatabaseLabel, databaseID int, updaterID int) ([]*api.DatabaseLabel, error) {
	oldLabels, err := s.FindDatabaseLabels(ctx, &api.DatabaseLabelFind{
		DatabaseID: &databaseID,
	})
	if err != nil {
		return nil, FormatError(err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var ret []*api.DatabaseLabel

	for _, oldLabel := range oldLabels {
		// Archive all old labels
		label, err := s.upsertDatabaseLabel(ctx, tx, &api.DatabaseLabelUpsert{
			UpdaterID:  updaterID,
			RowStatus:  api.Archived,
			DatabaseID: databaseID,
			Key:        oldLabel.Key,
			Value:      oldLabel.Value,
		})
		if err != nil {
			return nil, err
		}
		ret = append(ret, label)
	}

	for _, label := range labels {
		// Upsert all new labels
		label, err := s.upsertDatabaseLabel(ctx, tx, &api.DatabaseLabelUpsert{
			UpdaterID:  updaterID,
			RowStatus:  api.Normal,
			DatabaseID: databaseID,
			Key:        label.Key,
			Value:      label.Value,
		})
		if err != nil {
			return nil, err
		}
		ret = append(ret, label)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil

}
