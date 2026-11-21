package redshift

import (
	"context"

	omndriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Execute runs the script, reporting each step to the task run log. The
// transaction directive the script states decides whether its statements share
// one transaction or commit on their own; without one they share it.
//
// The directive's isolation level and ExecuteOptions.MaximumRetries are not
// passed: the driver this replaced applied neither, and redshift/driver
// documents both as unused because Redshift's transactions are SERIALIZABLE and
// its driver never retried a serialization failure.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// The log records a command's total affected rows and no per-result list,
	// matching what the driver this replaced recorded.
	logger := util.NewExecuteLogger(opts, util.WithOmitPerResultAffectedRows())
	if opts.CreateDatabase {
		return d.impl.Execute(ctx, statement, omndriver.ExecuteOptions{
			CreateDatabase: true,
			Log:            logger,
		})
	}

	transactionMode, _, cleanedStatement := util.ParseTransactionMode(statement)
	return d.impl.Execute(ctx, cleanedStatement, omndriver.ExecuteOptions{
		Transaction: transactionMode,
		Log:         logger,
	})
}
