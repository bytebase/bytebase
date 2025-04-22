package trino

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestTransformDMLToSelect(t *testing.T) {
	testCases := []struct {
		name                 string
		statement            string
		sourceDatabase       string
		targetDatabase       string
		tablePrefix          string
		expectedCount        int
		expectedTableName    string
		expectedSchemaName   string
		expectedDatabaseName string
		expectedSelectPrefix string
	}{
		{
			name:                 "Simple INSERT",
			statement:            "INSERT INTO users (id, name) VALUES (1, 'John')",
			sourceDatabase:       "test_db",
			targetDatabase:       "backup_db",
			tablePrefix:          "bak_",
			expectedCount:        1,
			expectedTableName:    "users",
			expectedSchemaName:   "public",
			expectedDatabaseName: "test_db",
			expectedSelectPrefix: "SELECT",
		},
		{
			name:                 "INSERT with qualified name",
			statement:            "INSERT INTO analytics.users (id, name) VALUES (1, 'John')",
			sourceDatabase:       "test_db",
			targetDatabase:       "backup_db",
			tablePrefix:          "bak_",
			expectedCount:        1,
			expectedTableName:    "users",
			expectedSchemaName:   "analytics",
			expectedDatabaseName: "test_db",
			expectedSelectPrefix: "SELECT",
		},
		{
			name:                 "INSERT with full qualified name",
			statement:            "INSERT INTO catalog1.analytics.users (id, name) VALUES (1, 'John')",
			sourceDatabase:       "test_db",
			targetDatabase:       "backup_db",
			tablePrefix:          "bak_",
			expectedCount:        1,
			expectedTableName:    "users",
			expectedSchemaName:   "analytics",
			expectedDatabaseName: "catalog1",
			expectedSelectPrefix: "SELECT",
		},
		{
			name:                 "UPDATE statement",
			statement:            "UPDATE users SET name = 'Jane' WHERE id = 1",
			sourceDatabase:       "test_db",
			targetDatabase:       "backup_db",
			tablePrefix:          "bak_",
			expectedCount:        1,
			expectedTableName:    "users",
			expectedSchemaName:   "public",
			expectedDatabaseName: "test_db",
			expectedSelectPrefix: "SELECT * FROM",
		},
		{
			name:                 "DELETE statement",
			statement:            "DELETE FROM users WHERE id = 1",
			sourceDatabase:       "test_db",
			targetDatabase:       "backup_db",
			tablePrefix:          "bak_",
			expectedCount:        1,
			expectedTableName:    "users",
			expectedSchemaName:   "public",
			expectedDatabaseName: "test_db",
			expectedSelectPrefix: "SELECT * FROM",
		},
		{
			name:                 "UPDATE with condition",
			statement:            "UPDATE orders SET total = 200.00 WHERE user_id = 5 AND id > 100",
			sourceDatabase:       "test_db",
			targetDatabase:       "backup_db",
			tablePrefix:          "bak_",
			expectedCount:        1,
			expectedTableName:    "orders",
			expectedSchemaName:   "public",
			expectedDatabaseName: "test_db",
			expectedSelectPrefix: "SELECT * FROM",
		},
		{
			name:                 "DELETE with complex condition",
			statement:            "DELETE FROM inactive_users WHERE last_login < DATE '2020-01-01' AND status = 'inactive'",
			sourceDatabase:       "test_db",
			targetDatabase:       "backup_db",
			tablePrefix:          "bak_",
			expectedCount:        1,
			expectedTableName:    "inactive_users",
			expectedSchemaName:   "public",
			expectedDatabaseName: "test_db",
			expectedSelectPrefix: "SELECT * FROM",
		},
		{
			name:           "Non-DML statement",
			statement:      "SELECT * FROM users",
			sourceDatabase: "test_db",
			expectedCount:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tCtx := base.TransformContext{}

			// Call the transform function
			backupStatements, err := TransformDMLToSelect(ctx, tCtx, tc.statement, tc.sourceDatabase, tc.targetDatabase, tc.tablePrefix)
			require.NoError(t, err, "TransformDMLToSelect should not error")

			// Verify the results
			assert.Equal(t, tc.expectedCount, len(backupStatements), "Expected backup statement count doesn't match")

			if tc.expectedCount > 0 {
				backupStmt := backupStatements[0]

				// Verify table and schema information
				assert.Equal(t, tc.expectedTableName, backupStmt.SourceTableName, "Table name doesn't match")
				assert.Equal(t, tc.expectedSchemaName, backupStmt.SourceSchema, "Schema name doesn't match")
				assert.Equal(t, tc.tablePrefix+tc.expectedTableName, backupStmt.TargetTableName, "Target table name doesn't match")

				// Check that the SELECT prefix is present in the transformed statement
				if backupStmt.Statement != "" {
					assert.True(t, strings.HasPrefix(strings.TrimSpace(backupStmt.Statement), tc.expectedSelectPrefix),
						"Expected SELECT statement to start with '%s', got: %s",
						tc.expectedSelectPrefix, backupStmt.Statement)
				}

				// For DELETE and UPDATE, check WHERE clause is preserved
				if strings.Contains(tc.statement, "WHERE") {
					assert.Contains(t, backupStmt.Statement, "WHERE", "WHERE clause should be preserved in transformed statement")
				}
			}
		})
	}
}
