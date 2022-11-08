// Package clickhouse is the plugin for ClickHouse driver.
package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
)

var (
	systemDatabases = map[string]bool{
		"system":             true,
		"information_schema": true,
		"INFORMATION_SCHEMA": true,
	}

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.ClickHouse, newDriver)
}

// Driver is the ClickHouse driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        db.Type

	db *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a ClickHouse driver.
func (driver *Driver) Open(_ context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	port := config.Port
	if port == "" {
		port = "9000"
	}
	addr := fmt.Sprintf("%s:%s", config.Host, port)
	// Set SSL configuration.
	tlsConfig, err := config.TLSConfig.GetSslConfig()
	if err != nil {
		return nil, errors.Wrap(err, "sql: tls config error")
	}
	// Default user name is "default".
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		TLS: tlsConfig,
		Settings: clickhouse.Settings{
			// Use a relative long value to avoid timeout on resource-intenstive query. Example failure:
			// failed: code: 160, message: Estimated query execution time (xxx seconds) is too long. Maximum: yyy. Estimated rows to process: zzzzzzzzz
			"max_execution_time": 300,
		},
		DialTimeout: 10 * time.Second,
	})

	log.Debug("Opening ClickHouse driver",
		zap.String("addr", addr),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)

	util.RegisterStats(string(dbType), config.Database, db)

	driver.dbType = dbType
	driver.db = db
	driver.connectionCtx = connCtx

	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	if driver.db != nil {
		util.UnregisterStats(driver.db)
	}
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetDBConnection gets a database connection.
func (driver *Driver) GetDBConnection(context.Context, string) (*sql.DB, error) {
	return driver.db, nil
}

// getVersion gets the version.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SELECT VERSION()"
	var version string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	return version, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool) error {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	f := func(stmt string) error {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
		return nil
	}

	if err := util.ApplyMultiStatements(strings.NewReader(statement), f); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}

// Query queries a SQL statement.
func (driver *Driver) Query(ctx context.Context, statement string, limit int, readOnly bool) ([]interface{}, error) {
	return util.Query(ctx, driver.dbType, driver.db, statement, limit, readOnly)
}
