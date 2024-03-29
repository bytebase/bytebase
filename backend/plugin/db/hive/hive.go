package hive

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/beltran/gohive"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// register splitter functions init().
	_ "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func hiveDriverFunc(db.DriverConfig) db.Driver {
	return &Driver{}
}

func init() {
	db.Register(storepb.Engine_HIVE, hiveDriverFunc)
}

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
	configuration.Database = config.Database
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
		return errors.Errorf("no database connection established")
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

// Transaction statements [BEGIN, COMMIT, ROLLBACK] are not supported in Hive 4.0 temporarily.
// Even in Hive's bucketed transaction table, all the statements are committed automatically by
// the Hive server.
func (d *Driver) Execute(ctx context.Context, statementsStr string, _ db.ExecuteOptions) (int64, error) {
	if d.dbClient == nil {
		return 0, errors.Errorf("no database connection established")
	}

	var (
		affectedRows int64
		cursor       = d.dbClient.Cursor()
	)
	defer cursor.Close()

	statements, err := base.SplitMultiSQL(storepb.Engine_HIVE, statementsStr)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to split statements")
	}

	for _, statement := range statements {
		cursor.Execute(ctx, strings.TrimRight(statement.Text, ";"), false)
		if cursor.Err != nil {
			return 0, errors.Wrapf(cursor.Err, "failed to execute statement %s", statement.Text)
		}
		operationStatus := cursor.Poll(false)
		affectedRows += operationStatus.GetNumModifiedRows()
	}

	return affectedRows, nil
}

// TODO(tommy): run query asynchronously.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statementsStr string, queryCtx *db.QueryContext) ([]*v1pb.QueryResult, error) {
	if d.dbClient == nil {
		return nil, errors.Errorf("no database connection established")
	}

	var results []*v1pb.QueryResult

	statements, err := base.SplitMultiSQL(storepb.Engine_HIVE, statementsStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split statements")
	}

	for _, statement := range statements {
		statementStr := strings.TrimRight(statement.Text, ";")
		if queryCtx != nil && queryCtx.Limit > 0 {
			statementStr = fmt.Sprintf("%s LIMIT %d", statementStr, queryCtx.Limit)
		}

		result, err := d.runSingleQuery(ctx, statementStr)
		if err != nil {
			result.Error = err.Error()
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// TODO(tommy): this func is used for admin mode.
func (d *Driver) RunStatement(ctx context.Context, _ *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	if err := d.SetRole(ctx, "admin"); err != nil {
		return nil, errors.Wrapf(err, "failed to switch role to admin")
	}
	results, err := d.QueryConn(ctx, nil, statement, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "")
	}
	return results, nil
}

// This function converts basic types to types that have implemented isRowValue_Kind interface.
func parseValueType(value any, gohiveType string) (*v1pb.RowValue, error) {
	var rowValue v1pb.RowValue
	switch gohiveType {
	case "BOOLEAN_TYPE":
		rowValue.Kind = &v1pb.RowValue_BoolValue{BoolValue: value.(bool)}
	case "TINYINT_TYPE":
		rowValue.Kind = &v1pb.RowValue_Int32Value{Int32Value: int32(value.(int8))}
	case "SMALLINT_TYPE":
		rowValue.Kind = &v1pb.RowValue_Int32Value{Int32Value: int32(value.(int16))}
	case "INT_TYPE":
		rowValue.Kind = &v1pb.RowValue_Int32Value{Int32Value: value.(int32)}
	case "BIGINT_TYPE":
		rowValue.Kind = &v1pb.RowValue_Int64Value{Int64Value: value.(int64)}
	// dangerous truncation: float64 -> float32.
	case "FLOAT_TYPE":
		rowValue.Kind = &v1pb.RowValue_FloatValue{FloatValue: float32(value.(float64))}
	case "BINARY_TYPE":
		rowValue.Kind = &v1pb.RowValue_BytesValue{BytesValue: value.([]byte)}
	default:
		if value == nil {
			rowValue.Kind = &v1pb.RowValue_StringValue{StringValue: ""}
		} else if gohiveType == "DOUBLE_TYPE" {
			// convert float64 to string to avoid trancation.
			rowValue.Kind = &v1pb.RowValue_StringValue{StringValue: strconv.FormatFloat(value.(float64), 'f', 20, 64)}
		} else {
			// convert all remaining types to string.
			rowValue.Kind = &v1pb.RowValue_StringValue{StringValue: value.(string)}
		}
	}
	return &rowValue, nil
}

func (d *Driver) runSingleQuery(ctx context.Context, statement string) (*v1pb.QueryResult, error) {
	var (
		result    = v1pb.QueryResult{}
		startTime = time.Now()
	)
	cursor := d.dbClient.Cursor()
	defer cursor.Close()

	// run query.
	cursor.Execute(ctx, statement, false)

	// Latency.
	result.Latency = durationpb.New(time.Since(startTime))

	// Statement.
	result.Statement = statement
	if cursor.Err != nil {
		return &result, errors.Wrapf(cursor.Err, "failed to execute statement")
	}

	columnNamesAndTypes := cursor.Description()
	if cursor.Err == nil {
		for _, row := range columnNamesAndTypes {
			result.ColumnNames = append(result.ColumnNames, row[0])
		}

		// process query results.
		for cursor.HasMore(ctx) {
			var queryRow v1pb.QueryRow
			rowMap := cursor.RowMap(ctx)
			for idx, columnName := range result.ColumnNames {
				gohiveTypeStr := columnNamesAndTypes[idx][1]
				val, err := parseValueType(rowMap[columnName], gohiveTypeStr)
				if err != nil {
					return &result, err
				}
				queryRow.Values = append(queryRow.Values, val)
			}

			// Rows.
			result.Rows = append(result.Rows, &queryRow)
		}
		return &result, nil
	}
	return nil, nil
}
