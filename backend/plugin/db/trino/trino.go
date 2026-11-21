// Package trino is Bytebase's Trino driver: a thin adapter over
// github.com/bytebase/omni/trino/driver. It maps a db.ConnectionConfig onto the
// contract's Config and the contract's results back onto the API's types; the
// connection, the sync queries, the splitter and the dump belong to the omni
// driver.
//
// The adapter keeps Bytebase's connection lifecycle and task-run log shape. It
// does not add an SSH tunnel or connection parameters: the old driver mapped
// neither, and the contract's Config.Dial and Config.Params stay unset.
//
// QueryConn takes a *sql.Conn the caller pooled, but the omni driver
// hands out its own Session; the driver keeps that session until Close, so the
// statements of one editor request share it as they shared the caller's
// connection before.
package trino

import (
	"context"
	"crypto/tls"
	"sync"

	omnidriver "github.com/bytebase/omni/driver"
	omnitrino "github.com/bytebase/omni/trino/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func init() {
	db.Register(storepb.Engine_TRINO, newDriver)
}

// Driver is Bytebase's Trino driver.
type Driver struct {
	driver *omnitrino.Driver
	// sessionMu guards session. A driver is opened per request, and its
	// QueryConn callers are sequential, but the adapter keeps the session for
	// the driver's lifetime so it must not be shared by two calls at once.
	sessionMu sync.Mutex
	session   omnidriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the contract's Config. The catalog is
// Config.Database: the database the connection names, else the data source's,
// else Trino's own default.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return nil, errors.New("trino: a data source is required")
	}
	database := config.ConnectionContext.DatabaseName
	if database == "" {
		database = dataSource.Database
	}
	driverConfig := omnidriver.Config{
		Host:            dataSource.Host,
		Port:            dataSource.Port,
		Database:        database,
		User:            dataSource.Username,
		Password:        config.Password,
		ApplicationName: "bytebase",
	}
	if dataSource.UseSsl {
		driverConfig.TLS = func(string) (*tls.Config, error) {
			return util.GetTLSConfig(dataSource)
		}
	}
	driver, err := omnitrino.Open(ctx, driverConfig)
	if err != nil {
		return nil, err
	}
	return &Driver{driver: driver}, nil
}

func (d *Driver) Close(context.Context) error {
	d.sessionMu.Lock()
	if d.session != nil {
		_ = d.session.Close()
		d.session = nil
	}
	d.sessionMu.Unlock()
	return d.driver.Close()
}

func (d *Driver) Ping(ctx context.Context) error {
	return d.driver.Ping(ctx)
}

// Execute runs the script's statements, reporting each step to the task run log.
// Trino commits DDL on its own and supports transactions for only some
// connectors' DML, so the omni driver runs each statement on its own and the
// contract's Transaction and Isolation are left unset.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.driver.Execute(ctx, statement, omnidriver.ExecuteOptions{
		MaxRetries: opts.MaximumRetries,
		Log:        &executeLogger{script: statement, opts: opts},
	})
}

// executeLogger bridges the contract's ExecuteLogger onto the task run log.
type executeLogger struct {
	script string
	opts   db.ExecuteOptions
}

func (l *executeLogger) CommandExecute(command omnidriver.Command) {
	l.opts.LogCommandExecute(commandRange(l.script, command), command.Text)
}

func (l *executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (l *executeLogger) TransactionControl(op omnidriver.TransactionOp, err error) {
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

// commandRange returns the byte range the log highlights for command. The
// contract ends a command's range at its terminating semicolon, which it
// excludes from Text; the task run log's range covers it, as the splitter
// Bytebase used to read statements with did.
func commandRange(script string, command omnidriver.Command) *storepb.Range {
	end := command.ByteEnd
	if end < len(script) && script[end] == ';' {
		end++
	}
	return &storepb.Range{Start: int32(command.ByteStart), End: int32(end)}
}
