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
//   - db.SQLDB returns nil for this driver, which holds no *sql.DB. The editor's
//     *sql.Conn is not the contract's session, so the adapter holds one session
//     from the first QueryConn call until Close, under a mutex: a request's
//     statements share a session.
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
	"crypto/x509"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/singleflight"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	oceanbasedriver "github.com/bytebase/omni/oceanbase/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
	ds := config.DataSource
	awsConfig, err := util.GetAWSConnectionConfig(ctx, config)
	if err != nil {
		return "", nil, errors.Wrap(err, "load aws config failed")
	}
	if err := util.AssumeRoleIfNeeded(ctx, &awsConfig, config.ConnectionContext, ds.GetAwsCredential()); err != nil {
		return "", nil, err
	}
	token, err := auth.BuildAuthToken(
		ctx, fmt.Sprintf("%s:%s", ds.GetHost(), ds.GetPort()), ds.GetRegion(), ds.GetUsername(), awsConfig.Credentials)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create authentication token")
	}
	tlsConfig, err := rdsTLSConfig(ctx, ds)
	if err != nil {
		return "", nil, err
	}
	return token, tlsConfig, nil
}

// rdsCertBundleURL is a variable so tests can point it at a local server.
var rdsCertBundleURL = "https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem"

// rdsCertFetchTimeout bounds the download so a stalled fetch cannot hold a
// connection attempt open indefinitely.
const rdsCertFetchTimeout = 30 * time.Second

// maxRDSCertBundleSize bounds the downloaded bundle so a misbehaving endpoint
// cannot stream unbounded data into memory (the real global bundle is ~165 KB).
const maxRDSCertBundleSize int64 = 4 << 20

// rdsCertPool caches the downloaded bundle for the process lifetime.
var rdsCertPool atomic.Pointer[x509.CertPool]

// rdsCertGroup collapses concurrent cold-start downloads into one request.
var rdsCertGroup singleflight.Group

// getRDSCertPool downloads and returns the RDS CA certificate pool.
// AWS RDS connection with IAM require TLS connection.
//
// refs:
// https://github.com/aws/aws-sdk-go/issues/1248
// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/mysql-ssl-connections.html
func getRDSCertPool(ctx context.Context) (*x509.CertPool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rdsCertBundleURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build request for rds cert")
	}

	client := &http.Client{Timeout: rdsCertFetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Without this an error page would reach AppendCertsFromPEM and surface as
	// "failed to parse RDS CA certificates", hiding the real cause.
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to download RDS CA certificates: %s", resp.Status)
	}

	pem, err := io.ReadAll(io.LimitReader(resp.Body, maxRDSCertBundleSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(pem)) > maxRDSCertBundleSize {
		return nil, errors.Errorf("RDS CA bundle exceeds %d bytes", maxRDSCertBundleSize)
	}

	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return nil, errors.Errorf("failed to parse RDS CA certificates")
	}

	return rootCertPool, nil
}

// cachedRDSCertPool returns the AWS RDS CA bundle, downloading it once per
// process. Only successes are cached, so a transient failure can still recover.
func cachedRDSCertPool(ctx context.Context) (*x509.CertPool, error) {
	if pool := rdsCertPool.Load(); pool != nil {
		return pool, nil
	}
	// Collapse a cold-start burst into one download. The shared fetch does not
	// inherit any one caller's context, so the caller that starts it cannot
	// cancel it for everybody else; each caller waits on its own context below.
	ch := rdsCertGroup.DoChan("", func() (any, error) {
		// Double check after entering singleflight.
		if pool := rdsCertPool.Load(); pool != nil {
			return pool, nil
		}
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rdsCertFetchTimeout)
		defer cancel()
		pool, err := getRDSCertPool(fetchCtx)
		if err != nil {
			return nil, err
		}
		rdsCertPool.Store(pool)
		return pool, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		pool, ok := res.Val.(*x509.CertPool)
		if !ok {
			return nil, errors.Errorf("unexpected RDS cert pool type %T", res.Val)
		}
		return pool, nil
	}
}

