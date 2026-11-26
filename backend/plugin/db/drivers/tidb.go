package drivers

import (
	"context"
	"fmt"
	"time"

	odriver "github.com/bytebase/omni/driver"
	tidbdriver "github.com/bytebase/omni/tidb/driver"
	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var tidbSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, err := tidbConfig(d, config)
		if err != nil {
			return nil, err
		}
		return tidbdriver.Open(ctx, cfg)
	},
	adapter: util.AdapterOptions{WithInstanceMetadata: true, NotFoundResource: "database"},
	execute: util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassMaxRetries:            true,
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		// A non-positive cap means "no rows" to the product and "no cap" to the
		// contract, so it is refused rather than read every row a policy meant
		// to bound.
		if queryContext.MaximumSQLResultSize <= 0 {
			return nil, errors.New("maximum SQL result size must be positive")
		}
		return d.QueryHeldConn(
			ctx,
			statement,
			util.OmniQueryOptions(queryContext, true, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON)),
			util.SessionQueryOptions{},
			util.QueryResultOptions{ClearExplainPlan: queryContext.Explain},
		)
	},
	// A BATCH ... DRY RUN probe passes Autocommit, because TiDB refuses it
	// inside a transaction.
	probe: func(*Driver) util.OmniProbeOptions {
		return util.OmniProbeOptions{
			BeginStatement:           "START TRANSACTION",
			RollbackStatement:        "ROLLBACK",
			RollbackWithoutCancel:    true,
			SkipEmptyPreExecutions:   true,
			CheckPreExecutionResults: true,
			FormatCell:               tidbProbeCell,
		}
	},
}

func tidbConfig(d *Driver, config db.ConnectionConfig) (odriver.Config, error) {
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
		Params:          util.CloneParams(dataSource.GetExtraConnectionParameters()),
		ApplicationName: "bytebase",
		EngineVersion:   config.ConnectionContext.EngineVersion,
		TLS:             util.LazyTLSFunc(dataSource),
	}
	if err := util.SetupSSHDial(dataSource, &cfg, &d.OmniAdapter, false); err != nil {
		return odriver.Config{}, err
	}
	return cfg, nil
}

// tidbProbeCell renders a timestamp the way the session spells it, so a probe's
// cell reads as the server's text did.
func tidbProbeCell(value odriver.Value) any {
	if value.Kind != odriver.Timestamp && value.Kind != odriver.TimestampTZ {
		return util.DefaultProbeCell(value)
	}
	text := value.Time.Format(time.DateTime)
	if value.Precision > 0 {
		fraction := fmt.Sprintf("%09d", value.Time.Nanosecond())
		if int(value.Precision) < len(fraction) {
			fraction = fraction[:value.Precision]
		}
		text += "." + fraction
	}
	return text
}
