// Package spanner is Bytebase's Cloud Spanner driver, adapting
// github.com/bytebase/omni/spanner/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - SyncInstance filters out PostgreSQL-dialect databases via the admin API because Bytebase's SPANNER parser is GoogleSQL-only.
//   - Emulator connections drop stored GCP credentials because Google's client rejects credentials combined with WithoutAuthentication.
//   - Execute logs transaction control entries only (per-statement command logs are skipped).
package spanner

import (
	"context"
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

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_SPANNER, newDriver)
}

// Driver is the Spanner driver.
type Driver struct {
	util.OmniAdapter
	// admin lists the instance's databases with their dialects so SyncInstance
	// can filter out PostgreSQL-dialect databases.
	admin        *spannerdb.DatabaseAdminClient
	instancePath string
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a Spanner driver targeting the configured project, instance, and database.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, err := spannerConfig(config)
	if err != nil {
		return nil, err
	}
	d := &Driver{
		instancePath: fmt.Sprintf("projects/%s/instances/%s", config.DataSource.GetProjectId(), config.DataSource.GetInstanceId()),
	}
	if err := d.OpenWith(ctx, util.AdapterOptions{
		NotFoundResource: "database",
		DatabaseName:     config.ConnectionContext.DatabaseName,
	}, func(adapter *util.OmniAdapter) (odriver.Driver, error) {
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
		adapter.AddCloser(admin)
		d.admin = admin
		return impl, nil
	}); err != nil {
		return nil, err
	}
	return d, nil
}

func spannerConfig(config db.ConnectionConfig) (odriver.Config, error) {
	ds := config.DataSource
	params := map[string]string{
		spannerdriver.ParamProject:  ds.GetProjectId(),
		spannerdriver.ParamInstance: ds.GetInstanceId(),
	}
	for key, value := range ds.GetExtraConnectionParameters() {
		params[key] = value
	}
	password := config.Password
	if password == "" {
		password = ds.GetGcpCredential().GetContent()
	}
	// Google's client refuses to combine a credential with WithoutAuthentication.
	if isEmulator(params) {
		password = ""
	}
	tlsFunc, err := util.StaticHostTLSFunc(ds)
	if err != nil {
		return odriver.Config{}, err
	}
	return odriver.Config{
		Host:     ds.GetHost(),
		Port:     ds.GetPort(),
		Database: util.ResolveDatabaseName(config),
		Password: password,
		Params:   params,
		TLS:      tlsFunc,
	}, nil
}

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

// Execute runs a SQL script through omni/spanner/driver, logging transaction
// control entries only.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassCreateDatabase:        true,
		PassMaxRetries:            true,
		LoggerOptions:             []util.ExecuteLoggerOption{util.WithSkipCommandLogs()},
	})
}

// QueryConn runs a SQL script on an ephemeral session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryEphemeralConn(ctx, statement, odriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	}, spannerQueryResultOptions(queryContext.Explain))
}

func spannerQueryResultOptions(isExplainRequest bool) util.QueryResultOptions {
	return util.QueryResultOptions{
		ClearExplainPlan: isExplainRequest,
		TypedExplainPlan: true,
		ConvertValue:     toRowValue,
	}
}

func toQueryResults(results []*odriver.Result, maxResultBytes int64, isExplainRequest bool) []*v1pb.QueryResult {
	return util.OmniResultsToQueryResultsWith(results, maxResultBytes, spannerQueryResultOptions(isExplainRequest))
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

// SyncInstance lists the instance's GoogleSQL databases, omitting
// PostgreSQL-dialect databases whose syntax Bytebase's SPANNER parser cannot read.
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

func isEmulator(params map[string]string) bool {
	switch strings.ToLower(params[spannerdriver.ParamEmulator]) {
	case "1", "true", "yes":
		return true
	}
	return false
}
