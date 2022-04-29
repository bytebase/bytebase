package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// labelKeyRaw is the store model for an LabelKey.
// Fields have exactly the same meanings as LabelKey.
type labelKeyRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	// bb.environment is a reserved key and identically mapped from environments. It has zero ID and its values are immutable.
	Key       string
	ValueList []string
}

// toLabelKey creates an instance of LabelKey based on the labelKeyRaw.
// This is intended to be called when we need to compose an LabelKey relationship.
func (raw *labelKeyRaw) toLabelKey() *api.LabelKey {
	labelKey := api.LabelKey{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Domain specific fields
		// bb.environment is a reserved key and identically mapped from environments. It has zero ID and its values are immutable.
		Key: raw.Key,
	}
	labelKey.ValueList = append(labelKey.ValueList, raw.ValueList...)
	return &labelKey
}

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

// toDatabaseLabel creates an instance of DatabaseLabel based on the databaseLabelRaw.
// This is intended to be called when we need to compose an DatabaseLabel relationship.
func (raw *databaseLabelRaw) toDatabaseLabel() *api.DatabaseLabel {
	return &api.DatabaseLabel{
		ID: raw.ID,

		// Standard fields
		RowStatus: raw.RowStatus,
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		DatabaseID: raw.DatabaseID,
		Key:        raw.Key,

		// Domain specific fields
		Value: raw.Value,
	}
}

// SetDatabaseLabelList sets the labels for a database.
func (s *Store) SetDatabaseLabelList(ctx context.Context, labelList []*api.DatabaseLabel, databaseID int, updaterID int) ([]*api.DatabaseLabel, error) {
	oldLabelRawList, err := s.findDatabaseLabelRaw(ctx, &api.DatabaseLabelFind{
		DatabaseID: &databaseID,
	})
	if err != nil {
		return nil, FormatError(err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	var ret []*api.DatabaseLabel

	for _, oldLabelRaw := range oldLabelRawList {
		// Archive all old labels
		// Skip environment label key because we don't store it.
		if oldLabelRaw.Key == api.EnvironmentKeyName {
			continue
		}
		upsert := &api.DatabaseLabelUpsert{
			UpdaterID:  updaterID,
			RowStatus:  api.Archived,
			DatabaseID: databaseID,
			Key:        oldLabelRaw.Key,
			Value:      oldLabelRaw.Value,
		}
		if _, err := s.upsertDatabaseLabelImpl(ctx, tx.PTx, upsert); err != nil {
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
		labelRaw, err := s.upsertDatabaseLabelImpl(ctx, tx.PTx, upsert)
		if err != nil {
			return nil, err
		}

		label, err := s.composeDatabaseLabel(ctx, labelRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose DatabaseLabel with databaseLabelRaw[%+v], error[%w]", labelRaw, err)
		}
		ret = append(ret, label)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil

}

// FindLabelKey finds a list of LabelKey instances
func (s *Store) FindLabelKey(ctx context.Context, find *api.LabelKeyFind) ([]*api.LabelKey, error) {
	labelKeyRawList, err := s.findLabelKeyRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find LabelKey list with LabelKeyFind[%+v], error[%w]", find, err)
	}
	var labelKeyList []*api.LabelKey
	for _, raw := range labelKeyRawList {
		labelKey, err := s.composeLabelKey(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose LabelKey with labelKeyRaw[%+v], error[%w]", raw, err)
		}
		labelKeyList = append(labelKeyList, labelKey)
	}
	return labelKeyList, nil
}

// PatchLabelKey patches an instance of LabelKey
func (s *Store) PatchLabelKey(ctx context.Context, patch *api.LabelKeyPatch) (*api.LabelKey, error) {
	labelKeyRaw, err := s.patchLabelKeyRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch LabelKey with LabelKeyPatch[%+v], error[%w]", patch, err)
	}
	labelKey, err := s.composeLabelKey(ctx, labelKeyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose LabelKey with labelKeyRaw[%+v], error[%w]", labelKeyRaw, err)
	}
	return labelKey, nil
}

// FindDatabaseLabel finds a list of DatabaseLabel instances
func (s *Store) FindDatabaseLabel(ctx context.Context, find *api.DatabaseLabelFind) ([]*api.DatabaseLabel, error) {
	labelKeyRawList, err := s.findDatabaseLabelRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find DatabaseLabel list with DatabaseLabelFind[%+v], error[%w]", find, err)
	}
	var labelKeyList []*api.DatabaseLabel
	for _, raw := range labelKeyRawList {
		labelKey, err := s.composeDatabaseLabel(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose DatabaseLabel with labelKeyRaw[%+v], error[%w]", raw, err)
		}
		labelKeyList = append(labelKeyList, labelKey)
	}
	return labelKeyList, nil
}

//
// private functions
//

// composeLabelKey composes an instance of LabelKey by labelKeyRaw
func (s *Store) composeLabelKey(ctx context.Context, raw *labelKeyRaw) (*api.LabelKey, error) {
	labelKey := raw.toLabelKey()

	creator, err := s.GetPrincipalByID(ctx, labelKey.CreatorID)
	if err != nil {
		return nil, err
	}
	labelKey.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, labelKey.UpdaterID)
	if err != nil {
		return nil, err
	}
	labelKey.Updater = updater

	return labelKey, nil
}

