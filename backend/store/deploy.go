package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// deploymentConfigRaw is the store model for an DeploymentConfig.
// Fields have exactly the same meanings as DeploymentConfig.
type deploymentConfigRaw struct {
	ID int

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

		// Domain specific fields
		Name:    raw.Name,
		Payload: raw.Payload,
	}
}

// UpsertDeploymentConfig upserts an instance of DeploymentConfig.
func (s *Store) UpsertDeploymentConfig(ctx context.Context, upsert *api.DeploymentConfigUpsert) (*api.DeploymentConfig, error) {
	deploymentConfigRaw, err := s.upsertDeploymentConfigRaw(ctx, upsert)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to upsert deployment config with DeploymentConfigUpsert[%+v]", upsert)
	}
	deploymentConfig, err := s.composeDeploymentConfig(ctx, deploymentConfigRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose DeploymentConfig with deploymentConfigRaw[%+v]", deploymentConfigRaw)
	}
	return deploymentConfig, nil
}

//
// private functions
//

func (s *Store) composeDeploymentConfig(ctx context.Context, raw *deploymentConfigRaw) (*api.DeploymentConfig, error) {
	deploymentConfig := raw.toDeploymentConfig()
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
	defer tx.Rollback()

	cfg, err := s.upsertDeploymentConfigImpl(ctx, tx, upsert)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return cfg, nil
}

func (*Store) upsertDeploymentConfigImpl(ctx context.Context, tx *Tx, upsert *api.DeploymentConfigUpsert) (*deploymentConfigRaw, error) {
	if upsert.Payload == "" {
		upsert.Payload = "{}"
	}
	query := `
		INSERT INTO deployment_config (
			creator_id,
			updater_id,
			project_id,
			name,
			config
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(project_id) DO UPDATE SET
			updater_id = excluded.updater_id,
			name = excluded.name,
			config = excluded.config
		RETURNING id, project_id, name, config
	`
	var cfg deploymentConfigRaw
	if err := tx.QueryRowContext(ctx, query,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.ProjectID,
		upsert.Name,
		upsert.Payload,
	).Scan(
		&cfg.ID,
		&cfg.ProjectID,
		&cfg.Name,
		&cfg.Payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	return &cfg, nil
}

// UpsertDeploymentConfigMessage is the message to upsert a deployment config.
type UpsertDeploymentConfigMessage struct {
	ProjectUID       int
	PrincipalUID     int
	DeploymentConfig *DeploymentConfigMessage
}

// DeploymentConfigMessage is the message for deployment config.
type DeploymentConfigMessage struct {
	Name     string
	Schedule *Schedule

	// Output only fields.
	// ID is the ID of the deployment config.
	UID int
}

// ToAPIDeploymentConfig converts the message to a legacy API deployment config.
func (d *DeploymentConfigMessage) ToAPIDeploymentConfig() (*api.DeploymentConfig, error) {
	schedule := d.Schedule.toAPIDeploymentSchedule()
	payload, err := json.Marshal(schedule)
	if err != nil {
		return nil, err
	}
	return &api.DeploymentConfig{
		ID:      d.UID,
		Name:    d.Name,
		Payload: string(payload),
	}, nil
}

// Schedule is the message for deployment schedule.
type Schedule struct {
	Deployments []*Deployment `json:"deployments"`
}

func (s *Schedule) toAPIDeploymentSchedule() *api.DeploymentSchedule {
	deployments := []*api.Deployment{}
	for _, d := range s.Deployments {
		deployments = append(deployments, &api.Deployment{
			Name: d.Name,
			Spec: &api.DeploymentSpec{
				Selector: d.Spec.Selector.toAPILabelSelector(),
			},
		})
	}
	return &api.DeploymentSchedule{
		Deployments: deployments,
	}
}

// Deployment is the message for deployment.
type Deployment struct {
	Name string          `json:"name"`
	Spec *DeploymentSpec `json:"spec"`
}

// DeploymentSpec is the message for deployment specification.
type DeploymentSpec struct {
	Selector *LabelSelector `json:"selector"`
}

// LabelSelector is the message for label selector.
type LabelSelector struct {
	// MatchExpressions is a list of label selector requirements. The requirements are ANDed.
	MatchExpressions []*LabelSelectorRequirement `json:"matchExpressions"`
}

func (ls *LabelSelector) toAPILabelSelector() *api.LabelSelector {
	labelSelector := &api.LabelSelector{}
	for _, r := range ls.MatchExpressions {
		operatorTp := api.InOperatorType
		switch r.Operator {
		case InOperatorType:
			operatorTp = api.InOperatorType
		case ExistsOperatorType:
			operatorTp = api.ExistsOperatorType
		}
		labelSelector.MatchExpressions = append(labelSelector.MatchExpressions, &api.LabelSelectorRequirement{
			Key:      r.Key,
			Operator: operatorTp,
			Values:   r.Values,
		})
	}
	return labelSelector
}

// OperatorType is the type of label selector requirement operator.
// Valid operators are In, Exists.
// Note: NotIn and DoesNotExist are not supported initially.
type OperatorType string

const (
	// InOperatorType is the operator type for In.
	InOperatorType OperatorType = "In"
	// ExistsOperatorType is the operator type for Exists.
	ExistsOperatorType OperatorType = "Exists"
)

// LabelSelectorRequirement is the message for label selector.
type LabelSelectorRequirement struct {
	// Key is the label key that the selector applies to.
	Key string `json:"key"`

	// Operator represents a key's relationship to a set of values.
	Operator OperatorType `json:"operator"`

	// Values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
	Values []string `json:"values"`
}

// GetDeploymentConfigV2 returns the deployment config.
func (s *Store) GetDeploymentConfigV2(ctx context.Context, projectUID int) (*DeploymentConfigMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, projectUID)

	var deploymentConfig DeploymentConfigMessage
	var payload string

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, `
		SELECT
			id,
			name,
			config
		FROM deployment_config
		WHERE `+strings.Join(where, " AND "),
		args...,
	).Scan(&deploymentConfig.UID, &deploymentConfig.Name, &payload); err != nil {
		if err == sql.ErrNoRows {
			// Return default deployment config.
			return s.getDefaultDeploymentConfigV2(ctx, projectUID)
		}
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	var schedule Schedule
	if err := json.Unmarshal([]byte(payload), &schedule); err != nil {
		return nil, err
	}
	deploymentConfig.Schedule = &schedule

	return &deploymentConfig, nil
}

