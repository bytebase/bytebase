package ast

// AlterTableStmt is the struct for alter table or view statement.
type AlterTableStmt struct {
	node

	Table         *TableDef
	AlterItemList []Node
}
