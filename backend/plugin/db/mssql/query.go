package mssql

import (
	"context"
	"database/sql"

	odriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs a script for the editor, one result per statement, then the
// result sets a procedure call returned after its first.
//
// The conn the editor holds is not used: the driver keeps one session, opened on
// the first call, so the statements of one script and of the stream's later
// requests share it, as they did on the pooled connection this replaced. Calls
// are serialized on it, which is also how a single connection behaved.
//
// A statement that fails ends the script with its error on its result; an error
// returned means the script could not be run at all, with the results of the
// batches before it.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	session, err := d.querySession(ctx)
	if err != nil {
		return nil, err
	}

	results, err := session.Query(ctx, statement, odriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  explainFormat(queryContext.Option),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		// The script could not be run at all; the session is not one to keep.
		d.discardSessionLocked(session)
		return convertResults(results, queryContext), err
	}
	if ctx.Err() != nil {
		// The client closes the connection when a statement's context ends, so the
		// stream's next statement must open a new session.
		d.discardSessionLocked(session)
	}
	return convertResults(results, queryContext), nil
}

// discardSessionLocked closes a session a statement took with it, so the next
// statement opens a new one instead of reusing a dead one. The caller holds
// sessionMu.
func (d *Driver) discardSessionLocked(session odriver.Session) {
	_ = session.Close()
	if d.session == session {
		d.session = nil
	}
}

// querySession returns the session the driver's queries share, opening it on
// first use. A session is one server connection, so its state (a session
// setting, a temporary table, an open transaction) carries to the next call.
func (d *Driver) querySession(ctx context.Context) (odriver.Session, error) {
	if d.session != nil {
		return d.session, nil
	}
	session, err := d.impl.Session(ctx)
	if err != nil {
		return nil, err
	}
	d.session = session
	return session, nil
}

// convertResults maps the contract's results onto the product's.
func convertResults(results []*odriver.Result, queryContext db.QueryContext) []*v1pb.QueryResult {
	if len(results) == 0 {
		return nil
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize))
	}
	return converted
}

// explainFormat is the plan format in mssql/driver's vocabulary. SQL Server has
// no JSON showplan; which formats an engine may be asked for is decided at the
// API boundary, so anything but XML takes the tabular plan.
func explainFormat(option *v1pb.QueryOption) string {
	if option.GetExplainFormat() == v1pb.QueryOption_XML {
		return "XML"
	}
	return "TEXT"
}

// executeLogger reports the driver contract's execute steps to the task run's
// log.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l executeLogger) CommandExecute(command odriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{
		Start: int32(command.ByteStart),
		End:   int32(command.ByteEnd),
	}, command.Text)
}

func (l executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (l executeLogger) TransactionControl(op odriver.TransactionOp, err error) {
	var t storepb.TaskRunLog_TransactionControl_Type
	switch op {
	case odriver.Begin:
		t = storepb.TaskRunLog_TransactionControl_BEGIN
	case odriver.Commit:
		t = storepb.TaskRunLog_TransactionControl_COMMIT
	case odriver.Rollback:
		t = storepb.TaskRunLog_TransactionControl_ROLLBACK
	default:
		return
	}
	l.opts.LogTransactionControl(t, errorString(err))
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
