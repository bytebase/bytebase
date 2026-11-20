// Package snowflake is Bytebase's Snowflake driver: an adapter over
// github.com/bytebase/omni/snowflake/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos.
package snowflake

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"sync"
	"unicode"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	snowflakedriver "github.com/bytebase/omni/snowflake/driver"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ db.Driver         = (*Driver)(nil)
	_ db.SessionQuerier = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_SNOWFLAKE, newDriver)
}

// Driver is the Snowflake driver.
type Driver struct {
	databaseName string
	impl         *snowflakedriver.Driver

	// sessionMu guards session and serializes QueryConn calls on one held
	// client connection from the first QueryConn until Close.
	sessionMu sync.Mutex
	session   odriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a Snowflake driver. It does not dial the server; Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	impl, err := snowflakedriver.Open(ctx, snowflakeConfig(config))
	if err != nil {
		return nil, err
	}
	d.impl = impl
	d.databaseName = config.ConnectionContext.DatabaseName
	return d, nil
}

// snowflakeConfig maps Bytebase's connection configuration onto the omni driver
// contract's Config.
func snowflakeConfig(config db.ConnectionConfig) odriver.Config {
	ds := config.DataSource
	cfg := odriver.Config{
		Host:     ds.GetHost(),
		Port:     ds.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		User:     ds.GetUsername(),
		Password: config.Password,
	}
	if key := ds.GetAuthenticationPrivateKey(); key != "" {
		cfg.Params = map[string]string{
			snowflakedriver.ParamPrivateKey: key,
		}
		if passphrase := ds.GetAuthenticationPrivateKeyPassphrase(); passphrase != "" {
			cfg.Params[snowflakedriver.ParamPrivateKeyPassphrase] = passphrase
		}
	}
	return cfg
}

// Close closes the held query session and the underlying driver.
func (d *Driver) Close(context.Context) error {
	d.sessionMu.Lock()
	var err error
	if d.session != nil {
		err = d.session.Close()
		d.session = nil
	}
	d.sessionMu.Unlock()

	if d.impl != nil {
		if closeErr := d.impl.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// OwnsQuerySession reports that QueryConn runs on the session the adapter holds
// rather than on the *sql.Conn a caller passes, which is unused.
func (*Driver) OwnsQuerySession() {}

// QueryConn queries a SQL statement on the driver's lazily opened session.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	if d.session == nil {
		session, err := d.impl.Session(ctx)
		if err != nil {
			return nil, err
		}
		d.session = session
	}

	session := d.session
	results, err := session.Query(ctx, statement, odriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  explainFormat(queryContext),
		MaskingEnabled: queryContext.MaskingEnabled,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	})
	if err != nil {
		_ = session.Close()
		d.session = nil
		return nil, err
	}
	if ctx.Err() != nil {
		_ = session.Close()
		d.session = nil
	}
	if len(results) == 0 {
		return nil, nil
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize))
	}
	return converted, nil
}

func explainFormat(queryContext db.QueryContext) string {
	switch db.ExplainFormat(queryContext.Option.GetExplainFormat()) {
	case base.ExplainFormatJSON:
		return "JSON"
	case base.ExplainFormatText:
		return "TEXT"
	default:
		return ""
	}
}

// Execute executes a SQL script and returns the affected rows.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	mode := config.Mode
	if mode == transaction.ModeUnspecified {
		mode = transaction.DefaultMode()
	}
	transactionMode := odriver.Transactional
	if mode == transaction.ModeOff {
		transactionMode = odriver.AutoCommit
	}
	return d.impl.Execute(ctx, cleanedStatement, odriver.ExecuteOptions{
		Transaction:    transactionMode,
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log:            executeLogger{opts: opts, sourceRange: directiveShift(statement, cleanedStatement)},
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
			sourceStarts[i] += len(scriptLines[next]) - len(strings.TrimLeftFunc(scriptLines[next], unicode.IsSpace))
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
	opts        db.ExecuteOptions
	sourceRange func(int, int) (int, int)
}

func (l executeLogger) CommandExecute(command odriver.Command) {
	start, end := command.ByteStart, command.ByteEnd
	if l.sourceRange != nil {
		start, end = l.sourceRange(start, end)
	}
	l.opts.LogCommandExecute(&storepb.Range{
		Start: int32(start),
		End:   int32(end),
	}, command.Text)
}

func (l executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (l executeLogger) TransactionControl(op odriver.TransactionOp, err error) {
	var control storepb.TaskRunLog_TransactionControl_Type
	switch op {
	case odriver.Begin:
		control = storepb.TaskRunLog_TransactionControl_BEGIN
	case odriver.Commit:
		control = storepb.TaskRunLog_TransactionControl_COMMIT
	case odriver.Rollback:
		control = storepb.TaskRunLog_TransactionControl_ROLLBACK
	default:
		return
	}
	l.opts.LogTransactionControl(control, errorString(err))
}

func (l executeLogger) Retry(err error, attempt, _ int) {
	l.opts.LogRetryInfo(err, attempt)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// SyncInstance syncs the Snowflake account's version, databases, and user roles.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]*storepb.InstanceRole, 0, len(instance.Roles))
	for _, role := range instance.Roles {
		roles = append(roles, &storepb.InstanceRole{
			Name:      role.Name,
			Attribute: role.Attribute,
		})
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
		Metadata:  &storepb.Instance{Roles: roles},
	}, nil
}

// SyncDBSchema syncs the schema of the database the driver was opened on.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	metadata, err := d.impl.SyncDatabase(ctx)
	if err != nil {
		if errors.Is(err, odriver.ErrNotFound) {
			return nil, common.Errorf(common.NotFound, "database %q not found", d.databaseName)
		}
		return nil, err
	}
	return metadata, nil
}

// Dump writes the database's server-rendered DDL.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, dbMetadata)
}
