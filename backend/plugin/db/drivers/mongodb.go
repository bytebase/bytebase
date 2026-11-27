package drivers

import (
	"context"
	"strings"

	odriver "github.com/bytebase/omni/driver"
	omnimongo "github.com/bytebase/omni/mongo/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var mongodbSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, options, err := mongodbConfig(config)
		if err != nil {
			return nil, err
		}
		// The tunnel needs the client Open would otherwise leave unclosed, so
		// the dial is set up here.
		if err := util.SetupSSHDial(config.DataSource, &cfg, &d.OmniAdapter, true); err != nil {
			return nil, err
		}
		return omnimongo.Open(ctx, cfg, options)
	},
	adapter: util.AdapterOptions{WithInstanceMetadata: true},
	// MongoDB runs one statement at a time, so no transaction option applies.
	execute: util.ExecuteConfig{PassMaxRetries: true},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryEphemeralConn(ctx, statement, util.OmniQueryOptions(queryContext, true, ""), util.QueryResultOptions{})
	},
}

func mongodbConfig(config db.ConnectionConfig) (odriver.Config, omnimongo.Options, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return odriver.Config{}, omnimongo.Options{}, errors.New("mongodb: a data source is required")
	}
	cfg := odriver.Config{
		Host:     mongodbHost(dataSource),
		Port:     dataSource.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		User:     dataSource.GetUsername(),
		Password: config.Password,
		Params:   mongodbParams(dataSource),
		TLS:      util.HostTLSFunc(dataSource),
	}
	return cfg, omnimongo.Options{SRV: dataSource.GetSrv()}, nil
}

// mongodbHost is the connection string's authority: the data source's host
// with its port, then every additional address with its own, because
// Config.Port is appended only to a single host that has none.
func mongodbHost(dataSource *storepb.DataSource) string {
	host := mongodbHostPort(dataSource.GetHost(), dataSource.GetPort())
	for _, address := range dataSource.GetAdditionalAddresses() {
		host += "," + mongodbHostPort(address.GetHost(), address.GetPort())
	}
	return host
}

func mongodbHostPort(host, port string) string {
	if port != "" {
		return joinHostPort(host, port)
	}
	bare := strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	if strings.Count(bare, ":") >= 2 {
		return "[" + bare + "]"
	}
	return host
}

// mongodbParams maps the data source's connection settings onto connection
// string options. The driver defaults authSource to admin and appName to
// "bytebase".
func mongodbParams(dataSource *storepb.DataSource) map[string]string {
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
		if key != "" {
			params[key] = value
		}
	}
	if len(params) == 0 {
		return nil
	}
	return params
}
