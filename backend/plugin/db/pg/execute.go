package pg

import (
	"context"

	omndriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Execute runs the script, reporting each step to the task run log. The
// transaction directive the script states decides whether its statements share
// one transaction or commit on their own; without one they share it.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	logger := util.NewExecuteLogger(opts)
	if opts.CreateDatabase {
		return d.impl.Execute(ctx, statement, omndriver.ExecuteOptions{
			CreateDatabase: true,
			MaxRetries:     opts.MaximumRetries,
			Log:            logger,
		})
	}

	transactionMode, _, cleanedStatement := util.ParseTransactionMode(statement)
	return d.impl.Execute(ctx, cleanedStatement, omndriver.ExecuteOptions{
		Transaction: transactionMode,
		MaxRetries:  opts.MaximumRetries,
		Log:         logger,
	})
}
