// Package databend is the plugin for Databend driver.
package databend

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	databend "github.com/datafuselabs/databend-go"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/standard"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	db.Register(storepb.Engine_DATABEND, newDriver)
}

// Driver is the Databend driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        storepb.Engine
	databaseName  string

	db *sql.DB
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a Databend driver.
func (d *Driver) Open(_ context.Context, dbType storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	addr := fmt.Sprintf("%s:%s", config.DataSource.Host, config.DataSource.Port)
	sslMode := "enable"
	if !config.DataSource.UseSsl {
		sslMode = databend.SSL_MODE_DISABLE
	}
	databendConfig := databend.Config{
		Tenant:    config.DataSource.Cluster,
		Warehouse: config.DataSource.WarehouseId,
		User:      config.DataSource.Username,
		Password:  config.DataSource.Password,
		Database:  config.ConnectionContext.DatabaseName,
		SSLMode:   sslMode,
		Host:      addr,
		Params:    config.DataSource.ExtraConnectionParameters,
	}
	conn, err := sql.Open("databend", databendConfig.FormatDSN())
	if err != nil {
		return nil, errors.Wrap(err, "sql: open error")
	}
	slog.Debug("Opening Databend driver",
		slog.String("addr", addr),
		slog.String("environment", config.ConnectionContext.EnvironmentID),
		slog.String("database", config.ConnectionContext.InstanceID),
	)

	d.dbType = dbType
	d.db = conn
	d.databaseName = config.ConnectionContext.DatabaseName
	d.connectionCtx = config.ConnectionContext
	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(context.Context) error {
	return d.db.Close()
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// GetDB gets the database.
func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// getVersion gets the version.
func (d *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SELECT VERSION()"
	var version string
	if err := d.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	return version, nil
}

// Execute executes a SQL statement.
func (d *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	singleSQLs, err := standard.SplitSQL(statement)
	if err != nil {
		return 0, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return 0, nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
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
