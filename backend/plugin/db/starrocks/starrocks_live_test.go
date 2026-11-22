package starrocks

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestStarRocksDriver exercises the adapter against a live server: ping, the
// instance and database sync, the query path (types, limit rewrite, explain,
// errors, affected rows, the size cap), a change script with its task-run log,
// and the server-stated dump. It needs a StarRocks server, which CI cannot
// provide, so it runs only when STARROCKS_TEST_HOST names one; STARROCKS_TEST_PORT
// defaults to 9030 and the login is root with no password.
func TestStarRocksDriver(t *testing.T) {
	host := os.Getenv("STARROCKS_TEST_HOST")
	if host == "" {
		t.Skip("STARROCKS_TEST_HOST is not set: a live StarRocks server is needed")
	}
	port := os.Getenv("STARROCKS_TEST_PORT")
	if port == "" {
		port = "9030"
	}
	ctx := context.Background()

	open := func(database string) *Driver {
		t.Helper()
		registered, err := db.Open(ctx, storepb.Engine_STARROCKS, db.ConnectionConfig{
			DataSource:        &storepb.DataSource{Host: host, Port: port, Username: "root"},
			ConnectionContext: db.ConnectionContext{DatabaseName: database},
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = registered.Close(ctx) })
		adapter, ok := registered.(*Driver)
		require.True(t, ok, "the registered driver is Bytebase's StarRocks adapter")
		return adapter
	}

	const databaseName = "bb_starrocks_driver_test"
	unbound := open("")
	require.NoError(t, unbound.Ping(ctx))

	setup := strings.Join([]string{
		"DROP DATABASE IF EXISTS " + databaseName,
		"CREATE DATABASE " + databaseName,
		"CREATE TABLE " + databaseName + ".types (" +
			"id INT NOT NULL, s VARCHAR(20) NULL, f DOUBLE NULL, dec DECIMAL(10,2) NULL, " +
			"dt DATETIME NULL, da DATE NULL, bo BOOLEAN NULL" +
			") ENGINE=OLAP DUPLICATE KEY(id) DISTRIBUTED BY HASH(id) BUCKETS 1 PROPERTIES(\"replication_num\"=\"1\")",
		"INSERT INTO " + databaseName + ".types VALUES " +
			"(1, 'hello', 2.25, 1234.56, '2026-03-04 05:06:07', '2026-03-04', true), " +
			"(2, NULL, NULL, NULL, NULL, NULL, NULL)",
		"CREATE TABLE " + databaseName + ".sources (k INT NOT NULL) " +
			"ENGINE=OLAP DUPLICATE KEY(k) DISTRIBUTED BY HASH(k) BUCKETS 1 PROPERTIES(\"replication_num\"=\"1\")",
		"INSERT INTO " + databaseName + ".sources VALUES (1),(2)",
		"CREATE VIEW " + databaseName + ".v_sources AS SELECT k FROM " + databaseName + ".sources",
		"CREATE MATERIALIZED VIEW " + databaseName + ".mv_on_view DISTRIBUTED BY HASH(k) REFRESH ASYNC " +
			"AS SELECT k FROM " + databaseName + ".v_sources",
	}, ";\n")
	_, err := unbound.Execute(ctx, "-- txn-mode = off\n"+setup, db.ExecuteOptions{})
	require.NoError(t, err)

	driver := open(databaseName)

	t.Run("sync instance reports the engine release", func(t *testing.T) {
		instance, err := driver.SyncInstance(ctx)
		require.NoError(t, err)
		// @@version_comment names the release; VERSION() reports the MySQL wire
		// version, and StarRocks states today's release in the comment only.
		require.Regexp(t, `^\d+\.\d+\.\d+$`, instance.Version)
		require.NotEqual(t, "8.0.33", instance.Version)
		var names []string
		for _, database := range instance.Databases {
			names = append(names, database.Name)
		}
		require.Contains(t, names, databaseName)
		require.NotContains(t, names, "information_schema")
		require.NotContains(t, names, "_statistics_")
		require.NotContains(t, names, "sys")
	})

	t.Run("sync database reads the schema", func(t *testing.T) {
		schema, err := driver.SyncDBSchema(ctx)
		require.NoError(t, err)
		require.Equal(t, databaseName, schema.Name)
		require.Len(t, schema.Schemas, 1)
		var tables, views, materializedViews []string
		for _, table := range schema.Schemas[0].Tables {
			tables = append(tables, table.Name)
		}
		for _, view := range schema.Schemas[0].Views {
			views = append(views, view.Name)
		}
		for _, view := range schema.Schemas[0].MaterializedViews {
			materializedViews = append(materializedViews, view.Name)
		}
		require.Equal(t, []string{"sources", "types"}, tables)
		require.Equal(t, []string{"v_sources"}, views)
		// An asynchronous materialized view is listed as a VIEW by
		// information_schema.tables, so its membership in the catalog is what
		// routes it here.
		require.Equal(t, []string{"mv_on_view"}, materializedViews)
	})

	t.Run("a missing database is not found", func(t *testing.T) {
		_, err := open("bb_no_such_database").SyncDBSchema(ctx)
		require.ErrorContains(t, err, `database "bb_no_such_database" not found`)
	})

	t.Run("query decodes the server's types", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, "SELECT * FROM types ORDER BY id", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		result := results[0]
		require.Equal(t, "SELECT * FROM types ORDER BY id", result.Statement)
		require.Equal(t, []string{"id", "s", "f", "dec", "dt", "da", "bo"}, result.ColumnNames)
		require.Equal(t, []string{"INT", "VARCHAR", "DOUBLE", "DECIMAL", "DATETIME", "DATE", "TINYINT"}, result.ColumnTypeNames)
		require.Equal(t, int64(2), result.RowsCount)

		first := result.Rows[0].Values
		require.Equal(t, int64(1), first[0].GetInt64Value())
		require.Equal(t, "hello", first[1].GetStringValue())
		require.Equal(t, 2.25, first[2].GetDoubleValue())
		require.Equal(t, "1234.56", first[3].GetStringValue())
		require.Equal(t, int32(0), first[4].GetTimestampValue().GetAccuracy())
		require.Equal(t, "2026-03-04", first[5].GetStringValue())
		// StarRocks states BOOLEAN as TINYINT, so the cell is an integer.
		require.Equal(t, int64(1), first[6].GetInt64Value())

		second := result.Rows[1].Values
		for i, value := range second {
			require.NotNil(t, value.GetNullValue(), "column %d of the all-NULL row", i)
		}
	})

	t.Run("limit and explain are rewritten before the statement runs", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, "SELECT id FROM types ORDER BY id", db.QueryContext{Limit: 1, MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "SELECT id FROM types ORDER BY id LIMIT 1", results[0].Statement)
		require.Equal(t, int64(1), results[0].RowsCount)

		// An existing, larger LIMIT is capped; a smaller one is kept.
		results, err = driver.QueryConn(ctx, "SELECT id FROM types ORDER BY id LIMIT 100", db.QueryContext{Limit: 1})
		require.NoError(t, err)
		require.Equal(t, "SELECT id FROM types ORDER BY id LIMIT 1", results[0].Statement)

		// The driver offers StarRocks one plan format, so the request's format
		// does not reach the statement.
		results, err = driver.QueryConn(ctx, "SELECT id FROM types", db.QueryContext{
			Explain: true,
			Option:  &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		})
		require.NoError(t, err)
		require.Equal(t, "EXPLAIN SELECT id FROM types", results[0].Statement)
		require.Equal(t, []string{"Explain String"}, results[0].ColumnNames)
		require.NotEmpty(t, results[0].Rows)
	})

	t.Run("a statement's error stops the script", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, "SELECT 1; SELECT * FROM missing_table; SELECT 2", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Empty(t, results[0].GetError())
		require.Contains(t, results[1].GetError(), "Unknown table")
	})

	t.Run("a session variable reaches the next call", func(t *testing.T) {
		// The editor asks for one statement per QueryConn call, so the session
		// has to outlive the call.
		_, err := driver.QueryConn(ctx, "SET @bb_session = 42", db.QueryContext{})
		require.NoError(t, err)
		results, err := driver.QueryConn(ctx, "SELECT @bb_session", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, []string{"@bb_session"}, results[0].ColumnNames)
		require.Equal(t, int64(42), results[0].Rows[0].Values[0].GetInt64Value())
	})

	t.Run("a write through the query path reports its affected rows", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, "INSERT INTO types (id) VALUES (3)", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, []string{"Affected Rows"}, results[0].ColumnNames)
		require.Equal(t, int64(1), results[0].Rows[0].Values[0].GetInt64Value())
	})

	t.Run("an ADMIN SHOW answers with rows", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, "ADMIN SHOW FRONTEND CONFIG", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.NotEmpty(t, results[0].ColumnNames)
		require.NotEmpty(t, results[0].Rows)
	})

	t.Run("the size cap truncates and the script continues", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, "SELECT id, repeat('x', 80) FROM types ORDER BY id;\nSELECT 7", db.QueryContext{MaximumSQLResultSize: 200})
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Equal(t, "Output of query exceeds max allowed output size of 0MB", results[0].GetError())
		require.NotEmpty(t, results[1].Rows)
	})

	t.Run("a zero-byte cap is no cap", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, "SELECT id FROM types ORDER BY id", db.QueryContext{})
		require.NoError(t, err)
		require.Empty(t, results[0].GetError())
		require.Equal(t, int64(3), results[0].RowsCount)
	})

	t.Run("a script reports its steps and its affected rows", func(t *testing.T) {
		var logs []*storepb.TaskRunLog
		opts := db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		}}
		affected, err := driver.Execute(ctx,
			"-- txn-mode = off\nINSERT INTO types (id) VALUES (10);\nINSERT INTO types (id) VALUES (11)", opts)
		require.NoError(t, err)
		// The count is the sum over the statements, and each one is logged.
		require.Equal(t, int64(2), affected)
		require.NotEmpty(t, logs)
		var commands int
		for _, entry := range logs {
			if entry.GetType() == storepb.TaskRunLog_COMMAND_EXECUTE {
				commands++
			}
		}
		require.Equal(t, 2, commands)
	})

	t.Run("a transactional script rolls back", func(t *testing.T) {
		var logs []*storepb.TaskRunLog
		opts := db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		}}
		_, err := driver.Execute(ctx, "INSERT INTO types (id) VALUES (20);\nINSERT INTO missing_table (id) VALUES (21)", opts)
		require.ErrorContains(t, err, "missing_table")
		require.Contains(t, logTypes(logs), storepb.TaskRunLog_TRANSACTION_CONTROL)
	})

	t.Run("an empty script does nothing", func(t *testing.T) {
		var logs []*storepb.TaskRunLog
		opts := db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		}}
		affected, err := driver.Execute(ctx, "", opts)
		require.NoError(t, err)
		require.Zero(t, affected)
		require.Empty(t, logs)
	})

	t.Run("dump writes the definitions the server states", func(t *testing.T) {
		var dumped strings.Builder
		require.NoError(t, driver.Dump(ctx, &dumped, nil))
		dump := dumped.String()

		// Tables first, in name order, then definitions in the order a restore
		// replays them: the materialized view reads the view, so it follows it.
		sources := strings.Index(dump, "-- Table structure for `sources`")
		types := strings.Index(dump, "-- Table structure for `types`")
		view := strings.Index(dump, "-- View structure for `v_sources`")
		materializedView := strings.Index(dump, "-- Materialized view structure for `mv_on_view`")
		require.NotEqual(t, -1, sources)
		require.Less(t, sources, types)
		require.Less(t, types, view)
		require.Less(t, view, materializedView)

		// No placeholders and no prologue: the statement is the server's own.
		require.NotContains(t, dump, "Temporary view structure")
		require.NotContains(t, dump, "DROP VIEW IF EXISTS")
		require.NotContains(t, dump, "SET @OLD_UNIQUE_CHECKS")
		require.Contains(t, dump, "CREATE TABLE `sources` (")
		require.Contains(t, dump, "DUPLICATE KEY(`k`)")
		require.Contains(t, dump, "CREATE MATERIALIZED VIEW `mv_on_view`")
	})
}

func logTypes(logs []*storepb.TaskRunLog) []storepb.TaskRunLog_Type {
	types := make([]storepb.TaskRunLog_Type, 0, len(logs))
	for _, entry := range logs {
		types = append(types, entry.GetType())
	}
	return types
}
