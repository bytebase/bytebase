package pg

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Execute runs the script, reporting each step to the task run log. The
// transaction directive the script states decides whether its statements share
// one transaction or commit on their own; without one they share it.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective:     true,
		SkipDirectiveOnCreateDatabase: true,
		PassCreateDatabase:            true,
		PassMaxRetries:                true,
	})
}
