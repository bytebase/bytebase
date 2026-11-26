package drivers

import (
	"context"

	databricksdriver "github.com/bytebase/omni/databricks/driver"
	odriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var databricksSpec = spec{
	open: func(ctx context.Context, _ *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		return databricksdriver.Open(ctx, databricksConfig(config))
	},
	adapter: util.AdapterOptions{EagerSession: true},
	// Databricks has no transaction control a script can issue, so the
	// directive is stripped and its mode not passed.
	execute: util.ExecuteConfig{
		StripDirectivesOnly: true,
		PassCreateDatabase:  true,
		PassMaxRetries:      true,
		ShiftCommandRanges:  true,
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryHeldConn(
			ctx,
			statement,
			util.OmniQueryOptions(queryContext, true, queryContext.Option.GetExplainFormat().String()),
			util.SessionQueryOptions{},
			util.QueryResultOptions{},
		)
	},
}

func databricksConfig(config db.ConnectionConfig) odriver.Config {
	ds := config.DataSource
	cfg := odriver.Config{
		Host:     ds.GetHost(),
		Database: config.ConnectionContext.DatabaseName,
		// The product stores the personal access token in the authentication
		// private key; a token the caller resolved wins when there is none.
		Password: ds.GetAuthenticationPrivateKey(),
		Params: map[string]string{
			databricksdriver.ParamWarehouseID: ds.GetWarehouseId(),
		},
	}
	if cfg.Password == "" {
		cfg.Password = config.Password
	}
	return cfg
}
