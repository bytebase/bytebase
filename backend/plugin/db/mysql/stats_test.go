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
	// testMySQLExplainTable is the plan the fake MySQL server returns for a tabular EXPLAIN, with the
	// columns in testMySQLExplainTableColumns.
	testMySQLExplainTable [][]driver.Value
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

var testMySQLExplainTableColumns = []string{"id", "select_type", "table", "rows", "filtered"}

func (c *testMySQLExplainConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	testMySQLStatements = append(testMySQLStatements, testMySQLStatement{conn: c.id, query: query})
	switch {
	case strings.HasPrefix(query, "EXPLAIN FORMAT=JSON "):
		return &testExplainResultRows{rows: [][]driver.Value{{testMySQLExplainJSON}}}, nil
	case strings.HasPrefix(query, "EXPLAIN "):
		return &testExplainResultRows{columns: testMySQLExplainTableColumns, rows: testMySQLExplainTable}, nil
	default:
		return nil, errors.Errorf("unexpected query %q", query)
	}
}

type testExplainResultRows struct {
	// columns defaults to the single EXPLAIN column of a JSON plan.
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *testExplainResultRows) Columns() []string {
	if r.columns != nil {
		return r.columns
	}
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
	testMySQLExplainTable = nil
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

func TestCountAffectedRowsFallsBackToTabularPlan(t *testing.T) {
	// MySQL 8.0 prints no source plan for an INSERT ... SELECT that computes window functions.
	const plan = `{"query_block":{"select_id":1,"table":{"insert":true,"select_id":1,"table_name":"t2","access_type":"ALL"}}}`
	for _, tc := range []struct {
		name      string
		statement string
		// table is the tabular plan, which defaults to a single table with 1000 rows and 33.33% filtered.
		table [][]driver.Value
		want  int64
	}{
		{
			name:      "rows of the first table estimated, scaled by filtered",
			statement: "INSERT INTO t2 SELECT id, ROW_NUMBER() OVER (ORDER BY id) FROM t WHERE c < 10;",
			want:      333,
		},
		{
			name:      "capped by LIMIT",
			statement: "INSERT INTO t2 SELECT id, ROW_NUMBER() OVER (ORDER BY id) FROM t WHERE c < 10 LIMIT 50;",
			want:      50,
		},
		{
			// 3 small rows each join 100 big rows; the subquery's own block does not count.
			name:      "rows joined in the first query block",
			statement: "INSERT INTO t2 SELECT big.id, ROW_NUMBER() OVER () FROM small JOIN big ON big.s_id = small.id WHERE small.id IN (SELECT id FROM s);",
			table: [][]driver.Value{
				{int64(1), "INSERT", "t2", nil, nil},
				{int64(1), "SIMPLE", "small", int64(3), 100.0},
				{int64(1), "SIMPLE", "big", int64(100), 100.0},
				{int64(2), "SUBQUERY", "s", int64(100), 10.0},
			},
			want: 300,
		},
		{
			// Each of the 300 joined rows changes a row of a and a row of b.
			name:      "multi-table update counts each target",
			statement: "UPDATE small JOIN big ON big.s_id = small.id SET small.x = 1, big.v = 1;",
			table: [][]driver.Value{
				{int64(1), "UPDATE", "small", int64(3), 100.0},
				{int64(1), "UPDATE", "big", int64(100), 100.0},
			},
			want: 600,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestMySQLDriver(t, plan)
			testMySQLExplainTable = tc.table
			if testMySQLExplainTable == nil {
				testMySQLExplainTable = [][]driver.Value{
					{int64(1), "INSERT", "t2", nil, nil},
					{int64(1), "SIMPLE", "t", int64(1000), 33.33},
				}
			}
			got, err := d.CountAffectedRows(context.Background(), tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, "EXPLAIN FORMAT=TRADITIONAL "+tc.statement, testMySQLStatements[len(testMySQLStatements)-1].query)
		})
	}
}

func TestCountAffectedRowsReportsUninterpretablePlan(t *testing.T) {
	// A plan without an estimate, with no tabular estimate either, fails rather than reading as zero rows.
	d := newTestMySQLDriver(t, `{"query_block":{"select_id":1,"table":{"insert":true,"select_id":1,"table_name":"t2","access_type":"ALL"}}}`)
	_, err := d.CountAffectedRows(context.Background(), "INSERT INTO t2 SELECT id, ROW_NUMBER() OVER () FROM t;")
	require.ErrorContains(t, err, `table "t2" in the plan has no row estimate`)
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
