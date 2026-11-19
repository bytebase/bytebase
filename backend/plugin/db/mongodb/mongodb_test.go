package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/protobuf/testing/protocmp"

	omnidriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestConnectionConfig pins the mapping from a data source onto the driver
// contract. The connection string the contract builds from it is omni's own
// TestConnectionURI, whose cases are these.
func TestConnectionConfig(t *testing.T) {
	tests := []struct {
		name       string
		connConfig db.ConnectionConfig
		want       omnidriver.Config
		wantSRV    bool
	}{
		{
			name: "host and port",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "localhost", Port: "27017"},
			},
			want: omnidriver.Config{Host: "localhost:27017", Port: "27017"},
		},
		{
			name: "direct connection",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "localhost", Port: "27017", DirectConnection: true},
			},
			want: omnidriver.Config{
				Host:   "localhost:27017",
				Port:   "27017",
				Params: map[string]string{"directConnection": "true"},
			},
		},
		{
			name: "database",
			connConfig: db.ConnectionConfig{
				DataSource:        &storepb.DataSource{Host: "localhost", Port: "27017"},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
			},
			want: omnidriver.Config{Host: "localhost:27017", Port: "27017", Database: "sampleDB"},
		},
		{
			name: "srv",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "cluster0.sample.mongodb.net",
					Username: "bytebase",
					Srv:      true,
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
				Password:          "passwd",
			},
			wantSRV: true,
			want: omnidriver.Config{
				Host:     "cluster0.sample.mongodb.net",
				Database: "sampleDB",
				User:     "bytebase",
				Password: "passwd",
			},
		},
		{
			name: "connection string options",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "cluster0.sample.mongodb.net",
					Username: "bytebase",
					Srv:      true,
					ExtraConnectionParameters: map[string]string{
						"readPreference":     "secondary",
						"readPreferenceTags": "dc:ny",
					},
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
				Password:          "passwd",
			},
			wantSRV: true,
			want: omnidriver.Config{
				Host:     "cluster0.sample.mongodb.net",
				Database: "sampleDB",
				User:     "bytebase",
				Password: "passwd",
				Params: map[string]string{
					"readPreference":     "secondary",
					"readPreferenceTags": "dc:ny",
				},
			},
		},
		{
			name: "authentication database",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                   "cluster0.sample.mongodb.net",
					Username:               "bytebase",
					AuthenticationDatabase: "bytebase",
					Srv:                    true,
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
				Password:          "passwd",
			},
			wantSRV: true,
			want: omnidriver.Config{
				Host:     "cluster0.sample.mongodb.net",
				Database: "sampleDB",
				User:     "bytebase",
				Password: "passwd",
				Params:   map[string]string{"authSource": "bytebase"},
			},
		},
		{
			name: "additional addresses carry their own ports",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "node1.cluster0.sample.mongodb.net",
					Port:     "27017",
					Username: "bytebase",
					AdditionalAddresses: []*storepb.DataSource_Address{
						{Host: "node2.cluster0.sample.mongodb.net", Port: "27017"},
						{Host: "node3.cluster0.sample.mongodb.net", Port: "27018"},
					},
					ReplicaSet:             "rs0",
					AuthenticationDatabase: "admin",
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
				Password:          "passwd",
			},
			want: omnidriver.Config{
				Host:     "node1.cluster0.sample.mongodb.net:27017,node2.cluster0.sample.mongodb.net:27017,node3.cluster0.sample.mongodb.net:27018",
				Port:     "27017",
				Database: "sampleDB",
				User:     "bytebase",
				Password: "passwd",
				Params:   map[string]string{"authSource": "admin", "replicaSet": "rs0"},
			},
		},
		{
			name: "TLS follows use_ssl",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "localhost", Port: "27017", UseSsl: true},
			},
			want: omnidriver.Config{Host: "localhost:27017", Port: "27017"},
		},
		{
			name: "IPv6 host with port",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "2001:db8::1", Port: "27017"},
			},
			want: omnidriver.Config{Host: "[2001:db8::1]:27017", Port: "27017"},
		},
		{
			name: "already-bracketed IPv6 host with port",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "[2001:db8::1]", Port: "27017"},
			},
			want: omnidriver.Config{Host: "[2001:db8::1]:27017", Port: "27017"},
		},
		{
			name: "IPv6 host without port",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "2001:db8::1"},
			},
			want: omnidriver.Config{Host: "[2001:db8::1]"},
		},
		{
			name: "multi-host IPv6 addresses",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host: "2001:db8::1",
					Port: "27017",
					AdditionalAddresses: []*storepb.DataSource_Address{
						{Host: "2001:db8::2", Port: "27018"},
					},
				},
			},
			want: omnidriver.Config{Host: "[2001:db8::1]:27017,[2001:db8::2]:27018", Port: "27017"},
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			got, options, err := toDriverConfig(tt.connConfig)
			a.NoError(err)
			if tt.connConfig.DataSource.GetUseSsl() {
				a.NotNil(got.TLS)
				got.TLS = nil
			}
			a.Equal(tt.want, got)
			a.Equal(tt.wantSRV, options.SRV)
		})
	}

	_, _, err := toDriverConfig(db.ConnectionConfig{})
	a.ErrorContains(err, "a data source is required")
}

