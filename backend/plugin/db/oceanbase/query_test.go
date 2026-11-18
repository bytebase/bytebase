package oceanbase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestQueryConnKeepsTheSession proves the editor's statements share one server
// session across calls, as they shared the *sql.Conn the old driver was handed:
// a user variable and a temporary table created by one call are still there in
// the next.
func TestQueryConnKeepsTheSession(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := "query_session"
	admin := server.openRaw(ctx, t, "")
	exec(t, admin, "DROP DATABASE IF EXISTS "+database, "CREATE DATABASE "+database)
	t.Cleanup(func() { exec(t, admin, "DROP DATABASE IF EXISTS "+database) })
	driver := server.open(ctx, t, database)

	results, err := driver.QueryConn(ctx, nil, "SET @bytebase_probe = 41", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)

	results, err = driver.QueryConn(ctx, nil, "SELECT @bytebase_probe + 1 AS answer", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, []string{"answer"}, results[0].ColumnNames)
	require.Equal(t, int64(42), results[0].Rows[0].Values[0].GetInt64Value())

	results, err = driver.QueryConn(ctx, nil, "CREATE TEMPORARY TABLE tmp_state (a INT)", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)

	for _, statement := range []string{
		"INSERT INTO tmp_state VALUES (7)",
		"SELECT a FROM tmp_state",
	} {
		results, err = driver.QueryConn(ctx, nil, statement, db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err, statement)
		require.Len(t, results, 1, statement)
		require.Empty(t, results[0].Error, statement)
		if statement == "SELECT a FROM tmp_state" {
			require.Equal(t, int64(7), results[0].Rows[0].Values[0].GetInt64Value())
		}
	}

	// A second session of the same server does not see the first one's
	// temporary table: the state is the session's, not the pool's luck.
	other := server.open(ctx, t, database)
	results, err = other.QueryConn(ctx, nil, "SELECT a FROM tmp_state", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Contains(t, results[0].Error, "doesn't exist")
}

// TestQueryConnDiscardsTheSessionAfterACanceledQuery proves a statement whose
// context ends does not leave the stream a session it cannot use: the client
// closed the connection under it, so the next statement opens a new one.
func TestQueryConnDiscardsTheSessionAfterACanceledQuery(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := "query_cancel"
	admin := server.openRaw(ctx, t, "")
	exec(t, admin, "DROP DATABASE IF EXISTS "+database, "CREATE DATABASE "+database)
	t.Cleanup(func() { exec(t, admin, "DROP DATABASE IF EXISTS "+database) })
	driver := server.open(ctx, t, database)

	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 20}
	_, err := driver.QueryConn(ctx, nil, "SET @bytebase_probe = 41", queryContext)
	require.NoError(t, err)

	canceledCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	_, _ = driver.QueryConn(canceledCtx, nil, "SELECT SLEEP(10)", queryContext)
	require.NotNil(t, canceledCtx.Err(), "the statement must have run past its context")
	require.Nil(t, driver.session, "a canceled statement's session is not one to keep")

	results, err := driver.QueryConn(ctx, nil, "SELECT 1 AS one", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)
	require.Equal(t, int64(1), results[0].Rows[0].Values[0].GetInt64Value())
}

// TestQueryConnMapsTheRequest pins what QueryConn does with a request's
// options: a limit rewrites the statement, an explain request plans it and
// names no typed plan, a statement that changes rows reports the count, and a
// failing statement's error is its result's.
func TestQueryConnMapsTheRequest(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := "query_map"
	admin := server.openRaw(ctx, t, "")
	exec(t, admin, "DROP DATABASE IF EXISTS "+database, "CREATE DATABASE "+database)
	t.Cleanup(func() { exec(t, admin, "DROP DATABASE IF EXISTS "+database) })
	driver := server.open(ctx, t, database)
	raw := server.openRaw(ctx, t, database)
	exec(t, raw,
		"CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(32))",
		"INSERT INTO t VALUES (1, 'a'), (2, 'b'), (3, 'c')",
	)

	t.Run("limit", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT id FROM t ORDER BY id", db.QueryContext{MaximumSQLResultSize: 1 << 20, Limit: 2})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "SELECT id FROM t ORDER BY id LIMIT 2", results[0].Statement)
		require.Len(t, results[0].Rows, 2)
	})

	t.Run("explain json", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT * FROM t", db.QueryContext{
			MaximumSQLResultSize: 1 << 20,
			Explain:              true,
			Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "EXPLAIN FORMAT=JSON SELECT * FROM t", results[0].Statement)
		require.NotEmpty(t, results[0].Rows)
		// The request named the format; the typed plan is not repeated on it.
		require.Nil(t, results[0].QueryPlan)
	})

	t.Run("affected rows", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "UPDATE t SET name = 'z' WHERE id = 1", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, []string{"Affected Rows"}, results[0].ColumnNames)
		require.Equal(t, int64(1), results[0].Rows[0].Values[0].GetInt64Value())
	})

	t.Run("statement error", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT * FROM no_such_table", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Contains(t, results[0].Error, "doesn't exist")
	})

	t.Run("later results stand after an error", func(t *testing.T) {
		results, err := driver.QueryConn(ctx, nil, "SELECT 1 AS a; SELECT * FROM no_such_table; SELECT 3 AS c", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Empty(t, results[0].Error)
		require.Contains(t, results[1].Error, "doesn't exist")
	})

	t.Run("syntax error fails the call", func(t *testing.T) {
		// Known difference: the old driver sent the statement and returned the
		// server's syntax error as that statement's result. The driver reads
		// every statement of a script before it sends the first, so an
		// unreadable script is the call's error and no statement runs.
		_, err := driver.QueryConn(ctx, nil, "SELECT 1 AS a; SELECT FROM", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.ErrorContains(t, err, "syntax error")
	})
}

// TestGetDBReturnsNoHandle pins the contract: the editor's *sql.Conn is not the
// driver's session, so there is no *sql.DB to hand out. The API already guards
// a nil handle.
func TestGetDBReturnsNoHandle(t *testing.T) {
	server := newOceanBaseTestServer(t)
	driver := server.open(context.Background(), t, "")
	require.Nil(t, driver.GetDB())
}
