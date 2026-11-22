// Package starrocks is Bytebase's StarRocks driver: an adapter over omni's
// starrocks/driver that maps Bytebase's connection config onto the driver
// contract and the contract's results back into the product's protos. The
// conversation with the server — sync queries, statement execution, result
// decoding, the server-stated dump — is the driver's; what stays here is the
// path to it (SSH tunnel, TLS material) and the product's types.
//
// GetDB returns nil: the driver owns its pool and hands out sessions, so a
// caller that needs a *sql.DB is told the engine exposes none. gh-ost's
// prerequisite probe is the one caller that asks; it refuses an engine outside
// ghost.SupportsEngine and never reaches this driver's nil pool.
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
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
	opened := false
	defer func() {
		if !opened {
			_ = d.Close(ctx)
		}
	}()

	d.OmniAdapter = util.OmniAdapter{
		WithInstanceMetadata: true,
		NotFoundResource:     "database",
		DatabaseName:         config.ConnectionContext.DatabaseName,
	}

	cfg, err := driverConfig(config, &d.OmniAdapter)
	if err != nil {
		return nil, err
	}
	impl, err := starrocksdriver.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	d.Driver = impl
	opened = true
	return d, nil
}

// driverConfig resolves everything Bytebase knows about reaching the server into
// the plain values the driver takes. The mapping is the MySQL family's: the same
// host/port or unix socket, the same tunnel as a Dial, and the same TLS material
// the old DSN registered with go-sql-driver.
func driverConfig(config db.ConnectionConfig, adapter *util.OmniAdapter) (omnidriver.Config, error) {
	ds := config.DataSource
	cfg := omnidriver.Config{
		Host:     ds.GetHost(),
		Port:     ds.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		User:     ds.GetUsername(),
		Password: config.Password,
		// The driver prefixes a statement it reads as a query with
		// "/*app=<name>*/", which is how Bytebase names its editor sessions.
		ApplicationName: "bytebase",
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}

	if err := util.SetupSSHDial(ds, &cfg, adapter, false); err != nil {
		return cfg, err
	}
	tlsFn, err := util.StaticTLSFunc(ds)
	if err != nil {
		return cfg, errors.Wrap(err, "sql: tls config error")
	}
	cfg.TLS = tlsFn
	return cfg, nil
}
