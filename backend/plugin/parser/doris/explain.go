package doris

import (
	"github.com/bytebase/omni/doris/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_DORIS, explainStatement)
}

// explainStatement implements base.ExplainStatement. Doris returns one plan
// format here whatever the caller asked for: its other EXPLAIN forms are plans of
// their own shape that nothing downstream reads (supportedExplainFormats in the
// API offers this engine the default plan only).
func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
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
	return "EXPLAIN " + planned, nil
}

// singleStatement returns the one statement in text. An explain request plans one
// statement at a time: both the drivers and the API gate split a multi-statement
// request first.
func singleStatement(text string) (ast.Node, error) {
	parsed, err := parseDorisSQL(text)
	if err != nil {
		return nil, err
	}
	if len(parsed) == 0 || parsed[0].Node() == nil {
		return nil, errors.New("EXPLAIN needs a statement to plan")
	}
	if len(parsed) > 1 {
		return nil, errors.New("EXPLAIN plans a single statement")
	}
	return parsed[0].Node(), nil
}

// explainedText returns the text within text of the statement explain plans, and
// whether it could be located.
func explainedText(explain *ast.ExplainStmt, text string) (string, bool) {
	inner := ast.NodeLoc(explain.Query)
	// The plan ends where the EXPLAIN does, which is past clauses that the
	// planned statement's own location can leave out.
	if inner.Start < 0 || inner.Start >= explain.Loc.End || explain.Loc.End > len(text) {
		return "", false
	}
	return text[inner.Start:explain.Loc.End], true
}
