package api

import (
	"testing"

	"github.com/kr/pretty"
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
			`{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}},{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"Exists"}]}}}]}`,
			&DeploymentSchedule{
				Deployments: []*Deployment{
					{
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
			`{"unmatchdeployments":[{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}},{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"Exists"}]}}}]}`,
			&DeploymentSchedule{},
			false,
		}, {
			"json",
			`{`,
			nil,
			true,
		}, {
			"inOperatorWithNoValue",
			`{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"In"}]}}}]}`,
			nil,
			true,
		}, {
			"existsOperatorWithValues",
			`{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"Exists","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			true,
		}, {
			"invalidOperator",
			`{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod"]},{"key":"location","operator":"invalid"}]}}}]}`,
			nil,
			true,
		}, {
			"missingEnvironment",
			`{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			true,
		}, {
			"environmentExistsOperator",
			`{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"Exists"},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			true,
		}, {
			"environmentMultiValues",
			`{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["prod", "dev"]},{"key":"location","operator":"In","values":["us-central1","europe-west1"]}]}}}]}`,
			nil,
			true,
		},
	}

	for _, test := range tests {
		cfg, err := ValidateAndGetDeploymentSchedule(test.payload)
		if err != nil != test.wantErr {
			t.Errorf("%q: GetDeploymentSchedule(%q) got error %v, wantErr %v.", test.name, test.payload, err, test.wantErr)
		}

		diff := pretty.Diff(cfg, test.wantCfg)
		if len(diff) > 0 {
			t.Errorf("%q: GetDeploymentSchedule(%q) got cfg %+v, want %+v, diff %+v.", test.name, test.payload, cfg, test.wantCfg, diff)
		}
	}
}
