package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestTransactionMode tests the configurable transaction mode feature across different database engines.
func TestTransactionMode(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		dbType           storepb.Engine
		containerFunc    func(context.Context, *testing.T) *testcontainer.Container
		setupScript      string
		testScriptOn     string
		testScriptOff    string
		expectRollbackOn bool // Whether we expect rollback for transaction mode on
		skipTransaction  bool // Some engines don't support transactions well
	}{
		{
			name:   "MySQL",
			dbType: storepb.Engine_MYSQL,
			containerFunc: func(ctx context.Context, t *testing.T) *testcontainer.Container {
				container, err := testcontainer.GetTestMySQLContainer(ctx)
				if err != nil {
					t.Fatalf("Failed to get MySQL container: %v", err)
				}
				return container
			},
			setupScript: `
				CREATE TABLE test_table (
					id INT PRIMARY KEY,
					value VARCHAR(100)
				);
			`,
			testScriptOn: `-- txn-mode = on
				INSERT INTO test_table (id, value) VALUES (1, 'test1');
				INSERT INTO test_table (id, value) VALUES (2, 'test2');
				INSERT INTO test_table (id, value) VALUES (1, 'duplicate'); -- This will fail
			`,
			testScriptOff: `-- txn-mode = off
				INSERT INTO test_table (id, value) VALUES (1, 'test1');
				INSERT INTO test_table (id, value) VALUES (2, 'test2');
				INSERT INTO test_table (id, value) VALUES (1, 'duplicate'); -- This will fail
			`,
			expectRollbackOn: true,
		},
		{
			name:   "PostgreSQL",
			dbType: storepb.Engine_POSTGRES,
			containerFunc: func(ctx context.Context, t *testing.T) *testcontainer.Container {
				return testcontainer.GetTestPgContainer(ctx, t)
			},
			setupScript: `
				CREATE TABLE test_table (
					id INTEGER PRIMARY KEY,
					value VARCHAR(100)
				);
			`,
			testScriptOn: `-- txn-mode = on
				INSERT INTO test_table (id, value) VALUES (1, 'test1');
				INSERT INTO test_table (id, value) VALUES (2, 'test2');
				INSERT INTO test_table (id, value) VALUES (1, 'duplicate'); -- This will fail
			`,
			testScriptOff: `-- txn-mode = off
				INSERT INTO test_table (id, value) VALUES (1, 'test1');
				INSERT INTO test_table (id, value) VALUES (2, 'test2');
				INSERT INTO test_table (id, value) VALUES (1, 'duplicate'); -- This will fail
			`,
			expectRollbackOn: true,
		},
		{
			name:   "Oracle",
			dbType: storepb.Engine_ORACLE,
			containerFunc: func(ctx context.Context, t *testing.T) *testcontainer.Container {
				return testcontainer.GetTestOracleContainer(ctx, t)
			},
			setupScript: `
				CREATE TABLE test_table (
					id NUMBER PRIMARY KEY,
					value VARCHAR2(100)
				)
			`,
			testScriptOn: `-- txn-mode = on
				INSERT INTO test_table (id, value) VALUES (1, 'test1');
				INSERT INTO test_table (id, value) VALUES (2, 'test2');
				INSERT INTO test_table (id, value) VALUES (1, 'duplicate'); -- This will fail
			`,
			testScriptOff: `-- txn-mode = off
				INSERT INTO test_table (id, value) VALUES (1, 'test1');
				INSERT INTO test_table (id, value) VALUES (2, 'test2');
				INSERT INTO test_table (id, value) VALUES (1, 'duplicate'); -- This will fail
			`,
			expectRollbackOn: true,
		},
		{
			name:   "SQL Server",
			dbType: storepb.Engine_MSSQL,
			containerFunc: func(ctx context.Context, t *testing.T) *testcontainer.Container {
				return testcontainer.GetTestMSSQLContainer(ctx, t)
			},
			setupScript: `
				CREATE TABLE test_table (
					id INT PRIMARY KEY,
					value VARCHAR(100)
				);
			`,
			testScriptOn: `-- txn-mode = on
				INSERT INTO test_table (id, value) VALUES (1, 'test1');
				INSERT INTO test_table (id, value) VALUES (2, 'test2');
				INSERT INTO test_table (id, value) VALUES (1, 'duplicate'); -- This will fail
			`,
			testScriptOff: `-- txn-mode = off
				INSERT INTO test_table (id, value) VALUES (1, 'test1');
				INSERT INTO test_table (id, value) VALUES (2, 'test2');
				INSERT INTO test_table (id, value) VALUES (1, 'duplicate'); -- This will fail
			`,
			expectRollbackOn: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Skip transaction tests for engines that don't support them well
			if test.skipTransaction {
				t.Skip("Skipping transaction test for engine with limited transaction support")
			}

			// Create container
			container := test.containerFunc(ctx, t)
			defer container.Close(ctx)

			// Create test database if needed
			testDBName := fmt.Sprintf("txnmode_test_%d", time.Now().UnixNano())
			if test.dbType == storepb.Engine_MYSQL {
				_, err := container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", testDBName))
				require.NoError(t, err)
			}

			// Get database connection
			driver, err := getDriver(ctx, test.dbType, container, testDBName)
			require.NoError(t, err)
			defer driver.Close(ctx)

			// Setup test table
			_, err = driver.Execute(ctx, test.setupScript, db.ExecuteOptions{})
			require.NoError(t, err)

			// Test transaction mode ON - expect rollback on failure
			t.Run("TransactionModeOn", func(t *testing.T) {
				// Execute script with transaction mode ON
				_, err := driver.Execute(ctx, test.testScriptOn, db.ExecuteOptions{})
				require.Error(t, err, "Expected error due to duplicate key")

				// Check if rollback occurred
				rowCount := getRowCount(ctx, t, driver, test.dbType)
				if test.expectRollbackOn {
					require.Equal(t, 0, rowCount, "Expected 0 rows due to rollback")
				} else {
					require.Greater(t, rowCount, 0, "Expected some rows even with error")
				}

				// Cleanup
				cleanupTable(ctx, t, driver, test.dbType)
			})

			// Test transaction mode OFF - expect partial success
			t.Run("TransactionModeOff", func(t *testing.T) {
				// Execute script with transaction mode OFF
				_, err := driver.Execute(ctx, test.testScriptOff, db.ExecuteOptions{})
				require.Error(t, err, "Expected error due to duplicate key")

				// Check if partial success occurred (first 2 inserts should succeed)
				rowCount := getRowCount(ctx, t, driver, test.dbType)
				require.Equal(t, 2, rowCount, "Expected 2 rows from successful inserts before failure")

				// Cleanup
				cleanupTable(ctx, t, driver, test.dbType)
			})
		})
	}
}

