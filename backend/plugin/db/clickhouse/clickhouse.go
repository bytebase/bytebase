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
//   - QueryConn asks the driver for one session and runs the script on it, so
//     statements a caller runs through one driver share a session, as the old
//     driver's editor handle did.
//   - GetDB returns nil. The omni driver owns its pool and hands out sessions
//     instead of a *sql.DB; every Bytebase caller already tolerates nil there.
//     The handle the editor used to open from it is gone, so a session is the
//     driver's, not the caller's.
//   - Execute reports each statement's range and the transaction control steps
//     to the task run's log.
//
// Known differences from the driver this replaced, all of them the omni
// driver's own behaviour rather than an adapter choice:
//
//   - A statement that returns no rows answers with an empty result instead of
//     the synthetic "Affected Rows: 0" row the old driver built. Execute still
//     answers 0, because clickhouse-go states no affected count on either path.
//   - Date, DateTime and DateTime64 values are typed timestamps carrying the
//     precision their type names, where the old driver wrote DateTime as an
//     RFC3339 string. Time and Time64 are clock strings and IPv4, IPv6 and
//     FixedString are their values, where the old driver answered NULL.
//   - A statement that reads rows is classified by the omni driver: SHOW,
//     DESCRIBE, DESC, EXISTS and CHECK run as queries with their rows, and an
//     EXPLAIN ANALYZE keeps the plan the server returned instead of discarding
//     it, where the old validator called both a write. A WITH whose body only
//     reads is a query too, however its CTE is spelled.
//   - The result-size cap is byte-accurate rather than proto.Size checked at
//     power-of-two row counts, and MaximumSQLResultSize of 0 means no cap where
//     the old driver read it as a zero-byte cap.
//   - A limit rewrite wraps only a statement that can stand as a subquery, and
//     drops trailing comments a CTE cannot hold; the old wrapper wrapped every
//     statement, so a SHOW or INSERT with a limit was refused by the server.
//   - SyncInstance lists databases and users in name order; the old queries
//     stated no ORDER BY, and its system-database NOT IN list was built in Go
//     map order, so the statement itself varied between processes. A sync
//     without a database name reads the connection's current database where the
//     old driver read the schema of the empty name.
//   - A sorting key, primary key or index expression is split at top-level
//     commas only, so a function or tuple keeps its own; the old driver split
//     every ", ".
//   - Dump writes the server's own SHOW CREATE statement for every object,
//     database-qualified and in an order that replays: tables and dictionaries,
//     then views, then materialized views and Dictionary-engine tables after
//     what they read, with the ".inner_id." storage tables left out. The old
//     dump rendered system.tables.create_table_query with the database prefix
//     stripped, wrote the storage tables, and put a materialized view before the
//     table it reads, so it could not be restored. This is a user-visible change
//     to a stored schema dump.
//   - Extra connection parameters now reach ClickHouse as settings, and a
//     tunnelled data source connects; the old driver ignored both.
//   - A connection the server refuses because the database is gone is
//     driver.ErrNotFound, where the old driver reported the client's
//     UNKNOWN_DATABASE error. A sync that runs on a session already open on a
//     database the server dropped still answers an empty schema, as before.
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

	// session is the one server session QueryConn runs on. The old driver ran
	// every statement on the *sql.Conn its caller opened from GetDB, so a
	// driver kept one session for its lifetime; this holds the same shape.
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
		// The old driver ignored a data source's extra parameters; the contract
		// carries them as ClickHouse settings, which is what they are.
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
// The transaction directive is read as before; ClickHouse cannot roll back, so
// the log still carries the BEGIN and ROLLBACK the old driver wrote.
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
// log, in the same shape the old driver wrote them.
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

// Retry logs nothing: this driver never retries, as before.
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
