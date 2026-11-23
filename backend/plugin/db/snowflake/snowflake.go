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
	"strconv"
	"strings"

	odriver "github.com/bytebase/omni/driver"
	snowflakedriver "github.com/bytebase/omni/snowflake/driver"
	snowflakeparser "github.com/bytebase/omni/snowflake/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_SNOWFLAKE, newDriver)
}

// Driver is the Snowflake driver.
type Driver struct {
	util.OmniAdapter
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
	d.OmniAdapter = util.OmniAdapter{
		Driver:               impl,
		WithInstanceMetadata: true,
		NotFoundResource:     "database",
		DatabaseName:         config.ConnectionContext.DatabaseName,
	}
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

// QueryConn queries a SQL statement on the driver's lazily opened session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	// Bytebase treats a non-positive cap as zero allowed bytes, whereas the
	// omni driver contract treats <= 0 as unbounded.
	queryOpts := util.OmniQueryOptions(
		queryContext,
		true,
		util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON),
	)
	queryOpts.MaxResultBytes = util.PositiveMaxResultBytes(queryContext.MaximumSQLResultSize)
	return d.QueryHeldConn(
		ctx,
		statement,
		queryOpts,
		util.SessionQueryOptions{KeepOnError: isRecoverableQueryError},
		util.QueryResultOptions{
			ClearExplainPlan:          queryContext.Explain,
			ClearExplainPlanOnError:   true,
			TypedExplainPlan:          true,
			IncludePlanContent:        true,
			FillZeroAffectedRows:      true,
			FormatOutOfRangeTimestamp: util.FormatDefaultTimestamp,
		},
	)
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

// Execute executes a SQL script and returns the affected rows.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassCreateDatabase:        true,
		PassMaxRetries:            true,
		ShiftCommandRanges:        true,
	})
}
