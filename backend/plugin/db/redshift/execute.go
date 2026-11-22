package redshift

import (
	"context"

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
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective:     true,
		SkipDirectiveOnCreateDatabase: true,
		PassCreateDatabase:            true,
		LoggerOptions:                 []util.ExecuteLoggerOption{util.WithOmitPerResultAffectedRows()},
	})
}
