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
	"fmt"
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

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_SPANNER, newDriver)
}

// Driver is the Spanner driver.
type Driver struct {
	util.OmniAdapter
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
	d.OmniAdapter = util.OmniAdapter{
		Driver:           impl,
		NotFoundResource: "database",
		DatabaseName:     config.ConnectionContext.DatabaseName,
	}
	d.AddCloser(admin)
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
		Database: util.ResolveDatabaseName(config),
		Password: config.Password,
		Params: map[string]string{
			spannerdriver.ParamProject:  ds.GetProjectId(),
			spannerdriver.ParamInstance: ds.GetInstanceId(),
		},
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

// Execute executes a SQL statement. spanner/driver splits the script by the
// database's dialect, applies a script that holds any DDL statement through the
// admin API whole, and runs the rest as DML.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// Spanner's driver logs only transaction control entries (no per-statement command logs).
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassCreateDatabase:        true,
		PassMaxRetries:            true,
		LoggerOptions:             []util.ExecuteLoggerOption{util.WithSkipCommandLogs()},
	})
}

// QueryConn queries a SQL statement. Spanner has no session state a statement
// can set, so each statement reads in a single-use transaction of its own.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	var results []*odriver.Result
	err := d.QueryEphemeral(ctx, func(session odriver.Session) error {
		var queryErr error
		results, queryErr = session.Query(ctx, statement, odriver.QueryOptions{
			Limit:          queryContext.Limit,
			Explain:        queryContext.Explain,
			MaxResultBytes: queryContext.MaximumSQLResultSize,
		})
		return queryErr
	})
	if err != nil {
		return nil, err
	}
	return toQueryResults(results, queryContext.MaximumSQLResultSize, queryContext.Explain), nil
}

// toQueryResults converts spanner/driver's results into the API's, keeping each
// statement's error, plan and latency.
func toQueryResults(results []*odriver.Result, maxResultBytes int64, isExplainRequest bool) []*v1pb.QueryResult {
	return util.OmniResultsToQueryResultsWith(results, maxResultBytes, util.QueryResultOptions{
		ClearExplainPlan: isExplainRequest,
		TypedExplainPlan: true,
		ConvertValue:     toRowValue,
	})
}

func toRowValue(value odriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case odriver.Float32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case odriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case odriver.Structured:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	case odriver.Bool, odriver.Bytes, odriver.Float64, odriver.Int64, odriver.String, odriver.Timestamp:
		return util.OmniRowValue(value)
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

// isEmulator reports whether the caller's parameters name the Cloud Spanner
// emulator, as spanner/driver spells the flag.
func isEmulator(params map[string]string) bool {
	switch strings.ToLower(params[spannerdriver.ParamEmulator]) {
	case "1", "true", "yes":
		return true
	}
	return false
}
