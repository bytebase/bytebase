package cockroachdb

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var cockroachExecuteConfig = util.ExecuteConfig{
	ParseTransactionDirective: true,
	PassCreateDatabase:        true,
}

// Execute runs the script's statements under the transaction directive the
// script starts with, reporting each step to the task run's log. An unspecified
// mode is transactional, and isolation directives and MaximumRetries are ignored.
func (d *Driver) Execute(ctx context.Context, script string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, script, opts, cockroachExecuteConfig)
}
