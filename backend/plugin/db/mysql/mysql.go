// Package mysql is Bytebase's MySQL driver: an adapter over
// github.com/bytebase/omni/mysql/driver, and over github.com/bytebase/omni/mariadb/driver
// for the MARIADB engine value this package also registers. It maps a
// db.ConnectionConfig onto the contract's Config, the contract's results back
// onto the product's types, and keeps the credential, tunnel and TLS material
// in Bytebase where it belongs.
//
// The adapter keeps what is product: the data source to contract conversion,
// the AWS RDS and Google Cloud SQL IAM paths, the RDS CA bundle, SSH tunnels,
// TLS material, the transaction directive a script carries, the task-run
// logger, the QueryContext mapping, the v1pb conversion, and the live-review
// capabilities db.Prober and db.RowEstimator.
//
// GetDB keeps returning a *sql.DB, because db.Driver still requires it and
// gh-ost runs its five binlog probes through it. The contract's pool is the
// driver's, so the adapter owns a second pool built from the same resolved
// config; GetDB hands out that pool and Close closes it. Every caller of
// GetDB compiles and behaves as before.
//
// QueryConn holds one contract session from the first call until Close. The
// editor asks one statement at a time on a driver it keeps, so a session
// variable, a temporary table or an open transaction survives between calls,
// as it did on the caller's pooled *sql.Conn. The *sql.Conn argument is unused:
// the contract's Session is not one.
//
// Known differences from the driver this replaced:
//   - A script's parse error arrives before any statement of it runs, because
//     the contract reads the whole script first; statements after a failure are
//     not sent.
//   - Unsigned integers and the binary families arrive as the contract's typed
//     values rather than as strings, and an empty binary cell is empty bytes
//     rather than NULL.
//   - The result-size cap is byte-accurate rather than checked at power-of-two
//     row counts. A cap of zero keeps the old zero-byte behaviour: one row and
//     the "0MB" cap error, because the product reads zero as "no rows".
//   - Execute ignores db.ExecuteOptions.CreateDatabase and MaximumRetries, as
//     the old driver did; the txn-mode and txn-isolation directives still decide
//     the transaction mode and isolation level.
//   - A probe runs on a contract session: START TRANSACTION, the pre-executions
//     and the statement, then ROLLBACK, as the old *sql.DB probe did. With
//     ProbeContext.Autocommit the statement runs alone.
//   - SyncDBSchema answers driver.ErrNotFound for a database the server no
//     longer has, where the old driver returned the server's error.
//   - Instance roles are read in (user, host) order, and an account whose name
//     contains a quote is read instead of skipped.
package mysql

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/singleflight"

	odriver "github.com/bytebase/omni/driver"
	mariadbast "github.com/bytebase/omni/mariadb/ast"
	mariadbd "github.com/bytebase/omni/mariadb/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	mysqlast "github.com/bytebase/omni/mysql/ast"
	mysqld "github.com/bytebase/omni/mysql/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ db.Driver         = (*Driver)(nil)
	_ db.Prober         = (*Driver)(nil)
	_ db.RowEstimator   = (*Driver)(nil)
	_ db.SessionQuerier = (*Driver)(nil)
	_ db.SQLDBProvider  = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_MYSQL, newDriver)
	db.Register(storepb.Engine_MARIADB, newDriver)
}

// omniDriver is the method set the MySQL and MariaDB contract drivers share.
type omniDriver interface {
	Ping(ctx context.Context) error
	Close() error
	SyncInstance(ctx context.Context) (*odriver.Instance, error)
	SyncDatabase(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error)
	Dump(ctx context.Context, w io.Writer, db *metadatapb.DatabaseSchemaMetadata) error
	Session(ctx context.Context) (odriver.Session, error)
	Execute(ctx context.Context, script string, opts odriver.ExecuteOptions) (int64, error)
	CountAffectedRows(ctx context.Context, statement string) (int64, error)
	ExplainJSON(ctx context.Context, statement string) (string, error)
}

