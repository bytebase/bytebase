package mongodb

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// requireMongosh checks if mongosh is installed and fails the test if not.
// These tests require mongosh to be installed because the MongoDB driver
// executes queries by shelling out to mongosh (see mongodb.go:174 and mongodb.go:344).
//
// TODO: These tests can be removed after migrating MongoDB driver to use Go driver API
// instead of shelling out to mongosh CLI.
//
// To install mongosh v2.5.0 (recommended version):
// - macOS: brew install mongosh
// - Linux: Download from https://github.com/mongodb-js/mongosh/releases/tag/v2.5.0
// - CI: Automatically installed in .github/workflows/backend-tests.yml
func requireMongosh(t *testing.T) {
	t.Helper()
	path, err := exec.LookPath("mongosh")
	if err != nil {
		t.Fatalf("mongosh is required but not found in PATH. Please install mongosh v2.5.0 to run this test.\n"+
			"Install instructions:\n"+
			"  macOS: brew install mongosh\n"+
			"  Linux: https://github.com/mongodb-js/mongosh/releases/tag/v2.5.0\n"+
			"Error: %v", err)
	}
	t.Logf("Using mongosh at: %s", path)
}

// TestQueryWithBracketNotation tests the critical user journey (CUJ) of querying
// MongoDB collections using bracket notation with different quote styles.
// This ensures the fix for PR #17282 (which changed to single-quote bracket notation
// for special characters) works correctly.
func TestQueryWithBracketNotation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MongoDB testcontainer test in short mode")
	}

	requireMongosh(t)

	ctx := context.Background()

	// Get MongoDB container from testcontainer utility
	container := testcontainer.GetTestMongoDBContainer(ctx, t)
	defer container.Close(ctx)

	// Create test database and collection with some data
	driver := &Driver{}
	connConfig := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Username: container.GetUsername(),
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "testdb",
		},
		Password: container.GetPassword(),
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_MONGODB, connConfig)
	require.NoError(t, err)
	defer openedDriver.Close(ctx)

	// Wait for MongoDB to be fully ready
	err = openedDriver.Ping(ctx)
	require.NoError(t, err, "Failed to ping MongoDB")

	// Create a collection with a special character in the name to test the real CUJ
	collectionName := "test.collection"

	// Insert test data
	insertStatement := `db['` + collectionName + `'].insertMany([
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35}
	]);`

	_, err = openedDriver.Execute(ctx, insertStatement, db.ExecuteOptions{})
	require.NoError(t, err, "Failed to insert test data")

	// Test cases for different bracket notation styles
	testCases := []struct {
		name      string
		statement string
	}{
		{
			name:      "double_quote_bracket",
			statement: `db["` + collectionName + `"].find().limit(50)`,
		},
		{
			name:      "single_quote_bracket",
			statement: `db['` + collectionName + `'].find().limit(50)`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := openedDriver.QueryConn(ctx, nil, tc.statement, db.QueryContext{
				Limit:                50,
				MaximumSQLResultSize: 10 * 1024 * 1024, // 10MB
			})

			// Verify the query succeeded
			require.NoError(t, err, "Query should not return an error")
			require.NotNil(t, results, "Results should not be nil")
			require.Len(t, results, 1, "Should return one query result")

			result := results[0]

			// Verify no error in the result
			require.Empty(t, result.Error, "Result should not contain an error")

			// Verify the result structure matches expected format
			require.Equal(t, []string{"result"}, result.ColumnNames, "Should have 'result' column")
			require.Equal(t, []string{"TEXT"}, result.ColumnTypeNames, "Column type should be 'TEXT'")

			// Verify we got exactly 3 rows back
			require.NotNil(t, result.Rows, "Rows should not be nil")
			require.Len(t, result.Rows, 3, "Should return 3 rows")

			// Verify each row structure and content
			for i, row := range result.Rows {
				require.Len(t, row.Values, 1, "Each row should have one value")
				require.NotNil(t, row.Values[0].GetStringValue(), "Value should be a string")

				// Verify the content is valid JSON that contains expected fields
				jsonStr := row.Values[0].GetStringValue()
				require.Contains(t, jsonStr, `"name"`, "Row %d should contain 'name' field", i)
				require.Contains(t, jsonStr, `"age"`, "Row %d should contain 'age' field", i)
				require.True(t,
					strings.Contains(jsonStr, "Alice") ||
						strings.Contains(jsonStr, "Bob") ||
						strings.Contains(jsonStr, "Charlie"),
					"Row %d should contain a valid name", i)
			}

			// Verify the statement is recorded correctly
			require.Equal(t, tc.statement, result.Statement, "Statement should match the input")
		})
	}
}

// TestQueryWithBracketNotationStructure tests the exact structure of query results
// using protocmp to ensure the result format is correct.
func TestQueryWithBracketNotationStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MongoDB testcontainer test in short mode")
	}

	requireMongosh(t)

	ctx := context.Background()

	// Get MongoDB container from testcontainer utility
	container := testcontainer.GetTestMongoDBContainer(ctx, t)
	defer container.Close(ctx)

	// Create test database and collection with deterministic data
	driver := &Driver{}
	connConfig := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Username: container.GetUsername(),
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "testdb",
		},
		Password: container.GetPassword(),
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_MONGODB, connConfig)
	require.NoError(t, err)
	defer openedDriver.Close(ctx)

	// Wait for MongoDB to be fully ready
	err = openedDriver.Ping(ctx)
	require.NoError(t, err, "Failed to ping MongoDB")

	collectionName := "users"

	// Insert a single deterministic document for easier verification
	insertStatement := `db['` + collectionName + `'].insertOne({"_id": 1, "name": "Test User", "age": 25});`
	_, err = openedDriver.Execute(ctx, insertStatement, db.ExecuteOptions{})
	require.NoError(t, err, "Failed to insert test data")

	// Query with bracket notation
	statement := `db['` + collectionName + `'].find().limit(1)`
	results, err := openedDriver.QueryConn(ctx, nil, statement, db.QueryContext{
		Limit:                50,
		MaximumSQLResultSize: 10 * 1024 * 1024,
	})

	require.NoError(t, err)
	require.Len(t, results, 1)

	result := results[0]

	// Build expected result structure (ignoring dynamic fields like Latency and RowsCount)
	expected := &v1pb.QueryResult{
		ColumnNames:     []string{"result"},
		ColumnTypeNames: []string{"TEXT"},
		Rows: []*v1pb.QueryRow{
			{
				Values: []*v1pb.RowValue{
					{
						Kind: &v1pb.RowValue_StringValue{
							StringValue: result.Rows[0].Values[0].GetStringValue(), // Use actual value for comparison
						},
					},
				},
			},
		},
		Statement: statement,
	}

	// Compare with protocmp, ignoring Latency and RowsCount which are non-deterministic
	diff := cmp.Diff(expected, result,
		protocmp.Transform(),
		protocmp.IgnoreFields(&v1pb.QueryResult{}, "latency", "rows_count"),
	)
	require.Empty(t, diff, "Result structure should match expected format")

	// Verify the actual JSON content
	jsonStr := result.Rows[0].Values[0].GetStringValue()
	require.Contains(t, jsonStr, `"_id"`)
	require.Contains(t, jsonStr, `"name"`)
	require.Contains(t, jsonStr, `"age"`)
	require.Contains(t, jsonStr, "Test User")
}
