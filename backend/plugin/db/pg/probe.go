package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Prober = (*Driver)(nil)

// Probe runs a live review rule's probe. When the connection was opened in
// tenant mode it reads the database owner and sets the session role to it first,
// so the probe plans the statement as the migration will run it. The mode comes
// from the driver's connection context, which a caller that opens a driver for a
// review must set.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	if d.connectionCtx.TenantMode {
		owner, err := d.GetCurrentDatabaseOwner(ctx)
		if err != nil {
			return db.ProbeResult{}, err
		}
		// Use SET SESSION ROLE to match the execution logic in backend/plugin/db/pg/pg.go
		pCtx.PreExecutions = append([]string{fmt.Sprintf("SET SESSION ROLE '%s';", owner)}, pCtx.PreExecutions...)
	}
	return util.ProbeSQL(ctx, d.db, pCtx, statement)
}