// Driver is Bytebase's MySQL driver.
type Driver struct {
	dbType storepb.Engine
	driver omniDriver
	// sqlDB is the direct-connection pool GetDB hands out. The contract's
	// pool is the driver's own and is not reachable, so this adapter owns a
	// second one built from the same resolved config. gh-ost runs its binlog
	// probes through it.
	sqlDB *sql.DB

	sshClient *ssh.Client
	// cloudSQLDialer owns the background certificate refresh; it lives and
	// dies with the driver.
	cloudSQLDialer *cloudsqlconn.Dialer

	// sessionMu guards session. The editor keeps one driver for a whole stream,
	// so one contract session serves it from the first QueryConn until Close.
	sessionMu sync.Mutex
	session   odriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the resolved data source onto the contract's Config.
func (d *Driver) Open(ctx context.Context, dbType storepb.Engine, connCfg db.ConnectionConfig) (db.Driver, error) {
	// A driver whose Open fails is discarded without a Close, so Open releases
	// the tunnel, Cloud SQL dialer and pools it acquired itself.
	success := false
	defer func() {
		if !success {
			_ = d.Close(ctx)
		}
	}()

	dataSource := connCfg.DataSource
	if dataSource == nil {
		return nil, errors.New("mysql: a data source is required")
	}
	d.dbType = dbType

	cfg := odriver.Config{
		Host:            dataSource.Host,
		Port:            dataSource.Port,
		Database:        connCfg.ConnectionContext.DatabaseName,
		User:            dataSource.Username,
		Password:        connCfg.Password,
		ApplicationName: "bytebase",
		EngineVersion:   connCfg.ConnectionContext.EngineVersion,
		Params:          dataSource.GetExtraConnectionParameters(),
	}

	switch dataSource.GetAuthenticationType() {
	case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		dialer, err := util.GetGCPConnectionConfig(ctx, connCfg)
		if err != nil {
			return nil, errors.Wrap(err, "load gcp config failed")
		}
		d.cloudSQLDialer = dialer
		instance := dataSource.Host
		cfg.Dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.Dial(ctx, instance)
		}
	case storepb.DataSource_AWS_RDS_IAM:
		if err := d.openRDS(ctx, connCfg, &cfg); err != nil {
			return nil, err
		}
	default:
		if dataSource.GetSshHost() != "" {
			sshClient, err := util.GetSSHClient(dataSource)
			if err != nil {
				return nil, err
			}
			d.sshClient = sshClient
			cfg.Dial = func(_ context.Context, _, address string) (net.Conn, error) {
				return sshClient.Dial("tcp", address)
			}
		}
		tlsConfig, err := util.GetTLSConfig(dataSource)
		if err != nil {
			return nil, errors.Wrap(err, "sql: tls config error")
		}
		if tlsConfig != nil {
			cfg.TLS = func(string) (*tls.Config, error) { return tlsConfig, nil }
		}
	}

	var opened omniDriver
	var err error
	switch dbType {
	case storepb.Engine_MYSQL:
		opened, err = mysqld.Open(ctx, cfg)
	case storepb.Engine_MARIADB:
		opened, err = mariadbd.Open(ctx, cfg)
	default:
		return nil, errors.Errorf("mysql: unsupported engine %s", dbType)
	}
	if err != nil {
		return nil, err
	}
	d.driver = opened

	// The direct-connection pool GetDB hands out uses the same resolved values,
	// including the tunnel and TLS config, so gh-ost reaches the same server.
	sqlDB, err := newSQLDB(cfg)
	if err != nil {
		return nil, err
	}
	d.sqlDB = sqlDB
	success = true
	return d, nil
}

