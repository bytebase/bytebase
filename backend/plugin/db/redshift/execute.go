package redshift

import (
	"context"

	omndriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
	if opts.CreateDatabase {
		return d.impl.Execute(ctx, statement, omndriver.ExecuteOptions{
			CreateDatabase: true,
			Log:            &executeLogger{opts: opts},
		})
	}

	config, cleanedStatement := base.ParseTransactionConfig(statement)
	mode := transaction.DefaultMode()
	if config.Mode != transaction.ModeUnspecified {
		mode = config.Mode
	}
	transactionMode := omndriver.Transactional
	if mode == transaction.ModeOff {
		transactionMode = omndriver.AutoCommit
	}
	return d.impl.Execute(ctx, cleanedStatement, omndriver.ExecuteOptions{
		Transaction: transactionMode,
		Log:         &executeLogger{opts: opts},
	})
}

// executeLogger bridges the contract's ExecuteLogger onto the task run log. The
// log records a command's total affected rows and no per-result list, which is
// what the driver this replaced recorded.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l *executeLogger) CommandExecute(command omndriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{Start: int32(command.ByteStart), End: int32(command.ByteEnd)}, command.Text)
}

func (l *executeLogger) CommandResponse(affectedRows int64, _ []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, nil, errorString(err))
}

func (l *executeLogger) TransactionControl(op omndriver.TransactionOp, err error) {
	l.opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_Type(storepb.TaskRunLog_TransactionControl_Type_value[string(op)]), errorString(err))
}

func (l *executeLogger) Retry(err error, attempt, _ int) {
	l.opts.LogRetryInfo(err, attempt)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
