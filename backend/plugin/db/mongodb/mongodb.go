// Package mongodb is Bytebase's MongoDB driver: a thin adapter over
// github.com/bytebase/omni/mongo/driver. The connection string, the sync
// queries, the mongosh execution, the result rendering and the dump belong to
// the omni driver; this package maps a db.ConnectionConfig onto the contract's
// Config and the contract's results back into the product's types.
//
// Bytebase registers this engine as MONGODB and dispatches through this
// package, while omni's tree is mongo/driver. Neither name is changed: the
// engine value is the product's, and the package path is what the driver
// registry, the parser registration and the completion registration already
// use, so a rename here would be churn outside the cutover's subject.
//
// The adapter keeps what is product-side:
//
//   - Open maps a data source onto driver.Config: the host list with each
//     additional address's own port, the database, the login, the auth
//     database, replica set, direct connection and extra connection parameters
//     as connection string options, mongodb+srv for an SRV data source, the TLS
//     material, and an SSH Dial. It does not reach the server; Ping does.
//   - Execute reports each statement to the task run log.
//   - SyncInstance converts the contract's roles into the instance metadata.
//
// Every db.Driver method, including the ones that answer "not supported":
//
//   - Open, Close, Ping, SyncInstance, SyncDBSchema and Dump delegate.
//   - GetDB returns nil. MongoDB has no database/sql handle, and the driver
//     this replaced answered nil too.
//   - QueryConn opens one contract session per call. MongoDB keeps no session
//     state this driver sets, so a session is not held between calls; conn is
//     unused, since the caller's *sql.Conn is not the contract's Session.
//   - Execute runs each statement on its own. There is no transaction: MongoDB
//     has none this path takes part in, so the contract's TransactionMode and
//     IsolationLevel are left unset, and ExecuteOptions.MaximumRetries reaches
//     the contract unused, as the driver never retries a statement. There is no
//     DDL either: a change script is mongosh, and its statements are run in
//     order.
//   - ReadOnly, QueryOptions.Schema and QueryOptions.MaskingEnabled do nothing:
//     MongoDB has no read-only session, no schema per session, and no masking
//     to withhold.
//
// Behaviour that moves with the driver, each pinned in cutover_parity_test.go
// except where noted:
//
//   - Dump is no longer a no-op. It renders the synced database as a mongosh
//     script that recreates its collections with their options, its indexes and
//     its views, where the old driver wrote nothing; a view whose synced
//     metadata carries no definition is named in a comment, since it cannot be
//     recreated without one. TestCutoverDumpRoundTrip drops, recreates and
//     re-dumps a database to prove the script restores it.
//   - SyncDBSchema records what that script reads and the old driver dropped: a
//     collection's create options, an index's options beyond its key and
//     uniqueness, and a view's definition. A time-series collection is synced
//     as a collection instead of being skipped; omni's tape pins that, since
//     the pinned harness server (4.4) predates time-series collections.
//   - SyncInstance names a user by its name where the old driver used the users
//     collection's _id, so an instance role named "admin.testuser" is now
//     "testuser".
//   - A missing database answers driver.ErrNotFound, whose message still holds
//     "database <name> does not exist".
//   - QueryConn answers one result per statement of a script, so a
//     multi-statement request runs instead of failing the client's
//     single-statement parse; a statement's failure is its own result's error
//     instead of the call's; and text the parser cannot split is one result
//     carrying the parser's error.
//   - A row limit no longer reaches countDocuments, where it changed what the
//     count counted, and MaximumSQLResultSize now caps a result, which the old
//     driver ignored.
//   - An explicit explain request reports the plan it produced on the result;
//     the rows are the same plan document.
//   - Every value that is not a document renders as {"value": ...}; the old
//     driver rendered an int32 or a double as its Go spelling and a null as
//     "<nil>".
//   - Execute reports the documents a write inserted, modified or deleted,
//     where the old driver reported none.
//   - An SSH data source now dials through its tunnel; the old driver ignored
//     the SSH fields and connected to the host directly.
//   - TLS material is read on each dial rather than once at Open, so invalid
//     material fails at the first connection instead of at Open.
//   - A connection parameter that names one of the driver's defaults
//     (authSource, appName) overrides it, where the old builder put both values
//     into the URI.
package mongodb

import (
	"context"
	"crypto/tls"
	"database/sql"
	"io"
	"net"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	omnimongo "github.com/bytebase/omni/mongo/driver"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_MONGODB, newDriver)
}

