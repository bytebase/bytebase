package redshift

import (
	"context"

	omndriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script on the driver's session, one result per statement.
// Calls are serialized, as the stream's one connection was.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryHeldSession(ctx, statement, omndriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	}, util.SessionQueryOptions{})
	if err != nil {
		return nil, err
	}
	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false), nil
}
