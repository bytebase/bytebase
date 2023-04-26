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
		if n.Type != nil {
			Walk(v, n.Type)
		}
		for _, cons := range n.ConstraintList {
			Walk(v, cons)
		}
	case *ColumnNameDef:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *ConstraintDef:
		if n.Foreign != nil {
			Walk(v, n.Foreign)
		}
	case *CopyStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *CreateIndexStmt:
		if n.Index != nil {
			Walk(v, n.Index)
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
	case *DeleteStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		if n.WhereClause != nil {
			Walk(v, n.WhereClause)
		}

		for _, like := range n.PatternLikeList {
			Walk(v, like)
		}
		for _, subquery := range n.SubqueryList {
			Walk(v, subquery)
		}
	case *DropColumnStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *DropConstraintStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *DropDatabaseStmt:
		// No members to walk through.
	case *DropIndexStmt:
		for _, indexDef := range n.IndexList {
			Walk(v, indexDef)
		}
	case *DropNotNullStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *DropTableStmt:
		for _, tableDef := range n.TableList {
			Walk(v, tableDef)
		}
	case *ExplainStmt:
		if n.Statement != nil {
			Walk(v, n.Statement)
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
		// No members to walk through.
	case *InsertStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		if n.Select != nil {
			Walk(v, n.Select)
		}
	case *PatternLikeDef:
		if n.Expression != nil {
			Walk(v, n.Expression)
		}
		if n.Pattern != nil {
			Walk(v, n.Pattern)
		}
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
	case *SelectStmt:
		if n.LQuery != nil {
			Walk(v, n.LQuery)
		}
		if n.RQuery != nil {
			Walk(v, n.RQuery)
		}
		for _, field := range n.FieldList {
			Walk(v, field)
		}
		if n.WhereClause != nil {
			Walk(v, n.WhereClause)
		}

		for _, like := range n.PatternLikeList {
			Walk(v, like)
		}
		for _, subquery := range n.SubqueryList {
			Walk(v, subquery)
		}
	case *SetNotNullStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *SetSchemaStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
	case *StringDef:
		// No members to walk through.
	case *SubqueryDef:
		if n.Select != nil {
			Walk(v, n.Select)
		}
	case *TableDef:
		// No members to walk through.
	case *UnconvertedExpressionDef:
		// No members to walk through.
	case *UpdateStmt:
		if n.Table != nil {
			Walk(v, n.Table)
		}
		if n.WhereClause != nil {
			Walk(v, n.WhereClause)
		}

		for _, like := range n.PatternLikeList {
			Walk(v, like)
		}
		for _, subquery := range n.SubqueryList {
			Walk(v, subquery)
		}
	}
}
