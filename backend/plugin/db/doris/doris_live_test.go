package doris

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestDorisDriver exercises the adapter against a live server: ping, the
// instance and database sync, the query path (types, limit rewrite, explain,
// errors, affected rows, the size cap), a change script with its task-run log,
// and the server-stated dump. It needs a Doris server, which CI cannot provide,
// so it runs only when DORIS_TEST_HOST names one; DORIS_TEST_PORT defaults to
// 9030 and the login is root with no password.
func TestDorisDriver(t *testing.T) {
	host := os.Getenv("DORIS_TEST_HOST")
	if host == "" {
		t.Skip("DORIS_TEST_HOST is not set: a live Doris server is needed")
	}
	port := os.Getenv("DORIS_TEST_PORT")
	if port == "" {
		port = "9030"
	}
	ctx := context.Background()

	open := func(database string) *Driver {
		t.Helper()
		registered, err := db.Open(ctx, storepb.Engine_DORIS, db.ConnectionConfig{
			DataSource:        &storepb.DataSource{Host: host, Port: port, Username: "root"},
			ConnectionContext: db.ConnectionContext{DatabaseName: database},
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = registered.Close(ctx) })
		adapter, ok := registered.(*Driver)
		require.True(t, ok, "the registered driver is Bytebase's Doris adapter")
		return adapter
	}

	const databaseName = "bb_doris_driver_test"
	unbound := open("")
	require.NoError(t, unbound.Ping(ctx))
	require.Nil(t, unbound.GetDB())

	const properties = `PROPERTIES("replication_num" = "1")`
	setup := strings.Join([]string{
		"DROP DATABASE IF EXISTS " + databaseName,
		"CREATE DATABASE " + databaseName,
		"CREATE TABLE " + databaseName + ".types (id INT NOT NULL COMMENT 'the id', s VARCHAR(20) DEFAULT 'x', " +
			"f DOUBLE, dec DECIMAL(10,2), dt DATETIME, da DATE, bo BOOLEAN) " +
			"ENGINE=OLAP DUPLICATE KEY(id) DISTRIBUTED BY HASH(id) BUCKETS 1 " + properties,
		"INSERT INTO " + databaseName + ".types VALUES " +
			"(1, 'hello', 2.25, 1234.56, '2026-03-04 05:06:07', '2026-03-04', true), " +
			"(2, NULL, NULL, NULL, NULL, NULL, NULL)",
		"CREATE TABLE " + databaseName + ".sources (k INT NOT NULL) " +
			"ENGINE=OLAP DUPLICATE KEY(k) DISTRIBUTED BY HASH(k) BUCKETS 1 " + properties,
		"INSERT INTO " + databaseName + ".sources VALUES (1),(2)",
		"CREATE VIEW " + databaseName + ".v_sources AS SELECT k FROM " + databaseName + ".sources",
		// A Doris materialized view reads base tables only: the server refuses a
		// definition that names a view ("can not contain VIEW").
		"CREATE MATERIALIZED VIEW " + databaseName + ".mv_sources BUILD IMMEDIATE REFRESH COMPLETE ON MANUAL " +
			"DISTRIBUTED BY HASH(k) BUCKETS 1 " + properties + " AS SELECT k FROM " + databaseName + ".sources",
	}, ";\n")
	_, err := unbound.Execute(ctx, "-- txn-mode = off\n"+setup, db.ExecuteOptions{})
	require.NoError(t, err)

	driver := open(databaseName)

	t.Run("sync instance reports the engine release", func(t *testing.T) {
		instance, err := driver.SyncInstance(ctx)
		require.NoError(t, err)
		// @@version_comment names the release; VERSION() reports the MySQL wire
		// compatibility version, 5.7.99, which is not the engine's release.
		require.Regexp(t, `^\d+\.\d+\.\d+$`, instance.Version)
		require.NotEqual(t, "5.7.99", instance.Version)
		var names []string
		for _, database := range instance.Databases {
			names = append(names, database.Name)
		}
		require.Contains(t, names, databaseName)
		for _, system := range []string{"information_schema", "_statistics_", "mysql", "__internal_schema"} {
			require.NotContains(t, names, system)
		}
		// Doris returns no usable character set or collation for a database.
		for _, database := range instance.Databases {
			require.Empty(t, database.CharacterSet)
			require.Empty(t, database.Collation)
		}
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
		for _, materializedView := range schema.Schemas[0].MaterializedViews {
			materializedViews = append(materializedViews, materializedView.Name)
		}
		require.Equal(t, []string{"sources", "types"}, tables)
		require.Equal(t, []string{"v_sources"}, views)
		// Doris reports an asynchronous materialized view as a BASE TABLE, so its
		// membership in mv_infos() is what routes it here.
		require.Equal(t, []string{"mv_sources"}, materializedViews)
		require.Equal(t, "SELECT `internal`.`"+databaseName+"`.`sources`.`k` FROM `internal`.`"+databaseName+"`.`sources`",
			strings.TrimSpace(schema.Schemas[0].MaterializedViews[0].Definition))

		// Column defaults keep the MySQL-family rendering: a quoted literal, the
		// CURRENT_TIMESTAMP function unquoted, and NULL for the rest.
		var columns []string
		for _, column := range schema.Schemas[0].Tables[1].Columns {
			columns = append(columns, column.Name+"="+column.Default)
		}
		// Doris states a single-quoted default unquoted, so the synced default is
		// the value alone; the MySQL-family rule keeps whatever the server states.
		require.Equal(t, []string{"id=", "s=x", "f=NULL", "dec=NULL", "dt=NULL", "da=NULL", "bo=NULL"}, columns)
		require.False(t, schema.Schemas[0].Tables[1].Columns[0].Nullable)
		require.Equal(t, "the id", schema.Schemas[0].Tables[1].Columns[0].Comment)
		// Doris returns no usable character set or collation for a database.
		require.Empty(t, schema.CharacterSet)
		require.Empty(t, schema.Collation)
	})

	t.Run("a missing database is not found", func(t *testing.T) {
		_, err := open("bb_no_such_database").SyncDBSchema(ctx)
		require.ErrorContains(t, err, `database "bb_no_such_database" not found`)
	})

	t.Run("query decodes the server's types", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT * FROM types ORDER BY id", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		result := results[0]
		require.Equal(t, "SELECT * FROM types ORDER BY id", result.Statement)
		require.Equal(t, []string{"id", "s", "f", "dec", "dt", "da", "bo"}, result.ColumnNames)
		// Doris's wire type for VARCHAR is the MySQL family's CHAR name.
		require.Equal(t, []string{"INT", "CHAR", "DOUBLE", "DECIMAL", "DATETIME", "DATE", "TINYINT"}, result.ColumnTypeNames)
		require.Equal(t, int64(2), result.RowsCount)

		first := result.Rows[0].Values
		require.Equal(t, int64(1), first[0].GetInt64Value())
		require.Equal(t, "hello", first[1].GetStringValue())
		require.Equal(t, 2.25, first[2].GetDoubleValue())
		// DECIMAL is text, as the MySQL-family driver states it.
		require.Equal(t, "1234.56", first[3].GetStringValue())
		require.Equal(t, int32(0), first[4].GetTimestampValue().GetAccuracy())
		require.Equal(t, "2026-03-04", first[5].GetStringValue())
		// Doris states BOOLEAN as TINYINT, so the cell is an integer.
		require.Equal(t, int64(1), first[6].GetInt64Value())

		second := result.Rows[1].Values
		for i, value := range second {
			require.NotNil(t, value.GetNullValue(), "column %d of the all-NULL row", i)
		}
	})

	t.Run("a timestamp keeps its fractional digits", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT CAST('2026-01-02 03:04:05.123' AS DATETIME(3))",
			db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		timestamp := results[0].Rows[0].Values[0].GetTimestampValue()
		require.Equal(t, int32(3), timestamp.GetAccuracy())
		require.Equal(t, 123_000_000, timestamp.GetGoogleTimestamp().AsTime().Nanosecond())
	})

	t.Run("limit and explain are rewritten before the statement runs", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT id FROM types ORDER BY id",
			db.QueryContext{Limit: 1, MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "SELECT id FROM types ORDER BY id LIMIT 1", results[0].Statement)
		require.Equal(t, int64(1), results[0].RowsCount)

		// An existing, larger LIMIT is capped.
		results, err = driver.QueryConn(ctx, nil, "SELECT id FROM types ORDER BY id LIMIT 100",
			db.QueryContext{Limit: 1, MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Equal(t, "SELECT id FROM types ORDER BY id LIMIT 1", results[0].Statement)

		// Doris refuses `#`, so a limited statement that carries one is wrapped in
		// a subquery instead of having the limit spliced before the comment, and
		// the wrap spans three lines.
		results, err = driver.QueryConn(ctx, nil, "SELECT id FROM types # hash comment",
			db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Equal(t, "SELECT * FROM (\nSELECT id FROM types # hash comment\n) result LIMIT 2;", results[0].Statement)
		require.Contains(t, results[0].GetError(), "mismatched input '#'")

		// The driver offers Doris one plan format, so the request's format does not
		// reach the statement, and the plan stays in the rows.
		results, err = driver.QueryConn(ctx, nil, "SELECT id FROM types", db.QueryContext{
			Explain: true,
			Option:  &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		})
		require.NoError(t, err)
		require.Equal(t, "EXPLAIN SELECT id FROM types", results[0].Statement)
		require.Equal(t, []string{"Explain String(Nereids Planner)"}, results[0].ColumnNames)
		require.NotEmpty(t, results[0].Rows)
		require.Nil(t, results[0].QueryPlan)
	})

	t.Run("a statement's error stops the script", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT 1; SELECT * FROM missing_table; SELECT 2",
			db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Empty(t, results[0].GetError())
		require.Contains(t, results[1].GetError(), "Table [missing_table] does not exist")
	})

	t.Run("a write through the query path reports its affected rows", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "INSERT INTO types (id) VALUES (3)", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, []string{"Affected Rows"}, results[0].ColumnNames)
		require.Equal(t, int64(1), results[0].Rows[0].Values[0].GetInt64Value())
	})

	t.Run("the size cap truncates and the script continues", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT id, repeat('x', 80) FROM types ORDER BY id;\nSELECT 7",
			db.QueryContext{MaximumSQLResultSize: 200})
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Equal(t, "Output of query exceeds max allowed output size of 0MB", results[0].GetError())
		require.NotEmpty(t, results[1].Rows)
	})

	t.Run("a zero-byte cap is no cap", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT id FROM types ORDER BY id", db.QueryContext{})
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
		var commands int
		for _, entry := range logs {
			if entry.GetType() == storepb.TaskRunLog_COMMAND_EXECUTE {
				commands++
			}
		}
		require.Equal(t, 2, commands)
	})

	t.Run("a transaction level is read and its level ignored", func(t *testing.T) {
		// Known difference, preserved: the pre-cutover driver never applied a
		// script's "-- txn-isolation" level, and this adapter keeps that so a
		// cutover does not change what a script does. Doris accepts the level
		// statement and documents READ COMMITTED as its only level, so applying
		// the request would add a statement whose effect is not the level asked
		// for. The script still runs.
		affected, err := driver.Execute(ctx,
			"-- txn-isolation = SERIALIZABLE\nINSERT INTO types (id) VALUES (12)", db.ExecuteOptions{})
		require.NoError(t, err)
		require.Equal(t, int64(1), affected)
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

	t.Run("delimiter directives are cut client-side", func(t *testing.T) {
		// The directives are the client's, and Doris refuses a script that carries
		// them; the driver cuts the script at them before sending anything.
		affected, err := driver.Execute(ctx,
			"DELIMITER //\nINSERT INTO types (id) VALUES (30)//\nINSERT INTO types (id) VALUES (31)//\nDELIMITER ;",
			db.ExecuteOptions{})
		require.NoError(t, err)
		require.Equal(t, int64(2), affected)

		// A procedure body holds semicolons, so the directive is what makes the
		// script one statement. The old driver sent the directives themselves and
		// the server refused the script.
		_, err = driver.Execute(ctx,
			"DELIMITER ;;\nCREATE PROCEDURE two_steps() BEGIN INSERT INTO types (id) VALUES (40); INSERT INTO types (id) VALUES (41); END;;\nDELIMITER ;",
			db.ExecuteOptions{})
		require.NoError(t, err)
		var dumped strings.Builder
		require.NoError(t, driver.Dump(ctx, &dumped, nil))
		require.Contains(t, dumped.String(), "-- Procedure structure for `TWO_STEPS`")
		require.Contains(t, dumped.String(), "CREATE PROCEDURE two_steps()")
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

		// Tables first, in name order; then the object whose definition reads one
		// of them.
		sources := strings.Index(dump, "-- Table structure for `sources`")
		types := strings.Index(dump, "-- Table structure for `types`")
		view := strings.Index(dump, "-- View structure for `v_sources`")
		materializedView := strings.Index(dump, "-- Materialized view structure for `mv_sources`")
		require.NotEqual(t, -1, sources)
		require.NotEqual(t, -1, view)
		require.NotEqual(t, -1, materializedView)
		require.Less(t, sources, types)
		require.Less(t, types, view)
		require.Less(t, types, materializedView)

		// No placeholders and no prologue: the statement is the server's own, and
		// the old document could not be replayed at all.
		require.NotContains(t, dump, "Temporary view structure")
		require.NotContains(t, dump, "DROP VIEW IF EXISTS")
		require.NotContains(t, dump, "SET @OLD_UNIQUE_CHECKS")
		require.NotContains(t, dump, "SET FOREIGN_KEY_CHECKS")
		require.Contains(t, dump, "CREATE TABLE `sources` (")
		require.Contains(t, dump, "DUPLICATE KEY(`k`)")
		require.Contains(t, dump, "CREATE MATERIALIZED VIEW mv_sources")

		// The old document could not be replayed at all: it began with
		// SET @OLD_UNIQUE_CHECKS, which Doris refuses ("Unknown system variable").
		// This one restores, so a second database holds the same objects.
		const restored = "bb_doris_driver_restored"
		_, err := unbound.Execute(ctx, "DROP DATABASE IF EXISTS "+restored+";\nCREATE DATABASE "+restored, db.ExecuteOptions{})
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = unbound.Execute(context.Background(), "DROP DATABASE IF EXISTS "+restored, db.ExecuteOptions{})
		})
		_, err = open(restored).Execute(ctx, dump, db.ExecuteOptions{})
		require.NoError(t, err)
		restoredSchema, err := open(restored).SyncDBSchema(ctx)
		require.NoError(t, err)
		sourceSchema, err := driver.SyncDBSchema(ctx)
		require.NoError(t, err)
		require.Equal(t, schemaObjects(sourceSchema), schemaObjects(restoredSchema))
	})
}

