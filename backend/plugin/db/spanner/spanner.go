// Package spanner is the plugin for the Spanner driver. It adapts omni's
// spanner/driver, which owns the conversation with Spanner — metadata sync, the
// server-DDL dump, queries and change scripts — to Bytebase's db.Driver
// contract.
//
// Known differences from the driver this replaced, kept because they are the
// driver's own behavior or because reproducing the old one would reimplement it:
//   - A script's statement text keeps its comments. A script the parser cannot
//     read answers with the parser's error on its statement result instead of
//     failing the QueryConn call; the old splitter removed comments and let the
//     server reject the text.
//   - The result-size cap is byte-accurate rather than checked at power-of-two
//     row counts, and a cap of 0 means no cap rather than a zero-byte cap.
//   - The caller's extra parameters reach the driver ("emulator" is read) and
//     an SSL data source's TLS settings are honoured; the old driver ignored
//     both. An emulator-pointed instance drops its stored Cloud credential,
//     which the client refuses to combine with unauthenticated access.
//   - GRANT and REVOKE are DDL for the admin API; the old prefix test sent them
//     down the DML path, where Spanner refused them.
//   - An explain request stops at the first statement that cannot be planned;
//     the old driver asked for every statement's plan even after a failure.
//   - SyncInstance keeps the old GoogleSQL filter: a PostgreSQL-dialect database
//     is not listed. omni can sync and dump one when a caller opens it directly.
package spanner

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"net"
	"strings"

	spannerdb "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	spannerdriver "github.com/bytebase/omni/spanner/driver"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_SPANNER, newDriver)
}

// Driver is the Spanner driver.
type Driver struct {
	config db.ConnectionConfig

	impl *spannerdriver.Driver
	// admin lists the instance's databases with their dialects. SyncInstance
	// keeps Bytebase's GoogleSQL-only listing until the parser routes by
	// database dialect, and spanner/driver's instance list does not carry one.
	admin        *spannerdb.DatabaseAdminClient
	instancePath string
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a Spanner driver. It must connect to a specific database.
// If database isn't provided, part of the driver cannot function.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, err := spannerConfig(config)
	if err != nil {
		return nil, err
	}
	impl, err := spannerdriver.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	adminOptions, err := databaseAdminOptions(cfg)
	if err != nil {
		_ = impl.Close()
		return nil, err
	}
	admin, err := spannerdb.NewDatabaseAdminClient(ctx, adminOptions...)
	if err != nil {
		_ = impl.Close()
		return nil, err
	}
	d.config = config
	d.impl = impl
	d.admin = admin
	d.instancePath = fmt.Sprintf("projects/%s/instances/%s", config.DataSource.GetProjectId(), config.DataSource.GetInstanceId())
	return d, nil
}

// spannerConfig resolves everything Bytebase knows about reaching the instance
// into the plain values spanner/driver's clients take.
func spannerConfig(config db.ConnectionConfig) (odriver.Config, error) {
	ds := config.DataSource
	cfg := odriver.Config{
		Host:     ds.GetHost(),
		Port:     ds.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		Password: config.Password,
		Params: map[string]string{
			spannerdriver.ParamProject:  ds.GetProjectId(),
			spannerdriver.ParamInstance: ds.GetInstanceId(),
		},
	}
	if cfg.Database == "" {
		cfg.Database = ds.GetDatabase()
	}
	// Bytebase carries a Cloud credential in the data source, where the
	// instance's IAM extension holds it. A caller that resolved one itself
	// passes it as the password, and that one wins.
	if cfg.Password == "" {
		if credential := ds.GetGcpCredential(); credential != nil {
			cfg.Password = credential.GetContent()
		}
	}
	// The caller's extra parameters go in verbatim. "emulator" is the one
	// spanner/driver reads; a name it does not know is ignored.
	for key, value := range ds.GetExtraConnectionParameters() {
		cfg.Params[key] = value
	}
	// The emulator accepts any caller, and the Google client refuses to build
	// one that both carries a credential and skips authentication. An instance
	// keeps its stored credential, so an emulator-pointed one would fail Open
	// with "options.WithoutAuthentication is incompatible with any option that
	// provides credentials"; the credential is meaningless to the emulator, so
	// drop it rather than refuse.
	if isEmulator(cfg.Params) {
		cfg.Password = ""
	}
	// TLS is required exactly when the caller asks for it; nil leaves the
	// client's transport, which verifies the server's certificate.
	if ds.GetUseSsl() {
		tlsConfig, err := util.GetTLSConfig(ds)
		if err != nil {
			return odriver.Config{}, err
		}
		cfg.TLS = func(host string) (*tls.Config, error) {
			byHost := tlsConfig.Clone()
			byHost.ServerName = host
			return byHost, nil
		}
	}
	return cfg, nil
}

