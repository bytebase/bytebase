package doris

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Execute executes a SQL statement. Doris accepts no DDL inside a transaction
// and commits each data change, so a transactional script that fails part way
// leaves the rows it already ran.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassCreateDatabase:        true,
		PassMaxRetries:            true,
	})
}

// executeOptions maps the product's options onto the driver's.
func executeOptions(statement string, opts db.ExecuteOptions) (string, omnidriver.ExecuteOptions) {
	transactionMode, _, cleanedStatement := util.ParseTransactionMode(statement)
	return cleanedStatement, omnidriver.ExecuteOptions{
		Transaction:    transactionMode,
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log:            util.NewExecuteLogger(opts),
		// Isolation stays empty: this driver has never applied a script's
		// "-- txn-isolation" level. Doris documents READ COMMITTED as its only
		// level, so passing the request through would run a statement whose
		// effect is not the level asked for; honouring it is its own change.
	}
}
