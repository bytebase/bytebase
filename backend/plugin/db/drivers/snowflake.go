package drivers

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

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var snowflakeSpec = spec{
	open: func(ctx context.Context, _ *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		return snowflakedriver.Open(ctx, snowflakeConfig(config))
	},
	adapter: util.AdapterOptions{WithInstanceMetadata: true, NotFoundResource: "database"},
	execute: util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassCreateDatabase:        true,
		PassMaxRetries:            true,
		ShiftCommandRanges:        true,
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		queryOpts := util.OmniQueryOptions(
			queryContext,
			true,
			util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON),
		)
		// The product reads a zero cap as "no rows", and the contract reads it as
		// "no cap".
		queryOpts.MaxResultBytes = util.PositiveMaxResultBytes(queryContext.MaximumSQLResultSize)
		return d.QueryHeldConn(
			ctx,
			statement,
			queryOpts,
			util.SessionQueryOptions{KeepOnError: isRecoverableSnowflakeQueryError},
			util.QueryResultOptions{
				ClearExplainPlan:          queryContext.Explain,
				ClearExplainPlanOnError:   true,
				TypedExplainPlan:          true,
				IncludePlanContent:        true,
				FillZeroAffectedRows:      true,
				FormatOutOfRangeTimestamp: util.FormatDefaultTimestamp,
			},
		)
	},
}

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

// isRecoverableSnowflakeQueryError reports whether err left the held session
// healthy, so later calls on the same stream keep temporary tables and open
// transactions. omni/snowflake/driver returns a top-level Session.Query error
// only on pre-execution parse and explain checks, before touching the
// connection, and on USE SCHEMA when QueryOptions.Schema is set.
func isRecoverableSnowflakeQueryError(ctx context.Context, err error) bool {
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
			// Snowflake server SQL compilation, execution and access-control codes
			// sit below gosnowflake's 260000+ client and network range; 390201 is
			// the object-not-found or unauthorized code, while 390111, 390112 and
			// 3903xx mean a closed or expired session or token.
			return (code > 0 && code < 260000) || code == 390201
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
// "%06d: ..." error prefix without importing gosnowflake.
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