// rdsRootCAs prefers an operator-supplied CA, so a deployment can serve RDS IAM
// connections without any outbound network access.
func rdsRootCAs(ctx context.Context, dataSource *storepb.DataSource) (*x509.CertPool, error) {
	if ca := dataSource.GetSslCa(); ca != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(ca)) {
			return nil, errors.Errorf("failed to parse ssl_ca certificates")
		}
		return pool, nil
	}
	return cachedRDSCertPool(ctx)
}

// rdsTLSConfig builds the TLS config for an RDS IAM connection. IAM auth always
// requires TLS, so this always returns a config.
func rdsTLSConfig(ctx context.Context, dataSource *storepb.DataSource) (*tls.Config, error) {
	if !dataSource.GetVerifyTlsCertificate() {
		// No verifier runs in this mode, so a root pool would never be consulted.
		return &tls.Config{InsecureSkipVerify: true}, nil // NOSONAR(go:S4830,go:S5527) operator disabled verification (verify_tls_certificate=false)
	}
	rootCertPool, err := rdsRootCAs(ctx, dataSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get RDS cert pool")
	}
	return &tls.Config{ // NOSONAR(go:S4830) VerifyPeerCertificate below verifies chain and hostname
		RootCAs:               rootCertPool,
		InsecureSkipVerify:    true, // superseded by VerifyPeerCertificate below
		VerifyPeerCertificate: util.CreateCertificateVerifier(rootCertPool, dataSource.GetHost()),
	}, nil
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

// GetDB returns nil: OceanBase is driven through omni's session, not a
// *sql.DB. The editor's *sql.Conn is unused, and a request's statements share
// one session.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// SyncInstance reads the server version, lower_case_table_names, accounts with
// their global grants, and the non-system databases.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]*storepb.InstanceRole, 0, len(instance.Roles))
	for _, role := range instance.Roles {
		roles = append(roles, &storepb.InstanceRole{
			Name:            role.Name,
			ConnectionLimit: role.ConnectionLimit,
			ValidUntil:      role.ValidUntil,
			Attribute:       role.Attribute,
		})
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
		Metadata: &storepb.Instance{
			MysqlLowerCaseTableNames: instance.LowerCaseTableNames,
			Roles:                    roles,
		},
	}, nil
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
	config, cleaned := base.ParseTransactionConfig(statement)
	statement = cleaned
	mode := config.Mode
	if mode == transaction.ModeUnspecified {
		mode = transaction.DefaultMode()
	}
	executeOptions := odriver.ExecuteOptions{
		Transaction: odriver.Transactional,
		Isolation:   odriver.IsolationLevel(config.Isolation),
		Log:         executeLogger{opts: opts},
	}
	if mode == transaction.ModeOff {
		executeOptions.Transaction = odriver.AutoCommit
	}
	return d.impl.Execute(ctx, statement, executeOptions)
}

// QueryConn runs a script for the editor on the driver's session, one result
// per statement, and stops at the first statement that fails. conn is unused:
// the caller's *sql.Conn, if it had one, is not the contract's Session.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
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
		ExplainFormat:  explainFormat(queryContext.Option.GetExplainFormat()),
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
	if len(results) == 0 {
		return nil, nil
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		queryResult := util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize)
		// An explain request named its own format, so the typed plan describes a
		// plan the caller already asked for; the driver this replaced reported it
		// only for an EXPLAIN statement the caller wrote.
		if queryContext.Explain {
			queryResult.QueryPlan = nil
		}
		converted = append(converted, queryResult)
	}
	return converted, nil
}

// explainFormat is the plan format in oceanbase/driver's vocabulary: JSON for
// a JSON plan and the engine's default for everything else, as the parser
// offers no other reader.
func explainFormat(format v1pb.QueryOption_ExplainFormat) string {
	if format == v1pb.QueryOption_JSON {
		return "JSON"
	}
	return ""
}

// executeLogger reports a change script's steps to the task run's log.
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
