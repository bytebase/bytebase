package drivers

import (
	"context"
	"fmt"
	"io"
	"strings"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	pgdriver "github.com/bytebase/omni/pg/driver"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// pgSpec opens a driver that reaches the server only in tenant mode, where
// every pooled connection runs SET ROLE to the database owner.
var pgSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, err := pgConfig(config)
		if err != nil {
			return nil, err
		}
		switch config.DataSource.GetAuthenticationType() {
		case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
			// The connector names the instance itself, so the host and port stay
			// empty and Config.Dial reaches the instance.
			if _, err := util.SetupCloudSQLDial(ctx, config, &cfg, &d.OmniAdapter); err != nil {
				return nil, err
			}
			cfg.Host = ""
			cfg.Port = ""
		case storepb.DataSource_AWS_RDS_IAM:
			password, err := util.RDSAuthToken(ctx, config)
			if err != nil {
				return nil, err
			}
			cfg.Password = password
		default:
		}
		if err := util.SetupSSHDial(config.DataSource, &cfg, &d.OmniAdapter, true); err != nil {
			return nil, err
		}
		return pgdriver.Open(ctx, cfg, pgdriver.Options{TenantMode: config.ConnectionContext.TenantMode})
	},
	adapter: util.AdapterOptions{WithInstanceMetadata: true, NotFoundResource: "database"},
	execute: util.ExecuteConfig{
		ParseTransactionDirective:     true,
		SkipDirectiveOnCreateDatabase: true,
		PassCreateDatabase:            true,
		PassMaxRetries:                true,
	},
	query:          pgQuery,
	probe:          pgProbeOptions,
	taskRunSession: pgTaskRunSession,
	syncInstance:   pgSyncInstance,
	// A dump restores the user's database, so it leaves out the backup schema
	// the product keeps archived rows in.
	dump: func(ctx context.Context, d *Driver, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
		return d.OmniAdapter.Dump(ctx, out, withoutSchema(metadata, common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)))
	},
}

func pgConfig(config db.ConnectionConfig) (odriver.Config, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return odriver.Config{}, errors.New("a data source is required")
	}
	if dataSource.GetUsername() == "" {
		return odriver.Config{}, errors.New("user must be set")
	}
	if err := util.ValidateTLSCertKeyPair(dataSource); err != nil {
		return odriver.Config{}, err
	}
	// An empty Database leaves the contract's default, postgres; pgx alone
	// would pick the login role's name, which an administrative role with no
	// same-named database cannot open.
	cfg := odriver.Config{
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
	// A cloud connector provides the TLS itself, so pgx's default negotiation
	// stays in place for both IAM paths.
	if dataSource.GetAuthenticationType() != storepb.DataSource_GOOGLE_CLOUD_SQL_IAM &&
		dataSource.GetAuthenticationType() != storepb.DataSource_AWS_RDS_IAM {
		cfg.TLS = util.PGTLSFunc(dataSource)
	}
	return cfg, nil
}

