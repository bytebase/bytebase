// Package clickhouse is the plugin for ClickHouse driver.
package clickhouse

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"reflect"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/standard"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	systemDatabases = map[string]bool{
		"system":             true,
		"information_schema": true,
		"INFORMATION_SCHEMA": true,
	}
	systemDatabaseClause = func() string {
		var l []string
		for k := range systemDatabases {
			l = append(l, fmt.Sprintf("'%s'", k))
		}
		return strings.Join(l, ", ")
	}()

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_CLICKHOUSE, newDriver)
}

// Driver is the ClickHouse driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        storepb.Engine
	databaseName  string

	db *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a ClickHouse driver.
func (driver *Driver) Open(_ context.Context, dbType storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	// Set SSL configuration.
	tlsConfig, err := config.TLSConfig.GetSslConfig()
	if err != nil {
		return nil, errors.Wrap(err, "sql: tls config error")
	}
	// Default user name is "default".
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		TLS: tlsConfig,
		Settings: clickhouse.Settings{
			// Use a relative long value to avoid timeout on resource-intenstive query. Example failure:
			// failed: code: 160, message: Estimated query execution time (xxx seconds) is too long. Maximum: yyy. Estimated rows to process: zzzzzzzzz
			"max_execution_time": 300,
		},
		DialTimeout: 10 * time.Second,
	})

	slog.Debug("Opening ClickHouse driver",
		slog.String("addr", addr),
		slog.String("environment", config.ConnectionContext.EnvironmentID),
		slog.String("database", config.ConnectionContext.InstanceID),
	)

	driver.dbType = dbType
	driver.db = conn
	driver.databaseName = config.Database
	driver.connectionCtx = config.ConnectionContext

	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetType returns the database type.
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_CLICKHOUSE
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// getVersion gets the version.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SELECT VERSION()"
	var version string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	return version, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	singleSQLs, err := standard.SplitSQL(statement)
	if err != nil {
		return 0, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return 0, nil
	}

	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	totalRowsAffected := int64(0)
	for _, singleSQL := range singleSQLs {
		sqlResult, err := tx.ExecContext(ctx, singleSQL.Text)
		if err != nil {
			return 0, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
		} else {
			totalRowsAffected += rowsAffected
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return totalRowsAffected, err
}

// RunStatement runs a SQL statement.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := standard.SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		startTime := time.Now()
		result, err := func() (*v1pb.QueryResult, error) {
			rows, err := conn.QueryContext(ctx, singleSQL.Text)
			if err != nil {
				// ClickHouse will return "driver: bad connection" if we use non-SELECT statement for Query(). We need to ignore the error.
				// nolint
				return nil, nil
			}
			defer rows.Close()

			result, err := convertRowsToQueryResult(rows)
			if err != nil {
				result = &v1pb.QueryResult{
					Error: err.Error(),
				}
			}
			result.Latency = durationpb.New(time.Since(startTime))
			result.Statement = singleSQL.Text
			return result, nil
		}()
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	// TODO(rebelice): implement multi-statement query
	var results []*v1pb.QueryResult

	result, err := driver.querySingleSQL(ctx, conn, statement, queryContext)
	if err != nil {
		results = append(results, &v1pb.QueryResult{
			Error: err.Error(),
		})
	} else {
		results = append(results, result)
	}

	return results, nil
}

func getStatementWithResultLimit(statement string, limit int) string {
	statement = strings.TrimRight(statement, " \n\t;")
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", statement, limit)
}

func (*Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	startTime := time.Now()

	if queryContext != nil && queryContext.Explain {
		statement = fmt.Sprintf("EXPLAIN %s", statement)
	} else if queryContext != nil && queryContext.Limit > 0 {
		statement = getStatementWithResultLimit(statement, queryContext.Limit)
	}

	// Clickhouse doesn't support READ ONLY transactions (Error: sql: driver does not support read-only transactions).
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result, err := convertRowsToQueryResult(rows)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement

	return result, err
}

