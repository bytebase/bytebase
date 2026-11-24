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
//   - QueryConn runs on the driver's own session, held from the first call
//     until Close with calls serialized, as the stream's one pooled connection
//     was: what a statement sets there (a SET, a temporary table, the
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
	"io"
	"strings"

	"cloud.google.com/go/cloudsqlconn"
	omndriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	pgdriver "github.com/bytebase/omni/pg/driver"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
	util.OmniAdapter
	impl *pgdriver.Driver

	// cloudSQLDialer refreshes its ephemeral certificate in the background until
	// closed, so it lives and dies with the driver.
	cloudSQLDialer *cloudsqlconn.Dialer

	connectionCtx db.ConnectionContext
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open resolves the data source's credential and network path into the contract's
// Config and opens the driver. It reaches the server only in tenant mode, where
// every pooled connection runs SET ROLE to the database owner, as before.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if err := d.OpenWith(ctx, util.AdapterOptions{
		WithInstanceMetadata: true,
		NotFoundResource:     "database",
		DatabaseName:         config.ConnectionContext.DatabaseName,
	}, func(adapter *util.OmniAdapter) (omndriver.Driver, error) {
		cfg, err := pgConfig(config)
		if err != nil {
			return nil, err
		}
		switch config.DataSource.GetAuthenticationType() {
		case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
			// The connector names the instance itself, so the contract's host and
			// port stay empty and Config.Dial reaches the instance.
			dialer, err := util.SetupCloudSQLDial(ctx, config, &cfg, adapter)
			if err != nil {
				return nil, err
			}
			d.cloudSQLDialer = dialer
			cfg.Host = ""
			cfg.Port = ""
		case storepb.DataSource_AWS_RDS_IAM:
			password, err := util.RDSAuthToken(ctx, config)
			if err != nil {
				return nil, err
			}
			cfg.Password = password
		default:
			// The data source's password authenticates, and Config.Password holds it.
		}

		if err := util.SetupSSHDial(config.DataSource, &cfg, adapter, true); err != nil {
			return nil, err
		}

		impl, err := pgdriver.Open(ctx, cfg, pgdriver.Options{TenantMode: config.ConnectionContext.TenantMode})
		if err != nil {
			return nil, err
		}
		d.impl = impl
		d.connectionCtx = config.ConnectionContext
		return impl, nil
	}); err != nil {
		return nil, err
	}
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
	if err := util.ValidateTLSCertKeyPair(dataSource); err != nil {
		return omndriver.Config{}, err
	}

	// An empty Database leaves the contract's own default, postgres. pgx alone
	// would resolve an omitted database to the login role's name, which an
	// administrative role with no same-named database cannot open.
	cfg := omndriver.Config{
		Host:            dataSource.GetHost(),
		Port:            dataSource.GetPort(),
		Database:        util.ResolveDatabaseName(config),
		User:            dataSource.GetUsername(),
		Password:        config.Password,
		Params:          util.CloneParams(dataSource.GetExtraConnectionParameters()),
		ApplicationName: util.ApplicationName(config.ConnectionContext),
		ReadOnly:        config.ConnectionContext.ReadOnly,
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}
	// A cloud connector provides the TLS itself — the Cloud SQL connector wraps
	// the tunnel, and the RDS token is the credential — and the connections this
	// replaced left pgx's default negotiation in place for both.
	if dataSource.GetAuthenticationType() != storepb.DataSource_GOOGLE_CLOUD_SQL_IAM &&
		dataSource.GetAuthenticationType() != storepb.DataSource_AWS_RDS_IAM {
		cfg.TLS = util.PGTLSFunc(dataSource)
	}
	return cfg, nil
}

// SyncInstance reads the server version, its user databases and its roles.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	meta, err := d.OmniAdapter.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	if d.connectionCtx.Vendor == storepb.Vendor_NEON {
		databases := make([]*metadatapb.DatabaseSchemaMetadata, 0, len(meta.Databases))
		for _, database := range meta.Databases {
			// Neon reserves "postgres" as a service-managed system database, whereas
			// other PostgreSQL instances commonly use it for application schemas.
			if database.Name == "postgres" {
				continue
			}
			databases = append(databases, database)
		}
		meta.Databases = databases
	}
	return meta, nil
}

// Dump writes the database's schema, without the backup schema the product keeps
// its archived rows in: a dump restores the user's database, not the archive.
func (d *Driver) Dump(ctx context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.OmniAdapter.Dump(ctx, out, withoutBackupSchema(metadata))
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

// Execute runs the script, reporting each step to the task run log. The
// transaction directive the script states decides whether its statements share
// one transaction or commit on their own; without one they share it.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective:     true,
		SkipDirectiveOnCreateDatabase: true,
		PassCreateDatabase:            true,
		PassMaxRetries:                true,
	})
}

// QueryConn runs the script on the driver's session, one result per statement.
// Calls are serialized, as the stream's one pooled connection was.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	explainFormat := strings.ToLower(util.OmniExplainFormat(
		queryContext.Option.GetExplainFormat(),
		"",
		v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON, v1pb.QueryOption_XML, v1pb.QueryOption_YAML,
	))
	results, err := d.QueryHeldSession(ctx, statement, util.OmniQueryOptions(queryContext, true, explainFormat), util.SessionQueryOptions{})
	if err != nil {
		return nil, err
	}

	converted := util.OmniResultsToQueryResultsWith(results, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
		ClearExplainPlan:          queryContext.Explain,
		FillZeroAffectedRows:      true,
		FormatOutOfRangeTimestamp: pgdriver.FormatTimestamp,
	})
	for i, result := range results {
		if result.Err != nil {
			converted[i].DetailedError = postgresErrorDetail(result.Err)
		}
	}
	return converted, nil
}

// postgresErrorDetail is the server's error fields for a failed statement, for a
// client that shows them beside the message. The contract's result carries the
// error pgx returned, which is the server's *pgconn.PgError.
func postgresErrorDetail(err error) *v1pb.QueryResult_PostgresError_ {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	return &v1pb.QueryResult_PostgresError_{
		PostgresError: &v1pb.QueryResult_PostgresError{
			Severity:         pgErr.Severity,
			Code:             pgErr.Code,
			Message:          pgErr.Message,
			Detail:           pgErr.Detail,
			Hint:             pgErr.Hint,
			Position:         pgErr.Position,
			InternalPosition: pgErr.InternalPosition,
			InternalQuery:    pgErr.InternalQuery,
			Where:            pgErr.Where,
			SchemaName:       pgErr.SchemaName,
			TableName:        pgErr.TableName,
			ColumnName:       pgErr.ColumnName,
			DataTypeName:     pgErr.DataTypeName,
			ConstraintName:   pgErr.ConstraintName,
			File:             pgErr.File,
			Line:             pgErr.Line,
			Routine:          pgErr.Routine,
		},
	}
}
