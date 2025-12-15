package databricks

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	// Databricks SQL.
	dbsql "github.com/databricks/databricks-sdk-go/service/sql"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func init() {
	db.Register(storepb.Engine_DATABRICKS, NewDatabricksDriver)
}

var _ db.Driver = (*Driver)(nil)

type Driver struct {
	curCatalog  string
	WarehouseID string
	Client      *databricks.WorkspaceClient
}

func NewDatabricksDriver() db.Driver {
	return &Driver{}
}

// Each Databricks driver is associated with a single Databricks Workspace (Workspace -> catalog -> schema -> table).
func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// Support Databricks native authentication.
	// ref: https://github.com/databricks/databricks-sdk-go?tab=readme-ov-file#databricks-native-authentication
	// only support token authentication.
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  config.DataSource.Host,
		Token: config.DataSource.GetAuthenticationPrivateKey(),
	})
	if err != nil {
		return nil, err
	}

	d.Client = client
	if config.DataSource.GetWarehouseId() == "" {
		return nil, errors.New("Warehouse ID must be set")
	}
	d.WarehouseID = config.DataSource.GetWarehouseId()
	d.curCatalog = config.ConnectionContext.DatabaseName
	return d, nil
}

func (*Driver) Close(_ context.Context) error {
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	_, err := d.Client.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{
		Path: "/",
	})
	if err != nil {
		return errors.Wrapf(err, "failed to ping instance")
	}
	return nil
}

func (*Driver) GetDB() *sql.DB {
	return nil
}

func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, _ db.QueryContext) ([]*v1pb.QueryResult, error) {
	var results []*v1pb.QueryResult
	stmts, err := base.SplitMultiSQL(storepb.Engine_DATABRICKS, statement)
	if err != nil {
		return nil, err
	}

	for _, stmt := range stmts {
		result := &v1pb.QueryResult{}
		startTime := time.Now()
		dataArr, colInfo, err := d.execStatementSync(ctx, stmt.Text)
		if err != nil {
			return nil, err
		}
		if dataArr == nil || colInfo == nil {
			break
		}

		colNames, colTypeNames := toStrColInfo(colInfo)
		result.ColumnNames = colNames
		result.ColumnTypeNames = colTypeNames

		// process rows.
		for _, rowData := range dataArr {
			queryRow := &v1pb.QueryRow{}
			// process a single row.
			for idx, rowVal := range rowData {
				v1pbRowVal, err := toV1pbRowVal(colInfo[idx].TypeName, rowVal)
				if err != nil {
					return nil, err
				}
				queryRow.Values = append(queryRow.Values, v1pbRowVal)
			}
			result.Rows = append(result.Rows, queryRow)
		}
		result.Latency = durationpb.New(time.Since(startTime))
		result.RowsCount = int64(len(result.Rows))
		results = append(results, result)
	}

	return results, nil
}

func (d *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	// Parse transaction mode from the script
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	statement = cleanedStatement
	transactionMode := config.Mode

	// Apply default when transaction mode is not specified
	if transactionMode == common.TransactionModeUnspecified {
		transactionMode = common.GetDefaultTransactionMode()
	}

	// Databricks has different transaction support based on the target:
	// - SQL Warehouses: Limited transaction support, primarily for Delta tables
	// - Unity Catalog: Better transaction support with ACID properties for Delta tables
	// Since Databricks SQL API doesn't expose direct transaction control (BEGIN/COMMIT/ROLLBACK),
	// and most DDL operations are auto-committed, we handle this at the statement execution level.

	// For transaction mode "off" or when dealing with non-transactional operations,
	// we execute statements in auto-commit mode (default behavior).
	// For transaction mode "on", we note that Databricks will handle transactionality
	// at the operation level for supported operations (e.g., Delta table operations).

	if transactionMode == common.TransactionModeOff {
		// Execute in auto-commit mode (default Databricks behavior)
		_, err := d.QueryConn(ctx, nil, statement, db.QueryContext{})
		return 0, err
	}

	// Transaction mode "on": Execute with implicit transaction support
	// Note: Databricks SQL Warehouses don't support explicit transaction control via the REST API.
	// Transactions are handled implicitly for supported operations (e.g., Delta table DML).
	// DDL operations are always auto-committed.
	_, err := d.QueryConn(ctx, nil, statement, db.QueryContext{})
	return 0, err
}

// Execute SQL statement synchronously and return row data or error.
func (d *Driver) execStatementSync(ctx context.Context, statement string) ([][]string, []dbsql.ColumnInfo, error) {
	resp, err := d.Client.StatementExecution.ExecuteAndWait(ctx, dbsql.ExecuteStatementRequest{
		Statement:   statement,
		WarehouseId: d.WarehouseID,
	})
	if err != nil {
		return nil, nil, err
	}
	if resp.Result == nil {
		return nil, nil, errors.New("no response")
	}

	if len(resp.Result.DataArray) != 0 {
		if resp.Manifest == nil || resp.Manifest.Schema == nil || len(resp.Manifest.Schema.Columns) == 0 {
			return nil, nil, errors.New("missing column info")
		}
		return resp.Result.DataArray, resp.Manifest.Schema.Columns, nil
	}
	return nil, nil, nil
}

// return a column type name array and a column name array.
func toStrColInfo(colInfo []dbsql.ColumnInfo) ([]string, []string) {
	colNames := []string{}
	colTypeNames := []string{}
	for _, col := range colInfo {
		colNames = append(colNames, col.Name)
		colTypeNames = append(colTypeNames, string(col.TypeName))
	}
	return colNames, colTypeNames
}

func toV1pbRowVal(colType dbsql.ColumnInfoTypeName, val string) (*v1pb.RowValue, error) {
	rowVal := v1pb.RowValue{}
	if val == "" && colType != dbsql.ColumnInfoTypeNameString {
		rowVal.Kind = &v1pb.RowValue_NullValue{}
		return &rowVal, nil
	}

	switch colType {
	case dbsql.ColumnInfoTypeNameBoolean:
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		rowVal.Kind = &v1pb.RowValue_BoolValue{BoolValue: boolVal}

	case dbsql.ColumnInfoTypeNameBinary:
		rowVal.Kind = &v1pb.RowValue_BytesValue{BytesValue: []byte(val)}

	case dbsql.ColumnInfoTypeNameShort:
		shortVal, err := strconv.ParseInt(val, 10, 16)
		if err != nil {
			return nil, err
		}
		rowVal.Kind = &v1pb.RowValue_Int32Value{Int32Value: int32(shortVal)}

	case dbsql.ColumnInfoTypeNameInt:
		i32Val, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return nil, err
		}
		rowVal.Kind = &v1pb.RowValue_Int32Value{Int32Value: int32(i32Val)}

	case dbsql.ColumnInfoTypeNameLong:
		i64Val, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, err
		}
		rowVal.Kind = &v1pb.RowValue_Int64Value{Int64Value: i64Val}

	case dbsql.ColumnInfoTypeNameFloat:
		floatVal, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return nil, err
		}
		rowVal.Kind = &v1pb.RowValue_FloatValue{FloatValue: float32(floatVal)}

	case dbsql.ColumnInfoTypeNameDouble:
		doubleVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, err
		}
		rowVal.Kind = &v1pb.RowValue_DoubleValue{DoubleValue: doubleVal}

	default:
		// convert all remaining types to string.
		rowVal.Kind = &v1pb.RowValue_StringValue{StringValue: val}
	}

	return &rowVal, nil
}
