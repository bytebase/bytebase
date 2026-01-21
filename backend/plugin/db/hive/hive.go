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

	if d.db == nil {
		return 0, errors.New("connection not initialized")
	}

	statements, err := base.SplitMultiSQL(storepb.Engine_HIVE, statementsStr)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to split statements")
	}

	var totalAffected int64
	for _, statement := range statements {
		query := strings.TrimRight(statement.Text, ";")

		result, err := d.db.ExecContext(ctx, query)
		if err != nil {
			return totalAffected, errors.Wrapf(err, "failed to execute statement: %s", query)
		}

		// Hive may not always return accurate row counts
		affected, _ := result.RowsAffected()
		totalAffected += affected
	}

	return totalAffected, nil
}

func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryCtx db.QueryContext) ([]*v1pb.QueryResult, error) {
	if d.db == nil {
		return nil, errors.New("connection not initialized")
	}

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

		result, err := d.queryStatementWithLimit(ctx, statement, queryCtx.MaximumSQLResultSize)
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

	// database/sql returns values as []byte or various Go types
	// We need to handle both the type name and the actual Go type
	switch gohiveType {
	case "BOOLEAN_TYPE", "BOOLEAN":
		if b, ok := value.(bool); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: b}}
		}
	case "TINYINT_TYPE", "TINYINT":
		if i, ok := value.(int64); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(i)}}
		}
	case "SMALLINT_TYPE", "SMALLINT":
		if i, ok := value.(int64); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(i)}}
		}
	case "INT_TYPE", "INT":
		if i, ok := value.(int64); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(i)}}
		}
	case "BIGINT_TYPE", "BIGINT":
		if i, ok := value.(int64); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: i}}
		}
	case "DOUBLE_TYPE", "DOUBLE", "FLOAT_TYPE", "FLOAT":
		if f, ok := value.(float64); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: strconv.FormatFloat(f, 'f', 20, 64)}}
		}
	case "BINARY_TYPE", "BINARY":
		if b, ok := value.([]byte); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: b}}
		}
	}

	// Default: convert to string
	// database/sql often returns values as []byte for string types
	switch v := value.(type) {
	case []byte:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(v)}}
	case string:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}}
	case int64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: strconv.FormatInt(v, 10)}}
	case float64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: strconv.FormatFloat(v, 'f', -1, 64)}}
	case bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: strconv.FormatBool(v)}}
	default:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", v)}}
	}
}

func (d *Driver) queryStatementWithLimit(ctx context.Context, statement string, limit int64) (*v1pb.QueryResult, error) {
	startTime := time.Now()

	rows, err := d.db.QueryContext(ctx, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute query")
	}
	defer rows.Close()

	result := &v1pb.QueryResult{
		Statement: statement,
	}

	// Get column information
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get column types")
	}

	for _, col := range columnTypes {
		result.ColumnNames = append(result.ColumnNames, col.Name())
		result.ColumnTypeNames = append(result.ColumnTypeNames, col.DatabaseTypeName())
	}

	// Prepare value holders for scanning
	numColumns := len(columnTypes)
	values := make([]any, numColumns)
	valuePtrs := make([]any, numColumns)
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Process query results
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}

		queryRow := &v1pb.QueryRow{}
		for i, columnType := range result.ColumnTypeNames {
			val := parseValueType(values[i], columnType)
			queryRow.Values = append(queryRow.Values, val)
		}

		result.Rows = append(result.Rows, queryRow)

		// Check size limit
		n := len(result.Rows)
		if (n&(n-1) == 0) && limit > 0 && int64(proto.Size(result)) > limit {
			result.Error = common.FormatMaximumSQLResultSizeMessage(limit)
			break
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	result.Latency = durationpb.New(time.Since(startTime))
	result.RowsCount = int64(len(result.Rows))
	return result, nil
}
