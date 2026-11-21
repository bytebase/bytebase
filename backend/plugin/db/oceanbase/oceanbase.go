// Package oceanbase is Bytebase's OceanBase driver: an adapter over
// github.com/bytebase/omni/oceanbase/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos. The conversation with OceanBase's MySQL-mode tenant —
// version, lower_case_table_names, accounts and their grants, the sync queries,
// the mysqldump-style dump, result decoding, EXPLAIN and change scripts — is
// the driver's.
//
// The adapter keeps what is product: the connection path (AWS RDS and Google
// Cloud SQL IAM tokens, the RDS CA bundle, SSH tunnels, TLS material, extra
// connection parameters), the transaction directive a script carries, the
// change-script logger, the QueryContext mapping, the v1pb conversion and the
// three advisor capabilities.
//
// OceanBase takes the MySQL family's connection mapping, because it speaks the
// MySQL wire protocol and go-sql-driver is its client: a host that starts with
// "/" is a unix socket, multiStatements and an unbounded maxAllowedPacket are
// always on, an extra parameter is passed through with allowAllFiles refused,
// and an SSH data source dials a tunnel. Those rules are db/mysql's too; this
// package states them once here rather than sharing its DSN builder, because
// the driver contract takes the values instead of a DSN.
//
// The dump renders the metadata this driver syncs. What that DDL states about an
// object — a generated column's clause, a materialized view and its container,
// an event's schedule, an external table's location — is omni's OceanBase
// catalog and sync, recorded and verified by omni's own corpus.
//
// Constraints of the engine and the contract, here rather than at each call:
//
//   - The editor's *sql.Conn is not the contract's session, so the adapter
//     holds one session from the first QueryConn call until Close, under a
//     mutex: a request's statements share a session.
//   - Config.ReadOnly and Config.TokenProvider are not mapped: OceanBase's MySQL
//     handshake has no read-only session parameter, and its login is a password.
//   - ExecuteOptions.CreateDatabase and MaximumRetries are not mapped, because the
//     driver this replaced read neither.
//   - A query result's unsigned integer column is a Uint64 value rather than a
//     string, and an empty binary cell is a Bytes value rather than NULL.
//   - QueryOptions.MaxResultBytes of 0 is no cap; the contract reads 0 that way,
//     where the driver this replaced compared sizes against it literally.
//   - A script the parser cannot read fails QueryConn instead of returning the
//     server's syntax error on that statement's result, because the driver reads
//     every statement before it sends the first.
//   - db.Prober.Probe runs on a session of its own: SELECT CONNECTION_ID(), then
//     START TRANSACTION, the pre-executions, the statement, and ROLLBACK.
//   - CountAffectedRows refuses a statement that is not exactly one INSERT,
//     UPDATE or DELETE before it reaches the server.
package oceanbase

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	oceanbasedriver "github.com/bytebase/omni/oceanbase/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_OCEANBASE, newDriver)
}

// applicationName is the /*app=…*/ comment every query and change-script
// statement carries, as the DSN it replaced carried it.
const applicationName = "bytebase"

// Driver is Bytebase's OceanBase driver.
type Driver struct {
	impl      *oceanbasedriver.Driver
	sshClient *ssh.Client

	// sessionMu guards session. The editor asks for one statement at a time on
	// a driver it keeps for the whole stream, so the adapter holds one session
	// for the driver's lifetime: what a statement sets (a user variable, a
	// temporary table, a session variable) holds for the next, as it did on the
	// caller's *sql.Conn.
	sessionMu sync.Mutex
	session   odriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server; Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, err := driverConfig(ctx, config, d)
	if err != nil {
		_ = d.Close(ctx)
		return nil, err
	}
	impl, err := oceanbasedriver.Open(ctx, cfg)
	if err != nil {
		_ = d.Close(ctx)
		return nil, err
	}
	d.impl = impl
	return d, nil
}

