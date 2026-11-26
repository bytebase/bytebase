package drivers

import (
	"context"

	dorisdriver "github.com/bytebase/omni/doris/driver"
	odriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var dorisSpec = mysqlForkSpec(func(ctx context.Context, cfg odriver.Config) (odriver.Driver, error) {
	return dorisdriver.Open(ctx, cfg)
})

// mysqlForkSpec is the spec of the MySQL-protocol analytical engines, Doris
// and StarRocks. They accept no DDL inside a transaction and commit each data
// change on its own.
func mysqlForkSpec(open func(context.Context, odriver.Config) (odriver.Driver, error)) spec {
	return spec{
		open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
			cfg, err := util.MySQLFamilyDriverConfig(config, &d.OmniAdapter)
			if err != nil {
				return nil, err
			}
			return open(ctx, cfg)
		},
		adapter: util.AdapterOptions{WithInstanceMetadata: true, NotFoundResource: "database"},
		execute: util.MySQLForkExecuteConfig,
		query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
			return d.QueryHeldConn(ctx, statement, util.MySQLForkQueryOptions(queryContext), util.SessionQueryOptions{}, util.MySQLForkQueryResultOptions())
		},
	}
}
