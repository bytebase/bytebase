package server

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/kr/pretty"
)

func TestGeneratePipelineCreateFromDeploymentSchedule(t *testing.T) {

	db := []int{0, 1, 2, 3, 4, 5, 6, 7}

	tests := []struct {
		name         string
		schedule     *api.DeploymentSchedule
		databaseList []*api.Database
		want         *api.PipelineCreate
		// Notice relevant position is preserved from databaseList to want.
		// e.g. in simpleDeployments the result is [db[0], db[1]] instead of [db[1], db[0]] in the first stage.
	}{
		{
			"Tenant databases matching the query in a stage should exclude all databases from previous stages.",
			&api.DeploymentSchedule{
				Deployments: []*api.Deployment{
					{
						Spec: &api.DeploymentSpec{
							Selector: &api.LabelSelector{
								MatchExpressions: []*api.LabelSelectorRequirement{
									{
										Key:      "bb.location",
										Operator: "In",
										Values:   []string{"us-central1", "europe-west1"},
									},
								},
							},
						},
					},
					{
						Spec: &api.DeploymentSpec{
							Selector: &api.LabelSelector{
								MatchExpressions: []*api.LabelSelectorRequirement{
									{
										Key:      "bb.location",
										Operator: "Exists",
										Values:   nil,
									},
								},
							},
						},
					},
				},
			},
			[]*api.Database{
				{
					ID:     db[0],
					Labels: "[{\"key\":\"bb.location\",\"value\":\"us-central1\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
				},
				{
					ID:     db[1],
					Labels: "[{\"key\":\"bb.location\",\"value\":\"earth\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
				},
			},
			&api.PipelineCreate{
				StageList: []api.StageCreate{
					{
						TaskList: []api.TaskCreate{
							{
								DatabaseID: &db[0],
							},
						},
					},
					{
						TaskList: []api.TaskCreate{
							{
								DatabaseID: &db[1],
							},
						},
					},
				},
			},
		},
		{
			"simpleDeployments",
			&api.DeploymentSchedule{
				Deployments: []*api.Deployment{
					{
						Spec: &api.DeploymentSpec{
							Selector: &api.LabelSelector{
								MatchExpressions: []*api.LabelSelectorRequirement{
									{
										Key:      "bb.location",
										Operator: "In",
										Values:   []string{"us-central1", "europe-west1"},
									},
								},
							},
						},
					},
					{
						Spec: &api.DeploymentSpec{
							Selector: &api.LabelSelector{
								MatchExpressions: []*api.LabelSelectorRequirement{
									{
										Key:      "bb.location",
										Operator: "In",
										Values:   []string{"earth"},
									},
								},
							},
						},
					},
				},
			},
			[]*api.Database{
				{
					ID:     db[0],
					Labels: "[{\"key\":\"bb.location\",\"value\":\"us-central1\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
				},
				{
					ID:     db[1],
					Labels: "[{\"key\":\"bb.location\",\"value\":\"europe-west1\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
				},
				{
					ID:     db[2],
					Labels: "[{\"key\":\"bb.location\",\"value\":\"earth\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
				},
			},
			&api.PipelineCreate{
				StageList: []api.StageCreate{
					{
						TaskList: []api.TaskCreate{
							{
								DatabaseID: &db[0],
							},
							{
								DatabaseID: &db[1],
							},
						},
					},
					{
						TaskList: []api.TaskCreate{
							{
								DatabaseID: &db[2],
							},
						},
					},
				},
			},
		},
		{
			"twoDifferentKeys",
			&api.DeploymentSchedule{
				Deployments: []*api.Deployment{

					{
						Spec: &api.DeploymentSpec{
							Selector: &api.LabelSelector{
								MatchExpressions: []*api.LabelSelectorRequirement{
									{
										Key:      "bb.tenant",
										Operator: "In",
										Values:   []string{"bytebase"},
									},
								},
							},
						},
					},
					{
						Spec: &api.DeploymentSpec{
							Selector: &api.LabelSelector{
								MatchExpressions: []*api.LabelSelectorRequirement{
									{
										Key:      "bb.location",
										Operator: "In",
										Values:   []string{"us-central1", "europe-west1"},
									},
								},
							},
						},
					},
				},
			},
			[]*api.Database{
				{
					ID:     db[0],
					Labels: "[{\"key\":\"bb.location\",\"value\":\"us-central1\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
				},
				{
					ID:     db[1],
					Labels: "[{\"key\":\"bb.location\",\"value\":\"europe-west1\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
				},
				{
					ID:     db[2],
					Labels: "[{\"key\":\"bb.location\",\"value\":\"earth\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
				},
			},
			&api.PipelineCreate{
				StageList: []api.StageCreate{
					{
						TaskList: []api.TaskCreate{
							{
								DatabaseID: &db[0],
							},
							{
								DatabaseID: &db[1],
							},
						},
					},
					{
						TaskList: []api.TaskCreate{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		create := generatePipelineCreateFromDeploymentSchedule(test.schedule, test.databaseList)

		diff := pretty.Diff(create, test.want)
		if len(diff) > 0 {
			t.Errorf("%q: GetDeploymentSchedule() got create %+v, want %+v, diff %+v.", test.name, create, test.want, diff)
		}
	}
}
