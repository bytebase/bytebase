package cockroachdb

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Execute runs the script's statements under the transaction directive the
// script starts with, reporting each step to the task run's log. The directive
// is a product convention, so the adapter reads it and hands the contract the
// mode it names. An unspecified mode is transactional, as it was before.
//
// The isolation directive and MaximumRetries are not passed: the old driver
// ignored both, and honoring them here would change what a script does.
func (d *Driver) Execute(ctx context.Context, script string, opts db.ExecuteOptions) (int64, error) {
	statement, driverOptions := executeOptions(script, opts)
	return d.inner.Execute(ctx, statement, driverOptions)
}

// executeOptions strips the transaction directive from the script and maps what
// is left onto the contract's execute options.
func executeOptions(script string, opts db.ExecuteOptions) (string, omnidriver.ExecuteOptions) {
	transactionMode, _, statement := util.ParseTransactionMode(script)
	return statement, omnidriver.ExecuteOptions{
		Transaction:    transactionMode,
		CreateDatabase: opts.CreateDatabase,
		Log:            util.NewExecuteLogger(opts),
	}
}
