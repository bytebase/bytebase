package store

import (
	"context"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DeploymentConfigMessage is the message for deployment config.
type DeploymentConfigMessage struct {
	Name   string
	Config *storepb.DeploymentConfig
}

// GetDeploymentConfigV2 returns the deployment config.
func (s *Store) GetDeploymentConfigV2(ctx context.Context, _ string) (*DeploymentConfigMessage, error) {
	return s.getDefaultDeploymentConfigV2(ctx)
}

func (s *Store) getDefaultDeploymentConfigV2(ctx context.Context) (*DeploymentConfigMessage, error) {
	environmentList, err := s.ListEnvironmentV2(ctx, &FindEnvironmentMessage{})
	if err != nil {
		return nil, err
	}
	var deployments []*storepb.ScheduleDeployment
	for _, environment := range environmentList {
		deployments = append(deployments, &storepb.ScheduleDeployment{
			Title: environment.ResourceID,
			// Use environment resource id rather than uuid to ensure consistent Id for the default deployment config.
			Id: environment.ResourceID,
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
