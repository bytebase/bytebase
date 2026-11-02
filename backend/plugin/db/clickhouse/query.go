package clickhouse

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/standard"
)

const aggregateFunctionErrMarker = `unsupported column type "AggregateFunction(`

// QueryConn queries a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := standard.SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptyStatements(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		if queryContext.Explain {
			explained, err := base.ExplainStatement(storepb.Engine_CLICKHOUSE, statement, db.ExplainFormat(queryContext.Option.GetExplainFormat()))
			if err != nil {
				return nil, err
			}
			statement = explained
		} else if queryContext.Limit > 0 {
			statement = base.StatementWithResultLimit(storepb.Engine_CLICKHOUSE, statement, queryContext.Limit, "")
		}
		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_CLICKHOUSE, statement)
		if err != nil {
			return nil, err
		}

		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				rows, err := conn.QueryContext(ctx, statement)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				r, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, queryContext.MaximumSQLResultSize)
				if err != nil {
					return nil, err
				}
				if err := rows.Err(); err != nil {
					return nil, err
				}
				return r, nil
			}

			sqlResult, err := conn.ExecContext(ctx, statement)
			if err != nil {
				return nil, err
			}
			affectedRows, err := sqlResult.RowsAffected()
			if err != nil {
				slog.Error("rowsAffected returns error", log.BBError(err))
			}
			return util.BuildAffectedRowsResult(affectedRows, nil), nil
		}()
		stop := false
		if err != nil {
			err = translateAggregateFunctionError(err)
			queryResult = &v1pb.QueryResult{
				Error: err.Error(),
			}
			stop = true
		}
		queryResult.Statement = statement
		queryResult.Latency = durationpb.New(time.Since(startTime))
		queryResult.RowsCount = int64(len(queryResult.Rows))
		results = append(results, queryResult)
		if stop {
			break
		}
	}

	return results, nil
}

// translateAggregateFunctionError rewrites the clickhouse-go driver's opaque
// "unsupported column type \"AggregateFunction(...)\"" error into actionable
// guidance pointing the user at -Merge combinators or finalizeAggregation(),
// since the driver cannot decode AggregateFunction intermediate state directly.
func translateAggregateFunctionError(err error) error {
	if err == nil || !strings.Contains(err.Error(), aggregateFunctionErrMarker) {
		return err
	}
	return errors.Wrap(err,
		"this query selects an AggregateFunction column, which stores aggregator intermediate state and cannot be decoded directly. "+
			"Wrap each AggregateFunction column with its -Merge combinator under GROUP BY (for example, maxMerge(col)), or with "+
			"finalizeAggregation(col) for per-row finalization")
}

func makeValueByTypeName(typeName string, columnType *sql.ColumnType) any {
	if strings.HasPrefix(typeName, "TUPLE") || strings.HasPrefix(typeName, "ARRAY") || strings.HasPrefix(typeName, "MAP") || strings.HasPrefix(typeName, "JSON") {
		// For TUPLE, ARRAY, MAP, JSON type in ClickHouse, we pass any and the driver will do the rest.
		var it any
		return &it
	}
	// We use ScanType to get the correct *Type and then do type assertions
	// following https://github.com/ClickHouse/clickhouse-go/blob/main/TYPES.md
	return reflect.New(columnType.ScanType()).Interface()
}

