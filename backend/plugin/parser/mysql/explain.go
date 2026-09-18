package mysql

import (
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_MYSQL, explainStatement)
	base.RegisterExplainStatementFunc(storepb.Engine_MARIADB, explainStatement)
	base.RegisterExplainStatementFunc(storepb.Engine_OCEANBASE, explainStatement)
}

// explainStatement implements base.ExplainStatement. MySQL returns one plan
// format here whatever the caller asked for: FORMAT=JSON and FORMAT=TREE are
// plans of their own shape that nothing downstream reads (the driver capability
// table offers this engine the default plan only).
func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", err
	}
	planned := statement
	if explain, ok := node.(*ast.ExplainStmt); ok {
		if explain.Analyze {
			return "", base.ErrExplainAnalyzeExecutes
		}
		if inner, ok := explainedText(explain, statement); ok {
			planned = inner
		} else {
			// Nothing to rebuild around, as for EXPLAIN FOR CONNECTION or the
			// EXPLAIN tbl_name that describes a table. Run what the user wrote
			// rather than plan a plan.
			return statement, nil
		}
	}
	return "EXPLAIN " + planned, nil
}

// singleStatement returns the one statement in text. An explain request plans one
// statement at a time: both the drivers and the API gate split a multi-statement
// request first.
func singleStatement(text string) (ast.Node, error) {
	list, err := ParseMySQL(text)
	if err != nil {
		return nil, convertOmniError(err, base.Statement{Text: text})
	}
	if list == nil || len(list.Items) == 0 {
		return nil, errors.New("EXPLAIN needs a statement to plan")
	}
	if len(list.Items) > 1 {
		return nil, errors.New("EXPLAIN plans a single statement")
	}
	return list.Items[0], nil
}

// explainedText returns the text within text of the statement explain plans, and
// whether it could be located.
func explainedText(explain *ast.ExplainStmt, text string) (string, bool) {
	// An executable comment moves every location after it, because the lexer
	// parses the comment's body with its markers spliced out of the text.
	if strings.Contains(text, "/*!") || strings.Contains(text, "/*M!") {
		return "", false
	}
	start, ok := explainedStart(explain.Stmt)
	if !ok {
		return "", false
	}
	// The plan ends where the EXPLAIN does, which is past clauses that the
	// planned statement's own location can leave out.
	if start < 0 || start >= explain.Loc.End || explain.Loc.End > len(text) {
		return "", false
	}
	return text[start:explain.Loc.End], true
}

// explainedStart returns where a statement that EXPLAIN can plan begins. The
// cases are what the grammar accepts after EXPLAIN; anything else it parses there
// describes a table rather than planning a statement.
func explainedStart(node ast.Node) (int, bool) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		return n.Loc.Start, true
	// REPLACE parses as an InsertStmt.
	case *ast.InsertStmt:
		return n.Loc.Start, true
	case *ast.UpdateStmt:
		return n.Loc.Start, true
	case *ast.DeleteStmt:
		return n.Loc.Start, true
	case *ast.TableStmt:
		return n.Loc.Start, true
	case *ast.ValuesStmt:
		return n.Loc.Start, true
	default:
		return 0, false
	}
}
