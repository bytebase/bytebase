// Package redshift is Bytebase's Redshift driver: an adapter over
// github.com/bytebase/omni/redshift/driver that maps a db.ConnectionConfig
// onto the driver contract and the contract's results back into the product's.
// The conversation with the server — connection configuration, session setup,
// metadata sync, statement rewriting and classification, execution, result
// decoding and the row estimate — is the driver's.
//
// The adapter keeps what is product: the data source to contract conversion,
// the network and TLS path (SSH tunnel, TLS material), the datashare flag,
// QueryContext/ExecuteOptions mapping, the v1pb result and error shape, and the
// task-run logger.
//
// Constraints of the engine and the contract, here rather than at each call:
//   - QueryConn takes the caller's *sql.Conn but runs on the driver's own
//     session: the contract's Querier hands out Sessions, and a driver cannot put
//     one on a *sql.Conn. The adapter holds that session from the first call
//     until Close and serializes the calls, so what a statement sets there (the
//     search_path, a temporary table) holds for the next one. It reports
//     OwnsQuerySession, so no caller opens a second connection for it.
//   - The data source's extra connection parameters reach no connection. The
//     driver this replaced read none of them; the only parameter the contract
//     takes here is ParamDatashare, which is the datashare flag.
//   - ApplicationName is not set: the driver this replaced set no
//     application_name, and the task-run session listing supports PostgreSQL and
//     CockroachDB only.
//   - TLS is required only when the data source carries a client certificate,
//     which is when the driver this replaced built a TLS configuration; a data
//     source with UseSsl alone leaves libpq's default negotiation in place.
//   - Dump writes nothing. Redshift's omni driver has no Dumper — redshift/catalog
//     has no renderer for a synced snapshot — and Bytebase's Dump returned nil, so
//     the no-op is kept rather than replaced by a renderer no server has checked.
//
// Behaviours that differ from the driver this replaced, each made by omni's
// driver and recorded below or in cutover_parity_test.go:
//   - The sync reads only user schemas: everything but information_schema whose
//     name starts with pg_. The old predicates excluded pg_catalog and
//     information_schema by name, so the sync failed for any non-superuser on
//     obj_description over Redshift's own pg_auto_copy/pg_internal schemas. It
//     also orders every read, reads a view's whole definition with SHOW VIEW
//     (pg_views truncates it at 256 bytes), reads a numeric column's precision
//     and scale, and keeps a datashare's views out of its table list.
//   - Execute splits every script with the engine's splitter. The old driver sent
//     a sheet longer than common.MaxSheetCheckSize, or one with more than
//     common.MaximumCommands statements, as a single command.
//   - Execute refuses a transactional script holding a statement Redshift commits
//     on its own (TRUNCATE, VACUUM, a database statement) or its own transaction
//     control, naming autocommit mode; the old driver ran it and let the
//     statement commit the transaction.
//   - A database-creation step runs one CREATE DATABASE statement and refuses a
//     script that holds initialization after it; the old driver ran the script's
//     lines one by one in the admin database.
//   - Autocommit mode runs every statement on one connection, so a script that
//     sets session state keeps it; the old driver used a pooled connection per
//     statement.
//   - A masked caller does not receive a procedure call's rows, which the
//     analysis cannot trace to columns; the old driver returned them.
//   - A datashare statement has the database's name stripped from the relations
//     it names, not from every occurrence in its text, so a literal that spells
//     the name keeps it.
//   - The task-run log records the total affected rows and no per-result list,
//     as before, though the contract reports both.
//   - A database missing from the server is driver.ErrNotFound, which the adapter
//     maps to the product's not-found error. The name it looks for is the one the
//     connection opened on, which falls back to the data source's database and
//     then to dev; the old driver compared only the connection context's name, so
//     a driver opened without one reported not-found for a database that exists.
package redshift

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"

	omndriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	redshiftdriver "github.com/bytebase/omni/redshift/driver"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver         = (*Driver)(nil)
	_ db.RowEstimator   = (*Driver)(nil)
	_ db.SessionQuerier = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_REDSHIFT, newDriver)
}

