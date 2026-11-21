package databricks

import (
	"context"

	odbdriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs each statement of the script in order on the driver's session
// and converts the results into the API's.
//
// An explain request runs each statement in its FORMATTED form, which is what a
// Databricks plan is; a statement the caller wrote as EXPLAIN runs as written.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.session.Query(ctx, statement, odbdriver.QueryOptions{
		// Known difference: the schema reaches the statement. The previous driver
		// sent only the catalog, so the editor's schema selection had no effect.
		Schema:        queryContext.Schema,
		Limit:         queryContext.Limit,
		Explain:       queryContext.Explain,
		ExplainFormat: queryContext.Option.GetExplainFormat().String(),
		// Known difference: the cap is enforced. The previous driver returned every
		// row the warehouse answered, however large.
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		// A Databricks result is the columns its statement traces, so the flag
		// changes nothing; it is passed for the contract's sake.
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		return nil, err
	}
	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false), nil
}
