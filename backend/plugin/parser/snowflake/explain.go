package snowflake

import (
	"github.com/bytebase/omni/snowflake/ast"
	"github.com/bytebase/omni/snowflake/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_SNOWFLAKE, explainStatement)
}

// explainStatement implements base.ExplainStatement. Snowflake spells its
// structured plan EXPLAIN USING JSON; any other format the caller names gets
// the tabular plan, which is the table Snowflake has always returned here.
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", err
	}
	planned := statement
	if explain, ok := node.(*ast.ExplainStmt); ok {
		if inner, ok := explainedText(explain, statement); ok {
			planned = inner
		} else {
			// Nothing to rebuild around: run what the user wrote rather than
			// plan a plan.
			return statement, nil
		}
	}
	if format == base.ExplainFormatJSON {
		return "EXPLAIN USING JSON " + planned, nil
	}
	return "EXPLAIN " + planned, nil
}

// DescribeExplain returns the output format of an EXPLAIN statement and whether
// it executes the statement it plans. Snowflake's EXPLAIN compiles a statement
// without running it, whichever format it names, so nothing here executes.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", false, false
	}
	explain, ok := node.(*ast.ExplainStmt)
	if !ok {
		return "", false, false
	}
	if explain.Format == ast.ExplainJSON {
		return "json", false, true
	}
	// The tabular and text formats are both tables of operations.
	return "text", false, true
}

// singleStatement returns the one statement in text. An explain request plans one
// statement at a time: both the drivers and the API gate split a multi-statement
// request first.
func singleStatement(text string) (ast.Node, error) {
	stmts, err := SplitSQL(text)
	if err != nil {
		return nil, err
	}
	stmts = base.FilterEmptyStatements(stmts)
	if len(stmts) == 0 {
		return nil, errors.New("EXPLAIN needs a statement to plan")
	}
	if len(stmts) > 1 {
		return nil, errors.New("EXPLAIN plans a single statement")
	}
	file, err := parser.Parse(stmts[0].Text)
	if err != nil {
		return nil, err
	}
	if file == nil || len(file.Stmts) == 0 {
		return nil, errors.New("EXPLAIN needs a statement to plan")
	}
	return file.Stmts[0], nil
}

// explainedText returns the text within text of the statement explain plans, and
// whether it could be located.
func explainedText(explain *ast.ExplainStmt, text string) (string, bool) {
	inner := ast.NodeLoc(explain.Stmt)
	// The plan ends where the EXPLAIN does, which is past clauses that the
	// planned statement's own location can leave out.
	if inner.Start < 0 || inner.Start >= explain.Loc.End || explain.Loc.End > len(text) {
		return "", false
	}
	return text[inner.Start:explain.Loc.End], true
}
