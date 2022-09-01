package server

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
	// Register postgres parser driver.
	_ "github.com/bytebase/bytebase/plugin/parser/engine/pg"
)

type testData struct {
	stmt     string
	taskType api.TaskType
	want     []api.TaskCheckResult
}

func TestStatementTypeCheck(t *testing.T) {
	tests := []testData{
		{
			stmt:     "CREATE TABLE t(a int, b int)",
			taskType: api.TaskDatabaseDataUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   "\"CREATE TABLE t(a int, b int)\" is not DML",
				},
			},
		},
		{
			stmt:     "ALTER TABLE t ADD COLUMN a int",
			taskType: api.TaskDatabaseDataUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   "\"ALTER TABLE t ADD COLUMN a int\" is not DML",
				},
			},
		},
		{
			stmt:     "INSERT INTO t values(1, 2, 3)",
			taskType: api.TaskDatabaseDataUpdate,
			want:     []api.TaskCheckResult(nil),
		},
		{
			stmt:     "COMMIT;",
			taskType: api.TaskDatabaseDataUpdate,
			want:     []api.TaskCheckResult(nil),
		},
		{
			stmt:     "CREATE TABLE t(a int, b int)",
			taskType: api.TaskDatabaseSchemaUpdate,
			want:     []api.TaskCheckResult(nil),
		},
		{
			stmt:     "ALTER TABLE t ADD COLUMN a int",
			taskType: api.TaskDatabaseSchemaUpdate,
			want:     []api.TaskCheckResult(nil),
		},
		{
			stmt:     "INSERT INTO t values(1, 2, 3)",
			taskType: api.TaskDatabaseSchemaUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDDL.Int(),
					Title:     "Alter schema can only run DDL",
					Content:   "\"INSERT INTO t values(1, 2, 3)\" is not DDL",
				},
			},
		},
		{
			stmt:     "COMMIT;",
			taskType: api.TaskDatabaseSchemaUpdate,
			want:     []api.TaskCheckResult(nil),
		},
	}

	for _, test := range tests {
		res, err := mysqlStatementTypeCheck(test.stmt, "", "", test.taskType)
		require.NoError(t, err)
		require.Equal(t, test.want, res)
		res, err = postgresqlStatementTypeCheck(test.stmt, test.taskType)
		require.NoError(t, err)
		require.Equal(t, test.want, res)
	}
}
