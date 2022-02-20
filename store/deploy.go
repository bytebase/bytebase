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
	_ api.DeploymentConfigService = (*DeploymentConfigService)(nil)
)

// DeploymentConfigService represents a service for managing deployment configurations.
type DeploymentConfigService struct {
	l  *zap.Logger
	db *DB
}

// NewDeploymentConfigService returns a new instance of DeploymentConfigService.
func NewDeploymentConfigService(logger *zap.Logger, db *DB) *DeploymentConfigService {
	return &DeploymentConfigService{l: logger, db: db}
}

// FindDeploymentConfig finds the deployment configuration in a project.
func (s *DeploymentConfigService) FindDeploymentConfig(ctx context.Context, find *api.DeploymentConfigFind) (*api.DeploymentConfig, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.PTx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			project_id,
			name,
			config
		FROM deployment_config
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	var ret []*api.DeploymentConfig
	for rows.Next() {
		var cfg api.DeploymentConfig
		if err := rows.Scan(
			&cfg.ID,
			&cfg.CreatorID,
			&cfg.CreatedTs,
			&cfg.UpdaterID,
			&cfg.UpdatedTs,
			&cfg.ProjectID,
			&cfg.Name,
			&cfg.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		ret = append(ret, &cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	switch len(ret) {
	case 0:
		return nil, nil
	case 1:
		return ret[0], nil
	default:
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d deployment configurations with filter %+v, expect 1", len(ret), find)}
	}
}

// UpsertDeploymentConfig upserts a deployment configuration to a project.
func (s *DeploymentConfigService) UpsertDeploymentConfig(ctx context.Context, upsert *api.DeploymentConfigUpsert) (*api.DeploymentConfig, error) {
	// Validate the deployment configuration.
	if _, err := api.ValidateAndGetDeploymentSchedule(upsert.Payload); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	cfg, err := s.pgUpsertDeploymentConfig(ctx, tx.PTx, upsert)
	if err != nil {
		return nil, FormatError(err)
	}
	if _, err := s.upsertDeploymentConfig(ctx, tx.Tx, upsert); err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return cfg, nil
}

func (s *DeploymentConfigService) upsertDeploymentConfig(ctx context.Context, tx *sql.Tx, upsert *api.DeploymentConfigUpsert) (*api.DeploymentConfig, error) {
	row, err := tx.QueryContext(ctx, `
	INSERT INTO deployment_config (
		creator_id,
		updater_id,
		project_id,
		name,
		config
	)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(project_id) DO UPDATE SET
		creator_id = excluded.creator_id,
		updater_id = excluded.updater_id,
		name = excluded.name,
		config = excluded.config
	RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, name, config
	`,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.ProjectID,
		upsert.Name,
		upsert.Payload,
	)

	if err != nil {
		return nil, err
	}
	defer row.Close()

	row.Next()
	var cfg api.DeploymentConfig
	if err := row.Scan(
		&cfg.ID,
		&cfg.CreatorID,
		&cfg.CreatedTs,
		&cfg.UpdaterID,
		&cfg.UpdatedTs,
		&cfg.ProjectID,
		&cfg.Name,
		&cfg.Payload,
	); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *DeploymentConfigService) pgUpsertDeploymentConfig(ctx context.Context, tx *sql.Tx, upsert *api.DeploymentConfigUpsert) (*api.DeploymentConfig, error) {
	row, err := tx.QueryContext(ctx, `
	INSERT INTO deployment_config (
		creator_id,
		updater_id,
		project_id,
		name,
		config
	)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT(project_id) DO UPDATE SET
		creator_id = excluded.creator_id,
		updater_id = excluded.updater_id,
		name = excluded.name,
		config = excluded.config
	RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, name, config
	`,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.ProjectID,
		upsert.Name,
		upsert.Payload,
	)

	if err != nil {
		return nil, err
	}
	defer row.Close()

	row.Next()
	var cfg api.DeploymentConfig
	if err := row.Scan(
		&cfg.ID,
		&cfg.CreatorID,
		&cfg.CreatedTs,
		&cfg.UpdaterID,
		&cfg.UpdatedTs,
		&cfg.ProjectID,
		&cfg.Name,
		&cfg.Payload,
	); err != nil {
		return nil, err
	}
	return &cfg, nil
}
