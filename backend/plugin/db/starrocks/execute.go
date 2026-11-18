package starrocks

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Execute executes a SQL statement. StarRocks accepts no DDL inside a
// transaction and commits each data change, so a transactional script that
// fails part way leaves the rows it already ran.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	statement = cleanedStatement
	transactionMode := config.Mode
	if transactionMode == transaction.ModeUnspecified {
		transactionMode = transaction.DefaultMode()
	}

	return d.impl.Execute(ctx, statement, executeOptions(opts, transactionMode))
}

// executeOptions maps the product's options onto the driver's.
func executeOptions(opts db.ExecuteOptions, mode transaction.Mode) omnidriver.ExecuteOptions {
	transactionMode := omnidriver.Transactional
	if mode == transaction.ModeOff {
		transactionMode = omnidriver.AutoCommit
	}
	return omnidriver.ExecuteOptions{
		Transaction:    transactionMode,
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log:            executeLogger{opts: opts},
		// Isolation stays empty: a StarRocks transaction takes no per-script
		// isolation level, so every script runs at the engine's default.
	}
}

// executeLogger reports a script's steps into the task run log: one entry per
// command with its byte range, and the transaction control.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l executeLogger) CommandExecute(cmd omnidriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{Start: int32(cmd.ByteStart), End: int32(cmd.ByteEnd)}, cmd.Text)
}

func (l executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errString(err))
}

func (l executeLogger) TransactionControl(op omnidriver.TransactionOp, err error) {
	var t storepb.TaskRunLog_TransactionControl_Type
	switch op {
	case omnidriver.Begin:
		t = storepb.TaskRunLog_TransactionControl_BEGIN
	case omnidriver.Commit:
		t = storepb.TaskRunLog_TransactionControl_COMMIT
	case omnidriver.Rollback:
		t = storepb.TaskRunLog_TransactionControl_ROLLBACK
	default:
		return
	}
	l.opts.LogTransactionControl(t, errString(err))
}

// Retry is unreachable: the driver does not retry a script.
func (executeLogger) Retry(error, int, int) {}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
