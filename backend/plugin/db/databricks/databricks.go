// Package databricks is Bytebase's Databricks driver, adapting
// github.com/bytebase/omni/databricks/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - Personal access token comes from DataSource.AuthenticationPrivateKey,
//     falling back to Config.Password; warehouse ID is passed as ParamWarehouseID.
//   - TLS, Dial, and extra connection parameters are unused (SDK owns transport).
//   - Execute strips transaction directives without setting transaction mode.
package databricks

import (
	"context"

	databricksdriver "github.com/bytebase/omni/databricks/driver"
	odbdriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_DATABRICKS, newDriver)
}

// Driver is Bytebase's Databricks driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a Databricks driver. It does not reach the workspace: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	d := &Driver{}
	if err := d.OpenWith(ctx, util.AdapterOptions{
		EagerSession: true,
	}, func(*util.OmniAdapter) (odbdriver.Driver, error) {
		return databricksdriver.Open(ctx, databricksConfig(config))
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// databricksConfig resolves everything Bytebase knows about reaching the
// workspace into the plain values databricks/driver takes.
func databricksConfig(config db.ConnectionConfig) odbdriver.Config {
	ds := config.DataSource
	cfg := odbdriver.Config{
		Host:     ds.GetHost(),
		Database: config.ConnectionContext.DatabaseName,
		// The product stores the personal access token in the data source's
		// authentication private key; a token a caller resolved itself wins when
		// the data source carries none.
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

// Execute runs the script's statements in order and returns the rows they
// affected. Databricks has no transaction control a script can issue, so the
// transaction mode the product sends changes nothing, which is what the previous
// driver did.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		StripDirectivesOnly: true,
		PassCreateDatabase:  true,
		PassMaxRetries:      true,
		ShiftCommandRanges:  true,
	})
}

// QueryConn runs each statement of the script on the held session and converts
// the results into API QueryResults.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(
		ctx,
		statement,
		util.OmniQueryOptions(queryContext, true, queryContext.Option.GetExplainFormat().String()),
		util.SessionQueryOptions{},
		util.QueryResultOptions{},
	)
}
