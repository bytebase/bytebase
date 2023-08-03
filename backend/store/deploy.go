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
	var deployments []*api.Deployment
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
	// Return the default deployment config if a project is not in tenant mode any more.
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{UID: &projectUID})
	if err != nil {
		return nil, err
	}
	if project.TenantMode != api.TenantModeTenant {
		return s.getDefaultDeploymentConfigV2(ctx)
	}

	if deploymentConfig, ok := s.projectIDDeploymentConfigCache.Load(projectUID); ok {
		return deploymentConfig.(*DeploymentConfigMessage), nil
	}
	where, args := []string{"TRUE"}, []any{}
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
			return s.getDefaultDeploymentConfigV2(ctx)
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	var schedule Schedule
	if err := json.Unmarshal([]byte(payload), &schedule); err != nil {
		return nil, err
	}
	deploymentConfig.Schedule = &schedule

	s.projectIDDeploymentConfigCache.Store(projectUID, &deploymentConfig)
	return &deploymentConfig, nil
}

// UpsertDeploymentConfigV2 upserts the deployment config.
func (s *Store) UpsertDeploymentConfigV2(ctx context.Context, projectUID, principalUID int, upsert *DeploymentConfigMessage) (*DeploymentConfigMessage, error) {
	payload, err := json.Marshal(upsert.Schedule)
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
		principalUID,
		principalUID,
		projectUID,
		upsert.Name,
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

	s.projectIDDeploymentConfigCache.Store(projectUID, &deploymentConfig)
	return &deploymentConfig, nil
}

func (s *Store) getDefaultDeploymentConfigV2(ctx context.Context) (*DeploymentConfigMessage, error) {
	environmentList, err := s.ListEnvironmentV2(ctx, &FindEnvironmentMessage{})
	if err != nil {
		return nil, err
	}
	scheduleList := &Schedule{}
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
		Schedule: scheduleList,
	}, nil
}
