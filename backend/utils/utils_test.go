package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestGetDatabaseMatrixFromDeploymentSchedule(t *testing.T) {
	dbs := []*store.DatabaseMessage{
		{
			InstanceID:   "instance1",
			DatabaseName: "hello",
			Metadata: &storepb.DatabaseMetadata{
				Labels: map[string]string{
					"location":    "us-central1",
					"tenant":      "bytebase",
					"environment": "dev",
				},
			},
		},
		{
			InstanceID:   "instance2",
			DatabaseName: "hello",
			Metadata: &storepb.DatabaseMetadata{
				Labels: map[string]string{
					"location":    "earth",
					"tenant":      "bytebase",
					"environment": "dev",
				},
			},
		},
		{
			InstanceID:   "instance3",
			DatabaseName: "hello",
			Metadata: &storepb.DatabaseMetadata{
				Labels: map[string]string{
					"location":    "europe-west1",
					"tenant":      "bytebase",
					"environment": "dev",
				},
			},
		},
		{
			InstanceID:   "instance4",
			DatabaseName: "hello",
			Metadata: &storepb.DatabaseMetadata{
				Labels: map[string]string{
					"location":    "earth",
					"environment": "dev",
				},
			},
		},
		{
			InstanceID:   "instance5",
			DatabaseName: "world",
			Metadata: &storepb.DatabaseMetadata{
				Labels: map[string]string{
					"location":    "earth",
					"environment": "dev",
				},
			},
		},
		{
			InstanceID:   "instance6",
			DatabaseName: "db1_us",
			Metadata: &storepb.DatabaseMetadata{
				Labels: map[string]string{
					"location":    "us",
					"environment": "dev",
				},
			},
		},
		{
			InstanceID:   "instance7",
			DatabaseName: "db1_eu",
			Metadata: &storepb.DatabaseMetadata{
				Labels: map[string]string{
					"location":    "eu",
					"environment": "dev",
				},
			},
		},
	}

	tests := []struct {
		name         string
		schedule     *storepb.Schedule
		databaseList []*store.DatabaseMessage
		want         [][]*store.DatabaseMessage
		// Notice relevant position is preserved from databaseList to want.
		// e.g. in simpleDeployments the result is [db[0], db[1]] instead of [db[1], db[0]] in the first stage.
	}{
		{
			"Tenant databases matching the query in a stage should exclude all databases from previous stages.",
			&storepb.Schedule{
				Deployments: []*storepb.ScheduleDeployment{
					{
						Spec: &storepb.DeploymentSpec{
							Selector: &storepb.LabelSelector{
								MatchExpressions: []*storepb.LabelSelectorRequirement{
									{
										Key:      "location",
										Operator: storepb.LabelSelectorRequirement_IN,
										Values:   []string{"us-central1", "europe-west1"},
									},
								},
							},
						},
					},
					{
						Spec: &storepb.DeploymentSpec{
							Selector: &storepb.LabelSelector{
								MatchExpressions: []*storepb.LabelSelectorRequirement{
									{
										Key:      "location",
										Operator: storepb.LabelSelectorRequirement_EXISTS,
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
			&storepb.Schedule{
				Deployments: []*storepb.ScheduleDeployment{
					{
						Spec: &storepb.DeploymentSpec{
							Selector: &storepb.LabelSelector{
								MatchExpressions: []*storepb.LabelSelectorRequirement{
									{
										Key:      "location",
										Operator: storepb.LabelSelectorRequirement_IN,
										Values:   []string{"us-central1", "europe-west1"},
									},
								},
							},
						},
					},
					{
						Spec: &storepb.DeploymentSpec{
							Selector: &storepb.LabelSelector{
								MatchExpressions: []*storepb.LabelSelectorRequirement{
									{
										Key:      "location",
										Operator: storepb.LabelSelectorRequirement_IN,
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
			&storepb.Schedule{
				Deployments: []*storepb.ScheduleDeployment{
					{
						Spec: &storepb.DeploymentSpec{
							Selector: &storepb.LabelSelector{
								MatchExpressions: []*storepb.LabelSelectorRequirement{
									{
										Key:      "tenant",
										Operator: storepb.LabelSelectorRequirement_IN,
										Values:   []string{"bytebase"},
									},
								},
							},
						},
					},
					{
						Spec: &storepb.DeploymentSpec{
							Selector: &storepb.LabelSelector{
								MatchExpressions: []*storepb.LabelSelectorRequirement{
									{
										Key:      "location",
										Operator: storepb.LabelSelectorRequirement_IN,
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
			&storepb.Schedule{
				Deployments: []*storepb.ScheduleDeployment{
					{
						Spec: &storepb.DeploymentSpec{
							Selector: &storepb.LabelSelector{
								MatchExpressions: []*storepb.LabelSelectorRequirement{
									{
										Key:      "location",
										Operator: storepb.LabelSelectorRequirement_IN,
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
			&storepb.Schedule{
				Deployments: []*storepb.ScheduleDeployment{
					{
						Spec: &storepb.DeploymentSpec{
							Selector: &storepb.LabelSelector{
								MatchExpressions: []*storepb.LabelSelectorRequirement{
									{
										Key:      "location",
										Operator: storepb.LabelSelectorRequirement_IN,
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
		taskCreateLists    [][]*store.TaskMessage
		taskIndexDAGLists  [][]store.TaskIndexDAG
		wantTaskCreateList []*store.TaskMessage
		wantTaskDAGList    []store.TaskIndexDAG
	}{
		{
			name: "simple, len=1",
			taskCreateLists: [][]*store.TaskMessage{
				{
					{}, {},
				},
			},
			taskIndexDAGLists: [][]store.TaskIndexDAG{
				{
					{FromIndex: 0, ToIndex: 1},
				},
			},
			wantTaskCreateList: []*store.TaskMessage{
				{}, {},
			},
			wantTaskDAGList: []store.TaskIndexDAG{
				{FromIndex: 0, ToIndex: 1},
			},
		},
		{
			name: "len=2",
			taskCreateLists: [][]*store.TaskMessage{
				{
					{}, {}, {}, {},
				},
				{
					{}, {}, {}, {},
				},
			},
			taskIndexDAGLists: [][]store.TaskIndexDAG{
				{
					{FromIndex: 0, ToIndex: 1},
					{FromIndex: 1, ToIndex: 3},
				},
				{
					{FromIndex: 1, ToIndex: 2},
				},
			},
			wantTaskCreateList: []*store.TaskMessage{
				{}, {}, {}, {}, {}, {}, {}, {},
			},
			wantTaskDAGList: []store.TaskIndexDAG{
				{FromIndex: 0, ToIndex: 1},
				{FromIndex: 1, ToIndex: 3},
				{FromIndex: 5, ToIndex: 6},
			},
		},
		{
			name: "len=3",
			taskCreateLists: [][]*store.TaskMessage{
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
			taskIndexDAGLists: [][]store.TaskIndexDAG{
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
			wantTaskCreateList: []*store.TaskMessage{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			wantTaskDAGList: []store.TaskIndexDAG{
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

func TestConvertBytesToUTF8String(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{
			input:    []byte{},
			expected: "",
		},
		{
			input:    []byte("hello"),
			expected: "hello",
		},
		{
			input:    []byte("你好"),
			expected: "你好",
		},
		{
			input:    []byte("Hello 世界 😊"),
			expected: "Hello 世界 😊",
		},
		{
			// string: SELECT "�ݱ�˼"
			input:    []byte{83, 69, 76, 69, 67, 84, 32, 34, 176, 221, 177, 180, 203, 188, 34},
			expected: "SELECT \"拜贝思\"",
		},
	}

	for _, test := range tests {
		actual, err := ConvertBytesToUTF8String(test.input)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, actual)
	}
}

func TestCheckDatabaseGroupMatch(t *testing.T) {
	tests := []struct {
		expression string
		database   *store.DatabaseMessage
		match      bool
	}{
		{
			expression: `resource.labels.unit == "gcp"`,
			database: &store.DatabaseMessage{
				Metadata: &storepb.DatabaseMetadata{
					Labels: map[string]string{
						"unit": "gcp",
					},
				},
			},
			match: true,
		},
		{
			expression: `resource.labels.unit == "aws"`,
			database: &store.DatabaseMessage{
				Metadata: &storepb.DatabaseMetadata{
					Labels: map[string]string{
						"unit": "gcp",
					},
				},
			},
			match: false,
		},
		{
			expression: `has(resource.labels.unit) && resource.labels.unit == "aws"`,
			database: &store.DatabaseMessage{
				Metadata: &storepb.DatabaseMetadata{},
			},
			match: false,
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		match, err := CheckDatabaseGroupMatch(ctx, test.expression, test.database)
		assert.NoError(t, err)
		assert.Equal(t, test.match, match)
	}
}