// TestExecuteLogsParseFailure pins BYT-9950: when the statement fails to
// split/parse, Execute must still emit a COMMAND_EXECUTE / COMMAND_RESPONSE
// pair carrying the parse error so the failure is visible in task run logs,
// and must report failure.
func TestExecuteLogsParseFailure(t *testing.T) {
	const statement = "db.users.find();\nthis is not mongosh"

	run := func(t *testing.T, logStatement bool) []*storepb.TaskRunLog {
		var logs []*storepb.TaskRunLog
		opts := db.ExecuteOptions{
			CreateTaskRunLog: func(_ time.Time, l *storepb.TaskRunLog) error {
				logs = append(logs, l)
				return nil
			},
			LogCommandStatement: logStatement,
		}

		// Open does not reach the server, and the parse fails before a
		// statement is sent, so this test needs none.
		opened, err := (&Driver{}).Open(context.Background(), storepb.Engine_MONGODB, db.ConnectionConfig{
			DataSource: &storepb.DataSource{Host: "localhost", Port: "27017"},
		})
		require.NoError(t, err)
		_, err = opened.Execute(context.Background(), statement, opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "syntax error")

		require.Len(t, logs, 2)
		require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
		require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)
		require.Contains(t, logs[1].CommandResponse.Error, "syntax error")
		return logs
	}

	t.Run("range logging", func(t *testing.T) {
		logs := run(t, false)
		require.Empty(t, logs[0].CommandExecute.Statement)
		require.Equal(t, int32(0), logs[0].CommandExecute.Range.GetStart())
		require.Equal(t, int32(len(statement)), logs[0].CommandExecute.Range.GetEnd())
	})

	t.Run("statement logging", func(t *testing.T) {
		logs := run(t, true)
		require.Nil(t, logs[0].CommandExecute.Range)
		require.Equal(t, statement, logs[0].CommandExecute.GetStatement())
	})
}

// TestQueryWithBracketNotation tests the critical user journey (CUJ) of querying
// MongoDB collections using bracket notation with different quote styles.
// This ensures the fix for PR #17282 (which changed to single-quote bracket notation
// for special characters) works correctly.
func TestQueryWithBracketNotation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewMongoDatabase(t)

	// Create test database and collection with some data
	driver := &Driver{}
	connConfig := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Username: container.GetUsername(),
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: database,
		},
		Password: container.GetPassword(),
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_MONGODB, connConfig)
	require.NoError(t, err)
	t.Cleanup(func() { openedDriver.Close(ctx) })

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
			t.Parallel()
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
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewMongoDatabase(t)

	// Create test database and collection with deterministic data
	driver := &Driver{}
	connConfig := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Username: container.GetUsername(),
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: database,
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

// TestQueryDoubleNotation reproduces GitHub issue #20895: BSON doubles holding
// large integers must render in mongosh notation (plain decimal), not the Go
// driver's scientific notation (e.g. 1.779696815227E+12).
func TestQueryDoubleNotation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewMongoDatabase(t)

	driver := &Driver{}
	connConfig := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Username: container.GetUsername(),
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: database,
		},
		Password: container.GetPassword(),
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_MONGODB, connConfig)
	require.NoError(t, err)
	defer openedDriver.Close(ctx)

	err = openedDriver.Ping(ctx)
	require.NoError(t, err, "Failed to ping MongoDB")

	// Insert through a client of the test's own so the field types are
	// guaranteed BSON doubles, matching the issue's data shape: a mongosh
	// literal writes an integer as int32 or int64.
	client, err := mongo.Connect(options.Client().ApplyURI(fmt.Sprintf(
		"mongodb://%s:%s@%s:%s/?authSource=admin",
		container.GetUsername(), container.GetPassword(), container.GetHost(), container.GetPort(),
	)))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	_, err = client.Database(database).Collection("users").InsertOne(ctx, bson.D{
		{Key: "_id", Value: int32(1129063441)},
		{Key: "name", Value: "repro user 1"},
		{Key: "created_at", Value: float64(1779696815227)},
		{Key: "updated_at", Value: float64(1779803079861)},
		{Key: "current_workspace_id", Value: float64(585723378473606)},
	})
	require.NoError(t, err, "Failed to insert test data")

	results, err := openedDriver.QueryConn(ctx, nil, `db.users.find()`, db.QueryContext{
		Limit:                50,
		MaximumSQLResultSize: 10 * 1024 * 1024,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)
	require.Len(t, results[0].Rows, 1)

	jsonStr := results[0].Rows[0].Values[0].GetStringValue()
	require.Equal(t,
		`{"_id":1129063441,"name":"repro user 1","created_at":1779696815227,"updated_at":1779803079861,"current_workspace_id":585723378473606}`,
		jsonStr)
}

