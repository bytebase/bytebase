// Package redshift is Bytebase's Redshift driver, adapting
// github.com/bytebase/omni/redshift/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - Only ParamDatashare is forwarded in Params; extra connection parameters are unused.
//   - TLS is configured only when a client certificate is provided; UseSsl alone
//     leaves libpq's default negotiation in place.
//   - Dump is a no-op because omni's Redshift driver implements no Dumper.
package redshift

import (
	"context"
	"io"

	omndriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	redshiftdriver "github.com/bytebase/omni/redshift/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_REDSHIFT, newDriver)
}

// Driver is Bytebase's Redshift driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open resolves the data source's network and TLS path into the contract's
// Config and opens the driver. It reaches the server only on Ping.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if err := d.OpenWith(ctx, util.AdapterOptions{
		WithInstanceMetadata: true,
		NotFoundResource:     "database",
		DatabaseName:         config.ConnectionContext.DatabaseName,
	}, func(adapter *util.OmniAdapter) (omndriver.Driver, error) {
		cfg, err := redshiftConfig(config)
		if err != nil {
			return nil, err
		}
		if err := util.SetupSSHDial(config.DataSource, &cfg, adapter, true); err != nil {
			return nil, err
		}
		return redshiftdriver.Open(ctx, cfg)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// redshiftConfig maps the data source onto the contract's Config, leaving the
// network and TLS path to Open.
func redshiftConfig(config db.ConnectionConfig) (omndriver.Config, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return omndriver.Config{}, errors.New("a data source is required")
	}
	if dataSource.GetUsername() == "" {
		return omndriver.Config{}, errors.New("user must be set")
	}
	if err := util.ValidateTLSCertKeyPair(dataSource); err != nil {
		return omndriver.Config{}, err
	}

	// An empty Database leaves the contract's own default, dev.
	cfg := omndriver.Config{
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
			return omndriver.Config{}, err
		}
		cfg.TLS = tlsFn
	}
	return cfg, nil
}

// Dump writes nothing: the contract has no renderer for a Redshift snapshot, and
// the driver this replaced wrote nothing either.
func (*Driver) Dump(context.Context, io.Writer, *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}

// CountAffectedRows returns the planner's estimate of the rows the statement
// modifies, which plancheck's affected-row limit reads.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.Driver.(db.RowEstimator).CountAffectedRows(ctx, statement)
}

// Execute runs the script and reports each step to the task run log.
// Isolation level and MaximumRetries are unused because Redshift runs in
// SERIALIZABLE mode without driver-level retries.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective:     true,
		SkipDirectiveOnCreateDatabase: true,
		PassCreateDatabase:            true,
		LoggerOptions:                 []util.ExecuteLoggerOption{util.WithOmitPerResultAffectedRows()},
	})
}

// QueryConn runs the script on the held session, one result per statement.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(ctx, statement, util.OmniQueryOptions(queryContext, true, ""), util.SessionQueryOptions{}, util.QueryResultOptions{})
}
