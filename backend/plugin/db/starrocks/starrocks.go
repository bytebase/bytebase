// Package starrocks is Bytebase's StarRocks driver: an adapter over omni's
// starrocks/driver that maps Bytebase's connection config onto the driver
// contract and the contract's results back into the product's protos. The
// conversation with the server — sync queries, statement execution, result
// decoding, the server-stated dump — is the driver's; what stays here is the
// path to it (SSH tunnel, TLS material) and the product's types.
//
// QueryConn opens its session once and holds it until Close, because the
// editor asks for one statement per call and a session variable one call sets
// must reach the next. The calls are serialized for the same reason one
// database connection serializes them.
//
// The measurements behind this adapter, and the behavior of the driver it
// replaced, are in #22084.
package starrocks

import (
	"context"

	omnidriver "github.com/bytebase/omni/driver"
	starrocksdriver "github.com/bytebase/omni/starrocks/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_STARROCKS, newDriver)
}

// Driver is Bytebase's StarRocks driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server: Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if err := d.OpenWith(ctx, util.AdapterOptions{
		WithInstanceMetadata: true,
		NotFoundResource:     "database",
		DatabaseName:         config.ConnectionContext.DatabaseName,
	}, func(adapter *util.OmniAdapter) (omnidriver.Driver, error) {
		cfg, err := util.MySQLFamilyDriverConfig(config, adapter)
		if err != nil {
			return nil, err
		}
		return starrocksdriver.Open(ctx, cfg)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// Execute runs a SQL script. StarRocks accepts no DDL inside a transaction and
// commits each data change independently.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.MySQLForkExecuteConfig)
}

// QueryConn queries a SQL statement on the driver's held session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(ctx, statement, util.MySQLForkQueryOptions(queryContext), util.SessionQueryOptions{}, util.MySQLForkQueryResultOptions())
}
