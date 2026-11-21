// Package starrocks is Bytebase's StarRocks driver: an adapter over omni's
// starrocks/driver that maps Bytebase's connection config onto the driver
// contract and the contract's results back into the product's protos. The
// conversation with the server — sync queries, statement execution, result
// decoding, the server-stated dump — is the driver's; what stays here is the
// path to it (SSH tunnel, TLS material) and the product's types.
//
// GetDB returns nil: the driver owns its pool and hands out sessions, so a
// caller that needs a *sql.DB is told the engine exposes none. gh-ost's
// prerequisite probe is the one caller that asks; it refuses an engine outside
// ghost.SupportsEngine and never reaches this driver's nil pool.
//
// QueryConn opens its session once and holds it until Close, because the
// editor asks for one statement per call and a session variable one call sets
// must reach the next. The calls are serialized for the same reason one
// database connection serializes them.
//
// The measurements behind this adapter, and the behavior of the driver it
// replaced, are in #22084.
package starrocks

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	starrocksdriver "github.com/bytebase/omni/starrocks/driver"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_STARROCKS, newDriver)
}

// Driver is Bytebase's StarRocks driver.
type Driver struct {
	config db.ConnectionConfig

	impl      *starrocksdriver.Driver
	sshClient *ssh.Client

	// mu guards session, and holds a caller's QueryConn for its duration the
	// way one database connection does: a statement and the next see the same
	// session, not two interleaved ones.
	mu      sync.Mutex
	session omnidriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server: Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, sshClient, err := driverConfig(config)
	if err != nil {
		return nil, err
	}
	impl, err := starrocksdriver.Open(ctx, cfg)
	if err != nil {
		if sshClient != nil {
			_ = sshClient.Close()
		}
		return nil, err
	}
	d.config = config
	d.impl = impl
	d.sshClient = sshClient
	return d, nil
}

// driverConfig resolves everything Bytebase knows about reaching the server into
// the plain values the driver takes. The mapping is the MySQL family's: the same
// host/port or unix socket, the same tunnel as a Dial, and the same TLS material
// the old DSN registered with go-sql-driver.
func driverConfig(config db.ConnectionConfig) (omnidriver.Config, *ssh.Client, error) {
	ds := config.DataSource
	cfg := omnidriver.Config{
		Host:     ds.GetHost(),
		Port:     ds.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		User:     ds.GetUsername(),
		Password: config.Password,
		// The driver prefixes a statement it reads as a query with
		// "/*app=<name>*/", which is how Bytebase names its editor sessions.
		ApplicationName: "bytebase",
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}

	var sshClient *ssh.Client
	if ds.GetSshHost() != "" {
		client, err := util.GetSSHClient(ds)
		if err != nil {
			return cfg, nil, err
		}
		sshClient = client
		cfg.Dial = func(_ context.Context, _, addr string) (net.Conn, error) {
			return client.Dial("tcp", addr)
		}
	}

	tlscfg, err := util.GetTLSConfig(ds)
	if err != nil {
		if sshClient != nil {
			_ = sshClient.Close()
		}
		return cfg, nil, errors.Wrap(err, "sql: tls config error")
	}
	if tlscfg != nil {
		// The material is resolved for this data source already: the host the
		// driver dials is the one the verification was built for, so the same
		// config serves every dial.
		cfg.TLS = func(string) (*tls.Config, error) {
			return tlscfg, nil
		}
	}
	return cfg, sshClient, nil
}

// Close closes the driver.
func (d *Driver) Close(context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	var err error
	if d.session != nil {
		err = multierr.Append(err, d.session.Close())
		d.session = nil
	}
	err = multierr.Append(err, d.impl.Close())
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
	}
	return err
}

// Ping pings the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// SyncInstance syncs the instance: its release, lower_case_table_names, and its
// user databases.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return util.OmniInstanceToMetadata(instance, true), nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	metadata, err := d.impl.SyncDatabase(ctx)
	if err != nil {
		return nil, syncDatabaseError(err, d.config.ConnectionContext.DatabaseName)
	}
	return metadata, nil
}

// syncDatabaseError reports a database the server does not have with the
// product's not-found error. Any other failure stays as the driver reported it.
func syncDatabaseError(err error, database string) error {
	if errors.Is(err, omnidriver.ErrNotFound) {
		return common.Errorf(common.NotFound, "database %q not found", database)
	}
	return err
}

// Dump dumps the database's schema as its server states it. The synced metadata
// does not narrow the dump: a StarRocks table's key model, distribution,
// properties and partitions are not in information_schema, so the server's own
// SHOW CREATE statements are the only complete definition.
func (d *Driver) Dump(ctx context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, metadata)
}