// composeDatabaseLabel composes an instance of DatabaseLabel by databaseLabelRaw
func (s *Store) composeDatabaseLabel(ctx context.Context, raw *databaseLabelRaw) (*api.DatabaseLabel, error) {
	databaseLabel := raw.toDatabaseLabel()

	creator, err := s.GetPrincipalByID(ctx, databaseLabel.CreatorID)
	if err != nil {
		return nil, err
	}
	databaseLabel.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, databaseLabel.UpdaterID)
	if err != nil {
		return nil, err
	}
	databaseLabel.Updater = updater

	return databaseLabel, nil
}

// findLabelKeyRaw retrieves a list of label keys for labels based on find.
func (s *Store) findLabelKeyRaw(ctx context.Context, find *api.LabelKeyFind) ([]*labelKeyRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	ret, err := s.findLabelKeyImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return ret, nil
}

func (s *Store) findLabelKeyImpl(ctx context.Context, tx *sql.Tx, find *api.LabelKeyFind) ([]*labelKeyRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.RowStatus; v != nil {
		where, args = append(where, "row_status = $1"), append(args, *v)
	}
	rows, err := tx.QueryContext(ctx, `
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
	// Iterate over result set and deserialize rows into list.
	var labelKeyRawList []*labelKeyRaw
	keymap := make(map[string]*labelKeyRaw)
	for rows.Next() {
		var labelKeyRaw labelKeyRaw
		if err := rows.Scan(
			&labelKeyRaw.ID,
			&labelKeyRaw.CreatorID,
			&labelKeyRaw.CreatedTs,
			&labelKeyRaw.UpdaterID,
			&labelKeyRaw.UpdatedTs,
			&labelKeyRaw.Key,
		); err != nil {
			return nil, FormatError(err)
		}

		labelKeyRawList = append(labelKeyRawList, &labelKeyRaw)
		keymap[labelKeyRaw.Key] = &labelKeyRaw
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	if err := rows.Close(); err != nil {
		return nil, FormatError(err)
	}

	// Find key values.
	valueRows, err := tx.QueryContext(ctx, `
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

	return labelKeyRawList, nil
}

type labelValueUpsert struct {
	rowStatus api.RowStatus
	updaterID int
	key       string
	value     string
}

// patchLabelKeyRaw patches a label key.
func (s *Store) patchLabelKeyRaw(ctx context.Context, patch *api.LabelKeyPatch) (*labelKeyRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	labelKeyRawList, err := s.findLabelKeyImpl(ctx, tx.PTx, &api.LabelKeyFind{})
	if err != nil {
		return nil, err
	}
	var labelKeyRaw *labelKeyRaw
	for _, raw := range labelKeyRawList {
		if raw.ID == patch.ID {
			labelKeyRaw = raw
		}
	}
	if labelKeyRaw == nil {
		return nil, common.Errorf(common.NotFound, fmt.Errorf("label key not found with ID %v", patch.ID))
	}

	// Generate label value upserts.
	var upserts []labelValueUpsert
	// Add all new values.
	for _, v := range patch.ValueList {
		upserts = append(upserts, labelValueUpsert{
			rowStatus: api.Normal,
			updaterID: patch.UpdaterID,
			key:       labelKeyRaw.Key,
			value:     v,
		})
	}
	// Archive old values that are not in new values.
	newValues := make(map[string]bool)
	for _, v := range patch.ValueList {
		newValues[v] = true
	}
	for _, v := range labelKeyRaw.ValueList {
		if _, ok := newValues[v]; !ok {
			upserts = append(upserts, labelValueUpsert{
				rowStatus: api.Archived,
				updaterID: patch.UpdaterID,
				key:       labelKeyRaw.Key,
				value:     v,
			})
		}
	}

	for _, upsert := range upserts {
		if err := s.upsertLabelValueImpl(ctx, tx.PTx, upsert); err != nil {
			return nil, err
		}
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	labelKeyRaw.ValueList = patch.ValueList
	return labelKeyRaw, nil
}

func (s *Store) upsertLabelValueImpl(ctx context.Context, tx *sql.Tx, upsert labelValueUpsert) error {
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

// findDatabaseLabelRaw finds the labels associated with the database.
func (s *Store) findDatabaseLabelRaw(ctx context.Context, find *api.DatabaseLabelFind) ([]*databaseLabelRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	databaseLabelList, err := s.findDatabaseLabelsImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return databaseLabelList, nil
}

func (s *Store) findDatabaseLabelsImpl(ctx context.Context, tx *sql.Tx, find *api.DatabaseLabelFind) ([]*databaseLabelRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
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

func (s *Store) upsertDatabaseLabelImpl(ctx context.Context, tx *sql.Tx, upsert *api.DatabaseLabelUpsert) (*databaseLabelRaw, error) {
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
	var dbLabelRaw databaseLabelRaw
	if err := row.Scan(
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

	return &dbLabelRaw, nil
}