// newSQLDB builds the direct-connection pool GetDB returns from the contract
// config Open already resolved. It never replaces the driver's own pool; it is
// the one a caller that needs database/sql (gh-ost's binlog probes) uses.
func newSQLDB(cfg odriver.Config) (*sql.DB, error) {
	params := []string{"multiStatements=true", "maxAllowedPacket=0"}
	keys := make([]string, 0, len(cfg.Params))
	for key := range cfg.Params {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if strings.EqualFold(strings.TrimSpace(key), "allowallfiles") {
			return nil, errors.Errorf("connection parameter %q is not allowed for security reasons", key)
		}
		params = append(params, fmt.Sprintf("%s=%s", key, cfg.Params[key]))
	}
	mc, err := gomysql.ParseDSN("u@tcp(placeholder:3306)/?" + strings.Join(params, "&"))
	if err != nil {
		return nil, err
	}
	if mc.AllowAllFiles {
		return nil, errors.New("connection parameter \"allowAllFiles\" is not allowed for security reasons")
	}
	mc.User = cfg.User
	mc.Passwd = cfg.Password
	mc.DBName = cfg.Database

	switch {
	case cfg.Dial != nil:
		mc.Net = "tcp"
		switch {
		case cfg.Host != "" && cfg.Port != "":
			mc.Addr = net.JoinHostPort(cfg.Host, cfg.Port)
		case cfg.Host != "":
			mc.Addr = cfg.Host
		default:
			mc.Addr = "dial"
		}
		mc.DialFunc = cfg.Dial
	case strings.HasPrefix(cfg.Host, "/"):
		mc.Net = "unix"
		mc.Addr = cfg.Host
	default:
		mc.Net = "tcp"
		mc.Addr = net.JoinHostPort(cfg.Host, cfg.Port)
	}
	if cfg.TLS != nil {
		tlsConfig, err := cfg.TLS(cfg.Host)
		if err != nil {
			return nil, err
		}
		mc.TLS = tlsConfig
		mc.TLSConfig = ""
		mc.AllowFallbackToPlaintext = false
	}
	connector, err := gomysql.NewConnector(mc)
	if err != nil {
		return nil, err
	}
	sqlDB := sql.OpenDB(connector)
	sqlDB.SetConnMaxLifetime(2 * time.Hour)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(15)
	return sqlDB, nil
}

func (*Driver) openRDS(ctx context.Context, connCfg db.ConnectionConfig, cfg *odriver.Config) error {
	dataSource := connCfg.DataSource
	dbEndpoint := fmt.Sprintf("%s:%s", dataSource.Host, dataSource.Port)
	awsConfig, err := util.GetAWSConnectionConfig(ctx, connCfg)
	if err != nil {
		return errors.Wrap(err, "load aws config failed")
	}
	if err := util.AssumeRoleIfNeeded(ctx, &awsConfig, connCfg.ConnectionContext, dataSource.GetAwsCredential()); err != nil {
		return err
	}
	token, err := auth.BuildAuthToken(ctx, dbEndpoint, dataSource.GetRegion(), dataSource.Username, awsConfig.Credentials)
	if err != nil {
		return errors.Wrap(err, "failed to create authentication token")
	}
	tlsConfig, err := rdsTLSConfig(ctx, dataSource)
	if err != nil {
		return err
	}
	cfg.Password = token
	cfg.TLS = func(string) (*tls.Config, error) { return tlsConfig, nil }
	// RDS IAM authenticates with the clear-password plugin, which the client
	// only sends when it is allowed to.
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["allowCleartextPasswords"] = "true"
	return nil
}

func (d *Driver) Close(context.Context) error {
	var err error
	d.sessionMu.Lock()
	if d.session != nil {
		err = multierr.Append(err, d.session.Close())
		d.session = nil
	}
	d.sessionMu.Unlock()
	if d.driver != nil {
		err = multierr.Append(err, d.driver.Close())
	}
	if d.sqlDB != nil {
		err = multierr.Append(err, d.sqlDB.Close())
	}
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
	}
	if d.cloudSQLDialer != nil {
		err = multierr.Append(err, d.cloudSQLDialer.Close())
	}
	return err
}

func (d *Driver) Ping(ctx context.Context) error {
	return d.driver.Ping(ctx)
}

// GetDB returns the direct-connection pool gh-ost's binlog probes use. It is
// not the contract's pool, which the driver keeps private.
func (d *Driver) GetDB() *sql.DB {
	return d.sqlDB
}

// OwnsQuerySession reports that QueryConn runs on the adapter's held session,
// not on a connection a caller opens from GetDB.
func (*Driver) OwnsQuerySession() {}

// SyncInstance converts the contract's instance into the API's metadata.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.driver.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]*storepb.InstanceRole, len(instance.Roles))
	for i, role := range instance.Roles {
		roles[i] = &storepb.InstanceRole{
			Name:            role.Name,
			ConnectionLimit: role.ConnectionLimit,
			ValidUntil:      role.ValidUntil,
			Attribute:       role.Attribute,
		}
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

func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.driver.SyncDatabase(ctx)
}

