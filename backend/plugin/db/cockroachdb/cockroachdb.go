// Package cockroachdb is Bytebase's CockroachDB driver: an adapter over
// github.com/bytebase/omni/cockroachdb/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's. The conversation with the server — connection configuration,
// session setup, metadata sync, statement classification, execution, result
// decoding and the plan diagram — is the driver's.
//
// CockroachDB is a PostgreSQL fork in Bytebase and its own tree in omni, so the
// adapter follows pg's shape where the two overlap: Config carries libpq's host
// and port lists, a TLS func bound to each host the driver dials, and the SSH
// tunnel as Config.Dial; QueryOptions.Schema is the session's search_path. Two
// parts of the pg path are not copied: Config.Params stays unset, because this
// data source has no extra connection parameters, and there is no tenant-mode
// prelude, because tenant mode does not apply to CockroachDB and its Open takes
// no engine options.
//
// QueryConn runs on one session held for the driver's lifetime and serializes
// its calls, as the editor stream's one pooled connection did: what a statement
// sets there (search_path, a temporary table) holds for the stream's next call.
// The caller's *sql.Conn is unused, so the adapter reports OwnsQuerySession.
//
// GetDB, through db.SQLDBProvider, returns the pool the task-run session listing
// asks for. It is not the query path, which is the driver's session; it is
// opened on first use, so a driver that never asks holds only the driver's pool.
//
// Execute maps the script's transaction directive and passes neither the
// isolation directive nor MaximumRetries: this engine ignored both, and honoring
// them would change what a script does.
package cockroachdb

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	omnicockroach "github.com/bytebase/omni/cockroachdb/driver"
	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver         = (*Driver)(nil)
	_ db.SQLDBProvider  = (*Driver)(nil)
	_ db.SessionQuerier = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_COCKROACHDB, newDriver)
}

// Driver is Bytebase's CockroachDB driver.
type Driver struct {
	inner         *omnicockroach.Driver
	config        omnidriver.Config
	connectionCtx db.ConnectionContext

	// sshClient is the tunnel the driver dials through, closed with it.
	sshClient *ssh.Client

	// sessionMu serializes QueryConn and guards session: the editor keeps one
	// driver for a stream, and one pooled connection served its statements one
	// at a time.
	sessionMu sync.Mutex
	session   omnidriver.Session

	// poolMu guards the pool GetDB opens on first use.
	poolMu  sync.Mutex
	pool    *sql.DB
	poolSet bool
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server: Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, err := cockroachConfig(config)
	if err != nil {
		return nil, err
	}

	if config.DataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(config.DataSource)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient
		cfg.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return util.DialSSHTunnel(ctx, sshClient, network, addr)
		}
	}

	inner, err := omnicockroach.Open(ctx, cfg)
	if err != nil {
		if d.sshClient != nil {
			_ = d.sshClient.Close()
			d.sshClient = nil
		}
		return nil, err
	}
	d.inner = inner
	d.config = cfg
	d.connectionCtx = config.ConnectionContext
	return d, nil
}

// cockroachConfig resolves what Bytebase knows about reaching a server into the
// plain values the contract's CockroachDB driver takes.
func cockroachConfig(config db.ConnectionConfig) (omnidriver.Config, error) {
	dataSource := config.DataSource
	// The old driver refused half a client certificate whether or not TLS was
	// asked for, so the mistake is reported here rather than left to a
	// connection that never presents it.
	if (dataSource.GetSslCert() == "") != (dataSource.GetSslKey() == "") {
		return omnidriver.Config{}, errors.New("ssl-cert and ssl-key must be both set or unset")
	}
	database := config.ConnectionContext.DatabaseName
	if database == "" {
		database = dataSource.GetDatabase()
	}
	cfg := omnidriver.Config{
		Host:            dataSource.GetHost(),
		Port:            dataSource.GetPort(),
		Database:        database,
		User:            dataSource.GetUsername(),
		Password:        config.Password,
		ApplicationName: applicationName(config.ConnectionContext),
		ReadOnly:        config.ConnectionContext.ReadOnly,
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}
	if dataSource.GetUseSsl() {
		// util.GetTLSConfig binds certificate verification to the data source's
		// host; ApplyPGTLSConfig rebinds it to the host of whichever server the
		// driver dials, so a multi-host list verifies each host's certificate
		// against its own name, as the old driver did.
		cfg.TLS = func(host string) (*tls.Config, error) {
			tlsConfig, err := util.GetTLSConfig(dataSource)
			if err != nil {
				return nil, err
			}
			if tlsConfig != nil {
				util.ApplyPGTLSConfig(tlsConfig, host, nil)
			}
			return tlsConfig, nil
		}
	}
	return cfg, nil
}

// applicationName identifies the connection to the server. A task run names
// itself so the server's activity views can find it.
func applicationName(connectionContext db.ConnectionContext) string {
	if connectionContext.TaskRunUID != nil {
		return fmt.Sprintf("bytebase-taskrun-%d", *connectionContext.TaskRunUID)
	}
	return "bytebase"
}

