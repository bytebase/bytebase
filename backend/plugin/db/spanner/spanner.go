// Package spanner is Bytebase's Cloud Spanner driver, adapting
// github.com/bytebase/omni/spanner/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - SyncInstance filters out PostgreSQL-dialect databases via spannerdriver.Options{GoogleSQLOnly: true} because Bytebase's SPANNER parser is GoogleSQL-only.
//   - Emulator connections drop stored GCP credentials because Google's client rejects credentials combined with WithoutAuthentication.
//   - Execute logs transaction control entries only (per-statement command logs are skipped).
package spanner

import (
	"context"
	"strings"

	odriver "github.com/bytebase/omni/driver"
	spannerdriver "github.com/bytebase/omni/spanner/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver = (*Driver)(nil)

	openSpannerDriver = func(ctx context.Context, cfg odriver.Config, opts ...spannerdriver.Options) (odriver.Driver, error) {
		return spannerdriver.Open(ctx, cfg, opts...)
	}
)

func init() {
	db.Register(storepb.Engine_SPANNER, newDriver)
}

// Driver is the Spanner driver.
type Driver struct {
	util.OmniAdapter
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
	d := &Driver{}
	if err := d.OpenWith(ctx, util.AdapterOptions{
		NotFoundResource: "database",
		DatabaseName:     config.ConnectionContext.DatabaseName,
	}, func(*util.OmniAdapter) (odriver.Driver, error) {
		return openSpannerDriver(ctx, cfg, spannerdriver.Options{GoogleSQLOnly: true})
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

func isEmulator(params map[string]string) bool {
	switch strings.ToLower(params[spannerdriver.ParamEmulator]) {
	case "1", "true", "yes":
		return true
	}
	return false
}
