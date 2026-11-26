package drivers

import (
	"context"

	bqdriver "github.com/bytebase/omni/bigquery/driver"
	odriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var bigquerySpec = spec{
	open: func(ctx context.Context, _ *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		return bqdriver.Open(ctx, bigqueryConfig(config))
	},
	adapter: util.AdapterOptions{NotFoundResource: "dataset", EagerSession: true},
	// A multi-statement script runs as one job whose status carries no
	// per-statement row count, so the log holds one command spanning it.
	execute: util.ExecuteConfig{PassMaxRetries: true},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryHeldConn(ctx, statement, util.OmniQueryOptions(queryContext, false, ""), util.SessionQueryOptions{}, util.QueryResultOptions{})
	},
}

func bigqueryConfig(config db.ConnectionConfig) odriver.Config {
	ds := config.DataSource
	cfg := odriver.Config{
		Host:     ds.GetHost(),
		Port:     ds.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		Password: config.Password,
		Params: map[string]string{
			bqdriver.ParamProject: ds.GetProjectId(),
		},
	}
	// A caller that resolved no credential leaves the data source's, which
	// holds the service account's JSON.
	if cfg.Password == "" {
		if credential := ds.GetGcpCredential(); credential != nil {
			cfg.Password = credential.GetContent()
		}
	}
	return cfg
}
