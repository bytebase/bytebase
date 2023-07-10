package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

func TestGetDatabaseMatrixFromDeploymentSchedule(t *testing.T) {
	dbs := []*store.DatabaseMessage{
		{
			UID:          0,
			DatabaseName: "hello",
			Labels: map[string]string{
				"bb.location":    "us-central1",
				"bb.tenant":      "bytebase",
				"bb.environment": "dev",
			},
		},
		{
			UID:          1,
			DatabaseName: "hello",
			Labels: map[string]string{
				"bb.location":    "earth",
				"bb.tenant":      "bytebase",
				"bb.environment": "dev",
			},
		},
		{
			UID:          2,
			DatabaseName: "hello",
			Labels: map[string]string{
				"bb.location":    "europe-west1",
				"bb.tenant":      "bytebase",
				"bb.environment": "dev",
			},
		},
		{
			UID:          3,
			DatabaseName: "hello",
			Labels: map[string]string{
				"bb.location":    "earth",
				"bb.environment": "dev",
			},
		},
		{
			UID:          4,
			DatabaseName: "world",
			Labels: map[string]string{
				"bb.location":    "earth",
				"bb.environment": "dev",
			},
		},
		{
			UID:          5,
			DatabaseName: "db1_us",
			Labels: map[string]string{
				"bb.location":    "us",
				"bb.environment": "dev",
			},
		},
		{
			UID:          6,
			DatabaseName: "db1_eu",
			Labels: map[string]string{
				"bb.location":    "eu",
				"bb.environment": "dev",
			},
		},
	}

	tests := []struct {
		name         string
		schedule     *api.DeploymentSchedule
		databaseList []*store.DatabaseMessage
		want         [][]*store.DatabaseMessage
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
			[]*store.DatabaseMessage{
				dbs[0], dbs[1],
			},
			[][]*store.DatabaseMessage{
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
			[]*store.DatabaseMessage{
				dbs[0], dbs[1], dbs[2],
			},
			[][]*store.DatabaseMessage{
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
			[]*store.DatabaseMessage{
				dbs[0], dbs[2], dbs[3],
			},
			[][]*store.DatabaseMessage{
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
			[]*store.DatabaseMessage{
				dbs[3], dbs[4],
			},
			[][]*store.DatabaseMessage{
				{dbs[3], dbs[4]},
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
			[]*store.DatabaseMessage{
				dbs[5], dbs[6],
			},
			[][]*store.DatabaseMessage{
				{dbs[5], dbs[6]},
			},
		},
	}

	for _, test := range tests {
		matrix, _ := GetDatabaseMatrixFromDeploymentSchedule(test.schedule, test.databaseList)
		assert.Equal(t, matrix, test.want, test.name)
	}
}

func TestMergeTaskCreateLists(t *testing.T) {
	tests := []struct {
		name               string
		taskCreateLists    [][]store.TaskMessage
		taskIndexDAGLists  [][]api.TaskIndexDAG
		wantTaskCreateList []store.TaskMessage
		wantTaskDAGList    []api.TaskIndexDAG
	}{
		{
			name: "simple, len=1",
			taskCreateLists: [][]store.TaskMessage{
				{
					{}, {},
				},
			},
			taskIndexDAGLists: [][]api.TaskIndexDAG{
				{
					{FromIndex: 0, ToIndex: 1},
				},
			},
			wantTaskCreateList: []store.TaskMessage{
				{}, {},
			},
			wantTaskDAGList: []api.TaskIndexDAG{
				{FromIndex: 0, ToIndex: 1},
			},
		},
		{
			name: "len=2",
			taskCreateLists: [][]store.TaskMessage{
				{
					{}, {}, {}, {},
				},
				{
					{}, {}, {}, {},
				},
			},
			taskIndexDAGLists: [][]api.TaskIndexDAG{
				{
					{FromIndex: 0, ToIndex: 1},
					{FromIndex: 1, ToIndex: 3},
				},
				{
					{FromIndex: 1, ToIndex: 2},
				},
			},
			wantTaskCreateList: []store.TaskMessage{
				{}, {}, {}, {}, {}, {}, {}, {},
			},
			wantTaskDAGList: []api.TaskIndexDAG{
				{FromIndex: 0, ToIndex: 1},
				{FromIndex: 1, ToIndex: 3},
				{FromIndex: 5, ToIndex: 6},
			},
		},
		{
			name: "len=3",
			taskCreateLists: [][]store.TaskMessage{
				{
					{}, {}, {}, {},
				},
				{
					{}, {}, {}, {},
				},
				{
					{}, {}, {}, {},
				},
			},
			taskIndexDAGLists: [][]api.TaskIndexDAG{
				{
					{FromIndex: 0, ToIndex: 1},
					{FromIndex: 1, ToIndex: 3},
				},
				{
					{FromIndex: 1, ToIndex: 2},
				},
				{
					{FromIndex: 1, ToIndex: 2},
				},
			},
			wantTaskCreateList: []store.TaskMessage{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			wantTaskDAGList: []api.TaskIndexDAG{
				{FromIndex: 0, ToIndex: 1},
				{FromIndex: 1, ToIndex: 3},
				{FromIndex: 5, ToIndex: 6},
				{FromIndex: 9, ToIndex: 10},
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			a := require.New(t)
			taskCreateList, taskIndexDAGList, err := MergeTaskCreateLists(test.taskCreateLists, test.taskIndexDAGLists)
			a.NoError(err)
			a.Equal(test.wantTaskCreateList, taskCreateList)
			a.Equal(test.wantTaskDAGList, taskIndexDAGList)
		})
	}
}

func TestGetRenderedStatement(t *testing.T) {
	testCases := []struct {
		material map[string]string
		template string
		expected string
	}{
		{
			material: map[string]string{
				"PASSWORD": "123",
			},
			template: "select * from table where password = ${{ secrets.PASSWORD }}",
			expected: "select * from table where password = 123",
		},
		{
			material: map[string]string{
				"PASSWORD": "123",
				"USER":     `"abc"`,
			},
			template: "INSERT INTO table (user, password) VALUES (${{ secrets.USER }}, ${{ secrets.PASSWORD }})",
			expected: `INSERT INTO table (user, password) VALUES ("abc", 123)`,
		},
		// Replace recursively case.
		{
			material: map[string]string{
				"PASSWORD": "${{ secrets.USER }}",
				"USER":     `"abc"`,
			},
			template: "INSERT INTO table (user, password) VALUES (${{ secrets.USER }}, ${{ secrets.PASSWORD }})",
			expected: `INSERT INTO table (user, password) VALUES ("abc", ${{ secrets.USER }})`,
		},
		{
			material: map[string]string{
				"PASSWORD": "123",
				"USER":     `"${{ secrets.PASSWORD }}"`,
			},
			template: "INSERT INTO table (user, password) VALUES (${{ secrets.USER }}, ${{ secrets.PASSWORD }})",
			expected: `INSERT INTO table (user, password) VALUES ("123", 123)`,
		},
		{
			material: map[string]string{
				"USER": `"abc"`,
			},
			template: "select * from table where password = ${{ secrets.PASSWORD }}",
			expected: "select * from table where password = ${{ secrets.PASSWORD }}",
		},
	}

	for _, tc := range testCases {
		actual := RenderStatement(tc.template, tc.material)
		assert.Equal(t, tc.expected, actual)
	}
}
