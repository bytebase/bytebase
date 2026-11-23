// Package bigquery is Bytebase's BigQuery driver: an adapter over
// github.com/bytebase/omni/bigquery/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos. The job conversation, the statement split, the result
// decode, the value mapping, the dataset sync queries and the dry run belong to
// the omni driver.
//
// The mapping the adapter owns:
//
//   - The Google Cloud project is the data source's project ID, passed as
//     driver.Config.Params["project"]. The dataset is the database the driver
//     was opened on, which every statement's job defaults to and SyncDBSchema
//     reads.
//   - The credential is the data source's GCP credential JSON. A caller that
//     resolved one itself passes it as Config.Password, and that one wins.
//     Empty leaves the Google authentication library's default chain, which is
//     what the previous driver did with no credential.
//   - Config.Host and Config.Port are the endpoint override, for a Private
//     Service Connect endpoint or a proxy. TLS and Dial are not set: the client
//     library owns its transport and holds no dialer, so a tunnelled data
//     source is not honoured, which is also what the previous driver did. The
//     caller's extra connection parameters are not read, as before.
//   - GetDB returns nil: BigQuery has no database/sql handle. Every Bytebase
//     caller already tolerates nil there.
//   - Execute sends the whole script as one job and reports one command
//     spanning the script, before and after the job, which is the shape the
//     previous driver logged. A `-- txn-mode` directive is not parsed: BigQuery
//     has no multi-statement transaction, and the comment reaches the job as
//     part of the script, as it did before.
//   - QueryConn runs on one session the adapter holds from Open to Close. The
//     session carries no state — every statement is a job of its own.
//   - Dump writes nothing: googlesql has no catalog that renders a synced
//     dataset back into DDL, and the previous driver wrote nothing either.
//   - SyncInstance reports the project's datasets and no version and no roles:
//     BigQuery states neither.
package bigquery

import (
	"context"

	bqdriver "github.com/bytebase/omni/bigquery/driver"
	odriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

// Open opens a BigQuery driver. It must connect to a specific dataset.
// If the dataset isn't provided, part of the driver cannot function.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg := bigQueryConfig(config)
	impl, err := bqdriver.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	// BigQuery has no session state a statement can set, because every statement
	// runs as a job of its own; the session is the driver. It is opened here so
	// the driver holds one from Open to Close, as the contract asks.
	session, err := impl.Session(ctx)
	if err != nil {
		_ = impl.Close()
		return nil, err
	}
	d := &Driver{
		OmniAdapter: util.OmniAdapter{
			Driver:           impl,
			NotFoundResource: "dataset",
			DatabaseName:     config.ConnectionContext.DatabaseName,
		},
	}
	d.SetSession(session)
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

// SyncInstance syncs the project's datasets. No version and no roles are
// reported, because BigQuery states neither.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.Driver.(odriver.Syncer).SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return &db.InstanceMetadata{Databases: instance.Databases}, nil
}
