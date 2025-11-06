package mongodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestQueryWithBracketNotation tests the critical user journey (CUJ) of querying
// MongoDB collections using bracket notation with different quote styles.
// This ensures the fix for PR #17282 (which changed to single-quote bracket notation
// for special characters) works correctly.
func TestQueryWithBracketNotation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MongoDB testcontainer test in short mode")
	}

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
				Limit:                 50,
				MaximumSQLResultSize: 10 * 1024 * 1024, // 10MB
			})

			// Verify the query succeeded
			require.NoError(t, err, "Query should not return an error")
			require.NotNil(t, results, "Results should not be nil")
			require.Len(t, results, 1, "Should return one query result")

			result := results[0]

			// Verify no error in the result
			require.Empty(t, result.Error, "Result should not contain an error")

			// Verify we got data back
			require.NotNil(t, result.Rows, "Rows should not be nil")
			require.Equal(t, 3, len(result.Rows), "Should return 3 rows")

			// Verify the result has the expected structure
			require.NotEmpty(t, result.ColumnNames, "Should have column names")
			require.Equal(t, "result", result.ColumnNames[0], "Column name should be 'result'")
		})
	}
}
