package hive

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/beltran/gohive/v2"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"

	// register splitter functions init().
	_ "github.com/bytebase/bytebase/backend/plugin/parser/standard"
)

func hiveDriverFunc() db.Driver {
	return &Driver{}
}

func init() {
	db.Register(storepb.Engine_HIVE, hiveDriverFunc)
}

type Driver struct {
	config     db.ConnectionConfig
	ctx        db.ConnectionContext
	db         *sql.DB
	connString string
}

var (
	_ db.Driver = (*Driver)(nil)
)

func buildHiveDSN(config db.ConnectionConfig) (string, error) {
	port := config.DataSource.Port
	if port == "" {
		port = "10000" // default Hive port
	}

	// Basic DSN format: hive://host:port/database
	dsn := fmt.Sprintf("hive://%s:%s", config.DataSource.Host, port)

	// Add database if specified
	if config.ConnectionContext.DatabaseName != "" {
		dsn = fmt.Sprintf("%s/%s", dsn, config.ConnectionContext.DatabaseName)
	}

	// Add authentication parameters
	auth := "NONE"
	service := "hive"

	if t, ok := config.DataSource.GetSaslConfig().GetMechanism().(*storepb.SASLConfig_KrbConfig); ok {
		auth = "KERBEROS"
		if t.KrbConfig.Primary != "" {
			service = t.KrbConfig.Primary
		}
	}

	dsn = fmt.Sprintf("%s?auth=%s&service=%s", dsn, auth, service)

	return dsn, nil
}

func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if config.DataSource.Host == "" {
		return nil, errors.Errorf("hostname not set")
	}

	// Build DSN connection string
	connString, err := buildHiveDSN(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build DSN")
	}

	// Handle Kerberos authentication if needed
	if t, ok := config.DataSource.GetSaslConfig().GetMechanism().(*storepb.SASLConfig_KrbConfig); ok {
		// Kerberos environment mutex
		util.Lock.Lock()
		defer util.Lock.Unlock()

		if err := util.BootKerberosEnv(t); err != nil {
			return nil, errors.Wrapf(err, "failed to init SASL environment")
		}
	}

	// Open database connection using v2 driver
	sqlDB, err := sql.Open("hive", connString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open hive connection")
	}

	// Configure connection pool (Hive doesn't support many concurrent connections well)
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(0) // connections don't expire

	// Verify connection works
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, errors.Wrap(err, "failed to ping hive server")
	}

	d.config = config
	d.ctx = config.ConnectionContext
	d.db = sqlDB
	d.connString = connString

	return d, nil
}

func (d *Driver) Close(_ context.Context) error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	if d.db == nil {
		return errors.New("connection not initialized")
	}
	if err := d.db.PingContext(ctx); err != nil {
		return errors.Wrapf(err, "bad connection")
	}
	return nil
}

func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// Transaction statements [BEGIN, COMMIT, ROLLBACK] are not supported in Hive 4.0 temporarily.
// Even in Hive's bucketed transaction table, all the statements are committed automatically by
// the Hive server.
func (d *Driver) Execute(ctx context.Context, statementsStr string, _ db.ExecuteOptions) (int64, error) {
	// Hive has limited transaction support:
	// - Only ACID tables support transactions, and even then it's limited
	// - Transaction statements (BEGIN, COMMIT, ROLLBACK) are not supported in Hive 4.0
	// - All statements are auto-committed by the Hive server
	// Due to these limitations, we execute statements individually regardless of transaction mode
	// but we still parse and respect the transaction mode directive for consistency

	var affectedRows int64

	cursor := d.conn.Cursor()
	defer cursor.Close()

	statements, err := base.SplitMultiSQL(storepb.Engine_HIVE, statementsStr)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to split statements")
	}

	for _, statement := range statements {
		query := strings.TrimRight(statement.Text, ";")
		if err := executeCursor(ctx, cursor, query); err != nil {
			return 0, err
		}
		operationStatus := cursor.Poll(false)
		affectedRows += operationStatus.GetNumModifiedRows()
	}

	return affectedRows, nil
}

func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryCtx db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_HIVE, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split statements")
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := util.TrimStatement(singleSQL.Text)
		if queryCtx.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		}

		result, err := queryStatementWithLimit(ctx, d.conn, statement, queryCtx.MaximumSQLResultSize)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}
	return results, nil
}

// This function converts basic types to types that have implemented isRowValue_Kind interface.
func parseValueType(value any, gohiveType string) *v1pb.RowValue {
	if value == nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	}
	switch gohiveType {
	case "BOOLEAN_TYPE":
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value.(bool)}}
	case "TINYINT_TYPE":
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(value.(int8))}}
	case "SMALLINT_TYPE":
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(value.(int16))}}
	case "INT_TYPE":
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: value.(int32)}}
	case "BIGINT_TYPE":
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.(int64)}}
	case "DOUBLE_TYPE", "FLOAT_TYPE":
		// convert float64 to string to avoid truncation, because our v1pb.RowValue_FloatValue is float32.
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: strconv.FormatFloat(value.(float64), 'f', 20, 64)}}
	case "BINARY_TYPE":
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.([]byte)}}
	default:
		// convert all remaining types to string.
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.(string)}}
	}
}

func queryStatement(ctx context.Context, conn *gohive.Connection, statement string) (*v1pb.QueryResult, error) {
	return queryStatementWithLimit(ctx, conn, statement, 0)
}

func queryStatementWithLimit(ctx context.Context, conn *gohive.Connection, statement string, limit int64) (*v1pb.QueryResult, error) {
	startTime := time.Now()

	cursor := conn.Cursor()
	defer cursor.Close()

	// run query.
	if err := executeCursor(ctx, cursor, statement); err != nil {
		return nil, err
	}

	result := &v1pb.QueryResult{
		Statement: statement,
	}

	// We will get an error when a certain statement doesn't need returned results.
	for _, row := range cursor.Description() {
		if len(row) == 0 {
			return nil, errors.New("description row has zero length")
		}
		result.ColumnNames = append(result.ColumnNames, row[0])
		result.ColumnTypeNames = append(result.ColumnTypeNames, row[1])
	}

	// process query results.
	for cursor.HasMore(ctx) {
		queryRow := &v1pb.QueryRow{}
		rowMap := cursor.RowMap(ctx)
		for i, columnName := range result.ColumnNames {
			columnType := result.ColumnTypeNames[i]
			val := parseValueType(rowMap[columnName], columnType)
			queryRow.Values = append(queryRow.Values, val)
		}

		// Rows.
		result.Rows = append(result.Rows, queryRow)
		n := len(result.Rows)
		if (n&(n-1) == 0) && limit > 0 && int64(proto.Size(result)) > limit {
			result.Error = common.FormatMaximumSQLResultSizeMessage(limit)
			break
		}
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.RowsCount = int64(len(result.Rows))
	return result, nil
}

func executeCursor(ctx context.Context, cursor *gohive.Cursor, statement string) error {
	cursor.Exec(ctx, statement)
	if cursor.Err != nil {
		return errors.Wrap(cursor.Err, "failed to execute statement")
	}
	return nil
}
