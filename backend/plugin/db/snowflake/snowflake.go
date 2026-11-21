// Package snowflake is Bytebase's Snowflake driver: an adapter over
// github.com/bytebase/omni/snowflake/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos.
//
// Adapter invariants:
//   - Open validates the configuration through snowflakedriver.Open without
//     dialing the account. QueryConn opens one odriver.Session lazily on first
//     use, serializes calls on it with sessionMu, preserves it across
//     recoverable SQL/parse errors (such as a failed USE SCHEMA), and discards
//     it on Close, context cancellation, or unrecoverable connection/session
//     errors.
//   - QueryConn normalizes successful columnless results with nil AffectedRows
//     to one "Affected Rows: 0" row, maps lowercase plan formats from
//     snowflake/analysis onto QueryOption_ExplainFormat for user-written EXPLAIN
//     statements (leaving Explain-mode requests to SQLService), and formats
//     timestamps outside google.protobuf.Timestamp's 0001..9999 range as text.
//   - Execute parses leading `-- txn-mode` directives into Transactional or
//     AutoCommit and maps logged command byte ranges back to the caller's script
//     with directiveShift.
package snowflake

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	snowflakedriver "github.com/bytebase/omni/snowflake/driver"
	snowflakeparser "github.com/bytebase/omni/snowflake/parser"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type snowflakeImpl interface {
	odriver.Driver
	odriver.Syncer
	odriver.Dumper
	odriver.Querier
	odriver.Executor
}

var (
	_ db.Driver     = (*Driver)(nil)
	_ snowflakeImpl = (*snowflakedriver.Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_SNOWFLAKE, newDriver)
}

// Driver is the Snowflake driver.
type Driver struct {
	databaseName string
	impl         snowflakeImpl

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
	var sessionErr error
	if d.session != nil {
		sessionErr = d.session.Close()
		d.session = nil
	}
	d.sessionMu.Unlock()

	var implErr error
	if d.impl != nil {
		implErr = d.impl.Close()
	}
	return errors.Join(sessionErr, implErr)
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// QueryConn queries a SQL statement on the driver's lazily opened session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()

	session, err := d.querySessionLocked(ctx)
	if err != nil {
		return nil, err
	}

	// Bytebase treats a non-positive cap as zero allowed bytes, whereas the
	// omni driver contract treats <= 0 as unbounded.
	maxResultBytes := queryContext.MaximumSQLResultSize
	if maxResultBytes <= 0 {
		maxResultBytes = 1
	}
	results, err := session.Query(ctx, statement, odriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON),
		MaskingEnabled: queryContext.MaskingEnabled,
		MaxResultBytes: maxResultBytes,
	})
	if err != nil {
		if !isRecoverableQueryError(ctx, err) {
			d.discardSessionLocked(session)
		}
		return nil, err
	}
	if ctx.Err() != nil {
		d.discardSessionLocked(session)
	}
	if len(results) == 0 {
		return nil, nil
	}

	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, toQueryResult(result, queryContext))
	}
	return converted, nil
}

// isRecoverableQueryError reports whether err left the held Snowflake session
// healthy so later calls on the same stream keep temporary tables and open
// transactions. omni/snowflake/driver returns a top-level Session.Query error
// only on pre-execution parse/explain checks (before touching the connection)
// and on USE SCHEMA when QueryOptions.Schema is non-empty.
func isRecoverableQueryError(ctx context.Context, err error) bool {
	if err == nil {
		return true
	}
	if ctx.Err() != nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, sql.ErrConnDone) ||
		errors.Is(err, sqldriver.ErrBadConn) {
		return false
	}
	var parseErr *snowflakeparser.ParseError
	if errors.As(err, &parseErr) {
		return true
	}
	switch err.Error() {
	case "EXPLAIN needs a statement to plan", "EXPLAIN plans a single statement":
		return true
	default:
	}
	return isRecoverableSnowflakeServerError(err)
}

func isRecoverableSnowflakeServerError(err error) bool {
	queue := []error{err}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil {
			continue
		}
		if code, sqlState, ok := parseSnowflakeErrorPrefix(cur.Error()); ok {
			if strings.HasPrefix(sqlState, "08") || strings.HasPrefix(sqlState, "57") {
				return false
			}
			// Snowflake server SQL compilation/execution/access-control codes
			// sit below gosnowflake's 260000+ client/network range; 390201 is
			// the GS object-not-found/unauthorized code (whereas 390111/390112/
			// 3903xx indicate closed or expired sessions/tokens).
			if (code > 0 && code < 260000) || code == 390201 {
				return true
			}
			return false
		}
		switch u := cur.(type) {
		case interface{ Unwrap() error }:
			queue = append(queue, u.Unwrap())
		case interface{ Unwrap() []error }:
			queue = append(queue, u.Unwrap()...)
		default:
		}
	}
	return false
}

