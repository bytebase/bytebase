package trino

import (
	"context"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script's statements on the driver's session, which lives
// until Close so an editor request's statements share it.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(
		ctx,
		statement,
		util.OmniQueryOptions(queryContext, true, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON)),
		util.SessionQueryOptions{
			WrapOpenError: "failed to open a session",
		},
		util.QueryResultOptions{
			// Bytebase showed "Affected Rows 0" for statements that report no count (DDL, SET SESSION).
			FillZeroAffectedRows: true,
		},
	)
}
