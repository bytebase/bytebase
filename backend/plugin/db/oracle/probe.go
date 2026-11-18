package oracle

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Prober = (*Driver)(nil)

// Probe runs a live review rule's probe. The dry run passes Autocommit, because
// BEGIN opens a PL/SQL block in Oracle, so the pg-style transaction cannot carry
// EXPLAIN PLAN FOR.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return util.ProbeSQL(ctx, d.db, pCtx, statement)
}