func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.driver.Dump(ctx, out, dbMetadata)
}

// Execute runs the script's statements, reporting each step to the task run
// log. The transaction directive is read here unless the caller already did.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	config, cleaned := base.ParseTransactionConfig(statement)
	if config.Mode == transaction.ModeUnspecified {
		config.Mode = transaction.DefaultMode()
	}
	mode := odriver.Transactional
	if config.Mode == transaction.ModeOff {
		mode = odriver.AutoCommit
	}
	// CreateDatabase and MaximumRetries are not mapped: the driver this
	// replaced read neither, and mapping them would change a create-database
	// script's transaction shape and retry a script the product did not retry.
	return d.driver.Execute(ctx, cleaned, odriver.ExecuteOptions{
		Transaction: mode,
		Isolation:   odriver.IsolationLevel(config.Isolation),
		Log:         &executeLogger{opts: opts},
	})
}

// QueryConn runs the script on the driver's session. The *sql.Conn argument is
// the caller's pooled connection, which the contract cannot accept; the adapter
// holds its own session instead, from the first call until Close, so a stream's
// session state survives between calls.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	if d.session == nil {
		session, err := d.driver.Session(ctx)
		if err != nil {
			return nil, err
		}
		d.session = session
	}
	session := d.session

	// The product reads a zero cap as "no rows", and the contract reads it as
	// "no cap". A one-byte cap keeps the old zero-byte behaviour without letting
	// the query read every row a policy meant to bound.
	maxResultBytes := queryContext.MaximumSQLResultSize
	if maxResultBytes <= 0 {
		maxResultBytes = 1
	}
	results, err := session.Query(ctx, statement, odriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  explainFormat(queryContext),
		MaxResultBytes: maxResultBytes,
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		// The client closes the connection when a statement's context ends, and an
		// errored statement leaves the session's state unknown; either way the next
		// statement on this stream would find a session it cannot use.
		_ = session.Close()
		d.session = nil
		return nil, err
	}
	if ctx.Err() != nil {
		_ = session.Close()
		d.session = nil
	}
	queryResults := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		queryResult := util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize)
		if queryContext.Explain {
			// An explain request's rows are the plan already; the old driver
			// reported no typed plan for one.
			queryResult.QueryPlan = nil
		}
		queryResults = append(queryResults, queryResult)
	}
	return queryResults, nil
}

// Probe runs a live review rule's probe on a fresh session. The contract has no
// probe primitive, so the adapter starts a transaction, runs the pre-executions
// and the statement in order, and rolls back, as the old *sql.DB probe did.
// pCtx.Autocommit runs the statement alone; a MySQL probe is an EXPLAIN, so
// either shape answers the same.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	session, err := d.driver.Session(ctx)
	if err != nil {
		return db.ProbeResult{}, err
	}
	defer session.Close()

	if !pCtx.Autocommit {
		if _, err := session.Query(ctx, "START TRANSACTION", odriver.QueryOptions{}); err != nil {
			return db.ProbeResult{}, errors.Wrap(err, "failed to start the probe transaction")
		}
		defer func() {
			_, _ = session.Query(ctx, "ROLLBACK", odriver.QueryOptions{})
		}()
	}
	for _, preExec := range pCtx.PreExecutions {
		if preExec == "" {
			continue
		}
		if _, err := session.Query(ctx, preExec, odriver.QueryOptions{}); err != nil {
			return db.ProbeResult{}, errors.Wrapf(err, "failed to execute pre-execution: %s", preExec)
		}
	}
	results, err := session.Query(ctx, statement, odriver.QueryOptions{})
	if err != nil {
		return db.ProbeResult{}, err
	}
	if len(results) == 0 {
		return db.ProbeResult{}, nil
	}
	result := results[len(results)-1]
	if result.Err != nil {
		return db.ProbeResult{}, result.Err
	}
	probeResult := db.ProbeResult{
		Columns:     make([]string, len(result.Columns)),
		ColumnTypes: make([]string, len(result.Columns)),
	}
	for i, column := range result.Columns {
		probeResult.Columns[i] = column.Name
		probeResult.ColumnTypes[i] = column.TypeName
	}
	for _, row := range result.Rows {
		values := make([]any, len(row))
		for i, value := range row {
			values[i] = probeValue(value)
		}
		probeResult.Rows = append(probeResult.Rows, values)
	}
	return probeResult, nil
}

