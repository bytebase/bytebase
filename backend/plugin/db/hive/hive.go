// Package hive is Bytebase's Hive driver: an adapter over
// github.com/bytebase/omni/hive/driver that maps Bytebase's connection config
// onto the driver contract, and the contract's results back into the product's.
// The conversation with HiveServer2 — session setup, the sync, the server-DDL
// dump, queries and change scripts — is the driver's.
//
// # What this package did before the cutover
//
// Nothing worked, and no test said so. The driver built a DSN that named no
// login — gohive's username and password were never set, only "auth" and
// "service" — so opening any data source ran SASL PLAIN with an empty username
// and the server answered `Bad SASL negotiation status: 3 (Error validating the
// login)`, which Open reported as `failed to ping hive server`. SyncDBSchema
// never got that far: it scanned DESCRIBE FORMATTED's columns into plain
// strings, and the answer's headings and separator rows hold NULL, so the first
// table failed with `sql: Scan error ... converting NULL to string` on
// `data_type`. Dump wrote CREATE INDEX statements for an index list the sync
// never filled, rendered a materialized view as CREATE MATERIALIZED VIEW ... AS
// with an empty body, and wrapped every table in a `-- TABLE structure for`
// header. Anything that did reach the server produced rows; the result-size cap
// was checked at power-of-two row counts, and a float column came back as text
// with twenty fractional digits. GetDB handed the product a *sql.DB and no
// product call site dereferenced it. Extra connection parameters, SSL, and the
// data source's SSH host were all ignored.
//
// # What a user of this engine sees differently
//
// Error or empty, to working:
//   - Open: a data source with a username and password authenticates, so every
//     operation below becomes reachable at all.
//   - SyncDBSchema: managed tables, external tables, views and materialized
//     views sync, with columns, comments, row count, data size and partitions;
//     the NULL-bearing DESCRIBE FORMATTED answer is read as such.
//   - Dump: the objects restore, because each one is the server's own
//     SHOW CREATE TABLE statement rather than the renderer's reconstruction.
//   - The editor's USE: a query's QueryContext.Schema selects the database
//     before the script runs.
//   - Extra connection parameters reach the Hive session, SSL is honoured, and
//     Kerberos maps to auth=KERBEROS/service=<primary> as it did.
//
// Shape changes:
//   - A DOUBLE or FLOAT cell is a Double/Float value rather than text with
//     twenty fractional digits, so exported CSV says 1.5 where it said
//     1.50000000000000000000.
//   - The result-size cap is estimated per row rather than measured with
//     proto.Size at power-of-two row counts, so a cut can fall one row later.
//   - A statement that fails in a script ends the script with its error on that
//     statement's QueryResult; the call itself succeeds, where it used to fail
//     with no results.
//   - Execute reports each command to the task run's log.
//   - GetDB returns nil.
//   - Dump text: qualified object names, `-- Table structure for`, the
//     terminator on the last line, and a blank line between objects.
//   - A database dropped under an open driver fails at its USE, which the
//     script runs first; the old driver ran from whatever the connection had
//     selected.
//   - An SSH data source is refused with the local-forward instruction: gohive
//     v2's connector keeps its dialer in an unexported field, so no caller can
//     hand the driver a tunnel. A local forwarder would restore it; Bytebase
//     has no Hive one, and util.PortFIFO is dead code, not a Hive path.
//
// One defect is kept: a failed statement's error is gohive's rendering of
// Hive's TStatus, whose fields gohive keeps as pointers it prints as addresses,
// so the message a user sees is unchanged in shape.
//
// The metadata Dump is given is not read: omni has no hive catalog, so nothing
// renders a synced schema back to HiveQL.
package hive

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"strings"

	odriver "github.com/bytebase/omni/driver"
	hivedriver "github.com/bytebase/omni/hive/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_HIVE, newDriver)
}

// Driver is Bytebase's Hive driver.
type Driver struct {
	impl *hivedriver.Driver
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract and reaches the
// server, as the driver this replaced did: a HiveServer2 that cannot be reached
// or will not accept the login fails here rather than at the first query.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, err := hiveConfig(config)
	if err != nil {
		return nil, err
	}
	impl, err := hivedriver.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := impl.Ping(ctx); err != nil {
		_ = impl.Close()
		return nil, errors.Wrap(err, "failed to ping hive server")
	}
	d.impl = impl
	return d, nil
}

// hiveConfig resolves what Bytebase knows about reaching a HiveServer2 into the
// plain values hive/driver's gohive connector takes.
func hiveConfig(config db.ConnectionConfig) (odriver.Config, error) {
	ds := config.DataSource
	if ds.GetSshHost() != "" {
		// gohive v2's connector keeps its dialer in an unexported field, so the
		// driver cannot dial through a tunnel. Say what to do instead of dialling
		// the direct host and silently ignoring the jump host, as the old driver
		// did.
		return odriver.Config{}, errors.Errorf("an SSH data source is not supported for Hive: forward a local port to %s and use the forwarded host and port instead", serverAddress(ds))
	}

	params := map[string]string{}
	if t, ok := ds.GetSaslConfig().GetMechanism().(*storepb.SASLConfig_KrbConfig); ok {
		// Kerberos needs its krb5.conf and keytab in place before gohive
		// connects; the environment is process-wide, so it is booted under the
		// same lock every other driver uses.
		util.Lock.Lock()
		defer util.Lock.Unlock()
		if err := util.BootKerberosEnv(t); err != nil {
			return odriver.Config{}, errors.Wrapf(err, "failed to init SASL environment")
		}
		params["auth"] = "KERBEROS"
		service := "hive"
		if t.KrbConfig.Primary != "" {
			service = t.KrbConfig.Primary
		}
		params["service"] = service
	}
	// The caller's own parameters reach the session, and win over the SASL
	// settings above.
	for key, value := range ds.GetExtraConnectionParameters() {
		params[key] = value
	}

	cfg := odriver.Config{
		Host:     ds.GetHost(),
		Port:     ds.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		User:     ds.GetUsername(),
		Password: config.Password,
		Params:   params,
	}
	if ds.GetUseSsl() {
		// GetTLSConfig binds certificate verification to the data source's host,
		// which is the host the driver dials: an SSH tunnel is refused above, so
		// the two cannot differ.
		cfg.TLS = func(string) (*tls.Config, error) {
			return util.GetTLSConfig(ds)
		}
	}
	return cfg, nil
}

