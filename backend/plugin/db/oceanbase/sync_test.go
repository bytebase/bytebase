package oceanbase

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"maps"
	"net"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// oceanBaseDSNEnv names a MySQL-mode OceanBase tenant user that can create databases and users,
// such as root@test:@tcp(127.0.0.1:2881)/. OceanBase needs about 8 GB, more than CI has; omni's
// scripts/oceanbase-oracle.sh boots one with secure_file_priv set and exports the variable.
const oceanBaseDSNEnv = "OCEANBASE_WALK_DSN"

type oceanBaseTestServer struct {
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
	return oceanBaseTestServer{host: host, port: port, user: cfg.User, password: cfg.Passwd, tenant: tenant}
}

func (s oceanBaseTestServer) open(ctx context.Context, t *testing.T, user, password, database string) *Driver {
	t.Helper()
	d, err := newDriver().Open(ctx, storepb.Engine_OCEANBASE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: user,
			Host:     s.host,
			Port:     s.port,
		},
		Password:          password,
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close(ctx)) })
	driver, ok := d.(*Driver)
	require.True(t, ok)
	return driver
}

// TestSyncTableNamespace covers the OceanBase objects that share the table namespace
// with base tables. A consumer can only conclude that a table name is free if sync lists every
// one of them.
func TestSyncTableNamespace(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	startNano := time.Now().UnixNano()
	database := fmt.Sprintf("sync_namespace_%d", startNano)
	reader := fmt.Sprintf("sync_reader_%d", startNano)

	admin := server.open(ctx, t, server.user, server.password, "")
	_, err := admin.db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", database))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE `%s`", database))
		require.NoError(t, err)
		_, err = admin.db.ExecContext(ctx, fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", reader))
		require.NoError(t, err)
	})

	driver := server.open(ctx, t, server.user, server.password, database)
	for _, statement := range []string{
		"CREATE TABLE t (id INT PRIMARY KEY, a INT) COMMENT 'base'",
		"CREATE VIEW v AS SELECT id, a FROM t",
		"CREATE EXTERNAL TABLE ext (c1 INT, c2 VARCHAR(10)) LOCATION = '/tmp' FORMAT = (TYPE = 'CSV')",
		"CREATE MATERIALIZED VIEW mv COMMENT 'materialized' AS SELECT id, a FROM t",
		"CREATE INDEX idx_mv_a ON mv (a)",
		"CREATE MATERIALIZED VIEW LOG ON t WITH PRIMARY KEY (a) INCLUDING NEW VALUES",
		"CREATE SEQUENCE seq",
	} {
		_, err := driver.db.ExecContext(ctx, statement)
		require.NoError(t, err, statement)
	}
	var container string
	require.NoError(t, driver.db.QueryRowContext(ctx,
		"SELECT CONTAINER_NAME FROM oceanbase.DBA_MVIEWS WHERE OWNER = ? AND MVIEW_NAME = 'mv'", database,
	).Scan(&container))

	census := oceanBaseObjectCensus(ctx, t, driver, database)
	require.ElementsMatch(t, []string{"t", "v", "ext", "mv", "mlog$_t", container}, census)
	for _, name := range census {
		_, err := driver.db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE `%s` (z INT)", name))
		requireMySQLErrorNumber(t, err, 1050, name)
	}
	// A sequence has a namespace of its own.
	_, err = driver.db.ExecContext(ctx, "CREATE TABLE seq (z INT)")
	require.NoError(t, err)
	_, err = driver.db.ExecContext(ctx, "DROP TABLE seq")
	require.NoError(t, err)

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Len(t, metadata.Schemas, 1)
	schema := metadata.Schemas[0]
	require.ElementsMatch(t, census, syncedRelationNames(schema))

	tables := make(map[string]*metadatapb.TableMetadata)
	for _, table := range schema.Tables {
		tables[table.Name] = table
	}
	require.ElementsMatch(t, []string{"t", "mlog$_t", container}, slices.Collect(maps.Keys(tables)))
	require.Equal(t, "materialized", tables[container].Comment)

	require.Len(t, schema.ExternalTables, 1)
	require.Equal(t, "ext", schema.ExternalTables[0].Name)
	require.Equal(t, []string{"c1", "c2"}, columnNames(schema.ExternalTables[0].Columns))

	require.Len(t, schema.MaterializedViews, 1)
	mv := schema.MaterializedViews[0]
	require.Equal(t, "mv", mv.Name)
	require.Equal(t, fmt.Sprintf("select `%[1]s`.`t`.`id` AS `id`,`%[1]s`.`t`.`a` AS `a` from `%[1]s`.`t`", database), mv.Definition)
	require.Equal(t, "materialized", mv.Comment)
	require.True(t, mv.SkipDump)
	require.Len(t, mv.Indexes, 1)
	require.Equal(t, "idx_mv_a", mv.Indexes[0].Name)
	require.Equal(t, []string{"a"}, mv.Indexes[0].Expressions)
	require.Empty(t, cmp.Diff(tables[container].Indexes, mv.Indexes, protocmp.Transform()))

	// Without SELECT on the oceanbase database the materialized views are unreadable; the
	// container table, synced without its view, shows that the list is incomplete.
	_, err = admin.db.ExecContext(ctx, fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY 'Reader_pw1'", reader))
	require.NoError(t, err)
	_, err = admin.db.ExecContext(ctx, fmt.Sprintf("GRANT SELECT, SHOW VIEW ON `%s`.* TO '%s'@'%%'", database, reader))
	require.NoError(t, err)
	asReader := server.open(ctx, t, reader+"@"+server.tenant, "Reader_pw1", database)
	metadata, err = asReader.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Empty(t, metadata.Schemas[0].MaterializedViews)
	require.ElementsMatch(t, []string{"t", "v", "ext", "mlog$_t", container}, syncedRelationNames(metadata.Schemas[0]))
}

