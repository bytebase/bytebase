// Package spanner is the plugin for BigQuery driver.
package bigquery

import (
	"context"
	"database/sql"
	"errors"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"

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
	config  db.ConnectionConfig
	connCtx db.ConnectionContext
	client  *bigquery.Client

	// databaseName is the currently connected database name.
	// databaseName string
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Spanner driver. It must connect to a specific database.
// If database isn't provided, part of the driver cannot function.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if config.Host == "" {
		return nil, errors.New("host cannot be empty")
	}
	d.config = config
	d.connCtx = config.ConnectionContext

	client, err := bigquery.NewClient(ctx, d.config.Host, option.WithCredentialsJSON([]byte(config.Password)))
	if err != nil {
		return nil, err
	}
	d.client = client
	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(_ context.Context) error {
	return d.client.Close()
}

// Ping pings the instance.
func (d *Driver) Ping(ctx context.Context) error {
	q := d.client.Query("SELECT 1")
	if _, err := q.Read(ctx); err != nil {
		return err
	}
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
