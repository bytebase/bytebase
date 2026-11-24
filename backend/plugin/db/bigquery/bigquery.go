// Package bigquery is Bytebase's BigQuery driver, adapting
// github.com/bytebase/omni/bigquery/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - Project ID is passed as ParamProject; dataset is the opened database.
//   - Credential defaults to DataSource.GcpCredential when Config.Password is empty.
//   - TLS, Dial, and extra connection parameters are unused (client library owns transport).
//   - Execute runs the script as one job without parsing transaction directives.
//   - SyncInstance reports datasets only (no version or roles).
package bigquery

import (
	"context"

	bqdriver "github.com/bytebase/omni/bigquery/driver"
	odriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_BIGQUERY, newDriver)
}

// Driver is Bytebase's BigQuery driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a BigQuery driver targeting the configured project and dataset.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	d := &Driver{}
	if err := d.OpenWith(ctx, util.AdapterOptions{
		NotFoundResource: "dataset",
		DatabaseName:     config.ConnectionContext.DatabaseName,
		EagerSession:     true,
	}, func(*util.OmniAdapter) (odriver.Driver, error) {
		return bqdriver.Open(ctx, bigQueryConfig(config))
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// bigQueryConfig resolves everything Bytebase knows about reaching the project
// into the plain values bigquery/driver takes.
func bigQueryConfig(config db.ConnectionConfig) odriver.Config {
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
	// A caller that resolved no credential of its own leaves the data source's,
	// which is where the product stores the service account's JSON.
	if cfg.Password == "" {
		if credential := ds.GetGcpCredential(); credential != nil {
			cfg.Password = credential.GetContent()
		}
	}
	return cfg
}

// Execute sends the whole script as one job. BigQuery's scripting runs a
// multi-statement script as one job whose status states the script and carries
// no per-statement row count, so the task run log holds one command spanning the
// script.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		PassMaxRetries: true,
	})
}

// QueryConn runs each statement of the script as its own job (or dry run when
// Explain is set) and returns one result per statement.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(ctx, statement, util.OmniQueryOptions(queryContext, false, ""), util.SessionQueryOptions{}, util.QueryResultOptions{})
}

// SyncInstance syncs the project's datasets. No version and no roles are
// reported, because BigQuery states neither.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.Driver.(odriver.Syncer).SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return &db.InstanceMetadata{Databases: instance.Databases}, nil
}
