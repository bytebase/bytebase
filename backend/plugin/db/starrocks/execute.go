package starrocks

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Execute executes a SQL statement. StarRocks accepts no DDL inside a
// transaction and commits each data change, so a transactional script that
// fails part way leaves the rows it already ran.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	cleanedStatement, options := executeOptions(statement, opts)
	return d.impl.Execute(ctx, cleanedStatement, options)
}

// executeOptions maps the product's options onto the driver's.
func executeOptions(statement string, opts db.ExecuteOptions) (string, omnidriver.ExecuteOptions) {
	transactionMode, _, cleanedStatement := util.ParseTransactionMode(statement)
	return cleanedStatement, omnidriver.ExecuteOptions{
		Transaction:    transactionMode,
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log:            util.NewExecuteLogger(opts),
		// Isolation stays empty: a StarRocks transaction takes no per-script
		// isolation level, so every script runs at the engine's default.
	}
}
