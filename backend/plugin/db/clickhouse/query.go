package clickhouse

import (
	"context"
	"database/sql"

	omnidriver "github.com/bytebase/omni/driver"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs each statement of the script in order on the driver's session.
// The caller's *sql.Conn is unused: GetDB returns nil, so there is no handle to
// open one from, and the session the adapter holds plays that part.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	session, err := d.querySession(ctx)
	if err != nil {
		return nil, err
	}
	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  queryContext.Option.GetExplainFormat().String(),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	})
	if err != nil {
		return nil, err
	}
	return toQueryResults(results, queryContext), nil
}

// toQueryResults converts the contract's results into the API's, keeping each
// statement's error, plan and latency.
func toQueryResults(results []*omnidriver.Result, queryContext db.QueryContext) []*v1pb.QueryResult {
	if len(results) == 0 {
		return nil
	}
	queryResults := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		queryResults = append(queryResults, toQueryResult(result, queryContext))
	}
	return queryResults
}

// toQueryResult converts one result. An explain request leaves the plan to the
// API handler, which describes the request's format: the driver's plan belongs
// to a statement the caller wrote as EXPLAIN.
func toQueryResult(result *omnidriver.Result, queryContext db.QueryContext) *v1pb.QueryResult {
	queryResult := &v1pb.QueryResult{
		Statement: result.Statement,
		Latency:   durationpb.New(result.Latency),
	}
	if !queryContext.Explain {
		queryResult.QueryPlan = planOf(result.Plan)
	}
	for _, column := range result.Columns {
		queryResult.ColumnNames = append(queryResult.ColumnNames, column.Name)
		queryResult.ColumnTypeNames = append(queryResult.ColumnTypeNames, column.TypeName)
	}
	for _, row := range result.Rows {
		values := make([]*v1pb.RowValue, 0, len(row))
		for _, value := range row {
			values = append(values, toRowValue(value))
		}
		queryResult.Rows = append(queryResult.Rows, &v1pb.QueryRow{Values: values})
	}
	queryResult.RowsCount = int64(len(queryResult.Rows))
	if result.Truncated {
		queryResult.Error = util.FormatMaximumSQLResultSizeMessage(queryContext.MaximumSQLResultSize)
	} else if result.Err != nil {
		queryResult.Error = result.Err.Error()
	}
	return queryResult
}

func planOf(plan *omnidriver.Plan) *v1pb.QueryResult_QueryPlan {
	if plan == nil {
		return nil
	}
	return db.TypedExplainPlan(plan.Format, plan.Executed, true)
}

// toRowValue maps one contract value onto the product's row value: a small
// integer stays 32-bit, a float32 stays a float, a decimal and a UUID are text,
// a timestamp is the typed value its ClickHouse type names, and a timestamp
// whose type names a zone keeps that zone and its offset.
func toRowValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Null:
		return util.NullRowValue
	case omnidriver.Bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value.Bool}}
	case omnidriver.Bytes:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case omnidriver.Float32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: float32(value.Float64)}}
	case omnidriver.Float64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case omnidriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(value.Int64)}}
	case omnidriver.Int64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case omnidriver.Uint32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(value.Uint64)}}
	case omnidriver.Uint64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: value.Uint64}}
	case omnidriver.String, omnidriver.Structured:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	case omnidriver.Timestamp:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{
			TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(value.Time),
				Accuracy:        value.Precision,
			},
		}}
	case omnidriver.TimestampTZ:
		zone, offset := value.Time.Zone()
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampTzValue{
			TimestampTzValue: &v1pb.RowValue_TimestampTZ{
				GoogleTimestamp: timestamppb.New(value.Time),
				Zone:            zone,
				Offset:          int32(offset),
				Accuracy:        value.Precision,
			},
		}}
	default:
		return util.NullRowValue
	}
}
