package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
)

func TestGetPeerTenantDatabase(t *testing.T) {
	dbs := []*api.Database{
		{
			ID:       0,
			Name:     "hello",
			Instance: &api.Instance{EnvironmentID: 0},
		},
		{
			ID:       1,
			Name:     "hello2",
			Instance: &api.Instance{EnvironmentID: 0},
		},
		{
			ID:       2,
			Name:     "hello",
			Instance: &api.Instance{EnvironmentID: 1},
		},
		{
			ID:       3,
			Name:     "world",
			Instance: &api.Instance{EnvironmentID: 2},
		},
	}

	tests := []struct {
		name          string
		pipeline      [][]*api.Database
		environmentID int
		want          *api.Database
	}{
		{
			"same environment",
			[][]*api.Database{
				{},
				{dbs[0], dbs[1]},
				nil,
				{dbs[3]},
				{dbs[2]},
			},
			1,
			dbs[2],
		},
		{
			"fallback",
			[][]*api.Database{
				{},
				{dbs[0], dbs[1]},
				nil,
				{dbs[3]},
			},
			1,
			dbs[0],
		},
	}

	for _, test := range tests {
		got := getPeerTenantDatabase(test.pipeline, test.environmentID)
		assert.Equal(t, got, test.want)
	}
}

func TestCheckCharacterSetCollationOwner(t *testing.T) {
	tests := []struct {
		dbType       db.Type
		characterSet string
		collation    string
		owner        string
		expectError  bool
	}{
		/* ClickHouse */
		// With character set or collation
		{
			dbType:       db.ClickHouse,
			characterSet: "utf8mb4",
			expectError:  true,
		},
		{
			dbType:      db.ClickHouse,
			collation:   "utf8mb4_0900_ai_ci",
			expectError: true,
		},
		// Normal
		{
			dbType:      db.ClickHouse,
			expectError: false,
		},

		/* Snowflake */
		// With character set or collation
		{
			dbType:       db.Snowflake,
			characterSet: "utf8mb4",
			expectError:  true,
		},
		{
			dbType:      db.Snowflake,
			collation:   "utf8mb4_0900_ai_ci",
			expectError: true,
		},
		// Normal
		{
			dbType:      db.Snowflake,
			expectError: false,
		},

		/* PostgreSQL */
		// Without owner
		{
			dbType:      db.Postgres,
			owner:       "",
			expectError: true,
		},
		// Without character set
		{
			dbType:      db.Postgres,
			owner:       "bytebase",
			collation:   "fr_FR",
			expectError: false,
		},
		// Without collation
		{
			dbType:       db.Postgres,
			owner:        "bytebase",
			characterSet: "UTF8",
			expectError:  false,
		},

		/* MySQL */
		// With character set or collation
		{
			dbType:       db.MySQL,
			characterSet: "utf8mb4",
			expectError:  true,
		},
		{
			dbType:      db.MySQL,
			collation:   "utf8mb4_0900_ai_ci",
			expectError: true,
		},
		// Normal
		{
			dbType:       db.MySQL,
			characterSet: "utf8mb4",
			collation:    "utf8mb4_0900_ai_ci",
			expectError:  false,
		},
	}
	for _, test := range tests {
		err := checkCharacterSetCollationOwner(test.dbType, test.characterSet, test.collation, test.owner)
		if test.expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestGetTaskAndDagListByVersion(t *testing.T) {
	databaseID1 := 1
	databaseID2 := 2
	tests := []struct {
		versionTaskList []*versionTask
		wantTaskList    []api.TaskCreate
		wantDAGList     []api.TaskIndexDAG
	}{
		{
			versionTaskList: []*versionTask{},
			wantTaskList:    nil,
			wantDAGList:     nil,
		},
		{
			versionTaskList: []*versionTask{
				{
					task:    &api.TaskCreate{DatabaseID: &databaseID1, Name: "task2"},
					version: "v2",
				},
				{
					task:    &api.TaskCreate{DatabaseID: &databaseID2, Name: "task2"},
					version: "v2",
				},
				{
					task:    &api.TaskCreate{DatabaseID: &databaseID1, Name: "task3"},
					version: "v3",
				},
				{
					task:    &api.TaskCreate{DatabaseID: &databaseID1, Name: "task1"},
					version: "v1",
				},
			},
			wantTaskList: []api.TaskCreate{
				{DatabaseID: &databaseID1, Name: "task1"},
				{DatabaseID: &databaseID1, Name: "task2"},
				{DatabaseID: &databaseID1, Name: "task3"},
				{DatabaseID: &databaseID2, Name: "task2"},
			},
			wantDAGList: []api.TaskIndexDAG{
				{FromIndex: 0, ToIndex: 1},
				{FromIndex: 1, ToIndex: 2},
			},
		},
	}

	for _, test := range tests {
		gotTaskList, gotDAGList := getTaskAndDagListByVersion(test.versionTaskList)
		assert.Equal(t, test.wantTaskList, gotTaskList)
		assert.Equal(t, test.wantDAGList, gotDAGList)
	}
}
