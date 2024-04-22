package hive

import (
	"context"
	"database/sql"
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
	connPool *FixedConnPool
	conn     *gohive.Connection
}

var (
	_          db.Driver = (*Driver)(nil)
	numMaxConn           = 5
)

func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if config.Host == "" {
		return nil, errors.Errorf("hostname not set")
	}

	d.config = config
	d.ctx = config.ConnectionContext

	if d.connPool == nil {
		if config.SASLConfig == nil || !config.SASLConfig.Check() {
			return nil, errors.New("SASL settings error")
		}
		if err := config.SASLConfig.InitEnv(); err != nil {
			return nil, errors.Wrapf(err, "failed to init SASL environment")
		}
		pool, err := CreateHiveConnPool(numMaxConn, &config)
		if err != nil {
			return nil, err
		}
		d.connPool = pool
	}

	newConn, err := d.connPool.Get(config.Database)
	if err != nil {
		return nil, errors.New("failed to get connection from pool")
	}
	d.conn = newConn

	return d, nil
}

func (d *Driver) Close(_ context.Context) error {
	d.connPool.Put(d.conn)
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	if d.conn == nil {
		return errors.Errorf("no database connection established")
	}
	cursor := d.conn.Cursor()
	defer cursor.Close()

	cursor.Exec(ctx, "SELECT 1")
	if cursor.Err != nil {
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
	if d.connPool == nil {
		return 0, errors.Errorf("no database connection established")
	}

	var affectedRows int64

	cursor := d.conn.Cursor()
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

func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statementsStr string, queryCtx *db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryWithConn(ctx, d.conn, statementsStr, queryCtx)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// this func is used for admin mode in SQL Editor.
func (d *Driver) RunStatement(ctx context.Context, _ *sql.Conn, statementsStr string) ([]*v1pb.QueryResult, error) {
	var results []*v1pb.QueryResult

	statements, err := base.SplitMultiSQL(storepb.Engine_HIVE, statementsStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split statements")
	}

	if err := SetRole(ctx, d.conn, "admin"); err != nil {
		return nil, err
	}

	for _, statement := range statements {
		statementStr := strings.TrimRight(statement.Text, ";")

		result, err := runSingleStatement(ctx, d.conn, statementStr)
		if err != nil {
			result.Error = err.Error()
			return nil, err
		}

		results = append(results, result)
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

func runSingleStatement(ctx context.Context, conn *gohive.Connection, statement string) (*v1pb.QueryResult, error) {
	var (
		result    = v1pb.QueryResult{}
		startTime = time.Now()
	)

	cursor := conn.Cursor()
	defer cursor.Close()

	// run query.
	cursor.Execute(ctx, statement, false)

	// Latency.
	result.Latency = durationpb.New(time.Since(startTime))

	// Statement.
	result.Statement = statement
	if cursor.Err != nil {
		return &result, errors.Wrapf(cursor.Err, "failed to execute statement %s", statement)
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

func (d *Driver) QueryWithConn(ctx context.Context, conn *gohive.Connection, statementsStr string, queryCtx *db.QueryContext) ([]*v1pb.QueryResult, error) {
	if d.connPool == nil {
		return nil, errors.Errorf("no database connection established")
	}

	if conn == nil {
		var err error
		conn, err = d.connPool.Get("")
		if err != nil {
			return nil, err
		}
	}

	var results []*v1pb.QueryResult

	statements, err := base.SplitMultiSQL(storepb.Engine_HIVE, statementsStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split statements")
	}

	if queryCtx != nil && !queryCtx.ReadOnly {
		if err := SetRole(ctx, conn, "admin"); err != nil {
			return nil, err
		}
	}

	for _, statement := range statements {
		statementStr := strings.TrimRight(statement.Text, ";")

		result, err := runSingleStatement(ctx, conn, statementStr)
		if err != nil {
			result.Error = err.Error()
			return nil, err
		}

		results = append(results, result)
	}
	return results, nil
}