// driverConfig resolves what Bytebase knows about reaching one OceanBase tenant
// into the plain values oceanbase/driver's go-sql-driver connector takes.
func driverConfig(ctx context.Context, config db.ConnectionConfig, d *Driver) (odriver.Config, error) {
	ds := config.DataSource
	cfg := odriver.Config{
		Host:            ds.GetHost(),
		Port:            ds.GetPort(),
		Database:        config.ConnectionContext.DatabaseName,
		User:            ds.GetUsername(),
		Password:        config.Password,
		ApplicationName: applicationName,
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}

	switch ds.GetAuthenticationType() {
	case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		dialer, err := util.GetGCPConnectionConfig(ctx, config)
		if err != nil {
			return odriver.Config{}, errors.Wrap(err, "load gcp config failed")
		}
		// The connector names its instance itself, so it ignores the address the
		// driver passes and dials the data source's host.
		cfg.Dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.Dial(ctx, ds.GetHost())
		}
		cfg.Params = map[string]string{"parseTime": "true"}
	case storepb.DataSource_AWS_RDS_IAM:
		token, tlsConfig, err := rdsLogin(ctx, config)
		if err != nil {
			return odriver.Config{}, err
		}
		cfg.Password = token
		cfg.TLS = func(string) (*tls.Config, error) {
			return tlsConfig, nil
		}
		cfg.Params = map[string]string{"allowCleartextPasswords": "true"}
	default:
		params, err := connectionParams(ds.GetExtraConnectionParameters())
		if err != nil {
			return odriver.Config{}, err
		}
		cfg.Params = params
		if ds.GetSshHost() != "" {
			sshClient, err := util.GetSSHClient(ds)
			if err != nil {
				return odriver.Config{}, err
			}
			d.sshClient = sshClient
			cfg.Dial = func(_ context.Context, network, address string) (net.Conn, error) {
				return sshClient.Dial(network, address)
			}
		}
		if ds.GetUseSsl() {
			// GetTLSConfig binds certificate verification to the data source's host,
			// which is the host the driver dials.
			cfg.TLS = func(string) (*tls.Config, error) {
				return util.GetTLSConfig(ds)
			}
		}
	}
	return cfg, nil
}

// connectionParams passes the data source's own parameters through, refusing
// the one that disables go-sql-driver's local-infile allowlist.
func connectionParams(extra map[string]string) (map[string]string, error) {
	params := make(map[string]string, len(extra))
	for key, value := range extra {
		if strings.EqualFold(strings.TrimSpace(key), "allowallfiles") {
			return nil, errors.Errorf("connection parameter %q is not allowed for security reasons", key)
		}
		params[key] = value
	}
	return params, nil
}

// rdsLogin returns the IAM authentication token and the TLS config an AWS RDS
// connection uses. IAM authentication always requires TLS.
func rdsLogin(ctx context.Context, config db.ConnectionConfig) (string, *tls.Config, error) {
	token, err := util.RDSAuthToken(ctx, config)
	if err != nil {
		return "", nil, err
	}
	tlsConfig, err := util.RDSTLSConfig(ctx, config.DataSource)
	if err != nil {
		return "", nil, err
	}
	return token, tlsConfig, nil
}

// Close closes the driver's session, its connections and the tunnel they used.
func (d *Driver) Close(context.Context) error {
	var err error
	d.sessionMu.Lock()
	if d.session != nil {
		err = multierr.Append(err, d.session.Close())
		d.session = nil
	}
	d.sessionMu.Unlock()
	if d.impl != nil {
		err = multierr.Append(err, d.impl.Close())
	}
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
	}
	return err
}

// Ping reaches the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// SyncInstance reads the server version, lower_case_table_names, accounts with
// their global grants, and the non-system databases.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return util.OmniInstanceToMetadata(instance, true), nil
}

// SyncDBSchema reads the database the driver was opened on.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.impl.SyncDatabase(ctx)
}

// Dump writes the database's DDL through omni's OceanBase catalog, which
// renders the synced metadata this driver returns.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, dbMetadata)
}

// Execute runs a change script under the transaction directive it carries.
// CreateDatabase and MaximumRetries are not mapped: the driver this replaced
// read neither.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	transactionMode, isolation, cleaned := util.ParseTransactionMode(statement)
	return d.impl.Execute(ctx, cleaned, odriver.ExecuteOptions{
		Transaction: transactionMode,
		Isolation:   isolation,
		Log:         util.NewExecuteLogger(opts),
	})
}

// QueryConn runs a script for the editor on the driver's session, one result
// per statement, and stops at the first statement that fails.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	if d.session == nil {
		session, err := d.impl.Session(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open a session")
		}
		d.session = session
	}
	session := d.session

	results, err := session.Query(ctx, statement, odriver.QueryOptions{
		Explain:        queryContext.Explain,
		ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_JSON),
		Limit:          queryContext.Limit,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		// The script could not be run at all; the session is not one to keep.
		_ = session.Close()
		d.session = nil
		return nil, err
	}
	if ctx.Err() != nil {
		// The client closes the connection when a statement's context ends, so the
		// stream's next statement must open a new session.
		_ = session.Close()
		d.session = nil
	}
	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, queryContext.Explain), nil
}