// probeValue converts one contract cell to the untyped value a rule reads.
func probeValue(value odriver.Value) any {
	switch value.Kind {
	case odriver.Bool:
		return value.Bool
	case odriver.Bytes:
		return value.Bytes
	case odriver.Float32, odriver.Float64:
		return value.Float64
	case odriver.Int32, odriver.Int64:
		return value.Int64
	case odriver.Uint32, odriver.Uint64:
		return value.Uint64
	case odriver.String:
		return value.String
	case odriver.Structured:
		return value.Structured
	case odriver.Timestamp, odriver.TimestampTZ:
		return value.Time
	default:
		return nil
	}
}

func explainFormat(queryContext db.QueryContext) string {
	switch db.ExplainFormat(queryContext.Option.GetExplainFormat()) {
	case base.ExplainFormatJSON:
		return "JSON"
	case base.ExplainFormatText:
		return "TEXT"
	default:
		return ""
	}
}

// CountAffectedRows returns the planner's estimate for one DML statement.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.driver.CountAffectedRows(ctx, statement)
}

// ExplainJSON returns EXPLAIN FORMAT=JSON for one statement.
func (d *Driver) ExplainJSON(ctx context.Context, statement string) (string, error) {
	return d.driver.ExplainJSON(ctx, statement)
}

// EstimateAffectedRows reads a MySQL node's estimate from plan.
func (d *Driver) EstimateAffectedRows(ctx context.Context, stmt mysqlast.Node, query, plan string) (int64, error) {
	driver, ok := d.driver.(*mysqld.Driver)
	if !ok {
		return 0, errors.New("mysql: EstimateAffectedRows needs the MySQL driver")
	}
	return driver.EstimateAffectedRows(ctx, stmt, query, plan)
}

// EstimateAffectedRowsMariaDB reads a MariaDB node's estimate from plan.
func (d *Driver) EstimateAffectedRowsMariaDB(ctx context.Context, stmt mariadbast.Node, query, plan string) (int64, error) {
	driver, ok := d.driver.(*mariadbd.Driver)
	if !ok {
		return 0, errors.New("mysql: EstimateAffectedRowsMariaDB needs the MariaDB driver")
	}
	return driver.EstimateAffectedRows(ctx, stmt, query, plan)
}

// executeLogger bridges the contract's ExecuteLogger onto the task run log.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l *executeLogger) CommandExecute(command odriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{Start: int32(command.ByteStart), End: int32(command.ByteEnd)}, command.Text)
}

func (l *executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (l *executeLogger) TransactionControl(op odriver.TransactionOp, err error) {
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

// rdsCertBundleURL is a variable so tests can point it at a local server.
var rdsCertBundleURL = "https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem"

const rdsCertFetchTimeout = 30 * time.Second

const maxRDSCertBundleSize int64 = 4 << 20

var (
	rdsCertPool  atomic.Pointer[x509.CertPool]
	rdsCertGroup singleflight.Group
)

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

func cachedRDSCertPool(ctx context.Context) (*x509.CertPool, error) {
	if pool := rdsCertPool.Load(); pool != nil {
		return pool, nil
	}
	ch := rdsCertGroup.DoChan("", func() (any, error) {
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

// rdsTLSConfig builds the TLS config for an RDS IAM connection.
func rdsTLSConfig(ctx context.Context, dataSource *storepb.DataSource) (*tls.Config, error) {
	if !dataSource.GetVerifyTlsCertificate() {
		return &tls.Config{InsecureSkipVerify: true}, nil // NOSONAR(go:S4830,go:S5527) operator disabled verification
	}
	rootCertPool, err := rdsRootCAs(ctx, dataSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get RDS cert pool")
	}
	return &tls.Config{ // NOSONAR(go:S4830) VerifyPeerCertificate below verifies chain and hostname
		RootCAs:               rootCertPool,
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: util.CreateCertificateVerifier(rootCertPool, dataSource.GetHost()),
	}, nil
}
