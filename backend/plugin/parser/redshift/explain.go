package redshift

import (
	redshiftast "github.com/bytebase/omni/redshift/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_REDSHIFT, explainStatement)
}

// explainStatement implements base.ExplainStatement. Redshift returns one plan
// format here whatever the caller asked for: its EXPLAIN has no machine-readable
// form (the driver capability table offers this engine the default plan only).
func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", err
	}
	planned := statement
	if explain, ok := node.(*redshiftast.ExplainStmt); ok {
		if hasOmniExplainAnalyze(explain) {
			return "", base.ErrExplainAnalyzeExecutes
		}
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
func singleStatement(text string) (redshiftast.Node, error) {
	stmts, err := ParseRedshift(text)
	if err != nil {
		return nil, err
	}
	var node redshiftast.Node
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
func explainedText(explain *redshiftast.ExplainStmt, text string) (string, bool) {
	// The statement runs from its WITH clause, which the location of a SELECT
	// leaves out, to the end of the EXPLAIN, which its location can also leave
	// out, as for ORDER BY.
	start := redshiftast.NodeLoc(explain.Query).Start
	if with := explainedWithClause(explain.Query); with != nil && with.Loc.Start >= 0 && with.Loc.Start < start {
		start = with.Loc.Start
	}
	if start < 0 || start >= explain.Loc.End || explain.Loc.End > len(text) {
		return "", false
	}
	return text[start:explain.Loc.End], true
}

// explainedWithClause returns the WITH clause of a statement EXPLAIN can plan.
func explainedWithClause(node redshiftast.Node) *redshiftast.WithClause {
	switch n := node.(type) {
	case *redshiftast.SelectStmt:
		return n.WithClause
	case *redshiftast.InsertStmt:
		return n.WithClause
	case *redshiftast.UpdateStmt:
		return n.WithClause
	case *redshiftast.DeleteStmt:
		return n.WithClause
	default:
		return nil
	}
}
