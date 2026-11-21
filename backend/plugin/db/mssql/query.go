package mssql

import (
	"context"

	odriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs a script for the editor, one result per statement, then the
// result sets a procedure call returned after its first.
//
// A statement that fails ends the script with its error on its result; an error
// returned means the script could not be run at all, with the results of the
// batches before it.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	session, err := d.querySession(ctx)
	if err != nil {
		return nil, err
	}

	results, err := session.Query(ctx, statement, odriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_XML),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		// The script could not be run at all; the session is not one to keep.
		d.discardSessionLocked(session)
		return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false), err
	}
	if ctx.Err() != nil {
		// The client closes the connection when a statement's context ends, so the
		// stream's next statement must open a new session.
		d.discardSessionLocked(session)
	}
	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false), nil
}

// discardSessionLocked closes a session a statement took with it, so the next
// statement opens a new one instead of reusing a dead one. The caller holds
// sessionMu.
func (d *Driver) discardSessionLocked(session odriver.Session) {
	_ = session.Close()
	if d.session == session {
		d.session = nil
	}
}

// querySession returns the session the driver's queries share, opening it on
// first use. A session is one server connection, so its state (a session
// setting, a temporary table, an open transaction) carries to the next call.
func (d *Driver) querySession(ctx context.Context) (odriver.Session, error) {
	if d.session != nil {
		return d.session, nil
	}
	session, err := d.impl.Session(ctx)
	if err != nil {
		return nil, err
	}
	d.session = session
	return session, nil
}
