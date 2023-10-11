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
		errPart string
	}{
		{
			"complexDeployments",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"environment","operator":"In","values":["prod"]},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}},{"name":"deployment2","spec":{"selector":{"matchExpressions":[{"key":"environment","operator":"In","values":["prod"]},{"key":"location","operator":"Exists"}]}}}]}`,
			&DeploymentSchedule{
				Deployments: []*Deployment{
					{
						Name: "deployment1",
						Spec: &DeploymentSpec{
							Selector: &LabelSelector{
								MatchExpressions: []*LabelSelectorRequirement{
									{
										Key:      "environment",
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
										Key:      "environment",
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
			"",
		}, {
			"invalidPayload",
			`{"unmatchdeployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"environment","operator":"In","values":["prod"]},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}},{"name":"deployment2","spec":{"selector":{"matchExpressions":[{"key":"environment","operator":"In","values":["prod"]},{"key":"location","operator":"Exists"}]}}}]}`,
			&DeploymentSchedule{},
			"",
		}, {
			"json",
			`{`,
			nil,
			"unexpected end of JSON input",
		}, {
			"inOperatorWithNoValue",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"environment","operator":"In","values":["prod"]},{"key":"location","operator":"In"}]}}}]}`,
			nil,
			"operator should have at least one value",
		}, {
			"existsOperatorWithValues",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"environment","operator":"In","values":["prod"]},{"key":"location","operator":"Exists","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			"operator shouldn't have values",
		}, {
			"invalidOperator",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"environment","operator":"In","values":["prod"]},{"key":"location","operator":"invalid"}]}}}]}`,
			nil,
			"has invalid operator",
		}, {
			"missingEnvironment",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			"label",
		}, {
			"environmentExistsOperator",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"environment","operator":"Exists"},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			"should must use operator",
		}, {
			"environmentMultiValues",
			`{"deployments":[{"name":"deployment1","spec":{"selector":{"matchExpressions":[{"key":"environment","operator":"In","values":["prod", "dev"]},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			"should must use operator",
		},
	}

	for _, test := range tests {
		cfg, err := ValidateAndGetDeploymentSchedule(test.payload)
		if test.errPart == "" {
			require.NoError(t, err)
		} else {
			require.Contains(t, err.Error(), test.errPart, test.name)
		}
		require.Equal(t, cfg, test.wantCfg)
	}
}
