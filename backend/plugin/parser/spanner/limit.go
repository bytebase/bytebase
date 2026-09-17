package spanner

import (
	"strconv"

	"github.com/bytebase/omni/googlesql/ast"
	"github.com/bytebase/omni/googlesql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_SPANNER, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for Spanner, which
// takes no engineVersion. It sets the LIMIT of the outermost query in stmt to at
// most limit. It never wraps stmt in a subquery, where Spanner rejects a WITH
// clause, so a query it cannot rewrite runs unchanged and queryStatement caps its
// rows.
func statementWithResultLimit(stmt string, limit int, _ string) string {
	file, errs := parser.Parse(stmt)
	if len(errs) > 0 || len(file.Stmts) != 1 {
		return stmt
	}
	query, ok := file.Stmts[0].(*ast.QueryStmt)
	// FOR UPDATE fails in queryStatement's read-only transaction, so there is no
	// point fitting a LIMIT before it.
	if !ok || query.ForUpdate {
		return stmt
	}
	if query.Limit == nil {
		end := query.Loc.End
		return stmt[:end] + " LIMIT " + strconv.Itoa(limit) + stmt[end:]
	}
	count, ok := query.Limit.(*ast.Literal)
	if !ok || count.Kind != ast.LitInt || count.Ival <= int64(limit) {
		return stmt
	}
	return stmt[:count.Loc.Start] + strconv.Itoa(limit) + stmt[count.Loc.End:]
}
