// Package oracle is the plugin for Oracle driver.
package oracle

import (
	"context"
	"database/sql"
	"strconv"

	// Import go-ora Oracle driver.
	"github.com/pkg/errors"
	go_ora "github.com/sijms/go-ora/v2"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Oracle, newDriver)
}

// Driver is the Oracle driver.
type Driver struct {
	db *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Snowflake driver.
func (driver *Driver) Open(_ context.Context, _ db.Type, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return nil, errors.Errorf("invalid port %q", config.Port)
	}
	options := make(map[string]string)
	if config.SID != "" {
		options["SID"] = config.SID
	}
	dsn := go_ora.BuildUrl(config.Host, port, config.ServiceName, config.Username, config.Password, options)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	driver.db = db
	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(_ context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.Oracle
}

// GetDBConnection gets a database connection.
func (driver *Driver) GetDBConnection(_ context.Context, _ string) (*sql.DB, error) {
	return driver.db, nil
}

// Execute executes a SQL statement and returns the affected rows.
func (*Driver) Execute(_ context.Context, _ string, _ bool) (int64, error) {
	// TODO(d): implement it.
	return 0, nil
}

// QueryConn querys a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]interface{}, error) {
	// TODO(d): support multi-statement.
	return util.Query(ctx, db.Oracle, conn, statement, queryContext)
}
