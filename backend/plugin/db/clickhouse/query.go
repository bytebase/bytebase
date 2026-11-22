package clickhouse

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs each statement of the script in order on the driver's session.
// Calls are serialized on that one session, as they were on the editor's connection.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryHeldSession(ctx, statement, omnidriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  queryContext.Option.GetExplainFormat().String(),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	}, util.SessionQueryOptions{
		ShouldDiscardResults: func(ctx context.Context, session omnidriver.Session, results []*omnidriver.Result) bool {
			// A statement's error does not say whether the client's connection
			// survived it: a server exception leaves the connection in sync,
			// while a decode or protocol error leaves it unusable.
			return firstError(results) != nil && !sessionAnswers(ctx, session)
		},
	})
	if err != nil {
		return nil, err
	}
	return toQueryResults(results, queryContext), nil
}

// firstError returns the first statement error, or nil.
func firstError(results []*omnidriver.Result) error {
	for _, result := range results {
		if result.Err != nil {
			return result.Err
		}
	}
	return nil
}

// sessionAnswers reports whether the session can still run a trivial statement.
// It is called only after a statement failed, so the round trip is on the error
// path.
func sessionAnswers(ctx context.Context, session omnidriver.Session) bool {
	results, err := session.Query(ctx, "SELECT 1", omnidriver.QueryOptions{})
	if err != nil || len(results) != 1 {
		return false
	}
	return results[0].Err == nil
}

// toQueryResults converts the contract's results into the API's, keeping each
// statement's error, plan and latency.
func toQueryResults(results []*omnidriver.Result, queryContext db.QueryContext) []*v1pb.QueryResult {
	return util.OmniResultsToQueryResultsWith(results, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
		ClearExplainPlan: queryContext.Explain,
		TypedExplainPlan: true,
		TruncatedWins:    true,
		ConvertValue:     toRowValue,
	})
}

func toRowValue(value omnidriver.Value) *v1pb.RowValue {
	if value.Kind == omnidriver.Structured {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	}
	return util.OmniRowValue(value)
}