// UpsertDeploymentConfigV2 upserts the deployment config.
func (s *Store) UpsertDeploymentConfigV2(ctx context.Context, upsert *UpsertDeploymentConfigMessage) (*DeploymentConfigMessage, error) {
	payload, err := json.Marshal(upsert.DeploymentConfig.Schedule)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal deployment config")
	}

	query := `
		INSERT INTO deployment_config (
			creator_id,
			updater_id,
			project_id,
			name,
			config
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(project_id) DO UPDATE SET
			updater_id = excluded.updater_id,
			name = excluded.name,
			config = excluded.config
		RETURNING id, name, config
	`
	var deploymentConfig DeploymentConfigMessage

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, query,
		upsert.PrincipalUID,
		upsert.PrincipalUID,
		upsert.ProjectUID,
		upsert.DeploymentConfig.Name,
		payload,
	).Scan(&deploymentConfig.UID, &deploymentConfig.Name, &payload); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if err := json.Unmarshal([]byte(payload), &deploymentConfig.Schedule); err != nil {
		return nil, err
	}
	return &deploymentConfig, nil
}

func (s *Store) getDefaultDeploymentConfigV2(ctx context.Context, projectID int) (*DeploymentConfigMessage, error) {
	environmentList, err := s.ListEnvironmentV2(ctx, &FindEnvironmentMessage{})
	if err != nil {
		return nil, err
	}
	scheduleList := Schedule{}
	for _, environment := range environmentList {
		scheduleList.Deployments = append(scheduleList.Deployments, &Deployment{
			Name: fmt.Sprintf("%s Stage", environment.Title),
			Spec: &DeploymentSpec{
				Selector: &LabelSelector{
					MatchExpressions: []*LabelSelectorRequirement{
						{Key: "bb.environment", Operator: InOperatorType, Values: []string{environment.ResourceID}},
					},
				},
			},
		})
	}
	return &DeploymentConfigMessage{
		UID:      0,
		Schedule: &scheduleList,
	}, nil
}
