package taskcheck

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
	// Register postgres parser driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
)

type testData struct {
	stmt     string
	taskType api.TaskType
	want     []api.TaskCheckResult
}

func TestStatementTypeCheck(t *testing.T) {
	tests := []testData{
		{
			stmt:     "CREATE DATABASE db\n;",
			taskType: api.TaskDatabaseSchemaUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeCreateDatabase.Int(),
					Title:     "Cannot create database",
					Content:   "The statement \"CREATE DATABASE db\n;\" creates database",
				},
			},
		},

		{
			stmt:     "DROP DATABASE db\n;",
			taskType: api.TaskDatabaseSchemaUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeDropDatabase.Int(),
					Title:     "Cannot drop database",
					Content:   "The statement \"DROP DATABASE db\n;\" drops database",
				},
			},
		},
		{
			stmt:     "CREATE DATABASE db\n;",
			taskType: api.TaskDatabaseDataUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeCreateDatabase.Int(),
					Title:     "Cannot create database",
					Content:   "The statement \"CREATE DATABASE db\n;\" creates database",
				},
				{
					Status:    api.TaskCheckStatusWarn,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   "\"CREATE DATABASE db\n;\" is not DML",
				},
			},
		},
		{
			stmt:     "DROP DATABASE db\n;",
			taskType: api.TaskDatabaseDataUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeDropDatabase.Int(),
					Title:     "Cannot drop database",
					Content:   "The statement \"DROP DATABASE db\n;\" drops database",
				},
				{
					Status:    api.TaskCheckStatusWarn,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   "\"DROP DATABASE db\n;\" is not DML",
				},
			},
		},
		{
			stmt:     "CREATE TABLE t(a int, b int)\n;",
			taskType: api.TaskDatabaseDataUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusWarn,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   "\"CREATE TABLE t(a int, b int)\n;\" is not DML",
				},
			},
		},
		{
			stmt:     "ALTER TABLE t ADD COLUMN a int\n;",
			taskType: api.TaskDatabaseDataUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusWarn,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   "\"ALTER TABLE t ADD COLUMN a int\n;\" is not DML",
				},
			},
		},
		{
			stmt:     "INSERT INTO t values(1, 2, 3)\n;",
			taskType: api.TaskDatabaseDataUpdate,
			want:     []api.TaskCheckResult(nil),
		},
		{
			stmt:     "COMMIT;",
			taskType: api.TaskDatabaseDataUpdate,
			want:     []api.TaskCheckResult(nil),
		},
		{
			stmt:     "SELECT max(x);",
			taskType: api.TaskDatabaseDataUpdate,
			want:     []api.TaskCheckResult(nil),
		},
		{
			stmt:     "CREATE TABLE t(a int, b int)\n;",
			taskType: api.TaskDatabaseSchemaUpdate,
			want:     []api.TaskCheckResult(nil),
		},
		{
			stmt:     "ALTER TABLE t ADD COLUMN a int\n;",
			taskType: api.TaskDatabaseSchemaUpdate,
			want:     []api.TaskCheckResult(nil),
		},
		{
			stmt:     "INSERT INTO t values(1, 2, 3)\n;",
			taskType: api.TaskDatabaseSchemaUpdate,
			want: []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusWarn,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDDL.Int(),
					Title:     "Alter schema can only run DDL",
					Content:   "\"INSERT INTO t values(1, 2, 3)\n;\" is not DDL",
				},
			},
		},
		{
			stmt:     "COMMIT;",
			taskType: api.TaskDatabaseSchemaUpdate,
			want:     []api.TaskCheckResult(nil),
		},
	}

	for i, test := range tests {
		res, err := mysqlStatementTypeCheck(test.stmt, "", "", test.taskType)
		require.NoError(t, err)
		require.Equal(t, test.want, res)
		res, err = postgresqlStatementTypeCheck(test.stmt, test.taskType)
		require.NoError(t, err)
		require.Equal(t, test.want, res, i)
	}
}
