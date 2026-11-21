package bigquery

import (
	"context"

	odriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs each statement of the script as its own job, in order, and
// returns one result per statement run.
//
// An explain request is a dry run, which is what BigQuery offers in place of
// EXPLAIN: each statement's result is one STRING column named "Validation
// Result" stating whether the statement is valid and how many bytes it would
// process.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.session.Query(ctx, statement, odriver.QueryOptions{
		// BigQuery has no session state and no schema below the dataset, so
		// QueryContext.Schema has nothing to set.
		//
		// Known difference: QueryOptions carries no operator identity, so the
		// operator_email job label the previous driver attached is not set.
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaskingEnabled: queryContext.MaskingEnabled,
		// Known difference: the cap is byte-accurate and reports Truncated, and 0
		// is no cap; the previous driver measured at power-of-two row counts and
		// stopped at the first row for 0.
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	})
	if err != nil {
		return nil, err
	}
	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false), nil
}
