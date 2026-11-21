package mysql

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func newTestDatabase(t *testing.T) (*testcontainer.Container, string) {
	t.Helper()
	return testcontainer.NewMySQLDatabase(t)
}

func openTestDatabaseDriver(t *testing.T, container *testcontainer.Container, database string) *Driver {
	t.Helper()
	return openTestDriver(context.Background(), t, storepb.Engine_MYSQL, container, container.GetUsername(), container.GetPassword(), database)
}

// startGeneralLog clears the server's query log and turns it on, so a test can
// assert the exact statements the driver issued rather than trusting its reply.
func startGeneralLog(t *testing.T, container *testcontainer.Container) {
	t.Helper()
	admin := container.GetDB()
	_, err := admin.Exec("SET GLOBAL general_log = 'OFF'")
	require.NoError(t, err)
	_, err = admin.Exec("SET GLOBAL log_output = 'TABLE'")
	require.NoError(t, err)
	_, err = admin.Exec("TRUNCATE TABLE mysql.general_log")
	require.NoError(t, err)
	_, err = admin.Exec("SET GLOBAL general_log = 'ON'")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = admin.Exec("SET GLOBAL general_log = 'OFF'") })
}

func generalLogQueries(t *testing.T, container *testcontainer.Container) []string {
	t.Helper()
	admin := container.GetDB()
	_, err := admin.Exec("SET GLOBAL general_log = 'OFF'")
	require.NoError(t, err)
	rows, err := admin.Query("SELECT argument FROM mysql.general_log WHERE command_type = 'Query' ORDER BY event_time, thread_id")
	require.NoError(t, err)
	defer rows.Close()
	var queries []string
	for rows.Next() {
		var query string
		require.NoError(t, rows.Scan(&query))
		queries = append(queries, query)
	}
	require.NoError(t, rows.Err())
	return queries
}

func TestAdapterSyncInstanceAndDump(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("USE `%s`; CREATE TABLE t (id INT PRIMARY KEY, note VARCHAR(64));", database))
	require.NoError(t, err)

	opened := openTestDatabaseDriver(t, container, database)
	require.NoError(t, opened.Ping(ctx))

	instance, err := opened.SyncInstance(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, instance.Version)
	require.NotNil(t, instance.Metadata)
	require.NotEmpty(t, instance.Databases)

	metadata, err := opened.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Len(t, metadata.Schemas, 1)
	require.Len(t, metadata.Schemas[0].Tables, 1)
	require.Equal(t, "t", metadata.Schemas[0].Tables[0].Name)

	var dump bytes.Buffer
	require.NoError(t, opened.Dump(ctx, &dump, metadata))
	require.Contains(t, dump.String(), "CREATE TABLE `t`")
}

func TestAdapterQueryConnExecuteAndMasking(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("USE `%s`; CREATE TABLE qitems (id INT PRIMARY KEY, note VARCHAR(64));", database))
	require.NoError(t, err)

	opened := openTestDatabaseDriver(t, container, database)

	rows, err := opened.Execute(ctx, "-- txn-mode=on\nINSERT INTO qitems VALUES (1, 'one'); INSERT INTO qitems VALUES (2, 'two');", db.ExecuteOptions{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, rows, int64(2))

	results, err := opened.QueryConn(ctx, nil, "SELECT id, note FROM qitems ORDER BY id", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)
	require.Equal(t, []string{"id", "note"}, results[0].ColumnNames)
	require.Len(t, results[0].Rows, 2)
	require.Equal(t, int64(1), results[0].Rows[0].Values[0].GetInt64Value())
	require.Equal(t, "one", results[0].Rows[0].Values[1].GetStringValue())

	// A procedure's result set has columns the statement does not trace, so a
	// masked query withholds its rows.
	_, err = opened.Execute(ctx, "CREATE PROCEDURE p() BEGIN SELECT 1 AS n; END", db.ExecuteOptions{})
	require.NoError(t, err)
	unmasked, err := opened.QueryConn(ctx, nil, "CALL p()", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, unmasked, 1)
	require.Len(t, unmasked[0].Rows, 1)
	masked, err := opened.QueryConn(ctx, nil, "CALL p()", db.QueryContext{MaximumSQLResultSize: 1 << 20, MaskingEnabled: true})
	require.NoError(t, err)
	require.Len(t, masked, 1)
	// The masking split withholds the procedure's rows; the contract reports the
	// withheld statement as a count, which the API renders as "Affected Rows: 0".
	require.Equal(t, []string{"Affected Rows"}, masked[0].ColumnNames)
	require.Equal(t, int64(0), masked[0].Rows[0].Values[0].GetInt64Value())
}

