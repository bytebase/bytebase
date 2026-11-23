// Package hive is Bytebase's Hive driver, adapting
// github.com/bytebase/omni/hive/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - SSH data sources are refused because gohive v2 does not expose its dialer.
//   - Kerberos bootstrap writes process-wide env/files, so util.Lock is held through Open and Ping.
//   - Execute always runs in AutoCommit mode because HiveServer2 4.0 rejects BEGIN/COMMIT/ROLLBACK.
//   - QueryContext.Limit is applied client-side on non-EXPLAIN results (Hive has no parser to inject LIMIT).
//   - Dump emits server SHOW CREATE statements rather than rendering synced metadata.
package hive

import (
	"context"
	"fmt"

	odriver "github.com/bytebase/omni/driver"
	hivedriver "github.com/bytebase/omni/hive/driver"
	"github.com/pkg/errors"

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
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open connects to HiveServer2 and verifies reachability with Ping. When
// Kerberos is configured, util.Lock is held across bootstrap, open, and ping.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if _, ok := config.DataSource.GetSaslConfig().GetMechanism().(*storepb.SASLConfig_KrbConfig); ok {
		util.Lock.Lock()
		defer util.Lock.Unlock()
	}
	cfg, err := hiveConfig(config)
	if err != nil {
		return nil, err
	}
	d := &Driver{}
	if err := d.OpenWith(ctx, util.AdapterOptions{}, func(*util.OmniAdapter) (odriver.Driver, error) {
		impl, err := hivedriver.Open(ctx, cfg)
		if err != nil {
			return nil, err
		}
		if err := impl.Ping(ctx); err != nil {
			_ = impl.Close()
			return nil, errors.Wrap(err, "failed to ping hive server")
		}
		return impl, nil
	}); err != nil {
		return nil, err
	}
	return d, nil
}

func hiveConfig(config db.ConnectionConfig) (odriver.Config, error) {
	ds := config.DataSource
	if ds.GetSshHost() != "" {
		return odriver.Config{}, errors.Errorf("an SSH data source is not supported for Hive: forward a local port to %s and use the forwarded host and port instead", serverAddress(ds))
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

func serverAddress(ds *storepb.DataSource) string {
	port := ds.GetPort()
	if port == "" {
		port = "10000"
	}
	return ds.GetHost() + ":" + port
}

// Execute runs the script's statements in AutoCommit mode without parsing
// transaction directives.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		DefaultTransaction: odriver.AutoCommit,
		PassMaxRetries:     true,
	})
}

// QueryConn runs a script on an ephemeral session with client-side row limiting
// for non-EXPLAIN queries.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryEphemeralConn(ctx, statement, odriver.QueryOptions{
		Schema:         queryContext.Schema,
		Explain:        queryContext.Explain,
		ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	}, hiveQueryResultOptions(queryContext))
}

func hiveQueryResultOptions(queryContext db.QueryContext) util.QueryResultOptions {
	limit := 0
	if !queryContext.Explain && queryContext.Limit > 0 {
		limit = queryContext.Limit
	}
	return util.QueryResultOptions{
		SkipPlan:             true,
		SkipMessages:         true,
		UppercaseColumnTypes: true,
		TruncatedWins:        true,
		Limit:                limit,
		ConvertValue:         toRowValue,
	}
}

func toRowValue(value odriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case odriver.Null:
		return util.NullRowValue
	case odriver.Bool, odriver.Bytes, odriver.Float32, odriver.Float64,
		odriver.Int32, odriver.Int64, odriver.Uint32, odriver.Uint64, odriver.String:
		return util.OmniRowValue(value)
	default:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value)}}
	}
}
