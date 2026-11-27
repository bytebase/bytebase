package drivers

import (
	"context"
	"io"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	redshiftdriver "github.com/bytebase/omni/redshift/driver"
	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var redshiftSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, err := redshiftConfig(config)
		if err != nil {
			return nil, err
		}
		if err := util.SetupSSHDial(config.DataSource, &cfg, &d.OmniAdapter, true); err != nil {
			return nil, err
		}
		return redshiftdriver.Open(ctx, cfg)
	},
	adapter: util.AdapterOptions{WithInstanceMetadata: true, NotFoundResource: "database"},
	// Isolation and MaximumRetries are not passed: Redshift runs SERIALIZABLE
	// and the driver does not retry.
	execute: util.ExecuteConfig{
		ParseTransactionDirective:     true,
		SkipDirectiveOnCreateDatabase: true,
		PassCreateDatabase:            true,
		LoggerOptions:                 []util.ExecuteLoggerOption{util.WithOmitPerResultAffectedRows()},
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryHeldConn(ctx, statement, util.OmniQueryOptions(queryContext, true, ""), util.SessionQueryOptions{}, util.QueryResultOptions{})
	},
	// The contract has no renderer for a Redshift snapshot, so a dump writes
	// nothing.
	dump: func(context.Context, *Driver, io.Writer, *metadatapb.DatabaseSchemaMetadata) error {
		return nil
	},
}

func redshiftConfig(config db.ConnectionConfig) (odriver.Config, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return odriver.Config{}, errors.New("a data source is required")
	}
	if dataSource.GetUsername() == "" {
		return odriver.Config{}, errors.New("user must be set")
	}
	if err := util.ValidateTLSCertKeyPair(dataSource); err != nil {
		return odriver.Config{}, err
	}
	// An empty Database leaves the contract's default, dev.
	cfg := odriver.Config{
		Host:          dataSource.GetHost(),
		Port:          dataSource.GetPort(),
		Database:      util.ResolveDatabaseName(config),
		User:          dataSource.GetUsername(),
		Password:      config.Password,
		ReadOnly:      config.ConnectionContext.ReadOnly,
		EngineVersion: config.ConnectionContext.EngineVersion,
	}
	if config.ConnectionContext.DataShare {
		cfg.Params = map[string]string{redshiftdriver.ParamDatashare: "true"}
	}
	if dataSource.GetSslCert() != "" {
		tlsFn, err := util.StaticTLSFunc(dataSource)
		if err != nil {
			return odriver.Config{}, err
		}
		cfg.TLS = tlsFn
	}
	return cfg, nil
}
