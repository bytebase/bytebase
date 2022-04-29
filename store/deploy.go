package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// deploymentConfigRaw is the store model for an DeploymentConfig.
// Fields have exactly the same meanings as DeploymentConfig.
type deploymentConfigRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	ProjectID int

	// Domain specific fields
	Name    string
	Payload string
}

// toDeploymentConfig creates an instance of DeploymentConfig based on the deploymentConfigRaw.
// This is intended to be called when we need to compose an DeploymentConfig relationship.
func (raw *deploymentConfigRaw) toDeploymentConfig() *api.DeploymentConfig {
	return &api.DeploymentConfig{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		ProjectID: raw.ProjectID,

		// Domain specific fields
		Name:    raw.Name,
		Payload: raw.Payload,
	}
}

// GetDeploymentConfigByProjectID gets an instance of DeploymentConfig
func (s *Store) GetDeploymentConfigByProjectID(ctx context.Context, projectID int) (*api.DeploymentConfig, error) {
	deploymentConfigRaw, err := s.getDeploymentConfigImpl(ctx, &api.DeploymentConfigFind{ProjectID: &projectID})
	if err != nil {
		return nil, fmt.Errorf("failed to get DeploymentConfig with projectID[%d], error[%w]", projectID, err)
	}
	if deploymentConfigRaw == nil {
		return nil, nil
	}
	deploymentConfig, err := s.composeDeploymentConfig(ctx, deploymentConfigRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose DeploymentConfig with deploymentConfigRaw[%+v], error[%w]", deploymentConfigRaw, err)
	}
	return deploymentConfig, nil
}

// UpsertDeploymentConfig upserts an instance of DeploymentConfig
func (s *Store) UpsertDeploymentConfig(ctx context.Context, upsert *api.DeploymentConfigUpsert) (*api.DeploymentConfig, error) {
	deploymentConfigRaw, err := s.upsertDeploymentConfigRaw(ctx, upsert)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert deployment config with DeploymentConfigUpsert[%+v], error[%w]", upsert, err)
	}
	deploymentConfig, err := s.composeDeploymentConfig(ctx, deploymentConfigRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose DeploymentConfig with deploymentConfigRaw[%+v], error[%w]", deploymentConfigRaw, err)
	}
	return deploymentConfig, nil
}

//
// private functions
//

func (s *Store) composeDeploymentConfig(ctx context.Context, raw *deploymentConfigRaw) (*api.DeploymentConfig, error) {
	deploymentConfig := raw.toDeploymentConfig()

	creator, err := s.GetPrincipalByID(ctx, deploymentConfig.CreatorID)
	if err != nil {
		return nil, err
	}
	deploymentConfig.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, deploymentConfig.UpdaterID)
	if err != nil {
		return nil, err
	}
	deploymentConfig.Updater = updater

	project, err := s.GetProjectByID(ctx, deploymentConfig.ProjectID)
	if err != nil {
		return nil, err
	}
	deploymentConfig.Project = project

	return deploymentConfig, nil
}

// upsertDeploymentConfigRaw upserts a deployment configuration to a project.
func (s *Store) upsertDeploymentConfigRaw(ctx context.Context, upsert *api.DeploymentConfigUpsert) (*deploymentConfigRaw, error) {
	// Validate the deployment configuration.
	if _, err := api.ValidateAndGetDeploymentSchedule(upsert.Payload); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	cfg, err := s.upsertDeploymentConfigImpl(ctx, tx.PTx, upsert)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return cfg, nil
}

// getDeploymentConfigImpl finds the deployment configuration in a project.
func (s *Store) getDeploymentConfigImpl(ctx context.Context, find *api.DeploymentConfigFind) (*deploymentConfigRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
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
	var ret []*deploymentConfigRaw
	for rows.Next() {
		var cfg deploymentConfigRaw
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

func (s *Store) upsertDeploymentConfigImpl(ctx context.Context, tx *sql.Tx, upsert *api.DeploymentConfigUpsert) (*deploymentConfigRaw, error) {
	if upsert.Payload == "" {
		upsert.Payload = "{}"
	}
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
	var cfg deploymentConfigRaw
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
