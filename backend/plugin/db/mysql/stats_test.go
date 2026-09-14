package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

var testOceanBaseExplainRows [][]driver.Value

var (
	// testMySQLExplainJSON is the plan the fake MySQL server returns for EXPLAIN FORMAT=JSON.
	testMySQLExplainJSON string
	// testMySQLSetErr is the error the fake MySQL server returns for SET statements.
	testMySQLSetErr error
	// testMySQLStatements records every statement the fake MySQL server receives, in order.
	testMySQLStatements  []testMySQLStatement
	testMySQLConnections int
)

type testMySQLStatement struct {
	conn  int
	query string
}

func init() {
	sql.Register("test_oceanbase_explain", testOceanBaseExplainDriver{})
	sql.Register("test_mysql_explain", testMySQLExplainDriver{})
}

type testOceanBaseExplainDriver struct{}

func (testOceanBaseExplainDriver) Open(string) (driver.Conn, error) {
	return testOceanBaseExplainConn{}, nil
}

type testOceanBaseExplainConn struct{}

func (testOceanBaseExplainConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (testOceanBaseExplainConn) Close() error {
	return nil
}

func (testOceanBaseExplainConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (testOceanBaseExplainConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &testExplainResultRows{rows: testOceanBaseExplainRows}, nil
}

type testMySQLExplainDriver struct{}

func (testMySQLExplainDriver) Open(string) (driver.Conn, error) {
	testMySQLConnections++
	return &testMySQLExplainConn{id: testMySQLConnections}, nil
}

type testMySQLExplainConn struct {
	id int
}

func (*testMySQLExplainConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (*testMySQLExplainConn) Close() error {
	return nil
}

func (*testMySQLExplainConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (c *testMySQLExplainConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	testMySQLStatements = append(testMySQLStatements, testMySQLStatement{conn: c.id, query: query})
	if !strings.HasPrefix(query, "SET ") {
		return nil, errors.Errorf("unexpected exec %q", query)
	}
	if testMySQLSetErr != nil {
		return nil, testMySQLSetErr
	}
	return driver.ResultNoRows, nil
}

func (c *testMySQLExplainConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	testMySQLStatements = append(testMySQLStatements, testMySQLStatement{conn: c.id, query: query})
	if !strings.HasPrefix(query, "EXPLAIN FORMAT=JSON ") {
		return nil, errors.Errorf("unexpected query %q", query)
	}
	return &testExplainResultRows{rows: [][]driver.Value{{testMySQLExplainJSON}}}, nil
}

type testExplainResultRows struct {
	rows [][]driver.Value
	idx  int
}

func (*testExplainResultRows) Columns() []string {
	return []string{"EXPLAIN"}
}

func (*testExplainResultRows) Close() error {
	return nil
}

func (r *testExplainResultRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

// newTestMySQLDriver returns a driver over the fake MySQL server whose pool keeps no idle
// connections, so statements that do not share a pinned connection land on different ones.
func newTestMySQLDriver(t *testing.T, plan string) *Driver {
	t.Helper()
	testMySQLExplainJSON = plan
	testMySQLSetErr = nil
	testMySQLStatements = nil
	db, err := sql.Open("test_mysql_explain", "")
	require.NoError(t, err)
	db.SetMaxIdleConns(0)
	t.Cleanup(func() { _ = db.Close() })
	return &Driver{dbType: storepb.Engine_MYSQL, db: db}
}

const (
	// The flagged UPDATE target is scanned first; the join filter on s comes after it.
	testUpdateTargetFirstPlan = `{"query_block":{"select_id":1,"nested_loop":[
		{"table":{"update":true,"table_name":"big","access_type":"ALL","rows_examined_per_scan":11330,"rows_produced_per_join":11330,"filtered":"100.00"}},
		{"table":{"table_name":"s","access_type":"eq_ref","rows_examined_per_scan":1,"rows_produced_per_join":1133,"filtered":"10.00"}}
	]}}`
	testInsertSelectPlan = `{"query_block":{"select_id":1,
		"table":{"insert":true,"table_name":"t2","access_type":"ALL"},
		"insert_from":{"table":{"table_name":"t","access_type":"ALL","rows_examined_per_scan":1000,"rows_produced_per_join":100,"filtered":"10.00"}}
	}}`
)

func TestCountAffectedRowsEstimatesFromJSONPlan(t *testing.T) {
	// A single-table plan of the SELECT a single-table UPDATE or DELETE is explained as.
	const filteredScanPlan = `{"query_block":{"select_id":1,"table":{"table_name":"td","access_type":"ALL","rows_examined_per_scan":1000,"rows_produced_per_join":100,"filtered":"10.00"}}}`
	for _, tc := range []struct {
		name      string
		plan      string
		statement string
		explained string
		want      int64
	}{
		{
			name:      "update bounded by the join after the target",
			plan:      testUpdateTargetFirstPlan,
			statement: "UPDATE big STRAIGHT_JOIN s ON big.s_id = s.id SET big.v = big.v + 1 WHERE s.flag = 1;",
			explained: "UPDATE big STRAIGHT_JOIN s ON big.s_id = s.id SET big.v = big.v + 1 WHERE s.flag = 1;",
			want:      1133,
		},
		{
			name:      "single-table update explained as a select",
			plan:      filteredScanPlan,
			statement: "UPDATE td SET c = 1 WHERE d = 0;",
			explained: "SELECT 1 FROM td WHERE d = 0",
			want:      100,
		},
		{
			name:      "update capped by LIMIT",
			plan:      filteredScanPlan,
			statement: "UPDATE td SET c = 1 WHERE d = 0 LIMIT 10;",
			explained: "SELECT 1 FROM td WHERE d = 0 LIMIT 10",
			want:      10,
		},
		{
			name:      "delete capped by LIMIT",
			plan:      filteredScanPlan,
			statement: "DELETE FROM td WHERE d = 0 ORDER BY id LIMIT 20;",
			explained: "SELECT 1 FROM td WHERE d = 0 ORDER BY id LIMIT 20",
			want:      20,
		},
		{
			name:      "insert select reads the source plan",
			plan:      testInsertSelectPlan,
			statement: "INSERT INTO t2 SELECT * FROM t WHERE d = 3;",
			explained: "INSERT INTO t2 SELECT * FROM t WHERE d = 3;",
			want:      100,
		},
		{
			name:      "insert select capped by LIMIT",
			plan:      testInsertSelectPlan,
			statement: "INSERT INTO t2 SELECT * FROM t WHERE d = 3 LIMIT 30;",
			explained: "INSERT INTO t2 SELECT * FROM t WHERE d = 3 LIMIT 30;",
			want:      30,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestMySQLDriver(t, tc.plan)
			got, err := d.CountAffectedRows(context.Background(), tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, "EXPLAIN FORMAT=JSON "+tc.explained, testMySQLStatements[1].query)
		})
	}
}

func TestCountAffectedRowsScopesJSONFormatVersionToTheExplain(t *testing.T) {
	const statement = "INSERT INTO t2 SELECT * FROM t WHERE d = 3;"
	for _, tc := range []struct {
		name           string
		setErr         error
		wantStatements []string
	}{
		{
			name: "server with explain_json_format_version",
			wantStatements: []string{
				"SET SESSION explain_json_format_version = 1",
				"EXPLAIN FORMAT=JSON " + statement,
				"SET SESSION explain_json_format_version = DEFAULT",
			},
		},
		{
			name:   "server without explain_json_format_version",
			setErr: &mysql.MySQLError{Number: 1193, Message: "Unknown system variable 'explain_json_format_version'"},
			wantStatements: []string{
				"SET SESSION explain_json_format_version = 1",
				"EXPLAIN FORMAT=JSON " + statement,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestMySQLDriver(t, testInsertSelectPlan)
			testMySQLSetErr = tc.setErr

			got, err := d.CountAffectedRows(context.Background(), statement)
			require.NoError(t, err)
			require.Equal(t, int64(100), got)
			var statements []string
			for _, s := range testMySQLStatements {
				statements = append(statements, s.query)
				require.Equal(t, testMySQLStatements[0].conn, s.conn)
			}
			require.Equal(t, tc.wantStatements, statements)
		})
	}

	t.Run("other SET failures stop the estimate", func(t *testing.T) {
		d := newTestMySQLDriver(t, testInsertSelectPlan)
		testMySQLSetErr = &mysql.MySQLError{Number: 1227, Message: "Access denied"}

		_, err := d.CountAffectedRows(context.Background(), statement)
		require.ErrorContains(t, err, "failed to set explain_json_format_version")
		require.Len(t, testMySQLStatements, 1)
	})
}

func TestCountAffectedRowsReportsUninterpretablePlan(t *testing.T) {
	// A plan without the target table must fail rather than read as zero rows.
	d := newTestMySQLDriver(t, `{"query_block":{"select_id":1,"table":{"table_name":"other","access_type":"ALL","rows_examined_per_scan":1000,"filtered":"100.00"}}}`)
	_, err := d.CountAffectedRows(context.Background(), "UPDATE td SET c = 1;")
	require.ErrorContains(t, err, `target table "td" is not in the plan`)
}

func TestCountAffectedRowsForOceanBase(t *testing.T) {
	db, err := sql.Open("test_oceanbase_explain", "")
	require.NoError(t, err)
	defer db.Close()
	d := &Driver{dbType: storepb.Engine_OCEANBASE, db: db}

	// OceanBase splits the JSON plan across rows.
	testOceanBaseExplainRows = [][]driver.Value{
		{`{`},
		{`  "ID":0,`},
		{`  "OPERATOR":"UPDATE",`},
		{`  "NAME":"",`},
		{`  "EST.ROWS":1000,`},
		{`  "EST.TIME(us)":7680,`},
		{`  "output":"",`},
		{`  "CHILD_1": {`},
		{`    "ID":1,`},
		{`    "OPERATOR":"TABLE RANGE SCAN",`},
		{`    "NAME":"dba_test_1",`},
		{`    "EST.ROWS":1000,`},
		{`    "EST.TIME(us)":91,`},
		{`    "output":"output([dba_test_1.id], [dba_test_1.log_id])"`},
		{`  }`},
		{`}`},
	}
	got, err := d.CountAffectedRows(context.Background(), "update dba_test_1 set log_id=1 where id < 1000;")
	require.NoError(t, err)
	require.Equal(t, int64(1000), got)

	testOceanBaseExplainRows = [][]driver.Value{{`{"ID":0,"OPERATOR":"UPDATE","NAME":""}`}}
	_, err = d.CountAffectedRows(context.Background(), "update dba_test_1 set log_id=1 where id < 1000;")
	require.ErrorContains(t, err, `operator "UPDATE" has no EST.ROWS`)

	testOceanBaseExplainRows = nil
	_, err = d.CountAffectedRows(context.Background(), "update dba_test_1 set log_id=1 where id < 1000;")
	require.Error(t, err)
}
