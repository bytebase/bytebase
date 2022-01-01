package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.SettingService = (*SettingService)(nil)
)

// SettingService represents a service for managing setting.
type SettingService struct {
	l  *zap.Logger
	db *DB
}

// NewSettingService returns a new instance of SettingService.
func NewSettingService(logger *zap.Logger, db *DB) *SettingService {
	return &SettingService{l: logger, db: db}
}

// CreateSettingIfNotExist creates a new setting only if the named setting does not exist.
func (s *SettingService) CreateSettingIfNotExist(ctx context.Context, create *api.SettingCreate) (*api.Setting, error) {
	// We do a find followed by a create if NOT found. Though SQLite supports UPSERT ON CONFLICT DO NOTHING syntax, it doesn't
	// support RETURNING in such case. So we have to use separate SELECT and INSERT anyway.
	find := &api.SettingFind{
		Name: &create.Name,
	}
	setting, err := s.FindSetting(ctx, find)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, FormatError(err)
		}
		defer tx.Rollback()

		setting, err = createSetting(ctx, tx, create)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, FormatError(err)
		}

		return setting, nil
	}

	return setting, nil
}

// FindSettingList retrieves a list of settings based on find.
func (s *SettingService) FindSettingList(ctx context.Context, find *api.SettingFind) ([]*api.Setting, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findSettingList(ctx, tx, find)
	if err != nil {
		return []*api.Setting{}, err
	}
	return list, nil
}

// FindSetting retrieves a single setting based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *SettingService) FindSetting(ctx context.Context, find *api.SettingFind) (*api.Setting, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findSettingList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d activities with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

// PatchSetting updates an existing setting by name.
// Returns ENOTFOUND if setting does not exist.
func (s *SettingService) PatchSetting(ctx context.Context, patch *api.SettingPatch) (*api.Setting, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	setting, err := patchSetting(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return setting, nil
}

// createSetting creates a new setting.
func createSetting(ctx context.Context, tx *Tx, create *api.SettingCreate) (*api.Setting, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO setting (
			creator_id,
			updater_id,
			name,
			value,
			description
		)
		VALUES (?, ?, ?, ?, ?)
		RETURNING name, value, description
	`,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		create.Value,
		create.Description,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var setting api.Setting
	if err := row.Scan(
		&setting.Name,
		&setting.Value,
		&setting.Description,
	); err != nil {
		return nil, FormatError(err)
	}

	return &setting, nil
}

func findSettingList(ctx context.Context, tx *Tx, find *api.SettingFind) (_ []*api.Setting, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.Name; v != nil {
		where, args = append(where, "name = ?"), append(args, *v)
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

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Setting, 0)
	for rows.Next() {
		var setting api.Setting
		if err := rows.Scan(
			&setting.CreatorID,
			&setting.CreatedTs,
			&setting.UpdaterID,
			&setting.UpdatedTs,
			&setting.Name,
			&setting.Value,
			&setting.Description,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &setting)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchSetting updates a setting by name. Returns the new state of the setting after update.
func patchSetting(ctx context.Context, tx *Tx, patch *api.SettingPatch) (*api.Setting, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "value = ?"), append(args, patch.Value)

	args = append(args, patch.Name)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE setting
		SET `+strings.Join(set, ", ")+`
		WHERE name = ?
		RETURNING creator_id, created_ts, updater_id, updated_ts, name, value, description
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var setting api.Setting
		if err := row.Scan(
			&setting.CreatorID,
			&setting.CreatedTs,
			&setting.UpdaterID,
			&setting.UpdatedTs,
			&setting.Name,
			&setting.Value,
			&setting.Description,
		); err != nil {
			return nil, FormatError(err)
		}

		return &setting, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("setting not found: %s", patch.Name)}
}