func TestSyncEvents(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := fmt.Sprintf("sync_events_%d", time.Now().UnixNano())

	admin := server.open(ctx, t, server.user, server.password, "")
	_, err := admin.db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", database))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE `%s`", database))
		require.NoError(t, err)
	})

	driver := server.open(ctx, t, server.user, server.password, database)
	for _, statement := range []string{
		"CREATE TABLE t (id INT)",
		"CREATE EVENT e_every ON SCHEDULE EVERY 1 DAY STARTS '2030-01-01 00:00:00' ON COMPLETION PRESERVE DISABLE DO INSERT INTO t VALUES (1)",
		"CREATE EVENT e_at ON SCHEDULE AT '2030-06-01 12:00:00' DO BEGIN INSERT INTO t VALUES (2); DELETE FROM t WHERE id = 1; END",
	} {
		_, err := driver.db.ExecContext(ctx, statement)
		require.NoError(t, err, statement)
	}

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	schema := metadata.Schemas[0]
	require.Equal(t, []string{"t"}, syncedRelationNames(schema))
	require.Empty(t, cmp.Diff([]*metadatapb.EventMetadata{
		{Name: "e_at"},
		{Name: "e_every"},
	}, schema.Events, protocmp.Transform()))
}

// OceanBase keeps listing a dropped database's events under its name, so a database created
// again under that name lists an event created again twice.
func TestSyncEventsOfRecreatedDatabase(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := fmt.Sprintf("sync_recreated_%d", time.Now().UnixNano())

	admin := server.open(ctx, t, server.user, server.password, "")
	t.Cleanup(func() {
		_, err := admin.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", database))
		require.NoError(t, err)
	})
	var listed int
	for range 2 {
		_, err := admin.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", database))
		require.NoError(t, err)
		_, err = admin.db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", database))
		require.NoError(t, err)
		driver := server.open(ctx, t, server.user, server.password, database)
		for _, statement := range []string{
			"CREATE TABLE t (id INT)",
			"CREATE EVENT e ON SCHEDULE EVERY 1 DAY STARTS '2030-01-01 00:00:00' DO DELETE FROM t",
		} {
			_, err := driver.db.ExecContext(ctx, statement)
			require.NoError(t, err, statement)
		}
		require.NoError(t, admin.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.EVENTS WHERE EVENT_SCHEMA = ? AND EVENT_NAME = 'e'", database,
		).Scan(&listed))

		metadata, err := driver.SyncDBSchema(ctx)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff([]*metadatapb.EventMetadata{{Name: "e"}}, metadata.Schemas[0].Events, protocmp.Transform()))
	}
	require.Equal(t, 2, listed)
}

