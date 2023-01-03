package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// databaseLabelRaw is the store model for an DatabaseLabel.
// Fields have exactly the same meanings as DatabaseLabel.
type databaseLabelRaw struct {
	ID int

	// Standard fields
	RowStatus api.RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	DatabaseID int
	Key        string

	// Domain specific fields
	Value string
}

// SetDatabaseLabelList sets the labels for a database.
func (s *Store) SetDatabaseLabelList(ctx context.Context, labelList []*api.DatabaseLabel, databaseID int, updaterID int) ([]*api.DatabaseLabel, error) {
	oldLabelRawList, err := s.findDatabaseLabelRaw(ctx, &api.DatabaseLabelFind{
		DatabaseID: databaseID,
	})
	if err != nil {
		return nil, FormatError(err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	for _, oldLabelRaw := range oldLabelRawList {
		// Archive all old labels
		// Skip environment label key because we don't store it.
		if oldLabelRaw.Key == api.EnvironmentLabelKey {
			continue
		}
		upsert := &api.DatabaseLabelUpsert{
			UpdaterID:  updaterID,
			RowStatus:  api.Archived,
			DatabaseID: databaseID,
			Key:        oldLabelRaw.Key,
			Value:      oldLabelRaw.Value,
		}
		if _, err := s.upsertDatabaseLabelImpl(ctx, tx, upsert); err != nil {
			return nil, err
		}
	}

	var ret []*api.DatabaseLabel
	for _, label := range labelList {
		// Upsert all new labels
		// Skip environment label key because we don't store it.
		if label.Key == api.EnvironmentLabelKey {
			continue
		}
		upsert := &api.DatabaseLabelUpsert{
			UpdaterID:  updaterID,
			RowStatus:  api.Normal,
			DatabaseID: databaseID,
			Key:        label.Key,
			Value:      label.Value,
		}
		labelRaw, err := s.upsertDatabaseLabelImpl(ctx, tx, upsert)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &api.DatabaseLabel{Key: labelRaw.Key, Value: labelRaw.Value})
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(databaseLabelCacheNamespace, databaseID, ret); err != nil {
		return nil, err
	}

	return ret, nil
}

// findDatabaseLabel finds a list of DatabaseLabel instances.
func (s *Store) findDatabaseLabel(ctx context.Context, find *api.DatabaseLabelFind) ([]*api.DatabaseLabel, error) {
	var l []*api.DatabaseLabel
	has, err := s.cache.FindCache(databaseLabelCacheNamespace, find.DatabaseID, &l)
	if err != nil {
		return nil, err
	}
	if has {
		return l, nil
	}

	labelKeyRawList, err := s.findDatabaseLabelRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find DatabaseLabel list with DatabaseLabelFind[%+v]", find)
	}
	for _, raw := range labelKeyRawList {
		l = append(l, &api.DatabaseLabel{Key: raw.Key, Value: raw.Value})
	}
	if err := s.cache.UpsertCache(databaseLabelCacheNamespace, find.DatabaseID, l); err != nil {
		return nil, err
	}
	return l, nil
}

// private functions
//
// findDatabaseLabelRaw finds the labels associated with the database.
func (s *Store) findDatabaseLabelRaw(ctx context.Context, find *api.DatabaseLabelFind) ([]*databaseLabelRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	databaseLabelList, err := s.findDatabaseLabelsImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return databaseLabelList, nil
}

func (*Store) findDatabaseLabelsImpl(ctx context.Context, tx *Tx, find *api.DatabaseLabelFind) ([]*databaseLabelRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, find.DatabaseID)
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
	var ret []*databaseLabelRaw
	for rows.Next() {
		var dbLabelRaw databaseLabelRaw
		if err := rows.Scan(
			&dbLabelRaw.ID,
			&dbLabelRaw.RowStatus,
			&dbLabelRaw.CreatorID,
			&dbLabelRaw.CreatedTs,
			&dbLabelRaw.UpdaterID,
			&dbLabelRaw.UpdatedTs,
			&dbLabelRaw.DatabaseID,
			&dbLabelRaw.Key,
			&dbLabelRaw.Value,
		); err != nil {
			return nil, FormatError(err)
		}

		ret = append(ret, &dbLabelRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil
}

func (*Store) upsertDatabaseLabelImpl(ctx context.Context, tx *Tx, upsert *api.DatabaseLabelUpsert) (*databaseLabelRaw, error) {
	// Upsert row into db_label
	query := `
		INSERT INTO db_label (
			row_status,
			creator_id,
			updater_id,
			database_id,
			key,
			value
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(database_id, key) DO UPDATE SET
			row_status = excluded.row_status,
			creator_id = excluded.creator_id,
			updater_id = excluded.updater_id,
			value = excluded.value
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, key, value
	`
	var dbLabelRaw databaseLabelRaw
	if err := tx.QueryRowContext(ctx, query,
		upsert.RowStatus,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.DatabaseID,
		upsert.Key,
		upsert.Value,
	).Scan(
		&dbLabelRaw.ID,
		&dbLabelRaw.RowStatus,
		&dbLabelRaw.CreatorID,
		&dbLabelRaw.CreatedTs,
		&dbLabelRaw.UpdaterID,
		&dbLabelRaw.UpdatedTs,
		&dbLabelRaw.DatabaseID,
		&dbLabelRaw.Key,
		&dbLabelRaw.Value,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &dbLabelRaw, nil
}