// TestAdapterHoldsOneSessionAcrossQueryConnCalls pins the session seam: the
// editor sends one statement at a time on a driver it keeps, so a user variable
// and a temporary table set by one call must be visible to the next.
func TestAdapterHoldsOneSessionAcrossQueryConnCalls(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	opened := openTestDatabaseDriver(t, container, database)

	_, err := opened.QueryConn(ctx, nil, "SET @cutover_marker = 41", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	results, err := opened.QueryConn(ctx, nil, "SELECT @cutover_marker + 1", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, int64(42), results[0].Rows[0].Values[0].GetInt64Value())

	_, err = opened.QueryConn(ctx, nil, "CREATE TEMPORARY TABLE cutover_temp (id INT)", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	_, err = opened.QueryConn(ctx, nil, "INSERT INTO cutover_temp VALUES (7)", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	results, err = opened.QueryConn(ctx, nil, "SELECT id FROM cutover_temp", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, int64(7), results[0].Rows[0].Values[0].GetInt64Value())
}

// TestAdapterDiscardsTheSessionAfterACanceledQuery: the MySQL client closes
// the connection when a statement's context ends, so the session must not be
// handed to the next statement in the same editor stream.
func TestAdapterDiscardsTheSessionAfterACanceledQuery(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	opened := openTestDatabaseDriver(t, container, database)
	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 20}

	_, err := opened.QueryConn(ctx, nil, "SET @cutover_marker = 41", queryContext)
	require.NoError(t, err)

	canceledCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	// The statement runs past its context; the failure itself reaches the caller
	// on the result, so this test pins only what the session does. It took the
	// connection with it.
	_, _ = opened.QueryConn(canceledCtx, nil, "SELECT SLEEP(10)", queryContext)
	require.NotNil(t, canceledCtx.Err(), "the statement must have run past its context")
	require.Nil(t, opened.session, "a canceled statement's session is not one to keep")

	// The stream continues on a new session rather than the dead one.
	results, err := opened.QueryConn(ctx, nil, "SELECT 1", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1)
}

// TestAdapterExecuteAppliesTxnIsolation proves the directive still reaches the
// server: the script opens with SET TRANSACTION ISOLATION LEVEL before START
// TRANSACTION, as the old driver did through go-sql-driver's BeginTx.
func TestAdapterExecuteAppliesTxnIsolation(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("USE `%s`; CREATE TABLE iso_source (id INT PRIMARY KEY);", database))
	require.NoError(t, err)
	opened := openTestDatabaseDriver(t, container, database)

	startGeneralLog(t, container)
	_, err = opened.Execute(ctx, "-- txn-mode = on\n-- txn-isolation = READ COMMITTED\nINSERT INTO iso_source VALUES (1);", db.ExecuteOptions{})
	require.NoError(t, err)
	queries := generalLogQueries(t, container)
	require.Contains(t, queries, "SET TRANSACTION ISOLATION LEVEL READ COMMITTED")
	require.Contains(t, queries, "START TRANSACTION")
	require.Contains(t, queries, "/*app=bytebase*/ INSERT INTO iso_source VALUES (1)")
	require.Contains(t, queries, "COMMIT")
}

// explainJSONFormatVersion reports whether the server has the session variable
// MySQL 8.3 added. An older server answers ER_UNKNOWN_SYSTEM_VARIABLE and the
// driver must carry on with the format it has.
func explainJSONFormatVersion(t *testing.T, container *testcontainer.Container) bool {
	t.Helper()
	var name, value string
	err := container.GetDB().QueryRow("SHOW VARIABLES LIKE 'explain_json_format_version'").Scan(&name, &value)
	if err != nil {
		return false
	}
	return name != ""
}

// TestAdapterAffectedRowsEstimateReadsTheJSONPlanAndResets pins the estimate
// path's statements: the format-version SET, the JSON EXPLAIN of the
// AffectedRowsQuery, and the reset that returns the pooled connection to the
// server's default.
func TestAdapterAffectedRowsEstimateReadsTheJSONPlanAndResets(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("USE `%s`; CREATE TABLE td (id INT PRIMARY KEY, d INT); INSERT INTO td VALUES (1, 0), (2, 0), (3, 0);", database))
	require.NoError(t, err)
	opened := openTestDatabaseDriver(t, container, database)

	startGeneralLog(t, container)
	count, err := opened.CountAffectedRows(ctx, "DELETE FROM td WHERE d = 0;")
	require.NoError(t, err)
	require.Greater(t, count, int64(0))
	queries := generalLogQueries(t, container)

	jsonExplain := ""
	for _, query := range queries {
		if strings.HasPrefix(query, "EXPLAIN FORMAT=JSON ") {
			jsonExplain = query
			break
		}
	}
	require.NotEmpty(t, jsonExplain, "the estimate must EXPLAIN the statement's AffectedRowsQuery: %v", queries)
	require.Contains(t, jsonExplain, "td")
	if explainJSONFormatVersion(t, container) {
		require.Contains(t, queries, "SET SESSION explain_json_format_version = 1")
		require.Contains(t, queries, "SET SESSION explain_json_format_version = DEFAULT")
	}
}

