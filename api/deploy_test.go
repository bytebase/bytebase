package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDeploymentSchedule(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantCfg *DeploymentSchedule
		wantErr bool
	}{
		{
			"complexDeployments",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}},{"name":"deployment2","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"Exists"}]}}}]}`,
			&DeploymentSchedule{
				Deployments: []*Deployment{
					{
						Name: "deployment1",
						Spec: &DeploymentSpec{
							Selector: &LabelSelector{
								MatchExpressions: []*LabelSelectorRequirement{
									{
										Key:      "bb.environment",
										Operator: "In",
										Values:   []string{"prod"},
									}, {
										Key:      "location",
										Operator: "In",
										Values:   []string{"us-central1", "europe-west1"},
									},
								},
							},
						},
					},
					{
						Name: "deployment2",
						Spec: &DeploymentSpec{
							Selector: &LabelSelector{
								MatchExpressions: []*LabelSelectorRequirement{
									{
										Key:      "bb.environment",
										Operator: "In",
										Values:   []string{"prod"},
									}, {
										Key:      "location",
										Operator: "Exists",
										Values:   nil,
									},
								},
							},
						},
					},
				},
			},
			false,
		}, {
			"invalidPayload",
			`{"unmatchdeployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}},{"name":"deployment2","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"Exists"}]}}}]}`,
			&DeploymentSchedule{},
			false,
		}, {
			"json",
			`{`,
			nil,
			true,
		}, {
			"inOperatorWithNoValue",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"In"}]}}}]}`,
			nil,
			true,
		}, {
			"existsOperatorWithValues",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"Exists","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			true,
		}, {
			"invalidOperator",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"invalid"}]}}}]}`,
			nil,
			true,
		}, {
			"missingEnvironment",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			true,
		}, {
			"environmentExistsOperator",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"Exists"},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			true,
		}, {
			"environmentMultiValues",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod", "dev"]},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			true,
		},
	}

	for _, test := range tests {
		cfg, err := ValidateAndGetDeploymentSchedule(test.payload)
		require.False(t, err != nil != test.wantErr)
		require.Equal(t, cfg, test.wantCfg)
	}
}
