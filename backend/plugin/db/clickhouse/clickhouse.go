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

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
	databaseName  string

	db *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a ClickHouse driver.
func (driver *Driver) Open(_ context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	// Set SSL configuration.
	tlsConfig, err := config.TLSConfig.GetSslConfig()
	if err != nil {
		return nil, errors.Wrap(err, "sql: tls config error")
	}
	// Default user name is "default".
	conn := clickhouse.OpenDB(&clickhouse.Options{
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
		zap.String("environment", connCtx.EnvironmentID),
		zap.String("database", connCtx.InstanceID),
	)

	driver.dbType = dbType
	driver.db = conn
	driver.databaseName = config.Database
	driver.connectionCtx = connCtx

	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.ClickHouse
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
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
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool) (int64, error) {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	totalRowsAffected := int64(0)
	f := func(stmt string) error {
		sqlResult, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			log.Debug("rowsAffected returns error", zap.Error(err))
		} else {
			totalRowsAffected += rowsAffected
		}

		return nil
	}

	if err := util.ApplyMultiStatements(strings.NewReader(statement), f); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return 0, err
}

// QueryConn querys a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]any, error) {
	return util.Query(ctx, driver.dbType, conn, statement, queryContext)
}

// QueryConn2 queries a SQL statement in a given connection.
func (*Driver) QueryConn2(_ context.Context, _ *sql.Conn, _ string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("not implemented")
}
