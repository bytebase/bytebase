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
//   - GetDB is not implemented. The previous driver returned nil for it, every
//     caller reaches the handle through db.SQLDB and branches on its absence, and
//     no Databricks feature read it. OwnsQuerySession is declared instead,
//     because QueryConn ignores the caller's *sql.Conn.
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
// Known differences from the driver this replaced, kept because they are the
// driver's own behavior or because reproducing the old one would reimplement it:
//
//   - The session's schema now reaches the statement. The previous driver sent
//     only the catalog and ran every unqualified name in the catalog's default
//     schema, so the editor's schema selection had no effect.
//   - A script's statements are trimmed of their trailing semicolons and leading
//     whitespace, which is the text the driver runs; the previous driver sent the
//     splitter's slice, semicolons included.
//   - A statement that returns no rows still reports its columns and does not
//     end the script. The previous driver stopped at the first statement whose
//     data array was empty, dropping that statement's result and every statement
//     after it.
//   - A failed statement reports its error on its own result and the statements
//     before it keep their results; the previous driver returned the call's error
//     with no results at all.
//   - The result-size cap is enforced and reported as Truncated, which the
//     previous driver did not do: it returned every row the warehouse answered,
//     however large.
//   - Execute returns the rows the script affected, summed from the
//     num_affected_rows columns, where the previous driver returned 0 for every
//     script, and it reports each command and its response to the task run log,
//     where the previous driver logged nothing.
//   - A statement the caller wrote as EXPLAIN carries QueryPlan{TEXT}: the
//     previous driver returned the plan's rows with no plan, so the viewer did
//     not draw them.
//   - SyncInstance lists a catalog without its schemas. The previous driver read
//     every schema and table of every catalog during an instance sync, and the
//     product reads only each catalog's name from it; SyncDBSchema reads one
//     catalog's schemas.
//   - The catalog, schema and table listings, the schemas within a catalog, the
//     tables within a schema and a view's dependency columns are in name or
//     position order, which the previous driver kept in the service's order.
//   - A statement the warehouse refuses is the SDK helper's text
//     ("FAILED: <code> <message>"), and Execute wraps it with "failed to split
//     statements"/"failed to execute statement"; the previous driver returned the
//     bare SDK error.
//   - A missing workspace, token or warehouse is refused by Open with the
//     driver's message naming the field, where the previous driver's SDK resolved
//     what it could from the environment and said "Warehouse ID must be set".
package databricks

import (
	"context"
	"io"

	databricksdriver "github.com/bytebase/omni/databricks/driver"
	odbdriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_DATABRICKS, newDriver)
}

// Driver is Bytebase's Databricks driver.
type Driver struct {
	config db.ConnectionConfig

	impl    *databricksdriver.Driver
	session odbdriver.Session
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
	return &Driver{config: config, impl: impl, session: session}, nil
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

// Close closes the session. The SDK client keeps no connection of its own.
func (d *Driver) Close(context.Context) error {
	if d.session == nil {
		return nil
	}
	err := d.session.Close()
	d.session = nil
	return err
}

// Ping asks the workspace for its root folder, which is Databricks' connectivity
// check: there is no SQL-level ping.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// OwnsQuerySession reports that QueryConn runs on the session the adapter holds
// rather than on the *sql.Conn a caller passes, which is unused.
func (*Driver) OwnsQuerySession() {}

// Execute runs the script's statements in order and returns the rows they
// affected. Databricks has no transaction control a script can issue, so the
// transaction mode the product sends changes nothing, which is what the previous
// driver did.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// The directive is removed from the script as before; its mode reaches no
	// option, because Databricks has no transaction control a script can issue,
	// and the session's transaction mode is not something the API accepts.
	_, cleaned := base.ParseTransactionConfig(statement)
	return d.impl.Execute(ctx, cleaned, odbdriver.ExecuteOptions{
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log:            executeLogger{opts: opts},
	})
}

// executeLogger reports a script's commands and their responses into the task run
// log. Databricks issues no transaction control, so there is nothing to log for
// one.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l executeLogger) CommandExecute(command odbdriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{
		Start: int32(command.ByteStart),
		End:   int32(command.ByteEnd),
	}, command.Text)
}

func (l executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (executeLogger) TransactionControl(odbdriver.TransactionOp, error) {}

func (executeLogger) Retry(error, int, int) {}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// SyncInstance syncs the SQL release and the workspace's catalog names. Roles
// stay empty: Unity Catalog exposes no role a sync reads.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
	}, nil
}

// SyncDBSchema syncs the catalog the driver was opened on.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.impl.SyncDatabase(ctx)
}

// Dump writes the server's own DDL for every object in the workspace. The
// synced metadata is not rendered and does not narrow the dump: omni has no
// databricks catalog that turns a synced schema back into Databricks DDL.
func (d *Driver) Dump(ctx context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, metadata)
}
