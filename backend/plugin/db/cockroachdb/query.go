package cockroachdb

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script on the driver's session, one result per statement,
// and serializes its calls.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryHeldSession(ctx, statement, omnidriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	}, util.SessionQueryOptions{})
	if err != nil {
		return nil, err
	}
	return util.OmniResultsToQueryResultsWith(results, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
		ClearExplainPlan:     queryContext.Explain,
		FillZeroAffectedRows: true,
	}), nil
}
