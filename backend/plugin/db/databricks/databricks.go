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
	"unicode"

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
// driver returned 0 for every script and logged nothing. A logged range names the
// caller's script, with the removed directive lines left out, because a task run
// log with LogCommandStatement off slices that script by the range.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// The directive is removed from the script as before; its mode reaches no
	// option, because Databricks has no transaction control a script can issue,
	// and the session's transaction mode is not something the API accepts.
	_, cleaned := base.ParseTransactionConfig(statement)
	return d.impl.Execute(ctx, cleaned, odbdriver.ExecuteOptions{
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log:            executeLogger{opts: opts, sourceRange: directiveShift(statement, cleaned)},
	})
}

// directiveShift maps the range the driver reports on the cleaned script back to
// the caller's script, which the task run log reads when LogCommandStatement is
// off. The removed directive lines are dropped from the range as well: the parser
// removes them from anywhere in the leading comment block, and a contiguous range
// that spans one would show the directive as if it had been executed.
func directiveShift(script, cleaned string) func(int, int) (int, int) {
	identity := func(start, end int) (int, int) { return start, end }
	if cleaned == script {
		return identity
	}
	scriptLines := strings.Split(script, "\n")
	scriptStarts := make([]int, len(scriptLines))
	cursor := 0
	for i, line := range scriptLines {
		scriptStarts[i] = cursor
		cursor += len(line) + 1
	}
	cleanedLines := strings.Split(cleaned, "\n")
	cleanedStarts := make([]int, len(cleanedLines))
	sourceStarts := make([]int, len(cleanedLines))
	cursor = 0
	for i, line := range cleanedLines {
		cleanedStarts[i] = cursor
		cursor += len(line) + 1
	}
	kept := make([]bool, len(scriptLines))
	next := 0
	for i, line := range cleanedLines {
		first, last := i == 0, i == len(cleanedLines)-1
		for next < len(scriptLines) && !sameLine(scriptLines[next], line, first, last) {
			next++
		}
		if next == len(scriptLines) {
			return identity
		}
		kept[next] = true
		sourceStarts[i] = scriptStarts[next]
		if i == 0 {
			sourceStarts[i] += len(scriptLines[next]) - len(strings.TrimLeft(scriptLines[next], " \t"))
		}
		next++
	}
	var removed [][2]int
	for i, line := range scriptLines {
		if !kept[i] {
			removed = append(removed, [2]int{scriptStarts[i], scriptStarts[i] + len(line) + 1})
		}
	}
	sourceOffset := func(at int) int {
		if at < 0 || at > len(cleaned) {
			return at
		}
		i := 0
		for i+1 < len(cleanedStarts) && cleanedStarts[i+1] <= at {
			i++
		}
		return sourceStarts[i] + (at - cleanedStarts[i])
	}
	return func(start, end int) (int, int) {
		sourceStart, sourceEnd := sourceOffset(start), sourceOffset(end)
		for _, span := range removed {
			if sourceStart < span[1] && sourceEnd > span[0] && span[1] > sourceStart {
				sourceStart = span[1]
			}
		}
		if sourceStart > sourceEnd {
			sourceStart = sourceEnd
		}
		return sourceStart, sourceEnd
	}
}

// sameLine reports whether a cleaned line is the caller's line. The parser trims
// the whole text, so the first kept line may have lost its leading whitespace and
// the last its trailing whitespace; anything else, including a last line's
// indentation, is copied verbatim.
func sameLine(scriptLine, cleanedLine string, first, last bool) bool {
	candidate := scriptLine
	if first {
		candidate = strings.TrimLeftFunc(candidate, unicode.IsSpace)
	}
	if last {
		candidate = strings.TrimRightFunc(candidate, unicode.IsSpace)
	}
	return candidate == cleanedLine
}

type executeLogger struct {
	opts db.ExecuteOptions
	// sourceRange maps a range the driver reports on its own script back to the
	// caller's, where the task run log reads the command from, and drops the
	// directive lines the parser removed.
	sourceRange func(int, int) (int, int)
}

func (l executeLogger) CommandExecute(command odbdriver.Command) {
	start, end := l.sourceRange(command.ByteStart, command.ByteEnd)
	l.opts.LogCommandExecute(&storepb.Range{
		Start: int32(start),
		End:   int32(end),
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
