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
	base.RegisterExplainStatementFunc(storepb.Engine_COCKROACHDB, explainStatementDefaultFormat)
}

// explainStatement implements base.ExplainStatement for PostgreSQL.
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", err
	}
	planned := statement
	if explain, ok := node.(*ast.ExplainStmt); ok {
		if isExplainAnalyzeOmni(explain) {
			return "", base.ErrExplainAnalyzeExecutes
		}
		inner, ok := explainedText(explain, statement)
		if !ok {
			// Nothing to rebuild around: run what the caller wrote rather than
			// plan a plan.
			return statement, nil
		}
		planned = inner
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

// explainStatementDefaultFormat plans in the engine's default format whatever the
// request asked for. CockroachDB's EXPLAIN takes its own option list rather than
// PostgreSQL's FORMAT, so it offers only the default plan (the driver capability
// table refuses any other format before a request gets here).
func explainStatementDefaultFormat(statement string, _ base.ExplainFormat) (string, error) {
	return explainStatement(statement, base.ExplainFormatDefault)
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
