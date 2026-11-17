package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Prober = (*Driver)(nil)

// Probe runs a live review rule's probe. In tenant mode it reads the database
// owner and sets the session role to it first, so the probe plans the statement
// as the migration will run it.
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
