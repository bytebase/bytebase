package redshift

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Execute runs the script and reports each step to the task run log.
// Isolation level and MaximumRetries are unused because Redshift runs in
// SERIALIZABLE mode without driver-level retries.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective:     true,
		SkipDirectiveOnCreateDatabase: true,
		PassCreateDatabase:            true,
		LoggerOptions:                 []util.ExecuteLoggerOption{util.WithOmitPerResultAffectedRows()},
	})
}
