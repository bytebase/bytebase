package oracle

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/omni/oracle/ast"
)

type omniIdentifier struct {
	name string
	loc  ast.Loc
}

func omniObjectName(name *ast.ObjectName, currentSchema string) string {
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

func omniLastObjectName(name *ast.ObjectName) string {
	if name == nil {
		return ""
	}
	return name.Name
}

func omniListStrings(list *ast.List) []string {
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

func omniColumnDefs(list *ast.List) []*ast.ColumnDef {
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

func omniTableConstraints(list *ast.List) []*ast.TableConstraint {
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

func omniAlterTableCmds(stmt *ast.AlterTableStmt) []*ast.AlterTableCmd {
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

func omniColumnConstraints(col *ast.ColumnDef) []*ast.ColumnConstraint {
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

func omniTypeName(tn *ast.TypeName) string {
	parts := omniListStrings(tnNames(tn))
	return strings.ToUpper(strings.Join(parts, "."))
}

func tnNames(tn *ast.TypeName) *ast.List {
	if tn == nil {
		return nil
	}
	return tn.Names
}

func omniFirstTypeModInt(tn *ast.TypeName) (int, bool) {
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

func omniColumnHasConstraint(col *ast.ColumnDef, typ ast.ConstraintType) bool {
	for _, c := range omniColumnConstraints(col) {
		if c.Type == typ {
			return true
		}
	}
	return false
}

// omniIdentifiers returns the names a statement defines: created objects and
// their columns, constraints, and partitions; aliases and CTEs; rename targets;
// and PL/SQL declarations. References are excluded because the author did not
// choose those names, including built-ins such as COUNT, coll.COUNT,
// SQL%ROWCOUNT, and SQLCODE.
func omniIdentifiers(node ast.Node) []omniIdentifier {
	var result []omniIdentifier
	seen := make(map[string]bool)
	add := func(name string, loc ast.Loc) {
		if name == "" {
			return
		}
		key := fmt.Sprintf("%s:%d", name, loc.Start)
		if seen[key] {
			return
		}
		seen[key] = true
		result = append(result, omniIdentifier{name: name, loc: loc})
	}
	addObject := func(name *ast.ObjectName) {
		if name != nil {
			add(name.Name, name.Loc)
		}
	}
	addStrings := func(list *ast.List, loc ast.Loc) {
		for _, name := range omniListStrings(list) {
			add(name, loc)
		}
	}
	// A TableConstraint node also names the target of DROP CONSTRAINT, so
	// constraint names are taken from the statements that define them.
	addTableConstraints := func(list *ast.List) {
		for _, c := range omniTableConstraints(list) {
			add(c.Name, c.Loc)
		}
	}
	omniWalk(node, func(n ast.Node) {
		switch x := n.(type) {
		case *ast.CreateTableStmt:
			addObject(x.Name)
			addTableConstraints(x.Constraints)
		case *ast.CreateViewStmt:
			addObject(x.Name)
			addStrings(x.Columns, x.Loc)
			addTableConstraints(x.Constraints)
			add(x.ConstraintName, x.Loc)
		case *ast.CreateIndexStmt:
			addObject(x.Name)
		case *ast.CreateSequenceStmt:
			addObject(x.Name)
		case *ast.CreateSynonymStmt:
			addObject(x.Name)
		case *ast.CreateTypeStmt:
			addObject(x.Name)
		case *ast.CreatePackageStmt:
			addObject(x.Name)
		case *ast.CreateProcedureStmt:
			addObject(x.Name)
		case *ast.CreateFunctionStmt:
			addObject(x.Name)
		case *ast.CreateTriggerStmt:
			addObject(x.Name)
		case *ast.RenameStmt:
			addObject(x.NewName)
		case *ast.AlterTableCmd:
			switch x.Action {
			case ast.AT_ADD_CONSTRAINT:
				if x.Constraint != nil {
					add(x.Constraint.Name, x.Constraint.Loc)
				}
			case ast.AT_RENAME, ast.AT_RENAME_COLUMN, ast.AT_RENAME_CONSTRAINT, ast.AT_RENAME_PARTITION:
				add(x.NewName, x.Loc)
			default:
			}
		case *ast.AlterIndexStmt:
			add(x.NewName, x.Loc)
		case *ast.AlterTriggerStmt:
			add(x.NewName, x.Loc)
		case *ast.ColumnDef:
			add(x.Name, x.Loc)
		case *ast.ColumnConstraint:
			add(x.Name, x.Loc)
		case *ast.PartitionDef:
			add(x.Name, x.Loc)
		case *ast.ResTarget:
			add(x.Name, x.Loc)
		case *ast.Alias:
			add(x.Name, x.Loc)
			addStrings(x.Cols, x.Loc)
		case *ast.CTE:
			add(x.Name, x.Loc)
			addStrings(x.Columns, x.Loc)
		case *ast.Parameter:
			add(x.Name, x.Loc)
		case *ast.PLSQLVarDecl:
			add(x.Name, x.Loc)
		case *ast.PLSQLCursorDecl:
			add(x.Name, x.Loc)
		case *ast.PLSQLTypeDecl:
			add(x.Name, x.Loc)
		case *ast.PLSQLSubtypeDecl:
			add(x.Name, x.Loc)
		case *ast.PLSQLLoop:
			// Loops with an Iteration declare their variables as PLSQLIterand nodes.
			if x.Iteration == nil {
				add(x.Iterator, x.Loc)
			}
		case *ast.PLSQLIterand:
			add(x.Name, x.Loc)
		case *ast.PLSQLForall:
			add(x.Index, x.Loc)
		default:
		}
	})
	return result
}

func omniWalk(node ast.Node, visit func(ast.Node)) {
	ast.Inspect(node, func(n ast.Node) bool {
		visit(n)
		return true
	})
}

func omniWalkPLSQLBlockStatements(block *ast.PLSQLBlock, visit func(ast.StmtNode) bool) {
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