// serverAddress is the host and port a refusal names.
func serverAddress(ds *storepb.DataSource) string {
	port := ds.GetPort()
	if port == "" {
		port = "10000"
	}
	return ds.GetHost() + ":" + port
}

func (d *Driver) Close(context.Context) error {
	return d.impl.Close()
}

// Ping reaches the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// GetDB returns nil: Hive is driven through omni's session, not a *sql.DB.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute runs the script's statements in order, reporting each to the task
// run's log. Hive has no transaction the client controls — HiveServer2 4.0
// refuses BEGIN, COMMIT and ROLLBACK — so every statement commits itself and
// the script's transaction directive stays a comment, as it was before.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.impl.Execute(ctx, statement, odriver.ExecuteOptions{
		Transaction: odriver.AutoCommit,
		MaxRetries:  opts.MaximumRetries,
		Log:         executeLogger{opts: opts},
	})
}

// QueryConn runs a script for the editor, one result per statement. Hive
// answers a statement that returns no rows with an operation that has none, so
// a DDL statement sent here reports Hive's error on its result instead of
// failing the call, which is how the driver this replaced behaved too.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	session, err := d.impl.Session(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	results, err := session.Query(ctx, statement, odriver.QueryOptions{
		Schema:         queryContext.Schema,
		Explain:        queryContext.Explain,
		ExplainFormat:  explainFormat(queryContext.Option.GetExplainFormat()),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, toQueryResult(result, queryContext.MaximumSQLResultSize))
	}
	return converted, nil
}

// explainFormat is the plan format in hive/driver's vocabulary: FORMATTED for
// JSON, plain EXPLAIN for everything else. QueryContext.Limit is not passed on:
// Hive has no parser here to place a LIMIT with, so the driver caps bytes only.
func explainFormat(format v1pb.QueryOption_ExplainFormat) string {
	if format == v1pb.QueryOption_JSON {
		return "JSON"
	}
	return "TEXT"
}

// SyncInstance reads the server version and its databases. Hive has no role a
// sync reads.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
	}, nil
}

// SyncDBSchema reads the database the driver was opened on.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.impl.SyncDatabase(ctx)
}

// Dump writes the database's DDL as the server states it. omni has no hive
// catalog, so the metadata is not rendered and does not narrow the dump.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, dbMetadata)
}

// toQueryResult maps one contract result onto the product's result. The plan a
// caller asked for arrives as rows, and the API names its format; the driver
// states no typed QueryPlan, as before.
func toQueryResult(result *odriver.Result, maxResultBytes int64) *v1pb.QueryResult {
	converted := &v1pb.QueryResult{
		Statement: result.Statement,
		Latency:   durationpb.New(result.Latency),
	}
	for _, column := range result.Columns {
		converted.ColumnNames = append(converted.ColumnNames, column.Name)
		converted.ColumnTypeNames = append(converted.ColumnTypeNames, strings.ToUpper(column.TypeName))
	}
	for _, row := range result.Rows {
		convertedRow := &v1pb.QueryRow{}
		for _, value := range row {
			convertedRow.Values = append(convertedRow.Values, toRowValue(value))
		}
		converted.Rows = append(converted.Rows, convertedRow)
	}
	converted.RowsCount = int64(len(converted.Rows))
	if result.Truncated {
		converted.Error = util.FormatMaximumSQLResultSizeMessage(maxResultBytes)
	} else if result.Err != nil {
		converted.Error = result.Err.Error()
	}
	return converted
}

// toRowValue maps one contract value onto the product's row value. A float is
// the number Hive stored, where the old driver wrote it as text with twenty
// fractional digits.
func toRowValue(value odriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case odriver.Null:
		return util.NullRowValue
	case odriver.Bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value.Bool}}
	case odriver.Bytes:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case odriver.Float32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: float32(value.Float64)}}
	case odriver.Float64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case odriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(value.Int64)}}
	case odriver.Int64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case odriver.Uint32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(value.Uint64)}}
	case odriver.Uint64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: value.Uint64}}
	case odriver.String:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	default:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value)}}
	}
}

// executeLogger reports the driver contract's execute steps to the task run's
// log.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l executeLogger) CommandExecute(command odriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{
		Start: int32(command.ByteStart),
		End:   int32(command.ByteEnd),
	}, command.Text)
}

func (l executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (l executeLogger) TransactionControl(op odriver.TransactionOp, err error) {
	var t storepb.TaskRunLog_TransactionControl_Type
	switch op {
	case odriver.Begin:
		t = storepb.TaskRunLog_TransactionControl_BEGIN
	case odriver.Commit:
		t = storepb.TaskRunLog_TransactionControl_COMMIT
	case odriver.Rollback:
		t = storepb.TaskRunLog_TransactionControl_ROLLBACK
	default:
		return
	}
	l.opts.LogTransactionControl(t, errorString(err))
}

func (l executeLogger) Retry(err error, attempt, _ int) {
	l.opts.LogRetryInfo(err, attempt)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