// Driver is Bytebase's MongoDB driver.
type Driver struct {
	inner     *omnimongo.Driver
	sshClient *ssh.Client
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	driverConfig, options, err := toDriverConfig(config)
	if err != nil {
		return nil, err
	}
	d := &Driver{}
	if dataSource := config.DataSource; dataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(dataSource)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient
		driverConfig.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			return util.DialSSHTunnel(ctx, sshClient, network, address)
		}
	}
	inner, err := omnimongo.Open(ctx, driverConfig, options)
	if err != nil {
		d.closeSSHClient()
		return nil, err
	}
	d.inner = inner
	return d, nil
}

// toDriverConfig maps a connection config onto the contract's config and
// options. The SSH Dial is Open's, because the tunnel needs the client Open
// would otherwise leave unclosed.
func toDriverConfig(config db.ConnectionConfig) (omnidriver.Config, omnimongo.Options, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return omnidriver.Config{}, omnimongo.Options{}, errors.New("mongodb: a data source is required")
	}
	driverConfig := omnidriver.Config{
		Host:     connectionHost(dataSource),
		Port:     dataSource.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		User:     dataSource.GetUsername(),
		Password: config.Password,
		Params:   connectionParams(dataSource),
	}
	if dataSource.GetUseSsl() {
		driverConfig.TLS = func(string) (*tls.Config, error) {
			return util.GetTLSConfig(dataSource)
		}
	}
	return driverConfig, omnimongo.Options{SRV: dataSource.GetSrv()}, nil
}

// connectionHost is the connection string's authority: the data source's host
// with its port, then every additional address with its own. Config.Port is
// appended to a single host that has none, so a host list carries each
// address's port itself.
func connectionHost(dataSource *storepb.DataSource) string {
	host := dataSource.GetHost()
	if port := dataSource.GetPort(); port != "" {
		host += ":" + port
	}
	for _, address := range dataSource.GetAdditionalAddresses() {
		host += "," + address.GetHost()
		if port := address.GetPort(); port != "" {
			host += ":" + port
		}
	}
	return host
}

// connectionParams maps the data source's connection settings onto connection
// string options. authSource defaults to admin in the driver, appName to
// "bytebase".
func connectionParams(dataSource *storepb.DataSource) map[string]string {
	params := map[string]string{}
	if authDatabase := dataSource.GetAuthenticationDatabase(); authDatabase != "" {
		params["authSource"] = authDatabase
	}
	if replicaSet := dataSource.GetReplicaSet(); replicaSet != "" {
		params["replicaSet"] = replicaSet
	}
	if dataSource.GetDirectConnection() {
		params["directConnection"] = "true"
	}
	for key, value := range dataSource.GetExtraConnectionParameters() {
		if key == "" {
			continue
		}
		params[key] = value
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

func (d *Driver) Close(context.Context) error {
	d.closeSSHClient()
	if d.inner == nil {
		return nil
	}
	return d.inner.Close()
}

func (d *Driver) closeSSHClient() {
	if d.sshClient != nil {
		_ = d.sshClient.Close()
		d.sshClient = nil
	}
}

func (d *Driver) Ping(ctx context.Context) error {
	return d.inner.Ping(ctx)
}

// GetDB returns no *sql.DB: the contract's session is not one.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute runs the script's statements in order, reporting each step to the
// task run log. Any transaction option is irrelevant: MongoDB runs one
// statement at a time here.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.inner.Execute(ctx, statement, omnidriver.ExecuteOptions{
		MaxRetries: opts.MaximumRetries,
		Log:        &executeLogger{opts: opts},
	})
}

// executeLogger bridges the contract's ExecuteLogger onto the task run log.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l *executeLogger) CommandExecute(command omnidriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{Start: int32(command.ByteStart), End: int32(command.ByteEnd)}, command.Text)
}

func (l *executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

// TransactionControl and Retry are never called: MongoDB has no transaction
// this path opens and the driver never retries a statement.
func (*executeLogger) TransactionControl(omnidriver.TransactionOp, error) {}

func (*executeLogger) Retry(error, int, int) {}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// QueryConn runs the script's statements on a session of its own. MongoDB has
// no session state, so nothing outlives the call, and conn is unused.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	session, err := d.inner.Session(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open a session")
	}
	defer func() { _ = session.Close() }()

	// MongoDB states a plan as one document and has no second format, so
	// queryContext.Option's explain format has nowhere to go.
	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		return nil, err
	}
	queryResults := make([]*v1pb.QueryResult, len(results))
	for i, result := range results {
		queryResults[i] = util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize)
	}
	return queryResults, nil
}

// SyncInstance reads the server version, its non-system databases and its
// users.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.inner.SyncInstance(ctx)
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

// SyncDBSchema reads the database's collections, their sizes and indexes, and
// its views.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.inner.SyncDatabase(ctx)
}

// Dump renders the synced database as a mongosh script.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.inner.Dump(ctx, out, dbMetadata)
}
