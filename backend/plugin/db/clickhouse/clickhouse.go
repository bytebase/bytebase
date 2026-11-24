// Package clickhouse is Bytebase's ClickHouse driver: an adapter over
// github.com/bytebase/omni/clickhouse/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos.
//
// Every db.Driver method delegates:
//
//   - Open maps the data source onto driver.Config: the native endpoint, the
//     database the editor opened, the login, TLS material, the caller's extra
//     parameters as ClickHouse settings, and an SSH Dial for a tunnelled data
//     source. It does not reach the server; Ping does.
//   - Ping, Close, Execute, SyncInstance, SyncDBSchema and Dump delegate.
//   - QueryConn asks the driver for one session and holds it, so statements a
//     caller runs through one driver share a session.
//   - Execute reports each statement's range and the transaction control steps
//     to the task run's log.
//
// These are the contracts a caller can rely on, stated because a stored dump
// and the editor's results depend on them:
//
//   - Dump writes the server's own SHOW CREATE statement for every object,
//     database-qualified and ordered so a restore replays: tables and
//     dictionaries, then views, then materialized views and Dictionary-engine
//     tables after what they read, with the ".inner_id." storage tables left
//     out. The metadata it is given is not rendered and does not narrow the
//     dump, so a dump restores and round-trips byte-identically.
//   - A statement that returns no rows answers with an empty result. Execute
//     answers 0 because clickhouse-go states no affected count.
//   - Date, DateTime and DateTime64 are typed timestamps carrying the precision
//     their type names, and a zoned one keeps its zone and offset; Time and
//     Time64 are clock strings; IPv4, IPv6 and FixedString are their values.
//   - SHOW, DESCRIBE, DESC, EXISTS and CHECK run as queries, and EXPLAIN
//     ANALYZE keeps the rows the server returns beside a typed QueryPlan.
//   - The result-size cap is byte-accurate and MaximumSQLResultSize 0 is no
//     cap. A limit rewrite wraps only a statement that can stand in a CTE and
//     drops trailing comments a CTE cannot hold.
//   - SyncInstance lists databases and users in name order. A sorting key,
//     primary key or index expression is split at top-level commas only, so a
//     function or tuple keeps its own.
//   - A connection the server refuses because the database is gone is
//     driver.ErrNotFound. A sync on a session already open on a dropped
//     database answers an empty schema.
//
// Extra connection parameters reach ClickHouse as settings, and an SSH data
// source connects through Config.Dial.
package clickhouse

import (
	"context"
	"log/slog"

	omnich "github.com/bytebase/omni/clickhouse/driver"
	omnidriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_CLICKHOUSE, newDriver)
}

// Driver is Bytebase's ClickHouse driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a ClickHouse driver. It does not reach the server: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, err := clickhouseConfig(config.DataSource, config.Password, config.ConnectionContext.DatabaseName)
	if err != nil {
		return nil, err
	}
	d := &Driver{}
	if err := d.OpenWith(ctx, util.AdapterOptions{WithInstanceMetadata: true}, func(adapter *util.OmniAdapter) (omnidriver.Driver, error) {
		if err := util.SetupSSHDial(config.DataSource, &cfg, adapter, true); err != nil {
			return nil, err
		}
		slog.Debug("Opening ClickHouse driver",
			slog.String("addr", config.DataSource.GetHost()+":"+config.DataSource.GetPort()),
			slog.String("environment", config.ConnectionContext.EnvironmentID),
			slog.String("database", config.ConnectionContext.InstanceID),
		)
		return omnich.Open(ctx, cfg)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// clickhouseConfig resolves everything Bytebase knows about reaching the server
// into the plain values clickhouse/driver takes.
func clickhouseConfig(dataSource *storepb.DataSource, password, database string) (omnidriver.Config, error) {
	tlsFn, err := util.SQLStaticTLSFunc(dataSource)
	if err != nil {
		return omnidriver.Config{}, err
	}
	return omnidriver.Config{
		Host:     dataSource.GetHost(),
		Port:     dataSource.GetPort(),
		Database: database,
		User:     dataSource.GetUsername(),
		Password: password,
		// A data source's extra parameters are ClickHouse settings, which is
		// what the contract carries them as.
		Params: dataSource.GetExtraConnectionParameters(),
		TLS:    tlsFn,
	}, nil
}

// Execute runs the script's statements, reporting each to the task run's log.
// ClickHouse cannot roll back, so a script still logs the BEGIN and ROLLBACK
// its transaction wrapper issues.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
	})
}

// QueryConn runs each statement of the script in order on the driver's session.
// Calls are serialized on that one session, as they were on the editor's connection.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(ctx, statement, omnidriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  queryContext.Option.GetExplainFormat().String(),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	}, util.SessionQueryOptions{
		ShouldDiscardResults: func(ctx context.Context, session omnidriver.Session, results []*omnidriver.Result) bool {
			// A statement's error does not say whether the client's connection
			// survived it: a server exception leaves the connection in sync,
			// while a decode or protocol error leaves it unusable.
			return firstError(results) != nil && !sessionAnswers(ctx, session)
		},
	}, clickhouseQueryResultOptions(queryContext.Explain))
}

// firstError returns the first statement error, or nil.
func firstError(results []*omnidriver.Result) error {
	for _, result := range results {
		if result.Err != nil {
			return result.Err
		}
	}
	return nil
}

// sessionAnswers reports whether the session can still run a trivial statement.
// It is called only after a statement failed, so the round trip is on the error
// path.
func sessionAnswers(ctx context.Context, session omnidriver.Session) bool {
	results, err := session.Query(ctx, "SELECT 1", omnidriver.QueryOptions{})
	if err != nil || len(results) != 1 {
		return false
	}
	return results[0].Err == nil
}

func clickhouseQueryResultOptions(clearExplainPlan bool) util.QueryResultOptions {
	return util.QueryResultOptions{
		ClearExplainPlan: clearExplainPlan,
		TypedExplainPlan: true,
		TruncatedWins:    true,
		ConvertValue:     toRowValue,
	}
}

func toRowValue(value omnidriver.Value) *v1pb.RowValue {
	if value.Kind == omnidriver.Structured {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	}
	return util.OmniRowValue(value)
}
