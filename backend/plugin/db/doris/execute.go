package doris

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Doris documents READ COMMITTED as its only level, so Isolation stays empty.
var dorisExecuteConfig = util.ExecuteConfig{
	ParseTransactionDirective: true,
	PassCreateDatabase:        true,
	PassMaxRetries:            true,
}

// Execute executes a SQL statement. Doris accepts no DDL inside a transaction
// and commits each data change, so a transactional script that fails part way
// leaves the rows it already ran.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, dorisExecuteConfig)
}

func executeOptions(statement string, opts db.ExecuteOptions) (string, omnidriver.ExecuteOptions) {
	return util.BuildExecuteOptions(statement, opts, dorisExecuteConfig)
}