// Driver is Bytebase's Redshift driver.
type Driver struct {
	impl *redshiftdriver.Driver

	sshClient *ssh.Client

	// sessionMu guards session and serializes QueryConn: the editor keeps one
	// driver for a whole stream, and one connection served those statements one
	// at a time.
	sessionMu sync.Mutex
	session   omndriver.Session

	connectionCtx db.ConnectionContext
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open resolves the data source's network and TLS path into the contract's
// Config and opens the driver. It reaches the server only on Ping.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// A driver whose Open fails is discarded without a Close, so Open releases
	// what it acquired itself.
	opened := false
	defer func() {
		if !opened {
			_ = d.Close(ctx)
		}
	}()

	cfg, err := redshiftConfig(config)
	if err != nil {
		return nil, err
	}
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

	impl, err := redshiftdriver.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	d.impl = impl
	d.connectionCtx = config.ConnectionContext
	opened = true
	return d, nil
}

// redshiftConfig maps the data source onto the contract's Config, leaving the
// network and TLS path to Open.
func redshiftConfig(config db.ConnectionConfig) (omndriver.Config, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return omndriver.Config{}, errors.New("a data source is required")
	}
	if dataSource.GetUsername() == "" {
		return omndriver.Config{}, errors.New("user must be set")
	}
	if (dataSource.GetSslCert() == "" && dataSource.GetSslKey() != "") ||
		(dataSource.GetSslCert() != "" && dataSource.GetSslKey() == "") {
		return omndriver.Config{}, errors.New("ssl-cert and ssl-key must be both set or unset")
	}

	// An empty Database leaves the contract's own default, dev.
	database := config.ConnectionContext.DatabaseName
	if database == "" {
		database = dataSource.GetDatabase()
	}

	cfg := omndriver.Config{
		Host:          dataSource.GetHost(),
		Port:          dataSource.GetPort(),
		Database:      database,
		User:          dataSource.GetUsername(),
		Password:      config.Password,
		ReadOnly:      config.ConnectionContext.ReadOnly,
		EngineVersion: config.ConnectionContext.EngineVersion,
	}
	if config.ConnectionContext.DataShare {
		cfg.Params = map[string]string{redshiftdriver.ParamDatashare: "true"}
	}
	if dataSource.GetSslCert() != "" {
		tlsConfig, err := util.GetTLSConfig(dataSource)
		if err != nil {
			return omndriver.Config{}, err
		}
		if tlsConfig != nil {
			cfg.TLS = func(string) (*tls.Config, error) { return tlsConfig, nil }
		}
	}
	return cfg, nil
}

// Close closes the driver's session and what Open acquired beside it.
func (d *Driver) Close(context.Context) error {
	d.sessionMu.Lock()
	if d.session != nil {
		_ = d.session.Close()
		d.session = nil
	}
	d.sessionMu.Unlock()

	var err error
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

// OwnsQuerySession reports that QueryConn runs on the adapter's held session,
// not on a connection a caller opens for it.
func (*Driver) OwnsQuerySession() {}

// SyncInstance reads the server version, its user databases and its users.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]*storepb.InstanceRole, 0, len(instance.Roles))
	for _, role := range instance.Roles {
		roles = append(roles, &storepb.InstanceRole{
			Name:      role.Name,
			Attribute: role.Attribute,
		})
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
		Metadata:  &storepb.Instance{Roles: roles},
	}, nil
}

// SyncDBSchema reads the schema of the database the driver is open on.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	metadata, err := d.impl.SyncDatabase(ctx)
	if err != nil {
		if errors.Is(err, omndriver.ErrNotFound) {
			return nil, common.Errorf(common.NotFound, "database %q not found", d.connectionCtx.DatabaseName)
		}
		return nil, err
	}
	return metadata, nil
}

// Dump writes nothing: the contract has no renderer for a Redshift snapshot, and
// the driver this replaced wrote nothing either.
func (*Driver) Dump(context.Context, io.Writer, *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}

// CountAffectedRows returns the planner's estimate of the rows the statement
// modifies, which plancheck's affected-row limit reads.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.impl.CountAffectedRows(ctx, statement)
}
