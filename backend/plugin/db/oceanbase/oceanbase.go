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
//   - The editor's *sql.Conn is not the contract's session, so the adapter
//     holds one session from the first QueryConn call until Close, under a
//     mutex: a request's statements share a session.
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
	"strings"
	"time"

	"github.com/pkg/errors"

	odriver "github.com/bytebase/omni/driver"
	oceanbasedriver "github.com/bytebase/omni/oceanbase/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver            = (*Driver)(nil)
	_ db.Prober            = (*Driver)(nil)
	_ db.RowEstimator      = (*Driver)(nil)
	_ db.ColumnExtraLookup = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_OCEANBASE, newDriver)
}

// applicationName is the /*app=…*/ comment every query and change-script
// statement carries, as the DSN it replaced carried it.
const applicationName = "bytebase"

// Driver is Bytebase's OceanBase driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server; Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if err := d.OpenWith(ctx, util.AdapterOptions{WithInstanceMetadata: true}, func(*util.OmniAdapter) (odriver.Driver, error) {
		cfg, err := driverConfig(ctx, config, d)
		if err != nil {
			return nil, err
		}
		return oceanbasedriver.Open(ctx, cfg)
	}); err != nil {
		return nil, err
	}
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
		if _, err := util.SetupCloudSQLDial(ctx, config, &cfg, &d.OmniAdapter); err != nil {
			return odriver.Config{}, err
		}
		cfg.Params = map[string]string{"parseTime": "true"}
	case storepb.DataSource_AWS_RDS_IAM:
		if err := util.ConfigureRDSMySQLIAM(ctx, config, &cfg); err != nil {
			return odriver.Config{}, err
		}
	default:
		params, err := connectionParams(ds.GetExtraConnectionParameters())
		if err != nil {
			return odriver.Config{}, err
		}
		cfg.Params = params
		if err := util.SetupSSHDial(ds, &cfg, &d.OmniAdapter, false); err != nil {
			return odriver.Config{}, err
		}
		cfg.TLS = util.LazyTLSFunc(ds)
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

// Execute runs a change script under the transaction directive it carries.
// CreateDatabase and MaximumRetries are not mapped: the driver this replaced
// read neither.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassIsolation:             true,
	})
}

// QueryConn runs a script for the editor on the driver's session, one result
// per statement, and stops at the first statement that fails.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(
		ctx,
		statement,
		util.OmniQueryOptions(queryContext, false, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_JSON)),
		util.SessionQueryOptions{WrapOpenError: "failed to open a session"},
		util.QueryResultOptions{ClearExplainPlan: queryContext.Explain},
	)
}

// Probe runs a live review rule's probe: the pre-executions and the statement
// in one transaction on one session, rolled back afterwards, as the *sql.DB
// handle it replaced did. ProbeContext.Autocommit runs the statement alone,
// which is what a rule whose statement OceanBase refuses inside a transaction
// asks for.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return d.ProbeOmni(ctx, pCtx, statement, util.OmniProbeOptions{
		BeginStatement:           "START TRANSACTION",
		RollbackStatement:        "ROLLBACK",
		RollbackWithoutCancel:    true,
		RollbackTimeout:          5 * time.Second,
		SkipEmptyPreExecutions:   true,
		CheckPreExecutionResults: true,
		FormatCell:               probeCell,
	})
}

func probeCell(value odriver.Value) any {
	switch value.Kind {
	case odriver.Bytes:
		return string(value.Bytes)
	case odriver.Int32:
		return int32(value.Int64)
	case odriver.Uint32:
		return uint32(value.Uint64)
	case odriver.Structured:
		return nil
	case odriver.Timestamp, odriver.TimestampTZ:
		return value.Time.Format(time.RFC3339Nano)
	default:
		return util.DefaultProbeCell(value)
	}
}

// CountAffectedRows returns OceanBase's planner estimate of the rows the
// INSERT, UPDATE or DELETE statement modifies, read from EXPLAIN FORMAT=JSON.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.Driver.(db.RowEstimator).CountAffectedRows(ctx, statement)
}

// ColumnExtra returns one column's INFORMATION_SCHEMA.COLUMNS.EXTRA, the
// lookup OceanBase's offline-DDL rule binds.
func (d *Driver) ColumnExtra(ctx context.Context, database, table, column string) (string, error) {
	return d.Driver.(db.ColumnExtraLookup).ColumnExtra(ctx, database, table, column)
}
