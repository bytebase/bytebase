package ast

// Visitor is the interface for visitor pattern.
type Visitor interface {
	Visit(Node) Visitor
}

// Walk walks the AST.
func Walk(v Visitor, node Node) {
	if v = v.Visit(node); v == nil {
		return
	}

	switch n := node.(type) {
	case *AddColumnListStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		for _, col := range n.ColumnList {
			Walk(v, col)
		}
	case *AddConstraintStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		if n.Constraint != nil {
			Walk(v, n.Constraint)
		}
	case *AlterTableStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		for _, cmd := range n.AlterItemList {
			Walk(v, cmd)
		}
	case *ChangeColumnStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		if n.Column != nil {
			Walk(v, n.Column)
		}
	case *ColumnDef:
	case *ConstraintDef:
		if n.Foreign != nil {
			Walk(v, n.Foreign)
		}
	case *CreateTableStmt:
		if n.Name != nil {
			Walk(v, n.Name)
		}
		for _, col := range n.ColumnList {
			Walk(v, col)
		}
		for _, cons := range n.ConstraintList {
			Walk(v, cons)
		}
	case *DropColumnStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *DropConstraintStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *DropIndexStmt:
		for _, indexDef := range n.IndexList {
			Walk(v, indexDef)
		}
	case *DropTableStmt:
		for _, tableDef := range n.TableList {
			Walk(v, tableDef)
		}
	case *ForeignDef:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *IndexDef:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		for _, keyDef := range n.KeyList {
			Walk(v, keyDef)
		}
	case *IndexKeyDef:
	case *RenameColumnStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *RenameConstraintStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *RenameIndexStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *RenameTableStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *TableDef:
	}
}