// heldSession returns the session QueryConn keeps, read under the lock that
// guards it.
func heldSession(d *Driver) omnidriver.Session {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	return d.session
}

// schemaObjects returns a schema's tables, views and materialized views by name,
// so a restored database can be compared with the one it was dumped from.
func schemaObjects(schema *metadatapb.DatabaseSchemaMetadata) []string {
	var objects []string
	for _, table := range schema.Schemas[0].Tables {
		objects = append(objects, "table:"+table.Name)
	}
	for _, view := range schema.Schemas[0].Views {
		objects = append(objects, "view:"+view.Name)
	}
	for _, view := range schema.Schemas[0].MaterializedViews {
		objects = append(objects, "materialized view:"+view.Name)
	}
	return objects
}

func logTypes(logs []*storepb.TaskRunLog) []storepb.TaskRunLog_Type {
	types := make([]storepb.TaskRunLog_Type, 0, len(logs))
	for _, entry := range logs {
		types = append(types, entry.GetType())
	}
	return types
}

// TestQueryConnKeepsOneSessionAndDropsItWhenTheContextEnds pins the session
// seam: the editor sends one statement per call on a driver it keeps, so the
// statements of a stream share a session, and a statement whose context ends
// must not leave that dead session to the stream's next statement.
func TestQueryConnKeepsOneSessionAndDropsItWhenTheContextEnds(t *testing.T) {
	host := os.Getenv("DORIS_TEST_HOST")
	if host == "" {
		t.Skip("DORIS_TEST_HOST is not set: a live Doris server is needed")
	}
	port := os.Getenv("DORIS_TEST_PORT")
	if port == "" {
		port = "9030"
	}
	ctx := context.Background()
	registered, err := db.Open(ctx, storepb.Engine_DORIS, db.ConnectionConfig{
		DataSource: &storepb.DataSource{Host: host, Port: port, Username: "root"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = registered.Close(ctx) })
	driver, ok := registered.(*Driver)
	require.True(t, ok, "the registered driver is Bytebase's Doris adapter")

	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 20}
	_, err = driver.QueryConn(ctx, nil, "SELECT 1", queryContext)
	require.NoError(t, err)
	held := heldSession(driver)
	require.NotNil(t, held, "the first statement opens the session")

	_, err = driver.QueryConn(ctx, nil, "SELECT 1", queryContext)
	require.NoError(t, err)
	require.True(t, held == heldSession(driver), "the stream's statements share one session")

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, _ = driver.QueryConn(canceledCtx, nil, "SELECT 1", queryContext)
	require.Nil(t, heldSession(driver), "a canceled statement's session is not one to keep")
}