// oceanBaseObjectCensus lists every object the catalog holds in the database except the
// database itself and its indexes.
func oceanBaseObjectCensus(ctx context.Context, t *testing.T, driver *Driver, database string) []string {
	t.Helper()
	rows, err := driver.db.QueryContext(ctx,
		"SELECT OBJECT_NAME FROM oceanbase.DBA_OBJECTS WHERE OWNER = ? AND OBJECT_TYPE NOT IN ('DATABASE', 'INDEX')", database)
	require.NoError(t, err)
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

func syncedRelationNames(schema *metadatapb.SchemaMetadata) []string {
	var names []string
	for _, table := range schema.Tables {
		names = append(names, table.Name)
	}
	for _, view := range schema.Views {
		names = append(names, view.Name)
	}
	for _, table := range schema.ExternalTables {
		names = append(names, table.Name)
	}
	for _, view := range schema.MaterializedViews {
		names = append(names, view.Name)
	}
	return names
}

func requireMySQLErrorNumber(t *testing.T, err error, number uint16, msgAndArgs ...any) {
	t.Helper()
	var mysqlErr *mysql.MySQLError
	require.ErrorAs(t, err, &mysqlErr, msgAndArgs...)
	require.Equal(t, number, mysqlErr.Number, msgAndArgs...)
}

var (
	// testOceanBaseMViewRows are the DBA_MVIEWS rows the fake OceanBase server returns.
	testOceanBaseMViewRows [][]driver.Value
	// testOceanBaseMViewErr, when set, fails the DBA_MVIEWS query.
	testOceanBaseMViewErr error
)

func init() {
	sql.Register("test_oceanbase_mviews", testOceanBaseMViewDriver{})
}

type testOceanBaseMViewDriver struct{}

func (testOceanBaseMViewDriver) Open(string) (driver.Conn, error) {
	return testOceanBaseMViewConn{}, nil
}

type testOceanBaseMViewConn struct{}

func (testOceanBaseMViewConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (testOceanBaseMViewConn) Close() error {
	return nil
}

func (testOceanBaseMViewConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (testOceanBaseMViewConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if testOceanBaseMViewErr != nil {
		return nil, testOceanBaseMViewErr
	}
	return &testExplainResultRows{columns: []string{"MVIEW_NAME", "CONTAINER_NAME", "QUERY"}, rows: testOceanBaseMViewRows}, nil
}

func newTestOceanBaseMViewDriver(t *testing.T, rows [][]driver.Value, queryErr error) *Driver {
	t.Helper()
	testOceanBaseMViewRows = rows
	testOceanBaseMViewErr = queryErr
	db, err := sql.Open("test_oceanbase_mviews", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &Driver{db: db}
}

func TestListMaterializedViews(t *testing.T) {
	ctx := context.Background()
	containerIndex := &metadatapb.IndexMetadata{Name: "idx_a", Type: "BTREE", Expressions: []string{"a"}, KeyLength: []int64{-1}, Descending: []bool{false}, Visible: true}
	tables := []*metadatapb.TableMetadata{
		{Name: "t"},
		{Name: "__mv_container_500006", Comment: "materialized", Indexes: []*metadatapb.IndexMetadata{containerIndex}},
	}

	d := newTestOceanBaseMViewDriver(t, [][]driver.Value{
		{"mv", "__mv_container_500006", "select `db`.`t`.`a` AS `a` from `db`.`t`"},
		{"mv_unlisted_container", "__mv_container_500009", "select 1 AS `1`"},
	}, nil)
	views, err := d.listMaterializedViews(ctx, "db", tables)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff([]*metadatapb.MaterializedViewMetadata{
		{Name: "mv", Definition: "select `db`.`t`.`a` AS `a` from `db`.`t`", Comment: "materialized", Indexes: []*metadatapb.IndexMetadata{containerIndex}, SkipDump: true},
		{Name: "mv_unlisted_container", Definition: "select 1 AS `1`", SkipDump: true},
	}, views, protocmp.Transform()))
	require.NotSame(t, containerIndex, views[0].Indexes[0])

	// A failed read, such as one without SELECT on the oceanbase database, does not fail the sync.
	d = newTestOceanBaseMViewDriver(t, nil, &mysql.MySQLError{Number: 1142, Message: "SELECT command denied to user 'reader'@'%' for table 'dba_mviews'"})
	views, err = d.listMaterializedViews(ctx, "db", tables)
	require.NoError(t, err)
	require.Empty(t, views)

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = d.listMaterializedViews(canceled, "db", tables)
	require.ErrorIs(t, err, context.Canceled)
}
