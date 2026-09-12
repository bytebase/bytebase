package tidb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

// dryRunFake stands in for TiDB. It records every statement the advisor sends,
// prefixed with "tx: " or "autocommit: ", and rejects any statement containing
// failOn.
var dryRunFake struct {
	queries []string
	failOn  string
}

func init() {
	sql.Register("test_tidb_dml_dry_run", dryRunFakeDriver{})
}

type dryRunFakeDriver struct{}

func (dryRunFakeDriver) Open(string) (driver.Conn, error) { return &dryRunFakeConn{}, nil }

type dryRunFakeConn struct{ inTx bool }

func (*dryRunFakeConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (*dryRunFakeConn) Close() error                        { return nil }
func (c *dryRunFakeConn) Begin() (driver.Tx, error)         { c.inTx = true; return c, nil }
func (c *dryRunFakeConn) Commit() error                     { c.inTx = false; return nil }
func (c *dryRunFakeConn) Rollback() error                   { c.inTx = false; return nil }

func (c *dryRunFakeConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	prefix := "autocommit: "
	if c.inTx {
		prefix = "tx: "
	}
	dryRunFake.queries = append(dryRunFake.queries, prefix+query)
	if dryRunFake.failOn != "" && strings.Contains(query, dryRunFake.failOn) {
		return nil, errors.New("rejected: " + query)
	}
	return dryRunFakeRows{}, nil
}

type dryRunFakeRows struct{}

func (dryRunFakeRows) Columns() []string         { return []string{"id"} }
func (dryRunFakeRows) Close() error              { return nil }
func (dryRunFakeRows) Next([]driver.Value) error { return io.EOF }

func TestStatementDMLDryRun(t *testing.T) {
	db, err := sql.Open("test_tidb_dml_dry_run", "")
	require.NoError(t, err)
	defer db.Close()

	sm := sheet.NewManager()
	rules := []*storepb.SQLReviewRule{{Type: storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, Level: storepb.SQLReviewRule_WARNING, Engine: storepb.Engine_TIDB}}

	tests := []struct {
		name        string
		statement   string
		failOn      string
		wantQueries []string
		wantAdvice  int
	}{
		{
			name:        "plain dml is explained in a transaction",
			statement:   "DELETE FROM t WHERE id = 1;",
			wantQueries: []string{"tx: EXPLAIN DELETE FROM t WHERE id = 1"},
		},
		{
			name:        "explain failure is reported",
			statement:   "DELETE FROM t WHERE missing_col = 1",
			failOn:      "missing_col",
			wantQueries: []string{"tx: EXPLAIN DELETE FROM t WHERE missing_col = 1"},
			wantAdvice:  1,
		},
		{
			name:      "batch dry runs in autocommit then explains the inner dml",
			statement: "BATCH ON id LIMIT 2 DELETE FROM t WHERE id > 0",
			wantQueries: []string{
				"autocommit: BATCH ON id LIMIT 2 DRY RUN DELETE FROM t WHERE id > 0",
				"tx: EXPLAIN DELETE FROM t WHERE id > 0",
			},
		},
		{
			name:      "batch update",
			statement: "BATCH ON id LIMIT 2 UPDATE t SET name = 'y' WHERE id > 0",
			wantQueries: []string{
				"autocommit: BATCH ON id LIMIT 2 DRY RUN UPDATE t SET name = 'y' WHERE id > 0",
				"tx: EXPLAIN UPDATE t SET name = 'y' WHERE id > 0",
			},
		},
		{
			name:      "batch insert select",
			statement: "BATCH ON id LIMIT 2 INSERT INTO t2 SELECT id, name FROM t",
			wantQueries: []string{
				"autocommit: BATCH ON id LIMIT 2 DRY RUN INSERT INTO t2 SELECT id, name FROM t",
				"tx: EXPLAIN INSERT INTO t2 SELECT id, name FROM t",
			},
		},
		{
			name:        "batch dry run failure skips the explain",
			statement:   "BATCH ON missing_col LIMIT 2 DELETE FROM t WHERE id > 0",
			failOn:      "missing_col",
			wantQueries: []string{"autocommit: BATCH ON missing_col LIMIT 2 DRY RUN DELETE FROM t WHERE id > 0"},
			wantAdvice:  1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dryRunFake.queries = nil
			dryRunFake.failOn = tc.failOn

			advices, err := advisor.SQLReviewCheck(context.Background(), sm, tc.statement, rules, advisor.Context{
				DBType:          storepb.Engine_TIDB,
				Driver:          db,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantQueries, dryRunFake.queries)
			require.Len(t, advices, tc.wantAdvice)
			for _, a := range advices {
				require.Equal(t, code.StatementDMLDryRunFailed.Int32(), a.Code)
			}
		})
	}
}