// TestAdapterAffectedRowsEstimateFallsBackToTraditional covers the JSON plan
// MySQL 8.0 does not fully state for an INSERT ... SELECT with a window: the
// driver falls back to the tabular EXPLAIN, as the old one did.
func TestAdapterAffectedRowsEstimateFallsBackToTraditional(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf(`
USE `+"`%s`"+`;
CREATE TABLE source_rows (id INT PRIMARY KEY);
INSERT INTO source_rows VALUES (1), (2), (3);
CREATE TABLE target_rows (id INT, n BIGINT);
`, database))
	require.NoError(t, err)
	opened := openTestDatabaseDriver(t, container, database)

	startGeneralLog(t, container)
	count, err := opened.CountAffectedRows(ctx, "INSERT INTO target_rows SELECT id, ROW_NUMBER() OVER (ORDER BY id) FROM source_rows;")
	require.NoError(t, err)
	require.Greater(t, count, int64(0))
	queries := generalLogQueries(t, container)

	var jsonExplain, traditionalExplain bool
	for _, query := range queries {
		if strings.HasPrefix(query, "EXPLAIN FORMAT=JSON ") {
			jsonExplain = true
		}
		if strings.HasPrefix(query, "EXPLAIN FORMAT=TRADITIONAL ") {
			traditionalExplain = true
		}
	}
	require.True(t, jsonExplain, "the estimate starts with the JSON plan: %v", queries)
	require.True(t, traditionalExplain, "a JSON plan without an estimate must fall back to EXPLAIN FORMAT=TRADITIONAL: %v", queries)
	if explainJSONFormatVersion(t, container) {
		require.Contains(t, queries, "SET SESSION explain_json_format_version = 1")
		require.Contains(t, queries, "SET SESSION explain_json_format_version = DEFAULT")
	}
}

func TestAdapterProbe(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	opened := openTestDatabaseDriver(t, container, database)

	result, err := opened.Probe(ctx, db.ProbeContext{PreExecutions: []string{"SET @probe_marker = 1"}}, "EXPLAIN SELECT @probe_marker")
	require.NoError(t, err)
	require.NotEmpty(t, result.Columns)
}

// requireSubsequence reports whether want appears in got in order. The server's
// general log carries connection setup queries too, so a test filters to the
// statements it recorded from the old driver rather than asserting the whole log.
func requireSubsequence(t *testing.T, got, want []string) {
	t.Helper()
	normalize := func(query string) string { return strings.Join(strings.Fields(query), " ") }
	i := 0
	for _, query := range got {
		if i < len(want) && normalize(query) == normalize(want[i]) {
			i++
		}
	}
	require.Equal(t, len(want), i, "want %v as a subsequence of %v", want, got)
}

