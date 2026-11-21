// Package tidb is Bytebase's TiDB driver: an adapter over
// github.com/bytebase/omni/tidb/driver. It maps a db.ConnectionConfig onto the
// contract's Config and the contract's results back onto the product's types;
// the connection, the sync queries, the plan estimate, the splitter, the query
// rewrites and the dump belong to the omni driver.
//
// The adapter keeps what is product: the data source to contract conversion,
// the SSH tunnel and TLS material, the `-- txn-mode` directive, the advisor
// probe, the v1pb conversion and the task-run logger.
//
// Constraints of the engine and the contract, here rather than at each call:
//   - Config.ReadOnly is unused: TiDB's MySQL wire handshake has no read-only
//     session parameter.
//   - Config.ApplicationName is "bytebase", which renders the /*app=bytebase*/
//     comment every statement of a query or a change script carries.
//   - Execute ignores ExecuteOptions.CreateDatabase and the `-- txn-isolation`
//     directive: a create-database script runs under the transaction directive,
//     and no isolation level is set.
//   - A change script still reads its transaction mode from
//     ParseTransactionConfig, so the directives do not reach the server.
//   - A probe runs on one session so its pre-executions and its statement share
//     a transaction, opened with the "START TRANSACTION" go-sql-driver's
//     BeginTx sends and ended with "ROLLBACK".
//   - QueryConn runs on the omni driver's Session, which the adapter keeps
//     until Close, so the statements of one editor request share it.
//   - Dump renders omni's catalog, whose partitions keep their ordinal order;
//     that order is what makes a partitioned table's dump restorable.
//   - Query results carry the contract's typed values: an unsigned integer is a
//     Uint64 value, the BLOB, BINARY and VARBINARY families are bytes, and an
//     empty binary is empty bytes rather than NULL.
//   - A result-size cap is checked against a row's real encoded size, and a cap
//     of zero is refused: the product reads 0 as "no rows" and the contract
//     reads it as "no cap".
//   - Instance roles are read in a deterministic (user, host) order, and a
//     grant names its account with proper quoting, so an account whose name
//     contains a quote is read.
package tidb

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	tidbdriver "github.com/bytebase/omni/tidb/driver"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver = (*Driver)(nil)

	// applicationName renders the /*app=bytebase*/ comment the old driver
	// prepended to a query and to a change script's commands.
	applicationName = "bytebase"
)

func init() {
	db.Register(storepb.Engine_TIDB, newDriver)
}

// Driver is Bytebase's TiDB driver.
type Driver struct {
	impl         *tidbdriver.Driver
	databaseName string
	// sshClient is the tunnel an SSH data source opened; Close closes it.
	sshClient *ssh.Client

	// sessionMu guards session. A driver is opened per request, and the editor
	// asks for one statement at a time, but the session is one server
	// connection: the lock covers acquisition, the query and teardown so two
	// calls can never interleave their statements on it.
	sessionMu sync.Mutex
	session   omnidriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the contract's Config. Like the driver
// this replaced, Open does not reach the server; Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	driverConfig, sshClient, err := buildConfig(config)
	if err != nil {
		return nil, err
	}
	impl, err := tidbdriver.Open(ctx, driverConfig)
	if err != nil {
		if sshClient != nil {
			_ = sshClient.Close()
		}
		return nil, err
	}
	d.impl = impl
	d.sshClient = sshClient
	d.databaseName = config.ConnectionContext.DatabaseName
	return d, nil
}

