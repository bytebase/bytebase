// Package spanner is the plugin for Spanner driver.
package spanner

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	spanner "cloud.google.com/go/spanner"
	spannerdb "cloud.google.com/go/spanner/admin/database/apiv1"

	"github.com/bytebase/bytebase/plugin/db"

	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	//go:embed spanner_migration_schema.sql
	migrationSchema string

	createBytebaseDatabaseStatement = `CREATE DATABASE bytebase`

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Spanner, newDriver)
}

// Driver is the Spanner driver.
type Driver struct {
	config   db.ConnectionConfig
	connCtx  db.ConnectionContext
	client   *spanner.Client
	dbClient *spannerdb.DatabaseAdminClient
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Spanner driver.
func (d *Driver) Open(ctx context.Context, _ db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	dsn := fmt.Sprintf("%s/databases/%s", config.Host, config.Database)
	client, err := spanner.NewClient(ctx, dsn, option.WithCredentialsJSON([]byte(config.Password)))
	if err != nil {
		return nil, err
	}
	dbClient, err := spannerdb.NewDatabaseAdminClient(ctx, option.WithCredentialsJSON([]byte(config.Password)))
	if err != nil {
		return nil, err
	}

	d.client = client
	d.dbClient = dbClient
	d.config = config
	d.connCtx = connCtx
	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(_ context.Context) error {
	d.client.Close()
	return d.dbClient.Close()
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	iter := d.client.Single().Query(ctx, spanner.NewStatement("SELECT 1"))
	defer iter.Stop()

	var i int64
	row, err := iter.Next()
	if err != nil {
		return err
	}
	if err := row.Column(0, &i); err != nil {
		return err
	}
	if i != 1 {
		return errors.New("expect to get 1")
	}
	if _, err = iter.Next(); err != iterator.Done {
		return errors.New("expect no more rows")
	}
	return nil
}

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.Spanner
}

// GetDBConnection gets a database connection.
func (*Driver) GetDBConnection(_ context.Context, _ string) (*sql.DB, error) {
	panic("not implemented")
}

// Execute executes a SQL statement.
func (*Driver) Execute(_ context.Context, _ string, _ bool) (int64, error) {
	panic("not implemented")
}

// Query queries a SQL statement.
func (*Driver) Query(_ context.Context, _ string, _ *db.QueryContext) ([]interface{}, error) {
	panic("not implemented")
}

func splitStatement(statement string) []string {
	var res []string
	for _, s := range strings.Split(statement, ";") {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			continue
		}
		res = append(res, trimmed)
	}
	return res
}
