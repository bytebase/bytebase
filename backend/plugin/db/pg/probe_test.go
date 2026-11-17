package pg

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// tenantFake answers the database owner lookup and records every statement the
// probe runs, with the transaction it ran in.
type tenantFake struct {
	conn *tenantFakeConn
}

func (f *tenantFake) Connect(context.Context) (driver.Conn, error) {
	f.conn = &tenantFakeConn{fake: f}
	return f.conn, nil
}

func (f *tenantFake) Driver() driver.Driver { return tenantFakeDriver{fake: f} }

type tenantFakeDriver struct{ fake *tenantFake }

func (d tenantFakeDriver) Open(string) (driver.Conn, error) {
	return d.fake.Connect(context.Background())
}

type tenantFakeConn struct {
	fake *tenantFake
	inTx bool

	statements []string
}

func (*tenantFakeConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (*tenantFakeConn) Close() error                        { return nil }
func (c *tenantFakeConn) Begin() (driver.Tx, error) {
	c.statements = append(c.statements, "begin")
	c.inTx = true
	return c, nil
}
func (c *tenantFakeConn) Commit() error { c.inTx = false; return nil }
func (c *tenantFakeConn) Rollback() error {
	c.statements = append(c.statements, "rollback")
	c.inTx = false
	return nil
}

func (c *tenantFakeConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.statements = append(c.statements, "exec: "+strings.TrimSpace(query))
	return driver.ResultNoRows, nil
}

func (c *tenantFakeConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	prefix := "autocommit: "
	if c.inTx {
		prefix = "tx: "
	}
	c.statements = append(c.statements, prefix+strings.TrimSpace(query))
	if strings.Contains(query, "pg_roles") {
		return &tenantFakeRows{value: "app_owner"}, nil
	}
	return &tenantFakeRows{value: `[{"Plan": {"Node Type": "ModifyTable", "Plan Rows": 1}}]`}, nil
}

type tenantFakeRows struct {
	value string
	done  bool
}

func (*tenantFakeRows) Columns() []string { return []string{"QUERY PLAN"} }
func (*tenantFakeRows) Close() error      { return nil }
func (r *tenantFakeRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	dest[0] = r.value
	r.done = true
	return nil
}

// TestProbeTenantMode proves the probe reads the database owner and sets the
// session role before the statement, in the order the tenant-mode prelude has
// always used, and rolls the transaction back.
func TestProbeTenantMode(t *testing.T) {
	fake := &tenantFake{}
	connection := sql.OpenDB(fake)
	defer connection.Close()

	driver := &Driver{db: connection, connectionCtx: db.ConnectionContext{TenantMode: true}}
	_, err := driver.Probe(context.Background(), db.ProbeContext{PreExecutions: []string{"SET search_path = app"}}, "EXPLAIN (FORMAT JSON) UPDATE t SET c = 1")
	require.NoError(t, err)

	// The owner is read before the transaction, then the role, the caller's settings, and the
	// statement run inside it; the rollback comes after the rows are read.
	var got []string
	for _, statement := range fake.conn.statements {
		got = append(got, strings.Join(strings.Fields(statement), " "))
	}
	require.Equal(t, []string{
		"autocommit: SELECT u.rolname FROM pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid) WHERE d.datname = current_database();",
		"begin",
		"exec: SET SESSION ROLE 'app_owner';",
		"exec: SET search_path = app",
		"tx: EXPLAIN (FORMAT JSON) UPDATE t SET c = 1",
		"rollback",
	}, got)
}

// TestProbeWithoutTenantMode proves a non-tenant probe runs the statement alone.
func TestProbeWithoutTenantMode(t *testing.T) {
	fake := &tenantFake{}
	connection := sql.OpenDB(fake)
	defer connection.Close()

	driver := &Driver{db: connection}
	_, err := driver.Probe(context.Background(), db.ProbeContext{}, "SELECT CURRENT_USER")
	require.NoError(t, err)
	require.Equal(t, []string{"begin", "tx: SELECT CURRENT_USER", "rollback"}, fake.conn.statements)
}
