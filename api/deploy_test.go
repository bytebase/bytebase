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
			`{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"location","operator":"IN","values":["us-central1","europe-west1"]}]}}},{"spec":{"selector":{"matchExpressions":[{"key":"location","operator":"EXISTS"}]}}}]}`,
			&DeploymentSchedule{
				Deployments: []*Deployment{
					{
						Spec: &DeploymentSpec{
							Selector: &LabelSelector{
								MatchExpressions: []*LabelSelectorRequirement{
									{
										Key:      "location",
										Operator: "IN",
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
										Key:      "location",
										Operator: "EXISTS",
										Values:   nil,
									},
								},
							},
						},
					},
				},
			},
			false,
		},
		{
			"invalidPayload",
			`{"unmatchdeployments":[{"spec":{"selector":{"matchExpressions":[{"key":"location","operator":"IN","values":["us-central1","europe-west1"]}]}}},{"spec":{"selector":{"matchExpressions":[{"key":"location","operator":"EXISTS"}]}}}]}`,
			&DeploymentSchedule{},
			false,
		},
		{
			"json",
			`{`,
			nil,
			true,
		},
	}

	for _, test := range tests {
		cfg, err := GetDeploymentSchedule(test.payload)
		if err != nil != test.wantErr {
			t.Errorf("%q: GetDeploymentSchedule(%q) got error %v, wantErr %v.", test.name, test.payload, err, test.wantErr)
		}

		diff := pretty.Diff(cfg, test.wantCfg)
		if len(diff) > 0 {
			t.Errorf("%q: GetDeploymentSchedule(%q) got cfg %+v, want %+v, diff %+v.", test.name, test.payload, cfg, test.wantCfg, diff)
		}
	}
}
