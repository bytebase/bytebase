package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// settingRaw is the store model for an Setting.
// Fields have exactly the same meanings as Setting.
type settingRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Name        api.SettingName
	Value       string
	Description string
}

// toSetting creates an instance of Setting based on the settingRaw.
// This is intended to be called when we need to compose an Setting relationship.
func (raw *settingRaw) toSetting() *api.Setting {
	return &api.Setting{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Domain specific fields
		Name:        raw.Name,
		Value:       raw.Value,
		Description: raw.Description,
	}
}

// CreateSettingIfNotExist creates an instance of Setting.
func (s *Store) CreateSettingIfNotExist(ctx context.Context, create *api.SettingCreate) (*api.Setting, bool, error) {
	// TODO(p0ny): re-enable it for prod release.
	if create.Name == api.SettingAppIM && s.db.mode == common.ReleaseModeProd {
		return nil, false, nil
	}

	settingRaw, created, err := s.createSettingRawIfNotExist(ctx, create)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to create Setting with SettingCreate[%+v]", create)
	}
	setting, err := s.composeSetting(ctx, settingRaw)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to compose Setting with settingRaw[%+v]", settingRaw)
	}
	return setting, created, nil
}

// FindSetting finds a list of Setting instances.
func (s *Store) FindSetting(ctx context.Context, find *api.SettingFind) ([]*api.Setting, error) {
	settingRawList, err := s.findSettingRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Setting list with SettingFind[%+v]", find)
	}
	var settingList []*api.Setting
	for _, raw := range settingRawList {
		setting, err := s.composeSetting(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Setting with settingRaw[%+v]", raw)
		}
		settingList = append(settingList, setting)
	}
	return settingList, nil
}

// GetSetting gets an instance of Setting.
func (s *Store) GetSetting(ctx context.Context, find *api.SettingFind) (*api.Setting, error) {
	// TODO(p0ny): re-enable it for release.
	if *find.Name == api.SettingAppIM && s.db.mode == common.ReleaseModeProd {
		return nil, nil
	}

	settingRaw, err := s.getSettingRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Setting with SettingFind[%+v]", find)
	}
	if settingRaw == nil {
		return nil, nil
	}
	setting, err := s.composeSetting(ctx, settingRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Setting with settingRaw[%+v]", settingRaw)
	}
	return setting, nil
}

// PatchSetting patches an instance of Setting.
func (s *Store) PatchSetting(ctx context.Context, patch *api.SettingPatch) (*api.Setting, error) {
	settingRaw, err := s.patchSettingRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Setting with SettingPatch[%+v]", patch)
	}
	setting, err := s.composeSetting(ctx, settingRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Setting with settingRaw[%+v]", settingRaw)
	}
	return setting, nil
}

//
// private functions
//

func (s *Store) composeSetting(ctx context.Context, raw *settingRaw) (*api.Setting, error) {
	setting := raw.toSetting()

	creator, err := s.GetPrincipalByID(ctx, setting.CreatorID)
	if err != nil {
		return nil, err
	}
	setting.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, setting.UpdaterID)
	if err != nil {
		return nil, err
	}
	setting.Updater = updater

	return setting, nil
}

// createSettingRawIfNotExist creates a new setting only if the named setting does not exist.
// The returned bool means the resource is created successfully.
func (s *Store) createSettingRawIfNotExist(ctx context.Context, create *api.SettingCreate) (*settingRaw, bool, error) {
	// We do a find followed by a create if NOT found. Though SQLite supports UPSERT ON CONFLICT DO NOTHING syntax, it doesn't
	// support RETURNING in such case. So we have to use separate SELECT and INSERT anyway.
	find := &api.SettingFind{
		Name: &create.Name,
	}
	setting, err := s.getSettingRaw(ctx, find)
	if err != nil {
		return nil, false, err
	}
	if setting == nil {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, false, FormatError(err)
		}
		defer tx.Rollback()

		setting, err := createSettingImpl(ctx, tx, create)
		if err != nil {
			return nil, false, err
		}

		if err := tx.Commit(); err != nil {
			return nil, false, FormatError(err)
		}

		return setting, true, nil
	}

	return setting, false, nil
}

// findSettingRaw retrieves a list of settings based on find.
func (s *Store) findSettingRaw(ctx context.Context, find *api.SettingFind) ([]*settingRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findSettingImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// getSettingRaw retrieves a single setting based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getSettingRaw(ctx context.Context, find *api.SettingFind) (*settingRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findSettingImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d activities with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

// patchSettingRaw updates an existing setting by name.
// Returns ENOTFOUND if setting does not exist.
func (s *Store) patchSettingRaw(ctx context.Context, patch *api.SettingPatch) (*settingRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	setting, err := patchSettingImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return setting, nil
}

// createSettingImpl creates a new setting.
func createSettingImpl(ctx context.Context, tx *Tx, create *api.SettingCreate) (*settingRaw, error) {
	// Insert row into database.
	query := `
		INSERT INTO setting (
			creator_id,
			updater_id,
			name,
			value,
			description
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING name, value, description
	`
	var settingRaw settingRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		create.Value,
		create.Description,
	).Scan(

		&settingRaw.Name,
		&settingRaw.Value,
		&settingRaw.Description,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &settingRaw, nil
}

func findSettingImpl(ctx context.Context, tx *Tx, find *api.SettingFind) ([]*settingRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.Name; v != nil {
		where, args = append(where, "name = $1"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			name,
			value,
			description
		FROM setting
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into settingRawList.
	var settingRawList []*settingRaw
	for rows.Next() {
		var settingRaw settingRaw
		if err := rows.Scan(
			&settingRaw.CreatorID,
			&settingRaw.CreatedTs,
			&settingRaw.UpdaterID,
			&settingRaw.UpdatedTs,
			&settingRaw.Name,
			&settingRaw.Value,
			&settingRaw.Description,
		); err != nil {
			return nil, FormatError(err)
		}

		settingRawList = append(settingRawList, &settingRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return settingRawList, nil
}

// patchSettingImpl updates a setting by name. Returns the new state of the setting after update.
func patchSettingImpl(ctx context.Context, tx *Tx, patch *api.SettingPatch) (*settingRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "value = $2"), append(args, patch.Value)

	args = append(args, patch.Name)

	var settingRaw settingRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, `
		UPDATE setting
		SET `+strings.Join(set, ", ")+`
		WHERE name = $3
		RETURNING creator_id, created_ts, updater_id, updated_ts, name, value, description
	`,
		args...,
	).Scan(
		&settingRaw.CreatorID,
		&settingRaw.CreatedTs,
		&settingRaw.UpdaterID,
		&settingRaw.UpdatedTs,
		&settingRaw.Name,
		&settingRaw.Value,
		&settingRaw.Description,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("setting not found: %s", patch.Name)}
		}
		return nil, FormatError(err)
	}
	return &settingRaw, nil
}
