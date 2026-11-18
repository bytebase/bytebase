package starrocks

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

// QueryConn queries a SQL statement in a given connection. The caller's
// connection is unused: the driver's session stands in for it, opened on the
// first call and held until Close so that a session variable one statement sets
// reaches the statements after it. Calls are serialized, as they were on one
// database connection.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	session, err := d.querySession(ctx)
	if err != nil {
		return nil, err
	}

	results, err := session.Query(ctx, statement, queryOptions(queryContext))
	if err != nil {
		return nil, err
	}
	return toQueryResults(results, queryContext.MaximumSQLResultSize), nil
}

// querySession returns the session QueryConn runs on, opening it on first use.
// The caller holds mu.
func (d *Driver) querySession(ctx context.Context) (omnidriver.Session, error) {
	if d.session != nil {
		return d.session, nil
	}
	session, err := d.impl.Session(ctx)
	if err != nil {
		return nil, err
	}
	d.session = session
	return session, nil
}

// queryOptions maps the product's query context onto the driver's.
func queryOptions(queryContext db.QueryContext) omnidriver.QueryOptions {
	return omnidriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	}
}

// toQueryResults converts the driver's results into the API's, keeping each
// statement's error, text and latency. The driver's plan description is not
// carried into QueryResult.QueryPlan: an explain request already answers with
// the plan's rows, which is what the editor shows for this engine.
func toQueryResults(results []*omnidriver.Result, maxResultBytes int64) []*v1pb.QueryResult {
	if len(results) == 0 {
		return nil
	}
	queryResults := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		queryResults = append(queryResults, toQueryResult(result, maxResultBytes))
	}
	return queryResults
}

func toQueryResult(result *omnidriver.Result, maxResultBytes int64) *v1pb.QueryResult {
	latency := durationpb.New(result.Latency)
	if result.AffectedRows != nil {
		affected := util.BuildAffectedRowsResult(*result.AffectedRows, nil)
		affected.Statement = result.Statement
		affected.Latency = latency
		affected.RowsCount = int64(len(affected.Rows))
		return affected
	}

	queryResult := &v1pb.QueryResult{
		Statement: result.Statement,
		Latency:   latency,
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
		queryResult.Error = util.FormatMaximumSQLResultSizeMessage(maxResultBytes)
	}
	if result.Err != nil {
		queryResult.Error = result.Err.Error()
	}
	return queryResult
}

func toRowValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Null:
		return util.NullRowValue
	case omnidriver.Bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value.Bool}}
	case omnidriver.Bytes:
		// StarRocks reports BIT and VARBIT as bytes, and a zero-byte value is a
		// NULL cell: the length is what decides, not whether the driver sent an
		// empty (non-nil) slice.
		if len(value.Bytes) == 0 {
			return util.NullRowValue
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case omnidriver.Float32, omnidriver.Float64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case omnidriver.Int32, omnidriver.Int64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case omnidriver.String:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	case omnidriver.Timestamp:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{
			TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(value.Time),
				Accuracy:        value.Precision,
			},
		}}
	default:
		// StarRocks has no other kind: the driver's unsigneds, zone-aware
		// timestamps and structured documents are other engines' shapes.
		return util.NullRowValue
	}
}
