// Package databricks is Bytebase's Databricks driver: an adapter over
// github.com/bytebase/omni/databricks/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos. The workspace conversation, the script splitter, the
// EXPLAIN rewrite, the result decode, the Unity Catalog sync and the server-DDL
// dump belong to the omni driver.
//
// The mapping the adapter owns:
//
//   - Config.Host is the workspace URL. The personal access token is the data
//     source's authentication private key, which is where the product stores it;
//     a caller that resolved a token itself passes it as Config.Password, and it
//     is used when the data source carries none. Token authentication is all
//     this driver supports, as before.
//   - The SQL warehouse is the data source's warehouse ID, passed as
//     driver.Config.Params["warehouseId"]. The catalog is the database the
//     driver was opened on: it is the session's catalog and what SyncDBSchema
//     reads.
//   - TLS and Dial are not set: the previous driver ignored an SSL data source
//     and an SSH tunnel, and the SDK's own transport verifies the workspace's
//     certificate against the system roots. The caller's extra connection
//     parameters are not read, as before.
//   - Execute sends the script's statements in order and reports each to the
//     task run log, with the rows the warehouse states in a num_affected_rows
//     column. The `-- txn-mode` directive is parsed and removed from the script
//     as before, and Databricks' Statement Execution API exposes no transaction
//     control, so the mode changes nothing.
//   - QueryConn runs on one session the adapter holds from Open to Close, and
//     passes QueryContext.Schema as the schema each statement runs in.
//   - SyncInstance reports the SQL release and the workspace's catalog names.
//     SyncDBSchema reads one catalog's schemas, tables, views, materialized views
//     and external tables.
//   - Dump writes the server's own SHOW CREATE TABLE for every object: omni has
//     no databricks catalog that renders a synced schema as Databricks DDL, so
//     the metadata argument is not read.
//
// The statement split, the EXPLAIN rewrite, the result decode, the cap, the
// session's schema and the dump are omni's driver's, not this adapter's, and
// several of them behave differently from the replaced driver. This PR's
// "Parity evidence" table records each difference and what it means for the
// product, and cutover_parity_test.go asserts both answers over the recorded
// workspace. Read those before changing the adapter.
package databricks

import (
	"context"

	databricksdriver "github.com/bytebase/omni/databricks/driver"
	odbdriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
	impl, err := databricksdriver.Open(ctx, databricksConfig(config))
	if err != nil {
		return nil, err
	}
	// The SQL Statement Execution API has no session state, so the session is the
	// driver; it is opened here so the driver holds one from Open to Close.
	session, err := impl.Session(ctx)
	if err != nil {
		_ = impl.Close()
		return nil, err
	}
	d := &Driver{
		OmniAdapter: util.OmniAdapter{Driver: impl},
	}
	d.SetSession(session)
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