// buildConfig resolves what Bytebase knows about reaching a TiDB server into the
// plain values tidb/driver takes. The returned SSH client, when not nil, is the
// tunnel the driver dials through and belongs to the caller to close.
func buildConfig(config db.ConnectionConfig) (omnidriver.Config, *ssh.Client, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return omnidriver.Config{}, nil, errors.New("a data source is required")
	}
	params := map[string]string{}
	for key, value := range dataSource.GetExtraConnectionParameters() {
		params[key] = value
	}
	driverConfig := omnidriver.Config{
		Host:            dataSource.Host,
		Port:            dataSource.Port,
		Database:        config.ConnectionContext.DatabaseName,
		User:            dataSource.Username,
		Password:        config.Password,
		Params:          params,
		ApplicationName: applicationName,
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}

	var sshClient *ssh.Client
	if dataSource.GetSshHost() != "" {
		client, err := util.GetSSHClient(dataSource)
		if err != nil {
			return omnidriver.Config{}, nil, err
		}
		sshClient = client
		driverConfig.Dial = func(_ context.Context, network, address string) (net.Conn, error) {
			return client.Dial(network, address)
		}
	}

	if dataSource.GetUseSsl() {
		// The contract asks for one host at a time so a caller can bind
		// verification to the host it reaches; Bytebase's material does not
		// depend on which one that is.
		driverConfig.TLS = func(string) (*tls.Config, error) {
			return util.GetTLSConfig(dataSource)
		}
	}
	return driverConfig, sshClient, nil
}

func (d *Driver) Close(context.Context) error {
	var err error
	d.sessionMu.Lock()
	if d.session != nil {
		err = multierr.Append(err, d.session.Close())
		d.session = nil
	}
	d.sessionMu.Unlock()
	err = multierr.Append(err, d.impl.Close())
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
	}
	return err
}

// Ping reaches the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// Execute runs the script's statements under the transaction directive it
// carries, reporting each step to the task run log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	mode, _, cleanedStatement := util.ParseTransactionMode(statement)
	return d.impl.Execute(ctx, cleanedStatement, omnidriver.ExecuteOptions{
		Transaction: mode,
		MaxRetries:  opts.MaximumRetries,
		Log:         util.NewExecuteLogger(opts),
	})
}

// QueryConn queries a SQL statement on the driver's session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	// A non-positive cap means "no rows" to the product and "no cap" to the
	// contract, so refuse it rather than read every row a policy meant to
	// bound.
	if queryContext.MaximumSQLResultSize <= 0 {
		return nil, errors.New("maximum SQL result size must be positive")
	}
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	session, err := d.querySessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		// The script could not be run at all; the session is not one to keep.
		d.discardSessionLocked(session)
		return nil, err
	}
	if ctx.Err() != nil {
		// The client closes the connection when a statement's context ends, so the
		// stream's next statement must open a new session.
		d.discardSessionLocked(session)
	}

	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, queryContext.Explain), nil
}

// discardSessionLocked closes a session a statement took with it, so the
// stream's next statement opens a new one instead of reusing the dead one. The
// caller holds sessionMu.
func (d *Driver) discardSessionLocked(session omnidriver.Session) {
	_ = session.Close()
	if d.session == session {
		d.session = nil
	}
}

// querySessionLocked returns the session QueryConn runs on, opening it on first
// use. The caller holds sessionMu.
func (d *Driver) querySessionLocked(ctx context.Context) (omnidriver.Session, error) {
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

// SyncInstance reads the server version, its user databases and its roles.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return util.OmniInstanceToMetadata(instance, true), nil
}

// SyncDBSchema syncs the connected database's schema. A database the server no
// longer has keeps answering the product's not-found error, as the old driver
// did when the database row was missing.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	metadata, err := d.impl.SyncDatabase(ctx)
	if err != nil {
		if errors.Is(err, omnidriver.ErrNotFound) {
			return nil, common.Errorf(common.NotFound, "database %q not found", d.databaseName)
		}
		return nil, err
	}
	return metadata, nil
}

// Dump writes the synced schema as TiDB's DDL.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, dbMetadata)
}

// CountAffectedRows returns TiDB's planner estimate of the rows statement
// modifies. The contract answers it for exactly one INSERT, UPDATE or DELETE and
// refuses any other statement before it reaches the server.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.impl.CountAffectedRows(ctx, statement)
}
