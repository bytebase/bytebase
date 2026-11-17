package tidb

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Prober = (*Driver)(nil)

// Probe runs a live review rule's probe. A BATCH ... DRY RUN statement passes
// Autocommit, because TiDB refuses it inside a transaction.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return util.ProbeSQL(ctx, d.db, pCtx, statement)
}
