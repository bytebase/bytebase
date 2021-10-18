package clickhouse

import (
	"bufio"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"io"
	"strings"

	clickhouse "github.com/ClickHouse/clickhouse-go"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"go.uber.org/zap"
)

//go:embed clickhouse_migration_schema.sql
var migrationSchema string

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.ClickHouse, newDriver)
}

type Driver struct {
	l             *zap.Logger
	connectionCtx db.ConnectionContext
	dbType        db.Type

	db *sql.DB
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{
		l: config.Logger,
	}
}

func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	protocol := "tcp"
	if strings.HasPrefix(config.Host, "/") {
		protocol = "unix"
	}

	params := []string{}

	port := config.Port
	if port == "" {
		port = "9000"
	}
	baseDSN := fmt.Sprintf("%s://%s:%s?", protocol, config.Host, port)

	// Default user name is "default".
	params = append(params, fmt.Sprintf("username=%s", config.Username))
	// Password is constructed later, after loggedDSN is constructed.
	if config.Database != "" {
		params = append(params, fmt.Sprintf("database=%s", config.Database))
	}
	// Set SSL configuration.
	tlsConfig, err := config.TlsConfig.GetSslConfig()
	if err != nil {
		return nil, fmt.Errorf("sql: tls config error: %v", err)
	}
	tlsKey := "db.clickhouse.tls"
	if tlsConfig != nil {
		if err := clickhouse.RegisterTLSConfig(tlsKey, tlsConfig); err != nil {
			return nil, fmt.Errorf("sql: failed to register tls config: %v", err)
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		defer clickhouse.DeregisterTLSConfig(tlsKey)
		params = append(params, fmt.Sprintf("tls_config=%s", tlsKey))
	}

	loggedDSN := fmt.Sprintf("%s%s&password=<<redacted password>>", baseDSN, strings.Join(params, "&"))

	if config.Password != "" {
		params = append(params, fmt.Sprintf("password=%s", config.Password))
	}
	dsn := fmt.Sprintf("%s%s", baseDSN, strings.Join(params, "&"))

	driver.l.Debug("Opening ClickHouse driver",
		zap.String("dsn", loggedDSN),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		panic(err)
	}
	driver.dbType = dbType
	driver.db = db
	driver.connectionCtx = connCtx

	return driver, nil
}

func (driver *Driver) Close(ctx context.Context) error {
	return driver.db.Close()
}

func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

func (driver *Driver) GetDbConnection(ctx context.Context, database string) (*sql.DB, error) {
	return driver.db, nil
}

func (driver *Driver) GetVersion(ctx context.Context) (string, error) {
	query := "SELECT VERSION()"
	versionRow, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return "", util.FormatErrorWithQuery(err, query)
	}
	defer versionRow.Close()

	var version string
	versionRow.Next()
	if err := versionRow.Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.DBUser, []*db.DBSchema, error) {
	// TODO(spinningbot): implement it.
	return nil, nil, nil
}

func (driver *Driver) Execute(ctx context.Context, statement string) error {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, statement)

	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}

// Migration related
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	// TODO(spinningbot): implement it.
	return false, nil
}

func (driver *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	// TODO(spinningbot): implement it.
	return nil
}

// ExecuteMigration will execute the migration for MySQL.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	// TODO(spinningbot): implement it.
	return 0, "", nil
}

func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	return nil, nil
}

func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	// TODO(spinningbot): implement it.
	return nil
}

func (driver *Driver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	// TODO(spinningbot): implement it.
	return nil
}
