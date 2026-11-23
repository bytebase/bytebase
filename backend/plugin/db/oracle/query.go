package oracle

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs a script for the editor, one result per statement, on the
// session the driver keeps until Close so an editor request's statements share
// it.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryHeldSession(ctx, withoutSQLPlusCommands(statement), queryOptions(queryContext), util.SessionQueryOptions{
		WrapOpenError: "failed to open a session",
	})
	if err != nil {
		return nil, err
	}
	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false), nil
}

// queryOptions maps the product's query context onto the contract's, masking
// included: a statement whose rows the analyzed span cannot trace must not
// return them.
func queryOptions(queryContext db.QueryContext) omnidriver.QueryOptions {
	return omnidriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	}
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
