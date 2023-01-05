// Package mongodb is the plugin for MongoDB driver.
package mongodb

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/mongoutil"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.MongoDB, newDriver)
}

// Driver is the MongoDB driver.
type Driver struct {
	dbBinDir      string
	connectionCtx db.ConnectionContext
	connCfg       db.ConnectionConfig
	client        *mongo.Client
}

func newDriver(dc db.DriverConfig) db.Driver {
	return &Driver{dbBinDir: dc.DbBinDir}
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
	driver.connCfg = connCfg
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

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.MongoDB
}

// SwitchDatabase switches the connected database.
func (*Driver) SwitchDatabase(context.Context, string) (func() error, error) {
	noop := func() error {
		return nil
	}
	return noop, nil
}

// GetDBConnection returns a database connection.
// It always return nil because it has not implemented the SQL interface, and we always return error, it's caller's responsibility to
// avoid calling this function for MongoDB.
func (*Driver) GetDBConnection(_ context.Context, _ string) (*sql.DB, error) {
	return nil, errors.New("mongodb doesn't support GetDBConnection")
}

// Execute executes a statement, always returns 0 as the number of rows affected because we execute the statement by mongosh, it's hard to catch the row effected number.
func (driver *Driver) Execute(_ context.Context, statement string, _ bool) (int64, error) {
	connectionURI := getMongoDBConnectionURI(driver.connCfg)
	// For MongoDB, we execute the statement in mongosh, which is a shell for MongoDB.
	// There are some ways to execute the statement in mongosh:
	// 1. Use the --eval option to execute the statement.
	// 2. Use the --file option to execute the statement from a file.
	// We choose the second way with the following reasons:
	// 1. The statement may too long to be executed in the command line.
	// 2. We cannot catch the error from the --eval option.

	// First, we create a temporary file to store the statement.
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "mongodb-statement")
	if err != nil {
		return 0, errors.Wrap(err, "failed to create temporary file")
	}
	defer os.Remove(tempFile.Name())
	if _, err := tempFile.WriteString(statement); err != nil {
		return 0, errors.Wrap(err, "failed to write statement to temporary file")
	}
	if err := tempFile.Close(); err != nil {
		return 0, errors.Wrap(err, "failed to close temporary file")
	}

	// Then, we execute the statement in mongosh.
	mongoshArgs := []string{
		connectionURI,
		"--quiet",
		"--file",
		tempFile.Name(),
	}
	// We don't use the CommandContext here because the statement may take a long time to execute.
	mongoshCmd := exec.Command(mongoutil.GetMongoshPath(driver.dbBinDir), mongoshArgs...)
	var errContent bytes.Buffer
	mongoshCmd.Stderr = &errContent
	if err := mongoshCmd.Run(); err != nil {
		return 0, errors.Wrapf(err, "failed to execute statement in mongosh: %s", errContent.String())
	}
	return 0, nil
}

// Query queries a statement.
func (driver *Driver) Query(ctx context.Context, statement string, _ *db.QueryContext) ([]interface{}, error) {
	connectionURI := getMongoDBConnectionURI(driver.connCfg)
	// For MongoDB query, we execute the statement in mongosh with flag --eval for the following reasons:
	// 1. Query always short, so it's safe to execute in the command line.
	// 2. We cannot catch the output if we use the --file option.

	mongoshArgs := []string{
		connectionURI,
		"--quiet",
		"--eval",
		statement,
	}

	mongoshCmd := exec.CommandContext(ctx, mongoutil.GetMongoshPath(driver.dbBinDir), mongoshArgs...)
	var errContent bytes.Buffer
	var outContent bytes.Buffer
	mongoshCmd.Stderr = &errContent
	mongoshCmd.Stdout = &outContent
	if err := mongoshCmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "failed to execute statement in mongosh: %s", errContent.String())
	}
	field := []string{"result"}
	types := []string{"TEXT"}
	rows := [][]interface{}{{outContent.String()}}
	return []interface{}{field, types, rows}, nil
}

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ string, _ io.Writer, _ bool) (string, error) {
	return "", nil
}

// Restore restores the backup read from src.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	panic("not implemented")
}

// getMongoDBConnectionURI returns the MongoDB connection URI.
// https://www.mongodb.com/docs/manual/reference/connection-string/
func getMongoDBConnectionURI(connConfig db.ConnectionConfig) string {
	connectionURI := "mongodb://"
	if connConfig.SRV {
		connectionURI = "mongodb+srv://"
	}
	if connConfig.Username != "" {
		percentEncodingUsername := replaceCharacterWithPercentEncoding(connConfig.Username)
		percentEncodingPassword := replaceCharacterWithPercentEncoding(connConfig.Password)
		connectionURI = fmt.Sprintf("%s%s:%s@", connectionURI, percentEncodingUsername, percentEncodingPassword)
	}
	connectionURI = fmt.Sprintf("%s%s", connectionURI, connConfig.Host)
	if connConfig.Port != "" {
		connectionURI = fmt.Sprintf("%s:%s", connectionURI, connConfig.Port)
	}
	if connConfig.Database != "" {
		connectionURI = fmt.Sprintf("%s/%s", connectionURI, connConfig.Database)
	}
	// We use admin as the default authentication database.
	// https://www.mongodb.com/docs/manual/reference/connection-string/#mongodb-urioption-urioption.authSource
	authenticationDatabase := connConfig.AuthenticationDatabase
	if authenticationDatabase == "" {
		authenticationDatabase = "admin"
	}

	if connConfig.Database == "" {
		connectionURI = fmt.Sprintf("%s/", connectionURI)
	}
	connectionURI = fmt.Sprintf("%s?authSource=%s", connectionURI, authenticationDatabase)

	return connectionURI
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
