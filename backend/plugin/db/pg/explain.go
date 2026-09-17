package pg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var postgresExplain = db.Explain{
	Formats: []v1pb.QueryOption_ExplainFormat{
		v1pb.QueryOption_TEXT,
		v1pb.QueryOption_JSON,
		v1pb.QueryOption_XML,
		v1pb.QueryOption_YAML,
	},
	DefaultFormat: v1pb.QueryOption_TEXT,
	Statement:     explainStatement,
}

// explainStatement returns the statement whose result is statement's plan in
// format, text by default. A statement that already is an EXPLAIN gets that
// format rather than a second EXPLAIN, and one whose plan would run it, as
// EXPLAIN ANALYZE does, is refused.
func explainStatement(statement string, format v1pb.QueryOption_ExplainFormat) (string, error) {
	if format == v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED {
		format = v1pb.QueryOption_TEXT
	}
	explained := withExplainFormat(statement, format)
	// A statement that does not parse is left to the query validator, which
	// refuses it with the same parser.
	if stmts, err := pgparser.ParsePg(explained); err == nil {
		for _, stmt := range stmts {
			if explain, ok := stmt.AST.(*ast.ExplainStmt); ok && pgparser.IsExplainAnalyze(explain) {
				return "", errors.New("explaining this statement would execute it, as EXPLAIN ANALYZE does; run it instead")
			}
		}
	}
	return explained, nil
}

// withExplainFormat prefixes statement with an EXPLAIN in format, or rebuilds
// an EXPLAIN statement with format in place of its own, keeping its other
// options.
func withExplainFormat(statement string, format v1pb.QueryOption_ExplainFormat) string {
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
			return "EXPLAIN " + statement
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

// typedExplainPlan describes the plan an EXPLAIN in an ordinary query returns,
// and is nil for any other statement. The Query handler describes the plans of
// an explain request.
func typedExplainPlan(statement string) *v1pb.QueryResult_QueryPlan {
	explain := pgparser.ParseExplain(statement)
	if explain == nil {
		return nil
	}
	// PostgreSQL accepts no format but text, json, xml and yaml, which the enum
	// values are named after.
	format := v1pb.QueryOption_ExplainFormat_value[strings.ToUpper(pgparser.ExplainFormat(explain))]
	return &v1pb.QueryResult_QueryPlan{
		Format:   v1pb.QueryOption_ExplainFormat(format),
		Executed: pgparser.IsExplainAnalyze(explain),
	}
}
