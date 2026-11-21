package cockroachdb

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script on the driver's session, one result per statement,
// and serializes its calls.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	session, err := d.querySession(ctx)
	if err != nil {
		return nil, err
	}
	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	})
	if err != nil {
		// The script could not be run at all; the session is not one to keep.
		_ = session.Close()
		d.session = nil
		return nil, err
	}
	if ctx.Err() != nil {
		_ = session.Close()
		d.session = nil
	}
	converted := util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, queryContext.Explain)
	for i, result := range results {
		// The editor's result shape is the product's: a statement that reports no
		// count (CREATE TABLE, SET) answers one "Affected Rows 0" row, and the
		// contract leaves the count nil for it.
		util.FillZeroAffectedRows(result, converted[i])
	}
	return converted, nil
}

// querySession returns the driver's session, opening it on first use. The caller
// holds sessionMu.
func (d *Driver) querySession(ctx context.Context) (omnidriver.Session, error) {
	if d.session != nil {
		return d.session, nil
	}
	session, err := d.inner.Session(ctx)
	if err != nil {
		return nil, err
	}
	d.session = session
	return session, nil
}
