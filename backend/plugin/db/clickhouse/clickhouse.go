// Package clickhouse is the plugin for ClickHouse driver.
package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/standard"
)

var (
	systemDatabases = map[string]bool{
		"system":             true,
		"information_schema": true,
		"INFORMATION_SCHEMA": true,
	}
	systemDatabaseClause = func() string {
		var l []string
		for k := range systemDatabases {
			l = append(l, fmt.Sprintf("'%s'", k))
		}
		return strings.Join(l, ", ")
	}()

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_CLICKHOUSE, newDriver)
}

// Driver is the ClickHouse driver.
type Driver struct {
	connectionCtx        db.ConnectionContext
	dbType               storepb.Engine
	databaseName         string
	maximumSQLResultSize int64

	db *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a ClickHouse driver.
func (driver *Driver) Open(_ context.Context, dbType storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	addr := fmt.Sprintf("%s:%s", config.DataSource.Host, config.DataSource.Port)
	tlsConfig, err := db.GetTLSConfig(config.DataSource)
	if err != nil {
		return nil, errors.Wrap(err, "sql: tls config error")
	}
	// Default user name is "default".
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: config.ConnectionContext.DatabaseName,
			Username: config.DataSource.GetUsername(),
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

	slog.Debug("Opening ClickHouse driver",
		slog.String("addr", addr),
		slog.String("environment", config.ConnectionContext.EnvironmentID),
		slog.String("database", config.ConnectionContext.InstanceID),
	)

	driver.dbType = dbType
	driver.db = conn
	driver.databaseName = config.ConnectionContext.DatabaseName
	driver.connectionCtx = config.ConnectionContext
	driver.maximumSQLResultSize = config.MaximumSQLResultSize

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
func (driver *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	singleSQLs, err := standard.SplitSQL(statement)
	if err != nil {
		return 0, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return 0, nil
	}

	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	totalRowsAffected := int64(0)
	for _, singleSQL := range singleSQLs {
		sqlResult, err := tx.ExecContext(ctx, singleSQL.Text)
		if err != nil {
			return 0, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
		} else {
			totalRowsAffected += rowsAffected
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return totalRowsAffected, err
}
