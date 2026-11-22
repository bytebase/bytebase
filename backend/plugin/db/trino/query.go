package trino

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script's statements on the driver's session, which lives
// until Close so an editor request's statements share it.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryHeldSession(ctx, statement, omnidriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	}, util.SessionQueryOptions{
		WrapOpenError: "failed to open a session",
	})
	if err != nil {
		return nil, err
	}
	return util.OmniResultsToQueryResultsWith(results, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
		// Bytebase showed "Affected Rows 0" for statements that report no count (DDL, SET SESSION).
		FillZeroAffectedRows: true,
	}), nil
}
