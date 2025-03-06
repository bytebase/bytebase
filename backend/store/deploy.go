package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/bytebase/bytebase/backend/common"
)

// DeploymentConfigMessage is the message for deployment config.
type DeploymentConfigMessage struct {
	Name   string
	Config *storepb.DeploymentConfig
}

// GetDeploymentConfigV2 returns the deployment config.
func (s *Store) GetDeploymentConfigV2(ctx context.Context, projectID string) (*DeploymentConfigMessage, error) {
	if v, ok := s.projectDeploymentCache.Get(projectID); ok {
		return v, nil
	}
	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("project = $%d", len(args)+1)), append(args, projectID)

	deploymentConfig := DeploymentConfigMessage{
		Config: &storepb.DeploymentConfig{},
	}
	var configB []byte

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, `
		SELECT
			name,
			config
		FROM deployment_config
		WHERE `+strings.Join(where, " AND "),
		args...,
	).Scan(&deploymentConfig.Name, &configB); err != nil {
		if err == sql.ErrNoRows {
			// Return default deployment config.
			return s.getDefaultDeploymentConfigV2(ctx)
		}
		return nil, errors.Wrapf(err, "failed to scan")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if err := common.ProtojsonUnmarshaler.Unmarshal(configB, deploymentConfig.Config); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal")
	}

	s.projectDeploymentCache.Add(projectID, &deploymentConfig)
	return &deploymentConfig, nil
}

// UpsertDeploymentConfigV2 upserts the deployment config.
func (s *Store) UpsertDeploymentConfigV2(ctx context.Context, projectID string, upsert *DeploymentConfigMessage) (*DeploymentConfigMessage, error) {
	configB, err := protojson.Marshal(upsert.Config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal deployment config")
	}

	query := `
		INSERT INTO deployment_config (
			project,
			name,
			config
		)
		VALUES ($1, $2, $3)
		ON CONFLICT(project) DO UPDATE SET
			name = excluded.name,
			config = excluded.config
		RETURNING name, config
	`
	deploymentConfig := DeploymentConfigMessage{
		Config: &storepb.DeploymentConfig{},
	}
	var newConfigB []byte

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, query,
		projectID,
		upsert.Name,
		configB,
	).Scan(&deploymentConfig.Name, &newConfigB); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if err := common.ProtojsonUnmarshaler.Unmarshal(newConfigB, deploymentConfig.Config); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal")
	}

	s.projectDeploymentCache.Add(projectID, &deploymentConfig)
	return &deploymentConfig, nil
}

func (s *Store) getDefaultDeploymentConfigV2(ctx context.Context) (*DeploymentConfigMessage, error) {
	environmentList, err := s.ListEnvironmentV2(ctx, &FindEnvironmentMessage{})
	if err != nil {
		return nil, err
	}
	var deployments []*storepb.ScheduleDeployment
	for i, environment := range environmentList {
		deployments = append(deployments, &storepb.ScheduleDeployment{
			Title: environment.ResourceID,
			// Use index rather than uuid to ensure consistent Id for the default deployment config.
			Id: fmt.Sprintf("%d", i),
			Spec: &storepb.DeploymentSpec{
				Selector: &storepb.LabelSelector{
					MatchExpressions: []*storepb.LabelSelectorRequirement{
						{Key: "environment", Operator: storepb.LabelSelectorRequirement_IN, Values: []string{environment.ResourceID}},
					},
				},
			},
		})
	}
	return &DeploymentConfigMessage{
		Config: &storepb.DeploymentConfig{
			Schedule: &storepb.Schedule{
				Deployments: deployments,
			},
		},
	}, nil
}
