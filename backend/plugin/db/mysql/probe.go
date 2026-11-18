package mysql

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Prober       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)
)

// Probe runs a live review rule's probe. MySQL and MariaDB share this driver, so
// it serves both engine values.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return util.ProbeSQL(ctx, d.db, pCtx, statement)
}
