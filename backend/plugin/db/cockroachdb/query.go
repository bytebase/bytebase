package cockroachdb

import (
	"context"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script on the driver's session, one result per statement,
// and serializes its calls.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(
		ctx,
		statement,
		util.OmniQueryOptions(queryContext, true, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON)),
		util.SessionQueryOptions{},
		util.QueryResultOptions{
			ClearExplainPlan:     queryContext.Explain,
			FillZeroAffectedRows: true,
		},
	)
}
