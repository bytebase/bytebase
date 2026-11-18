// Package pg is Bytebase's PostgreSQL driver: an adapter over
// github.com/bytebase/omni/pg/driver that maps a db.ConnectionConfig onto the
// driver contract and the contract's results back into the product's. The
// conversation with the server — connection configuration, session setup,
// metadata sync, planning, execution, result decoding and the dump — is the
// driver's.
//
// The adapter keeps what is product: the data source to contract conversion,
// the credential and network path (Cloud SQL IAM dialer, RDS IAM token, SSH
// tunnel, TLS material), QueryContext/ExecuteOptions mapping, the v1pb result
// and error shape, the task-run logger, the backup schema a dump omits, the
// advisor probe and the task-run session read.
//
// Constraints of the engine and the contract, here rather than at each call:
//   - Config.TokenProvider is unused: it is the contract's federated-token seam
//     for SQL Server, and pg/driver reads only Config.User and Config.Password.
//     An RDS IAM login is resolved here, once per Open, as before, and its extra
//     connection parameters now reach the session where the old RDS path
//     dropped them.
//   - QueryConn takes the caller's *sql.Conn but runs on the driver's own
//     session: the contract's Querier hands out Sessions, and a driver cannot
//     put one on a *sql.Conn. The driver holds that session from the first call
//     until Close and serializes the calls, as the stream's one pooled
//     connection was: what a statement sets there (a SET, a temporary table, the
//     search_path) holds for the next call.
//   - The probe wraps BEGIN, the tenant-mode prelude, the statement and
//     ROLLBACK on one session: the database owner is read on the connection the
//     probe runs on, not on a second pooled one.
//   - A cell the engine states as a timestamp outside google.protobuf.Timestamp
//     (PostgreSQL allows 4713 BC to 294276 AD) is written as the text the old
//     conversion wrote; the contract's value cannot carry it.
//   - A statement that reports no affected-row count (CREATE TABLE, ALTER TABLE,
//     SET) answers the editor's one "Affected Rows 0" row: the contract leaves
//     the count nil, and the result shape is the product's.
//
// Behaviours that differ from the driver this replaced, each fixed in
// pg/driver or made by the contract:
//   - Execute keeps the script's order in auto-commit mode, classifies SET
//     ROLE, non-transactional and superuser statements by AST, and splits a
//     CREATE DATABASE script into statements rather than lines.
//   - An empty BYTEA cell is empty bytes; it used to answer NULL.
//   - Tenant mode runs SET ROLE on every pooled connection rather than on
//     whichever one Open used.
//   - The sync stores what the server states and the old queries skipped:
//     sequence owners by attnum, generated-column storage kind, constraint
//     comments, collations, range types, foreign data wrappers and servers,
//     partition triggers from PostgreSQL 13, and a view's relation-level
//     dependencies.
//   - A range type's support functions are no longer stored as functions — the
//     type's own CREATE makes them — so a dump no longer writes their bare names.
//   - A foreign table carries its server and options, and no external database
//     name: the current database's name was stored there before, and no dump
//     read it.
//   - A dropped database answers driver.ErrNotFound, which a sync records as a
//     failure instead of storing an empty schema.
//   - The task-run session read classifies the blocked sessions after reading
//     every row, so a blocker with a lower PID is kept.
package pg

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"net"
	"sync"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	omndriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	pgdriver "github.com/bytebase/omni/pg/driver"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver               = (*Driver)(nil)
	_ db.Prober               = (*Driver)(nil)
	_ db.RowEstimator         = (*Driver)(nil)
	_ db.TaskRunSessionReader = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_POSTGRES, newDriver)
}

// Driver is Bytebase's PostgreSQL driver.
type Driver struct {
	impl *pgdriver.Driver

	sshClient *ssh.Client
	// cloudSQLDialer refreshes its ephemeral certificate in the background until
	// closed, so it lives and dies with the driver.
	cloudSQLDialer *cloudsqlconn.Dialer

	// sessionMu guards session and serializes QueryConn: the editor keeps one
	// driver for a whole stream, and one pooled connection served those
	// statements one at a time.
	sessionMu sync.Mutex
	session   omndriver.Session

	connectionCtx db.ConnectionContext
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open resolves the data source's credential and network path into the contract's
// Config and opens the driver. It reaches the server only in tenant mode, where
// every pooled connection runs SET ROLE to the database owner, as before.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// A driver whose Open fails is discarded without a Close, so Open releases
	// what it acquired itself.
	opened := false
	defer func() {
		if !opened {
			_ = d.Close(ctx)
		}
	}()

	cfg, err := pgConfig(config)
	if err != nil {
		return nil, err
	}
	switch config.DataSource.GetAuthenticationType() {
	case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		// The connector names the instance itself, so the contract's host and
		// port stay empty and Config.Dial reaches the instance.
		dialer, err := util.GetGCPConnectionConfig(ctx, config)
		if err != nil {
			return nil, errors.Wrap(err, "load gcp config failed")
		}
		d.cloudSQLDialer = dialer
		instance := config.DataSource.GetHost()
		cfg.Host = ""
		cfg.Port = ""
		cfg.Dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.Dial(ctx, instance)
		}
	case storepb.DataSource_AWS_RDS_IAM:
		password, err := getRDSConnectionPassword(ctx, config)
		if err != nil {
			return nil, err
		}
		cfg.Password = password
	default:
		// The data source's password authenticates, and Config.Password holds it.
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

	impl, err := pgdriver.Open(ctx, cfg, pgdriver.Options{TenantMode: config.ConnectionContext.TenantMode})
	if err != nil {
		return nil, err
	}
	d.impl = impl
	d.connectionCtx = config.ConnectionContext
	opened = true
	return d, nil
}

