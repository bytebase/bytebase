// Package drivers is Bytebase's database drivers: one adapter over the omni
// driver contract (github.com/bytebase/omni/driver), configured per engine.
//
// An engine's file holds what Bytebase owns about it: how a data source maps
// onto the contract's Config (credentials, IAM, tunnels, TLS), and how the
// product's execute, query and probe calls map onto the engine's driver.
// Everything after the dial is the omni driver's.
package drivers

import (
	"context"
	"io"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func init() {
	for engine, s := range map[storepb.Engine]spec{
		storepb.Engine_BIGQUERY:      bigquerySpec,
		storepb.Engine_CASSANDRA:     cassandraSpec,
		storepb.Engine_CLICKHOUSE:    clickhouseSpec,
		storepb.Engine_COCKROACHDB:   cockroachdbSpec,
		storepb.Engine_COSMOSDB:      cosmosdbSpec,
		storepb.Engine_DATABRICKS:    databricksSpec,
		storepb.Engine_DORIS:         dorisSpec,
		storepb.Engine_DYNAMODB:      dynamodbSpec,
		storepb.Engine_ELASTICSEARCH: elasticsearchSpec,
		storepb.Engine_HIVE:          hiveSpec,
		storepb.Engine_MARIADB:       mariadbSpec,
		storepb.Engine_MONGODB:       mongodbSpec,
		storepb.Engine_MSSQL:         mssqlSpec,
		storepb.Engine_MYSQL:         mysqlSpec,
		storepb.Engine_OCEANBASE:     oceanbaseSpec,
		storepb.Engine_ORACLE:        oracleSpec,
		storepb.Engine_POSTGRES:      pgSpec,
		storepb.Engine_REDIS:         redisSpec,
		storepb.Engine_REDSHIFT:      redshiftSpec,
		storepb.Engine_SNOWFLAKE:     snowflakeSpec,
		storepb.Engine_SPANNER:       spannerSpec,
		storepb.Engine_STARROCKS:     starrocksSpec,
		storepb.Engine_TIDB:          tidbSpec,
		storepb.Engine_TRINO:         trinoSpec,
	} {
		db.Register(engine, func() db.Driver { return newDriver(s) })
	}
}

// spec is what one engine's driver does beyond the shared adapter. open and
// query are required; a nil hook takes the adapter's behavior.
type spec struct {
	// open maps the connection config onto the engine's Config and opens its
	// driver. Tunnels and dialers it acquires register their closers on d.
	open func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error)
	// adapter configures metadata sync and the held session. DatabaseName is
	// filled in from the connection.
	adapter util.AdapterOptions
	execute util.ExecuteConfig
	query   func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error)
	// probe makes the driver a db.Prober with these options.
	probe func(d *Driver) util.OmniProbeOptions
	// taskRunSession makes the driver a db.TaskRunSessionReader.
	taskRunSession func(ctx context.Context, d *Driver, applicationName string) (*v1pb.TaskRunSession, error)

	ping         func(ctx context.Context, d *Driver) error
	syncInstance func(ctx context.Context, d *Driver) (*db.InstanceMetadata, error)
	dump         func(ctx context.Context, d *Driver, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error
}

// Driver is Bytebase's driver for every engine, over the engine's omni driver.
type Driver struct {
	util.OmniAdapter
	spec   spec
	config db.ConnectionConfig
}

func newDriver(s spec) *Driver {
	return &Driver{spec: s}
}

// Open opens the engine's driver for the connection.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	opened := &Driver{spec: d.spec, config: config}
	opts := d.spec.adapter
	opts.DatabaseName = config.ConnectionContext.DatabaseName
	if err := opened.OpenWith(ctx, opts, func(*util.OmniAdapter) (odriver.Driver, error) {
		return d.spec.open(ctx, opened, config)
	}); err != nil {
		return nil, err
	}
	return opened, nil
}

// Unwrap lists the capabilities db.As finds: the probe and session reader the
// engine's spec states, then whatever the omni driver implements itself
// (db.RowEstimator, db.ColumnExtraLookup, a private-link owner).
func (d *Driver) Unwrap() []any {
	var capabilities []any
	if d.spec.probe != nil {
		capabilities = append(capabilities, prober{d})
	}
	if d.spec.taskRunSession != nil {
		capabilities = append(capabilities, taskRunSessionReader{d})
	}
	return append(capabilities, d.Driver)
}

// Ping reaches the server.
func (d *Driver) Ping(ctx context.Context) error {
	if d.spec.ping != nil {
		return d.spec.ping(ctx, d)
	}
	return d.OmniAdapter.Ping(ctx)
}

// Execute runs a change script, reporting each step to the task run log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, d.spec.execute)
}

// QueryConn runs a script for the SQL editor.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.spec.query(ctx, d, statement, queryContext)
}

// SyncInstance reads the server version, its databases and its roles.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	if d.spec.syncInstance != nil {
		return d.spec.syncInstance(ctx, d)
	}
	return d.OmniAdapter.SyncInstance(ctx)
}

// Dump writes the database's schema DDL.
func (d *Driver) Dump(ctx context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	if d.spec.dump != nil {
		return d.spec.dump(ctx, d, out, metadata)
	}
	return d.OmniAdapter.Dump(ctx, out, metadata)
}

type prober struct{ d *Driver }

func (p prober) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return p.d.ProbeOmni(ctx, pCtx, statement, p.d.spec.probe(p.d))
}

// Unwrap keeps the driver's other capabilities reachable from the probe the
// advisor holds.
func (p prober) Unwrap() []any {
	return p.d.Unwrap()
}

type taskRunSessionReader struct{ d *Driver }

func (r taskRunSessionReader) TaskRunSession(ctx context.Context, applicationName string) (*v1pb.TaskRunSession, error) {
	return r.d.spec.taskRunSession(ctx, r.d, applicationName)
}
