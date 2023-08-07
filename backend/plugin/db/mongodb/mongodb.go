// Package mongodb is the plugin for MongoDB driver.
package mongodb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/resources/mongoutil"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(db.MongoDB, newDriver)
}

// Driver is the MongoDB driver.
type Driver struct {
	dbBinDir      string
	connectionCtx db.ConnectionContext
	connCfg       db.ConnectionConfig
	client        *mongo.Client
	databaseName  string
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
	driver.databaseName = connCfg.Database
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

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute executes a statement, always returns 0 as the number of rows affected because we execute the statement by mongosh, it's hard to catch the row effected number.
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool, _ db.ExecuteOptions) (int64, error) {
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
	mongoshCmd := exec.CommandContext(ctx, mongoutil.GetMongoshPath(driver.dbBinDir), mongoshArgs...)
	var errContent bytes.Buffer
	mongoshCmd.Stderr = &errContent
	if err := mongoshCmd.Run(); err != nil {
		return 0, errors.Wrapf(err, "failed to execute statement in mongosh: %s", errContent.String())
	}
	return 0, nil
}

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", nil
}

// Restore restores the backup read from src.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	panic("not implemented")
}

// getMongoDBConnectionURI returns the MongoDB connection URI.
// https://www.mongodb.com/docs/manual/reference/connection-string/
func getMongoDBConnectionURI(connConfig db.ConnectionConfig) string {
	u := &url.URL{
		Scheme: "mongodb",
		// In RFC, there can be no tailing slash('/') in the path if the path is empty and the query is not empty.
		// For mongosh, it can handle this case correctly, but for driver, it will throw the error likes "error parsing uri: must have a / before the query ?".
		Path: "/",
	}
	if connConfig.SRV {
		u.Scheme = "mongodb+srv"
	}
	if connConfig.Username != "" {
		u.User = url.UserPassword(connConfig.Username, connConfig.Password)
	}
	u.Host = connConfig.Host
	if connConfig.Port != "" {
		u.Host = fmt.Sprintf("%s:%s", u.Host, connConfig.Port)
	}
	if connConfig.Database != "" {
		u.Path = connConfig.Database
	}
	authDatabase := "admin"
	if connConfig.AuthenticationDatabase != "" {
		authDatabase = connConfig.AuthenticationDatabase
	}

	u.RawQuery = fmt.Sprintf("authSource=%s", authDatabase)
	return u.String()
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	simpleStatement := false
	if _, err := parser.ParseMongo(statement); err == nil {
		simpleStatement = true
	}
	startTime := time.Now()
	connectionURI := getMongoDBConnectionURI(driver.connCfg)
	// For MongoDB query, we execute the statement in mongosh with flag --eval for the following reasons:
	// 1. Query always short, so it's safe to execute in the command line.
	// 2. We cannot catch the output if we use the --file option.

	evalArg := statement
	if simpleStatement {
		evalArg = fmt.Sprintf("a = %s; if (typeof a.toArray === 'function') {EJSON.stringify(a.toArray())} else {EJSON.stringify(a)}", strings.TrimRight(statement, " \t\n\r\f;"))
	}
	mongoshArgs := []string{
		connectionURI,
		"--quiet",
		"--eval",
		evalArg,
	}

	mongoshCmd := exec.CommandContext(ctx, mongoutil.GetMongoshPath(driver.dbBinDir), mongoshArgs...)
	var errContent bytes.Buffer
	var outContent bytes.Buffer
	mongoshCmd.Stderr = &errContent
	mongoshCmd.Stdout = &outContent
	if err := mongoshCmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "failed to execute statement in mongosh: %s", errContent.String())
	}

	if simpleStatement {
		// We make best-effort attempt to parse the content and fallback to single bulk result on failure.
		result, err := getSimpleStatementResult(outContent.Bytes())
		if err != nil {
			log.Error("failed to get simple statement result", zap.String("content", outContent.String()), zap.Error(err))
		} else {
			result.Latency = durationpb.New(time.Since(startTime))
			result.Statement = statement
			return []*v1pb.QueryResult{result}, nil
		}
	}

	return []*v1pb.QueryResult{{
		ColumnNames:     []string{"result"},
		ColumnTypeNames: []string{"TEXT"},
		Rows: []*v1pb.QueryRow{{
			Values: []*v1pb.RowValue{{
				Kind: &v1pb.RowValue_StringValue{StringValue: outContent.String()},
			}},
		}},
		Latency:   durationpb.New(time.Since(startTime)),
		Statement: statement,
	}}, nil
}

func getSimpleStatementResult(data []byte) (*v1pb.QueryResult, error) {
	var a any
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	var rows []any
	aa, ok := a.([]any)
	if ok {
		rows = aa
	} else {
		rows = []any{a}
	}

	result := &v1pb.QueryResult{
		ColumnNames:     []string{"_id", "result"},
		ColumnTypeNames: []string{"TEXT", "TEXT"},
	}
	for _, v := range rows {
		id := ""
		m, ok := v.(map[string]any)
		if ok {
			// Flatten "_id" object.
			idObj, ok := m["_id"]
			if ok {
				objIDObj, ok := idObj.(map[string]any)
				if ok {
					idStr, ok := objIDObj["$oid"].(string)
					if ok {
						id = idStr
					}
				}
				if id == "" {
					r, err := json.MarshalIndent(idObj, "", "	")
					if err != nil {
						return nil, err
					}
					id = string(r)
				}
			}

			// Remove "_id" from result.
			delete(m, "_id")
			v = m
		}

		r, err := json.MarshalIndent(v, "", "	")
		if err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, &v1pb.QueryRow{
			Values: []*v1pb.RowValue{
				{Kind: &v1pb.RowValue_StringValue{StringValue: id}},
				{Kind: &v1pb.RowValue_StringValue{StringValue: string(r)}},
			},
		})
	}
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
func (driver *Driver) RunStatement(ctx context.Context, _ *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return driver.QueryConn(ctx, nil, statement, nil)
}
