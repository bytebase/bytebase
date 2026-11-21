package oracle

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs a script for the editor, one result per statement, on the
// session the driver keeps until Close so an editor request's statements share
// it.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	if d.session == nil {
		session, err := d.impl.Session(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open a session")
		}
		d.session = session
	}

	session := d.session
	results, err := session.Query(ctx, withoutSQLPlusCommands(statement), queryOptions(queryContext))
	if err != nil {
		// The script could not be run at all; the session is not one to keep.
		d.discardSessionLocked(session)
		return nil, err
	}
	if ctx.Err() != nil {
		// The client closes the connection when a statement's context ends, so the
		// next statement must open a new session.
		d.discardSessionLocked(session)
	}
	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false), nil
}

// discardSessionLocked closes a session a statement took with it, so the next
// statement opens a new one instead of reusing a dead one. The caller holds
// sessionMu.
func (d *Driver) discardSessionLocked(session omnidriver.Session) {
	_ = session.Close()
	if d.session == session {
		d.session = nil
	}
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
	if !pCtx.Autocommit || len(pCtx.PreExecutions) > 0 {
		return db.ProbeResult{}, errors.New("an Oracle probe runs alone: the contract's session cannot roll back pre-executions")
	}
	session, err := d.impl.Session(ctx)
	if err != nil {
		return db.ProbeResult{}, err
	}
	defer func() { _ = session.Close() }()

	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{})
	if err != nil {
		return db.ProbeResult{}, err
	}
	if len(results) == 0 {
		return db.ProbeResult{}, nil
	}
	result := results[0]
	if result.Err != nil {
		return db.ProbeResult{}, result.Err
	}
	probeResult := db.ProbeResult{
		Columns:     make([]string, len(result.Columns)),
		ColumnTypes: make([]string, len(result.Columns)),
		Rows:        make([][]any, 0, len(result.Rows)),
	}
	for i, column := range result.Columns {
		probeResult.Columns[i] = column.Name
		probeResult.ColumnTypes[i] = column.TypeName
	}
	for _, row := range result.Rows {
		values := make([]any, len(row))
		for i, value := range row {
			values[i] = probeValue(value)
		}
		probeResult.Rows = append(probeResult.Rows, values)
	}
	return probeResult, nil
}

// probeValue is one plan cell in the plain types the advisor reads.
func probeValue(value omnidriver.Value) any {
	switch value.Kind {
	case omnidriver.Bool:
		return value.Bool
	case omnidriver.Bytes:
		return value.Bytes
	case omnidriver.Float32, omnidriver.Float64:
		return value.Float64
	case omnidriver.Int32, omnidriver.Int64:
		return value.Int64
	case omnidriver.Uint32, omnidriver.Uint64:
		return value.Uint64
	case omnidriver.String:
		return value.String
	case omnidriver.Timestamp, omnidriver.TimestampTZ:
		return value.Time
	default:
		return nil
	}
}
