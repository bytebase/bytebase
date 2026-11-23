package mssql

import (
	"context"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs a script on the held session and converts any results
// (including partial results from earlier batches when an error occurs).
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryHeldSession(ctx, statement, util.OmniQueryOptions(
		queryContext,
		false,
		util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_XML),
	), util.SessionQueryOptions{})
	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false), err
}