// databaseAdminOptions builds the admin client's options from the same resolved
// config spanner/driver's own clients take. It exists only for SyncInstance's
// dialect listing; every other call is the driver's.
func databaseAdminOptions(cfg odriver.Config) ([]option.ClientOption, error) {
	var options []option.ClientOption
	if cfg.Password != "" {
		credential, err := util.GCPCredentialOption([]byte(cfg.Password))
		if err != nil {
			return nil, err
		}
		options = append(options, credential)
	}
	endpoint := ""
	if cfg.Host != "" {
		endpoint = cfg.Host
		if cfg.Port != "" {
			endpoint = net.JoinHostPort(cfg.Host, cfg.Port)
		}
		options = append(options, option.WithEndpoint(endpoint))
	}
	if isEmulator(cfg.Params) {
		options = append(options, option.WithoutAuthentication())
		if cfg.TLS == nil {
			options = append(options, option.WithGRPCDialOption(
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			))
		}
	}
	if cfg.TLS != nil {
		host := endpoint
		if splitHost, _, err := net.SplitHostPort(endpoint); err == nil {
			host = splitHost
		}
		tlsConfig, err := cfg.TLS(host)
		if err != nil {
			return nil, err
		}
		if tlsConfig == nil {
			return nil, errors.New("TLS configuration must not be nil")
		}
		options = append(options, option.WithGRPCDialOption(
			grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		))
	}
	if cfg.Dial != nil {
		options = append(options, option.WithGRPCDialOption(
			grpc.WithContextDialer(func(ctx context.Context, address string) (net.Conn, error) {
				return cfg.Dial(ctx, "tcp", address)
			}),
		))
	}
	return options, nil
}

// Close closes the driver.
func (d *Driver) Close(_ context.Context) error {
	var closeErr error
	if d.admin != nil {
		closeErr = d.admin.Close()
	}
	if err := d.impl.Close(); closeErr == nil {
		closeErr = err
	}
	return closeErr
}

// Ping pings the instance.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute executes a SQL statement. spanner/driver splits the script by the
// database's dialect, applies a script that holds any DDL statement through the
// admin API whole, and runs the rest as DML.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	config, cleaned := base.ParseTransactionConfig(statement)
	statement = cleaned
	transactionMode := config.Mode
	if transactionMode == transaction.ModeUnspecified {
		transactionMode = transaction.DefaultMode()
	}

	executeOptions := odriver.ExecuteOptions{
		Transaction:    odriver.Transactional,
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log:            executeLogger{opts: opts},
	}
	if transactionMode == transaction.ModeOff {
		executeOptions.Transaction = odriver.AutoCommit
	}
	return d.impl.Execute(ctx, statement, executeOptions)
}

// executeLogger reports a script's transaction control into the task run log.
// Spanner's driver logs nothing else: it logged no per-statement command line
// before the cutover either.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (executeLogger) CommandExecute(odriver.Command) {}

func (executeLogger) CommandResponse(int64, []int64, error) {}

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
	l.opts.LogTransactionControl(t, errString(err))
}

func (executeLogger) Retry(error, int, int) {}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// QueryConn queries a SQL statement in a given connection. Spanner has no
// session state a statement can set, so the connection the caller used to hold
// is unused and each statement reads in a single-use transaction of its own.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	session, err := d.impl.Session(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	results, err := session.Query(ctx, statement, odriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	})
	if err != nil {
		return nil, err
	}
	return toQueryResults(results, queryContext.MaximumSQLResultSize, queryContext.Explain), nil
}

// toQueryResults converts spanner/driver's results into the API's, keeping each
// statement's error, plan and latency.
func toQueryResults(results []*odriver.Result, maxResultBytes int64, isExplainRequest bool) []*v1pb.QueryResult {
	if len(results) == 0 {
		return nil
	}
	queryResults := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		queryResults = append(queryResults, toQueryResult(result, maxResultBytes, isExplainRequest))
	}
	return queryResults
}

