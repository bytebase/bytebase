// Package tidb is Bytebase's TiDB driver: an adapter over
// github.com/bytebase/omni/tidb/driver. It maps a db.ConnectionConfig onto the
// contract's Config and the contract's results back onto the product's types;
// the connection, the sync queries, the plan estimate, the splitter, the query
// rewrites and the dump belong to the omni driver.
//
// The adapter keeps what is product: the data source to contract conversion,
// the SSH tunnel and TLS material, the `-- txn-mode` directive, the advisor
// probe, the v1pb conversion and the task-run logger.
//
// Constraints of the engine and the contract, here rather than at each call:
//   - Config.ReadOnly is unused: TiDB's MySQL wire handshake has no read-only
//     session parameter.
//   - Config.ApplicationName is "bytebase", which renders the /*app=bytebase*/
//     comment every statement of a query or a change script carries.
//   - Execute ignores ExecuteOptions.CreateDatabase and the `-- txn-isolation`
//     directive: a create-database script runs under the transaction directive,
//     and no isolation level is set.
//   - A change script still reads its transaction mode from
//     ParseTransactionConfig, so the directives do not reach the server.
//   - A probe runs on one session so its pre-executions and its statement share
//     a transaction, opened with the "START TRANSACTION" go-sql-driver's
//     BeginTx sends and ended with "ROLLBACK".
//   - QueryConn runs on the omni driver's Session, which the adapter keeps
//     until Close, so the statements of one editor request share it.
//   - Dump renders omni's catalog, whose partitions keep their ordinal order;
//     that order is what makes a partitioned table's dump restorable.
//   - Query results carry the contract's typed values: an unsigned integer is a
//     Uint64 value, the BLOB, BINARY and VARBINARY families are bytes, and an
//     empty binary is empty bytes rather than NULL.
//   - A result-size cap is checked against a row's real encoded size, and a cap
//     of zero is refused: the product reads 0 as "no rows" and the contract
//     reads it as "no cap".
//   - Instance roles are read in a deterministic (user, host) order, and a
//     grant names its account with proper quoting, so an account whose name
//     contains a quote is read.
package tidb

import (
	"context"
	"fmt"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	tidbdriver "github.com/bytebase/omni/tidb/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver       = (*Driver)(nil)
	_ db.Prober       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)

	// applicationName renders the /*app=bytebase*/ comment the old driver
	// prepended to a query and to a change script's commands.
	applicationName = "bytebase"
)

func init() {
	db.Register(storepb.Engine_TIDB, newDriver)
}

// Driver is Bytebase's TiDB driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the contract's Config. Like the driver
// this replaced, Open does not reach the server; Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if err := d.OpenWith(ctx, util.AdapterOptions{
		WithInstanceMetadata: true,
		NotFoundResource:     "database",
		DatabaseName:         config.ConnectionContext.DatabaseName,
	}, func(adapter *util.OmniAdapter) (omnidriver.Driver, error) {
		driverConfig, err := buildConfig(config, adapter)
		if err != nil {
			return nil, err
		}
		return tidbdriver.Open(ctx, driverConfig)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// buildConfig resolves what Bytebase knows about reaching a TiDB server into the
// plain values tidb/driver takes.
func buildConfig(config db.ConnectionConfig, adapter *util.OmniAdapter) (omnidriver.Config, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return omnidriver.Config{}, errors.New("a data source is required")
	}
	driverConfig := omnidriver.Config{
		Host:            dataSource.Host,
		Port:            dataSource.Port,
		Database:        config.ConnectionContext.DatabaseName,
		User:            dataSource.Username,
		Password:        config.Password,
		Params:          util.CloneParams(dataSource.GetExtraConnectionParameters()),
		ApplicationName: applicationName,
		EngineVersion:   config.ConnectionContext.EngineVersion,
		TLS:             util.LazyTLSFunc(dataSource),
	}
	if err := util.SetupSSHDial(dataSource, &driverConfig, adapter, false); err != nil {
		return omnidriver.Config{}, err
	}
	return driverConfig, nil
}

// Execute runs the script's statements under the transaction directive it
// carries, reporting each step to the task run log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassMaxRetries:            true,
	})
}

// QueryConn queries a SQL statement on the driver's session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	// A non-positive cap means "no rows" to the product and "no cap" to the
	// contract, so refuse it rather than read every row a policy meant to
	// bound.
	if queryContext.MaximumSQLResultSize <= 0 {
		return nil, errors.New("maximum SQL result size must be positive")
	}
	return d.QueryHeldConn(
		ctx,
		statement,
		util.OmniQueryOptions(queryContext, true, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON)),
		util.SessionQueryOptions{},
		util.QueryResultOptions{ClearExplainPlan: queryContext.Explain},
	)
}

// CountAffectedRows returns TiDB's planner estimate of the rows statement
// modifies. The contract answers it for exactly one INSERT, UPDATE or DELETE and
// refuses any other statement before it reaches the server.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.Driver.(db.RowEstimator).CountAffectedRows(ctx, statement)
}

// Probe runs a live review rule's probe. A BATCH ... DRY RUN statement passes
// Autocommit, because TiDB refuses it inside a transaction.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return d.ProbeOmni(ctx, pCtx, statement, util.OmniProbeOptions{
		BeginStatement:           "START TRANSACTION",
		RollbackStatement:        "ROLLBACK",
		RollbackWithoutCancel:    true,
		SkipEmptyPreExecutions:   true,
		CheckPreExecutionResults: true,
		FormatCell:               probeCell,
	})
}

// probeCell maps one contract value onto the Go value a probe result carries.
func probeCell(value omnidriver.Value) any {
	switch value.Kind {
	case omnidriver.Timestamp, omnidriver.TimestampTZ:
		return timestampText(value)
	default:
		return util.DefaultProbeCell(value)
	}
}

// timestampText renders a timestamp the way the session spells it, so a probe's
// cell reads the same as it did when the driver scanned the server's text.
func timestampText(value omnidriver.Value) string {
	text := value.Time.Format(time.DateTime)
	if value.Precision > 0 {
		fraction := fmt.Sprintf("%09d", value.Time.Nanosecond())
		if int(value.Precision) < len(fraction) {
			fraction = fraction[:value.Precision]
		}
		text += "." + fraction
	}
	return text
}
