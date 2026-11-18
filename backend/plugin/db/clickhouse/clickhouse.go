// Package clickhouse is Bytebase's ClickHouse driver: an adapter over
// github.com/bytebase/omni/clickhouse/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos.
//
// Every db.Driver method delegates:
//
//   - Open maps the data source onto driver.Config: the native endpoint, the
//     database the editor opened, the login, TLS material, the caller's extra
//     parameters as ClickHouse settings, and an SSH Dial for a tunnelled data
//     source. It does not reach the server; Ping does.
//   - Ping, Close, Execute, SyncInstance, SyncDBSchema and Dump delegate.
//   - QueryConn asks the driver for one session and holds it, so statements a
//     caller runs through one driver share a session.
//   - GetDB returns nil: the omni driver owns its pool and hands out sessions
//     instead of a *sql.DB. Every Bytebase caller already tolerates nil there,
//     and no ClickHouse feature read the handle.
//   - Execute reports each statement's range and the transaction control steps
//     to the task run's log.
//
// These are the contracts a caller can rely on, stated because a stored dump
// and the editor's results depend on them:
//
//   - Dump writes the server's own SHOW CREATE statement for every object,
//     database-qualified and ordered so a restore replays: tables and
//     dictionaries, then views, then materialized views and Dictionary-engine
//     tables after what they read, with the ".inner_id." storage tables left
//     out. The metadata it is given is not rendered and does not narrow the
//     dump, so a dump restores and round-trips byte-identically.
//   - A statement that returns no rows answers with an empty result. Execute
//     answers 0 because clickhouse-go states no affected count.
//   - Date, DateTime and DateTime64 are typed timestamps carrying the precision
//     their type names, and a zoned one keeps its zone and offset; Time and
//     Time64 are clock strings; IPv4, IPv6 and FixedString are their values.
//   - SHOW, DESCRIBE, DESC, EXISTS and CHECK run as queries, and EXPLAIN
//     ANALYZE keeps the rows the server returns beside a typed QueryPlan.
//   - The result-size cap is byte-accurate and MaximumSQLResultSize 0 is no
//     cap. A limit rewrite wraps only a statement that can stand in a CTE and
//     drops trailing comments a CTE cannot hold.
//   - SyncInstance lists databases and users in name order. A sorting key,
//     primary key or index expression is split at top-level commas only, so a
//     function or tuple keeps its own.
//   - A connection the server refuses because the database is gone is
//     driver.ErrNotFound. A sync on a session already open on a dropped
//     database answers an empty schema.
//
// Extra connection parameters reach ClickHouse as settings, and an SSH data
// source connects through Config.Dial.
package clickhouse

import (
	"context"
	"crypto/tls"
	"database/sql"
	"io"
	"log/slog"
	"net"
	"sync"

	omnich "github.com/bytebase/omni/clickhouse/driver"
	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_CLICKHOUSE, newDriver)
}

// Driver is Bytebase's ClickHouse driver.
type Driver struct {
	impl      *omnich.Driver
	sshClient *ssh.Client

	// session is the one server session QueryConn runs on, held for the
	// driver's lifetime so statements a caller runs through one driver share
	// it.
	sessionMu sync.Mutex
	session   omnidriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a ClickHouse driver. It does not reach the server: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, err := clickhouseConfig(config.DataSource, config.Password, config.ConnectionContext.DatabaseName)
	if err != nil {
		return nil, err
	}
	d := &Driver{}
	if config.DataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(config.DataSource)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient
		cfg.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			return util.DialSSHTunnel(ctx, sshClient, network, address)
		}
	}

	slog.Debug("Opening ClickHouse driver",
		slog.String("addr", config.DataSource.GetHost()+":"+config.DataSource.GetPort()),
		slog.String("environment", config.ConnectionContext.EnvironmentID),
		slog.String("database", config.ConnectionContext.InstanceID),
	)

	impl, err := omnich.Open(ctx, cfg)
	if err != nil {
		d.closeSSHClient()
		return nil, err
	}
	d.impl = impl
	return d, nil
}

// clickhouseConfig resolves everything Bytebase knows about reaching the server
// into the plain values clickhouse/driver takes.
func clickhouseConfig(dataSource *storepb.DataSource, password, database string) (omnidriver.Config, error) {
	tlsConfig, err := util.GetTLSConfig(dataSource)
	if err != nil {
		return omnidriver.Config{}, errors.Wrap(err, "sql: tls config error")
	}
	cfg := omnidriver.Config{
		Host:     dataSource.GetHost(),
		Port:     dataSource.GetPort(),
		Database: database,
		User:     dataSource.GetUsername(),
		Password: password,
		// A data source's extra parameters are ClickHouse settings, which is
		// what the contract carries them as.
		Params: dataSource.GetExtraConnectionParameters(),
	}
	if tlsConfig != nil {
		cfg.TLS = func(string) (*tls.Config, error) { return tlsConfig, nil }
	}
	return cfg, nil
}

func (d *Driver) closeSSHClient() {
	if d.sshClient != nil {
		_ = d.sshClient.Close()
		d.sshClient = nil
	}
}

// Close closes the session, the pool and the SSH tunnel, if any.
func (d *Driver) Close(context.Context) error {
	var err error
	d.sessionMu.Lock()
	session := d.session
	d.session = nil
	d.sessionMu.Unlock()
	if session != nil {
		err = multierr.Append(err, session.Close())
	}
	if d.impl != nil {
		err = multierr.Append(err, d.impl.Close())
	}
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
		d.sshClient = nil
	}
	return err
}

// Ping pings the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// GetDB returns nil: the omni driver owns its pool and hands out sessions
// instead. Every caller already branches on the handle being absent, and no
// ClickHouse feature read it.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute runs the script's statements, reporting each to the task run's log.
// ClickHouse cannot roll back, so a script still logs the BEGIN and ROLLBACK
// its transaction wrapper issues.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	config, cleaned := base.ParseTransactionConfig(statement)
	transactionMode := config.Mode
	if transactionMode == transaction.ModeUnspecified {
		transactionMode = transaction.DefaultMode()
	}

	executeOptions := omnidriver.ExecuteOptions{
		Transaction: omnidriver.Transactional,
		Log:         executeLogger{opts: opts},
	}
	if transactionMode == transaction.ModeOff {
		executeOptions.Transaction = omnidriver.AutoCommit
	}
	return d.impl.Execute(ctx, cleaned, executeOptions)
}

// executeLogger reports the driver contract's execute steps to the task run's
// log.
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
	l.opts.LogTransactionControl(t, errorString(err))
}

// Retry logs nothing: this driver never retries.
func (executeLogger) Retry(error, int, int) {}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// SyncInstance syncs the server's version, databases and users.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	var roles []*storepb.InstanceRole
	for _, role := range instance.Roles {
		roles = append(roles, &storepb.InstanceRole{Name: role.Name})
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
		Metadata:  &storepb.Instance{Roles: roles},
	}, nil
}

// SyncDBSchema syncs a single database schema. A database the server no longer
// has is the driver's ErrNotFound.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.impl.SyncDatabase(ctx)
}

// Dump writes the database's DDL as the server states it.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, dbMetadata)
}

// querySession returns the driver's one session, opening it on the first query
// so Open still does not reach the server.
func (d *Driver) querySession(ctx context.Context) (omnidriver.Session, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
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
