package starrocks

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func toQueryResults(results []*omnidriver.Result, maxResultBytes int64) []*v1pb.QueryResult {
	return util.OmniResultsToQueryResultsWith(results, maxResultBytes, util.QueryResultOptions{
		SkipPlan:     true,
		SkipMessages: true,
		ConvertValue: toRowValue,
	})
}

func toQueryResult(result *omnidriver.Result, maxResultBytes int64) *v1pb.QueryResult {
	return util.OmniResultToQueryResultWith(result, maxResultBytes, util.QueryResultOptions{
		SkipPlan:     true,
		SkipMessages: true,
		ConvertValue: toRowValue,
	})
}

// QueryConn queries a SQL statement on the driver's session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryHeldSession(ctx, statement, queryOptions(queryContext), util.SessionQueryOptions{})
	if err != nil {
		return nil, err
	}
	return toQueryResults(results, queryContext.MaximumSQLResultSize), nil
}

// queryOptions maps the product's query context onto the driver's.
//
// MaskingEnabled is carried because this engine is a MySQL fork and MySQL's
// driver withholds the rows of a statement masking cannot trace: a stored
// procedure call answers with rows no column of the statement names. This
// driver classifies a CALL as a command, so it returns no such rows today and
// the flag changes nothing until the withholding lands here.
func queryOptions(queryContext db.QueryContext) omnidriver.QueryOptions {
	return omnidriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	}
}

func toRowValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Bytes:
		// StarRocks reports BIT and VARBIT as bytes, and a zero-byte value is a
		// NULL cell: the length is what decides, not whether the driver sent an
		// empty (non-nil) slice.
		if len(value.Bytes) == 0 {
			return util.NullRowValue
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case omnidriver.Float32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case omnidriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case omnidriver.Bool, omnidriver.Float64, omnidriver.Int64, omnidriver.String, omnidriver.Timestamp:
		return util.OmniRowValue(value)
	default:
		// StarRocks has no other kind: the driver's unsigneds, zone-aware
		// timestamps and structured documents are other engines' shapes.
		return util.NullRowValue
	}
}
