package mysql

import (
	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_MYSQL, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_MARIADB, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_OCEANBASE, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
// Only SELECT, EXPLAIN, SHOW, SET, and DESCRIBE are allowed in read-only mode.
// EXPLAIN ANALYZE is treated as non-read-only since it actually executes the query.
func validateQuery(statement string) (bool, bool, error) {
	stmts, err := ParseMySQLOmni(statement)
	if err != nil {
		return false, false, err
	}

	hasExecute := false
	readOnly := true
	for _, node := range stmts.Items {
		switch stmt := node.(type) {
		case *ast.SelectStmt:
			// SELECT is always allowed.
		case *ast.ExplainStmt:
			if stmt.Analyze {
				readOnly = false
			}
		case *ast.ShowStmt:
			// SHOW is always allowed.
		case *ast.SetStmt, *ast.SetPasswordStmt, *ast.SetDefaultRoleStmt, *ast.SetRoleStmt, *ast.SetResourceGroupStmt, *ast.SetTransactionStmt:
			hasExecute = true
		default:
			return false, false, nil
		}
	}
	return readOnly, !hasExecute, nil
}
