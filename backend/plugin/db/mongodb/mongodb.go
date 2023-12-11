// Package mongodb is the plugin for MongoDB driver.
package mongodb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/resources/mongoutil"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var _ db.Driver = (*Driver)(nil)
var shellEscaper = regexp.MustCompile(`[^\w@%+=:,./-]`)

func init() {
	db.Register(storepb.Engine_MONGODB, newDriver)
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
func (driver *Driver) Open(ctx context.Context, _ storepb.Engine, connCfg db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
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
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_MONGODB
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
func (driver *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	simpleStatement := isMongoStatement(statement)
	startTime := time.Now()
	connectionURI := getMongoDBConnectionURI(driver.connCfg)
	// For MongoDB query, we execute the statement in mongosh with flag --eval for the following reasons:
	// 1. Query always short, so it's safe to execute in the command line.
	// 2. We cannot catch the output if we use the --file option.

	// When you see here, it's probable that your mongosh output has been truncated. Errrrrh!
	// It appears that the truncation occurs specifically within a Linux container, while everything functions as expected on MacOS.
	// This might be due to the surprising Javascript async behavior or unflushed buffer.
	// We put a sleep(0) for the eval() to wait for its completion.
	// In some cases, the sleep(0) would not work properly, try to let shell to take the responsibility of read from pipe, this is the reason we call sh -c.
	evalArg := statement
	if simpleStatement {
		limit := ""
		if queryContext != nil && queryContext.Limit > 0 {
			limit = fmt.Sprintf(".slice(0, %d)", queryContext.Limit)
		}
		evalArg = fmt.Sprintf(`a = %s; if (typeof a.toArray === 'function') {print(EJSON.stringify(a.toArray()%s)); sleep(0);} else {print(EJSON.stringify(a)); sleep(0);}`, strings.TrimRight(statement, " \t\n\r\f;"), limit)
	}
	// We will use single quotes for the evalArg, so we need to escape the single quotes in the statement.
	if shellEscaper.MatchString(evalArg) {
		evalArg = strings.ReplaceAll(evalArg, `'`, `'"'`)
	}
	evalArg = fmt.Sprintf(`'%s'`, evalArg)

	fileName := fmt.Sprintf("mongodb-query-%s-%s", driver.connCfg.ConnectionDatabase, uuid.New().String())
	defer func() {
		// While error occurred in mongosh, the temporary file may not created, so we ignore the error here.
		_ = os.Remove(fileName)
	}()
	mongoshArgs := []string{
		mongoutil.GetMongoshPath(driver.dbBinDir),
		connectionURI,
		"--quiet",
		"--eval",
		evalArg,
		">",
		fileName,
	}

	shellArgs := []string{
		"-c",
		strings.Join(mongoshArgs, " "),
	}
	shCmd := exec.CommandContext(ctx, "sh", shellArgs...)
	var errContent bytes.Buffer
	var outContent bytes.Buffer
	shCmd.Stderr = &errContent
	shCmd.Stdout = &outContent
	if err := shCmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "failed to execute statement in mongosh: \n stdout: %s\n stderr: %s", outContent.String(), errContent.String())
	}

	f, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open file: %s", fileName)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file: %s", fileName)
	}

	if simpleStatement {
		// We make best-effort attempt to parse the content and fallback to single bulk result on failure.
		result, err := getSimpleStatementResult(content)
		if err != nil {
			slog.Error("failed to get simple statement result", slog.String("content", string(content)), log.BBError(err))
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
	rows, err := convertRows(data)
	if err != nil {
		return nil, err
	}

	columns, columnIndexMap, illegal := getColumns(rows)
	result := &v1pb.QueryResult{
		ColumnNames: columns,
	}
	for range columns {
		result.ColumnTypeNames = append(result.ColumnTypeNames, "TEXT")
	}

	for _, v := range rows {
		m, ok := v.(map[string]any)
		if !ok || illegal {
			r, err := json.MarshalIndent(v, "", "	")
			if err != nil {
				return nil, err
			}
			result.Rows = append(result.Rows, &v1pb.QueryRow{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_StringValue{StringValue: string(r)}},
				},
			})
			continue
		}

		values := make([]*v1pb.RowValue, len(columns))
		for k, v := range m {
			if k == "_id" {
				id, err := convertIDString(v)
				if err != nil {
					return nil, err
				}
				values[0] = &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: id}}
				continue
			}

			r, err := json.MarshalIndent(v, "", "	")
			if err != nil {
				return nil, err
			}
			index := columnIndexMap[k]
			values[index] = &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(r)}}
		}
		for i := 0; i < len(values); i++ {
			if values[i] == nil {
				values[i] = &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{}}
			}
		}

		result.Rows = append(result.Rows, &v1pb.QueryRow{
			Values: values,
		})
	}
	return result, nil
}

func convertIDString(idObj any) (string, error) {
	objIDObj, ok := idObj.(map[string]any)
	if ok {
		idStr, ok := objIDObj["$oid"].(string)
		if ok {
			return idStr, nil
		}
	}

	r, err := json.MarshalIndent(idObj, "", "	")
	if err != nil {
		return "", err
	}
	return string(r), nil
}

func getColumns(rows []any) ([]string, map[string]int, bool) {
	columnSet := make(map[string]bool)
	for _, v := range rows {
		m, ok := v.(map[string]any)
		if !ok {
			return []string{"result"}, map[string]int{"result": 0}, true
		}
		for k := range m {
			if _, ok := columnSet[k]; ok {
				continue
			}
			columnSet[k] = true
		}
	}

	columns, columnIndexMap := getOrderedColumns(columnSet)
	return columns, columnIndexMap, false
}

func getOrderedColumns(columnSet map[string]bool) ([]string, map[string]int) {
	var columns []string
	for k := range columnSet {
		columns = append(columns, k)
	}
	sort.Slice(columns, func(i int, j int) bool {
		if columns[i] == "_id" {
			return true
		}
		if columns[j] == "_id" {
			return false
		}
		return columns[i] < columns[j]
	})

	columnIndexMap := make(map[string]int)
	for i, column := range columns {
		columnIndexMap[column] = i
	}
	return columns, columnIndexMap
}

func convertRows(data []byte) ([]any, error) {
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
	return rows, nil
}

// RunStatement runs a SQL statement in a given connection.
func (driver *Driver) RunStatement(ctx context.Context, _ *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return driver.QueryConn(ctx, nil, statement, nil)
}

func isMongoStatement(statement string) bool {
	statement = strings.TrimLeft(statement, " \n\t")
	statement = strings.ToLower(statement)
	return strings.HasPrefix(statement, "db.")
}
