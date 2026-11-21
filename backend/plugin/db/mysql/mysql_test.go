package mysql

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	odriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func newTestDatabase(t *testing.T) (*testcontainer.Container, string) {
	t.Helper()
	return testcontainer.NewMySQLDatabase(t)
}

func openTestDatabaseDriver(t *testing.T, container *testcontainer.Container, database string) *Driver {
	t.Helper()
	return openTestDriver(context.Background(), t, storepb.Engine_MYSQL, container, container.GetUsername(), container.GetPassword(), database)
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

	results, err := opened.QueryConn(ctx, "SELECT id, note FROM qitems ORDER BY id", db.QueryContext{MaximumSQLResultSize: 1 << 20})
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
	unmasked, err := opened.QueryConn(ctx, "CALL p()", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, unmasked, 1)
	require.Len(t, unmasked[0].Rows, 1)
	masked, err := opened.QueryConn(ctx, "CALL p()", db.QueryContext{MaximumSQLResultSize: 1 << 20, MaskingEnabled: true})
	require.NoError(t, err)
	require.Len(t, masked, 1)
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

	_, err := opened.QueryConn(ctx, "SET @session_marker = 41", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	results, err := opened.QueryConn(ctx, "SELECT @session_marker + 1", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, int64(42), results[0].Rows[0].Values[0].GetInt64Value())

	_, err = opened.QueryConn(ctx, "CREATE TEMPORARY TABLE session_temp (id INT)", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	_, err = opened.QueryConn(ctx, "INSERT INTO session_temp VALUES (7)", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	results, err = opened.QueryConn(ctx, "SELECT id FROM session_temp", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, int64(7), results[0].Rows[0].Values[0].GetInt64Value())
}

// TestAdapterDiscardsTheSessionAfterACanceledQuery pins that a canceled
// statement discards the held session so the next editor call starts fresh.
func TestAdapterDiscardsTheSessionAfterACanceledQuery(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	opened := openTestDatabaseDriver(t, container, database)
	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 20}

	_, err := opened.QueryConn(ctx, "SET @session_marker = 41", queryContext)
	require.NoError(t, err)

	canceledCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	_, _ = opened.QueryConn(canceledCtx, "SELECT SLEEP(10)", queryContext)
	require.NotNil(t, canceledCtx.Err(), "the statement must have run past its context")
	require.Nil(t, opened.Session(), "a canceled statement's session is not one to keep")

	results, err := opened.QueryConn(ctx, "SELECT 1", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1)
}

type fakeMySQLDriver struct {
	execScript string
	execOpts   odriver.ExecuteOptions
}

func (*fakeMySQLDriver) Ping(context.Context) error { return nil }
func (*fakeMySQLDriver) Close() error               { return nil }

func (f *fakeMySQLDriver) Execute(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	return 1, nil
}

// TestAdapterExecuteMapsTxnDirectives pins how Execute maps transaction mode
// and isolation directives while leaving CreateDatabase and MaximumRetries
// unset.
func TestAdapterExecuteMapsTxnDirectives(t *testing.T) {
	fake := &fakeMySQLDriver{}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

	rows, err := d.Execute(context.Background(), "-- txn-mode = on\n-- txn-isolation = READ COMMITTED\nINSERT INTO t VALUES (1);", db.ExecuteOptions{
		CreateDatabase: true,
		MaximumRetries: 3,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	require.Equal(t, "INSERT INTO t VALUES (1);", fake.execScript)
	require.Equal(t, odriver.Transactional, fake.execOpts.Transaction)
	require.Equal(t, odriver.ReadCommitted, fake.execOpts.Isolation)
	require.False(t, fake.execOpts.CreateDatabase)
	require.Zero(t, fake.execOpts.MaxRetries)

	_, err = d.Execute(context.Background(), "-- txn-mode = off\nUPDATE t SET v = 1;", db.ExecuteOptions{})
	require.NoError(t, err)
	require.Equal(t, odriver.AutoCommit, fake.execOpts.Transaction)
}

func TestAdapterProbeAndCountAffectedRows(t *testing.T) {
	ctx := context.Background()
	container, database := newTestDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("USE `%s`; CREATE TABLE td (id INT PRIMARY KEY, d INT); INSERT INTO td VALUES (1, 0), (2, 0), (3, 0);", database))
	require.NoError(t, err)
	opened := openTestDatabaseDriver(t, container, database)

	result, err := opened.Probe(ctx, db.ProbeContext{PreExecutions: []string{"SET @probe_marker = 1"}}, "EXPLAIN SELECT @probe_marker")
	require.NoError(t, err)
	require.NotEmpty(t, result.Columns)

	count, err := opened.CountAffectedRows(ctx, "DELETE FROM td WHERE d = 0;")
	require.NoError(t, err)
	require.Greater(t, count, int64(0))
}

// TestAdapterServesMariaDBThroughItsOwnDriver pins that the MARIADB engine
// dispatches to omni/mariadb/driver across sync, dump, execute, and query.
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
	results, err := opened.QueryConn(ctx, "SELECT note FROM m WHERE id = 1", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "one", results[0].Rows[0].Values[0].GetStringValue())
}
