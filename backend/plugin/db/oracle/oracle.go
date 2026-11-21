// Package oracle is Bytebase's Oracle driver: an adapter over
// github.com/bytebase/omni/oracle/driver that maps Bytebase's connection config
// onto the driver contract and the contract's results back into the product's.
// The conversation with Oracle — session setup, sync, the metadata-rendered
// dump, queries, change scripts, plan estimates and the database-link probe — is
// the driver's.
//
// The adapter keeps what is product: the data source to contract conversion
// (TLS material, the SSH tunnel, SID and service name, extra connection
// parameters), the QueryContext/ExecuteOptions mapping, the transaction-mode
// directive, GetDB-shaped callers, the v1pb conversion and the task-run logger.
//
// Known differences from the driver this replaced; each is either pinned by a
// test in cutover_parity_test.go or named here as the cutover proof measured it:
//   - The private-link guard takes no connection: OwnsPrivateDatabaseLink asks
//     the driver, which opens its own session. Every linked read used to be
//     refused once GetDB returned nil, so this is the fix the cutover owed.
//   - A SELECT that declares PL/SQL returns its rows. The old driver asked
//     ValidateSQLForEditor and ran anything it did not call a query through Exec,
//     dropping the rows; the contract decides by the statement's shape.
//   - QueryContext.Schema switches the session, so the editor's schema choice
//     reaches Oracle; the old driver ignored it. The driver switches back when
//     the session closes.
//   - The result-size cut is byte-accurate, where the old driver checked
//     proto.Size at power-of-two row counts. The message is the same.
//   - EXPLAIN PLAN statement IDs are a per-driver counter, not a timestamp with
//     a random suffix. The plan's statements only differ in that ID.
//   - A dump is the old text except for two additions: the sequence's full
//     definition (the old sync read only its name) and COMMENT statements (the
//     old renderer wrote none). A synced key column renders quoted once, as
//     before; the doubled "PRIMARY KEY (""ID"")" omni#1388 records is
//     backend/plugin/schema/oracle quoting a column that already arrives quoted,
//     on the migration path rather than in a dump.
//   - A schema the server no longer has answers driver.ErrNotFound, so the sync
//     records a failure instead of overwriting the schema with nothing.
//   - An SSH data source dials through Config.Dial; the old driver dialled the
//     direct host and ignored the jump host.
//   - TIMESTAMP WITH LOCAL TIME ZONE is rendered in the session's zone, which is
//     what SQL*Plus prints, where the old driver relabelled whichever zone
//     go-ora reported as UTC. The two agree while the session zone is the
//     database's, so the corpus renders the same value on both sides.
//   - Open defaults an empty port to 1521 and refuses an empty host or user up
//     front, where the old driver failed inside URL parsing or at first use.
//
// The transaction-mode directive stays here and its isolation level is not
// applied, as before: Oracle's BEGIN opens a PL/SQL block, so the isolation
// directive was never read.
//
// A SQL*Plus command is not SQL and Oracle does not execute one over a session,
// and the driver sends every segment it splits: the adapter blanks the command
// out of the script first, which keeps the script running as it did. Blanks
// rather than deletion keep every other statement's byte offsets, so the
// statement after a command starts with the blanked command as leading
// whitespace — the one difference this leaves in a result's or a log's statement
// text.
package oracle

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"sync"

	omnidriver "github.com/bytebase/omni/driver"
	oracledriver "github.com/bytebase/omni/oracle/driver"
	oracleparser "github.com/bytebase/omni/oracle/parser"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ db.Driver       = (*Driver)(nil)
	_ db.Prober       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_ORACLE, newDriver)
}

// Driver is Bytebase's Oracle driver.
type Driver struct {
	impl      *oracledriver.Driver
	sshClient *ssh.Client
	// sessionMu guards session, which QueryConn keeps until Close so an editor
	// request's statements share one session, as the caller's *sql.Conn did.
	sessionMu sync.Mutex
	session   omnidriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It reaches the
// server when the config names a schema, because the contract's Open does: a
// schema the account cannot switch to fails here rather than at the first
// statement, as it did before.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	dataSource := config.DataSource
	cfg := omnidriver.Config{
		Host:          dataSource.GetHost(),
		Port:          dataSource.GetPort(),
		Database:      config.ConnectionContext.DatabaseName,
		User:          dataSource.GetUsername(),
		Password:      config.Password,
		Params:        connectionParams(dataSource),
		EngineVersion: config.ConnectionContext.EngineVersion,
	}
	// The TLS callback keeps certificate verification bound to the host go-ora
	// dials, which is why it clears util.GetTLSConfig's own verifier: go-ora sets
	// ServerName per server, and a SERVER address or a listener redirect is
	// checked against its own name.
	if dataSource.GetUseSsl() {
		cfg.TLS = func(string) (*tls.Config, error) {
			return getTLSConfig(dataSource)
		}
	}
	if dataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(dataSource)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient
		cfg.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			return util.DialSSHTunnel(ctx, sshClient, network, address)
		}
	}

	impl, err := oracledriver.Open(ctx, cfg)
	if err != nil {
		d.closeSSHClient()
		return nil, err
	}
	d.impl = impl
	return d, nil
}