// pgConfig maps the data source onto the contract's Config, leaving the
// credential and dial path to Open.
func pgConfig(config db.ConnectionConfig) (omndriver.Config, error) {
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

	// An empty Database leaves the contract's own default, postgres. pgx alone
	// would resolve an omitted database to the login role's name, which an
	// administrative role with no same-named database cannot open.
	database := config.ConnectionContext.DatabaseName
	if database == "" {
		database = dataSource.GetDatabase()
	}
	applicationName := "bytebase"
	if config.ConnectionContext.TaskRunUID != nil {
		applicationName = fmt.Sprintf("bytebase-taskrun-%d", *config.ConnectionContext.TaskRunUID)
	}

	params := make(map[string]string, len(dataSource.GetExtraConnectionParameters()))
	for key, value := range dataSource.GetExtraConnectionParameters() {
		params[key] = value
	}

	cfg := omndriver.Config{
		Host:            dataSource.GetHost(),
		Port:            dataSource.GetPort(),
		Database:        database,
		User:            dataSource.GetUsername(),
		Password:        config.Password,
		Params:          params,
		ApplicationName: applicationName,
		ReadOnly:        config.ConnectionContext.ReadOnly,
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}
	// A cloud connector provides the TLS itself — the Cloud SQL connector wraps
	// the tunnel, and the RDS token is the credential — and the connections this
	// replaced left pgx's default negotiation in place for both.
	if dataSource.GetUseSsl() &&
		dataSource.GetAuthenticationType() != storepb.DataSource_GOOGLE_CLOUD_SQL_IAM &&
		dataSource.GetAuthenticationType() != storepb.DataSource_AWS_RDS_IAM {
		cfg.TLS = func(host string) (*tls.Config, error) {
			return util.GetPGTLSConfigForHost(dataSource, host)
		}
	}
	return cfg, nil
}

// getRDSConnectionPassword returns an AWS RDS IAM authentication token for the
// data source's login.
//
// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.Connecting.Go.html
func getRDSConnectionPassword(ctx context.Context, config db.ConnectionConfig) (string, error) {
	cfg, err := util.GetAWSConnectionConfig(ctx, config)
	if err != nil {
		return "", errors.Wrap(err, "load aws config failed")
	}

	// Handle cross-account role assumption if configured
	if err := util.AssumeRoleIfNeeded(ctx, &cfg, config.ConnectionContext, config.DataSource.GetAwsCredential()); err != nil {
		return "", err
	}

	dbEndpoint := fmt.Sprintf("%s:%s", config.DataSource.GetHost(), config.DataSource.GetPort())
	authenticationToken, err := auth.BuildAuthToken(
		ctx, dbEndpoint, config.DataSource.GetRegion(), config.DataSource.GetUsername(), cfg.Credentials)
	if err != nil {
		return "", errors.Wrap(err, "failed to create authentication token")
	}

	return authenticationToken, nil
}

// Close closes the driver and what Open acquired beside it.
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
	if d.cloudSQLDialer != nil {
		err = multierr.Append(err, d.cloudSQLDialer.Close())
	}
	return err
}

// Ping reaches the database, and fails before any statement when the server does
// not have it.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// SyncInstance reads the server version, its user databases and its roles.
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
	databases := make([]*metadatapb.DatabaseSchemaMetadata, 0, len(instance.Databases))
	for _, database := range instance.Databases {
		// Neon reserves "postgres" as a service-managed system database, whereas
		// other PostgreSQL instances commonly use it for application schemas.
		if d.connectionCtx.Vendor == storepb.Vendor_NEON && database.Name == "postgres" {
			continue
		}
		databases = append(databases, database)
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: databases,
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

// Dump writes the database's schema, without the backup schema the product keeps
// its archived rows in: a dump restores the user's database, not the archive.
func (d *Driver) Dump(ctx context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, withoutBackupSchema(metadata))
}

// withoutBackupSchema returns metadata without the backup schema, which the
// dump would otherwise recreate.
func withoutBackupSchema(metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	if metadata == nil {
		return nil
	}
	filtered := proto.CloneOf(metadata)
	backupSchema := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
	schemas := filtered.Schemas[:0]
	for _, schema := range filtered.Schemas {
		if schema.Name == backupSchema {
			continue
		}
		schemas = append(schemas, schema)
	}
	filtered.Schemas = schemas
	return filtered
}

// GetDB returns nil: the driver's session is pg/driver's pool, not a *sql.DB this
// adapter holds.
func (*Driver) GetDB() *sql.DB {
	return nil
}