func pgQuery(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
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

// postgresErrorDetail is the server's error fields for a failed statement. The
// result carries the error pgx returned, which is the server's *pgconn.PgError.
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

func pgSyncInstance(ctx context.Context, d *Driver) (*db.InstanceMetadata, error) {
	meta, err := d.OmniAdapter.SyncInstance(ctx)
	if err != nil || d.config.ConnectionContext.Vendor != storepb.Vendor_NEON {
		return meta, err
	}
	// Neon reserves "postgres" as a service-managed system database, whereas
	// other PostgreSQL instances commonly use it for application schemas.
	databases := make([]*metadatapb.DatabaseSchemaMetadata, 0, len(meta.Databases))
	for _, database := range meta.Databases {
		if database.Name != "postgres" {
			databases = append(databases, database)
		}
	}
	meta.Databases = databases
	return meta, nil
}

// withoutSchema returns a copy of metadata without the named schema.
func withoutSchema(metadata *metadatapb.DatabaseSchemaMetadata, name string) *metadatapb.DatabaseSchemaMetadata {
	if metadata == nil {
		return nil
	}
	filtered := proto.CloneOf(metadata)
	schemas := filtered.Schemas[:0]
	for _, schema := range filtered.Schemas {
		if schema.Name != name {
			schemas = append(schemas, schema)
		}
	}
	filtered.Schemas = schemas
	return filtered
}

// pgProbeOptions runs the transaction, the tenant-mode prelude, the
// pre-executions, the statement and the rollback on one session, so the owner
// the prelude switches to is the one the statement plans as.
func pgProbeOptions(d *Driver) util.OmniProbeOptions {
	var prepareTx func(context.Context, odriver.Session) ([]string, error)
	if d.config.ConnectionContext.TenantMode {
		prepareTx = func(ctx context.Context, session odriver.Session) ([]string, error) {
			owner, err := pgDatabaseOwner(ctx, session)
			if err != nil {
				return nil, err
			}
			// SET SESSION ROLE, as the executor's tenant-mode prelude does.
			return []string{fmt.Sprintf("SET SESSION ROLE '%s';", owner)}, nil
		}
	}
	return util.OmniProbeOptions{
		BeginStatement:           "BEGIN;",
		RollbackStatement:        "ROLLBACK;",
		PrepareTx:                prepareTx,
		CheckPreExecutionResults: true,
		RequireResult:            true,
		FormatCell:               pgProbeCell,
	}
}

func pgDatabaseOwner(ctx context.Context, session odriver.Session) (string, error) {
	const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
	results, err := session.Query(ctx, query, odriver.QueryOptions{})
	if err != nil {
		return "", err
	}
	if len(results) != 1 {
		return "", errors.New("failed to get database owner: no result")
	}
	if results[0].Err != nil {
		return "", results[0].Err
	}
	if len(results[0].Rows) != 1 || len(results[0].Rows[0]) != 1 || results[0].Rows[0][0].Kind != odriver.String {
		return "", errors.New("failed to get database owner: unexpected result shape")
	}
	return results[0].Rows[0][0].String, nil
}

// pgProbeCell converts one cell for a rule. A binary or temporal cell is text,
// as a rule reads it.
func pgProbeCell(value odriver.Value) any {
	switch value.Kind {
	case odriver.Null:
		return nil
	case odriver.Bool:
		return value.Bool
	case odriver.Int64, odriver.Int32, odriver.Uint64, odriver.Uint32:
		return value.Int64
	case odriver.Float64, odriver.Float32:
		return value.Float64
	case odriver.Bytes:
		return string(value.Bytes)
	case odriver.Timestamp, odriver.TimestampTZ:
		return pgdriver.FormatTimestamp(value.Time, value.Kind == odriver.TimestampTZ)
	default:
		return value.String
	}
}

// pgTaskRunSession reads the session that carries applicationName with the
// sessions it blocks and the sessions blocking it.
func pgTaskRunSession(ctx context.Context, d *Driver, applicationName string) (*v1pb.TaskRunSession, error) {
	impl, ok := d.Driver.(*pgdriver.Driver)
	if !ok {
		return nil, errors.Errorf("session monitoring needs the PostgreSQL driver, not %T", d.Driver)
	}
	activity, err := impl.Activity(ctx, applicationName)
	if err != nil {
		return nil, err
	}
	postgres := &v1pb.TaskRunSession_Postgres{}
	if activity.Session != nil {
		postgres.Session = pgSession(activity.Session)
	}
	for _, session := range activity.BlockedSessions {
		postgres.BlockedSessions = append(postgres.BlockedSessions, pgSession(session))
	}
	for _, session := range activity.BlockingSessions {
		postgres.BlockingSessions = append(postgres.BlockingSessions, pgSession(session))
	}
	return &v1pb.TaskRunSession{
		Session: &v1pb.TaskRunSession_Postgres_{Postgres: postgres},
	}, nil
}

func pgSession(session *pgdriver.BackendSession) *v1pb.TaskRunSession_Postgres_Session {
	converted := &v1pb.TaskRunSession_Postgres_Session{
		Pid:             session.PID,
		BlockedByPids:   session.BlockedByPIDs,
		Query:           session.Query,
		State:           session.State,
		WaitEventType:   session.WaitEventType,
		WaitEvent:       session.WaitEvent,
		Datname:         session.Database,
		Usename:         session.User,
		ApplicationName: session.ApplicationName,
		ClientAddr:      session.ClientAddr,
		ClientPort:      session.ClientPort,
		BackendStart:    timestamppb.New(session.BackendStart),
	}
	if session.XactStart != nil {
		converted.XactStart = timestamppb.New(*session.XactStart)
	}
	if session.QueryStart != nil {
		converted.QueryStart = timestamppb.New(*session.QueryStart)
	}
	return converted
}