// Helper functions
func getDriver(ctx context.Context, engine storepb.Engine, container *testcontainer.Container, testDBName string) (db.Driver, error) {
	var dbDatabase string
	switch engine {
	case storepb.Engine_MYSQL:
		dbDatabase = testDBName
	case storepb.Engine_POSTGRES:
		dbDatabase = "postgres"
	case storepb.Engine_ORACLE:
		dbDatabase = "FREEPDB1"
	case storepb.Engine_MSSQL:
		dbDatabase = "master"
	default:
		dbDatabase = ""
	}

	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Username: "root",
			Database: dbDatabase,
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: dbDatabase,
		},
		Password: "root",
	}

	// Special handling for different engines based on testcontainer setup
	switch engine {
	case storepb.Engine_MYSQL:
		config.DataSource.Username = "root"
		config.Password = "root-password"
	case storepb.Engine_POSTGRES:
		config.DataSource.Username = "postgres"
		config.Password = "root-password"
	case storepb.Engine_ORACLE:
		config.DataSource.Username = "testuser"
		config.Password = "testpass"
		config.DataSource.Database = ""
		config.ConnectionContext.DatabaseName = ""
		// Set service name for Oracle
		config.DataSource.ServiceName = "FREEPDB1"
	case storepb.Engine_MSSQL:
		config.DataSource.Username = "sa"
		config.Password = "Test123!"
	default:
		// Use default config for other engines
	}

	driver, err := db.Open(ctx, engine, config)
	if err != nil {
		return nil, err
	}

	return driver, nil
}

func getRowCount(ctx context.Context, t *testing.T, driver db.Driver, _ storepb.Engine) int {
	query := "SELECT COUNT(*) FROM test_table"

	// Use QueryConn to get results
	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	results, err := driver.QueryConn(ctx, conn, query, db.QueryContext{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1)
	require.Len(t, results[0].Rows[0].Values, 1)

	// Extract count value
	switch v := results[0].Rows[0].Values[0].Kind.(type) {
	case *v1pb.RowValue_Int32Value:
		return int(v.Int32Value)
	case *v1pb.RowValue_Int64Value:
		return int(v.Int64Value)
	case *v1pb.RowValue_StringValue:
		// Some databases return count as string
		count := 0
		if _, err := fmt.Sscanf(v.StringValue, "%d", &count); err != nil {
			t.Fatalf("Failed to parse count from string: %v", err)
		}
		return count
	default:
		t.Fatalf("Unexpected count value type: %T", v)
		return 0
	}
}

func cleanupTable(ctx context.Context, t *testing.T, driver db.Driver, _ storepb.Engine) {
	_, err := driver.Execute(ctx, "DELETE FROM test_table", db.ExecuteOptions{})
	require.NoError(t, err)
}
