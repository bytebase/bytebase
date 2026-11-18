package oceanbase

import (
	"context"
	"database/sql"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// oceanBaseDSNEnv names a MySQL-mode OceanBase tenant user that can create databases and users,
// such as root@test:@tcp(127.0.0.1:2881)/. OceanBase needs about 8 GB, more than CI has; omni's
// scripts/oceanbase-oracle.sh boots one with secure_file_priv set and exports the variable.
const oceanBaseDSNEnv = "OCEANBASE_WALK_DSN"

type oceanBaseTestServer struct {
	dsn        string
	host, port string
	user       string
	password   string
	// tenant is the login suffix after "@" that every user of the tenant signs in with.
	tenant string
}

func newOceanBaseTestServer(t *testing.T) oceanBaseTestServer {
	t.Helper()
	dsn := os.Getenv(oceanBaseDSNEnv)
	if dsn == "" {
		t.Skipf("%s is not set", oceanBaseDSNEnv)
	}
	cfg, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	host, port, err := net.SplitHostPort(cfg.Addr)
	require.NoError(t, err)
	_, tenant, _ := strings.Cut(cfg.User, "@")
	return oceanBaseTestServer{dsn: dsn, host: host, port: port, user: cfg.User, password: cfg.Passwd, tenant: tenant}
}

// open returns the adapter open on database, for the tests that exercise the
// driver the product uses.
func (s oceanBaseTestServer) open(ctx context.Context, t *testing.T, database string) *Driver {
	t.Helper()
	opened, err := newDriver().Open(ctx, storepb.Engine_OCEANBASE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: s.user,
			Host:     s.host,
			Port:     s.port,
		},
		Password:          s.password,
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, opened.Close(ctx)) })
	driver, ok := opened.(*Driver)
	require.True(t, ok)
	return driver
}

// openRaw returns a connection of the test's own, for creating fixtures and for
// reading the server's catalogue independently of what the adapter syncs.
func (s oceanBaseTestServer) openRaw(_ context.Context, t *testing.T, database string) *sql.DB {
	t.Helper()
	cfg, err := mysql.ParseDSN(s.dsn)
	require.NoError(t, err)
	cfg.DBName = database
	cfg.MultiStatements = true
	connector, err := mysql.NewConnector(cfg)
	require.NoError(t, err)
	opened := sql.OpenDB(connector)
	t.Cleanup(func() { _ = opened.Close() })
	return opened
}

// exec runs each statement on the connection, which holds one statement's text
// even when it is a compound body.
func exec(t *testing.T, connection *sql.DB, statements ...string) {
	t.Helper()
	for _, statement := range statements {
		_, err := connection.ExecContext(context.Background(), statement)
		require.NoError(t, err, statement)
	}
}
