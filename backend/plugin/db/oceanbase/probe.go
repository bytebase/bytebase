package oceanbase

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Prober            = (*Driver)(nil)
	_ db.RowEstimator      = (*Driver)(nil)
	_ db.ColumnExtraLookup = (*Driver)(nil)
)

// Probe runs a live review rule's probe.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return util.ProbeSQL(ctx, d.db, pCtx, statement)
}

// ColumnExtra returns one column's INFORMATION_SCHEMA.COLUMNS.EXTRA, the lookup
// OceanBase's offline-DDL rule binds.
func (d *Driver) ColumnExtra(ctx context.Context, database, table, column string) (string, error) {
	return util.QueryColumnExtra(ctx, d.db, database, table, column)
}
