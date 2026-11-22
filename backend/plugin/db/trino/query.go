package trino

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script's statements on the driver's session, which lives
// until Close so an editor request's statements share it.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.Query(ctx, func(ctx context.Context) (omnidriver.Session, error) {
		session, err := d.OpenSession(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open a session")
		}
		return session, nil
	}, util.SessionQueryOptions{}, func(session omnidriver.Session) ([]*omnidriver.Result, error) {
		return session.Query(ctx, statement, omnidriver.QueryOptions{
			Schema:         queryContext.Schema,
			Limit:          queryContext.Limit,
			Explain:        queryContext.Explain,
			ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON),
			MaxResultBytes: queryContext.MaximumSQLResultSize,
			MaskingEnabled: queryContext.MaskingEnabled,
		})
	})
	if err != nil {
		return nil, err
	}
	queryResults := util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false)
	for i, result := range results {
		// Known difference from the contract: Bytebase showed "Affected Rows 0"
		// for a statement the engine states no count for, such as DDL and SET
		// SESSION.
		util.FillZeroAffectedRows(result, queryResults[i])
	}
	return queryResults, nil
}
