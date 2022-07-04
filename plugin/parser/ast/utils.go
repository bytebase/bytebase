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
	case *ChangeColumnStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		if n.Column != nil {
			Walk(v, n.Column)
		}
	case *RenameColumnStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		if n.Column != nil {
			Walk(v, n.Column)
		}
	case *AlterTableStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		for _, cmd := range n.AlterItemList {
			Walk(v, cmd)
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
	case *ForeignDef:
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
