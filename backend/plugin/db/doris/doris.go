// Package doris is Bytebase's Apache Doris driver: an adapter over omni's
// doris/driver that maps Bytebase's connection config onto the driver contract
// and the contract's results back into the product's protos. The conversation
// with the server — sync queries, statement execution, result decoding, the
// server-stated dump — is the driver's; what stays here is the path to it (SSH
// tunnel, TLS material) and the product's types.
package doris

import (
	"context"

	dorisdriver "github.com/bytebase/omni/doris/driver"
	omnidriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_DORIS, newDriver)
}

// Driver is Bytebase's Apache Doris driver.
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
		return dorisdriver.Open(ctx, cfg)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// Execute runs a SQL script. Doris accepts no DDL inside a transaction and
// commits each data change independently.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.MySQLForkExecuteConfig)
}

// QueryConn queries a SQL statement on the driver's held session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(ctx, statement, util.MySQLForkQueryOptions(queryContext), util.SessionQueryOptions{}, util.MySQLForkQueryResultOptions())
}
