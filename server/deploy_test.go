package server

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/api"
)

func TestGetDatabaseMatrixFromDeploymentSchedule(t *testing.T) {
	dbs := []*api.Database{
		{
			ID:     0,
			Name:   "hello",
			Labels: "[{\"key\":\"bb.location\",\"value\":\"us-central1\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
		},
		{
			ID:     1,
			Name:   "hello",
			Labels: "[{\"key\":\"bb.location\",\"value\":\"earth\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
		},
		{
			ID:     2,
			Name:   "hello",
			Labels: "[{\"key\":\"bb.location\",\"value\":\"europe-west1\"},{\"key\":\"bb.tenant\",\"value\":\"bytebase\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
		},
		{
			ID:     3,
			Name:   "hello",
			Labels: "[{\"key\":\"bb.location\",\"value\":\"earth\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
		},
		{
			ID:     4,
			Name:   "world",
			Labels: "[{\"key\":\"bb.location\",\"value\":\"earth\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
		},
		{
			ID:     5,
			Name:   "db1_us",
			Labels: "[{\"key\":\"bb.location\",\"value\":\"us\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
		},
		{
			ID:     6,
			Name:   "db1_eu",
			Labels: "[{\"key\":\"bb.location\",\"value\":\"eu\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
		},
	}

	tests := []struct {
		name                 string
		schedule             *api.DeploymentSchedule
		baseDatabaseName     string
		databaseNameTemplate string
		databaseList         []*api.Database
		want                 [][]*api.Database
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
			"hello",
			"{{DB_NAME}}",
			[]*api.Database{
				dbs[0], dbs[1],
			},
			[][]*api.Database{
				{dbs[0]},
				{dbs[1]},
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
			"hello",
			"{{DB_NAME}}",
			[]*api.Database{
				dbs[0], dbs[1], dbs[2],
			},
			[][]*api.Database{
				{dbs[0], dbs[2]},
				{dbs[1]},
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
			"hello",
			"{{DB_NAME}}",
			[]*api.Database{
				dbs[0], dbs[2], dbs[3],
			},
			[][]*api.Database{
				{dbs[0], dbs[2]},
				nil,
			},
		},
		{
			"differentDatabaseNames",
			&api.DeploymentSchedule{
				Deployments: []*api.Deployment{

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
			"world",
			"{{DB_NAME}}",
			[]*api.Database{
				dbs[3], dbs[4],
			},
			[][]*api.Database{
				{dbs[4]},
			},
		},
		{
			"useDatabaseNameTemplate",
			&api.DeploymentSchedule{
				Deployments: []*api.Deployment{

					{
						Spec: &api.DeploymentSpec{
							Selector: &api.LabelSelector{
								MatchExpressions: []*api.LabelSelectorRequirement{
									{
										Key:      "bb.location",
										Operator: "In",
										Values:   []string{"us", "eu"},
									},
								},
							},
						},
					},
				},
			},
			"db1",
			"{{DB_NAME}}_{{LOCATION}}",
			[]*api.Database{
				dbs[5], dbs[6],
			},
			[][]*api.Database{
				{dbs[5], dbs[6]},
			},
		},
	}

	for _, test := range tests {
		matrix, _ := getDatabaseMatrixFromDeploymentSchedule(test.schedule, test.baseDatabaseName, test.databaseNameTemplate, test.databaseList)
		fmt.Printf("Barny1: %d\n", len(matrix))
		fmt.Printf("Barny2: %d\n", len(test.want))
		assert.EqualValues(t, matrix, test.want, test.name)
	}
}
