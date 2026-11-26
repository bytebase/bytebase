package drivers

import (
	"context"

	"github.com/pkg/errors"

	odriver "github.com/bytebase/omni/driver"
	mysqld "github.com/bytebase/omni/mysql/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var mysqlSpec = mysqlFamilySpec(func(ctx context.Context, cfg odriver.Config) (odriver.Driver, error) {
	return mysqld.Open(ctx, cfg)
})

// mysqlFamilySpec is the spec MySQL and MariaDB share: the same data source
// paths (Cloud SQL IAM, RDS IAM, SSH and TLS) and the same probe.
func mysqlFamilySpec(open func(context.Context, odriver.Config) (odriver.Driver, error)) spec {
	return spec{
		open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
			cfg, err := mysqlFamilyConfig(ctx, d, config)
			if err != nil {
				return nil, err
			}
			return open(ctx, cfg)
		},
		adapter: util.AdapterOptions{WithInstanceMetadata: true},
		// CreateDatabase and MaximumRetries are not passed: mapping them would
		// change a create-database script's transaction shape and retry a script
		// the product does not retry.
		execute: util.ExecuteConfig{
			ParseTransactionDirective: true,
			PassIsolation:             true,
		},
		query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
			queryOpts := util.OmniQueryOptions(
				queryContext,
				false,
				util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON),
			)
			// The product reads a zero cap as "no rows", and the contract reads
			// it as "no cap".
			queryOpts.MaxResultBytes = util.PositiveMaxResultBytes(queryContext.MaximumSQLResultSize)
			return d.QueryHeldConn(ctx, statement, queryOpts, util.SessionQueryOptions{}, util.QueryResultOptions{ClearExplainPlan: queryContext.Explain})
		},
		// A MySQL-family probe is an EXPLAIN, so the autocommit and transaction
		// shapes answer the same.
		probe: func(*Driver) util.OmniProbeOptions {
			return util.OmniProbeOptions{
				BeginStatement:               "START TRANSACTION",
				RollbackStatement:            "ROLLBACK",
				AllowAutocommitPreExecutions: true,
				SkipEmptyPreExecutions:       true,
				TakeLastResult:               true,
			}
		},
	}
}

func mysqlFamilyConfig(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Config, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return odriver.Config{}, errors.New("a data source is required")
	}
	cfg := odriver.Config{
		Host:            dataSource.Host,
		Port:            dataSource.Port,
		Database:        config.ConnectionContext.DatabaseName,
		User:            dataSource.Username,
		Password:        config.Password,
		ApplicationName: "bytebase",
		EngineVersion:   config.ConnectionContext.EngineVersion,
		Params:          dataSource.GetExtraConnectionParameters(),
	}
	switch dataSource.GetAuthenticationType() {
	case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		if _, err := util.SetupCloudSQLDial(ctx, config, &cfg, &d.OmniAdapter); err != nil {
			return odriver.Config{}, err
		}
	case storepb.DataSource_AWS_RDS_IAM:
		if err := util.ConfigureRDSMySQLIAM(ctx, config, &cfg); err != nil {
			return odriver.Config{}, err
		}
	default:
		if err := util.SetupSSHDial(dataSource, &cfg, &d.OmniAdapter, false); err != nil {
			return odriver.Config{}, err
		}
		tlsFn, err := util.SQLStaticTLSFunc(dataSource)
		if err != nil {
			return odriver.Config{}, err
		}
		cfg.TLS = tlsFn
	}
	return cfg, nil
}
