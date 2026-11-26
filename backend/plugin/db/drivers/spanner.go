package drivers

import (
	"context"
	"strings"

	odriver "github.com/bytebase/omni/driver"
	spannerdriver "github.com/bytebase/omni/spanner/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var spannerSpec = spec{
	open: func(ctx context.Context, _ *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, err := spannerConfig(config)
		if err != nil {
			return nil, err
		}
		return spannerdriver.Open(ctx, cfg)
	},
	adapter: util.AdapterOptions{NotFoundResource: "database"},
	execute: util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassCreateDatabase:        true,
		PassMaxRetries:            true,
		LoggerOptions:             []util.ExecuteLoggerOption{util.WithSkipCommandLogs()},
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryEphemeralConn(ctx, statement, odriver.QueryOptions{
			Limit:          queryContext.Limit,
			Explain:        queryContext.Explain,
			MaxResultBytes: queryContext.MaximumSQLResultSize,
		}, util.QueryResultOptions{
			ClearExplainPlan: queryContext.Explain,
			TypedExplainPlan: true,
			ConvertValue:     spannerRowValue,
		})
	},
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
	switch strings.ToLower(params[spannerdriver.ParamEmulator]) {
	case "1", "true", "yes":
		password = ""
	default:
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

func spannerRowValue(value odriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case odriver.Float32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case odriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case odriver.Structured:
		return stringValue(value.String)
	case odriver.Bool, odriver.Bytes, odriver.Float64, odriver.Int64, odriver.String, odriver.Timestamp:
		return util.OmniRowValue(value)
	default:
		return util.NullRowValue
	}
}
