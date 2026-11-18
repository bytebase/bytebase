package clickhouse

import (
	"bytes"
	"context"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

const testDatabase = "bb_clickhouse_test"

// TestClickHouseDriver exercises the adapter against a live server: ping, the
// instance and database syncs, the editor's query path and its read/write
// classification, the result-size cap, the session a driver holds, the task-run
// log, the execution-time default, and that Dump delegates to the driver.
func TestClickHouseDriver(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.SharedClickHouseContainer(t)

	open := func(database string) *Driver {
		t.Helper()
		driver, err := newDriver().Open(ctx, storepb.Engine_CLICKHOUSE, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Host:     container.GetHost(),
				Port:     container.GetPort(),
				Username: container.GetUsername(),
			},
			ConnectionContext: db.ConnectionContext{DatabaseName: database},
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = driver.Close(ctx) })
		clickhouse, ok := driver.(*Driver)
		require.True(t, ok, "the registered driver is the ClickHouse adapter")
		return clickhouse
	}

	server := open("")
	require.NoError(t, server.Ping(ctx))
	require.Nil(t, server.GetDB(), "the omni driver owns its pool and hands out sessions")
	for _, statement := range []string{
		"DROP DATABASE IF EXISTS " + testDatabase,
		"CREATE DATABASE " + testDatabase,
	} {
		_, err := server.Execute(ctx, statement, db.ExecuteOptions{})
		require.NoError(t, err)
	}

	database := open(testDatabase)

	// The corpus a dump has to order: a materialized view whose storage is an
	// inner table, a view, a data-skipping index, and a sorting key holding a
	// function's comma.
	var executeLogs []*storepb.TaskRunLog
	_, err := database.Execute(ctx, `
CREATE TABLE t (
  id Int64,
  i8 Int8,
  u32 UInt32,
  f32 Float32,
  s String,
  dt64 DateTime64(3),
  tm Time,
  ip4 IPv4,
  ip6 IPv6,
  arr Array(Int32),
  tup Tuple(Int32, String),
  n Nullable(String)
) ENGINE = MergeTree ORDER BY id;
ALTER TABLE t ADD INDEX idx_s s TYPE bloom_filter GRANULARITY 4;
INSERT INTO t VALUES (1, -1, 2, 1.5, 'hello', '2026-11-18 07:30:00.123', '07:30:00', '192.168.1.1', '2001:db8::1', [1, 2], (7, 'x'), 'n');
INSERT INTO t VALUES (2, -2, 3, 2.5, 'world', '2026-11-18 07:30:00.456', '07:30:01', '192.168.1.2', '2001:db8::2', [3], (8, 'y'), NULL);
CREATE TABLE keys (a Int64, b Int64) ENGINE = MergeTree ORDER BY (a, intDiv(b, 2)) PRIMARY KEY a;
CREATE VIEW v AS SELECT id, s FROM t;
CREATE MATERIALIZED VIEW mv ENGINE = MergeTree ORDER BY id AS SELECT id, s FROM t;
CREATE TABLE agg (k UInt64, a AggregateFunction(sum, UInt64)) ENGINE = AggregatingMergeTree ORDER BY k;
`, db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
		executeLogs = append(executeLogs, entry)
		return nil
	}})
	require.NoError(t, err)

	t.Run("execute reports the script's steps", func(t *testing.T) {
		require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN,
			executeLogs[0].GetTransactionControl().GetType())
		require.Equal(t, storepb.TaskRunLog_TransactionControl_COMMIT,
			executeLogs[len(executeLogs)-1].GetTransactionControl().GetType())
		commands := 0
		for _, entry := range executeLogs {
			if entry.Type == storepb.TaskRunLog_COMMAND_EXECUTE {
				commands++
				require.NotNil(t, entry.GetCommandExecute().GetRange())
				require.Greater(t, entry.GetCommandExecute().GetRange().GetEnd(),
					entry.GetCommandExecute().GetRange().GetStart())
			}
		}
		require.Equal(t, 8, commands)
	})

	t.Run("a failed script logs the rollback ClickHouse cannot take", func(t *testing.T) {
		var logs []*storepb.TaskRunLog
		_, err := database.Execute(ctx, "CREATE TABLE nowhere (id Int64) ENGINE = MergeTree ORDER BY id;\nINSERT INTO missing_table VALUES (1)", db.ExecuteOptions{
			CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
				logs = append(logs, entry)
				return nil
			},
		})
		require.ErrorContains(t, err, "missing_table")
		require.Equal(t, storepb.TaskRunLog_TransactionControl_ROLLBACK,
			logs[len(logs)-1].GetTransactionControl().GetType())
	})

	t.Run("a txn-mode = off script logs no transaction control", func(t *testing.T) {
		var logs []*storepb.TaskRunLog
		_, err := database.Execute(ctx, "-- txn-mode = off\nINSERT INTO t (id) VALUES (3);\nINSERT INTO t (id) VALUES (4)", db.ExecuteOptions{
			CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
				logs = append(logs, entry)
				return nil
			},
		})
		require.NoError(t, err)
		for _, entry := range logs {
			require.NotEqual(t, storepb.TaskRunLog_TRANSACTION_CONTROL, entry.Type)
		}
	})

	t.Run("sync instance", func(t *testing.T) {
		instance, err := database.SyncInstance(ctx)
		require.NoError(t, err)
		require.Contains(t, instance.Version, ".")
		var databases []string
		for _, database := range instance.Databases {
			databases = append(databases, database.Name)
		}
		require.Contains(t, databases, testDatabase)
		require.NotContains(t, databases, "system")
		require.IsIncreasing(t, databases, "the omni driver lists databases in name order")
		var roles []string
		for _, role := range instance.Metadata.GetRoles() {
			roles = append(roles, role.Name)
		}
		require.Contains(t, roles, "default")
	})

	t.Run("sync database", func(t *testing.T) {
		schema, err := database.SyncDBSchema(ctx)
		require.NoError(t, err)
		require.Len(t, schema.Schemas, 1)
		tables := map[string]bool{}
		var views []string
		for _, table := range schema.Schemas[0].Tables {
			tables[table.Name] = true
		}
		for _, view := range schema.Schemas[0].Views {
			views = append(views, view.Name)
		}
		require.Contains(t, views, "v")
		// The per-object fidelity (engines, sorting keys, the view's storage) is
		// the omni driver's testdata/sync/database.json; this pins that the
		// synced metadata reaches the product unchanged.
		require.Contains(t, tables, "t")
		require.Contains(t, tables, "mv")
	})

	t.Run("typed values", func(t *testing.T) {
		results, err := database.QueryConn(ctx, nil, "SELECT i8, u32, f32, dt64, tm, ip4, ip6, tup, n FROM t WHERE id = 1", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].Error)
		values := results[0].Rows[0].Values
		require.Equal(t, int32(-1), values[0].GetInt32Value())
		require.Equal(t, uint32(2), values[1].GetUint32Value())
		require.Equal(t, float32(1.5), values[2].GetFloatValue())
		// A DateTime64 is a typed timestamp with the precision its type names.
		require.Equal(t, int32(3), values[3].GetTimestampValue().GetAccuracy())
		require.Equal(t, "2026-11-18T07:30:00.123Z", values[3].GetTimestampValue().GetGoogleTimestamp().AsTime().Format(time.RFC3339Nano))
		require.Equal(t, "07:30:00", values[4].GetStringValue())
		require.Equal(t, "192.168.1.1", values[5].GetStringValue())
		require.Equal(t, "2001:db8::1", values[6].GetStringValue())
		require.Equal(t, `[7,"x"]`, values[7].GetStringValue())
		require.Equal(t, "n", values[8].GetStringValue())
	})

	t.Run("a row set that answers with rows is a query", func(t *testing.T) {
		for _, statement := range []string{"SHOW TABLES", "DESCRIBE t", "EXISTS TABLE t"} {
			results, err := database.QueryConn(ctx, nil, statement, db.QueryContext{MaximumSQLResultSize: 1 << 20})
			require.NoError(t, err)
			require.Len(t, results, 1)
			require.Empty(t, results[0].Error)
			require.NotEqual(t, []string{"Affected Rows"}, results[0].ColumnNames, statement)
			require.NotEmpty(t, results[0].Rows, statement)
		}
	})

	t.Run("explain analyze keeps its plan", func(t *testing.T) {
		results, err := database.QueryConn(ctx, nil, "EXPLAIN ANALYZE SELECT count() FROM t", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].Error)
		require.Equal(t, []string{"explain"}, results[0].ColumnNames)
		require.NotEmpty(t, results[0].Rows)
		require.NotNil(t, results[0].QueryPlan)
		require.True(t, results[0].QueryPlan.GetExecuted())
	})

	t.Run("an explain request runs the rewritten statement", func(t *testing.T) {
		results, err := database.QueryConn(ctx, nil, "SELECT id FROM t", db.QueryContext{Explain: true, MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "EXPLAIN SELECT id FROM t", results[0].Statement)
		require.Nil(t, results[0].QueryPlan, "the API handler describes an explain request's plan")
	})

	t.Run("a write answers with no rows", func(t *testing.T) {
		// A write answers with no rows: clickhouse-go states no affected count,
		// so the result carries neither columns nor an "Affected Rows" row, and
		// Execute answers 0.
		results, err := database.QueryConn(ctx, nil, "INSERT INTO t (id) VALUES (5)", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].Error)
		require.Empty(t, results[0].ColumnNames)
		require.Empty(t, results[0].Rows)
		require.Equal(t, int64(0), results[0].RowsCount)
	})

	t.Run("a driver holds one session", func(t *testing.T) {
		_, err := database.QueryConn(ctx, nil, "SET max_threads = 3", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		results, err := database.QueryConn(ctx, nil, "SELECT toString(getSetting('max_threads'))", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "3", results[0].Rows[0].Values[0].GetStringValue())
	})

	t.Run("the driver sets the execution-time default", func(t *testing.T) {
		// The pinned omni driver sets max_execution_time = 300 before applying a
		// data source's own parameters, so a restrictive server profile does not
		// fail a resource-intensive sync or query with code 160.
		results, err := database.QueryConn(ctx, nil, "SELECT toString(getSetting('max_execution_time'))", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "300", results[0].Rows[0].Values[0].GetStringValue())
	})

	t.Run("the result-size cap", func(t *testing.T) {
		// 0 is no cap in the contract.
		results, err := database.QueryConn(ctx, nil, "SELECT id FROM t", db.QueryContext{})
		require.NoError(t, err)
		require.Empty(t, results[0].Error)
		require.Greater(t, len(results[0].Rows), 1)

		results, err = database.QueryConn(ctx, nil, "SELECT id FROM t", db.QueryContext{MaximumSQLResultSize: 1})
		require.NoError(t, err)
		require.Contains(t, results[0].Error, "exceeds max allowed output size")
		require.Len(t, results[0].Rows, 1)
	})

	t.Run("a limit rewrites a query only", func(t *testing.T) {
		results, err := database.QueryConn(ctx, nil, "SELECT id FROM t", db.QueryContext{Limit: 1, MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results[0].Rows, 1)
		require.Contains(t, results[0].Statement, "LIMIT 1")

		// A statement that cannot stand in the CTE still answers with rows when
		// a limit is set; the driver issues it as written.
		results, err = database.QueryConn(ctx, nil, "SHOW TABLES", db.QueryContext{Limit: 1, MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Empty(t, results[0].Error)
		require.NotEmpty(t, results[0].Rows)
	})

	t.Run("a statement that fails ends the script", func(t *testing.T) {
		fresh := open(testDatabase)
		results, err := fresh.QueryConn(ctx, nil, "SELECT 1;\nSELECT nope FROM t;\nSELECT 2", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Empty(t, results[0].Error)
		require.Contains(t, results[1].Error, "Unknown expression identifier")
	})

	t.Run("an aggregate function column explains itself", func(t *testing.T) {
		// A decode error can leave the client's connection unusable, so this
		// runs on a driver of its own.
		fresh := open(testDatabase)
		results, err := fresh.QueryConn(ctx, nil, "SELECT * FROM agg", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Contains(t, results[0].Error, "finalizeAggregation(col)")
	})

	t.Run("dump delegates to the driver", func(t *testing.T) {
		// The DDL corpus, its order and its round trip are engine fidelity, and
		// the omni driver's own TestDump and TestDumpRestores hold them. This
		// pins only the adapter seam: the writer and the metadata argument
		// reach the driver, and metadata does not narrow a server-DDL dump.
		var dump bytes.Buffer
		require.NoError(t, database.Dump(ctx, &dump, &metadatapb.DatabaseSchemaMetadata{}))
		require.Contains(t, dump.String(), "CREATE TABLE "+testDatabase+".t")
	})

	t.Run("a dropped database is not found", func(t *testing.T) {
		_, err := server.Execute(ctx, "DROP DATABASE "+testDatabase, db.ExecuteOptions{})
		require.NoError(t, err)
		// A driver opened on the database after it went away: its first
		// connection is refused with UNKNOWN_DATABASE, which the driver reports
		// as ErrNotFound.
		fresh := open(testDatabase)
		_, err = fresh.SyncDBSchema(ctx)
		require.ErrorIs(t, err, omnidriver.ErrNotFound)
	})
}
