package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
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
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	ret, err := s.findLabelKeyList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil
}

func (s *LabelService) findLabelKeyList(ctx context.Context, tx *Tx, find *api.LabelKeyFind) ([]*api.LabelKey, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.RowStatus; v != nil {
		where, args = append(where, "row_status = ?"), append(args, *v)
	}
	rows, err := tx.Tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
		    key
		FROM label_key
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	var ret []*api.LabelKey
	keymap := make(map[string]*api.LabelKey)
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
		keymap[labelKey.Key] = &labelKey
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	// Find key values.
	valueRows, err := tx.Tx.QueryContext(ctx, `
		SELECT
			key,
			value
		FROM label_value
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer valueRows.Close()

	for valueRows.Next() {
		var key, value string
		if err := valueRows.Scan(
			&key,
			&value,
		); err != nil {
			return nil, FormatError(err)
		}
		labelKey, ok := keymap[key]
		if !ok {
			return nil, common.Errorf(common.Internal, fmt.Errorf("label value doesn't have a label key, key %q, value %q", key, value))
		}
		labelKey.ValueList = append(labelKey.ValueList, value)
	}
	if err := valueRows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil
}

type labelValueUpsert struct {
	rowStatus api.RowStatus
	updaterID int
	key       string
	value     string
}

// PatchLabelKey patches a label key.
func (s *LabelService) PatchLabelKey(ctx context.Context, patch *api.LabelKeyPatch) (*api.LabelKey, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	ret, err := s.findLabelKeyList(ctx, tx, &api.LabelKeyFind{})
	if err != nil {
		return nil, err
	}
	var labelKey *api.LabelKey
	for _, k := range ret {
		if k.ID == patch.ID {
			labelKey = k
		}
	}
	if labelKey == nil {
		return nil, common.Errorf(common.NotFound, fmt.Errorf("label key ID not found: %v", patch.ID))
	}

	// Generate label value upserts.
	var upserts []labelValueUpsert
	// Add all new values.
	for _, v := range patch.ValueList {
		upserts = append(upserts, labelValueUpsert{
			rowStatus: api.Normal,
			updaterID: patch.UpdaterID,
			key:       labelKey.Key,
			value:     v,
		})
	}
	// Archive old values that are not in new values.
	newValues := make(map[string]bool)
	for _, v := range patch.ValueList {
		newValues[v] = true
	}
	for _, v := range labelKey.ValueList {
		if _, ok := newValues[v]; !ok {
			upserts = append(upserts, labelValueUpsert{
				rowStatus: api.Archived,
				updaterID: patch.UpdaterID,
				key:       labelKey.Key,
				value:     v,
			})
		}
	}

	for _, upsert := range upserts {
		if err := s.pgUpsertLabelValue(ctx, tx.Tx, upsert); err != nil {
			return nil, err
		}
		if err := s.upsertLabelValue(ctx, tx.PTx, upsert); err != nil {
			return nil, err
		}
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	labelKey.ValueList = patch.ValueList
	return labelKey, nil
}

func (s *LabelService) upsertLabelValue(ctx context.Context, tx *sql.Tx, upsert labelValueUpsert) error {
	// Upsert row into label_value
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO label_value (
			row_status,
			creator_id,
			updater_id,
			key,
			value
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(key, value) DO UPDATE SET
				row_status = excluded.row_status,
				creator_id = excluded.creator_id,
				updater_id = excluded.updater_id
		`,
		upsert.rowStatus,
		upsert.updaterID,
		upsert.updaterID,
		upsert.key,
		upsert.value,
	); err != nil {
		return FormatError(err)
	}
	return nil
}

func (s *LabelService) pgUpsertLabelValue(ctx context.Context, tx *sql.Tx, upsert labelValueUpsert) error {
	// Upsert row into label_value
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO label_value (
			row_status,
			creator_id,
			updater_id,
			key,
			value
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(key, value) DO UPDATE SET
			row_status = excluded.row_status,
			creator_id = excluded.creator_id,
			updater_id = excluded.updater_id
		`,
		upsert.rowStatus,
		upsert.updaterID,
		upsert.updaterID,
		upsert.key,
		upsert.value,
	); err != nil {
		return FormatError(err)
	}
	return nil
}

// FindDatabaseLabelList finds the labels associated with the database.
func (s *LabelService) FindDatabaseLabelList(ctx context.Context, find *api.DatabaseLabelFind) ([]*api.DatabaseLabel, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	databaseLabelList, err := s.findDatabaseLabels(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
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
	rows, err := tx.Tx.QueryContext(ctx, `
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

func (s *LabelService) upsertDatabaseLabel(ctx context.Context, tx *sql.Tx, upsert *api.DatabaseLabelUpsert) (*api.DatabaseLabel, error) {
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

func (s *LabelService) pgUpsertDatabaseLabel(ctx context.Context, tx *sql.Tx, upsert *api.DatabaseLabelUpsert) (*api.DatabaseLabel, error) {
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
		VALUES ($1, $2, $3, $4, $5, $6)
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

// SetDatabaseLabelList sets the labels for a database.
func (s *LabelService) SetDatabaseLabelList(ctx context.Context, labelList []*api.DatabaseLabel, databaseID int, updaterID int) ([]*api.DatabaseLabel, error) {
	oldLabelList, err := s.FindDatabaseLabelList(ctx, &api.DatabaseLabelFind{
		DatabaseID: &databaseID,
	})
	if err != nil {
		return nil, FormatError(err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	var ret []*api.DatabaseLabel

	for _, oldLabel := range oldLabelList {
		// Archive all old labels
		// Skip environment label key because we don't store it.
		if oldLabel.Key == api.EnvironmentKeyName {
			continue
		}
		upsert := &api.DatabaseLabelUpsert{
			UpdaterID:  updaterID,
			RowStatus:  api.Archived,
			DatabaseID: databaseID,
			Key:        oldLabel.Key,
			Value:      oldLabel.Value,
		}
		if _, err := s.pgUpsertDatabaseLabel(ctx, tx.Tx, upsert); err != nil {
			return nil, err
		}
		if _, err := s.upsertDatabaseLabel(ctx, tx.PTx, upsert); err != nil {
			return nil, err
		}
	}

	for _, label := range labelList {
		// Upsert all new labels
		// Skip environment label key because we don't store it.
		if label.Key == api.EnvironmentKeyName {
			continue
		}
		upsert := &api.DatabaseLabelUpsert{
			UpdaterID:  updaterID,
			RowStatus:  api.Normal,
			DatabaseID: databaseID,
			Key:        label.Key,
			Value:      label.Value,
		}
		label, err := s.pgUpsertDatabaseLabel(ctx, tx.Tx, upsert)
		if err != nil {
			return nil, err
		}
		if _, err := s.upsertDatabaseLabel(ctx, tx.PTx, upsert); err != nil {
			return nil, err
		}
		ret = append(ret, label)
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil

}
