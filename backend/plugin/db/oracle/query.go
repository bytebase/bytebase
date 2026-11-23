package oracle

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs a script for the editor on the held session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(ctx, withoutSQLPlusCommands(statement), queryOptions(queryContext), util.SessionQueryOptions{
		WrapOpenError: "failed to open a session",
	}, util.QueryResultOptions{})
}

func queryOptions(queryContext db.QueryContext) omnidriver.QueryOptions {
	return util.OmniQueryOptions(queryContext, true, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON))
}

// Probe runs a live review rule's probe on a session of its own. The Oracle
// rule's dry run passes Autocommit, because BEGIN opens a PL/SQL block, so
// EXPLAIN PLAN FOR cannot be wrapped in a transaction.
//
// A probe that is not autocommit is refused rather than run: the contract's
// Session has no transaction control, so a probe's statements could not be
// rolled back, and running them would leave what a dry run exists to avoid.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return d.ProbeOmni(ctx, pCtx, statement, util.OmniProbeOptions{
		AutocommitOnly: true,
		FormatCell:     probeValue,
	})
}

// probeValue is one plan cell in the plain types the advisor reads.
func probeValue(value omnidriver.Value) any {
	if value.Kind == omnidriver.Structured {
		return nil
	}
	return util.DefaultProbeCell(value)
}