// TestAdapterExecuteStatementOrderMatchesRecordedOldDriver compares Execute with
// expectations recorded from the old driver before it was deleted: one
// CONNECTION_ID read, START TRANSACTION, each command with the app comment, and
// COMMIT. The old path is gone, so this is the recorded form of the
// differential corpus.
func TestAdapterExecuteStatementOrderMatchesRecordedOldDriver(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("USE `%s`; CREATE TABLE p (id INT PRIMARY KEY, v INT); INSERT INTO p VALUES (1, 0), (2, 0);", database))
	require.NoError(t, err)
	opened := openTestDatabaseDriver(t, container, database)

	startGeneralLog(t, container)
	rows, err := opened.Execute(ctx, "-- txn-mode = on\nUPDATE p SET v = 2 WHERE id = 1;\nDELETE FROM p WHERE id = 2;", db.ExecuteOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	requireSubsequence(t, generalLogQueries(t, container), []string{
		"SELECT CONNECTION_ID()",
		"START TRANSACTION",
		"/*app=bytebase*/ UPDATE p SET v = 2 WHERE id = 1",
		"/*app=bytebase*/ DELETE FROM p WHERE id = 2",
		"COMMIT",
	})

	startGeneralLog(t, container)
	rows, err = opened.Execute(ctx, "-- txn-mode = off\nUPDATE p SET v = 3 WHERE id = 1;", db.ExecuteOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	queries := generalLogQueries(t, container)
	requireSubsequence(t, queries, []string{
		"SELECT CONNECTION_ID()",
		"/*app=bytebase*/ UPDATE p SET v = 3 WHERE id = 1",
	})
	require.NotContains(t, queries, "START TRANSACTION")
	require.NotContains(t, queries, "COMMIT")
}

// TestAdapterQueryConnResultShapeMatchesRecordedOldDriver records the result a
// statement returns: column names and types, rows, and the affected-rows shape
// for a write. These are the old driver's shapes for the same statements.
func TestAdapterQueryConnResultShapeMatchesRecordedOldDriver(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("USE `%s`; CREATE TABLE r (id INT PRIMARY KEY, note VARCHAR(64), n BIGINT UNSIGNED, b BLOB); INSERT INTO r VALUES (1, 'one', 18446744073709551615, X'00FF');", database))
	require.NoError(t, err)
	opened := openTestDatabaseDriver(t, container, database)

	results, err := opened.QueryConn(ctx, nil, "SELECT id, note, n, b FROM r", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	result := results[0]
	require.Empty(t, result.Error)
	require.Equal(t, []string{"id", "note", "n", "b"}, result.ColumnNames)
	require.Equal(t, []string{"INT", "VARCHAR", "UNSIGNED BIGINT", "BLOB"}, result.ColumnTypeNames)
	require.Equal(t, int64(1), result.RowsCount)
	require.Equal(t, int64(1), result.Rows[0].Values[0].GetInt64Value())
	require.Equal(t, "one", result.Rows[0].Values[1].GetStringValue())
	// The old conversion returned an unsigned BIGINT as text; the contract's is
	// typed, which is a recorded cutover difference.
	require.Equal(t, uint64(18446744073709551615), result.Rows[0].Values[2].GetUint64Value())
	require.Equal(t, []byte{0x00, 0xff}, result.Rows[0].Values[3].GetBytesValue())

	writeResults, err := opened.QueryConn(ctx, nil, "UPDATE r SET note = 'changed' WHERE id = 1", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, writeResults, 1)
	require.Equal(t, []string{"Affected Rows"}, writeResults[0].ColumnNames)
	require.Equal(t, int64(1), writeResults[0].Rows[0].Values[0].GetInt64Value())
}

// TestAdapterServesMariaDBThroughItsOwnDriver pins that the engine value this
// package has always registered still works after the MySQL cutover. MariaDB is
// the fork's own contract driver, not the MySQL one: it carries the MariaDB
// grammar (RETURNING, sequences) and its own sync and dump.
func TestAdapterServesMariaDBThroughItsOwnDriver(t *testing.T) {
	ctx := context.Background()
	container, database := testcontainer.NewMariaDBDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("USE `%s`; CREATE TABLE m (id INT PRIMARY KEY, note VARCHAR(32));", database))
	require.NoError(t, err)

	opened := openTestDriver(ctx, t, storepb.Engine_MARIADB, container, container.GetUsername(), container.GetPassword(), database)
	require.NoError(t, opened.Ping(ctx))
	instance, err := opened.SyncInstance(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, instance.Version)

	metadata, err := opened.SyncDBSchema(ctx)
	require.NoError(t, err)
	var dump bytes.Buffer
	require.NoError(t, opened.Dump(ctx, &dump, metadata))
	require.Contains(t, dump.String(), "CREATE TABLE `m`")

	_, err = opened.Execute(ctx, "INSERT INTO m VALUES (1, 'one')", db.ExecuteOptions{})
	require.NoError(t, err)
	results, err := opened.QueryConn(ctx, nil, "SELECT note FROM m WHERE id = 1", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "one", results[0].Rows[0].Values[0].GetStringValue())
}
