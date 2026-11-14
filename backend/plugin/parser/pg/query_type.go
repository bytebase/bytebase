package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/ast"
)

// intoClause returns the INTO clause of n, searching set-operation arms
// (UNION/INTERSECT/EXCEPT) — the parser attaches INTO to the first arm, not the root.
func intoClause(n *ast.SelectStmt) *ast.IntoClause {
	if n == nil {
		return nil
	}
	if n.IntoClause != nil {
		return n.IntoClause
	}
	if c := intoClause(n.Larg); c != nil {
		return c
	}
	return intoClause(n.Rarg)
}

// isExplainAnalyzeStmt reports whether an ExplainStmt executes its query: its last ANALYZE option
// is not FALSE, OFF, or 0, the values PostgreSQL reads as false.
func isExplainAnalyzeStmt(n *ast.ExplainStmt) bool {
	if n.Options == nil {
		return false
	}
	analyze := false
	for _, item := range n.Options.Items {
		de, ok := item.(*ast.DefElem)
		if !ok || !strings.EqualFold(de.Defname, "analyze") {
			continue
		}
		switch arg := de.Arg.(type) {
		case *ast.String:
			analyze = !strings.EqualFold(arg.Str, "false") && !strings.EqualFold(arg.Str, "off")
		case *ast.Integer:
			analyze = arg.Ival != 0
		default:
			analyze = true
		}
	}
	return analyze
}