// toQueryResult converts one result. isExplainRequest keeps the typed plan out
// of an explain request's results: QueryPlan describes a plan the caller's own
// statement asked for, not the one the request asked the driver to build.
func toQueryResult(result *odriver.Result, maxResultBytes int64, isExplainRequest bool) *v1pb.QueryResult {
	queryResult := &v1pb.QueryResult{
		Statement: result.Statement,
		Latency:   durationpb.New(result.Latency),
	}
	if !isExplainRequest {
		queryResult.QueryPlan = planOf(result.Plan)
	}
	if result.AffectedRows != nil {
		affected := util.BuildAffectedRowsResult(*result.AffectedRows, nil)
		affected.Statement = result.Statement
		affected.Latency = queryResult.Latency
		affected.RowsCount = int64(len(affected.Rows))
		return affected
	}
	for _, column := range result.Columns {
		queryResult.ColumnNames = append(queryResult.ColumnNames, column.Name)
		queryResult.ColumnTypeNames = append(queryResult.ColumnTypeNames, column.TypeName)
	}
	for _, row := range result.Rows {
		values := make([]*v1pb.RowValue, 0, len(row))
		for _, value := range row {
			values = append(values, toRowValue(value))
		}
		queryResult.Rows = append(queryResult.Rows, &v1pb.QueryRow{Values: values})
	}
	queryResult.RowsCount = int64(len(queryResult.Rows))
	if result.Truncated {
		queryResult.Error = util.FormatMaximumSQLResultSizeMessage(maxResultBytes)
	}
	if result.Err != nil {
		queryResult.Error = result.Err.Error()
	}
	return queryResult
}

func planOf(plan *odriver.Plan) *v1pb.QueryResult_QueryPlan {
	if plan == nil {
		return nil
	}
	return db.TypedExplainPlan(plan.Format, plan.Executed, true)
}

func toRowValue(value odriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case odriver.Bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value.Bool}}
	case odriver.Bytes:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case odriver.Float32, odriver.Float64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case odriver.Int32, odriver.Int64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case odriver.String, odriver.Structured:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	case odriver.Timestamp:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{
			TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(value.Time),
				Accuracy:        value.Precision,
			},
		}}
	default:
		return util.NullRowValue
	}
}

// SyncInstance syncs the instance's GoogleSQL databases. A PostgreSQL-dialect
// database is filtered out, as it was before the cutover: SPANNER's split and
// query span are GoogleSQL-only, so listing one would let the editor run a
// statement its masking analysis cannot read.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var databases []*metadatapb.DatabaseSchemaMetadata
	iter := d.admin.ListDatabases(ctx, &databasepb.ListDatabasesRequest{Parent: d.instancePath})
	for {
		database, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if database.DatabaseDialect == databasepb.DatabaseDialect_POSTGRESQL {
			continue
		}
		databaseName, err := getDatabaseFromResource(database.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database name from %s", database.Name)
		}
		databases = append(databases, &metadatapb.DatabaseSchemaMetadata{Name: databaseName})
	}
	return &db.InstanceMetadata{Databases: databases}, nil
}

// getDatabaseFromResource returns <database> from
// projects/<project>/instances/<instance>/databases/<database>.
func getDatabaseFromResource(name string) (string, error) {
	const marker = "/databases/"
	index := strings.LastIndex(name, marker)
	if index < 0 || index+len(marker) == len(name) {
		return "", errors.New("invalid DSN")
	}
	return name[index+len(marker):], nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	metadata, err := d.impl.SyncDatabase(ctx)
	if err != nil {
		if errors.Is(err, odriver.ErrNotFound) {
			return nil, common.Errorf(common.NotFound, "database %q not found", d.config.ConnectionContext.DatabaseName)
		}
		return nil, err
	}
	return metadata, nil
}

// Dump dumps the database's schema as its server states it.
func (d *Driver) Dump(ctx context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, metadata)
}

// isEmulator reports whether the caller's parameters name the Cloud Spanner
// emulator, as spanner/driver spells the flag.
func isEmulator(params map[string]string) bool {
	switch strings.ToLower(params[spannerdriver.ParamEmulator]) {
	case "1", "true", "yes":
		return true
	}
	return false
}
