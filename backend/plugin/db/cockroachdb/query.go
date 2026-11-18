package cockroachdb

import (
	"context"
	"database/sql"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script on the driver's session, one result per statement,
// and serializes its calls. The caller's *sql.Conn is unused: the contract's
// session is the driver's, and it is the one a statement's settings hold on.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
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
		ExplainFormat:  explainFormat(queryContext.Option.GetExplainFormat()),
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
	if len(results) == 0 {
		return nil, nil
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		queryResult := util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize)
		// The editor's result shape is the product's: a statement that reports no
		// count (CREATE TABLE, SET) answers one "Affected Rows 0" row, and the
		// contract leaves the count nil for it.
		if len(result.Columns) == 0 && result.AffectedRows == nil && result.Err == nil {
			affected := util.BuildAffectedRowsResult(0, queryResult.Messages)
			queryResult.ColumnNames = affected.ColumnNames
			queryResult.ColumnTypeNames = affected.ColumnTypeNames
			queryResult.Rows = affected.Rows
			queryResult.RowsCount = 1
		}
		// An explain request asked for one plan, in the format it named; the
		// driver describes the statement it rewrote, which the caller already
		// knows.
		if queryContext.Explain {
			queryResult.QueryPlan = nil
		}
		converted = append(converted, queryResult)
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

// explainFormat is the contract's name for the plan the caller asked for.
// CockroachDB states a plan as text unless DISTSQL's JSON is asked for, so
// every format but JSON is the plain EXPLAIN the old driver issued.
func explainFormat(format v1pb.QueryOption_ExplainFormat) string {
	if format == v1pb.QueryOption_JSON {
		return "JSON"
	}
	return "TEXT"
}
