// Package mongodb is Bytebase's MongoDB driver: an adapter over
// github.com/bytebase/omni/mongo/driver. The connection string, the sync
// queries, the mongosh execution, the result rendering and the dump are the
// driver's; this package maps a db.ConnectionConfig onto the contract's Config
// and the contract's results back into the product's types.
//
// Bytebase registers this engine as MONGODB and dispatches through this
// package, while omni's tree is mongo/driver. Neither name changes: the engine
// value is the product's, and the package path is what the driver registry, the
// parser registration and the completion registration already use.
//
// What stays product-side is the path to the server and the task run's log:
// Open maps a data source onto driver.Config, including an SSH Dial, the TLS
// material and the connection string options; Execute reports each statement to
// the task run log; SyncInstance converts the contract's roles into the instance
// metadata.
//
// Every db.Driver method:
//
//   - Ping, Close, SyncInstance, SyncDBSchema and Dump delegate.
//   - GetDB returns nil: MongoDB has no database/sql handle.
//   - QueryConn opens one session per call; MongoDB keeps no session state.
//   - Execute runs each statement on its own. MongoDB has no transaction this
//     path takes part in, so TransactionMode and IsolationLevel are unset, and
//     no DDL, so a change script is mongosh; MaximumRetries reaches the contract
//     unused.
//   - ReadOnly, QueryOptions.Schema and QueryOptions.MaskingEnabled do nothing:
//     MongoDB has no read-only session, no schema per session and no masking to
//     withhold.
//
// Where the driver's answer differs from the driver this replaced is named case
// by case in the parity test's known-difference tables (cutover_parity_test.go).
package mongodb

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strings"

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
		driverConfig.TLS = func(host string) (*tls.Config, error) {
			return util.GetTLSConfigForHost(dataSource, host)
		}
	}
	return driverConfig, omnimongo.Options{SRV: dataSource.GetSrv()}, nil
}

// connectionHost is the connection string's authority: the data source's host
// with its port, then every additional address with its own. Config.Port is
// appended to a single host that has none, so a host list carries each
// address's port itself.
func connectionHost(dataSource *storepb.DataSource) string {
	host := formatHostPort(dataSource.GetHost(), dataSource.GetPort())
	for _, address := range dataSource.GetAdditionalAddresses() {
		host += "," + formatHostPort(address.GetHost(), address.GetPort())
	}
	return host
}

func formatHostPort(host, port string) string {
	bare := strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	if port != "" {
		return net.JoinHostPort(bare, port)
	}
	if strings.Count(bare, ":") >= 2 {
		return "[" + bare + "]"
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

// Execute runs the script's statements in order, reporting each step to the
// task run log. Any transaction option is irrelevant: MongoDB runs one
// statement at a time here.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.inner.Execute(ctx, statement, omnidriver.ExecuteOptions{
		MaxRetries: opts.MaximumRetries,
		Log:        util.NewExecuteLogger(opts),
	})
}

// QueryConn runs the script's statements on a session of its own. MongoDB has
// no session state, so nothing outlives the call.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
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
	return util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, false), nil
}

// SyncInstance reads the server version, its non-system databases and its
// users.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.inner.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return util.OmniInstanceToMetadata(instance, true), nil
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
