package databricks

import (
	"context"
	"database/sql"

	odbdriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs each statement of the script in order on the driver's session
// and converts the results into the API's. The caller's *sql.Conn is unused:
// GetDB returns nil, so there is no handle to open one from.
//
// An explain request runs each statement in its FORMATTED form, which is what a
// Databricks plan is; a statement the caller wrote as EXPLAIN runs as written.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.session.Query(ctx, statement, odbdriver.QueryOptions{
		// The editor's schema selection is the schema each statement runs in;
		// the catalog is the session's.
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  queryContext.Option.GetExplainFormat().String(),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		// A Databricks result is the columns its statement traces, so the flag
		// changes nothing; it is passed for the contract's sake.
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize))
	}
	return converted, nil
}
