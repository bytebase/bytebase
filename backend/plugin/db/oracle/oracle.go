// Package snowflake is the plugin for Snowflake driver.
package snowflake

import (
	"context"
	"database/sql"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Oracle, newDriver)
}

// Driver is the Oracle driver.
type Driver struct {
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Snowflake driver.
func (*Driver) Open(_ context.Context, _ db.Type, _ db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
	// TODO(d): implement it.
	return nil, nil
}

// Close closes the driver.
func (*Driver) Close(_ context.Context) error {
	// TODO(d): implement it.
	return nil
}

// Ping pings the database.
func (*Driver) Ping(_ context.Context) error {
	// TODO(d): implement it.
	return nil
}

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.Oracle
}

// GetDBConnection gets a database connection.
func (*Driver) GetDBConnection(_ context.Context, _ string) (*sql.DB, error) {
	// TODO(d): implement it.
	return nil, nil
}

// Execute executes a SQL statement and returns the affected rows.
func (*Driver) Execute(_ context.Context, _ string, _ bool) (int64, error) {
	// TODO(d): implement it.
	return 0, nil
}

// QueryConn querys a SQL statement in a given connection.
func (*Driver) QueryConn(_ context.Context, _ *sql.Conn, _ string, _ *db.QueryContext) ([]interface{}, error) {
	// TODO(d): implement it.
	return nil, nil
}
