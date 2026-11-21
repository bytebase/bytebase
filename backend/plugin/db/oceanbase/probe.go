package oceanbase

import (
	"context"
	"time"

	odriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return d.ProbeOmni(ctx, pCtx, statement, util.OmniProbeOptions{
		BeginStatement:           "START TRANSACTION",
		RollbackStatement:        "ROLLBACK",
		RollbackWithoutCancel:    true,
		RollbackTimeout:          5 * time.Second,
		SkipEmptyPreExecutions:   true,
		CheckPreExecutionResults: true,
		FormatCell:               probeCell,
	})
}

func probeCell(value odriver.Value) any {
	switch value.Kind {
	case odriver.Bytes:
		return string(value.Bytes)
	case odriver.Int32:
		return int32(value.Int64)
	case odriver.Uint32:
		return uint32(value.Uint64)
	case odriver.Structured:
		return nil
	case odriver.Timestamp, odriver.TimestampTZ:
		return value.Time.Format(time.RFC3339Nano)
	default:
		return util.DefaultProbeCell(value)
	}
}

// CountAffectedRows returns OceanBase's planner estimate of the rows the
// INSERT, UPDATE or DELETE statement modifies, read from EXPLAIN FORMAT=JSON.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.Driver.(db.RowEstimator).CountAffectedRows(ctx, statement)
}

// ColumnExtra returns one column's INFORMATION_SCHEMA.COLUMNS.EXTRA, the
// lookup OceanBase's offline-DDL rule binds.
func (d *Driver) ColumnExtra(ctx context.Context, database, table, column string) (string, error) {
	return d.Driver.(db.ColumnExtraLookup).ColumnExtra(ctx, database, table, column)
}
