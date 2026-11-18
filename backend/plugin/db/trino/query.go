package trino

import (
	"context"
	"database/sql"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script's statements on the driver's session, which lives
// until Close so an editor request's statements share it. conn is unused: the
// caller's *sql.Conn, if it had one, is not the contract's Session.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	if d.session == nil {
		session, err := d.driver.Session(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open a session")
		}
		d.session = session
	}

	results, err := d.session.Query(ctx, statement, omnidriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  explainFormat(queryContext.Option.GetExplainFormat()),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		return nil, err
	}
	queryResults := make([]*v1pb.QueryResult, len(results))
	for i, result := range results {
		// Known difference from the contract: Bytebase showed "Affected Rows 0"
		// for a statement the engine states no count for, such as DDL and SET
		// SESSION. The contract leaves the count nil, so the adapter restores the
		// row the old driver built.
		if len(result.Columns) == 0 && result.AffectedRows == nil && result.Err == nil && !result.Truncated {
			zero := int64(0)
			result = &omnidriver.Result{Statement: result.Statement, Latency: result.Latency, AffectedRows: &zero}
		}
		queryResults[i] = util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize)
	}
	return queryResults, nil
}

// explainFormat names the plan format the API's enum asks for in the contract's
// terms. Trino states a plan in text or in JSON, and an unset format leaves the
// engine's default.
func explainFormat(format v1pb.QueryOption_ExplainFormat) string {
	switch format {
	case v1pb.QueryOption_TEXT:
		return "TEXT"
	case v1pb.QueryOption_JSON:
		return "JSON"
	default:
		return ""
	}
}
