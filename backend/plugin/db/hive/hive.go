package hive

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/beltran/gohive"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type Driver struct {
	config   db.ConnectionConfig
	ctx      db.ConnectionContext
	dbClient *gohive.Connection
}

var (
	_ db.Driver = (*Driver)(nil)
)

func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// field legality check.
	if config.Username == "" {
		return nil, errors.Errorf("user not set")
	}
	if config.Host == "" {
		return nil, errors.Errorf("hostname not set")
	}
	d.config = config
	d.ctx = config.ConnectionContext

	// initialize database connection.
	configuration := gohive.NewConnectConfiguration()
	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return nil, errors.Errorf("conversion failure for 'port' [string -> int]")
	}
	// TODO(tommy): actually there are various kinds of authentication to choose among [SASL, KERBEROS, NOSASL, PLAIN SASL]
	// "NONE" refers to PLAIN SASL that doesn't need authentication.
	authMethods := "NONE"
	conn, errConn := gohive.Connect(config.Host, port, authMethods, configuration)
	if errConn != nil {
		return nil, errors.Errorf("failed to establish connection")
	}
	d.dbClient = conn
	return d, nil
}

func (d *Driver) Close(_ context.Context) error {
	err := d.dbClient.Close()
	if err != nil {
		return errors.Errorf("faild to close connection")
	}
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	if d.dbClient == nil {
		return errors.Errorf("database not connected")
	}
	if _, err := d.QueryConn(ctx, nil, "SELECT 1", &db.QueryContext{}); err != nil {
		return errors.Errorf("bad connection")
	}
	return nil
}

func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_HIVE
}

func (*Driver) GetDB() *sql.DB {
	return nil
}

// TODO(tommy): check whether the statement is readonly?
func (d *Driver) Execute(ctx context.Context, statements string, _ db.ExecuteOptions) (int64, error) {
	if d.dbClient == nil {
		return 0, errors.Errorf("database not connected")
	}
	var rowCount int64
	cursor := d.dbClient.Cursor()
	cursor.Close()
	// TODO(tommy): support multiple statements execution.
	cursor.Execute(ctx, statements, false)

	if cursor.Err != nil {
		return 0, errors.Wrapf(cursor.Err, "failed to execute statement")
	}
	operationStatus := cursor.Poll(false)
	rowCount = operationStatus.GetNumModifiedRows()
	return rowCount, nil
}

// Used for execute readonly SELECT statement.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statements string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	if d.dbClient == nil {
		return nil, errors.Errorf("database not connected")
	}

	cursor := d.dbClient.Cursor()
	defer cursor.Close()

	var results []*v1pb.QueryResult
	for _, statement := range splitStatements(statements) {
		result, err := runSingleStatement(ctx, statement, cursor)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// RunStatement will execute the statement and return the result, for both SELECT and non-SELECT statements.
func (*Driver) RunStatement(_ context.Context, _ *sql.Conn, _ string) ([]*v1pb.QueryResult, error) {
	return nil, errors.Errorf("Not implemeted")
}

// This function converts basic types to types that have implemented isRowValue_Kind interface.
func parseValueType(value any) (*v1pb.RowValue, error) {
	var rowValue v1pb.RowValue
	switch t := value.(type) {
	case nil:
		return nil, errors.Errorf("value cannot be %v", t)
	case bool:
		rowValue.Kind = &v1pb.RowValue_BoolValue{BoolValue: value.(bool)}
	case int8:
		rowValue.Kind = &v1pb.RowValue_Int32Value{Int32Value: int32(value.(int8))}
	case int16:
		rowValue.Kind = &v1pb.RowValue_Int32Value{Int32Value: int32(value.(int16))}
	case int32:
		rowValue.Kind = &v1pb.RowValue_Int32Value{Int32Value: value.(int32)}
	case int64:
		rowValue.Kind = &v1pb.RowValue_Int32Value{Int32Value: value.(int32)}
	// TODO(tommy): dangerous truncation float64 -> float32.
	case float64:
		rowValue.Kind = &v1pb.RowValue_FloatValue{FloatValue: value.(float32)}
	case string:
		rowValue.Kind = &v1pb.RowValue_StringValue{StringValue: value.(string)}
	case []byte:
		rowValue.Kind = &v1pb.RowValue_BytesValue{BytesValue: value.([]byte)}
	default:
		return nil, errors.Errorf("not supported type")
	}
	return &rowValue, nil
}

func runSingleStatement(ctx context.Context, statement string, cursor *gohive.Cursor) (*v1pb.QueryResult, error) {
	// TODO(tommy): support latency!
	cursor.Execute(ctx, statement, false)
	if cursor.Err != nil {
		return nil, errors.Wrapf(cursor.Err, "failed to execute statement")
	}

	var result v1pb.QueryResult
	// process query results.
	for cursor.HasMore(ctx) {
		for columnName, value := range cursor.RowMap(ctx) {
			// ColumnNames.
			result.ColumnNames = append(result.ColumnNames, columnName)
			// Rows.
			var queryRow v1pb.QueryRow
			val, err := parseValueType(value)
			if err != nil {
				return nil, err
			}
			queryRow.Values = append(queryRow.Values, val)
			result.Rows = append(result.Rows, &queryRow)
			result.Statement = statement
		}
	}
	return &result, nil
}

// TODO(tommy): Finite State Machine.
func splitStatements(statement string) []string {
	return []string{statement}
}
