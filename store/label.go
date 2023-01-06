package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
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
