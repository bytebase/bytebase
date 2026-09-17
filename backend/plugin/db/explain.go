package db

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// ExplainStatement returns the statement that produces a query plan for
// statement on engine, and whether engine explains by running that statement.
// It is the single source of that statement: a driver calls it to build the
// statement it runs, and the Query handler calls it to validate the exact
// statement that will run (an EXPLAIN ANALYZE of a write is not read-only and is
// refused before execution).
//
// The second return is false for engines whose EXPLAIN is not a bare prefix and
// so does not execute the wrapped statement — Oracle (EXPLAIN PLAN FOR), SQL
// Server (SET SHOWPLAN), Spanner/BigQuery (plan APIs) — and for engines with no
// EXPLAIN at all. Those drivers produce a plan on their own and pass "" here.
//
// On PostgreSQL, a statement that already is an EXPLAIN is not prefixed again;
// its FORMAT is set to the requested one, text by default, instead.
func ExplainStatement(engine storepb.Engine, statement string, format v1pb.QueryOption_ExplainFormat) (string, bool) {
	switch engine {
	case storepb.Engine_POSTGRES:
		return explainPostgres(statement, format), true
	case storepb.Engine_MYSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_TIDB,
		storepb.Engine_REDSHIFT,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_STARROCKS,
		storepb.Engine_DORIS,
		storepb.Engine_HIVE,
		storepb.Engine_TRINO:
		return fmt.Sprintf("EXPLAIN %s", statement), true
	default:
		return "", false
	}
}

func explainPostgres(statement string, format v1pb.QueryOption_ExplainFormat) string {
	if format == v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED {
		format = v1pb.QueryOption_TEXT
	}
	explain := pgparser.ParseExplain(statement)
	if explain != nil && pgparser.ExplainFormat(explain) == strings.ToLower(format.String()) {
		return statement
	}
	start := -1
	if explain != nil {
		start = pgparser.ExplainedStatementStart(explain)
	}
	// An EXPLAIN whose explained statement cannot be found is prefixed like any
	// other statement, which gives a syntax error the query validator refuses.
	if start < 0 || start > len(statement) {
		if format == v1pb.QueryOption_TEXT {
			return fmt.Sprintf("EXPLAIN %s", statement)
		}
		return fmt.Sprintf("EXPLAIN (FORMAT %s) %s", format, statement)
	}
	var options []string
	if explain.Options != nil {
		for _, item := range explain.Options.Items {
			option, ok := item.(*ast.DefElem)
			if !ok || option.Defname == "format" {
				continue
			}
			switch arg := option.Arg.(type) {
			case *ast.String:
				options = append(options, option.Defname+" "+arg.Str)
			case *ast.Integer:
				options = append(options, option.Defname+" "+strconv.FormatInt(arg.Ival, 10))
			case *ast.Float:
				options = append(options, option.Defname+" "+arg.Fval)
			default:
				options = append(options, option.Defname)
			}
		}
	}
	options = append(options, fmt.Sprintf("FORMAT %s", format))
	return fmt.Sprintf("EXPLAIN (%s) %s", strings.Join(options, ", "), statement[start:])
}
