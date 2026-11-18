package oceanbase

import (
	"context"
	"time"

	"github.com/pkg/errors"

	odriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

var (
	_ db.Prober            = (*Driver)(nil)
	_ db.RowEstimator      = (*Driver)(nil)
	_ db.ColumnExtraLookup = (*Driver)(nil)
)

// Probe runs a live review rule's probe: the pre-executions and the statement
// in one transaction on one session, rolled back afterwards, as the *sql.DB
// handle it replaced did. ProbeContext.Autocommit runs the statement alone,
// which is what a rule whose statement OceanBase refuses inside a transaction
// asks for.
//
// The session reads SELECT CONNECTION_ID() before the transaction, and the
// driver prefixes a query with the /*app=bytebase*/ comment the old probe did
// not carry; a plan's rows are what the driver decoded rather than the raw
// *sql.Rows.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	if pCtx.Autocommit && len(pCtx.PreExecutions) > 0 {
		return db.ProbeResult{}, errors.New("a probe with pre-executions cannot run outside a transaction")
	}

	session, err := d.impl.Session(ctx)
	if err != nil {
		return db.ProbeResult{}, err
	}
	defer session.Close()

	if pCtx.Autocommit {
		// Autocommit was asked for precisely because the engine refuses the
		// statement inside a transaction, so no START TRANSACTION here.
		return runProbeStatement(ctx, session, statement)
	}

	if _, err := runProbeStatement(ctx, session, "START TRANSACTION"); err != nil {
		return db.ProbeResult{}, errors.Wrap(err, "failed to begin probe transaction")
	}
	// Roll back whatever ran, even after the caller's context is gone: the
	// session's connection returns to the pool.
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, _ = session.Query(rollbackCtx, "ROLLBACK", odriver.QueryOptions{})
	}()

	for _, preExec := range pCtx.PreExecutions {
		if preExec == "" {
			continue
		}
		if _, err := runProbeStatement(ctx, session, preExec); err != nil {
			return db.ProbeResult{}, errors.Wrapf(err, "failed to execute pre-execution: %s", preExec)
		}
	}
	return runProbeStatement(ctx, session, statement)
}

// runProbeStatement runs one probe statement: the session's error is whether the
// script could be read, and the statement's own error is on its result.
func runProbeStatement(ctx context.Context, session odriver.Session, statement string) (db.ProbeResult, error) {
	results, err := session.Query(ctx, statement, odriver.QueryOptions{})
	if err != nil {
		return db.ProbeResult{}, err
	}
	if len(results) == 0 {
		return db.ProbeResult{}, nil
	}
	if results[0].Err != nil {
		return db.ProbeResult{}, results[0].Err
	}
	return toProbeResult(results[0]), nil
}

// toProbeResult reshapes one contract result into the probe result, which
// carries a cell as the Go value the plan reader expects: a string for text, a
// number for a number, and nil for NULL.
func toProbeResult(result *odriver.Result) db.ProbeResult {
	probeResult := db.ProbeResult{
		Columns:     make([]string, 0, len(result.Columns)),
		ColumnTypes: make([]string, 0, len(result.Columns)),
	}
	for _, column := range result.Columns {
		probeResult.Columns = append(probeResult.Columns, column.Name)
		probeResult.ColumnTypes = append(probeResult.ColumnTypes, column.TypeName)
	}
	for _, row := range result.Rows {
		cells := make([]any, 0, len(row))
		for _, value := range row {
			cells = append(cells, probeCell(value))
		}
		probeResult.Rows = append(probeResult.Rows, cells)
	}
	return probeResult
}

func probeCell(value odriver.Value) any {
	switch value.Kind {
	case odriver.Null:
		return nil
	case odriver.Bool:
		return value.Bool
	case odriver.Bytes:
		return string(value.Bytes)
	case odriver.Float32, odriver.Float64:
		return value.Float64
	case odriver.Int32:
		return int32(value.Int64)
	case odriver.Int64:
		return value.Int64
	case odriver.Uint32:
		return uint32(value.Uint64)
	case odriver.Uint64:
		return value.Uint64
	case odriver.String:
		return value.String
	case odriver.Timestamp, odriver.TimestampTZ:
		return value.Time.Format(time.RFC3339Nano)
	default:
		return nil
	}
}

// CountAffectedRows returns OceanBase's planner estimate of the rows the
// INSERT, UPDATE or DELETE statement modifies, read from EXPLAIN FORMAT=JSON.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.impl.CountAffectedRows(ctx, statement)
}

// ColumnExtra returns one column's INFORMATION_SCHEMA.COLUMNS.EXTRA, the
// lookup OceanBase's offline-DDL rule binds.
func (d *Driver) ColumnExtra(ctx context.Context, database, table, column string) (string, error) {
	return d.impl.ColumnExtra(ctx, database, table, column)
}
