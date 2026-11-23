//nolint:revive
package util

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// probeFake records the statements a probe runs, and whether each ran inside a
// transaction.
type probeFake struct {
	statements []string
}

func (f *probeFake) Connect(context.Context) (driver.Conn, error) {
	return &probeFakeConn{fake: f}, nil
}
func (f *probeFake) Driver() driver.Driver { return probeFakeDriver{fake: f} }

type probeFakeDriver struct{ fake *probeFake }

func (d probeFakeDriver) Open(string) (driver.Conn, error) { return &probeFakeConn{fake: d.fake}, nil }

type probeFakeConn struct {
	fake *probeFake
	inTx bool
}

func (*probeFakeConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (*probeFakeConn) Close() error                        { return nil }
func (c *probeFakeConn) Begin() (driver.Tx, error)         { c.inTx = true; return c, nil }
func (c *probeFakeConn) Commit() error                     { c.inTx = false; return nil }
func (c *probeFakeConn) Rollback() error                   { c.inTx = false; return nil }

func (c *probeFakeConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.fake.statements = append(c.fake.statements, "exec: "+query)
	return driver.ResultNoRows, nil
}

func (c *probeFakeConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	prefix := "autocommit: "
	if c.inTx {
		prefix = "tx: "
	}
	c.fake.statements = append(c.fake.statements, prefix+query)
	if query == "SELECT refused" {
		return nil, errors.New("refused")
	}
	return &probeFakeRows{}, nil
}

type probeFakeRows struct{ done bool }

func (*probeFakeRows) Columns() []string { return []string{"QUERY PLAN"} }
func (*probeFakeRows) Close() error      { return nil }
func (r *probeFakeRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	dest[0] = "plan"
	r.done = true
	return nil
}

func TestProbeSQL(t *testing.T) {
	ctx := context.Background()
	fake := &probeFake{}
	connection := sql.OpenDB(fake)
	defer connection.Close()

	t.Run("pre-executions and the statement run in one transaction, then roll back", func(t *testing.T) {
		fake.statements = nil
		res, err := ProbeSQL(ctx, connection, db.ProbeContext{PreExecutions: []string{"SET search_path = app"}}, "EXPLAIN SELECT 1")
		require.NoError(t, err)
		require.Equal(t, []string{"exec: SET search_path = app", "tx: EXPLAIN SELECT 1"}, fake.statements)
		require.Equal(t, []string{"QUERY PLAN"}, res.Columns)
		plan, err := res.Plan()
		require.NoError(t, err)
		require.Equal(t, "plan", plan)
	})

	t.Run("autocommit runs the statement alone", func(t *testing.T) {
		fake.statements = nil
		_, err := ProbeSQL(ctx, connection, db.ProbeContext{Autocommit: true}, "EXPLAIN PLAN FOR SELECT 1")
		require.NoError(t, err)
		require.Equal(t, []string{"autocommit: EXPLAIN PLAN FOR SELECT 1"}, fake.statements)
	})

	t.Run("autocommit rejects pre-executions", func(t *testing.T) {
		fake.statements = nil
		_, err := ProbeSQL(ctx, connection, db.ProbeContext{Autocommit: true, PreExecutions: []string{"SET search_path = app"}}, "SELECT 1")
		require.ErrorContains(t, err, "cannot run outside a transaction")
		require.Empty(t, fake.statements)
	})

	t.Run("a refused statement returns the server's error", func(t *testing.T) {
		_, err := ProbeSQL(ctx, connection, db.ProbeContext{}, "SELECT refused")
		require.ErrorContains(t, err, "refused")
	})
}

func TestProbeResultPlan(t *testing.T) {
	// OceanBase splits its plan across rows; PostgreSQL prints it as one cell.
	plan, err := db.ProbeResult{Rows: [][]any{{`{"ID":0,`}, {`"OPERATOR":"X"}`}}}.Plan()
	require.NoError(t, err)
	require.Equal(t, `{"ID":0,"OPERATOR":"X"}`, plan)

	_, err = db.ProbeResult{}.Plan()
	require.Error(t, err)
}

func TestProbeOmni(t *testing.T) {
	ctx := context.Background()

	t.Run("transactional probe prepares, runs pre-executions, rolls back on error", func(t *testing.T) {
		var ran []string
		s := &fakeOmniSession{
			queryFunc: func(_ context.Context, stmt string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				ran = append(ran, stmt)
				if strings.HasPrefix(stmt, "EXPLAIN") {
					return []*omnidriver.Result{{
						Columns: []omnidriver.Column{{Name: "PLAN", TypeName: "TEXT"}},
						Rows:    [][]omnidriver.Value{{{Kind: omnidriver.String, String: "ok"}}},
					}}, nil
				}
				return []*omnidriver.Result{{}}, nil
			},
		}
		q := &fakeOmniDriver{sessions: []*fakeOmniSession{s}}

		res, err := ProbeOmni(ctx, q, db.ProbeContext{PreExecutions: []string{"", "SET a = 1"}}, "EXPLAIN SELECT 1", OmniProbeOptions{
			BeginStatement:           "START TRANSACTION",
			RollbackStatement:        "ROLLBACK",
			RollbackWithoutCancel:    true,
			SkipEmptyPreExecutions:   true,
			CheckPreExecutionResults: true,
			PrepareTx: func(context.Context, omnidriver.Session) ([]string, error) {
				return []string{"SET ROLE owner"}, nil
			},
		})
		require.NoError(t, err)
		require.True(t, s.closed)
		require.Equal(t, []string{"START TRANSACTION", "SET ROLE owner", "SET a = 1", "EXPLAIN SELECT 1", "ROLLBACK"}, ran)
		require.Equal(t, [][]any{{"ok"}}, res.Rows)
	})

	t.Run("autocommit only and pre-execution validation", func(t *testing.T) {
		q := &fakeOmniDriver{}
		_, err := ProbeOmni(ctx, q, db.ProbeContext{Autocommit: true, PreExecutions: []string{"SET a = 1"}}, "SELECT 1", OmniProbeOptions{})
		require.ErrorContains(t, err, "cannot run outside a transaction")

		_, err = ProbeOmni(ctx, q, db.ProbeContext{}, "SELECT 1", OmniProbeOptions{AutocommitOnly: true})
		require.ErrorContains(t, err, "autocommit-only")
		require.Equal(t, 0, q.sessionCalls)
	})

	t.Run("result error triggers rollback and surfaces error", func(t *testing.T) {
		var ran []string
		s := &fakeOmniSession{
			queryFunc: func(_ context.Context, stmt string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				ran = append(ran, stmt)
				if stmt == "EXPLAIN BAD" {
					return []*omnidriver.Result{{Err: errors.New("bad sql")}}, nil
				}
				return []*omnidriver.Result{{}}, nil
			},
		}
		q := &fakeOmniDriver{sessions: []*fakeOmniSession{s}}
		_, err := ProbeOmni(ctx, q, db.ProbeContext{}, "EXPLAIN BAD", OmniProbeOptions{
			BeginStatement:    "BEGIN;",
			RollbackStatement: "ROLLBACK;",
		})
		require.ErrorContains(t, err, "bad sql")
		require.Equal(t, []string{"BEGIN;", "EXPLAIN BAD", "ROLLBACK;"}, ran)
	})
}
