// Package spanner is the plugin for BigQuery driver.
package bigquery

import (
	"context"
	"database/sql"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_BIGQUERY, newDriver)
}

// Driver is the Spanner driver.
type Driver struct {
	// config  db.ConnectionConfig
	// connCtx db.ConnectionContext

	// databaseName is the currently connected database name.
	// databaseName string
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Spanner driver. It must connect to a specific database.
// If database isn't provided, part of the driver cannot function.
func (*Driver) Open(_ context.Context, _ storepb.Engine, _ db.ConnectionConfig) (db.Driver, error) {
	return nil, nil
}

// Close closes the driver.
func (*Driver) Close(_ context.Context) error {
	return nil
}

// Ping pings the instance.
func (*Driver) Ping(_ context.Context) error {
	return nil
}

// GetType returns the database type.
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_BIGQUERY
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute executes a SQL statement.
func (*Driver) Execute(_ context.Context, _ string, _ db.ExecuteOptions) (int64, error) {
	return 0, nil
}

// QueryConn queries a SQL statement in a given connection.
func (*Driver) QueryConn(_ context.Context, _ *sql.Conn, _ string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, nil
}

// RunStatement executes a SQL statement.
func (*Driver) RunStatement(_ context.Context, _ *sql.Conn, _ string) ([]*v1pb.QueryResult, error) {
	return nil, nil
}