// connectionParams is the driver contract's Params for a data source: the
// data source's extra parameters in their own spelling, then the endpoint's
// service name field and SID field unless an extra parameter names the same
// option.
//
// go-ora upper-cases option keys, so two spellings of one option would leave the
// winner to map iteration order; the fields yield to the caller's parameters in
// any spelling, which is what the old driver's option map did.
func connectionParams(dataSource *storepb.DataSource) map[string]string {
	params := make(map[string]string, len(dataSource.GetExtraConnectionParameters())+2)
	named := make(map[string]bool, len(params))
	for key, value := range dataSource.GetExtraConnectionParameters() {
		params[key] = value
		named[strings.ToUpper(key)] = true
	}
	for option, value := range map[string]string{
		"SERVICE_NAME": dataSource.GetServiceName(),
		"SID":          dataSource.GetSid(),
	} {
		if value == "" || named[option] {
			continue
		}
		params[option] = value
	}
	// An unset dial timeout leaves go-ora to the transport's own, which behind an
	// unreachable primary can outlast the whole connect check before the next
	// address in a SERVER list is tried. go-ora reads either spelling, and
	// upper-cases the keys, so the check is on the upper-cased names.
	if !named["CONNECTION TIMEOUT"] && !named["CONNECT TIMEOUT"] {
		params["CONNECTION TIMEOUT"] = "10"
	}
	return params
}

// getTLSConfig returns the TLS configuration for the data source's TLS
// section. It leaves the name check to crypto/tls instead of
// util.GetTLSConfig's verifier, which checks the data source's Host: go-ora
// sets ServerName to each server it dials, so a SERVER address or a listener
// redirect is checked against its own name.
func getTLSConfig(dataSource *storepb.DataSource) (*tls.Config, error) {
	config, err := util.GetTLSConfig(dataSource)
	if err != nil || config == nil {
		return config, err
	}
	config.InsecureSkipVerify = !dataSource.GetVerifyTlsCertificate()
	config.VerifyPeerCertificate = nil
	return config, nil
}

func (d *Driver) Close(context.Context) error {
	d.sessionMu.Lock()
	if d.session != nil {
		_ = d.session.Close()
		d.session = nil
	}
	d.sessionMu.Unlock()
	d.closeSSHClient()
	if d.impl != nil {
		return d.impl.Close()
	}
	return nil
}

func (d *Driver) closeSSHClient() {
	if d.sshClient != nil {
		_ = d.sshClient.Close()
		d.sshClient = nil
	}
}

// Ping reaches the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// Execute runs the script's statements, reporting each to the task run's log.
// The script's transaction-mode directive is parsed here, because the product
// sends it inside the statement text; the isolation directive is not applied,
// as before.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		return 0, errors.New("create database is not supported for Oracle")
	}
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	mode := config.Mode
	if mode == transaction.ModeUnspecified {
		mode = transaction.DefaultMode()
	}
	transactionMode := omnidriver.Transactional
	if mode == transaction.ModeOff {
		transactionMode = omnidriver.AutoCommit
	}
	return d.impl.Execute(ctx, withoutSQLPlusCommands(cleanedStatement), omnidriver.ExecuteOptions{
		Transaction: transactionMode,
		MaxRetries:  opts.MaximumRetries,
		Log:         executeLogger{opts: opts},
	})
}

// withoutSQLPlusCommands blanks the SQL*Plus commands of a script, which Oracle
// does not execute over a session: a script that starts with SET DEFINE OFF, as
// the SQL*Plus scripts the console imports do, used to run because Bytebase's
// splitter skipped them, and the driver sends every segment it splits.
//
// The commands are blanked rather than removed, so every other statement keeps
// its byte offsets and the range a task run logs still names the script's own
// text.
func withoutSQLPlusCommands(script string) string {
	var commands []oracleparser.Segment
	for _, segment := range oracleparser.Split(script) {
		if segment.Kind == oracleparser.SegmentSQLPlusCommand {
			commands = append(commands, segment)
		}
	}
	if len(commands) == 0 {
		return script
	}
	blanked := []byte(script)
	for _, command := range commands {
		end := command.ByteEnd
		// The command's line break goes with it, so the statement after it starts
		// at its own first character rather than at a blank line.
		for end < len(blanked) && (blanked[end] == '\r' || blanked[end] == '\n') {
			end++
		}
		for i := command.ByteStart; i < end; i++ {
			blanked[i] = ' '
		}
	}
	return string(blanked)
}

// CountAffectedRows returns the optimizer's estimate of the rows a DML
// statement changes.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.impl.CountAffectedRows(ctx, statement)
}

// OwnsPrivateDatabaseLink reports whether the driver's account owns a private
// database link. The driver asks with its own session; the old signature took
// the caller's *sql.Conn, which the cutover makes nil, and every linked read was
// refused because of it.
func (d *Driver) OwnsPrivateDatabaseLink(ctx context.Context) (bool, error) {
	return d.impl.OwnsPrivateDatabaseLink(ctx)
}

// executeLogger reports the driver contract's execute steps to the task run's
// log, which is how a task run shows what ran and what Oracle answered.
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
	var control storepb.TaskRunLog_TransactionControl_Type
	switch op {
	case omnidriver.Begin:
		control = storepb.TaskRunLog_TransactionControl_BEGIN
	case omnidriver.Commit:
		control = storepb.TaskRunLog_TransactionControl_COMMIT
	case omnidriver.Rollback:
		control = storepb.TaskRunLog_TransactionControl_ROLLBACK
	default:
		return
	}
	l.opts.LogTransactionControl(control, errorString(err))
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
