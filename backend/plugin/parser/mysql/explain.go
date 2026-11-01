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
}

// explainStatement implements base.ExplainStatement. MySQL and MariaDB both
// spell their structured plan FORMAT=JSON; any other format the caller names
// gets the engine's default plan, since FORMAT=TREE is a shape of its own that
// nothing downstream reads (the driver capability table refuses the rest).
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
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
	if format == base.ExplainFormatJSON {
		return "EXPLAIN FORMAT=JSON " + planned, nil
	}
	return "EXPLAIN " + planned, nil
}

// DescribeExplain returns the output format of an EXPLAIN statement and whether
// it executes the statement it plans. It is not an explain at all unless it
// plans a statement: EXPLAIN tbl_name and DESCRIBE tbl_name parse here too and
// describe a table's columns, which is a result like any other.
//
// An omitted FORMAT is not text: from MySQL 8.0.32 it follows the session's
// explain_format, which the statement alone does not reveal. It is reported as
// no format at all, which the caller reads as unknown.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", false, false
	}
	explain, ok := node.(*ast.ExplainStmt)
	if !ok {
		return "", false, false
	}
	if _, plans := explainedStart(explain.Stmt); !plans {
		return "", false, false
	}
	switch {
	case strings.EqualFold(explain.Format, "json"):
		return "json", explain.Analyze, true
	case explain.Format == "":
		return "", explain.Analyze, true
	default:
		// TRADITIONAL is a table and TREE a tree of lines; both are text.
		return "text", explain.Analyze, true
	}
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
	// The parser reads the body of an executable comment, so a location can fall
	// inside one and the slice would leave the comment unclosed.
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
