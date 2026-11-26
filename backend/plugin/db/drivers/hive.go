package drivers

import (
	"context"

	odriver "github.com/bytebase/omni/driver"
	hivedriver "github.com/bytebase/omni/hive/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// hiveSpec opens and pings HiveServer2. With Kerberos, util.Lock is held from
// the environment bootstrap until the ping answers.
var hiveSpec = spec{
	open: func(ctx context.Context, _ *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		if _, ok := config.DataSource.GetSaslConfig().GetMechanism().(*storepb.SASLConfig_KrbConfig); ok {
			util.Lock.Lock()
			defer util.Lock.Unlock()
		}
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
		return impl, nil
	},
	execute: util.ExecuteConfig{
		DefaultTransaction: odriver.AutoCommit,
		PassMaxRetries:     true,
	},
	// The row limit is applied while converting, except to EXPLAIN.
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		limit := 0
		if !queryContext.Explain && queryContext.Limit > 0 {
			limit = queryContext.Limit
		}
		return d.QueryEphemeralConn(ctx, statement, odriver.QueryOptions{
			Schema:         queryContext.Schema,
			Explain:        queryContext.Explain,
			ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON),
			MaxResultBytes: queryContext.MaximumSQLResultSize,
		}, util.QueryResultOptions{
			SkipPlan:             true,
			SkipMessages:         true,
			UppercaseColumnTypes: true,
			TruncatedWins:        true,
			Limit:                limit,
			ConvertValue:         hiveRowValue,
		})
	},
}

func hiveConfig(config db.ConnectionConfig) (odriver.Config, error) {
	ds := config.DataSource
	if ds.GetSshHost() != "" {
		port := ds.GetPort()
		if port == "" {
			port = "10000"
		}
		return odriver.Config{}, errors.Errorf("an SSH data source is not supported for Hive: forward a local port to %s:%s and use the forwarded host and port instead", ds.GetHost(), port)
	}
	params := map[string]string{}
	if t, ok := ds.GetSaslConfig().GetMechanism().(*storepb.SASLConfig_KrbConfig); ok {
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
	for key, value := range ds.GetExtraConnectionParameters() {
		params[key] = value
	}
	return odriver.Config{
		Host:     ds.GetHost(),
		Port:     ds.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		User:     ds.GetUsername(),
		Password: config.Password,
		Params:   params,
		TLS:      util.LazyTLSFunc(ds),
	}, nil
}

func hiveRowValue(value odriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case odriver.Null:
		return util.NullRowValue
	case odriver.Bool, odriver.Bytes, odriver.Float32, odriver.Float64,
		odriver.Int32, odriver.Int64, odriver.Uint32, odriver.Uint64, odriver.String:
		return util.OmniRowValue(value)
	default:
		return printedValue(value)
	}
}
