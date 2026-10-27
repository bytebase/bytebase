package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_POSTGRES, explainStatement)
	// CockroachDB parses through the PostgreSQL grammar here as it does for the
	// read-only gate (see query.go).
	base.RegisterExplainStatementFunc(storepb.Engine_COCKROACHDB, cockroachExplainStatement)
}

// plannedStatement returns the statement an EXPLAIN should be built around, and
// whether one can be: a statement that already asks for a plan is rebuilt around
// the statement it plans, and one whose planned statement cannot be located runs
// as the caller wrote it rather than becoming the plan of a plan.
func plannedStatement(statement string) (planned string, rebuild bool, err error) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", false, err
	}
	explain, ok := node.(*ast.ExplainStmt)
	if !ok {
		return statement, true, nil
	}
	if isExplainAnalyzeOmni(explain) {
		return "", false, base.ErrExplainAnalyzeExecutes
	}
	inner, ok := explainedText(explain, statement)
	if !ok {
		return statement, false, nil
	}
	return inner, true, nil
}

// explainStatement implements base.ExplainStatement for PostgreSQL.
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	planned, rebuild, err := plannedStatement(statement)
	if err != nil {
		return "", err
	}
	if !rebuild {
		return planned, nil
	}
	switch format {
	case base.ExplainFormatJSON:
		return "EXPLAIN (FORMAT JSON) " + planned, nil
	case base.ExplainFormatXML:
		return "EXPLAIN (FORMAT XML) " + planned, nil
	case base.ExplainFormatYAML:
		return "EXPLAIN (FORMAT YAML) " + planned, nil
	default:
		return "EXPLAIN " + planned, nil
	}
}

// DescribeExplain returns the output format of an EXPLAIN statement and
// whether it executes the statement it plans.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", false, false
	}
	explain, ok := node.(*ast.ExplainStmt)
	if !ok {
		return "", false, false
	}
	return explainFormat(explain), isExplainAnalyzeOmni(explain), true
}

func explainFormat(explain *ast.ExplainStmt) string {
	format := "text"
	if explain.Options == nil {
		return format
	}
	for _, item := range explain.Options.Items {
		option, ok := item.(*ast.DefElem)
		if !ok || !strings.EqualFold(option.Defname, "format") {
			continue
		}
		format = ""
		if arg, ok := option.Arg.(*ast.String); ok {
			format = strings.ToLower(arg.Str)
		}
	}
	return format
}

// cockroachExplainStatement implements base.ExplainStatement for CockroachDB,
// whose EXPLAIN takes its own option list rather than PostgreSQL's FORMAT and
// has no option that returns a plan as a document.
//
// What it does have is DISTSQL, which returns the text plan as always and a
// link to a diagram of the physical flow beside it; the payload in that link is
// the one machine-readable plan CockroachDB publishes, and the driver decodes
// it. The statement does not execute, as EXPLAIN ANALYZE (DISTSQL) would.
func cockroachExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	planned, rebuild, err := plannedStatement(statement)
	if err != nil {
		return "", err
	}
	if !rebuild {
		return planned, nil
	}
	if format == base.ExplainFormatJSON {
		return "EXPLAIN (DISTSQL) " + planned, nil
	}
	return "EXPLAIN " + planned, nil
}

// singleStatement returns the one statement in text. An explain request plans one
// statement at a time: both the drivers and the API gate split a multi-statement
// request first.
func singleStatement(text string) (ast.Node, error) {
	stmts, err := ParsePg(text)
	if err != nil {
		return nil, err
	}
	var node ast.Node
	for _, stmt := range stmts {
		if stmt.AST == nil {
			continue
		}
		if node != nil {
			return nil, errors.New("EXPLAIN plans a single statement")
		}
		node = stmt.AST
	}
	if node == nil {
		return nil, errors.New("EXPLAIN needs a statement to plan")
	}
	return node, nil
}

// explainedText returns the text within text of the statement explain plans, and
// whether it could be located.
func explainedText(explain *ast.ExplainStmt, text string) (string, bool) {
	// The statement runs from its WITH clause, which the location of a SELECT leaves out, to the end
	// of the EXPLAIN, which its location can also leave out, as for ORDER BY.
	start := ast.NodeLoc(explain.Query).Start
	if with := getWithClause(explain.Query); with != nil && with.Loc.Start >= 0 && with.Loc.Start < start {
		start = with.Loc.Start
	}
	if start < 0 || start >= explain.Loc.End || explain.Loc.End > len(text) {
		return "", false
	}
	return text[start:explain.Loc.End], true
}