// TestQueryConnExplain pins the plan the query-plan viewer draws. MongoDB asks
// for a plan with a cursor method rather than a keyword, and only the
// queryPlanner verbosity describes the query without running it.
func TestQueryConnExplain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewMongoDatabase(t)

	driver := &Driver{}
	opened, err := driver.Open(ctx, storepb.Engine_MONGODB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Username: container.GetUsername(),
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
		Password:          container.GetPassword(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { opened.Close(ctx) })
	require.NoError(t, opened.Ping(ctx))

	_, err = opened.Execute(ctx, `db.orders.insertMany([
		{"customer_id": 1, "total": 100}, {"customer_id": 2, "total": 900}
	]);`, db.ExecuteOptions{})
	require.NoError(t, err)

	explain := func(statement string, asExplain bool) *v1pb.QueryResult {
		t.Helper()
		results, err := opened.QueryConn(ctx, nil, statement, db.QueryContext{
			Explain:              asExplain,
			Limit:                5000,
			MaximumSQLResultSize: 1 << 30,
			Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].GetError())
		return results[0]
	}
	planner := func(result *v1pb.QueryResult) map[string]any {
		t.Helper()
		require.Len(t, result.GetRows(), 1)
		var plan map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.GetRows()[0].GetValues()[0].GetStringValue()), &plan))
		queryPlanner, ok := plan["queryPlanner"].(map[string]any)
		require.True(t, ok, "no queryPlanner in %v", plan)
		return queryPlanner
	}

	const statement = `db.orders.find({total: {$gt: 500}})`
	plan := explain(statement, true)
	require.Equal(t, statement+`.explain("queryPlanner")`, plan.GetStatement())
	require.Contains(t, planner(plan), "winningPlan")

	// A statement that already asks for a plan keeps its query and loses its
	// verbosity, so an explain request never runs what it plans.
	replanned := explain(statement+`.explain("executionStats")`, true)
	require.Equal(t, statement+`.explain("queryPlanner")`, replanned.GetStatement())
	require.NotContains(t, planner(replanned), "executionStats")

	// The client runs a whole script in one call, but a plan covers one
	// statement, so a script is planned a statement at a time rather than
	// refused for holding two.
	const second = `db.orders.find({total: {$lt: 500}})`
	scripted, err := opened.QueryConn(ctx, nil, statement+"; "+second, db.QueryContext{
		Explain: true, MaximumSQLResultSize: 1 << 30,
		Option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
	})
	require.NoError(t, err)
	require.Len(t, scripted, 2)
	require.Equal(t, statement+`.explain("queryPlanner")`, scripted[0].GetStatement())
	require.Equal(t, second+`.explain("queryPlanner")`, scripted[1].GetStatement())
	require.Contains(t, planner(scripted[1]), "winningPlan")

	// Run on a statement that explains itself reports the plan it returned.
	ran := explain(statement+`.explain()`, false)
	require.Equal(t, v1pb.QueryOption_JSON, ran.GetQueryPlan().GetFormat())
	require.False(t, ran.GetQueryPlan().GetExecuted())

	// A measured plan runs the query, which the result has to say.
	measured := explain(statement+`.explain("executionStats")`, false)
	require.Equal(t, v1pb.QueryOption_JSON, measured.GetQueryPlan().GetFormat())
	require.True(t, measured.GetQueryPlan().GetExecuted())

	// An ordinary query is no plan at all.
	require.Nil(t, explain(statement, false).GetQueryPlan())

	// A write has no plan, and planning it would mean running it. The refusal
	// is the statement's own result error, as a statement's failure is.
	refused, err := opened.QueryConn(ctx, nil, `db.orders.insertOne({a: 1})`, db.QueryContext{
		Explain: true, MaximumSQLResultSize: 1 << 30,
		Option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
	})
	require.NoError(t, err)
	require.Len(t, refused, 1)
	require.Contains(t, refused[0].GetError(), "insertOne")
}