func convertValue(typeName string, columnType *sql.ColumnType, value any) *v1pb.RowValue {
	// handle TUPLE ARRAY MAP JSON
	if v, ok := value.(*any); ok && v != nil {
		value, err := json.Marshal(v)
		if err != nil {
			slog.Error("failed to marshal value", log.BBError(err))
			return util.NullRowValue
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(value)}}
	}

	// not nullable
	if v, ok := value.(*int); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: int64(*v)}}
	}
	if v, ok := value.(*int8); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(*v)}}
	}
	if v, ok := value.(*int16); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(*v)}}
	}
	if v, ok := value.(*int32); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: *v}}
	}
	if v, ok := value.(*int64); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: *v}}
	}
	if v, ok := value.(*uint); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: uint64(*v)}}
	}
	if v, ok := value.(*uint8); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(*v)}}
	}
	if v, ok := value.(*uint16); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(*v)}}
	}
	if v, ok := value.(*uint32); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: *v}}
	}
	if v, ok := value.(*uint64); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: *v}}
	}
	if v, ok := value.(*float32); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: *v}}
	}
	if v, ok := value.(*float64); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: *v}}
	}
	if v, ok := value.(*string); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: *v}}
	}
	if v, ok := value.(*bool); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: *v}}
	}
	if v, ok := value.(*time.Time); ok && v != nil {
		return convertClickHouseTime(typeName, columnType, *v)
	}
	if v, ok := value.(*big.Int); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v.String()}}
	}
	if v, ok := value.(*decimal.Decimal); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v.String()}}
	}
	if v, ok := value.(*uuid.UUID); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v.String()}}
	}
	if v, ok := value.(*orb.Point); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: wkt.MarshalString(*v)}}
	}
	if v, ok := value.(*orb.Polygon); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: wkt.MarshalString(*v)}}
	}
	if v, ok := value.(*orb.Ring); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: wkt.MarshalString(*v)}}
	}
	if v, ok := value.(*orb.MultiPolygon); ok && v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: wkt.MarshalString(*v)}}
	}

	// nullable
	if v, ok := value.(**int); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: int64(**v)}}
	}
	if v, ok := value.(**int8); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(**v)}}
	}
	if v, ok := value.(**int16); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(**v)}}
	}
	if v, ok := value.(**int32); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: **v}}
	}
	if v, ok := value.(**int64); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: **v}}
	}
	if v, ok := value.(**uint); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: uint64(**v)}}
	}
	if v, ok := value.(**uint8); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(**v)}}
	}
	if v, ok := value.(**uint16); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(**v)}}
	}
	if v, ok := value.(**uint32); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: **v}}
	}
	if v, ok := value.(**uint64); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: **v}}
	}
	if v, ok := value.(**float32); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: **v}}
	}
	if v, ok := value.(**float64); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: **v}}
	}
	if v, ok := value.(**string); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: **v}}
	}
	if v, ok := value.(**bool); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: **v}}
	}
	if v, ok := value.(**time.Time); ok && *v != nil {
		return convertClickHouseTime(typeName, columnType, **v)
	}
	if v, ok := value.(**big.Int); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: (*v).String()}}
	}
	if v, ok := value.(**decimal.Decimal); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: (*v).String()}}
	}
	if v, ok := value.(**uuid.UUID); ok && *v != nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: (*v).String()}}
	}
	return util.NullRowValue
}

// convertClickHouseTime renders the time.Time the driver decoded a Date,
// Date32, DateTime or DateTime64 column into. Every one of them used to be
// formatted as RFC3339Nano, so a date read back as 2026-01-02T00:00:00Z and a
// DateTime as 2026-01-02T12:34:56Z rather than as the values ClickHouse prints.
//
// A DateTime keeps the wall clock and drops the zone, as MySQL, PostgreSQL and
// Oracle already do here: the driver returns the time in the column's timezone,
// so dropping it shows the value the server would print whatever zone it runs
// in. There is no TIME case because the ClickHouse release the tests run
// against has no time-of-day type: its TIME names an Int64, TIMESTAMP a DateTime.
func convertClickHouseTime(typeName string, columnType *sql.ColumnType, t time.Time) *v1pb.RowValue {
	if isClickHouseDate(typeName) {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: t.Format(time.DateOnly)}}
	}
	// DecimalSize reports a DateTime64's fractional digits as its precision and
	// leaves the scale at 0, unlike the engines that take Accuracy from scale.
	digits, _, _ := columnType.DecimalSize()
	stripped := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	return &v1pb.RowValue{
		Kind: &v1pb.RowValue_TimestampValue{
			TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(stripped),
				Accuracy:        int32(digits),
			},
		},
	}
}

// isClickHouseDate reports whether a column decoded to a time.Time holds a
// calendar date rather than a point in time. A nullable column is named
// Nullable(Date) and a zoned one DateTime('Europe/Paris'), so this reads the
// base name out of the wrapper and the arguments; comparing a prefix would not
// do, since DateTime starts with Date.
func isClickHouseDate(typeName string) bool {
	base := strings.TrimPrefix(typeName, "NULLABLE(")
	base, _, _ = strings.Cut(base, "(")
	base = strings.TrimSuffix(base, ")")
	return base == "DATE" || base == "DATE32"
}
