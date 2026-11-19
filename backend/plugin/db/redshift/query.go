package redshift

import (
	"context"
	"database/sql"

	omndriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script on the driver's session, one result per statement.
// The caller's *sql.Conn is unused: the contract's session is the driver's, and
// it is the one a statement's settings hold on. Calls are serialized, as the
// stream's one connection was.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	session, err := d.querySession(ctx)
	if err != nil {
		return nil, err
	}

	results, err := session.Query(ctx, statement, omndriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
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

	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize))
	}
	return converted, nil
}

// querySession returns the driver's session, opening it on first use. The caller
// holds sessionMu.
func (d *Driver) querySession(ctx context.Context) (omndriver.Session, error) {
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
