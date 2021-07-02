package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
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
		Name: create.Name,
	}
	setting, err := s.FindSetting(ctx, find)
	if err != nil {
		if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
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
		return nil, err
	}

	return setting, nil
}

// FindSetting retrieves a single setting based on find.
// Returns ENOTFOUND if no matching record.
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
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("setting not found: %+v", find)}
	} else if len(list) > 1 {
		return nil, &bytebase.Error{Code: bytebase.ECONFLICT, Message: fmt.Sprintf("found %d activities with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
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
		create.CreatorId,
		create.CreatorId,
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
	where, args := []string{"name = ?"}, []interface{}{find.Name}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
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
