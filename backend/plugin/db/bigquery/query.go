package bigquery

import (
	"context"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs each statement of the script as its own job (or dry run when
// Explain is set) and returns one result per statement.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(ctx, statement, util.OmniQueryOptions(queryContext, false, ""), util.SessionQueryOptions{}, util.QueryResultOptions{})
}
