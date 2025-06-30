package schemasync

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestTodoLogicForColumnDefaultMigration(t *testing.T) {
	testCases := []struct {
		name         string
		engine       storepb.Engine
		expectedTodo bool
		description  string
	}{
		{
			name:         "PostgreSQL",
			engine:       storepb.Engine_POSTGRES,
			expectedTodo: false,
			description:  "PostgreSQL sync now writes to Default field with schema qualification",
		},
		{
			name:         "MySQL",
			engine:       storepb.Engine_MYSQL,
			expectedTodo: true,
			description:  "MySQL sync hasn't been updated yet, needs migration",
		},
		{
			name:         "SQL Server",
			engine:       storepb.Engine_MSSQL,
			expectedTodo: true,
			description:  "SQL Server sync hasn't been updated yet, needs migration",
		},
		{
			name:         "Oracle",
			engine:       storepb.Engine_ORACLE,
			expectedTodo: true,
			description:  "Oracle sync hasn't been updated yet, needs migration",
		},
		{
			name:         "TiDB",
			engine:       storepb.Engine_TIDB,
			expectedTodo: true,
			description:  "TiDB sync hasn't been updated yet, needs migration",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use the centralized logic
			todo := common.EngineNeedsColumnDefaultMigration(tc.engine)

			require.Equal(t, tc.expectedTodo, todo,
				"Engine %s: %s. Expected todo=%t, got todo=%t",
				tc.name, tc.description, tc.expectedTodo, todo)
		})
	}
}

func TestEnginesNeedingMigrationConsistency(t *testing.T) {
	// Verify that PostgreSQL doesn't need migration using the centralized logic
	require.False(t, common.EngineNeedsColumnDefaultMigration(storepb.Engine_POSTGRES),
		"PostgreSQL should not need migration")

	// Verify that other common engines do need migration
	require.True(t, common.EngineNeedsColumnDefaultMigration(storepb.Engine_MYSQL),
		"MySQL should need migration")
	require.True(t, common.EngineNeedsColumnDefaultMigration(storepb.Engine_MSSQL),
		"SQL Server should need migration")
}
