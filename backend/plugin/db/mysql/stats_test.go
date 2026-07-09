package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

var testOceanBaseExplainRows [][]driver.Value
var testMySQLExplainRows [][]driver.Value

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

func (testMySQLExplainConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &testMySQLExplainResultRows{rows: testMySQLExplainRows}, nil
}

type testMySQLExplainResultRows struct {
	rows [][]driver.Value
	idx  int
}

func (*testMySQLExplainResultRows) Columns() []string {
	return []string{"id", "select_type", "table", "type", "rows", "filtered"}
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
