// Package clickhouse is the plugin for ClickHouse driver.
package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
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
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	systemDatabases = map[string]bool{
		"system":             true,
		"information_schema": true,
		"INFORMATION_SCHEMA": true,
	}

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.ClickHouse, newDriver)
}

// Driver is the ClickHouse driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        db.Type
	databaseName  string

	db *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a ClickHouse driver.
func (driver *Driver) Open(_ context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
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

	log.Debug("Opening ClickHouse driver",
		zap.String("addr", addr),
		zap.String("environment", connCtx.EnvironmentID),
		zap.String("database", connCtx.InstanceID),
	)

	driver.dbType = dbType
	driver.db = conn
	driver.databaseName = config.Database
	driver.connectionCtx = connCtx

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
func (*Driver) GetType() db.Type {
	return db.ClickHouse
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
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool, _ db.ExecuteOptions) (int64, error) {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	totalRowsAffected := int64(0)
	f := func(stmt string) error {
		sqlResult, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			log.Debug("rowsAffected returns error", zap.Error(err))
		} else {
			totalRowsAffected += rowsAffected
		}

		return nil
	}

	if err := util.ApplyMultiStatements(strings.NewReader(statement), f); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return totalRowsAffected, err
}

// RunStatement runs a SQL statement.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	var results []*v1pb.QueryResult
	if err := util.ApplyMultiStatements(strings.NewReader(statement), func(stmt string) error {
		startTime := time.Now()
		rows, err := conn.QueryContext(ctx, statement)
		if err != nil {
			// TODO(d): ClickHouse will return "driver: bad connection" if we use non-SELECT statement for Query(). We need to ignore the error.
			//nolint
			return nil
		}
		defer rows.Close()

		result, err := convertRowsToQueryResult(rows)
		if err != nil {
			result = &v1pb.QueryResult{
				Error: err.Error(),
			}
		}
		result.Latency = durationpb.New(time.Since(startTime))
		result.Statement = strings.TrimRight(statement, " \n\t;")

		results = append(results, result)
		return nil
	}); err != nil {
		return nil, err
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

func getStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", stmt, limit)
}

func (*Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	startTime := time.Now()
	statement = strings.TrimRight(statement, " \n\t;")

	stmt := statement
	if !strings.HasPrefix(stmt, "EXPLAIN") && queryContext.Limit > 0 {
		stmt = getStatementWithResultLimit(stmt, queryContext.Limit)
	}

	// Clickhouse doesn't support READ ONLY transactions (Error: sql: driver does not support read-only transactions).
	if queryContext.ReadOnly {
		queryContext.ReadOnly = false
	}

	tx, err := conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: queryContext.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, stmt)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, stmt)
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

	data, err := readRowsForClickhouse(rows, columnTypes, columnTypeNames)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &v1pb.QueryResult{
		ColumnNames:     columnNames,
		ColumnTypeNames: columnTypeNames,
		Rows:            data,
	}, nil
}

func readRowsForClickhouse(rows *sql.Rows, columnTypes []*sql.ColumnType, columnTypeNames []string) ([]*v1pb.QueryRow, error) {
	var data []*v1pb.QueryRow

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
			return nil, err
		}

		var rowData v1pb.QueryRow
		for i := range cols {
			// handle TUPLE ARRAY MAP
			if v, ok := cols[i].(*any); ok && v != nil {
				value, err := structpb.NewValue(*v)
				if err != nil {
					return nil, errors.Errorf("failed to convert value to structpb.Value: %v", err)
				}
				rowData.Values = append(rowData.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_ValueValue{ValueValue: value}})
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

		data = append(data, &rowData)
	}

	return data, nil
}
