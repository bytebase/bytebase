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
// QueryConn holds one contract session from the first call until Close. The
// editor asks one statement at a time on a driver it keeps, so a session
// variable, a temporary table or an open transaction survives between calls,
// as it did on the caller's pooled connection.
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

	"github.com/pkg/errors"

	odriver "github.com/bytebase/omni/driver"
	mariadbast "github.com/bytebase/omni/mariadb/ast"
	mariadbd "github.com/bytebase/omni/mariadb/driver"
	mysqlast "github.com/bytebase/omni/mysql/ast"
	mysqld "github.com/bytebase/omni/mysql/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver       = (*Driver)(nil)
	_ db.Prober       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_MYSQL, newDriver)
	db.Register(storepb.Engine_MARIADB, newDriver)
}

// Driver is Bytebase's MySQL driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the resolved data source onto the contract's Config.
func (d *Driver) Open(ctx context.Context, dbType storepb.Engine, connCfg db.ConnectionConfig) (db.Driver, error) {
	dataSource := connCfg.DataSource
	if dataSource == nil {
		return nil, errors.New("mysql: a data source is required")
	}
	if err := d.OpenWith(ctx, util.AdapterOptions{WithInstanceMetadata: true}, func(adapter *util.OmniAdapter) (odriver.Driver, error) {
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
			if _, err := util.SetupCloudSQLDial(ctx, connCfg, &cfg, adapter); err != nil {
				return nil, err
			}
		case storepb.DataSource_AWS_RDS_IAM:
			if err := util.ConfigureRDSMySQLIAM(ctx, connCfg, &cfg); err != nil {
				return nil, err
			}
		default:
			if err := util.SetupSSHDial(dataSource, &cfg, adapter, false); err != nil {
				return nil, err
			}
			tlsFn, err := util.SQLStaticTLSFunc(dataSource)
			if err != nil {
				return nil, err
			}
			cfg.TLS = tlsFn
		}

		switch dbType {
		case storepb.Engine_MYSQL:
			return mysqld.Open(ctx, cfg)
		case storepb.Engine_MARIADB:
			return mariadbd.Open(ctx, cfg)
		default:
			return nil, errors.Errorf("mysql: unsupported engine %s", dbType)
		}
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// Execute runs the script's statements, reporting each step to the task run
// log. The transaction directive is read here unless the caller already did.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// CreateDatabase and MaximumRetries are not mapped: the driver this
	// replaced read neither, and mapping them would change a create-database
	// script's transaction shape and retry a script the product did not retry.
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassIsolation:             true,
	})
}

// QueryConn runs the script on the driver's session. The adapter holds its own
// session from the first call until Close, so a stream's session state survives
// between calls.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	// The product reads a zero cap as "no rows", and the contract reads it as
	// "no cap". A one-byte cap keeps the old zero-byte behaviour without letting
	// the query read every row a policy meant to bound.
	queryOpts := util.OmniQueryOptions(
		queryContext,
		false,
		util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON),
	)
	queryOpts.MaxResultBytes = util.PositiveMaxResultBytes(queryContext.MaximumSQLResultSize)
	return d.QueryHeldConn(
		ctx,
		statement,
		queryOpts,
		util.SessionQueryOptions{},
		util.QueryResultOptions{ClearExplainPlan: queryContext.Explain},
	)
}

// Probe runs a live review rule's probe on a fresh session. The contract has no
// probe primitive, so the adapter starts a transaction, runs the pre-executions
// and the statement in order, and rolls back, as the old *sql.DB probe did.
// pCtx.Autocommit runs the statement alone; a MySQL probe is an EXPLAIN, so
// either shape answers the same.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return d.ProbeOmni(ctx, pCtx, statement, util.OmniProbeOptions{
		BeginStatement:               "START TRANSACTION",
		RollbackStatement:            "ROLLBACK",
		AllowAutocommitPreExecutions: true,
		SkipEmptyPreExecutions:       true,
		TakeLastResult:               true,
	})
}

// CountAffectedRows returns the planner's estimate for one DML statement.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.Driver.(db.RowEstimator).CountAffectedRows(ctx, statement)
}

// ExplainJSON returns EXPLAIN FORMAT=JSON for one statement.
func (d *Driver) ExplainJSON(ctx context.Context, statement string) (string, error) {
	return d.Driver.(interface {
		ExplainJSON(context.Context, string) (string, error)
	}).ExplainJSON(ctx, statement)
}

// EstimateAffectedRows reads a MySQL node's estimate from plan.
func (d *Driver) EstimateAffectedRows(ctx context.Context, stmt mysqlast.Node, query, plan string) (int64, error) {
	driver, ok := d.Driver.(*mysqld.Driver)
	if !ok {
		return 0, errors.New("mysql: EstimateAffectedRows needs the MySQL driver")
	}
	return driver.EstimateAffectedRows(ctx, stmt, query, plan)
}

// EstimateAffectedRowsMariaDB reads a MariaDB node's estimate from plan.
func (d *Driver) EstimateAffectedRowsMariaDB(ctx context.Context, stmt mariadbast.Node, query, plan string) (int64, error) {
	driver, ok := d.Driver.(*mariadbd.Driver)
	if !ok {
		return 0, errors.New("mysql: EstimateAffectedRowsMariaDB needs the MariaDB driver")
	}
	return driver.EstimateAffectedRows(ctx, stmt, query, plan)
}
