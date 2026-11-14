package oracle

import (
	"github.com/bytebase/omni/oracle/ast"
)

func lastObjectName(name *ast.ObjectName) string {
	if name == nil {
		return ""
	}
	return name.Name
}

func listItems(list *ast.List) []ast.Node {
	if list == nil {
		return nil
	}
	return list.Items
}

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

func walk(node ast.Node, visit func(ast.Node)) {
	ast.Inspect(node, func(n ast.Node) bool {
		visit(n)
		return true
	})
}

func walkPLSQLBlockStatements(block *ast.PLSQLBlock, visit func(ast.StmtNode) bool) {
	if block == nil {
		return
	}
	visited := make(map[ast.Node]bool)
	ast.Inspect(block, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		if node == block {
			return true
		}
		stmt, ok := node.(ast.StmtNode)
		if !ok {
			return true
		}
		return visit(stmt)
	})
}
