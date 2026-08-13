package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var testOceanBaseExplainRows [][]driver.Value
var testMySQLExplainRows [][]driver.Value

// testMySQLExplainJSON is returned for `EXPLAIN FORMAT=JSON` queries; when empty, the
// JSON query returns no rows and CountAffectedRows falls back to the tabular rows.
var testMySQLExplainJSON string

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
	return &testOceanBaseExplainResultRows{rows: testOceanBaseExplainRows}, nil
}

type testOceanBaseExplainResultRows struct {
	rows [][]driver.Value
	idx  int
}

func (*testOceanBaseExplainResultRows) Columns() []string {
	return []string{"Query Plan"}
}

func (*testOceanBaseExplainResultRows) Close() error {
	return nil
}

func (r *testOceanBaseExplainResultRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

type testMySQLExplainDriver struct{}

func (testMySQLExplainDriver) Open(string) (driver.Conn, error) {
	return testMySQLExplainConn{}, nil
}

type testMySQLExplainConn struct{}

func (testMySQLExplainConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (testMySQLExplainConn) Close() error {
	return nil
}

func (testMySQLExplainConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (testMySQLExplainConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.HasPrefix(query, "EXPLAIN FORMAT=JSON") {
		var rows [][]driver.Value
		if testMySQLExplainJSON != "" {
			rows = [][]driver.Value{{testMySQLExplainJSON}}
		}
		return &testMySQLExplainResultRows{columns: []string{"EXPLAIN"}, rows: rows}, nil
	}
	return &testMySQLExplainResultRows{
		columns: []string{"id", "select_type", "table", "type", "rows", "filtered"},
		rows:    testMySQLExplainRows,
	}, nil
}

type testMySQLExplainResultRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *testMySQLExplainResultRows) Columns() []string {
	return r.columns
}

func (*testMySQLExplainResultRows) Close() error {
	return nil
}

func (r *testMySQLExplainResultRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

func TestCountAffectedRowsCapsExplainEstimateByLimit(t *testing.T) {
	testMySQLExplainJSON = ""
	testMySQLExplainRows = [][]driver.Value{
		{int64(1), "SIMPLE", "td", "ALL", int64(1000), "100.00"},
	}
	db, err := sql.Open("test_mysql_explain", "")
	require.NoError(t, err)
	defer db.Close()

	driver := &Driver{db: db}

	for _, tc := range []struct {
		statement string
		want      int64
	}{
		{
			statement: "UPDATE td SET c = 1 WHERE c = 0 LIMIT 10;",
			want:      10,
		},
		{
			statement: "DELETE FROM td WHERE c = 0 LIMIT 20;",
			want:      20,
		},
		{
			statement: "INSERT INTO td SELECT * FROM source WHERE c = 0 LIMIT 30;",
			want:      30,
		},
		{
			statement: "UPDATE td SET c = 1 WHERE c = 0;",
			want:      1000,
		},
	} {
		got, err := driver.CountAffectedRows(context.Background(), tc.statement)
		require.NoError(t, err)
		require.Equal(t, tc.want, got, tc.statement)
	}
}

func TestCountAffectedRowsPrefersJSONPlanTargetNode(t *testing.T) {
	// The tabular EXPLAIN's first row is a driving-table scan estimate; the JSON plan
	// flags the update target whose cumulative estimate is far smaller (BYT-9858).
	testMySQLExplainJSON = `{"query_block":{"select_id":1,"nested_loop":[
		{"table":{"table_name":"t","access_type":"ALL","rows_examined_per_scan":100232,"rows_produced_per_join":3006,"filtered":"3.00"}},
		{"table":{"update":true,"table_name":"o","access_type":"ref","rows_examined_per_scan":3,"rows_produced_per_join":1144,"filtered":"10.00"}}
	]}}`
	testMySQLExplainRows = [][]driver.Value{
		{int64(1), "SIMPLE", "t", "ALL", int64(100232), "3.00"},
		{int64(1), "UPDATE", "o", "ref", int64(3), "10.00"},
	}
	defer func() { testMySQLExplainJSON = "" }()

	db, err := sql.Open("test_mysql_explain", "")
	require.NoError(t, err)
	defer db.Close()

	driver := &Driver{db: db}

	got, err := driver.CountAffectedRows(context.Background(), "UPDATE o SET c = 1 WHERE c IS NULL AND EXISTS (SELECT 1 FROM t WHERE t.k = o.k);")
	require.NoError(t, err)
	require.Equal(t, int64(1144), got)

	// The statement LIMIT still caps the JSON-plan estimate.
	got, err = driver.CountAffectedRows(context.Background(), "UPDATE o SET c = 1 WHERE c IS NULL LIMIT 10;")
	require.NoError(t, err)
	require.Equal(t, int64(10), got)
}

func TestCountAffectedRowsForOceanBaseConcatenatesExplainRows(t *testing.T) {
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
	db, err := sql.Open("test_oceanbase_explain", "")
	require.NoError(t, err)
	defer db.Close()

	got, err := countAffectedRowsForOceanBase(context.Background(), db, "update dba_test_1 set log_id=1 where id < 1000;")
	require.NoError(t, err)
	require.Equal(t, int64(1000), got)
}
