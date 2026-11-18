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
// The statement split, the EXPLAIN rewrite, the result decode, the cap, the
// session's schema and the dump are omni's driver's, not this adapter's, and
// several of them behave differently from the replaced driver. This PR's
// "Parity evidence" table records each difference and what it means for the
// product, and cutover_parity_test.go asserts both answers over the recorded
// workspace. Read those before changing the adapter.
package databricks

import (
	"context"
	"io"
	"strings"

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
//
// Known difference: the count is the summed num_affected_rows the warehouse
// states, and each command and its response reach the task run log. The previous
// driver returned 0 for every script and logged nothing.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// The directive is removed from the script as before; its mode reaches no
	// option, because Databricks has no transaction control a script can issue,
	// and the session's transaction mode is not something the API accepts.
	_, cleaned := base.ParseTransactionConfig(statement)
	return d.impl.Execute(ctx, cleaned, odbdriver.ExecuteOptions{
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log:            executeLogger{opts: opts, offset: skippedPrefix(statement, cleaned)},
	})
}

// skippedPrefix is the number of bytes the directive removal cut from the head
// of the caller's script. A task run log with LogCommandStatement off recovers a
// command's text from the caller's script by range, so the driver's ranges, which
// name the script it was given, are shifted back by that much.
func skippedPrefix(script, cleaned string) int32 {
	if cleaned == "" {
		return int32(len(script))
	}
	if i := strings.Index(script, cleaned); i > 0 {
		return int32(i)
	}
	return 0
}

// executeLogger reports a script's commands and their responses into the task run
// log. Databricks issues no transaction control, so there is nothing to log for
// one.
type executeLogger struct {
	opts db.ExecuteOptions
	// offset is the number of bytes the caller's script had before the text the
	// driver split, so the log's ranges name the caller's script rather than the
	// cleaned one.
	offset int32
}

func (l executeLogger) CommandExecute(command odbdriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{
		Start: int32(command.ByteStart) + l.offset,
		End:   int32(command.ByteEnd) + l.offset,
	}, command.Text)
}

func (l executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (executeLogger) TransactionControl(odbdriver.TransactionOp, error) {}

// Retry records a retried statement. Databricks' driver retries nothing today, so
// this only fires if it starts to.
func (l executeLogger) Retry(err error, attempt, _ int) {
	l.opts.LogRetryInfo(err, attempt)
}

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
