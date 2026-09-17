package pg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	omniparser "github.com/bytebase/omni/pg/parser"
	"github.com/pkg/errors"
)

// ExplainStatement returns the statement whose result is statement's query
// plan in format: JSON, XML, YAML, or text for any other name. A statement that
// already is an EXPLAIN gets format in place of its own rather than a second
// EXPLAIN, and one whose plan would run it, as EXPLAIN ANALYZE does, is
// refused.
func ExplainStatement(statement, format string) (string, error) {
	format = strings.ToLower(format)
	if format != "json" && format != "xml" && format != "yaml" {
		format = "text"
	}
	explained := withExplainFormat(statement, format)
	// A statement that does not parse is left to the query validator, which
	// refuses it with the same parser.
	if stmts, err := ParsePg(explained); err == nil {
		for _, stmt := range stmts {
			if explain, ok := stmt.AST.(*ast.ExplainStmt); ok && isExplainAnalyzeOmni(explain) {
				return "", errors.New("explaining this statement would execute it, as EXPLAIN ANALYZE does; run it instead")
			}
		}
	}
	return explained, nil
}

// DescribeExplain returns the output format an EXPLAIN statement names, in
// lower case, and whether it runs the statement it explains. ok is false when
// statement is not an EXPLAIN.
func DescribeExplain(statement string) (format string, analyze bool, ok bool) {
	explain := parseExplain(statement)
	if explain == nil {
		return "", false, false
	}
	return explainFormat(explain), isExplainAnalyzeOmni(explain), true
}

// withExplainFormat prefixes statement with an EXPLAIN in format, or rebuilds
// an EXPLAIN statement with format in place of its own, keeping its other
// options.
func withExplainFormat(statement, format string) string {
	explain := parseExplain(statement)
	if explain != nil && explainFormat(explain) == format {
		return statement
	}
	start := -1
	if explain != nil {
		start = explainedStatementStart(explain)
	}
	// An EXPLAIN whose explained statement cannot be found is prefixed like any
	// other statement, which gives a syntax error the query validator refuses.
	if start < 0 || start > len(statement) {
		if format == "text" {
			return "EXPLAIN " + statement
		}
		return fmt.Sprintf("EXPLAIN (FORMAT %s) %s", strings.ToUpper(format), statement)
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
	options = append(options, "FORMAT "+strings.ToUpper(format))
	return fmt.Sprintf("EXPLAIN (%s) %s", strings.Join(options, ", "), statement[start:])
}

// parseExplain returns the EXPLAIN that statement consists of, or nil when
// statement is anything else.
func parseExplain(statement string) *ast.ExplainStmt {
	// Most statements are not an EXPLAIN, and the first token says so without a
	// full parse.
	if omniparser.NewLexer(statement).NextToken().Type != omniparser.EXPLAIN {
		return nil
	}
	stmts, err := ParsePg(statement)
	if err != nil || len(stmts) != 1 {
		return nil
	}
	explain, ok := stmts[0].AST.(*ast.ExplainStmt)
	if !ok {
		return nil
	}
	return explain
}

// explainFormat returns the output format an EXPLAIN names, as PostgreSQL
// reads it from the last FORMAT option: "text" when it names none, and "" when
// the option has no name to read.
func explainFormat(explain *ast.ExplainStmt) string {
	format := "text"
	if explain.Options == nil {
		return format
	}
	for _, item := range explain.Options.Items {
		option, ok := item.(*ast.DefElem)
		if !ok || option.Defname != "format" {
			continue
		}
		format = ""
		if arg, ok := option.Arg.(*ast.String); ok {
			format = arg.Str
		}
	}
	return format
}
