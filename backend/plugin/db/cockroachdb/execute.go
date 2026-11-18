package cockroachdb

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
	config, statement := base.ParseTransactionConfig(script)
	transactionMode := omnidriver.Transactional
	if config.Mode == transaction.ModeOff {
		transactionMode = omnidriver.AutoCommit
	}
	return statement, omnidriver.ExecuteOptions{
		Transaction:    transactionMode,
		CreateDatabase: opts.CreateDatabase,
		Log:            executeLogger{opts: opts},
	}
}

// executeLogger reports the contract's execute steps to the task run's log.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l executeLogger) CommandExecute(command omnidriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{
		Start: int32(command.ByteStart),
		End:   int32(command.ByteEnd),
	}, command.Text)
}

func (l executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (l executeLogger) TransactionControl(op omnidriver.TransactionOp, err error) {
	l.opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_Type(storepb.TaskRunLog_TransactionControl_Type_value[string(op)]), errorString(err))
}

func (l executeLogger) Retry(err error, attempt, _ int) {
	l.opts.LogRetryInfo(err, attempt)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