func convertRowsToQueryResult(rows *sql.Rows) (*v1pb.QueryResult, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	result := &v1pb.QueryResult{
		ColumnNames:     columnNames,
		ColumnTypeNames: columnTypeNames,
	}
	if err := readRowsForClickhouse(result, rows, columnTypes, columnTypeNames); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func readRowsForClickhouse(result *v1pb.QueryResult, rows *sql.Rows, columnTypes []*sql.ColumnType, columnTypeNames []string) error {
	for rows.Next() {
		cols := make([]any, len(columnTypes))
		for i, name := range columnTypeNames {
			// The ClickHouse driver uses *Type rather than sql.NullType to scan nullable fields
			// as described in https://github.com/ClickHouse/clickhouse-go/issues/754
			// TODO: remove this workaround once fixed.
			if strings.HasPrefix(name, "TUPLE") || strings.HasPrefix(name, "ARRAY") || strings.HasPrefix(name, "MAP") {
				// For TUPLE, ARRAY, MAP type in ClickHouse, we pass any and the driver will do the rest.
				var it any
				cols[i] = &it
			} else {
				// We use ScanType to get the correct *Type and then do type assertions
				// following https://github.com/ClickHouse/clickhouse-go/blob/main/TYPES.md
				cols[i] = reflect.New(columnTypes[i].ScanType()).Interface()
			}
		}

		if err := rows.Scan(cols...); err != nil {
			return err
		}

		var rowData v1pb.QueryRow
		for i := range cols {
			// handle TUPLE ARRAY MAP
			if v, ok := cols[i].(*any); ok && v != nil {
				value, err := json.Marshal(v)
				if err != nil {
					return errors.Wrapf(err, "failed to marshal value")
				}
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(value)}})
				continue
			}

			// not nullable
			if v, ok := cols[i].(*int); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: int64(*v)}})
				continue
			}
			if v, ok := cols[i].(*int8); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(*v)}})
				continue
			}
			if v, ok := cols[i].(*int16); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(*v)}})
				continue
			}
			if v, ok := cols[i].(*int32); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: *v}})
				continue
			}
			if v, ok := cols[i].(*int64); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: *v}})
				continue
			}
			if v, ok := cols[i].(*uint); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: uint64(*v)}})
				continue
			}
			if v, ok := cols[i].(*uint8); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(*v)}})
				continue
			}
			if v, ok := cols[i].(*uint16); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(*v)}})
				continue
			}
			if v, ok := cols[i].(*uint32); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: *v}})
				continue
			}
			if v, ok := cols[i].(*uint64); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: *v}})
				continue
			}
			if v, ok := cols[i].(*float32); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: *v}})
				continue
			}
			if v, ok := cols[i].(*float64); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: *v}})
				continue
			}
			if v, ok := cols[i].(*string); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: *v}})
				continue
			}
			if v, ok := cols[i].(*bool); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: *v}})
				continue
			}
			if v, ok := cols[i].(*time.Time); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v.Format(time.RFC3339Nano)}})
				continue
			}
			if v, ok := cols[i].(*big.Int); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v.String()}})
				continue
			}
			if v, ok := cols[i].(*decimal.Decimal); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v.String()}})
				continue
			}
			if v, ok := cols[i].(*uuid.UUID); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v.String()}})
				continue
			}
			if v, ok := cols[i].(*orb.Point); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: wkt.MarshalString(*v)}})
				continue
			}
			if v, ok := cols[i].(*orb.Polygon); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: wkt.MarshalString(*v)}})
				continue
			}
			if v, ok := cols[i].(*orb.Ring); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: wkt.MarshalString(*v)}})
				continue
			}
			if v, ok := cols[i].(*orb.MultiPolygon); ok && v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: wkt.MarshalString(*v)}})
				continue
			}

			// nullable
			if v, ok := cols[i].(**int); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: int64(**v)}})
				continue
			}
			if v, ok := cols[i].(**int8); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(**v)}})
				continue
			}
			if v, ok := cols[i].(**int16); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(**v)}})
				continue
			}
			if v, ok := cols[i].(**int32); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: **v}})
				continue
			}
			if v, ok := cols[i].(**int64); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: **v}})
				continue
			}
			if v, ok := cols[i].(**uint); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: uint64(**v)}})
				continue
			}
			if v, ok := cols[i].(**uint8); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(**v)}})
				continue
			}
			if v, ok := cols[i].(**uint16); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(**v)}})
				continue
			}
			if v, ok := cols[i].(**uint32); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: **v}})
				continue
			}
			if v, ok := cols[i].(**uint64); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: **v}})
				continue
			}
			if v, ok := cols[i].(**float32); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: **v}})
				continue
			}
			if v, ok := cols[i].(**float64); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: **v}})
				continue
			}
			if v, ok := cols[i].(**string); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: **v}})
				continue
			}
			if v, ok := cols[i].(**bool); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: **v}})
				continue
			}
			if v, ok := cols[i].(**time.Time); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: (*v).Format(time.RFC3339Nano)}})
				continue
			}
			if v, ok := cols[i].(**big.Int); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: (*v).String()}})
				continue
			}
			if v, ok := cols[i].(**decimal.Decimal); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: (*v).String()}})
				continue
			}
			if v, ok := cols[i].(**uuid.UUID); ok && *v != nil {
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: (*v).String()}})
				continue
			}
			rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}})
		}

		result.Rows = append(result.Rows, &rowData)
		n := len(result.Rows)
		if (n&(n-1) == 0) && proto.Size(result) > common.MaximumSQLResultSize {
			result.Error = common.MaximumSQLResultSizeExceeded
			return nil
		}
	}

	return nil
}