// parseSnowflakeErrorPrefix parses gosnowflake's "%06d (%s): ..." or
// "%06d: ..." error prefix without importing gosnowflake directly.
func parseSnowflakeErrorPrefix(msg string) (int, string, bool) {
	if len(msg) < 8 {
		return 0, "", false
	}
	for i := range 6 {
		if msg[i] < '0' || msg[i] > '9' {
			return 0, "", false
		}
	}
	code, err := strconv.Atoi(msg[:6])
	if err != nil {
		return 0, "", false
	}
	rest := msg[6:]
	if strings.HasPrefix(rest, " (") {
		end := strings.Index(rest[2:], "): ")
		if end <= 0 || end > 8 {
			return 0, "", false
		}
		return code, rest[2 : 2+end], true
	}
	if strings.HasPrefix(rest, ": ") {
		return code, "", true
	}
	return 0, "", false
}

func (d *Driver) querySessionLocked(ctx context.Context) (odriver.Session, error) {
	if d.session != nil {
		return d.session, nil
	}
	session, err := d.impl.Session(ctx)
	if err != nil {
		return nil, err
	}
	d.session = session
	return session, nil
}

func (d *Driver) discardSessionLocked(session odriver.Session) {
	_ = session.Close()
	if d.session == session {
		d.session = nil
	}
}

func toQueryResult(result *odriver.Result, queryContext db.QueryContext) *v1pb.QueryResult {
	if result == nil {
		return &v1pb.QueryResult{}
	}
	normalized := normalizeOutOfRangeTimestamps(result)
	queryResult := util.OmniResultToQueryResult(normalized, queryContext.MaximumSQLResultSize)
	util.FillZeroAffectedRows(normalized, queryResult)
	// Explain requests leave QueryPlan unset so SQLService stamps the requested
	// format; user-written EXPLAIN statements map snowflake/analysis's lowercase
	// format names ("text", "json") through db.TypedExplainPlan.
	if queryContext.Explain || normalized.Err != nil {
		queryResult.QueryPlan = nil
	} else if normalized.Plan != nil {
		queryResult.QueryPlan = db.TypedExplainPlan(normalized.Plan.Format, normalized.Plan.Executed, true)
		queryResult.QueryPlan.Content = normalized.Plan.Content
	}
	return queryResult
}

// normalizeOutOfRangeTimestamps converts Timestamp/TimestampTZ cells outside
// google.protobuf.Timestamp's [0001-01-01, 9999-12-31] range into strings so
// protobuf serialization never fails on extreme Snowflake dates.
func normalizeOutOfRangeTimestamps(result *odriver.Result) *odriver.Result {
	var cloned *odriver.Result
	for i, row := range result.Rows {
		var clonedRow []odriver.Value
		for j, cell := range row {
			if cell.Kind != odriver.Timestamp && cell.Kind != odriver.TimestampTZ {
				continue
			}
			if timestamppb.New(cell.Time).CheckValid() == nil {
				continue
			}
			if cloned == nil {
				copyResult := *result
				copyResult.Rows = append([][]odriver.Value(nil), result.Rows...)
				cloned = &copyResult
			}
			if clonedRow == nil {
				clonedRow = append([]odriver.Value(nil), row...)
				cloned.Rows[i] = clonedRow
			}
			text := cell.Time.Format(time.RFC3339Nano)
			if cell.Kind == odriver.Timestamp {
				text = cell.Time.UTC().Format("2006-01-02 15:04:05.999999999")
			}
			clonedRow[j] = odriver.Value{Kind: odriver.String, String: text}
		}
	}
	if cloned != nil {
		return cloned
	}
	return result
}

// Execute executes a SQL script and returns the affected rows.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	transactionMode, _, cleanedStatement := util.ParseTransactionMode(statement)
	shift := util.DirectiveShift(statement, cleanedStatement)
	return d.impl.Execute(ctx, cleanedStatement, odriver.ExecuteOptions{
		Transaction:    transactionMode,
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log: util.NewExecuteLogger(opts, util.WithCommandRange(func(command odriver.Command) *storepb.Range {
			start, end := shift(command.ByteStart, command.ByteEnd)
			return &storepb.Range{Start: int32(start), End: int32(end)}
		})),
	})
}

// SyncInstance syncs the Snowflake account's version, databases, and user roles.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return util.OmniInstanceToMetadata(instance, true), nil
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