func (d *Driver) Close(context.Context) error {
	d.sessionMu.Lock()
	session := d.session
	d.session = nil
	d.sessionMu.Unlock()

	d.poolMu.Lock()
	pool := d.pool
	d.pool = nil
	d.poolMu.Unlock()

	var err error
	if session != nil {
		err = multierr.Append(err, session.Close())
	}
	if pool != nil {
		err = multierr.Append(err, pool.Close())
	}
	if d.inner != nil {
		err = multierr.Append(err, d.inner.Close())
	}
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
		d.sshClient = nil
	}
	return err
}

// Ping reaches the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.inner.Ping(ctx)
}

// GetDB returns the pool db.SQLDB callers ask for: the task-run session listing
// reads a *sql.DB and has no driver method for CockroachDB's activity. It is not
// the query path. A mapping error answers nil, which SQLDB callers tolerate.
func (d *Driver) GetDB() *sql.DB {
	d.poolMu.Lock()
	defer d.poolMu.Unlock()
	if !d.poolSet {
		d.poolSet = true
		pool, err := sqlPool(d.config)
		if err != nil {
			return nil
		}
		d.pool = pool
	}
	return d.pool
}

// OwnsQuerySession reports that QueryConn runs on the adapter's held session,
// not on a connection a caller opens from GetDB.
func (*Driver) OwnsQuerySession() {}

// sqlPool opens the pool GetDB returns from the contract's Config. It is a
// second pool beside the driver's, and only db.SQLDB callers use it.
func sqlPool(cfg omnidriver.Config) (*sql.DB, error) {
	dsn := make([]string, 0, 4+len(cfg.Params))
	if cfg.Host != "" || cfg.Port != "" {
		hostPort, err := util.PGHostPort(cfg.Host, cfg.Port)
		if err != nil {
			return nil, err
		}
		dsn = append(dsn, hostPort)
	}
	if cfg.TLS != nil {
		dsn = append(dsn, "sslmode=require")
	}
	if routingID := getRoutingIDFromCockroachCloudURL(cfg.Host); routingID != "" {
		dsn = append(dsn, "options="+util.QuotePGValue("--cluster="+routingID))
	}
	for key, value := range cfg.Params {
		dsn = append(dsn, key+"="+value)
	}
	connConfig, err := pgx.ParseConfig(strings.Join(dsn, " "))
	if err != nil {
		return nil, err
	}
	connConfig.User = cfg.User
	connConfig.Password = cfg.Password
	connConfig.Database = cfg.Database
	if connConfig.Database == "" {
		connConfig.Database = "postgres"
	}
	if cfg.TLS != nil {
		if connConfig.TLSConfig, err = cfg.TLS(connConfig.Host); err != nil {
			return nil, err
		}
		for _, fallback := range connConfig.Fallbacks {
			if fallback.TLSConfig, err = cfg.TLS(fallback.Host); err != nil {
				return nil, err
			}
		}
	}
	if cfg.Dial != nil {
		connConfig.DialFunc = cfg.Dial
	}
	if cfg.ApplicationName != "" {
		connConfig.RuntimeParams["application_name"] = cfg.ApplicationName
	}
	if cfg.ReadOnly {
		connConfig.RuntimeParams["default_transaction_read_only"] = "true"
	}
	return stdlib.OpenDB(*connConfig), nil
}

// getRoutingIDFromCockroachCloudURL returns the routing ID a Cockroach Cloud
// host names in its first label, or the empty string when the host is not one.
// The server is reached through a shared gateway until the ID is passed as
// --cluster, so the pool GetDB opens needs it as much as the driver does.
func getRoutingIDFromCockroachCloudURL(host string) string {
	host = strings.TrimSpace(host)
	if !strings.HasSuffix(host, "cockroachlabs.cloud") {
		return ""
	}
	parts := strings.Split(host, ".")
	// routing-id[.xxx].cockroachlabs.cloud
	if len(parts) > 2 {
		return parts[0]
	}
	return ""
}

// SyncInstance reads the server version, its user databases and its roles.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.inner.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]*storepb.InstanceRole, 0, len(instance.Roles))
	for _, role := range instance.Roles {
		roles = append(roles, &storepb.InstanceRole{Name: role.Name, Attribute: role.Attribute})
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
		Metadata:  &storepb.Instance{Roles: roles},
	}, nil
}

// SyncDBSchema reads the schema of the database the driver was opened on. A
// database the server no longer has is reported as not found.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	metadata, err := d.inner.SyncDatabase(ctx)
	if err != nil {
		if errors.Is(err, omnidriver.ErrNotFound) {
			return nil, common.Errorf(common.NotFound, "database %q not found", d.connectionCtx.DatabaseName)
		}
		return nil, err
	}
	return metadata, nil
}

// Dump writes the server's own DDL for the database. The synced metadata is not
// rendered and does not narrow what the server states.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.inner.Dump(ctx, out, dbMetadata)
}

// CountAffectedRows returns the planner's estimate of the rows a statement
// modifies. plancheck reads it through db.RowEstimator.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.inner.CountAffectedRows(ctx, statement)
}
