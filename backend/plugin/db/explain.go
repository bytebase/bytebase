package db

import (
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	omniparser "github.com/bytebase/omni/pg/parser"
	"github.com/pkg/errors"

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
// its FORMAT is set to the requested one, text by default, instead. The error
// refuses a PostgreSQL statement whose plan would execute it, as EXPLAIN
// ANALYZE does.
func ExplainStatement(engine storepb.Engine, statement string, format v1pb.QueryOption_ExplainFormat) (string, bool, error) {
	switch engine {
	case storepb.Engine_POSTGRES:
		explain, err := explainPostgres(statement, format)
		return explain, true, err
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
		return fmt.Sprintf("EXPLAIN %s", statement), true, nil
	default:
		return "", false, nil
	}
}

func explainPostgres(statement string, format v1pb.QueryOption_ExplainFormat) (string, error) {
	name := "text"
	if format == v1pb.QueryOption_JSON || format == v1pb.QueryOption_XML {
		name = strings.ToLower(format.String())
	}
	explain := pgparser.ParseExplain(statement)
	switch {
	case explain == nil && name == "text":
		statement = "EXPLAIN " + statement
	case explain == nil:
		statement = fmt.Sprintf("EXPLAIN (FORMAT %s) %s", format, statement)
	case pgparser.ExplainFormat(explain) != name:
		var err error
		if statement, err = setExplainFormat(statement, name); err != nil {
			return "", err
		}
	default:
	}
	// A statement that does not parse is left to the query validator, which
	// refuses it.
	if stmts, err := pgparser.ParsePg(statement); err == nil {
		for _, stmt := range stmts {
			if explain, ok := stmt.AST.(*ast.ExplainStmt); ok && pgparser.IsExplainAnalyze(explain) {
				return "", errors.New("explaining this statement would execute it, as EXPLAIN ANALYZE does; run it instead")
			}
		}
	}
	return statement, nil
}

// setExplainFormat rewrites an EXPLAIN to produce format, keeping its other
// options as written.
func setExplainFormat(statement, format string) (string, error) {
	tokens := omniparser.Tokenize(statement)
	// tokens[0] is EXPLAIN. The options follow it in parentheses, or as the
	// ANALYZE and VERBOSE keywords.
	var options []string
	i := 1
	if i < len(tokens) && tokens[i].Type == '(' {
		depth, start := 0, i+1
		for ; i < len(tokens); i++ {
			switch tokens[i].Type {
			case '(':
				depth++
			case ')':
				depth--
			default:
			}
			if depth == 0 || depth == 1 && tokens[i].Type == ',' {
				if start < i && tokens[start].Str != "format" {
					options = append(options, statement[tokens[start].Loc:tokens[i-1].End])
				}
				start = i + 1
			}
			if depth == 0 {
				break
			}
		}
		i++
	} else {
		for ; i < len(tokens) && (tokens[i].Type == omniparser.ANALYZE || tokens[i].Type == omniparser.ANALYSE || tokens[i].Type == omniparser.VERBOSE); i++ {
			options = append(options, strings.ToUpper(tokens[i].Str))
		}
	}
	if i >= len(tokens) {
		return "", errors.New("failed to read the EXPLAIN options")
	}
	options = append(options, "FORMAT "+strings.ToUpper(format))
	return fmt.Sprintf("EXPLAIN (%s) %s", strings.Join(options, ", "), statement[tokens[i].Loc:]), nil
}
