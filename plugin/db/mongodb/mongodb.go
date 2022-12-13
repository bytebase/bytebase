// Package mongodb is the plugin for MongoDB driver.
package mongodb

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

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

// Driver is the MongoDB driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	client        *mongo.Client
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a MongoDB driver.
func (driver *Driver) Open(ctx context.Context, _ db.Type, connCfg db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	connectionURI := getMongoDBConnectionURI(connCfg)
	opts := options.Client().ApplyURI(connectionURI)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create MongoDB client")
	}
	driver.client = client
	driver.connectionCtx = connCtx
	return driver, nil
}

// Close closes the MongoDB driver.
func (driver *Driver) Close(ctx context.Context) error {
	if err := driver.client.Disconnect(ctx); err != nil {
		return errors.Wrap(err, "failed to disconnect MongoDB")
	}
	return nil
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	if err := driver.client.Ping(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to ping MongoDB")
	}
	return nil
}

// GetDBConnection returns a database connection.
func (*Driver) GetDBConnection(_ context.Context, _ string) (*sql.DB, error) {
	panic("not implemented")
}

// Execute executes a statement.
func (*Driver) Execute(_ context.Context, _ string, _ bool) (int64, error) {
	panic("not implemented")
}

// Query queries a statement.
func (*Driver) Query(_ context.Context, _ string, _ *db.QueryContext) ([]interface{}, error) {
	panic("not implemented")
}

// SyncInstance syncs the instance meta.
func (*Driver) SyncInstance(_ context.Context) (*db.InstanceMeta, error) {
	panic("not implemented")
}

// SyncDBSchema syncs the database schema.
func (*Driver) SyncDBSchema(_ context.Context, _ string) (*db.Schema, error) {
	panic("not implemented")
}

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ string, _ io.Writer, _ bool) (string, error) {
	panic("not implemented")
}

// Restore restores the backup read from src.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	panic("not implemented")
}

// getMongoDBConnectionURI returns the MongoDB connection URI.
// https://www.mongodb.com/docs/manual/reference/connection-string/
func getMongoDBConnectionURI(connConfig db.ConnectionConfig) string {
	connectionURL := "mongodb://"
	if connConfig.SRV {
		connectionURL = "mongodb+srv://"
	}
	if connConfig.Username != "" {
		percentEncodingUsername := replaceCharacterWithPercentEncoding(connConfig.Username)
		percentEncodingPassword := replaceCharacterWithPercentEncoding(connConfig.Password)
		connectionURL = fmt.Sprintf("%s%s:%s@", connectionURL, percentEncodingUsername, percentEncodingPassword)
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

func replaceCharacterWithPercentEncoding(s string) string {
	m := map[string]string{
		":": `%3A`,
		"/": `%2F`,
		"?": `%3F`,
		"#": `%23`,
		"[": `%5B`,
		"]": `%5D`,
		"@": `%40`,
	}
	for k, v := range m {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}
