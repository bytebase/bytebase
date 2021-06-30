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
	_ api.ConfigService = (*ConfigService)(nil)
)

// ConfigService represents a service for managing config.
type ConfigService struct {
	l  *zap.Logger
	db *DB
}

// NewConfigService returns a new instance of ConfigService.
func NewConfigService(logger *zap.Logger, db *DB) *ConfigService {
	return &ConfigService{l: logger, db: db}
}

// CreateConfigIfNotExist creates a new config only if the named config does not exist.
func (s *ConfigService) CreateConfigIfNotExist(ctx context.Context, create *api.ConfigCreate) (*api.Config, error) {
	// We do a find followed by a create if NOT found. Though SQLite supports UPSERT ON CONFLICT DO NOTHING syntax, it doesn't
	// support RETURNING in such case. So we have to use separate SELECT and INSERT anyway.
	find := &api.ConfigFind{
		Name: create.Name,
	}
	config, err := s.FindConfig(ctx, find)
	if err != nil {
		if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
			tx, err := s.db.BeginTx(ctx, nil)
			if err != nil {
				return nil, FormatError(err)
			}
			defer tx.Rollback()

			config, err = createConfig(ctx, tx, create)
			if err != nil {
				return nil, err
			}

			if err := tx.Commit(); err != nil {
				return nil, FormatError(err)
			}

			return config, nil
		}
		return nil, err
	}

	return config, nil
}

// FindConfig retrieves a single config based on find.
// Returns ENOTFOUND if no matching record.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *ConfigService) FindConfig(ctx context.Context, find *api.ConfigFind) (*api.Config, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findConfigList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("config not found: %+v", find)}
	} else if len(list) > 1 {
		return nil, &bytebase.Error{Code: bytebase.ECONFLICT, Message: fmt.Sprintf("found %d activities with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

// createConfig creates a new config.
func createConfig(ctx context.Context, tx *Tx, create *api.ConfigCreate) (*api.Config, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO config (
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
	var config api.Config
	if err := row.Scan(
		&config.Name,
		&config.Value,
		&config.Description,
	); err != nil {
		return nil, FormatError(err)
	}

	return &config, nil
}

func findConfigList(ctx context.Context, tx *Tx, find *api.ConfigFind) (_ []*api.Config, err error) {
	// Build WHERE clause.
	where, args := []string{"name = ?"}, []interface{}{find.Name}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    name,
		    value,
			description
		FROM config
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Config, 0)
	for rows.Next() {
		var config api.Config
		if err := rows.Scan(
			&config.Name,
			&config.Value,
			&config.Description,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &config)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}
