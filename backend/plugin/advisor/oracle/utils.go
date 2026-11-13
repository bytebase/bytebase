package oracle

import (
	"fmt"
	"strconv"
	"strings"

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

func listStrings(list *ast.List) []string {
	if list == nil {
		return nil
	}
	result := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		switch n := item.(type) {
		case *ast.String:
			result = append(result, n.Str)
		case *ast.ColumnRef:
			result = append(result, n.Column)
		default:
		}
	}
	return result
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

func typeName(tn *ast.TypeName) string {
	parts := listStrings(tnNames(tn))
	return strings.ToUpper(strings.Join(parts, "."))
}

func tnNames(tn *ast.TypeName) *ast.List {
	if tn == nil {
		return nil
	}
	return tn.Names
}

func firstTypeModInt(tn *ast.TypeName) (int, bool) {
	if tn == nil || tn.TypeMods == nil || len(tn.TypeMods.Items) == 0 {
		return 0, false
	}
	switch n := tn.TypeMods.Items[0].(type) {
	case *ast.Integer:
		return int(n.Ival), true
	case *ast.Float:
		v, err := strconv.Atoi(n.Fval)
		return v, err == nil
	}
	return 0, false
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
