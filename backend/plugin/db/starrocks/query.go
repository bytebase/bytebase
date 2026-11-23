package starrocks

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func toQueryResults(results []*omnidriver.Result, maxResultBytes int64) []*v1pb.QueryResult {
	return util.OmniResultsToQueryResultsWith(results, maxResultBytes, util.MySQLForkQueryResultOptions())
}

func toQueryResult(result *omnidriver.Result, maxResultBytes int64) *v1pb.QueryResult {
	return util.OmniResultToQueryResultWith(result, maxResultBytes, util.MySQLForkQueryResultOptions())
}

// QueryConn queries a SQL statement on the driver's session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryHeldSession(ctx, statement, queryOptions(queryContext), util.SessionQueryOptions{})
	if err != nil {
		return nil, err
	}
	return toQueryResults(results, queryContext.MaximumSQLResultSize), nil
}

func queryOptions(queryContext db.QueryContext) omnidriver.QueryOptions {
	return util.MySQLForkQueryOptions(queryContext)
}

func toRowValue(value omnidriver.Value) *v1pb.RowValue {
	return util.MySQLForkRowValue(value)
}
