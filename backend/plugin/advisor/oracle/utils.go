package oracle

import (
	"fmt"

	"github.com/bytebase/omni/oracle/ast"
)

func objectName(name *ast.ObjectName, currentSchema string) string {
	if name == nil {
		return ""
	}
	schema := name.Schema
	if schema == "" {
		schema = currentSchema
	}
	if schema == "" {
		return name.Name
	}
	return fmt.Sprintf("%s.%s", schema, name.Name)
}

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

func columnDefs(list *ast.List) []*ast.ColumnDef {
	if list == nil {
		return nil
	}
	result := make([]*ast.ColumnDef, 0, len(list.Items))
	for _, item := range list.Items {
		if col, ok := item.(*ast.ColumnDef); ok {
			result = append(result, col)
		}
	}
	return result
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

func columnConstraints(col *ast.ColumnDef) []*ast.ColumnConstraint {
	if col == nil || col.Constraints == nil {
		return nil
	}
	result := make([]*ast.ColumnConstraint, 0, len(col.Constraints.Items))
	for _, item := range col.Constraints.Items {
		if c, ok := item.(*ast.ColumnConstraint); ok {
			result = append(result, c)
		}
	}
	return result
}

func columnHasConstraint(col *ast.ColumnDef, typ ast.ConstraintType) bool {
	for _, c := range columnConstraints(col) {
		if c.Type == typ {
			return true
		}
	}
	return false
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
