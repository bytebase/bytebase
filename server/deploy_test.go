package server

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/kr/pretty"
)

func TestGeneratePipelineCreateFromDeploymentSchedule(t *testing.T) {

	db := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}

	tests := []struct {
		name      string
		schedule  *api.DeploymentSchedule
		labelList []*api.DatabaseLabel
		want      *api.PipelineCreate
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
			[]*api.DatabaseLabel{
				{
					DatabaseID: db[0],
					Key:        "bb.location",
					Value:      "earth",
				},
				{
					DatabaseID: db[1],
					Key:        "bb.location",
					Value:      "us-central1",
				},
			},
			&api.PipelineCreate{
				StageList: []api.StageCreate{
					{},
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
			[]*api.DatabaseLabel{
				{
					DatabaseID: db[0],
					Key:        "bb.location",
					Value:      "earth",
				},
				{
					DatabaseID: db[1],
					Key:        "bb.location",
					Value:      "us-central1",
				},
				{
					DatabaseID: db[2],
					Key:        "bb.location",
					Value:      "europe-west1",
				},
				{
					DatabaseID: db[0],
					Key:        "bb.tenant",
					Value:      "bytebase",
				},
			},
			&api.PipelineCreate{
				StageList: []api.StageCreate{
					{
						TaskList: []api.TaskCreate{
							{
								DatabaseID: &db[1],
							},
							{
								DatabaseID: &db[2],
							},
						},
					},
					{
						TaskList: []api.TaskCreate{
							{
								DatabaseID: &db[0],
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
			[]*api.DatabaseLabel{
				{
					DatabaseID: db[0],
					Key:        "bb.location",
					Value:      "earth",
				},
				{
					DatabaseID: db[1],
					Key:        "bb.location",
					Value:      "us-central1",
				},
				{
					DatabaseID: db[2],
					Key:        "bb.location",
					Value:      "europe-west1",
				},
				{
					DatabaseID: db[0],
					Key:        "bb.tenant",
					Value:      "bytebase",
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
							{
								DatabaseID: &db[2],
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		create := generatePipelineCreateFromDeploymentSchedule(test.schedule, test.labelList)

		diff := pretty.Diff(create, test.want)
		if len(diff) > 0 {
			t.Errorf("%q: GetDeploymentSchedule() got create %+v, want %+v, diff %+v.", test.name, create, test.want, diff)
		}
	}
}
