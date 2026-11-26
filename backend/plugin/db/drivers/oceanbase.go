package drivers

import (
	"context"
	"strings"
	"time"

	odriver "github.com/bytebase/omni/driver"
	oceanbasedriver "github.com/bytebase/omni/oceanbase/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var oceanbaseSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, err := oceanbaseConfig(ctx, d, config)
		if err != nil {
			return nil, err
		}
		return oceanbasedriver.Open(ctx, cfg)
	},
	adapter: util.AdapterOptions{WithInstanceMetadata: true},
	// CreateDatabase and MaximumRetries are not passed, as for MySQL.
	execute: util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassIsolation:             true,
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryHeldConn(
			ctx,
			statement,
			util.OmniQueryOptions(queryContext, false, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_JSON)),
			util.SessionQueryOptions{WrapOpenError: "failed to open a session"},
			util.QueryResultOptions{ClearExplainPlan: queryContext.Explain},
		)
	},
	// Autocommit runs the statement alone, for a rule whose statement OceanBase
	// refuses inside a transaction.
	probe: func(*Driver) util.OmniProbeOptions {
		return util.OmniProbeOptions{
			BeginStatement:           "START TRANSACTION",
			RollbackStatement:        "ROLLBACK",
			RollbackWithoutCancel:    true,
			RollbackTimeout:          5 * time.Second,
			SkipEmptyPreExecutions:   true,
			CheckPreExecutionResults: true,
			FormatCell:               oceanbaseProbeCell,
		}
	},
}

// oceanbaseConfig resolves one OceanBase tenant's data source into the plain
// values oceanbase/driver's go-sql-driver connector takes.
func oceanbaseConfig(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Config, error) {
	ds := config.DataSource
	cfg := odriver.Config{
		Host:            ds.GetHost(),
		Port:            ds.GetPort(),
		Database:        config.ConnectionContext.DatabaseName,
		User:            ds.GetUsername(),
		Password:        config.Password,
		ApplicationName: "bytebase",
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}
	switch ds.GetAuthenticationType() {
	case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		if _, err := util.SetupCloudSQLDial(ctx, config, &cfg, &d.OmniAdapter); err != nil {
			return odriver.Config{}, err
		}
		cfg.Params = map[string]string{"parseTime": "true"}
	case storepb.DataSource_AWS_RDS_IAM:
		if err := util.ConfigureRDSMySQLIAM(ctx, config, &cfg); err != nil {
			return odriver.Config{}, err
		}
	default:
		params, err := oceanbaseParams(ds.GetExtraConnectionParameters())
		if err != nil {
			return odriver.Config{}, err
		}
		cfg.Params = params
		if err := util.SetupSSHDial(ds, &cfg, &d.OmniAdapter, false); err != nil {
			return odriver.Config{}, err
		}
		cfg.TLS = util.LazyTLSFunc(ds)
	}
	return cfg, nil
}

// oceanbaseParams passes the data source's own parameters through, refusing
// the one that disables go-sql-driver's local-infile allowlist.
func oceanbaseParams(extra map[string]string) (map[string]string, error) {
	params := make(map[string]string, len(extra))
	for key, value := range extra {
		if strings.EqualFold(strings.TrimSpace(key), "allowallfiles") {
			return nil, errors.Errorf("connection parameter %q is not allowed for security reasons", key)
		}
		params[key] = value
	}
	return params, nil
}

func oceanbaseProbeCell(value odriver.Value) any {
	switch value.Kind {
	case odriver.Bytes:
		return string(value.Bytes)
	case odriver.Int32:
		return int32(value.Int64)
	case odriver.Uint32:
		return uint32(value.Uint64)
	case odriver.Structured:
		return nil
	case odriver.Timestamp, odriver.TimestampTZ:
		return value.Time.Format(time.RFC3339Nano)
	default:
		return util.DefaultProbeCell(value)
	}
}
