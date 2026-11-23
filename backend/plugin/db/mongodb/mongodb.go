// Package mongodb is Bytebase's MongoDB driver, adapting
// github.com/bytebase/omni/mongo/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - Bytebase registers this engine as MONGODB while omni's tree is mongo/driver.
//   - QueryConn opens one ephemeral session per call (MongoDB keeps no session state).
//   - Execute runs each mongosh statement independently without transaction control.
package mongodb

import (
	"context"
	"net"
	"strings"

	omnidriver "github.com/bytebase/omni/driver"
	omnimongo "github.com/bytebase/omni/mongo/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_MONGODB, newDriver)
}

// Driver is Bytebase's MongoDB driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	driverConfig, options, err := toDriverConfig(config)
	if err != nil {
		return nil, err
	}
	d := &Driver{}
	if err := d.OpenWith(ctx, util.AdapterOptions{WithInstanceMetadata: true}, func(adapter *util.OmniAdapter) (omnidriver.Driver, error) {
		if err := util.SetupSSHDial(config.DataSource, &driverConfig, adapter, true); err != nil {
			return nil, err
		}
		return omnimongo.Open(ctx, driverConfig, options)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// toDriverConfig maps a connection config onto the contract's config and
// options. The SSH Dial is Open's, because the tunnel needs the client Open
// would otherwise leave unclosed.
func toDriverConfig(config db.ConnectionConfig) (omnidriver.Config, omnimongo.Options, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return omnidriver.Config{}, omnimongo.Options{}, errors.New("mongodb: a data source is required")
	}
	driverConfig := omnidriver.Config{
		Host:     connectionHost(dataSource),
		Port:     dataSource.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		User:     dataSource.GetUsername(),
		Password: config.Password,
		Params:   connectionParams(dataSource),
		TLS:      util.HostTLSFunc(dataSource),
	}
	return driverConfig, omnimongo.Options{SRV: dataSource.GetSrv()}, nil
}

// connectionHost is the connection string's authority: the data source's host
// with its port, then every additional address with its own. Config.Port is
// appended to a single host that has none, so a host list carries each
// address's port itself.
func connectionHost(dataSource *storepb.DataSource) string {
	host := formatHostPort(dataSource.GetHost(), dataSource.GetPort())
	for _, address := range dataSource.GetAdditionalAddresses() {
		host += "," + formatHostPort(address.GetHost(), address.GetPort())
	}
	return host
}

func formatHostPort(host, port string) string {
	bare := strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	if port != "" {
		return net.JoinHostPort(bare, port)
	}
	if strings.Count(bare, ":") >= 2 {
		return "[" + bare + "]"
	}
	return host
}

// connectionParams maps the data source's connection settings onto connection
// string options. authSource defaults to admin in the driver, appName to
// "bytebase".
func connectionParams(dataSource *storepb.DataSource) map[string]string {
	params := map[string]string{}
	if authDatabase := dataSource.GetAuthenticationDatabase(); authDatabase != "" {
		params["authSource"] = authDatabase
	}
	if replicaSet := dataSource.GetReplicaSet(); replicaSet != "" {
		params["replicaSet"] = replicaSet
	}
	if dataSource.GetDirectConnection() {
		params["directConnection"] = "true"
	}
	for key, value := range dataSource.GetExtraConnectionParameters() {
		if key == "" {
			continue
		}
		params[key] = value
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

// Execute runs the script's statements in order, reporting each step to the
// task run log. Any transaction option is irrelevant: MongoDB runs one
// statement at a time here.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		PassMaxRetries: true,
	})
}

// QueryConn runs the script's statements on an ephemeral session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryEphemeralConn(ctx, statement, util.OmniQueryOptions(queryContext, true, ""), util.QueryResultOptions{})
}
