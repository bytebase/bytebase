// Package mongodb is the plugin for MongoDB driver.
package mongodb

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/db"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.MongoDB, newDriver)
}

type Driver struct {
	client *mongo.Client
}

func newDriver(dc db.DriverConfig) db.Driver {
	return &Driver{}
}

func (driver *Driver) Open(ctx context.Context, _ db.Type, connCfg db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	connectionURI := getMongoDBConnectionURI(connCfg)
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionURI))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create MongoDB client")
	}
	err = client.Connect(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to MongoDB")
	}
	driver.client = client
	return driver, nil
}

func (driver *Driver) Close(ctx context.Context) error {
	if err := driver.client.Disconnect(ctx); err != nil {
		return errors.Wrap(err, "failed to disconnect MongoDB")
	}
	return nil
}

func (driver *Driver) Ping(ctx context.Context) error {
	if err := driver.client.Ping(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to ping MongoDB")
	}
	return nil
}

func (driver *Driver) GetDBConnection(ctx context.Context, database string) (*sql.DB, error) {
	panic("not implemented")
}

func (driver *Driver) Execute(ctx context.Context, statement string, createDatabase bool) (int64, error) {
	panic("not implemented")
}

func (driver *Driver) Query(ctx context.Context, statement string, queryContext *db.QueryContext) ([]interface{}, error) {
	panic("not implemented")
}

func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMeta, error) {
	panic("not implemented")
}

func (driver *Driver) SyncDBSchema(ctx context.Context, database string) (*db.Schema, error) {
	panic("not implemented")
}

func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	panic("not implemented")
}

func (driver *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	panic("not implemented")
}

func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	panic("not implemented")
}

func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	panic("not implemented")
}

func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) (string, error) {
	panic("not implemented")
}

func (driver *Driver) Restore(ctx context.Context, src io.Reader) error {
	panic("not implemented")
}

func getMongoDBConnectionURI(connConfig db.ConnectionConfig) string {
	connectionURL := "mongodb://"
	if connConfig.SRV {
		connectionURL = "mongodb+srv://"
	}
	if connConfig.Username != "" {
		connectionURL = fmt.Sprintf("%s%s:%s@", connectionURL, connConfig.Username, connConfig.Password)
	}
	connectionURL = fmt.Sprintf("%s%s", connectionURL, connConfig.Host)
	if connConfig.Port != "" {
		connectionURL = fmt.Sprintf("%s:%s", connectionURL, connConfig.Port)
	}
	if connConfig.Database != "" {
		connectionURL = fmt.Sprintf("%s/%s", connectionURL, connConfig.Database)
	}
	return connectionURL
}
