package oracle

import (
	"github.com/bytebase/omni/oracle/ast"
)

func tableConstraints(list *ast.List) []*ast.TableConstraint {
	if list == nil {
		return nil
	}
	result := make([]*ast.TableConstraint, 0, len(list.Items))
	for _, item := range list.Items {
		if c, ok := item.(*ast.TableConstraint); ok {
			result = append(result, c)
		}
	}
	return result
}

func alterTableCmds(stmt *ast.AlterTableStmt) []*ast.AlterTableCmd {
	if stmt == nil || stmt.Actions == nil {
		return nil
	}
	result := make([]*ast.AlterTableCmd, 0, len(stmt.Actions.Items))
	for _, item := range stmt.Actions.Items {
		if cmd, ok := item.(*ast.AlterTableCmd); ok {
			result = append(result, cmd)
		}
	}
	return result
}
